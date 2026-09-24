// mirasim2api 桌面托盘 app：macOS 常驻菜单栏（LSUIElement，无 Dock 图标），
// Windows 常驻系统托盘（无控制台窗）。后台运行网关服务，菜单一键打开管理后台、
// 复制接口地址、查看状态、开关「开机自动启动」与退出。
package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/getlantern/systray"

	"mirasim2api/internal/autostart"
	"mirasim2api/internal/server"
)

//go:embed assets/trayTemplate.png
var trayIcon []byte

// version 由 build_app.sh 经 -ldflags "-X main.version=…" 注入，默认开发版。
var version = "0.1.0-dev"

type app struct {
	cancel context.CancelFunc
	logger *slog.Logger

	mu      sync.Mutex
	status  string // 状态行短文案（菜单内显示，须克制长度）
	tip     string // 状态行 tooltip（可放完整信息）
	addr    string // 运行中时的监听地址（host:port）
	running bool
	done    chan struct{} // server.Run 返回后关闭

	mStatus    *systray.MenuItem
	mAdmin     *systray.MenuItem
	mCopy      *systray.MenuItem
	mDataDir   *systray.MenuItem
	mAutostart *systray.MenuItem
	mQuit      *systray.MenuItem
}

func main() {
	// 未显式配置时给桌面场景设置安全默认：仅本机监听 + 用户级数据目录（按平台取值）。
	setDefaultEnv("HOST", "127.0.0.1")
	setDefaultEnv("DATA_DIR", defaultDataDir())

	logger := setupLogger(os.Getenv("DATA_DIR"))

	ctx, cancel := context.WithCancel(context.Background())
	a := &app{cancel: cancel, logger: logger, status: "启动中…", done: make(chan struct{})}

	go func() {
		defer close(a.done)
		err := server.Run(ctx, logger, func(addr string) {
			logger.Info("mirasim2api 已启动", "addr", "http://"+addr)
			a.mu.Lock()
			a.addr = addr
			a.running = true
			a.status = "● 运行中 · " + addr
			a.tip = "http://" + addr + "/admin/"
			a.mu.Unlock()
			a.refresh()
		})
		a.mu.Lock()
		a.running = false
		if err != nil && ctx.Err() == nil {
			logger.Error("网关启动失败", "err", err)
			a.status, a.tip = failStatus(err, envOr("PORT", "8787"))
		} else {
			a.status, a.tip = "已停止", ""
		}
		a.mu.Unlock()
		a.refresh()
	}()

	systray.Run(a.onReady, a.onExit)
}

// failStatus 把底层错误翻译成一行短状态 + tooltip 详情。
func failStatus(err error, port string) (status, tip string) {
	switch {
	case strings.Contains(err.Error(), "address already in use"):
		return "⚠ 端口 " + port + " 被占用", err.Error() + "\n退出占用该端口的进程后重新打开 App"
	default:
		return "⚠ 启动失败", err.Error()
	}
}

// setDefaultEnv 仅在变量未设置（或为空）时补给默认值，不覆盖用户环境。
func setDefaultEnv(key, val string) {
	if v, ok := os.LookupEnv(key); !ok || v == "" {
		_ = os.Setenv(key, val)
	}
}

// setupLogger 同时输出到数据目录下的日志文件（追加），便于排查后台运行问题。
func setupLogger(dataDir string) *slog.Logger {
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	f, err := os.OpenFile(filepath.Join(logDir, "app.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return slog.New(slog.NewTextHandler(f, nil))
}

func (a *app) onReady() {
	setTrayIcon()
	systray.SetTooltip("mirasim2api v" + version)

	a.mStatus = systray.AddMenuItem(a.status, a.tip)
	a.mStatus.Disable()
	systray.AddSeparator()
	a.mAdmin = systray.AddMenuItem("打开管理后台", "在浏览器打开 /admin/")
	a.mCopy = systray.AddMenuItem("复制接口地址", "")
	a.mDataDir = systray.AddMenuItem("打开数据目录", "SQLite / master.key / 日志")
	systray.AddSeparator()
	if autostart.Available() {
		a.mAutostart = systray.AddMenuItemCheckbox("开机自动启动", "登录后自动运行", autostart.Enabled())
	}
	systray.AddSeparator()
	a.mQuit = systray.AddMenuItem("退出", "停止网关并退出")

	a.refresh() // 服务可能先于菜单就绪（如端口秒占），建完项后同步一次状态
	go a.clickLoop()
}

func (a *app) clickLoop() {
	var autostartCh <-chan struct{}
	if a.mAutostart != nil {
		autostartCh = a.mAutostart.ClickedCh
	}
	for {
		select {
		case <-a.mAdmin.ClickedCh:
			openURL(a.adminURL())
		case <-a.mCopy.ClickedCh:
			if addr := a.currentAddr(); addr != "" {
				copyToClipboard("http://" + addr)
				a.flash("已复制 " + addr)
			}
		case <-a.mDataDir.ClickedCh:
			openURL(os.Getenv("DATA_DIR"))
		case <-autostartCh:
			a.toggleAutostart()
		case <-a.mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func (a *app) toggleAutostart() {
	on := autostart.Enabled()
	if err := autostart.SetEnabled(!on); err != nil {
		a.logger.Error("切换开机自启失败", "err", err)
		a.flash("切换失败，详见日志")
		return
	}
	if !on {
		a.mAutostart.Check()
		a.flash("已开启，下次登录自动运行")
	} else {
		a.mAutostart.Uncheck()
		a.flash("已关闭开机自动启动")
	}
}

// flash 在状态行短暂显示一条提示，2.5 秒后恢复。文案不超过 10 个汉字。
func (a *app) flash(msg string) {
	if a.mStatus == nil {
		return
	}
	a.mStatus.SetTitle("✦ " + msg)
	time.AfterFunc(2500*time.Millisecond, a.refresh)
}

// refresh 把内存状态刷到状态菜单项（systray 可在任意 goroutine 调用）。
func (a *app) refresh() {
	a.mu.Lock()
	status, tip := a.status, a.tip
	a.mu.Unlock()
	if a.mStatus == nil {
		return
	}
	a.mStatus.SetTitle(status)
	a.mStatus.SetTooltip(tip)
}

// adminURL 返回后台地址；未就绪时回退到配置的 HOST:PORT。
func (a *app) adminURL() string {
	return "http://" + a.currentAddr() + "/admin/"
}

func (a *app) currentAddr() string {
	a.mu.Lock()
	addr := a.addr
	a.mu.Unlock()
	if addr != "" {
		return addr
	}
	return fmt.Sprintf("%s:%s", os.Getenv("HOST"), envOr("PORT", "8787"))
}

func (a *app) onExit() {
	a.cancel()
	select {
	case <-a.done:
	case <-time.After(12 * time.Second): // 兜底，不卡在退出
	}
	a.logger.Info("mirasim2api 已退出")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// label 是 LaunchAgent 标识，与桌面 app 的 bundle identifier 保持一致。
const label = "io.mirasim2api.app"

// Available 报告当前平台是否支持开机自启管理（macOS 恒为 true）。
func Available() bool { return true }

// Enabled 报告 LaunchAgent 是否已注册（plist 存在且指向当前可执行文件）。
func Enabled() bool {
	raw, err := os.ReadFile(plistPath())
	if err != nil {
		return false
	}
	exe, err := execPath()
	if err != nil {
		return true // 有 plist 但拿不到自身路径时，按已注册处理
	}
	return strings.Contains(string(raw), exe)
}

// SetEnabled 注册/注销登录启动。开启仅写入 plist（下次登录生效，由 launchd
// 在用户 GUI 会话中拉起，托盘菜单栏图标正常显示）；关闭同时尝试 bootout 已加载项。
func SetEnabled(on bool) error {
	if on {
		return install()
	}
	// bootout 失败（未加载等）不视为错误，plist 移除即达目的。
	_ = exec.Command("launchctl", "bootout", "gui/"+uid()+"/"+label).Run()
	if err := os.Remove(plistPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func install() error {
	exe, err := execPath()
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	logDir := filepath.Join(home, "Library", "Logs", "mirasim2api")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>ProcessType</key>
	<string>Interactive</string>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>
`, label, xmlEscape(exe), xmlEscape(filepath.Join(logDir, "agent.log")), xmlEscape(filepath.Join(logDir, "agent.log")))

	if err := os.MkdirAll(filepath.Dir(plistPath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(plistPath(), []byte(plist), 0o644)
}

func plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist")
}

// execPath 返回当前进程可执行文件的绝对路径（解析符号链接）。
func execPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func uid() string {
	out, err := exec.Command("id", "-u").Output()
	if err != nil {
		return "501"
	}
	return strings.TrimSpace(string(out))
}

func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

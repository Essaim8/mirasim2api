//go:build windows

package main

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/getlantern/systray"
)

//go:embed assets/tray.ico
var trayICO []byte

// sysProcAttrHideConsole 隐藏 cmd/clip 等子进程的控制台黑窗。
func sysProcAttrHideConsole() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

// setTrayIcon Windows 托盘用 .ico（彩色渐变款，任务栏深浅主题均可辨识）。
func setTrayIcon() { systray.SetIcon(trayICO) }

// openURL 用系统默认程序打开 http(s) 链接或本地目录。
func openURL(target string) {
	if target == "" {
		return
	}
	cmd := exec.Command("cmd", "/c", "start", "", target)
	cmd.SysProcAttr = sysProcAttrHideConsole()
	_ = cmd.Start()
}

// copyToClipboard 经 clip.exe 写入剪贴板。
func copyToClipboard(s string) {
	cmd := exec.Command("clip")
	cmd.Stdin = strings.NewReader(s)
	cmd.SysProcAttr = sysProcAttrHideConsole()
	_ = cmd.Run()
}

// defaultDataDir Windows 下用 %LOCALAPPDATA%\mirasim2api。
func defaultDataDir() string {
	if v := os.Getenv("LOCALAPPDATA"); v != "" {
		return filepath.Join(v, "mirasim2api")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "AppData", "Local", "mirasim2api")
}

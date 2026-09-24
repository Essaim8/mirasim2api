//go:build !darwin && !windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/getlantern/systray"
)

// setTrayIcon 非 macOS/Windows 平台用普通图标。
func setTrayIcon() { systray.SetIcon(trayIcon) }

// openURL 尝试 xdg-open（Linux）打开链接或目录。
func openURL(target string) {
	if target == "" {
		return
	}
	_ = exec.Command("xdg-open", target).Start()
}

// copyToClipboard 非 macOS/Windows 平台暂未支持（静默忽略）。
func copyToClipboard(string) {}

// defaultDataDir Linux 下遵循 XDG：~/.local/share/mirasim2api。
func defaultDataDir() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "mirasim2api")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "mirasim2api")
}

//go:build darwin

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/getlantern/systray"
)

// setTrayIcon 使用 template 图标，自动适配 macOS 深/浅色菜单栏。
func setTrayIcon() {
	systray.SetTemplateIcon(trayIcon, trayIcon)
}

// openURL 用系统默认方式打开 http(s) 链接或本地目录。
func openURL(target string) {
	if target == "" {
		return
	}
	_ = exec.Command("open", target).Start()
}

// copyToClipboard 经 pbcopy 写入剪贴板。
func copyToClipboard(s string) {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(s)
	_ = cmd.Run()
}

// defaultDataDir macOS 下用 ~/Library/Application Support/mirasim2api。
func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "mirasim2api")
}

//go:build windows

package autostart

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// runValue 是 HKCU\...\Run 下的值名。
const runValue = "Mirasim2API"

// Available 报告当前平台是否支持开机自启管理（Windows 恒为 true）。
func Available() bool { return true }

// Enabled 报告 Run 键中是否已注册当前可执行文件。
func Enabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	val, _, err := k.GetStringValue(runValue)
	if err != nil {
		return false
	}
	exe, err := execPath()
	if err != nil {
		return true // 有注册项但拿不到自身路径时，按已注册处理
	}
	return filepath.Clean(val) == filepath.Clean(exe)
}

// SetEnabled 注册/注销登录启动（HKCU Run 键，无需管理员权限）。
func SetEnabled(on bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开 Run 注册表键失败: %w", err)
	}
	defer k.Close()
	if !on {
		if err := k.DeleteValue(runValue); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}
	exe, err := execPath()
	if err != nil {
		return err
	}
	return k.SetStringValue(runValue, exe)
}

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// execPath 返回当前进程可执行文件的绝对路径（解析符号链接）。
func execPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

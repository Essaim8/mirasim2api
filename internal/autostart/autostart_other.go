//go:build !darwin && !windows

package autostart

import "errors"

// Available 报告当前平台是否支持开机自启管理。
func Available() bool { return false }

// Enabled 在非 macOS 平台恒为 false。
func Enabled() bool { return false }

// SetEnabled 在非 macOS 平台返回不支持错误。
func SetEnabled(bool) error { return errors.New("当前平台暂不支持开机自启") }

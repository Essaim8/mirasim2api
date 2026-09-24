//go:build darwin

package autostart

import (
	"os"
	"strings"
	"testing"
)

// TestSetEnabledRoundTrip 真实地写入/移除 LaunchAgent plist 并校验状态。
// 结束时务必 Disable，避免在用户机器上留下注册项。
func TestSetEnabledRoundTrip(t *testing.T) {
	t.Cleanup(func() { _ = SetEnabled(false) })

	if Enabled() {
		t.Skip("已存在注册项，跳过以免破坏用户配置")
	}
	if err := SetEnabled(true); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if !Enabled() {
		t.Fatal("Enable 后 Enabled() 应为 true")
	}
	raw, err := os.ReadFile(plistPath())
	if err != nil {
		t.Fatalf("读 plist: %v", err)
	}
	exe, _ := execPath()
	if !strings.Contains(string(raw), label) || !strings.Contains(string(raw), exe) {
		t.Fatalf("plist 内容不完整:\n%s", raw)
	}
	if !strings.Contains(string(raw), "<key>RunAtLoad</key>") {
		t.Fatal("plist 缺少 RunAtLoad")
	}

	if err := SetEnabled(false); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if Enabled() {
		t.Fatal("Disable 后 Enabled() 应为 false")
	}
	if _, err := os.Stat(plistPath()); !os.IsNotExist(err) {
		t.Fatalf("plist 应被移除: %v", err)
	}
}

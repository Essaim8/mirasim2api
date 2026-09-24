#!/bin/bash
# 交叉编译 Windows x64 托盘版 Mirasim2API.exe（可在 macOS/Linux 上跑）。
# 用法: deploy/windows/build_exe.sh [输出目录，默认 build/windows]；VERSION=x.y.z 可选。
set -euo pipefail
cd "$(dirname "$0")/../.."
ROOT="$(pwd)"
OUT="${1:-$ROOT/build/windows}"
VERSION="${VERSION:-0.1.0-dev}"
mkdir -p "$OUT"

WORKDIR="$(mktemp -d)"
SYSO=cmd/desktop/rsrc_windows_amd64.syso
cleanup() { rm -rf "$WORKDIR" "$SYSO"; }
trap cleanup EXIT

# 程序清单：asInvoker 免提权；PerMonitorV2 DPI 感知，托盘图标在高 DPI 下不糊。
cat > "$WORKDIR/app.manifest" <<'MANIFEST'
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity version="1.0.0.0" processorArchitecture="amd64" name="io.mirasim2api.app" type="win32"/>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security>
      <requestedPrivileges>
        <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
      </requestedPrivileges>
    </security>
  </trustInfo>
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings>
      <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
      <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">permonitorv2,permonitor,system</dpiAwareness>
    </windowsSettings>
  </application>
</assembly>
MANIFEST

echo "→ 生成 exe 资源（图标 + manifest）"
go run github.com/akavel/rsrc@latest \
	-manifest "$WORKDIR/app.manifest" \
	-ico cmd/desktop/assets/tray.ico \
	-arch amd64 \
	-o "$SYSO"

echo "→ 编译 cmd/desktop (windows/amd64, 纯静态, GUI 子系统无控制台窗)"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
	go build -trimpath -ldflags "-s -w -H windowsgui -X main.version=$VERSION" \
	-o "$OUT/Mirasim2API.exe" ./cmd/desktop

( cd "$OUT" && zip -q -X "Mirasim2API-windows-amd64.zip" Mirasim2API.exe )

echo "完成:"
echo "  $OUT/Mirasim2API.exe"
echo "  $OUT/Mirasim2API-windows-amd64.zip"
echo "拷到 Windows 双击即用；数据目录 %LOCALAPPDATA%\\mirasim2api，"
echo "托盘菜单里勾选「开机自动启动」即写入 HKCU Run 键。"

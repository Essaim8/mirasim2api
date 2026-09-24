#!/bin/bash
# 一键构建 Mirasim2API.app（macOS 菜单栏托盘 app）。
# 用法: deploy/macos/build_app.sh [输出目录，默认 build/macos]
set -euo pipefail
cd "$(dirname "$0")/../.."
ROOT="$(pwd)"
OUT="${1:-$ROOT/build/macos}"
APP="$OUT/Mirasim2API.app"

# 建议带版本: ./deploy/macos/build_app.sh 前 export VERSION=0.1.0
VERSION="${VERSION:-0.1.0-dev}"

echo "→ 编译 cmd/desktop (CGO、darwin/arm64)"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" \
	-o "$APP/Contents/MacOS/mirasim2api" ./cmd/desktop

echo "→ 生成 AppIcon.icns"
ICONSET="$OUT/AppIcon.iconset"
rm -rf "$ICONSET"
go run ./deploy/macos/genicon -iconset "$ICONSET"
if command -v iconutil >/dev/null; then
	iconutil -c icns "$ICONSET" -o "$APP/Contents/Resources/AppIcon.icns"
else
	echo "  (未找到 iconutil，跳过 .icns，仅保留 PNG iconset)"
fi

echo "→ 写 Info.plist"
cat > "$APP/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key><string>Mirasim2API</string>
	<key>CFBundleDisplayName</key><string>Mirasim2API</string>
	<key>CFBundleIdentifier</key><string>io.mirasim2api.app</string>
	<key>CFBundleVersion</key><string>$VERSION</string>
	<key>CFBundleShortVersionString</key><string>$VERSION</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleExecutable</key><string>mirasim2api</string>
	<key>CFBundleIconFile</key><string>AppIcon</string>
	<key>LSMinimumSystemVersion</key><string>12.0</string>
	<key>LSUIElement</key><true/>
	<key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
EOF

if command -v codesign >/dev/null; then
	echo "→ ad-hoc 签名"
	codesign --force --deep --sign - "$APP" >/dev/null 2>&1 || echo "  (签名失败可忽略，本地运行不受影响)"
fi

echo "✓ 完成: $APP"
echo "  双击运行；或 cp -R 到 /Applications 后在启动台打开。"
echo "  数据与日志: ~/Library/Application Support/mirasim2api"

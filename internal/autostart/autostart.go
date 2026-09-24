// Package autostart 管理「登录后自动启动」。
//
//	macOS：用户 LaunchAgent（~/Library/LaunchAgents/io.mirasim2api.app.plist），
//	       RunAtLoad 生效，勾选后下次登录自动拉起（托盘菜单栏图标正常显示）。
//	Windows：HKCU\Software\Microsoft\Windows\CurrentVersion\Run 注册表值，
//	       指向当前 exe，无需管理员权限，登录即启动。
//	其他平台：Available() = false，菜单项不展示。
package autostart

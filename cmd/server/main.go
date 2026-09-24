// mirasim2api 服务入口（CLI）：加载配置、初始化 SQLite 与设备身份、
// 装配账号池 / 计费 / 网关 / 管理后台，启动 HTTP 服务。
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"mirasim2api/internal/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	onReady := func(addr string) {
		logger.Info("mirasim2api 已启动",
			"addr", "http://"+addr,
			"admin", "http://"+addr+"/admin/")
	}
	if err := server.Run(ctx, logger, onReady); err != nil {
		logger.Error("启动失败", "err", err)
		os.Exit(1)
	}
}

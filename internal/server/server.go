// Package server 装配并运行 mirasim2api HTTP 服务（网关 + 管理后台）。
// 启动逻辑从 cmd/server 抽出，供 CLI 与桌面托盘 app 复用。
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"mirasim2api/internal/admin"
	"mirasim2api/internal/billing"
	"mirasim2api/internal/config"
	"mirasim2api/internal/gateway"
	"mirasim2api/internal/mirasim"
	"mirasim2api/internal/pool"
	"mirasim2api/internal/runtimecfg"
	"mirasim2api/internal/store"
	"mirasim2api/web"
)

// Run 加载配置并启动 HTTP 服务，直到 ctx 取消（优雅停机）或监听失败。
// onReady 在端口绑定成功后回调一次（可为 nil），参数是实际监听地址（host:port）。
func Run(ctx context.Context, logger *slog.Logger, onReady func(addr string)) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// 安全闸：未设管理密码且绑定非 loopback，任何人都能进后台。
	if cfg.AdminPassword == "" && !isLoopback(cfg.Host) {
		return fmt.Errorf("未设置 ADMIN_PASSWORD 且 HOST=%s 非 loopback，拒绝启动；"+
			"请设置管理密码或将 HOST 改为 127.0.0.1", cfg.Host)
	}

	st, err := store.Open(cfg.DataDir, cfg.MasterKey)
	if err != nil {
		return err
	}
	defer st.Close()

	// 本实例共享的 Ed25519 设备身份（加密落盘，重启不变）。
	devicePriv, _, _, err := st.LoadOrCreateDeviceKey()
	if err != nil {
		return fmt.Errorf("加载设备身份失败: %w", err)
	}

	// 运行时设置访问器：数据库优先，环境变量兜底，再到内置默认。
	settings := runtimecfg.New(st, envSnapshot(), map[string]string{
		runtimecfg.KeyCloakMode:         cfg.ClaudeCloakMode,
		runtimecfg.KeyCapacityRetries:   itoa(cfg.CapacityRetries),
		runtimecfg.KeyCapacityBackoffMS: itoa(cfg.CapacityBackoffMS),
	})
	settingsFn := settings.Get

	// 上游代理：运行时设置优先，Config 环境变量兜底。
	proxyFunc := func() string {
		if v := settings.Get(runtimecfg.KeyUpstreamProxy); v != "" {
			return v
		}
		return cfg.UpstreamProxy
	}

	// 账号池：共享设备身份 + 运行时代理；refresh token 轮换由 pool 回写。
	pl := pool.New(st, func(acc *store.Account) (*mirasim.Client, error) {
		return mirasim.NewClientFunc(acc.RefreshToken, devicePriv,
			cfg.RelayURL, cfg.AuthURL, cfg.ClientVersion, cfg.SealPubkey, proxyFunc)
	})

	rec := billing.NewRecorder(st)
	defer rec.Close()

	mux := http.NewServeMux()
	mux.Handle("/v1/", gateway.NewHandler(gateway.Deps{
		Store: st, Pool: pl, Config: cfg, Billing: rec, Settings: settingsFn,
	}))
	// 裸路径（无 /v1 前缀）也进网关。
	mux.Handle("/", gateway.NewHandler(gateway.Deps{
		Store: st, Pool: pl, Config: cfg, Billing: rec, Settings: settingsFn,
	}))
	mux.Handle("/admin/", admin.NewHandler(admin.Deps{
		Store: st, Pool: pl, Config: cfg, Settings: settings, WebFS: web.FS, Logger: logger,
	}))

	go pl.StartLimitsLoop(ctx)

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
	}

	addr := net.JoinHostPort(cfg.Host, itoa(cfg.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("监听 %s 失败: %w", addr, err)
	}
	if onReady != nil {
		onReady(ln.Addr().String())
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("收到退出信号，优雅停机…")
		shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shCtx)
	}
}

// isLoopback 判断监听地址是否仅 loopback（空/localhost/127.x/::1 视为本地）。
func isLoopback(host string) bool {
	if host == "" || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// envSnapshot 捕获显式设置过的可热更环境变量（区分 env 与 default 来源）。
func envSnapshot() map[string]string {
	out := map[string]string{}
	for key, envName := range runtimecfg.EnvKeys {
		if v, ok := os.LookupEnv(envName); ok && v != "" {
			out[key] = v
		}
	}
	return out
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

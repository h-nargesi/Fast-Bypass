package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fast-bypass/internal/app"
	"fast-bypass/internal/config"
	"fast-bypass/internal/server"
	"fast-bypass/internal/store"
)

func main() {
	cfg := config.Load()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	log.Info("Sabalan has started ...")

	st, err := store.Open(cfg.SQLitePath)
	if err != nil {
		log.Error("open database", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	application, err := app.New(cfg, log, st)
	if err != nil {
		log.Error("init app", "err", err)
		os.Exit(1)
	}
	if err := application.BootstrapAdmin(context.Background()); err != nil {
		log.Error("bootstrap", "err", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      server.New(application),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Info("listening", "addr", cfg.HTTPAddr, "mikrotik_fake", cfg.MikrotikFake, "mikrotik_api", cfg.MikrotikAPI, "mikrotik_host", cfg.MikrotikHost, "mikrotik_port", cfg.MikrotikPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

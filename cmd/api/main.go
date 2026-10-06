package main

import (
	"context"
	"github.com/Arjunren/asset-management-system/internal/config"
	"github.com/Arjunren/asset-management-system/internal/httpapi"
	"github.com/Arjunren/asset-management-system/internal/postgres"
	"github.com/Arjunren/asset-management-system/internal/service"
	"github.com/go-chi/cors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	l := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, e := config.Load()
	if e != nil {
		l.Error("invalid config", "error", e)
		os.Exit(1)
	}
	c, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	db, e := postgres.Open(c, cfg.DatabaseURL)
	if e != nil {
		l.Error("database unavailable", "error", e)
		os.Exit(1)
	}
	defer db.Close()
	h := httpapi.New(service.New(db, cfg.JWTSecret, cfg.JWTExpiry), l).Router()
	h = cors.Handler(cors.Options{AllowedOrigins: cfg.AllowedOrigins, AllowedMethods: []string{"GET", "POST", "PUT", "OPTIONS"}, AllowedHeaders: []string{"Authorization", "Content-Type"}, MaxAge: 300})(h)
	srv := &http.Server{Addr: cfg.Addr, Handler: h, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		l.Info("server started", "address", cfg.Addr)
		if e := srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			l.Error("server failed", "error", e)
			stop()
		}
	}()
	<-c.Done()
	x, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(x)
}

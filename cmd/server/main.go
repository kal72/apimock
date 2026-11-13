package main

import (
	"apimock/internal/config"
	"apimock/internal/handler"
	"apimock/internal/repository"
	"apimock/internal/usecase"
	"apimock/internal/web"
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
func run() error {
	settings, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is required; see .env.example")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	startup, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err = pool.Ping(startup); err != nil {
		return err
	}
	repo := repository.New(pool)
	if err = repo.Migrate(startup); err != nil {
		return err
	}
	auth := usecase.AuthService{Repo: repo}
	if err = auth.Bootstrap(startup, settings.AdminUsername, settings.AdminPassword); err != nil {
		return err
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	h := handler.Handler{
		Auth:    &handler.Identity{Service: auth},
		Config:  settings,
		Service: usecase.NewService(repo),
		Assets:  http.StripPrefix("/assets/", http.FileServer(http.FS(web.Assets))),
		Ready: func(r *http.Request) error {
			c, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			return pool.Ping(c)
		},
	}
	server := &http.Server{Addr: addr, Handler: h.Routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	done := make(chan error, 1)
	go func() {
		slog.Info("Mocklab ready", "url", "http://"+addr+"/app")
		done <- server.ListenAndServe()
	}()
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		slog.Info("Server received shutdown signal, shutting down gracefully...")
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}

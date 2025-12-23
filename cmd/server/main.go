package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"metadata-api/internal/api"
	"metadata-api/internal/db"
)

func main() {
	var (
		addr   = flag.String("addr", ":8000", "listen address")
		dbPath = flag.String("db", "", "path to spotify_clean.sqlite3")
	)
	flag.Parse()

	if *dbPath == "" {
		slog.Error("db path required")
		os.Exit(1)
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	handler := api.New(database)
	srv := &http.Server{
		Addr:         *addr,
		Handler:      handler.Routes(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		slog.Info("starting server", "addr", *addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}


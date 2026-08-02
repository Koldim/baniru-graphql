package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"baniru-test/internal/config"
	"baniru-test/internal/graph"
	"baniru-test/internal/store"
)

func main() {
	_ = godotenv.Load()

	cfg := config.FromEnv()

	ctx := context.Background()
	db, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("подключение к бд: %v", err)
	}
	defer db.Close()

	srv := graph.NewServer(db)

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("graphql слушает %s", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
}
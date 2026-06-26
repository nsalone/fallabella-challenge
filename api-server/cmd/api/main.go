package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-server/internal/api"
	"api-server/internal/db"
	"api-server/internal/repository"
)

const defaultAddr = ":8080"

func main() {
	log.SetFlags(log.LstdFlags)

	ctx := context.Background()

	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	products := repository.NewProductRepository(pool)
	handler := api.NewHandler(products)
	router := api.NewRouter(handler)

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("api server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen and serve: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("shutdown requested")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
}

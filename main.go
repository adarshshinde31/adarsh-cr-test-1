package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adarshshinde/feature-flags/internal/api"
	"github.com/adarshshinde/feature-flags/internal/flags"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dir := flag.String("data-dir", "./data/flags", "directory used by the file-backed store")
	flag.Parse()

	store, err := flags.NewFileStore(*dir)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           api.NewServer(store).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("feature-flags listening on %s (data dir: %s)", *addr, *dir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

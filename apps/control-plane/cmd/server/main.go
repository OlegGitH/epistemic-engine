package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/analysis"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/httpapi"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/service"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/store"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/verification"
)

func main() {
	addr := env("CONTROL_PLANE_ADDR", ":8080")
	mode := env("ANALYZER_MODE", "rules")
	var analyzer analysis.Analyzer = analysis.NewRulesAnalyzer()
	if mode == "openai" {
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			log.Fatal("OPENAI_API_KEY is required when ANALYZER_MODE=openai")
		}
		analyzer = analysis.NewOpenAIAnalyzer(key, env("OPENAI_MODEL", "gpt-5.6"))
	}
	var repo store.Repository = store.NewMemory()
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		postgres, err := store.NewPostgres(context.Background(), databaseURL)
		if err != nil {
			log.Fatalf("connect PostgreSQL: %v", err)
		}
		defer postgres.Close()
		repo = postgres
	}
	var executor verification.Runner = verification.DisabledRunner{}
	if env("EXECUTION_MODE", "recorded") == "docker" {
		images := strings.Split(env("VERIFICATION_IMAGES", "python:3.12-alpine"), ",")
		executor = verification.NewDockerRunner(env("VERIFICATION_ROOT", "../../demo"), images...)
	}
	svc := service.New(repo, analyzer, service.WithExecutor(executor))
	server := &http.Server{Addr: addr, Handler: httpapi.New(svc), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 0, IdleTimeout: 60 * time.Second}
	go func() {
		log.Printf("epistemic control plane listening on %s (analyzer=%s)", addr, mode)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

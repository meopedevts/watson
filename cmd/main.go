package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/meopedevts/watson/config"
	ghpkg "github.com/meopedevts/watson/internal/github"
	"github.com/meopedevts/watson/internal/reviewer"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "print reviews to stdout instead of posting on GitHub")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	cfg.DryRun = *dryRun

	executor := ghpkg.NewShellExecutor()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	username, err := ghpkg.ResolveCurrentUser(ctx, executor)
	if err != nil {
		log.Fatalf("resolve GitHub user: %v", err)
	}
	cfg.GitHubReviewerUsername = username

	rev := reviewer.NewReviewer(cfg, executor, logger)

	interval := time.Duration(cfg.PollIntervalMinutes) * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("watson started",
		"pollIntervalMinutes", cfg.PollIntervalMinutes,
		"claudeModel", cfg.ClaudeModel,
		"dryRun", cfg.DryRun,
		"reviewer", cfg.GitHubReviewerUsername,
	)

	// Process immediately on startup, then on each tick.
	if err := rev.ProcessPendingPRs(ctx); err != nil {
		logger.Error("tick failed", "err", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := rev.ProcessPendingPRs(ctx); err != nil {
				logger.Error("tick failed", "err", err)
			}
		case <-ctx.Done():
			logger.Info("shutting down")
			return
		}
	}
}

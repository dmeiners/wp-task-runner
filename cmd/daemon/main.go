package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/redis/go-redis/v9"

    "github.com/dmeiners/wp-task-runner/internal/config"
    "github.com/dmeiners/wp-task-runner/internal/runner"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Setup Valkey/Redis client
    redisClient := redis.NewClient(&redis.Options{
        Addr:     cfg.Valkey.Addr,
        Password: cfg.Valkey.Password,
        DB:       cfg.Valkey.DB,
    })
    defer redisClient.Close()

    // Create runner
    r := runner.New(cfg, redisClient)

    // Graceful shutdown
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    log.Printf("wp-task-runner daemon started. Listening on %d queues: %v", len(cfg.Queues), cfg.Queues)

    // Run the main loop
    if err := r.Start(ctx); err != nil && err != context.Canceled {
        log.Printf("Daemon stopped with error: %v", err)
    }

    log.Println("wp-task-runner daemon stopped gracefully")
}

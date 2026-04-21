package runner

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/redis/go-redis/v9"

    "github.com/dmeiners/wp-task-runner/internal/config"
    "github.com/dmeiners/wp-task-runner/internal/task"
)

type Runner struct {
    cfg         *config.Config
    redisClient *redis.Client
}

func New(cfg *config.Config, redisClient *redis.Client) *Runner {
    return &Runner{
        cfg:         cfg,
        redisClient: redisClient,
    }
}

func (r *Runner) Start(ctx context.Context) error {
    logger.Printf("wp-task-runner daemon started. Listening on %d queues: %v", 
        len(r.cfg.Queues), r.cfg.Queues)

    for {
        select {
        case <-ctx.Done():
            logger.Println("Received shutdown signal, stopping daemon...")
            return ctx.Err()

        default:
            // BRPOP with a reasonable timeout so we can check for shutdown signal periodically
            result, err := r.redisClient.BRPop(ctx, 5*time.Second, r.cfg.Queues...).Result()
            
            if err == redis.Nil {
                // Timeout reached - just loop and check for shutdown again
                continue
            }
            if err != nil {
                logger.Printf("BRPOP error: %v - retrying...", err)
                time.Sleep(1 * time.Second)
                continue
            }

            queueName := result[0]
            taskJSON := result[1]

            var t task.Task
            if err := json.Unmarshal([]byte(taskJSON), &t); err != nil {
                logger.Printf("Invalid JSON from queue %s: %v", queueName, err)
                continue
            }

            logger.Printf("Processing task from %s: %s for domain %s (request_id: %s)", 
                queueName, t.Action, t.Domain, t.RequestID)

            if err := r.executeTask(t); err != nil {
                logger.Printf("Task failed (request_id: %s): %v", t.RequestID, err)
            } else {
                logger.Printf("Task completed successfully (request_id: %s)", t.RequestID)
            }
        }
    }
}

func (r *Runner) executeTask(t task.Task) error {
    // Build full document root: /var/www/{domain}/public
    docRoot := filepath.Join(r.cfg.Paths.BasePath, t.Domain, r.cfg.Paths.DocumentFolder)

    log.Printf("Running WP-CLI with path: %s", docRoot)

    // Build WP-CLI arguments
    args := []string{"--path=" + docRoot}

    var cmdName string

    switch t.Action {
    case "plugin-install", "plugin-update":
        cmdName = "plugin"
        args = append(args, strings.TrimPrefix(t.Action, "plugin-"), t.Slug)
        if t.Version != "" {
            args = append(args, "--version="+t.Version)
        }

    case "theme-install", "theme-update":
        cmdName = "theme"
        args = append(args, strings.TrimPrefix(t.Action, "theme-"), t.Slug)
        if t.Version != "" {
            args = append(args, "--version="+t.Version)
        }

    case "core-update":
        cmdName = "core"
        args = append(args, "update")
        if t.Version != "" {
            args = append(args, "--version="+t.Version)
        }

    default:
        return fmt.Errorf("unknown action: %s", t.Action)
    }

    // Run WP-CLI directly as the current user (ubuntu) - NO SUDO
    cmd := exec.Command(r.cfg.WPCLI.Path, append([]string{cmdName}, args...)...)

    log.Printf("Running WP-CLI with command: %s", cmd)

    output, err := cmd.CombinedOutput()

    // Logging
    logLine := fmt.Sprintf("Domain: %s | Action: %s | Slug: %s | Version: %s\nOutput:\n%s",
        t.Domain, t.Action, t.Slug, t.Version, string(output))

    logFile := r.cfg.Logging.File
    if logFile == "" {
        logFile = "/var/log/wp-task-runner.log"
    }

    f, openErr := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if openErr == nil {
        fmt.Fprintf(f, "[%s] %s\n", time.Now().Format(time.RFC3339), logLine)
        f.Close()
    }

    return err
}

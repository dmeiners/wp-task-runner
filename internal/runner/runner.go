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
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        result, err := r.redisClient.BRPop(ctx, 0, r.cfg.Queues...).Result()
        if err != nil {
            if err == redis.Nil {
                continue
            }
            log.Printf("BRPOP error: %v - retrying in 1s", err)
            time.Sleep(1 * time.Second)
            continue
        }

        queueName := result[0]
        taskJSON := result[1]

        var t task.Task
        if err := json.Unmarshal([]byte(taskJSON), &t); err != nil {
            log.Printf("Invalid JSON from queue %s: %v", queueName, err)
            continue
        }

        log.Printf("Processing task from %s: %s for domain %s (request_id: %s)", queueName, t.Action, t.Domain, t.RequestID)

        if err := r.executeTask(t); err != nil {
            log.Printf("Task failed (request_id: %s): %v", t.RequestID, err)
        } else {
            log.Printf("Task completed successfully (request_id: %s)", t.RequestID)
        }
    }
}

func (r *Runner) executeTask(t task.Task) error {
    // Determine file owner
    owner, exists := r.cfg.DomainOwners[t.Domain]
    if !exists {
        owner = r.cfg.DomainOwners["default_owner"]
        if owner == "" {
            owner = "ubuntu"
        }
    }

    // Build full document root: /var/www/{domain}/public
    docRoot := filepath.Join(r.cfg.Paths.BasePath, t.Domain, r.cfg.Paths.DocumentFolder)

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

    // Full sudo command
    fullArgs := append([]string{"-u", owner, "--non-interactive", r.cfg.WPCLI.Path, cmdName}, args...)

    cmd := exec.Command("sudo", fullArgs...)
    output, err := cmd.CombinedOutput()

    // Logging
    logLine := fmt.Sprintf("Domain: %s | Action: %s | Slug: %s | Version: %s | Owner: %s\nOutput:\n%s",
        t.Domain, t.Action, t.Slug, t.Version, owner, string(output))

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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// WebhookPayload matches the POST /api/v1/webhook JSON schema.
type WebhookPayload struct {
	Service string                 `json:"service"`
	Action  string                 `json:"action"`
	Time    string                 `json:"time"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	containerName := os.Getenv("TOKPOINT_CONTAINER_NAME")
	if containerName == "" {
		containerName = "tokpoint"
	}

	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL = "http://127.0.0.1:8080/api/v1/webhook"
	}

	hookToken := os.Getenv("HOOK_TOKEN")
	if hookToken == "" {
		log.Fatal("[dockerwatch] HOOK_TOKEN env var is required")
	}

	serviceName := os.Getenv("TOKPOINT_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "Tokpoint"
	}

	log.Printf("[dockerwatch] watching container=%q service=%q webhook=%s", containerName, serviceName, webhookURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGTERM/SIGINT for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		log.Printf("[dockerwatch] received signal %v, shutting down", sig)
		cancel()
	}()

	// Connect to Docker daemon
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("[dockerwatch] failed to create Docker client: %v", err)
	}
	defer cli.Close()

	// Verify Docker connectivity
	if _, err := cli.Ping(ctx); err != nil {
		log.Fatalf("[dockerwatch] cannot connect to Docker daemon: %v (ensure /var/run/docker.sock is accessible)", err)
	}

	// Filter events for the specific container
	filterArgs := filters.NewArgs()
	filterArgs.Add("type", "container")
	filterArgs.Add("container", containerName)

	eventsCh, errsCh := cli.Events(ctx, events.ListOptions{
		Filters: filterArgs,
	})

	log.Printf("[dockerwatch] listening for Docker events on container %q", containerName)

	for {
		select {
		case <-ctx.Done():
			log.Println("[dockerwatch] context cancelled, exiting")
			return

		case err := <-errsCh:
			if err != nil && ctx.Err() == nil {
				log.Printf("[dockerwatch] Docker events error: %v, reconnecting in 5s", err)
				time.Sleep(5 * time.Second)
				eventsCh, errsCh = cli.Events(ctx, events.ListOptions{
					Filters: filterArgs,
				})
			}

		case event := <-eventsCh:
			action := mapDockerAction(event)
			if action == "" {
				continue // Irrelevant event
			}

			log.Printf("[dockerwatch] container event: action=%s status=%s", event.Action, action)

			meta := map[string]interface{}{
				"docker_action": string(event.Action),
				"container":     containerName,
				"source":        "docker-watcher",
			}

			payload := WebhookPayload{
				Service: serviceName,
				Action:  action,
				Time:    time.Now().UTC().Format(time.RFC3339),
				Meta:    meta,
			}

			if err := sendWebhook(webhookURL, hookToken, payload); err != nil {
				log.Printf("[dockerwatch] failed to send webhook: %v", err)
			} else {
				log.Printf("[dockerwatch] sent %s webhook for %s", action, serviceName)
			}
		}
	}
}

// mapDockerAction maps Docker container events to "up" or "down" actions.
func mapDockerAction(event events.Message) string {
	action := string(event.Action)

	// Handle health_status events
	if strings.HasPrefix(action, "health_status") {
		if strings.Contains(action, "healthy") && !strings.Contains(action, "unhealthy") {
			return "up"
		}
		if strings.Contains(action, "unhealthy") {
			return "down"
		}
		return ""
	}

	switch action {
	case "start", "unpause":
		return "up"
	case "die", "stop", "kill", "oom", "pause":
		return "down"
	default:
		return "" // Ignore create, attach, detach, etc.
	}
}

// sendWebhook POSTs the payload to the webhook endpoint with auth.
func sendWebhook(url, token string, payload WebhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hook-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
	}

	return nil
}

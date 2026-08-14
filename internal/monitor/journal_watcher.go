package monitor

import (
	"encoding/json"
	"log"
	"os/exec"
	"status-page/internal/db"
	"strings"
	"time"
)

type JournalEntry struct {
	Message     string `json:"MESSAGE"`
	SystemdUnit string `json:"_SYSTEMD_UNIT"`
}

type DockerEvent struct {
	Status string `json:"status"`
	Action string `json:"Action"`
	Actor  struct {
		Attributes struct {
			Name string `json:"name"`
		} `json:"Attributes"`
	} `json:"Actor"`
}

func StartJournalWatcher() {
	log.Println("[JOURNAL] Starting systemd and docker watchers...")
	
	go watchSystemd()
	go watchDocker()
}

func watchSystemd() {
	cmd := exec.Command("journalctl", "-f", "-o", "json", "--no-pager")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("[JOURNAL] Error getting stdout pipe:", err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Println("[JOURNAL] Error starting journalctl:", err)
		return
	}
	defer cmd.Wait()

	decoder := json.NewDecoder(stdout)
	for {
		var entry JournalEntry
		if err := decoder.Decode(&entry); err != nil {
			// Agar stream tugasa yoki xatolik bo'lsa, qaytamiz (yoki ignore)
			if err.Error() == "EOF" {
				break
			}
			continue
		}

		if entry.SystemdUnit != "" && strings.HasSuffix(entry.SystemdUnit, ".service") {
			msg := entry.Message
			
			if strings.HasPrefix(msg, "Started ") {
				logID, err := db.LogMaintenanceEvent("start", entry.SystemdUnit+" ishga tushirildi", entry.SystemdUnit)
				if err == nil {
					db.CompleteMaintenanceEvent(logID)
					notifyFrontend()
				}
			} else if strings.HasPrefix(msg, "Stopped ") {
				logID, err := db.LogMaintenanceEvent("stop", entry.SystemdUnit+" to'xtatildi", entry.SystemdUnit)
				if err == nil {
					db.CompleteMaintenanceEvent(logID)
					notifyFrontend()
				}
			} else if strings.Contains(msg, "restarted") || strings.Contains(msg, "Restarted") {
				logID, err := db.LogMaintenanceEvent("restart", entry.SystemdUnit+" qayta ishga tushirildi", entry.SystemdUnit)
				if err == nil {
					db.CompleteMaintenanceEvent(logID)
					notifyFrontend()
				}
			}
		}
	}
}

func watchDocker() {
	cmd := exec.Command("docker", "events", "--format", "{{json .}}", "--filter", "type=container")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("[DOCKER] Error getting stdout pipe:", err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Println("[DOCKER] Error starting docker events (maybe docker is not running?):", err)
		return
	}
	defer cmd.Wait()

	decoder := json.NewDecoder(stdout)
	for {
		var event DockerEvent
		if err := decoder.Decode(&event); err != nil {
			if err.Error() == "EOF" {
				break
			}
			continue
		}

		containerName := event.Actor.Attributes.Name
		if containerName == "" {
			continue
		}

		// Docker action: "start", "stop", "restart", "die"
		switch event.Action {
		case "start":
			logID, err := db.LogMaintenanceEvent("start", "Docker: "+containerName+" ishga tushirildi", containerName)
			if err == nil {
				db.CompleteMaintenanceEvent(logID)
				notifyFrontend()
			}
		case "stop", "die":
			logID, err := db.LogMaintenanceEvent("stop", "Docker: "+containerName+" to'xtatildi", containerName)
			if err == nil {
				db.CompleteMaintenanceEvent(logID)
				notifyFrontend()
			}
		case "restart":
			logID, err := db.LogMaintenanceEvent("restart", "Docker: "+containerName+" qayta ishga tushirildi", containerName)
			if err == nil {
				db.CompleteMaintenanceEvent(logID)
				notifyFrontend()
			}
		}
	}
}

// Global update trigger for websocket
var lastUpdate time.Time

func notifyFrontend() {
	if time.Since(lastUpdate) < 2*time.Second {
		return
	}
	lastUpdate = time.Now()
	
	if GlobalScheduler != nil && GlobalScheduler.OnUpdate != nil {
		GlobalScheduler.OnUpdate(nil) 
	}
}

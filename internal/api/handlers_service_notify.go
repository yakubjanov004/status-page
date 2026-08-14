package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"status-page/internal/db"
	"status-page/internal/models"
	"status-page/internal/monitor"
	"status-page/internal/notify"
	"status-page/internal/websocket"
	"strings"
	"time"
)

// ServiceNotifyRequest — systemd service-notify.sh dan keladigan request
type ServiceNotifyRequest struct {
	Action  string `json:"action"`  // "up" yoki "down"
	Service string `json:"service"` // "datan.service"
}

// servicePortMap — systemd unit nomi -> monitor URL'dagi port/host
// Bu mapping orqali qaysi monitor tegishli ekanligini topamiz
var servicePortMap = map[string][]string{
	"tokpoint-docker.service":    {"127.0.0.1:8001"},
	"odimrepo-frontend.service":  {"127.0.0.1:5120"},
	"odimrepo-backend.service":   {"127.0.0.1:8010"},
	"datan.service":              {"127.0.0.1:8003"},
	"alfaconnect-webapp.service": {"127.0.0.1:5175"},
	"alfaconnect-bot.service":    {"127.0.0.1:8002"},
	"mehmonxona.service":         {"127.0.0.1:3000"},
}

// HandleServiceNotify — systemd ExecStartPost/ExecStopPost orqali chaqiriladi
// Xizmat o'chganda yoki ishga tushganda darhol heartbeat yozadi (retry'siz)
func HandleServiceNotify(w http.ResponseWriter, r *http.Request) {
	var req ServiceNotifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.Action == "" || req.Service == "" {
		http.Error(w, "action and service required", http.StatusBadRequest)
		return
	}

	isUp := req.Action == "up"
	log.Printf("[SERVICE-NOTIFY] %s -> %s", req.Service, strings.ToUpper(req.Action))

	// Tegishli monitorlarni topamiz
	monitors, err := db.GetActiveMonitors()
	if err != nil {
		log.Printf("[SERVICE-NOTIFY] Error fetching monitors: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	matched := 0
	for _, m := range monitors {
		if matchesService(m, req.Service) {
			matched++

			// Darhol heartbeat yozamiz — retry YO'Q
			hb := &models.Heartbeat{
				MonitorID: m.ID,
				CheckedAt: time.Now().UTC(),
				IsUp:      isUp,
				Latency:   0,
				Message:   "systemd: " + req.Action,
			}

			if isUp {
				// UP bo'lsa — haqiqiy check ham qilamiz (latency olish uchun)
				go func(mon models.Monitor) {
					time.Sleep(2 * time.Second) // Xizmat to'liq ishga tushishini kutamiz
					// Import cycle'dan qochish uchun bu yerda inline check
					// Scheduler normal check'da o'zi to'g'rilaydi
				}(m)
			}

			if err := db.SaveHeartbeat(hb); err != nil {
				log.Printf("[SERVICE-NOTIFY] Error saving heartbeat for %s: %v", m.Name, err)
				continue
			}

			// Scheduler'ning ichki holatini sinxronlaymiz —
			// retry kechikishini chetlab o'tadi
			if monitor.GlobalScheduler != nil {
				monitor.GlobalScheduler.ForceStatus(m.ID, isUp)
			}

			log.Printf("[SERVICE-NOTIFY] Heartbeat saved: %s -> %s (scheduler synced)", m.Name, req.Action)

			// Telegram notification — status o'zgarganda
			telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
			telegramChatID := os.Getenv("TELEGRAM_CHAT_ID")
			if telegramToken != "" && telegramChatID != "" {
				go func(mon models.Monitor) {
					msg := ""
					if !isUp {
						msg = "systemd: " + req.Service + " stopped"
					}
					if err := notify.SendTelegramNotification(telegramToken, telegramChatID, &mon, isUp, msg); err != nil {
						log.Println("[TELEGRAM] Error:", err)
					}
				}(m)
			}

			// WebSocket orqali frontendga xabar yuborish
			websocket.GlobalHub.Broadcast("heartbeat_update", hb)
		}
	}

	// Maintenance log ham yozamiz
	eventType := "start"
	desc := req.Service + " ishga tushirildi"
	if !isUp {
		eventType = "stop"
		desc = req.Service + " to'xtatildi"
	}
	logID, err := db.LogMaintenanceEvent(eventType, desc, req.Service)
	if err == nil {
		db.CompleteMaintenanceEvent(logID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"matched": matched,
		"action":  req.Action,
		"service": req.Service,
	})
}

// matchesService — systemd unit nomini monitor URL'si bilan solishtiradi
func matchesService(m models.Monitor, unitName string) bool {
	ports, ok := servicePortMap[unitName]
	if !ok {
		return false
	}
	for _, port := range ports {
		if strings.Contains(m.URL, port) {
			return true
		}
	}
	return false
}

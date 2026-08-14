package main

import (
	"log"
	"net/http"

	"status-page/internal/api"
	"status-page/internal/config"
	"status-page/internal/db"
	"status-page/internal/discovery"
	"status-page/internal/models"
	"status-page/internal/monitor"
	"status-page/internal/websocket"
)

func main() {
	cfg := config.Load()

	if err := db.Init(cfg.DBPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Nginx va Systemd fayllaridan loyihalarni avtomatik qo'shadi
	discovery.AutoSeed()

	// Monitorlarni yoqamiz
	scheduler := monitor.InitScheduler()
	scheduler.OnUpdate = func(hb *models.Heartbeat) {
		websocket.GlobalHub.Broadcast("heartbeat_update", hb)
	}
	scheduler.StartAll()

	// Systemd jurnalini kuzatish (xizmatlar restart/stop bo'lishi)
	monitor.StartJournalWatcher()

	// Eski heartbeatlarni tozalash
	db.StartCleanupJob()

	// Router — faqat public API va WebSocket
	r := api.NewRouter(cfg)
	r.NotFound(api.SpaHandler("./web"))

	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

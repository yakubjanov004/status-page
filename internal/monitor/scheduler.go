package monitor

import (
	"log"
	"os"
	"status-page/internal/db"
	"status-page/internal/models"
	"status-page/internal/notify"
	"sync"
	"time"
)

type Scheduler struct {
	stopChans  map[int]chan struct{}
	lastStatus map[int]bool // Tracks the last known isUp state
	mu         sync.Mutex
	OnUpdate   func(hb *models.Heartbeat) // Callback for websockets/notifications
}

var GlobalScheduler *Scheduler

func InitScheduler() *Scheduler {
	s := &Scheduler{
		stopChans:  make(map[int]chan struct{}),
		lastStatus: make(map[int]bool),
	}
	GlobalScheduler = s
	return s
}

func (s *Scheduler) StartAll() {
	monitors, err := db.GetActiveMonitors()
	if err != nil {
		log.Println("Error fetching active monitors:", err)
		return
	}

	for _, m := range monitors {
		mCopy := m
		s.StartMonitor(&mCopy)
	}
}

func (s *Scheduler) StartMonitor(m *models.Monitor) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop existing if any
	if ch, exists := s.stopChans[m.ID]; exists {
		close(ch)
		delete(s.stopChans, m.ID)
	}

	if !m.IsActive {
		return
	}

	stopCh := make(chan struct{})
	s.stopChans[m.ID] = stopCh

	go func(monitor *models.Monitor, stop <-chan struct{}) {
		log.Printf("Starting scheduler for monitor %d: %s (interval: %ds)\n", monitor.ID, monitor.Name, monitor.IntervalSeconds)
		
		// Run immediately once
		s.runCheck(monitor)

		ticker := time.NewTicker(time.Duration(monitor.IntervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.runCheck(monitor)
			case <-stop:
				log.Printf("Stopping scheduler for monitor %d: %s\n", monitor.ID, monitor.Name)
				return
			}
		}
	}(m, stopCh)
}

func (s *Scheduler) StopMonitor(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, exists := s.stopChans[id]; exists {
		close(ch)
		delete(s.stopChans, id)
	}
}

func (s *Scheduler) runCheck(m *models.Monitor) {
	hb := Check(m)
	err := db.SaveHeartbeat(hb)
	if err != nil {
		log.Println("Failed to save heartbeat:", err)
	}

	s.mu.Lock()
	last, exists := s.lastStatus[m.ID]
	s.lastStatus[m.ID] = hb.IsUp
	s.mu.Unlock()

	// Notify if state changed
	if !exists || last != hb.IsUp {
		log.Printf("Monitor %d status changed to %v\n", m.ID, hb.IsUp)
		telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
		telegramChatID := os.Getenv("TELEGRAM_CHAT_ID")
		if telegramToken != "" && telegramChatID != "" {
			go func() {
				if err := notify.SendTelegramNotification(telegramToken, telegramChatID, m, hb.IsUp, hb.ErrorMessage); err != nil {
					log.Println("Failed to send telegram notification:", err)
				}
			}()
		}
	}

	if s.OnUpdate != nil {
		s.OnUpdate(hb)
	}
}

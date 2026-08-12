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

const maxRetries = 3 // Kuma kabi — 3 marta qayta urinadi

type Scheduler struct {
	stopChans  map[int]chan struct{}
	lastStatus map[int]bool
	retries    map[int]int // Retry counter per monitor
	mu         sync.Mutex
	OnUpdate   func(hb *models.Heartbeat)
}

var GlobalScheduler *Scheduler

func InitScheduler() *Scheduler {
	s := &Scheduler{
		stopChans:  make(map[int]chan struct{}),
		lastStatus: make(map[int]bool),
		retries:    make(map[int]int),
	}
	GlobalScheduler = s
	return s
}

func (s *Scheduler) StartAll() {
	monitors, err := db.GetActiveMonitors()
	if err != nil {
		log.Println("[SCHEDULER] Error fetching monitors:", err)
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
		log.Printf("[SCHEDULER] Monitor %d: %s started (every %ds)", monitor.ID, monitor.Name, monitor.IntervalSeconds)

		// Birinchi check — darhol
		s.runCheck(monitor)

		interval := time.Duration(monitor.IntervalSeconds) * time.Second
		if interval < 20*time.Second {
			interval = 20 * time.Second
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.runCheck(monitor)
			case <-stop:
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

	// Retry logikasi (Kuma kabi)
	s.mu.Lock()
	retryCount := s.retries[m.ID]

	if !hb.IsUp {
		// DOWN — retry qilamiz
		if retryCount < maxRetries {
			s.retries[m.ID] = retryCount + 1
			s.mu.Unlock()
			// Retry paytida heartbeat saqlamaymiz — faqat log
			log.Printf("[RETRY] Monitor %d: %s attempt %d/%d", m.ID, m.Name, retryCount+1, maxRetries)
			return
		}
		// maxRetries ga yetdi — haqiqatan DOWN
	} else {
		// UP — retry counter reset
		s.retries[m.ID] = 0
	}

	last, exists := s.lastStatus[m.ID]
	s.lastStatus[m.ID] = hb.IsUp
	s.mu.Unlock()

	// Heartbeat saqlash
	if err := db.SaveHeartbeat(hb); err != nil {
		log.Printf("[SCHEDULER] Failed to save heartbeat for %s: %v", m.Name, err)
	}

	// Status o'zgardimi?
	if !exists || last != hb.IsUp {
		statusStr := "UP ✅"
		if !hb.IsUp {
			statusStr = "DOWN ❌"
		}
		log.Printf("[STATUS CHANGE] %s → %s", m.Name, statusStr)

		// Telegram notification
		telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
		telegramChatID := os.Getenv("TELEGRAM_CHAT_ID")
		if telegramToken != "" && telegramChatID != "" {
			go func() {
				if err := notify.SendTelegramNotification(telegramToken, telegramChatID, m, hb.IsUp, hb.Message); err != nil {
					log.Println("[TELEGRAM] Error:", err)
				}
			}()
		}
	}

	if s.OnUpdate != nil {
		s.OnUpdate(hb)
	}
}

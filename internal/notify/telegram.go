package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"status-page/internal/models"
)

// In a real scenario, this would be read from the DB notifications table.
// For simplicity, we just take it from environment variables or a basic config.
func SendTelegramNotification(token, chatID string, m *models.Monitor, isUp bool, errMsg string) error {
	if token == "" || chatID == "" {
		return nil // Not configured
	}

	status := "✅ UP"
	if !isUp {
		status = "❌ DOWN"
	}

	text := fmt.Sprintf("Monitor Status Changed!\n\nName: %s\nURL: %s\nStatus: %s\nError: %s", m.Name, m.URL, status, errMsg)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body, _ := json.Marshal(map[string]string{
		"chat_id": chatID,
		"text":    text,
	})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send telegram message, status: %d", resp.StatusCode)
	}

	return nil
}

package db

import (
	"database/sql"
	"status-page/internal/models"
	"strconv"
	"strings"
	"time"
)

func GetAllMonitors() ([]models.Monitor, error) {
	rows, err := DB.Query(`SELECT id, name, type, url, interval_seconds, timeout_seconds, expected_status_code, is_active, show_on_public_page, project_id, component_type, created_at FROM monitors`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []models.Monitor
	for rows.Next() {
		var m models.Monitor
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.URL, &m.IntervalSeconds, &m.TimeoutSeconds, &m.ExpectedStatusCode, &m.IsActive, &m.ShowOnPublicPage, &m.ProjectID, &m.ComponentType, &m.CreatedAt); err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return monitors, nil
}

func GetActiveMonitors() ([]models.Monitor, error) {
	rows, err := DB.Query(`SELECT id, name, type, url, interval_seconds, timeout_seconds, expected_status_code, is_active, show_on_public_page, project_id, component_type, created_at FROM monitors WHERE is_active = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []models.Monitor
	for rows.Next() {
		var m models.Monitor
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.URL, &m.IntervalSeconds, &m.TimeoutSeconds, &m.ExpectedStatusCode, &m.IsActive, &m.ShowOnPublicPage, &m.ProjectID, &m.ComponentType, &m.CreatedAt); err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return monitors, nil
}

func SaveHeartbeat(hb *models.Heartbeat) error {
	res, err := DB.Exec(`INSERT INTO heartbeats (monitor_id, is_up, response_time_ms, status_code, error_message, checked_at) VALUES (?, ?, ?, ?, ?, ?)`,
		hb.MonitorID, hb.IsUp, hb.Latency, hb.StatusCode, hb.Message, hb.CheckedAt)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		hb.ID = int(id)
	}
	return nil
}

// Additional CRUD operations for Monitor can go here...
func CreateMonitor(m *models.Monitor) error {
	res, err := DB.Exec(`INSERT INTO monitors (name, type, url, interval_seconds, timeout_seconds, expected_status_code, is_active, show_on_public_page, project_id, component_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.Name, m.Type, m.URL, m.IntervalSeconds, m.TimeoutSeconds, m.ExpectedStatusCode, m.IsActive, m.ShowOnPublicPage, m.ProjectID, m.ComponentType)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		m.ID = int(id)
	}
	return nil
}

func UpdateMonitor(m *models.Monitor) error {
	_, err := DB.Exec(`UPDATE monitors SET name = ?, type = ?, url = ?, interval_seconds = ?, timeout_seconds = ?, expected_status_code = ?, is_active = ?, show_on_public_page = ?, project_id = ?, component_type = ? WHERE id = ?`,
		m.Name, m.Type, m.URL, m.IntervalSeconds, m.TimeoutSeconds, m.ExpectedStatusCode, m.IsActive, m.ShowOnPublicPage, m.ProjectID, m.ComponentType, m.ID)
	return err
}

func DeleteMonitor(id int) error {
	_, err := DB.Exec(`DELETE FROM heartbeats WHERE monitor_id = ?`, id)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`DELETE FROM monitors WHERE id = ?`, id)
	return err
}

func GetRecentHeartbeats(monitorID int, limit int) ([]models.Heartbeat, error) {
	rows, err := DB.Query(`SELECT id, monitor_id, is_up, response_time_ms, status_code, error_message, checked_at FROM heartbeats WHERE monitor_id = ? ORDER BY checked_at DESC LIMIT ?`, monitorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hbs []models.Heartbeat
	for rows.Next() {
		var hb models.Heartbeat
		var errMsg sql.NullString
		if err := rows.Scan(&hb.ID, &hb.MonitorID, &hb.IsUp, &hb.Latency, &hb.StatusCode, &errMsg, &hb.CheckedAt); err != nil {
			return nil, err
		}
		if errMsg.Valid {
			hb.Message = errMsg.String
		}
		hbs = append(hbs, hb)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hbs, nil
}

func LogMaintenanceEvent(eventType, description, serviceName string) (int, error) {
	res, err := DB.Exec(`INSERT INTO maintenance_log (event_type, description, service_name) VALUES (?, ?, ?)`, eventType, description, serviceName)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func CompleteMaintenanceEvent(id int) error {
	_, err := DB.Exec(`
		UPDATE maintenance_log 
		SET ended_at = CURRENT_TIMESTAMP, 
		    duration_seconds = CAST((julianday(CURRENT_TIMESTAMP) - julianday(started_at)) * 86400 AS INTEGER)
		WHERE id = ?`, id)
	return err
}

// NormalizeServiceKey — systemd unit ("datan-backend.service"), docker
// konteyner nomi ("datan_backend_1") va monitor asosida qurilgan kalitni
// ("datan-backend") bir xil formatga keltiradi, shunda ularni solishtirish
// mumkin bo'ladi.
func NormalizeServiceKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "docker: ")
	s = strings.TrimSuffix(s, ".service")
	s = strings.NewReplacer("_", "-", ".", "-").Replace(s)

	// Docker compose konteynerlar oxiriga "-1", "-2" kabi raqam qo'shishi mumkin —
	// buni olib tashlaymiz.
	for {
		idx := strings.LastIndex(s, "-")
		if idx == -1 || idx == len(s)-1 {
			break
		}
		if _, err := strconv.Atoi(s[idx+1:]); err == nil {
			s = s[:idx]
			continue
		}
		break
	}
	return s
}

// RestartStat — bitta xizmat (service) uchun restart soni va oxirgi marta
// qachon restart qilingani.
type RestartStat struct {
	Count  int
	LastAt time.Time
}

// GetRestartStats — maintenance_log jadvalidagi barcha "restart" turidagi
// hodisalarni normalized service_name bo'yicha guruhlab, har biri uchun
// sonini va oxirgi vaqtini qaytaradi.
func GetRestartStats() (map[string]RestartStat, error) {
	rows, err := DB.Query(`SELECT service_name, started_at FROM maintenance_log WHERE event_type = 'restart' AND service_name != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]RestartStat)
	for rows.Next() {
		var serviceName string
		var startedAt time.Time
		if err := rows.Scan(&serviceName, &startedAt); err != nil {
			continue
		}
		key := NormalizeServiceKey(serviceName)
		if key == "" {
			continue
		}
		st := stats[key]
		st.Count++
		if startedAt.After(st.LastAt) {
			st.LastAt = startedAt
		}
		stats[key] = st
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

func GetRecentMaintenanceLogs(limit int) ([]models.MaintenanceLog, error) {
	rows, err := DB.Query(`SELECT id, event_type, description, service_name, started_at, ended_at, duration_seconds FROM maintenance_log ORDER BY started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.MaintenanceLog
	for rows.Next() {
		var log models.MaintenanceLog
		var endedAt sql.NullTime
		var durationSecs sql.NullInt64
		if err := rows.Scan(&log.ID, &log.EventType, &log.Description, &log.ServiceName, &log.StartedAt, &endedAt, &durationSecs); err != nil {
			return nil, err
		}
		if endedAt.Valid {
			log.EndedAt = &endedAt.Time
		}
		if durationSecs.Valid {
			dur := int(durationSecs.Int64)
			log.DurationSeconds = &dur
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}

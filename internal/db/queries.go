package db

import (
	"database/sql"
	"status-page/internal/models"
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
	return monitors, nil
}

func SaveHeartbeat(hb *models.Heartbeat) error {
	res, err := DB.Exec(`INSERT INTO heartbeats (monitor_id, is_up, response_time_ms, status_code, error_message, checked_at) VALUES (?, ?, ?, ?, ?, ?)`,
		hb.MonitorID, hb.IsUp, hb.ResponseTimeMs, hb.StatusCode, hb.ErrorMessage, hb.CheckedAt)
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
		// error_message could be null, need to handle
		var errMsg sql.NullString
		if err := rows.Scan(&hb.ID, &hb.MonitorID, &hb.IsUp, &hb.ResponseTimeMs, &hb.StatusCode, &errMsg, &hb.CheckedAt); err != nil {
			return nil, err
		}
		if errMsg.Valid {
			hb.ErrorMessage = errMsg.String
		}
		hbs = append(hbs, hb)
	}
	
	// Reverse to get chronological order if needed, but usually we want to send them as is and reverse on frontend or here
	return hbs, nil
}

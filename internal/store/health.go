package store

import "database/sql"

type HealthInfo struct {
	OpenConns    int    `json:"open_conns"`
	IdleConns    int    `json:"idle_conns"`
	Status       string `json:"status"`
	InUse        int    `json:"in_use"`
	WaitDuration string `json:"wait_duration"`
}

type Health struct {
	db *sql.DB
}

func (h *Health) Get() (*HealthInfo, error) {
	info := &HealthInfo{
		Status: "up",
	}
	if err := h.db.Ping(); err != nil {
		info.Status = "down"
		return info, err
	}
	stats := h.db.Stats()

	info.OpenConns = stats.OpenConnections
	info.IdleConns = stats.Idle
	info.InUse = stats.InUse
	info.WaitDuration = stats.WaitDuration.String()
	return info, nil
}

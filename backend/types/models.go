package types

import "time"

type Device struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Status   string    `json:"status"`
	Battery  int       `json:"battery"`
	LastSeen time.Time `json:"last_seen"`
}

type DeviceMessage struct {
	Type      string `json:"type"`
	Battery   int    `json:"battery,omitempty"`
	Status    string `json:"status,omitempty"`
	CommandID string `json:"command_id,omitempty"`
}

type CommandRequest struct {
	Type string `json:"type"`
}

type EnrollRequest struct {
	DeviceID string `json:"id"`
	Name     string `json:"name"`
}

type EnrollResponse struct {
	Token string `json:"token"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type Command struct {
	ID            string     `json:"id"`
	DeviceID      string     `json:"device_id"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	DeliveredAt   *time.Time `json:"delivered_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	FailureReason *string    `json:"failure_reason,omitempty"`
}

type CommandMetrics struct {
	Total               int64   `json:"total"`
	Active              int64   `json:"active"`
	Completed           int64   `json:"completed"`
	Failed              int64   `json:"failed"`
	AverageCompletionMS float64 `json:"average_completion_ms"`
}

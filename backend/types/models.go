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
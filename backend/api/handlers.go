package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"MDM-matrix/hub"
	"MDM-matrix/types"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func HandleEnroll(dbPool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req types.EnrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DeviceID == "" {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		token, err := generateToken()
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		query := `INSERT INTO devices (id, name, token) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET token = EXCLUDED.token`
		_, err = dbPool.Exec(context.Background(), query, req.DeviceID, req.Name, token)
		if err != nil {
			http.Error(w, "Failed to register device", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.EnrollResponse{Token: token})
	}
}

func HandleLogin(dbPool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req types.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		var hash string
		err := dbPool.QueryRow(context.Background(), "SELECT password_hash FROM admin_user WHERE username = $1", req.Username).Scan(&hash)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": req.Username,
			"exp":      time.Now().Add(24 * time.Hour).Unix(),
		})

		tokenString, err := token.SignedString(jwtSecret)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.LoginResponse{Token: tokenString})
	}
}

func HandleGetDevices(dbPool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rows, err := dbPool.Query(context.Background(), "SELECT id, name, status, battery, last_seen FROM devices ORDER BY id ASC")
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		devices := []types.Device{}
		for rows.Next() {
			var d types.Device
			if err := rows.Scan(&d.ID, &d.Name, &d.Status, &d.Battery, &d.LastSeen); err == nil {
				devices = append(devices, d)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(devices)
	}
}

func HandleDispatchCommand(dbPool *pgxpool.Pool, h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		parts := strings.Split(r.URL.Path, "/")
		if len(parts) != 4 || parts[1] != "devices" || parts[3] != "command" {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		deviceID := parts[2]

		var req types.CommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Type == "" {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		var cmdID string
		err := dbPool.QueryRow(context.Background(),
			"INSERT INTO commands (device_id, type) VALUES ($1, $2) RETURNING id",
			deviceID, req.Type).Scan(&cmdID)
		if err != nil {
			http.Error(w, "Failed to save command", http.StatusInternalServerError)
			return
		}

		h.Mu.RLock()
		conn, isOnline := h.Conns[deviceID]
		h.Mu.RUnlock()

		if isOnline {
			wsMsg := map[string]interface{}{"type": "command", "command_id": cmdID, "action": req.Type}
			if err := conn.WriteJSON(wsMsg); err == nil {
				log.Printf("Dispatched '%s' command to %s", req.Type, deviceID)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"command_id": cmdID, "status": "dispatched_or_queued"})
	}
}

// UPDATED: Now accepts adminHub to broadcast changes
func HandleWS(dbPool *pgxpool.Pool, h *hub.Hub, adminHub *hub.AdminHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		var deviceID string
		if err := dbPool.QueryRow(context.Background(), "SELECT id FROM devices WHERE token = $1", token).Scan(&deviceID); err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		conn, err := hub.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		h.Add(deviceID, conn)
		
		// Broadcast that the device came online
		adminHub.Broadcast(map[string]interface{}{
			"event": "device_update",
			"device_id": deviceID,
			"status": "online",
		})

		defer func() {
			h.Remove(deviceID)
			dbPool.Exec(context.Background(), "UPDATE devices SET status = 'offline' WHERE id = $1", deviceID)
			
			// Broadcast that the device went offline
			adminHub.Broadcast(map[string]interface{}{
				"event": "device_update",
				"device_id": deviceID,
				"status": "offline",
			})
		}()

		// Fetch pending commands...
		rows, err := dbPool.Query(context.Background(), "SELECT id, type FROM commands WHERE device_id = $1 AND status = 'pending' ORDER BY created_at ASC", deviceID)
		if err == nil {
			for rows.Next() {
				var cmdID, cmdType string
				if err := rows.Scan(&cmdID, &cmdType); err == nil {
					wsMsg := map[string]interface{}{"type": "command", "command_id": cmdID, "action": cmdType}
					conn.WriteJSON(wsMsg)
				}
			}
			rows.Close()
		}

		const pongWait = 15 * time.Second
		conn.SetReadDeadline(time.Now().Add(pongWait))

		for {
			var msg types.DeviceMessage
			if err := conn.ReadJSON(&msg); err != nil {
				break
			}

			if msg.Type == "heartbeat" {
				conn.SetReadDeadline(time.Now().Add(pongWait))
				dbPool.Exec(context.Background(), "UPDATE devices SET status = $1, battery = $2, last_seen = NOW() WHERE id = $3", msg.Status, msg.Battery, deviceID)
				
				// Broadcast heartbeat data instantly
				adminHub.Broadcast(map[string]interface{}{
					"event": "device_update",
					"device_id": deviceID,
					"status": msg.Status,
					"battery": msg.Battery,
				})

			} else if msg.Type == "ack" && msg.CommandID != "" {
				conn.SetReadDeadline(time.Now().Add(pongWait))
				dbPool.Exec(context.Background(), "UPDATE commands SET status = 'completed', completed_at = NOW() WHERE id = $1", msg.CommandID)
				
				// Broadcast that a command finished
				adminHub.Broadcast(map[string]interface{}{
					"event": "command_completed",
					"device_id": deviceID,
					"command_id": msg.CommandID,
				})
			}
		}
	}
}

// HandleAdminWS manages live connections to the React dashboard
func HandleAdminWS(adminHub *hub.AdminHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Authenticate via query string token
		tokenString := r.URL.Query().Get("token")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// 2. Upgrade connection
		conn, err := hub.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Admin WS upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		adminHub.Add(conn)
		defer adminHub.Remove(conn)

		// 3. Keep connection alive and wait for disconnect
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}
}
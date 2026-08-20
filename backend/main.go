package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time" // New

	"github.com/golang-jwt/jwt/v5" // New
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt" // New
)

var jwtSecret = []byte("super-secret-mdm-key-change-me")

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}
// EnrollRequest represents the JSON payload sent by the device
type EnrollRequest struct {
	DeviceID string `json:"id"`
	Name     string `json:"name"`
}

// EnrollResponse represents the JSON we send back
type EnrollResponse struct {
	Token string `json:"token"`
}

// Device represents the device state sent to the React dashboard
type Device struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Status   string    `json:"status"`
	Battery  int       `json:"battery"`
	LastSeen time.Time `json:"last_seen"`
}
// generateToken creates a cryptographically secure 32-character hex string
func generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// handleEnroll processes the device enrollment
func handleEnroll(dbPool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only allow POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Stream-decode the JSON request directly into our struct (O(1) auxiliary space)
		var req EnrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DeviceID == "" {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// Generate a new secure token
		token, err := generateToken()
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Insert or update the device in the database.
		// Using ON CONFLICT handles the case where a device re-enrolls.
		// This uses the primary key index, meaning the operation is O(log N) time complexity.
		query := `
			INSERT INTO devices (id, name, token) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (id) DO UPDATE SET token = EXCLUDED.token
		`
		
		_, err = dbPool.Exec(context.Background(), query, req.DeviceID, req.Name, token)
		if err != nil {
			log.Printf("Database error: %v", err)
			http.Error(w, "Failed to register device", http.StatusInternalServerError)
			return
		}

		// Send the token back to the device as JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(EnrollResponse{Token: token})
	}
}

// Upgrader upgrades standard HTTP connections to WebSockets
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// Hub maintains the set of active device connections
type Hub struct {
	mu    sync.RWMutex
	Conns map[string]*websocket.Conn
}

func NewHub() *Hub {
	return &Hub{
		Conns: make(map[string]*websocket.Conn),
	}
}

func (h *Hub) Add(deviceID string, conn *websocket.Conn) {
	h.mu.Lock() // Exclusive lock for writing
	defer h.mu.Unlock()
	h.Conns[deviceID] = conn
}

func (h *Hub) Remove(deviceID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.Conns, deviceID)
}

// DeviceMessage represents incoming data from the device
type DeviceMessage struct {
	Type    string `json:"type"`
	Battery int    `json:"battery"`
	Status  string `json:"status"`
	CommandID string `json:"command_id,omitempty"`
}
type CommandRequest struct {
	Type string `json:"type"` // e.g., "lock", "wipe", "update_policy"
}

func handleWS(dbPool *pgxpool.Pool, hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract token from URL query parameters
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		// 2. Validate token and get device ID
		var deviceID string
		err := dbPool.QueryRow(context.Background(), "SELECT id FROM devices WHERE token = $1", token).Scan(&deviceID)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// 3. Upgrade HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		// 4. Register the device in the Hub
		hub.Add(deviceID, conn)
		log.Printf("Device %s connected", deviceID)

		// Ensure cleanup when the loop exits (device disconnects)
		defer func() {
			hub.Remove(deviceID)
			// Mark device offline in DB
			dbPool.Exec(context.Background(), "UPDATE devices SET status = 'offline' WHERE id = $1", deviceID)
			log.Printf("Device %s disconnected", deviceID)
		}()

		rows, err := dbPool.Query(context.Background(),
			"SELECT id, type FROM commands WHERE device_id = $1 AND status = 'pending' ORDER BY created_at ASC",
			deviceID)
		
		if err != nil {
			log.Printf("Failed to fetch pending commands for %s: %v", deviceID, err)
		} else {
			// Iterate through the rows and dispatch them
			for rows.Next() {
				var cmdID, cmdType string
				if err := rows.Scan(&cmdID, &cmdType); err != nil {
					log.Printf("Error scanning pending command: %v", err)
					continue
				}

				wsMsg := map[string]interface{}{
					"type":       "command",
					"command_id": cmdID,
					"action":     cmdType,
				}
				
				if err := conn.WriteJSON(wsMsg); err != nil {
					log.Printf("Failed to push queued command %s: %v", cmdID, err)
					break // Stop processing if the connection broke
				}
				log.Printf("Pushed queued command '%s' to %s", cmdType, deviceID)
			}
			rows.Close() // Free up database resources (O(1) space cleanup)
		}

		// 5. Read loop: Listen for incoming messages (Heartbeats)
		// 5. Read loop: Listen for incoming messages (Heartbeats & Acks)
		for {
			var msg DeviceMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				// Client disconnected or sent invalid data
				break 
			}

			if msg.Type == "heartbeat" {
				_, dbErr := dbPool.Exec(context.Background(), 
					"UPDATE devices SET status = $1, battery = $2, last_seen = NOW() WHERE id = $3",
					msg.Status, msg.Battery, deviceID)
				if dbErr != nil {
					log.Printf("Failed to update heartbeat for %s: %v", deviceID, dbErr)
				}
			} else if msg.Type == "ack" && msg.CommandID != "" {
				// Device finished the command, update database
				_, dbErr := dbPool.Exec(context.Background(),
					"UPDATE commands SET status = 'completed', completed_at = NOW() WHERE id = $1",
					msg.CommandID)
				if dbErr != nil {
					log.Printf("Failed to update command ack for %s: %v", msg.CommandID, dbErr)
				} else {
					log.Printf("Command %s successfully completed by device %s", msg.CommandID, deviceID)
				}
			}
		}
	}
}

// handleDispatchCommand triggers a command to a specific device
func handleDispatchCommand(dbPool *pgxpool.Pool, hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract device ID from URL path (Format: /devices/<id>/command)
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) != 4 || parts[1] != "devices" || parts[3] != "command" {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		deviceID := parts[2]

		var req CommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Type == "" {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// 1. Save command to DB (Status defaults to 'pending')
		var cmdID string
		err := dbPool.QueryRow(context.Background(),
			"INSERT INTO commands (device_id, type) VALUES ($1, $2) RETURNING id",
			deviceID, req.Type).Scan(&cmdID)
		
		if err != nil {
			log.Printf("Database error: %v", err)
			http.Error(w, "Failed to save command", http.StatusInternalServerError)
			return
		}

		// 2. Check if device is currently connected to the Hub
		hub.mu.RLock()
		conn, isOnline := hub.Conns[deviceID]
		hub.mu.RUnlock()

		if isOnline {
			// 3. Device is online, push command instantly over WebSocket
			wsMsg := map[string]interface{}{
				"type":       "command",
				"command_id": cmdID,
				"action":     req.Type,
			}
			
			if err := conn.WriteJSON(wsMsg); err != nil {
				log.Printf("Failed to push command to %s: %v", deviceID, err)
			} else {
				log.Printf("Dispatched '%s' command to %s", req.Type, deviceID)
			}
		} else {
			// Device is offline. It stays 'pending' in the database (Phase 4 will handle this)
			log.Printf("Device %s is offline. Command '%s' queued.", deviceID, req.Type)
		}

		// Return success to the admin dashboard
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"command_id": cmdID,
			"status":     "dispatched_or_queued",
		})
	}
}


// handleLogin verifies credentials and issues a JWT
func handleLogin(dbPool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Fetch the password hash from the database
		var hash string
		err := dbPool.QueryRow(context.Background(), 
			"SELECT password_hash FROM admin_user WHERE username = $1", req.Username).Scan(&hash)
		
		if err != nil {
			log.Printf("Login DB Error: %v", err)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Compare the provided password with the hash (Time complexity: bounded by bcrypt cost factor)
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
			log.Printf("Bcrypt Mismatch: %v", err)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Generate the JWT
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
		json.NewEncoder(w).Encode(LoginResponse{Token: tokenString})
	}
}

// authMiddleware protects routes by requiring a valid JWT
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Format expected: "Bearer <token>"
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		
		// Parse and validate the token mathematically
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Token is valid, proceed to the actual handler
		next(w, r)
	}
}

// corsMiddleware allows the React app to communicate with the Go API
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // In production, restrict this to your Vercel URL
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests from the browser
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// handleGetDevices fetches the fleet status for the dashboard
func handleGetDevices(dbPool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rows, err := dbPool.Query(context.Background(), "SELECT id, name, status, battery, last_seen FROM devices ORDER BY id ASC")
		if err != nil {
			log.Printf("Error fetching devices: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close() // Crucial: prevents memory leaks (O(1) space cleanup)

		devices := []Device{}
		for rows.Next() {
			var d Device
			if err := rows.Scan(&d.ID, &d.Name, &d.Status, &d.Battery, &d.LastSeen); err == nil {
				devices = append(devices, d)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(devices)
	}
}

func main() {
	// 1. Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// 2. Initialize Database Connection Pool
	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}
	defer dbPool.Close()

	// 3. Verify connection
	err = dbPool.Ping(context.Background())
	if err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}
	fmt.Println("Connected to Supabase successfully!")

	hub := NewHub()
	// 4. Start a basic HTTP server
	http.HandleFunc("/health", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Server is healthy"))
	}))

	http.HandleFunc("/enroll", corsMiddleware(handleEnroll(dbPool)))
	http.HandleFunc("/ws", corsMiddleware(handleWS(dbPool, hub)))
	
	http.HandleFunc("/login", corsMiddleware(handleLogin(dbPool)))
	
	// NEW: Endpoint to list all devices (Protected by Auth AND CORS)
	http.HandleFunc("/devices", corsMiddleware(authMiddleware(handleGetDevices(dbPool)))) 
	
	// UPDATED: Dispatch command route (Protected by Auth AND CORS)
	http.HandleFunc("/devices/", corsMiddleware(authMiddleware(handleDispatchCommand(dbPool, hub))))

	port := ":8080"
	fmt.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}

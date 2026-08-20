package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type IncomingMsg struct {
	Type      string `json:"type"`
	CommandID string `json:"command_id"`
	Action    string `json:"action"`
}

type EnrollResponse struct {
	Token string `json:"token"`
}

func main() {
	fleetSize := 5
	var wg sync.WaitGroup

	for i := 1; i <= fleetSize; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			deviceID := fmt.Sprintf("fleet-device-%03d", i)
			deviceName := fmt.Sprintf("Simulated Device %d", i)
			startDevice(deviceID, deviceName)
		}(i)
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("🚀 Fleet of %d devices started! Press Ctrl+C to stop.", fleetSize)
	wg.Wait()
}

func startDevice(deviceID, name string) {
	token := enrollDevice(deviceID, name)
	if token == "" {
		log.Printf("[%s] ❌ Failed to enroll", deviceID)
		return
	}
	log.Printf("[%s] ✅ Enrolled successfully", deviceID)

	// Backoff configuration
	baseBackoff := 1 * time.Second
	maxBackoff := 30 * time.Second
	currentBackoff := baseBackoff

	// Infinite loop to keep the device alive forever
	for {
		startTime := time.Now()
		err := connectAndRun(deviceID, token)
		log.Printf("[%s] ❌ Disconnected: %v", deviceID, err)

		if time.Since(startTime) > 5*time.Second {
			currentBackoff = baseBackoff
		}

		// Calculate backoff with jitter (O(1) time complexity)
		jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
		sleepTime := currentBackoff + jitter

		log.Printf("[%s] 🔄 Retrying in %v...", deviceID, sleepTime)
		time.Sleep(sleepTime)

		// Exponentially increase backoff for the next potential failure
		currentBackoff *= 2
		if currentBackoff > maxBackoff {
			currentBackoff = maxBackoff
		}
	}
}

// connectAndRun handles the actual active connection. 
// It returns an error if the connection drops, triggering the backoff loop.
func connectAndRun(deviceID, token string) error {
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws", RawQuery: "token=" + token}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	defer c.Close()
	log.Printf("🔌 [%s] Connected to server!", deviceID)

	var writeMutex sync.Mutex
	processedCmds := make(map[string]bool)
	var cacheMutex sync.RWMutex

	// Channel to signal when the read loop crashes
	done := make(chan error, 1)

	go func() {
		for {
			var msg IncomingMsg
			err := c.ReadJSON(&msg)
			if err != nil {
				done <- err // Signal that the connection broke
				return
			}
			
			if msg.Type == "command" {
				cacheMutex.RLock()
				alreadyProcessed := processedCmds[msg.CommandID]
				cacheMutex.RUnlock()

				if alreadyProcessed {
					ackMsg := map[string]interface{}{"type": "ack", "command_id": msg.CommandID}
					writeMutex.Lock()
					c.WriteJSON(ackMsg)
					writeMutex.Unlock()
					continue
				}

				cacheMutex.Lock()
				processedCmds[msg.CommandID] = true
				cacheMutex.Unlock()

				log.Printf("📥 [%s] Received command [%s]", deviceID, msg.Action)
				time.Sleep(2 * time.Second) 
				log.Printf("✅ [%s] Executed [%s]. Sending ack...", deviceID, msg.Action)
				
				ackMsg := map[string]interface{}{"type": "ack", "command_id": msg.CommandID}
				writeMutex.Lock()
				c.WriteJSON(ackMsg)
				writeMutex.Unlock()
			}
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Select blocks until either the ticker fires OR the read loop sends an error
	for {
		select {
		case err := <-done:
			return err // Bubbles the error up to trigger backoff
		case <-ticker.C:
			msg := map[string]interface{}{"type": "heartbeat", "status": "online", "battery": 85}
			writeMutex.Lock()
			err := c.WriteJSON(msg)
			writeMutex.Unlock()
			if err != nil {
				return err
			}
		}
	}
}

func enrollDevice(id, name string) string {
	reqBody, _ := json.Marshal(map[string]string{"id": id, "name": name})
	resp, err := http.Post("http://localhost:8080/enroll", "application/json", bytes.NewBuffer(reqBody))
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()
	var enrollResp EnrollResponse
	json.NewDecoder(resp.Body).Decode(&enrollResp)
	return enrollResp.Token
}
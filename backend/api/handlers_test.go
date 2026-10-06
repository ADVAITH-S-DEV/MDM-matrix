package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"MDM-matrix/hub"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type fakeRow struct{ scan func(...interface{}) error }

func (row fakeRow) Scan(dest ...interface{}) error { return row.scan(dest...) }

type fakeDatabase struct {
	row       pgx.Row
	execCalls []string
}

func (db *fakeDatabase) Exec(_ context.Context, sql string, _ ...interface{}) (pgconn.CommandTag, error) {
	db.execCalls = append(db.execCalls, sql)
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}
func (db *fakeDatabase) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, nil
}
func (db *fakeDatabase) QueryRow(context.Context, string, ...interface{}) pgx.Row { return db.row }

func TestHandleLoginSuccess(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-at-least-long-enough")
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	db := &fakeDatabase{row: fakeRow{scan: func(dest ...interface{}) error {
		*(dest[0].(*string)) = string(hash)
		return nil
	}}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"username":"admin","password":"correct-password"}`))
	response := httptest.NewRecorder()
	HandleLogin(db)(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload map[string]string
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil || payload["token"] == "" {
		t.Fatalf("expected JWT response, got %v", payload)
	}
}

func TestHandleLoginRejectsWrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	db := &fakeDatabase{row: fakeRow{scan: func(dest ...interface{}) error { *(dest[0].(*string)) = string(hash); return nil }}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
	response := httptest.NewRecorder()
	HandleLogin(db)(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestDispatchCommandQueuesOfflineDevice(t *testing.T) {
	db := &fakeDatabase{row: fakeRow{scan: func(dest ...interface{}) error {
		*(dest[0].(*string)) = "00000000-0000-0000-0000-000000000001"
		return nil
	}}}
	request := httptest.NewRequest(http.MethodPost, "/devices/device-1/command", strings.NewReader(`{"type":"lock"}`))
	response := httptest.NewRecorder()
	HandleDispatchCommand(db, hub.NewHub())(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", response.Code, response.Body.String())
	}
	if len(db.execCalls) != 2 {
		t.Fatalf("expected created and queued events, got %d calls", len(db.execCalls))
	}
}

func TestDispatchUnlockQueuesOfflineDevice(t *testing.T) {
	db := &fakeDatabase{row: fakeRow{scan: func(dest ...interface{}) error {
		*(dest[0].(*string)) = "00000000-0000-0000-0000-000000000001"
		return nil
	}}}
	request := httptest.NewRequest(http.MethodPost, "/devices/device-1/command", strings.NewReader(`{"type":"unlock"}`))
	response := httptest.NewRecorder()
	HandleDispatchCommand(db, hub.NewHub())(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", response.Code, response.Body.String())
	}
}

func TestDispatchCommandRejectsUnsupportedType(t *testing.T) {
	db := &fakeDatabase{}
	request := httptest.NewRequest(http.MethodPost, "/devices/device-1/command", strings.NewReader(`{"type":"reboot_everything"}`))
	response := httptest.NewRecorder()
	HandleDispatchCommand(db, hub.NewHub())(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}

func TestCommandMetrics(t *testing.T) {
	db := &fakeDatabase{row: fakeRow{scan: func(dest ...interface{}) error {
		*(dest[0].(*int64)), *(dest[1].(*int64)), *(dest[2].(*int64)), *(dest[3].(*int64)), *(dest[4].(*float64)) = 10, 2, 7, 1, 1250
		return nil
	}}}
	response := httptest.NewRecorder()
	HandleGetCommandMetrics(db)(response, httptest.NewRequest(http.MethodGet, "/metrics/commands", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"completed":7`) {
		t.Fatalf("unexpected metrics: %s", response.Body.String())
	}
}

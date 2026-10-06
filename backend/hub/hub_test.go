package hub

import "testing"

func TestHubAddAndRemove(t *testing.T) {
	h := NewHub()
	h.Add("device-1", nil)
	if _, exists := h.Conns["device-1"]; !exists {
		t.Fatal("device was not added")
	}
	h.Remove("device-1")
	if _, exists := h.Conns["device-1"]; exists {
		t.Fatal("device was not removed")
	}
}

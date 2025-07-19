package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type Event struct {
	Name      string      `json:"name"`
	EventType string      `json:"event_type"`
	Payload   interface{} `json:"payload"`
}

type EventManager struct {
	handlers map[string]chan Event
	mu       sync.Mutex
}

func NewEventManager() *EventManager {
	em := &EventManager{
		handlers: make(map[string]chan Event),
	}

	return em
}

func (em *EventManager) RegisterHandler(name string, ch chan Event) {
	em.mu.Lock()
	defer em.mu.Unlock()
	if _, exists := em.handlers[name]; !exists {
		em.handlers[name] = ch
	}
}

func (em *EventManager) UnregisterHandler(name string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	if ch, exists := em.handlers[name]; exists {
		close(ch)
		delete(em.handlers, name)
	}
}

func (h *handler) eventHandler(w http.ResponseWriter, r *http.Request) {
	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		fmt.Println("Error decoding event:", err)
		http.Error(w, "Invalid event data", http.StatusBadRequest)
		return
	}

	h.eventManager.mu.Lock()
	ch, exists := h.eventManager.handlers[event.Name]
	h.eventManager.mu.Unlock()

	if !exists {
		http.Error(w, "No handler registered for this database", http.StatusNotFound)
		return
	}

	ch <- event

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "event received"})
}

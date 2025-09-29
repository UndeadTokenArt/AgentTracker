package hub

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"

	"agentTracker/internal/models"
)

type Client struct {
	Conn   *websocket.Conn
	UID    string
	IsDM   bool
	Group  string
	SendCh chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]bool // group -> clients
}

func New() *Hub {
	return &Hub{clients: make(map[string]map[*Client]bool)}
}

func (h *Hub) AddClient(group string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.clients[group]
	if m == nil {
		m = make(map[*Client]bool)
		h.clients[group] = m
	}
	m[c] = true
}

func (h *Hub) RemoveClient(group string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.clients[group]; ok {
		delete(m, c)
		if len(m) == 0 {
			delete(h.clients, group)
		}
	}
}

type Outgoing struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// BroadcastState sends current group state to all clients in the group. DM sees HP, players do not.
func (h *Hub) BroadcastState(group string, g *models.Group) {
	h.mu.RLock()
	clients := h.clients[group]
	h.mu.RUnlock()
	for c := range clients {
		payload := struct {
			Group   string          `json:"group"`
			Round   int             `json:"round"`
			Turn    int             `json:"turn"`
			DMUID   string          `json:"dmUid"`
			Entries []models.Entity `json:"entries"`
		}{Group: g.Code, Round: g.Round, Turn: g.TurnIndex, DMUID: g.DMUID}
		if c.IsDM {
			payload.Entries = g.Entities
		} else {
			// hide HP from players
			sanitized := make([]models.Entity, len(g.Entities))
			for i, e := range g.Entities {
				if e.Type == models.Monster {
					e.HP = 0
					e.MaxHP = 0
				}
				sanitized[i] = e
			}
			payload.Entries = sanitized
		}
		msg := Outgoing{Type: "state", Data: payload}
		b, _ := json.Marshal(msg)
		select {
		case c.SendCh <- b:
		default:
			log.Println("slow client, dropping")
		}
	}
}

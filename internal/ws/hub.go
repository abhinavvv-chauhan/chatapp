package ws

import (
	"strings"

	"github.com/abhinavvv-chauhan/chat-app/internal/worker"
)

type BroadcastMessage struct {
	ChannelID string
	Data      []byte
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan BroadcastMessage
	register   chan *Client
	unregister chan *Client
	WorkerPool *worker.Pool
}

func NewHub(wp *worker.Pool) *Hub {
	return &Hub{
		broadcast:  make(chan BroadcastMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		WorkerPool: wp,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.broadcastPresence(client.channelID)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.broadcastPresence(client.channelID)
			}

		case msg := <-h.broadcast:
			for client := range h.clients {
				if client.channelID == msg.ChannelID {
					select {
					case client.send <- msg.Data:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
		}
	}
}

func (h *Hub) broadcastPresence(channelID string) {
	userMap := make(map[string]string)
	for client := range h.clients {
		if client.channelID == channelID {
			userMap[client.userID] = client.username
		}
	}

	var pairs []string
	for id, name := range userMap {
		pairs = append(pairs, id+":"+name)
	}

	payload := []byte("PRESENCE:" + strings.Join(pairs, ","))

	for client := range h.clients {
		if client.channelID == channelID {
			select {
			case client.send <- payload:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}
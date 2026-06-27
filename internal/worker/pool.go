package worker

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/SherClockHolmes/webpush-go"
)

type Task struct {
	Type    string
	Payload interface{}
}

type PushPayload struct {
	Subscription *webpush.Subscription
	Message      map[string]string
}

type Pool struct {
	TaskQueue   chan Task
	WorkerCount int
}

func NewPool(workerCount int, queueSize int) *Pool {
	return &Pool{
		TaskQueue:   make(chan Task, queueSize),
		WorkerCount: workerCount,
	}
}

func (p *Pool) Start() {
	for i := 1; i <= p.WorkerCount; i++ {
		go p.worker(i)
	}
}

func (p *Pool) worker(id int) {
	vapidPublicKey := os.Getenv("VAPID_PUBLIC_KEY")
	vapidPrivateKey := os.Getenv("VAPID_PRIVATE_KEY")

	for task := range p.TaskQueue {
		switch task.Type {
		case "web_push":
			pushData, ok := task.Payload.(PushPayload)
			if !ok {
				slog.Error("Invalid payload type for web_push")
				continue
			}

			messageBytes, _ := json.Marshal(pushData.Message)

			resp, err := webpush.SendNotification(messageBytes, pushData.Subscription, &webpush.Options{
				Subscriber:      "mailto:admin@nexuschat.com",
				VAPIDPublicKey:  vapidPublicKey,
				VAPIDPrivateKey: vapidPrivateKey,
				TTL:             30,
			})

			if err != nil {
				slog.Error("Failed to send web push", "error", err)
			} else {
				defer resp.Body.Close()
				slog.Info("Web push dispatched successfully", "status", resp.StatusCode)
			}
		}
	}
}
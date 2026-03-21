package service

import (
	"encoding/json"
	"fmt"
	"log"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/model"
)

// PushService はプッシュ通知の送信を担当する。
type PushService struct {
	vapidPublicKey  string
	vapidPrivateKey string
}

func NewPushService(vapidPublicKey, vapidPrivateKey string) *PushService {
	return &PushService{
		vapidPublicKey:  vapidPublicKey,
		vapidPrivateKey: vapidPrivateKey,
	}
}

// PushPayload はプッシュ通知のペイロード。
type PushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url,omitempty"`
}

// SendToSubscription は指定サブスクリプションにプッシュ通知を送信する。
func (p *PushService) SendToSubscription(sub model.Subscription, payload PushPayload) error {
	if sub.Platform != "web" {
		// iOS/Android は今後 FCM/APNs で対応
		log.Printf("push to %s platform not yet implemented, skipping subscription %s", sub.Platform, sub.SubscriptionID)
		return nil
	}

	// Web Push subscription JSON をパース
	var wpSub webpush.Subscription
	if err := json.Unmarshal([]byte(sub.Token), &wpSub); err != nil {
		return fmt.Errorf("failed to parse web push subscription: %w", err)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal push payload: %w", err)
	}

	resp, err := webpush.SendNotification(payloadBytes, &wpSub, &webpush.Options{
		VAPIDPublicKey:  p.vapidPublicKey,
		VAPIDPrivateKey: p.vapidPrivateKey,
		Subscriber:      "mailto:neptune-app@example.com",
	})
	if err != nil {
		return fmt.Errorf("failed to send web push: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("web push returned status %d", resp.StatusCode)
	}

	return nil
}

// SendToSubscriptions は複数のサブスクリプションにプッシュ通知を送信する。
// 個別の送信失敗はログに記録して続行する。
func (p *PushService) SendToSubscriptions(subs []model.Subscription, payload PushPayload) int {
	sent := 0
	for _, sub := range subs {
		if err := p.SendToSubscription(sub, payload); err != nil {
			log.Printf("failed to send push to %s: %v", sub.SubscriptionID, err)
			continue
		}
		sent++
	}
	return sent
}

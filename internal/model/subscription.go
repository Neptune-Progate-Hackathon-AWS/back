package model

import "time"

// Subscription はプッシュ通知用のデバイストークン。
type Subscription struct {
	SubscriptionID string    `dynamodbav:"subscriptionId"`
	UserID         string    `dynamodbav:"userId"`
	Platform       string    `dynamodbav:"platform"` // ios, android, web
	Token          string    `dynamodbav:"token"`     // デバイストークンまたは Web Push subscription JSON
	CreatedAt      time.Time `dynamodbav:"createdAt"`
}

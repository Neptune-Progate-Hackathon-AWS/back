package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/model"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/service"
)

const nearbyCheckRadius = 1000.0 // 1km

// POST /subscriptions
func (s *Server) CreateSubscription(ctx context.Context, request api.CreateSubscriptionRequestObject) (api.CreateSubscriptionResponseObject, error) {
	userID := extractUserID(ctx)

	sub := model.Subscription{
		SubscriptionID: uuid.New().String(),
		UserID:         userID,
		Platform:       string(request.Body.Platform),
		Token:          request.Body.Token,
		CreatedAt:      time.Now(),
	}

	if err := s.subscriptionRepo.Save(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	subUUID, _ := uuid.Parse(sub.SubscriptionID)
	return api.CreateSubscription201JSONResponse{
		SubscriptionId: subUUID,
		Platform:       api.SubscriptionPlatform(sub.Platform),
		CreatedAt:      sub.CreatedAt,
	}, nil
}

// DELETE /subscriptions/{subscriptionId}
func (s *Server) DeleteSubscription(ctx context.Context, request api.DeleteSubscriptionRequestObject) (api.DeleteSubscriptionResponseObject, error) {
	subID := request.SubscriptionId.String()

	existing, err := s.subscriptionRepo.FindByID(ctx, subID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	if existing == nil {
		return api.DeleteSubscription404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse(api.Error{
				Code:    "NOT_FOUND",
				Message: "指定されたサブスクリプションが見つかりません",
			}),
		}, nil
	}

	if err := s.subscriptionRepo.Delete(ctx, subID); err != nil {
		return nil, fmt.Errorf("failed to delete subscription: %w", err)
	}

	return api.DeleteSubscription204Response{}, nil
}

// POST /location/check
func (s *Server) CheckLocation(ctx context.Context, request api.CheckLocationRequestObject) (api.CheckLocationResponseObject, error) {
	userID := extractUserID(ctx)
	lat := request.Body.Lat
	lng := request.Body.Lng

	// 全トイレを取得して 1km 以内をカウント
	toilets, err := s.toiletRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list toilets: %w", err)
	}

	nearbyCount := 0
	for _, t := range toilets {
		if calculateDistance(lat, lng, t.Lat, t.Lng) <= nearbyCheckRadius {
			nearbyCount++
		}
	}

	notified := false
	var message *string

	// 周辺にトイレが 0 件 → プッシュ通知で警告
	if nearbyCount == 0 && s.pushService != nil {
		subs, err := s.subscriptionRepo.FindByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get subscriptions: %w", err)
		}

		if len(subs) > 0 {
			payload := service.PushPayload{
				Title: "トイレ情報がありません",
				Body:  "現在地の周辺 1km 以内に登録されたトイレがありません。トイレ情報を投稿して共有しましょう！",
				URL:   "/",
			}
			sent := s.pushService.SendToSubscriptions(subs, payload)
			if sent > 0 {
				notified = true
				msg := "周辺にトイレが見つかりません。プッシュ通知で警告を送信しました。"
				message = &msg
			}
		}
	}

	return api.CheckLocation200JSONResponse{
		NearbyCount: nearbyCount,
		Notified:    notified,
		Message:     message,
	}, nil
}

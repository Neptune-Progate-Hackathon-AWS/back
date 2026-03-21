package handler

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/model"
)

// toToilet は API のリクエストを内部の model に変換する。
func toToilet(req *api.CreateToiletRequest, id string) model.Toilet {
	t := model.Toilet{
		ToiletID:           id,
		Name:               req.Name,
		Brand:              string(req.Brand),
		Lat:                req.Lat,
		Lng:                req.Lng,
		ImageKey:           derefString(req.ImageKey),
		ToiletType:         string(req.ToiletType),
		RequiresPermission: req.RequiresPermission,
	}
	if req.Address != nil {
		t.Address = *req.Address
	}
	if req.Note != nil {
		t.Note = *req.Note
	}
	return t
}

// presignGetURL は S3 の GET 用 Presigned URL を生成する。
func presignGetURL(ctx context.Context, presignClient *s3.PresignClient, bucketName, imageKey string) string {
	if imageKey == "" {
		return ""
	}
	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(imageKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = 1 * time.Hour
	})
	if err != nil {
		return ""
	}
	return req.URL
}

// toAPIToilet は内部の model を API の Toilet 型に変換する。
func toAPIToilet(ctx context.Context, presignClient *s3.PresignClient, t model.Toilet, bucketName string) api.Toilet {
	toiletUUID, _ := uuid.Parse(t.ToiletID)

	resp := api.Toilet{
		ToiletId:           toiletUUID,
		Name:               t.Name,
		Brand:              api.ToiletBrand(t.Brand),
		Lat:                t.Lat,
		Lng:                t.Lng,
		ImageUrl:           presignGetURL(ctx, presignClient, bucketName, t.ImageKey),
		ToiletType:         api.ToiletToiletType(t.ToiletType),
		RequiresPermission: t.RequiresPermission,
		CreatedAt:          t.CreatedAt,
	}
	if t.Address != "" {
		resp.Address = &t.Address
	}
	if t.Note != "" {
		resp.Note = &t.Note
	}
	return resp
}

// toAPIVote は内部の model.Vote を API の Vote 型に変換する。
func toAPIVote(ctx context.Context, presignClient *s3.PresignClient, v model.Vote, bucketName string) api.Vote {
	toiletUUID, _ := uuid.Parse(v.ToiletID)

	resp := api.Vote{
		ToiletId:           toiletUUID,
		UserId:             v.UserID,
		ToiletType:         api.VoteToiletType(v.ToiletType),
		RequiresPermission: v.RequiresPermission,
		CreatedAt:          v.CreatedAt,
	}
	if !v.UpdatedAt.IsZero() {
		resp.UpdatedAt = &v.UpdatedAt
	}
	if v.Note != "" {
		resp.Note = &v.Note
	}
	if v.ImageKey != "" {
		imageURL := presignGetURL(ctx, presignClient, bucketName, v.ImageKey)
		resp.ImageUrl = &imageURL
	}
	return resp
}

// aggregateVotes は投票一覧から多数決で toiletType と requiresPermission を集計する。
func aggregateVotes(votes []model.Vote) (string, bool) {
	sharedCount := 0
	permCount := 0

	for _, v := range votes {
		if v.ToiletType == "shared" {
			sharedCount++
		}
		if v.RequiresPermission {
			permCount++
		}
	}

	toiletType := "separated"
	if sharedCount > len(votes)-sharedCount {
		toiletType = "shared"
	}

	requiresPermission := permCount > len(votes)-permCount

	return toiletType, requiresPermission
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

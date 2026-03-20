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
		ImageKey:           req.ImageKey,
		MaleCount:          req.MaleCount,
		FemaleCount:        req.FemaleCount,
		MultipurposeCount:  req.MultipurposeCount,
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
		MaleCount:          t.MaleCount,
		FemaleCount:        t.FemaleCount,
		MultipurposeCount:  t.MultipurposeCount,
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

package handler

import (
	"fmt"

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

// toImageURL は S3 バケット名と画像キーから公開URLを組み立てる。
func toImageURL(bucketName, imageKey string) string {
	if imageKey == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucketName, imageKey)
}

// toAPIToilet は内部の model を API の Toilet 型に変換する。
func toAPIToilet(t model.Toilet, bucketName string) api.Toilet {
	toiletUUID, _ := uuid.Parse(t.ToiletID)

	resp := api.Toilet{
		ToiletId:           toiletUUID,
		Name:               t.Name,
		Brand:              api.ToiletBrand(t.Brand),
		Lat:                t.Lat,
		Lng:                t.Lng,
		ImageUrl:           toImageURL(bucketName, t.ImageKey),
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

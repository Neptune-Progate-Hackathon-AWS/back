package handler

import (
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

// toCreateResponse は内部の model を API のレスポンスに変換する。
func toCreateResponse(t model.Toilet) api.CreateToilet201JSONResponse {
	toiletUUID, _ := uuid.Parse(t.ToiletID)

	resp := api.CreateToilet201JSONResponse{
		ToiletId:           toiletUUID,
		Name:               t.Name,
		Brand:              api.ToiletBrand(t.Brand),
		Lat:                t.Lat,
		Lng:                t.Lng,
		ImageUrl:           "", // TODO: S3のURLに変換
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

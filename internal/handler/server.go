package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/repository"
)

// Server は oapi-codegen が生成した StrictServerInterface を実装する。
type Server struct {
	toiletRepo *repository.ToiletRepository
}

func NewServer(toiletRepo *repository.ToiletRepository) *Server {
	return &Server{toiletRepo: toiletRepo}
}

// コンパイル時にインターフェース実装を保証する。
var _ api.StrictServerInterface = (*Server)(nil)

var errNotImplemented = fmt.Errorf("not implemented")

// POST /images/presigned-url
func (s *Server) CreatePresignedUrl(ctx context.Context, request api.CreatePresignedUrlRequestObject) (api.CreatePresignedUrlResponseObject, error) {
	return nil, errNotImplemented
}

// POST /toilets
func (s *Server) CreateToilet(ctx context.Context, request api.CreateToiletRequestObject) (api.CreateToiletResponseObject, error) {
	t := toToilet(request.Body, uuid.New().String())
	t.CreatedAt = time.Now()

	if err := s.toiletRepo.Save(ctx, t); err != nil {
		return nil, fmt.Errorf("failed to create toilet: %w", err)
	}

	return toCreateResponse(t), nil
}

// GET /toilets
func (s *Server) ListToilets(ctx context.Context, request api.ListToiletsRequestObject) (api.ListToiletsResponseObject, error) {
	return nil, errNotImplemented
}

// GET /toilets/{toiletId}
func (s *Server) GetToilet(ctx context.Context, request api.GetToiletRequestObject) (api.GetToiletResponseObject, error) {
	return nil, errNotImplemented
}

package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/repository"
)

// Server は oapi-codegen が生成した StrictServerInterface を実装する。
type Server struct {
	presignClient *s3.PresignClient
	bucketName    string
	toiletRepo    *repository.ToiletRepository
}

func NewServer(s3Client *s3.Client, bucketName string, toiletRepo *repository.ToiletRepository) *Server {
	return &Server{
		presignClient: s3.NewPresignClient(s3Client),
		bucketName:    bucketName,
		toiletRepo:    toiletRepo,
	}
}

// コンパイル時にインターフェース実装を保証する。
var _ api.StrictServerInterface = (*Server)(nil)

// POST /images/presigned-url
func (s *Server) CreatePresignedUrl(ctx context.Context, request api.CreatePresignedUrlRequestObject) (api.CreatePresignedUrlResponseObject, error) {
	imageKey := fmt.Sprintf("uploads/%s", uuid.New().String())

	presignedReq, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(imageKey),
		ContentType: aws.String(string(request.Body.ContentType)),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = 5 * time.Minute
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	return api.CreatePresignedUrl200JSONResponse{
		UploadUrl: presignedReq.URL,
		ImageKey:  imageKey,
		ExpiresIn: 300,
	}, nil
}

// POST /toilets
func (s *Server) CreateToilet(ctx context.Context, request api.CreateToiletRequestObject) (api.CreateToiletResponseObject, error) {
	t := toToilet(request.Body, uuid.New().String())
	t.CreatedAt = time.Now()

	if err := s.toiletRepo.Save(ctx, t); err != nil {
		return nil, fmt.Errorf("failed to create toilet: %w", err)
	}

	return api.CreateToilet201JSONResponse(toAPIToilet(ctx, s.presignClient, t, s.bucketName)), nil
}

// GET /toilets
func (s *Server) ListToilets(ctx context.Context, request api.ListToiletsRequestObject) (api.ListToiletsResponseObject, error) {
	toilets, err := s.toiletRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list toilets: %w", err)
	}

	searchLat := float64(request.Params.Lat)
	searchLng := float64(request.Params.Lng)
	searchRadius := 1000.0
	if request.Params.Radius != nil {
		searchRadius = float64(*request.Params.Radius)
	}

	apiToilets := make([]api.Toilet, 0, len(toilets))
	for _, t := range toilets {
		distance := calculateDistance(searchLat, searchLng, float64(t.Lat), float64(t.Lng))
		if distance > searchRadius {
			continue
		}
		apiToilets = append(apiToilets, toAPIToilet(ctx, s.presignClient, t, s.bucketName))
	}

	return api.ListToilets200JSONResponse{Toilets: apiToilets}, nil
}

// GET /toilets/{toiletId}
func (s *Server) GetToilet(ctx context.Context, request api.GetToiletRequestObject) (api.GetToiletResponseObject, error) {
	t, err := s.toiletRepo.FindByID(ctx, request.ToiletId.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get toilet: %w", err)
	}

	if t == nil {
		return api.GetToilet404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse(api.Error{
				Code:    "NOT_FOUND",
				Message: "指定されたトイレが見つかりません",
			}),
		}, nil
	}

	return api.GetToilet200JSONResponse(toAPIToilet(ctx, s.presignClient, *t, s.bucketName)), nil
}

// DELETE /toilets/{toiletId}
func (s *Server) DeleteToilet(ctx context.Context, request api.DeleteToiletRequestObject) (api.DeleteToiletResponseObject, error) {
	t, err := s.toiletRepo.FindByID(ctx, request.ToiletId.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get toilet: %w", err)
	}

	if t == nil {
		return api.DeleteToilet404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse(api.Error{
				Code:    "NOT_FOUND",
				Message: "指定されたトイレが見つかりません",
			}),
		}, nil
	}

	if err := s.toiletRepo.Delete(ctx, request.ToiletId.String()); err != nil {
		return nil, fmt.Errorf("failed to delete toilet: %w", err)
	}

	return api.DeleteToilet204Response{}, nil
}

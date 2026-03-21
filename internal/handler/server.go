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
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
)

// Server 構造体に rekognitionClient を「追加」する
type Server struct {
	presignClient     *s3.PresignClient
	bucketName        string
	toiletRepo        *repository.ToiletRepository
	rekognitionClient *rekognition.Client // ★ここだけ追加！
}

// NewServer の引数と中身に rekognitionClient を「追加」する
func NewServer(
	s3Client *s3.Client, 
	bucketName string, 
	toiletRepo *repository.ToiletRepository,
	rekognitionClient *rekognition.Client, 
) *Server {
	return &Server{
		presignClient:     s3.NewPresignClient(s3Client),
		bucketName:        bucketName,
		toiletRepo:        toiletRepo,
		rekognitionClient: rekognitionClient, // ★セットする処理を追加！
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
	// --- ★ここから追加：AI画像審査の関所 ---
	// 画像キー(ImageKey)が送られてきた場合のみ、保存前にAI審査を実施
	if request.Body.ImageKey != nil && *request.Body.ImageKey != "" {
		// rekognition.go で作ったAI審査関数を呼び出す
		isToilet, err := s.isToiletImage(ctx, *request.Body.ImageKey)
		if err != nil {
			return nil, fmt.Errorf("failed to validate image: %w", err)
		}
		
		if !isToilet {
			// トイレ以外と判定されたら、DBに保存せずに400エラーで弾く
			return api.CreateToilet400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse(api.Error{
					Code:    "BAD_REQUEST",
					Message: "画像にトイレが検出されませんでした。別の画像を試してください。",
				}),
			}, nil
		}
	}
	// --- ★追加ここまで ---

	// 審査を通過した（または画像が送られなかった）場合のみ、ここから下の保存処理が実行される
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

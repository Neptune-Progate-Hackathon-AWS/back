package handler

import (
	"context"
	"fmt"
	"time"

	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/model"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/repository"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/service"
)

// Server は oapi-codegen が生成した StrictServerInterface を実装する。
type Server struct {
	presignClient    *s3.PresignClient
	bucketName       string
	toiletRepo       *repository.ToiletRepository
	reportRepo       *repository.ReportRepository
	subscriptionRepo *repository.SubscriptionRepository
	pushService      *service.PushService
}

func NewServer(s3Client *s3.Client, bucketName string, toiletRepo *repository.ToiletRepository, reportRepo *repository.ReportRepository, subscriptionRepo *repository.SubscriptionRepository, pushService *service.PushService) *Server {
	return &Server{
		presignClient:    s3.NewPresignClient(s3Client),
		bucketName:       bucketName,
		toiletRepo:       toiletRepo,
		reportRepo:       reportRepo,
		subscriptionRepo: subscriptionRepo,
		pushService:      pushService,
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

// autoDeleteThreshold は報告件数がこの値に達したらトイレを自動削除する。
const autoDeleteThreshold = 3

// POST /toilets/{toiletId}/reports
func (s *Server) CreateReport(ctx context.Context, request api.CreateReportRequestObject) (api.CreateReportResponseObject, error) {
	toiletID := request.ToiletId.String()

	// トイレ存在確認
	t, err := s.toiletRepo.FindByID(ctx, toiletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get toilet: %w", err)
	}
	if t == nil {
		return api.CreateReport404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse(api.Error{
				Code:    "NOT_FOUND",
				Message: "指定されたトイレが見つかりません",
			}),
		}, nil
	}

	// TODO: JWT から userID を取得する。現在は仮実装。
	userID := extractUserID(ctx)

	// 重複チェック
	exists, err := s.reportRepo.ExistsByToiletAndUser(ctx, toiletID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicate report: %w", err)
	}
	if exists {
		return api.CreateReport409JSONResponse(api.Error{
			Code:    "DUPLICATE_REPORT",
			Message: "すでに報告済みです",
		}), nil
	}

	// 報告保存
	report := model.Report{
		ReportID:  uuid.New().String(),
		ToiletID:  toiletID,
		UserID:    userID,
		Reason:    string(request.Body.Reason),
		CreatedAt: time.Now(),
	}
	if request.Body.Comment != nil {
		report.Comment = *request.Body.Comment
	}

	if err := s.reportRepo.Save(ctx, report); err != nil {
		// DynamoDB の条件式失敗（重複）の場合
		if strings.Contains(err.Error(), "ConditionalCheckFailed") {
			return api.CreateReport409JSONResponse(api.Error{
				Code:    "DUPLICATE_REPORT",
				Message: "すでに報告済みです",
			}), nil
		}
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	// 報告件数を確認し、閾値以上ならトイレを自動削除
	count, err := s.reportRepo.CountByToiletID(ctx, toiletID)
	if err != nil {
		return nil, fmt.Errorf("failed to count reports: %w", err)
	}

	if count >= autoDeleteThreshold {
		if err := s.toiletRepo.Delete(ctx, toiletID); err != nil {
			return nil, fmt.Errorf("failed to auto-delete toilet: %w", err)
		}
	}

	return api.CreateReport201JSONResponse(api.CreateReportResponse{
		Message:     "報告を受け付けました",
		ReportCount: count,
	}), nil
}

// GET /toilets/{toiletId}/reports/count
func (s *Server) GetReportCount(ctx context.Context, request api.GetReportCountRequestObject) (api.GetReportCountResponseObject, error) {
	count, err := s.reportRepo.CountByToiletID(ctx, request.ToiletId.String())
	if err != nil {
		return nil, fmt.Errorf("failed to count reports: %w", err)
	}

	return api.GetReportCount200JSONResponse(api.ReportCountResponse{
		Count: count,
	}), nil
}

// extractUserID はコンテキストからユーザーIDを取得する。
// API Gateway の Cognito Authorizer が設定した値を利用する。
// ローカル開発時は "anonymous" を返す。
func extractUserID(ctx context.Context) string {
	// TODO: API Gateway の Cognito Authorizer から sub claim を取得する実装
	// 現時点ではヘッダーから取得する簡易実装
	_ = ctx
	return "anonymous"
}

package handler

import (
	"context"
	"fmt"
	"time"

	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	core "github.com/awslabs/aws-lambda-go-api-proxy/core"
	"github.com/google/uuid"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/model"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/repository"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/service"
)

// Server は oapi-codegen が生成した StrictServerInterface を実装する。
type Server struct {
	presignClient     *s3.PresignClient
	bucketName        string
	toiletRepo        *repository.ToiletRepository
	voteRepo          *repository.VoteRepository
	reportRepo        *repository.ReportRepository
	subscriptionRepo  *repository.SubscriptionRepository
	pushService       *service.PushService
	navigationService *service.NavigationService
}

func NewServer(s3Client *s3.Client, bucketName string, toiletRepo *repository.ToiletRepository, voteRepo *repository.VoteRepository, reportRepo *repository.ReportRepository, subscriptionRepo *repository.SubscriptionRepository, pushService *service.PushService, navigationService *service.NavigationService) *Server {
	return &Server{
		presignClient:     s3.NewPresignClient(s3Client),
		bucketName:        bucketName,
		toiletRepo:        toiletRepo,
		voteRepo:          voteRepo,
		reportRepo:        reportRepo,
		subscriptionRepo:  subscriptionRepo,
		pushService:       pushService,
		navigationService: navigationService,
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
	now := time.Now()
	t.CreatedAt = now

	if err := s.toiletRepo.Save(ctx, t); err != nil {
		return nil, fmt.Errorf("failed to create toilet: %w", err)
	}

	// 初回投票を自動作成
	userID := extractUserID(ctx)
	vote := model.Vote{
		ToiletID:           t.ToiletID,
		UserID:             userID,
		ToiletType:         t.ToiletType,
		RequiresPermission: t.RequiresPermission,
		Note:               t.Note,
		ImageKey:           t.ImageKey,
		CreatedAt:          now,
	}
	if err := s.voteRepo.Save(ctx, vote); err != nil {
		return nil, fmt.Errorf("failed to create initial vote: %w", err)
	}

	apiToilet := toAPIToilet(ctx, s.presignClient, t, s.bucketName)
	voteCount := 1
	apiToilet.VoteCount = &voteCount
	apiVote := toAPIVote(ctx, s.presignClient, vote, s.bucketName)
	apiToilet.MyVote = &apiVote

	return api.CreateToilet201JSONResponse(apiToilet), nil
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

		apiToilet := toAPIToilet(ctx, s.presignClient, t, s.bucketName)

		// 投票集計
		votes, err := s.voteRepo.FindByToiletID(ctx, t.ToiletID)
		if err == nil && len(votes) > 0 {
			count := len(votes)
			apiToilet.VoteCount = &count
			aggregatedType, aggregatedPerm := aggregateVotes(votes)
			apiToilet.ToiletType = api.ToiletToiletType(aggregatedType)
			apiToilet.RequiresPermission = aggregatedPerm
		}

		apiToilets = append(apiToilets, apiToilet)
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

	apiToilet := toAPIToilet(ctx, s.presignClient, *t, s.bucketName)

	// 投票集計
	votes, err := s.voteRepo.FindByToiletID(ctx, t.ToiletID)
	if err == nil && len(votes) > 0 {
		count := len(votes)
		apiToilet.VoteCount = &count
		aggregatedType, aggregatedPerm := aggregateVotes(votes)
		apiToilet.ToiletType = api.ToiletToiletType(aggregatedType)
		apiToilet.RequiresPermission = aggregatedPerm

		apiVotes := make([]api.Vote, 0, len(votes))
		for _, v := range votes {
			apiVotes = append(apiVotes, toAPIVote(ctx, s.presignClient, v, s.bucketName))
		}
		apiToilet.Votes = &apiVotes

		// myVote
		userID := extractUserID(ctx)
		for _, v := range votes {
			if v.UserID == userID {
				myVote := toAPIVote(ctx, s.presignClient, v, s.bucketName)
				apiToilet.MyVote = &myVote
				break
			}
		}
	}

	return api.GetToilet200JSONResponse(apiToilet), nil
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

	// 報告理由バリデーション（生成コードは Valid() を自動呼び出ししないため明示的に検証）
	if !request.Body.Reason.Valid() {
		return api.CreateReport400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse(api.Error{
				Code:    "INVALID_REASON",
				Message: "無効な報告理由です",
			}),
		}, nil
	}

	// コメント文字数バリデーション（日本語文字列を考慮して rune 単位で計測）
	if request.Body.Comment != nil && len([]rune(*request.Body.Comment)) > 500 {
		return api.CreateReport400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse(api.Error{
				Code:    "COMMENT_TOO_LONG",
				Message: "コメントは500文字以内で入力してください",
			}),
		}, nil
	}

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

// POST /toilets/{toiletId}/votes
func (s *Server) CreateVote(ctx context.Context, request api.CreateVoteRequestObject) (api.CreateVoteResponseObject, error) {
	toiletID := request.ToiletId.String()

	// トイレ存在確認
	t, err := s.toiletRepo.FindByID(ctx, toiletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get toilet: %w", err)
	}
	if t == nil {
		return api.CreateVote404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse(api.Error{
				Code:    "NOT_FOUND",
				Message: "指定されたトイレが見つかりません",
			}),
		}, nil
	}

	userID := extractUserID(ctx)
	now := time.Now()

	vote := model.Vote{
		ToiletID:           toiletID,
		UserID:             userID,
		ToiletType:         string(request.Body.ToiletType),
		RequiresPermission: request.Body.RequiresPermission,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if request.Body.Note != nil {
		vote.Note = *request.Body.Note
	}
	if request.Body.ImageKey != nil {
		vote.ImageKey = *request.Body.ImageKey
	}

	if err := s.voteRepo.Save(ctx, vote); err != nil {
		return nil, fmt.Errorf("failed to save vote: %w", err)
	}

	return api.CreateVote201JSONResponse(toAPIVote(ctx, s.presignClient, vote, s.bucketName)), nil
}

// GET /toilets/{toiletId}/votes
func (s *Server) ListVotes(ctx context.Context, request api.ListVotesRequestObject) (api.ListVotesResponseObject, error) {
	toiletID := request.ToiletId.String()

	// トイレ存在確認
	t, err := s.toiletRepo.FindByID(ctx, toiletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get toilet: %w", err)
	}
	if t == nil {
		return api.ListVotes404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse(api.Error{
				Code:    "NOT_FOUND",
				Message: "指定されたトイレが見つかりません",
			}),
		}, nil
	}

	votes, err := s.voteRepo.FindByToiletID(ctx, toiletID)
	if err != nil {
		return nil, fmt.Errorf("failed to list votes: %w", err)
	}

	apiVotes := make([]api.Vote, 0, len(votes))
	for _, v := range votes {
		apiVotes = append(apiVotes, toAPIVote(ctx, s.presignClient, v, s.bucketName))
	}

	return api.ListVotes200JSONResponse{Votes: apiVotes}, nil
}

// extractUserID はコンテキストからユーザーIDを取得する。
// API Gateway の Cognito Authorizer が設定した sub claim を利用する。
// ローカル開発時や認証情報がない場合は "anonymous" を返す。
func extractUserID(ctx context.Context) string {
	gatewayCtx, ok := core.GetAPIGatewayContextFromContext(ctx)
	if !ok {
		return "anonymous"
	}
	sub, ok := gatewayCtx.Authorizer["sub"]
	if !ok {
		return "anonymous"
	}
	subStr, ok := sub.(string)
	if !ok || subStr == "" {
		return "anonymous"
	}
	return subStr
}

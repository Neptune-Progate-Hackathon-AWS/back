package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/repository"
)

// Server は oapi-codegen が生成した StrictServerInterface を実装する。
type Server struct {
	dynamoClient  *dynamodb.Client // あなたのロジック用
	s3Client      *s3.Client       // あなたのロジック用
	presignClient *s3.PresignClient
	bucketName    string
	toiletRepo    *repository.ToiletRepository // チームメイトの整理用
}

func NewServer(dbClient *dynamodb.Client, s3Client *s3.Client, bucketName string, toiletRepo *repository.ToiletRepository) *Server {
	return &Server{
		dynamoClient:  dbClient,
		s3Client:      s3Client,
		presignClient: s3.NewPresignClient(s3Client),
		bucketName:    bucketName,
		toiletRepo:    toiletRepo,
	}
}

// コンパイル時にインターフェース実装を保証する
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
	// ここはあなたの「動くロジック」を優先して実装します
	toiletID := uuid.New().String()
	now := time.Now()

	item := map[string]interface{}{
		"toiletId":           toiletID,
		"name":               request.Body.Name,
		"brand":              string(request.Body.Brand),
		"lat":                request.Body.Lat,
		"lng":                request.Body.Lng,
		"imageKey":           request.Body.ImageKey,
		"maleCount":          request.Body.MaleCount,
		"femaleCount":        request.Body.FemaleCount,
		"multipurposeCount":  request.Body.MultipurposeCount,
		"requiresPermission": request.Body.RequiresPermission,
		"createdAt":          now.Format(time.RFC3339),
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}

	_, err = s.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("ToiletsTable"),
		Item:      av,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to put item: %w", err)
	}

	var toilet api.Toilet
	attributevalue.UnmarshalMap(av, &toilet)
	toilet.ImageUrl = fmt.Sprintf("https://%s.s3.us-east-1.amazonaws.com/%s", s.bucketName, request.Body.ImageKey)

	return api.CreateToilet201JSONResponse(toilet), nil
}

// GET /toilets
// 3. 【トイレ一覧取得】
func (s *Server) ListToilets(ctx context.Context, request api.ListToiletsRequestObject) (api.ListToiletsResponseObject, error) {
	output, err := s.dynamoClient.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String("ToiletsTable"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	// DTOを修正：DBにあるけど api.Toilet にはない ImageKey を明示的に受け取る
	type toiletDTO struct {
		api.Toilet
		ToiletId string `dynamodbav:"toiletId"`
		ImageKey string `dynamodbav:"imageKey"` // これを追加！
	}

	var dtos []toiletDTO
	if err := attributevalue.UnmarshalListOfMaps(output.Items, &dtos); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dtos: %w", err)
	}

	var toilets []api.Toilet
	for _, d := range dtos {
		t := d.Toilet
		// IDのパース
		if parsedID, err := uuid.Parse(d.ToiletId); err == nil {
			t.ToiletId = parsedID
		}
		
		// URLの組み立て：d.ImageKey を使います（t.ImageKey は存在しないため）
		if d.ImageKey != "" {
			t.ImageUrl = fmt.Sprintf("https://%s.s3.us-east-1.amazonaws.com/%s", s.bucketName, d.ImageKey)
		}
		
		toilets = append(toilets, t)
	}

	return api.ListToilets200JSONResponse{Toilets: toilets}, nil
}

// GET /toilets/{toiletId}
// 4. 【トイレ詳細取得】 GET /toilets/{toiletId}
func (s *Server) GetToilet(ctx context.Context, request api.GetToiletRequestObject) (api.GetToiletResponseObject, error) {
	output, err := s.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("ToiletsTable"),
		Key: map[string]types.AttributeValue{
			"toiletId": &types.AttributeValueMemberS{Value: request.ToiletId.String()},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if output.Item == nil {
		return api.GetToilet404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse(api.Error{
				Code:    "NOT_FOUND",
				Message: "指定されたトイレが見つかりません",
			}),
		}, nil
	}

	// DTOを修正：ImageKey を受け取れるようにする
	type toiletDTO struct {
		api.Toilet
		ToiletId string `dynamodbav:"toiletId"`
		ImageKey string `dynamodbav:"imageKey"` // これを追加！
	}

	var d toiletDTO
	if err := attributevalue.UnmarshalMap(output.Item, &d); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dto: %w", err)
	}

	toilet := d.Toilet
	if parsedID, err := uuid.Parse(d.ToiletId); err == nil {
		toilet.ToiletId = parsedID
	}

	// URLの組み立て：d.ImageKey を使います
	if d.ImageKey != "" {
		toilet.ImageUrl = fmt.Sprintf("https://%s.s3.us-east-1.amazonaws.com/%s", s.bucketName, d.ImageKey)
	}

	return api.GetToilet200JSONResponse(toilet), nil
}
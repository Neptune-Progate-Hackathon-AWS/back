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
)

type Server struct {
	dynamoClient  *dynamodb.Client
	s3Client      *s3.Client
	presignClient *s3.PresignClient
	bucketName    string
}

func NewServer(dbClient *dynamodb.Client, s3Client *s3.Client, bucketName string) *Server {
	return &Server{
		dynamoClient:  dbClient,
		s3Client:      s3Client,
		presignClient: s3.NewPresignClient(s3Client),
		bucketName:    bucketName,
	}
}

// 1. 【画像アップロード用URL発行】
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

	// あなたの api.gen.go では PresignedUrlResponse の別名になっています
	return api.CreatePresignedUrl200JSONResponse{
		UploadUrl: presignedReq.URL,
		ImageKey:  imageKey,
		ExpiresIn: 300,
	}, nil
}

// 2. 【トイレ登録】
func (s *Server) CreateToilet(ctx context.Context, request api.CreateToiletRequestObject) (api.CreateToiletResponseObject, error) {
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

	// あなたの api.gen.go では Toilet の別名になっています
	return api.CreateToilet201JSONResponse(toilet), nil
}

// 3. 【トイレ一覧取得】
func (s *Server) ListToilets(ctx context.Context, request api.ListToiletsRequestObject) (api.ListToiletsResponseObject, error) {
	output, err := s.dynamoClient.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String("ToiletsTable"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	var toilets []api.Toilet
	err = attributevalue.UnmarshalListOfMaps(output.Items, &toilets)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal list: %w", err)
	}

	// ここは Toilets という名前のフィールドを持つ構造体です
	return api.ListToilets200JSONResponse{
		Toilets: toilets,
	}, nil
}

// 4. 【トイレ詳細取得】
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
		// 【大注目！】ここがエラーの核心でした。
		// GetToilet404JSONResponse は NotFoundJSONResponse を「埋め込んだ」構造体です。
		return api.GetToilet404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse(api.Error{
				Code:    "NOT_FOUND",
				Message: "指定されたトイレが見つかりません",
			}),
		}, nil
	}

	var toilet api.Toilet
	err = attributevalue.UnmarshalMap(output.Item, &toilet)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	// 200 OK は Toilet の別名です
	return api.GetToilet200JSONResponse(toilet), nil
}
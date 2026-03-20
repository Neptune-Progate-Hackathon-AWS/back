package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/handler"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/repository"
)

func main() {
	ctx := context.Background()

	// AWS設定の読み込み
	opts := []func(*config.LoadOptions) error{}
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	if region := os.Getenv("AWS_REGION"); region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		log.Fatalf("AWS設定の読み込みに失敗しました: %v", err)
	}

	// AWSクライアントとリポジトリの組み立て
	dbClient := dynamodb.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)
	toiletRepo := repository.NewToiletRepository(dbClient)

	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		log.Fatal("S3_BUCKET_NAME 環境変数が設定されていません")
	}

	server := handler.NewServer(s3Client, bucketName, toiletRepo)

	// ルーター設定
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://main.d3mags6w0gkuer.amplifyapp.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	api.HandlerFromMux(api.NewStrictHandler(server, nil), r)

	// Lambda環境ではLambdaハンドラーとして起動、それ以外はHTTPサーバー
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		chiLambda := chiadapter.New(r)
		lambda.Start(chiLambda.ProxyWithContext)
	} else {
		addr := ":8080"
		fmt.Printf("Server listening on %s\n", addr)
		log.Fatal(http.ListenAndServe(addr, r))
	}
}

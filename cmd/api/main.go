package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/location"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/handler"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/repository"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/service"
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
	reportRepo := repository.NewReportRepository(dbClient)
	subscriptionRepo := repository.NewSubscriptionRepository(dbClient)

	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		log.Fatal("S3_BUCKET_NAME 環境変数が設定されていません")
	}

	// VAPID鍵が設定されている場合のみ PushService を有効化
	var pushSvc *service.PushService
	vapidPub := os.Getenv("VAPID_PUBLIC_KEY")
	vapidPriv := os.Getenv("VAPID_PRIVATE_KEY")
	if vapidPub != "" && vapidPriv != "" {
		pushSvc = service.NewPushService(vapidPub, vapidPriv)
		log.Println("Web Push enabled")
	} else {
		log.Println("VAPID keys not set, Web Push disabled")
	}

	// NavigationService: Location Service + Bedrock
	locationClient := location.NewFromConfig(cfg)

	bedrockRegion := os.Getenv("BEDROCK_REGION")
	if bedrockRegion == "" {
		bedrockRegion = "us-east-1"
	}
	bedrockClient := bedrockruntime.NewFromConfig(cfg, func(o *bedrockruntime.Options) {
		o.Region = bedrockRegion
	})

	calculatorName := os.Getenv("ROUTE_CALCULATOR_NAME")
	if calculatorName == "" {
		calculatorName = "neptune-route-calculator"
	}
	navigationService := service.NewNavigationService(locationClient, bedrockClient, calculatorName)

	server := handler.NewServer(s3Client, bucketName, toiletRepo, reportRepo, subscriptionRepo, pushSvc, navigationService)

	// ルーター設定
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:5173", "https://main.d3mags6w0gkuer.amplifyapp.com", "https://d337uiklw4m572.cloudfront.net"},
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

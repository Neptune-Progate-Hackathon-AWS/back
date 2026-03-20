package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3" // S3用のパッケージを追加
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/handler"
)

func main() {
	ctx := context.Background()

	// 1. AWSの設定を読み込む
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile("hackathon"),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		log.Fatalf("AWS設定の読み込みに失敗しました: %v", err)
	}

	// 2. 各サービス用のクライアント（窓口）を作成
	dbClient := dynamodb.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg) // S3用のクライアントを新設

	// 自分でAWSコンソールで作ったS3バケット名を入れてください
	bucketName := "neptune-toilet-images" 

	// 3. ルーター (chi) の設定
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// 4. サーバー(Handler)の初期化
	// DynamoDBクライアント、S3クライアント、バケット名をセットで渡します
	server := handler.NewServer(dbClient, s3Client, bucketName)

	// 5. 自動生成されたAPI定義と自分たちの処理を紐付ける
	h := api.HandlerFromMux(api.NewStrictHandler(server, nil), r)

	// 6. サーバー起動
	addr := ":8080"
	fmt.Printf("Server listening on %s (AWS Region: us-east-1)\n", addr)
	log.Fatal(http.ListenAndServe(addr, h))
}
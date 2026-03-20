package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/handler"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/repository"
)

func main() {
	ctx := context.Background()

	// 1. AWS設定の読み込み（チームメイトの柔軟な方式を採用）
	opts := []func(*config.LoadOptions) error{}
	
	// プロファイル指定があれば読み込む（なければデフォルト）
	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "hackathon" // あなたの環境に合わせてデフォルトをhackathonに
	}
	opts = append(opts, config.WithSharedConfigProfile(profile))

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1" // デフォルトをus-east-1に
	}
	opts = append(opts, config.WithRegion(region))

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		log.Fatalf("AWS設定の読み込みに失敗しました: %v", err)
	}

	// 2. クライアントとリポジトリの準備
	dbClient := dynamodb.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)
	
	// さきほど成功したバケット名
	bucketName := "neptune-toilet-images" 

	// チームメイトが作ったRepository
	toiletRepo := repository.NewToiletRepository(dbClient)

	// 3. サーバー(Handler)の初期化
	// 引数を、あなたが修正した server.go の NewServer(dbClient, s3Client, bucketName, toiletRepo) に合わせます
	server := handler.NewServer(dbClient, s3Client, bucketName, toiletRepo)

	// 4. ルーター設定
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// 自動生成されたAPI定義と自分たちの処理を紐付ける
	h := api.HandlerFromMux(api.NewStrictHandler(server, nil), r)

	addr := ":8080"
	fmt.Printf("Server listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, h))
}
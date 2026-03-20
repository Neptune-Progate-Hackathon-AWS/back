package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

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

	// DynamoDBクライアント → Repository → Handler の順に組み立てる
	dbClient := dynamodb.NewFromConfig(cfg)
	toiletRepo := repository.NewToiletRepository(dbClient)
	server := handler.NewServer(toiletRepo)

	// ルーター設定
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := api.HandlerFromMux(api.NewStrictHandler(server, nil), r)

	addr := ":8080"
	fmt.Printf("Server listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, h))
}

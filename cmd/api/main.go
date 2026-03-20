package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/handler"
)

func main() {
	ctx := context.Background()

	// 1. AWSの設定を読み込む (ハッカソン用プロファイルを使用)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile("hackathon"),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		log.Fatalf("AWS設定の読み込みに失敗しました: %v", err)
	}

	// 2. DynamoDBクライアントを作成
	dbClient := dynamodb.NewFromConfig(cfg)

	// 3. ルーター (chi) の設定
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// 4. サーバー(Handler)の初期化
	// DynamoDBクライアントを渡して、データベースを使えるようにします
	server := handler.NewServer(dbClient)

	// 5. 自動生成されたAPI定義と自分たちの処理を紐付ける
	// ここは元のコードの「正しい書き方」を維持しています
	h := api.HandlerFromMux(api.NewStrictHandler(server, nil), r)

	// 6. サーバー起動
	addr := ":8080"
	fmt.Printf("Server listening on %s (AWS Region: us-east-1)\n", addr)
	log.Fatal(http.ListenAndServe(addr, h))
}
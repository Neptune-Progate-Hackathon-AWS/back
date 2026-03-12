# コンビニトイレマップ API

## セットアップ

```bash
# Go インストール
brew install go

# 依存解決
go mod tidy
```

## 開発コマンド

```bash
# ローカル起動（:8080）
go run cmd/api/main.go

# OpenAPI → Go コード再生成
make generate

# Lambda 用バイナリビルド
make build
```

## ディレクトリ構成

```
├── openapi.yml              ← API仕様（フロントと共有する source of truth）
├── oapi-codegen.cfg.yaml    ← コード生成設定
├── Makefile
├── cmd/api/
│   └── main.go              ← エントリポイント
└── internal/
    ├── api/
    │   └── openapi.gen.go   ← 自動生成（直接編集しない）
    └── handler/
        └── server.go        ← ここにAPIの実装を書く
```

## 開発の流れ

1. `openapi.yml` を編集（API 仕様変更時）
2. `make generate` で Go コード再生成
3. `internal/handler/server.go` にコンパイルエラーが出るので、新しいメソッドを実装
4. `go run cmd/api/main.go` で動作確認

## API ドキュメント

[Swagger UI で見る](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/Neptune-Progate-Hackathon-AWS/back/main/openapi.yml)

## フロントエンドとの連携

フロント側は Orval で以下の URL からAPIクライアントを自動生成する：

```
https://raw.githubusercontent.com/Neptune-Progate-Hackathon-AWS/back/main/openapi.yml
```

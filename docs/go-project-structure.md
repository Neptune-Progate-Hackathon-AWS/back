# Go プロジェクト構成ガイド

## ディレクトリ構成

```
back/
├── cmd/api/main.go              # エントリーポイント（配線だけ）
├── internal/                    # 外部から import 不可の private コード
│   ├── api/openapi.gen.go       # 自動生成（編集しない）
│   ├── handler/                 # リクエスト/レスポンス処理
│   │   ├── server.go            # ハンドラー本体
│   │   └── converter.go         # API型 ⇔ model型 の変換
│   ├── model/                   # ドメインモデル（型定義）
│   │   └── toilet.go
│   └── repository/              # DBアクセス層
│       └── toilet.go
├── openapi.yml                  # API設計図
├── oapi-codegen.cfg.yaml        # コード生成設定
├── Makefile                     # make generate でコード生成
├── go.mod / go.sum              # 依存管理（package.json相当）
└── docker-compose.yml           # ローカル開発環境
```

## レイヤーの役割

```
リクエスト
  ↓
handler    リクエスト受け取り → model に変換 → repository に委譲 → レスポンス返却
  ↓
repository DynamoDB への保存・取得だけ
  ↓
model      アプリ内部のデータ型定義（API層・DB層と独立）
```

### 各層のルール

| 層 | 責務 | 依存してよいもの |
|----|------|-----------------|
| `cmd/` | DI の組み立て、サーバー起動 | handler, repository, api |
| `handler/` | リクエスト/レスポンスの変換、エラーハンドリング | model, repository, api |
| `repository/` | DB の読み書き | model |
| `model/` | 型定義のみ | なし（最下層） |

**循環参照は Go では禁止。** model → api のように下から上への依存はできない。

## コード生成の流れ

```
openapi.yml → make generate → internal/api/openapi.gen.go
```

- `openapi.gen.go` は**絶対に手で編集しない**
- openapi.yml を変更したら `make generate` を実行
- 生成されるもの: リクエスト/レスポンスの型、ルーティング、StrictServerInterface

## Go の基本ルール（TypeScript との対比）

### 可視性
- **大文字始まり = public** (`NewServer`, `Toilet`)
- **小文字始まり = private** (`toiletRepo`, `toToilet`)
- TypeScript の `export` / 未export に相当

### エラーハンドリング
```go
// Go: try/catch がない。関数が error を返し、毎回チェックする
result, err := doSomething()
if err != nil {
    return fmt.Errorf("context: %w", err)  // %w でエラーを wrap
}
```

### ポインタ
```go
// * がついた型 → nil になりうる（optional に相当）
Address *string   // nil or "渋谷区..."

// 値を取り出すときは * をつける
if req.Address != nil {
    t.Address = *req.Address
}
```

### struct タグ
```go
type Toilet struct {
    ToiletID string `dynamodbav:"toiletId"`  // DynamoDB のカラム名を指定
    Name     string `json:"name"`            // JSON のフィールド名を指定
}
```
タグをつけると `MarshalMap(t)` だけで自動的に変換される。

### メソッドレシーバ
```go
// TypeScript: class の this が暗黙的
// Go: レシーバを明示的に書く
func (r *ToiletRepository) Save(ctx context.Context, t model.Toilet) error {
    // r が this に相当
}
```

### 日付フォーマット
```go
// Go は「2006-01-02 15:04:05」という具体的な日時をテンプレートに使う
// 覚え方: 1月2日 3(15)時4分5秒 2006年 -7時間
t.Format("2006-01-02T15:04:05Z07:00")
```

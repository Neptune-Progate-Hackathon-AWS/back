# バックエンド依頼: 投票（Vote）機能の追加

## 背景

現在、1つのコンビニトイレに対して1人しか情報登録できない。
複数ユーザーが「投票」できる仕組みに変更し、情報の信頼性を上げたい。

## 現状のアーキテクチャ

- API Gateway + Lambda + DynamoDB
- 認証: Cognito Authorizer (JWT)
- 既存テーブル: Toilets（toiletId がパーティションキー）

## 要求する変更

### 1. DynamoDB: Votes テーブルを新規作成

| 属性 | 型 | キー | 説明 |
|---|---|---|---|
| toiletId | String | PK | トイレID |
| userId | String | SK | Cognito ユーザーID |
| toiletType | String | - | "shared" or "separated" |
| requiresPermission | Boolean | - | 店員許可が必要か |
| note | String | - | 備考（任意、max 500） |
| imageKey | String | - | S3画像キー（任意） |
| createdAt | String | - | ISO 8601 |
| updatedAt | String | - | ISO 8601 |

- 同一ユーザーが同一トイレに二重投票 → 上書き更新（upsert）

### 2. OpenAPI: エンドポイント追加

#### `POST /toilets/{toiletId}/votes` — 投票する

```yaml
/toilets/{toiletId}/votes:
  post:
    tags: [votes]
    summary: トイレ情報に投票
    operationId: createVote
    security:
      - cognitoAuth: []
    parameters:
      - $ref: "#/components/parameters/toiletId"
    requestBody:
      required: true
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/CreateVoteRequest"
    responses:
      "201":
        description: 投票成功
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Vote"
      "400":
        $ref: "#/components/responses/BadRequest"
      "401":
        $ref: "#/components/responses/Unauthorized"
      "404":
        $ref: "#/components/responses/NotFound"
```

#### `GET /toilets/{toiletId}/votes` — 投票一覧

```yaml
  get:
    tags: [votes]
    summary: トイレの投票一覧を取得
    operationId: listVotes
    security:
      - cognitoAuth: []
    parameters:
      - $ref: "#/components/parameters/toiletId"
    responses:
      "200":
        description: 投票一覧
        content:
          application/json:
            schema:
              type: object
              required: [votes]
              properties:
                votes:
                  type: array
                  items:
                    $ref: "#/components/schemas/Vote"
      "401":
        $ref: "#/components/responses/Unauthorized"
      "404":
        $ref: "#/components/responses/NotFound"
```

### 3. OpenAPI: スキーマ追加

```yaml
CreateVoteRequest:
  type: object
  required:
    - toiletType
    - requiresPermission
  properties:
    toiletType:
      type: string
      enum: [shared, separated]
    requiresPermission:
      type: boolean
    note:
      type: string
      maxLength: 500
    imageKey:
      type: string

Vote:
  type: object
  required:
    - toiletId
    - userId
    - toiletType
    - requiresPermission
    - createdAt
  properties:
    toiletId:
      type: string
      format: uuid
    userId:
      type: string
    toiletType:
      type: string
      enum: [shared, separated]
    requiresPermission:
      type: boolean
    note:
      type: string
    imageUrl:
      type: string
      format: uri
    createdAt:
      type: string
      format: date-time
    updatedAt:
      type: string
      format: date-time
```

### 4. Toilet レスポンスに集計フィールドを追加

Toilet スキーマに以下を追加:

```yaml
# 既存フィールドに追加
voteCount:
  type: integer
  description: 投票数
  example: 5
votes:
  type: array
  description: 投票一覧（listToilets では省略可、getToilet で返す）
  items:
    $ref: "#/components/schemas/Vote"
myVote:
  description: リクエストユーザー自身の投票（あれば）
  $ref: "#/components/schemas/Vote"
```

**toiletType / requiresPermission は投票の多数決で集計した値を返す:**
- `toiletType`: "shared" と "separated" の投票数が多い方
- `requiresPermission`: true の投票数が過半数なら true

### 5. createToilet の変更

`POST /toilets` でトイレを新規作成した時、**同時に初回投票も自動作成**する。
つまり createToilet のリクエストに含まれる toiletType / requiresPermission は、
そのまま投稿者の最初の Vote としても記録する。

### 6. Lambda 実装のポイント

- `createVote`: Votes テーブルに PUT（upsert）。userId は JWT の sub クレームから取得
- `listVotes`: toiletId で Query
- `getToilet` / `listToilets`: Votes テーブルから集計して voteCount / toiletType / requiresPermission を算出
  - listToilets は voteCount だけ返せば OK（votes 配列は省略）
  - getToilet は votes / myVote も返す
- `createToilet`: 既存処理 + Votes テーブルに初回投票を書き込む

## フロント側の対応（こちらでやる）

- openapi.yml 更新後に `orval` で再生成
- クイック登録: 未登録なら createToilet / 登録済みなら createVote
- 詳細画面: voteCount 表示、自分の投票状態表示
- 投票ボタン UI

## 優先度

高: createVote, Toilet レスポンスの集計フィールド追加
中: listVotes, myVote
低: ポイント制（後回しでOK）

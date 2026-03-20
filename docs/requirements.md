# コンビニトイレマップ 設計書

## 1. プロジェクト概要

コンビニのトイレ位置情報を共有・検索できる Web アプリケーション。
ユーザーがトイレ情報を投稿し、地図上で検索できる。

---

## 2. 機能要件

### 2.1 認証

- ユーザー登録（メール + パスワード + ニックネーム）
- メール確認コードによる本人確認
- ログイン / ログアウト
- トークンリフレッシュ
- **フロントから Cognito SDK（Amplify Auth）を直接利用**。バックエンドに認証エンドポイントは持たない
- API Gateway の Cognito Authorizer で JWT を検証

### 2.2 トイレ情報の登録

- ユーザーがコンビニ名・ブランド・住所・位置情報とともにトイレ情報を投稿（1回の操作で完結）
- 投稿内容：コンビニ名、ブランド、住所、位置情報、画像、男性用/女性用/多目的トイレの個数、店員許可の要否、自由記入欄
- 画像は S3 に presigned URL でアップロード

### 2.3 トイレ情報の検索・表示

- 指定位置 + 半径でトイレ一覧を取得（デフォルト 1000m、100〜5000m）
- 地図上にピン表示
- トイレ詳細の表示

### 2.4 画像審査（MVP後）

- 投稿時に Rekognition でトイレ画像かどうかを審査（信頼度 70% 以上で通過）

### 2.5 虚偽報告（MVP後）

- 「トイレがない」証拠画像付きで報告
- Rekognition で証拠画像を審査（信頼度 60% 以上）
- 同一トイレに対して有効報告 3 件以上 → マップから自動削除

### 2.6 ナビゲーション（MVP後）

- 現在位置から最寄りトイレへの経路を提案
- Amazon Location Service で徒歩ルート計算
- Amazon Bedrock（Claude 3 Haiku）で提案文を生成
- 地図上にポリラインで経路表示

### 2.7 プッシュ通知（MVP後）

- デバイストークン登録（iOS / Android / Web）
- 定期的に現在位置を送信
- 周辺 1km 以内にトイレが 0 件 → プッシュ通知で警告

---

## 3. 技術スタック

### フロントエンド

| 項目                 | 技術                                     |
| -------------------- | ---------------------------------------- |
| 言語                 | TypeScript                               |
| フレームワーク       | Next.js                                  |
| ホスティング         | AWS Amplify Hosting                      |
| 認証                 | Amplify Auth（Cognito SDK 直接）         |
| 地図                 | Amazon Location Service + MapLibre GL JS |
| API クライアント生成 | Orval（OpenAPI → TypeScript）            |

### バックエンド

| 項目       | 技術                          |
| ---------- | ----------------------------- |
| 言語       | Go                            |
| ルーター   | chi                           |
| デプロイ   | API Gateway + Lambda（ARM64） |
| ランタイム | provided.al2023               |
| コード生成 | oapi-codegen（OpenAPI → Go）  |

### ローカル開発

| 項目           | 技術                              |
| -------------- | --------------------------------- |
| DB             | DynamoDB Local（docker-compose）  |

---

## 4. AWS サービス構成

### MVP

```
[ユーザー]
    │
    ├── Cognito（認証・JWT発行）← フロントから直接
    │
    ├── API Gateway（REST API）+ Cognito Authorizer
    │       │
    │       └── Lambda（ビジネスロジック）
    │               │
    │               ├── DynamoDB（データストア）
    │               └── S3（画像保存）
    │
    └── Amplify Hosting（フロントエンド）
```

### MVP後に追加

- Rekognition（画像審査）
- Location Service（経路計算）
- Bedrock（AI提案文生成）
- EventBridge → End User Messaging Push（プッシュ通知）

---

## 5. データモデル（DynamoDB）

### Toilets（トイレ情報）

| フィールド         | 型            | 説明                                                                            |
| ------------------ | ------------- | ------------------------------------------------------------------------------- |
| toiletId (PK)      | String (UUID) | トイレID                                                                        |
| name               | String        | コンビニ店舗名                                                                  |
| brand              | String        | ブランド（seven_eleven, family_mart, lawson, mini_stop, daily_yamazaki, other） |
| address            | String        | 住所                                                                            |
| lat                | Number        | 緯度                                                                            |
| lng                | Number        | 経度                                                                            |
| geohash            | String        | 位置検索用（GSI: GeoIndex）                                                     |
| imageUrl           | String        | 画像URL                                                                         |
| maleCount          | Number        | 男性用個数                                                                      |
| femaleCount        | Number        | 女性用個数                                                                      |
| multipurposeCount  | Number        | 多目的個数                                                                      |
| requiresPermission | Boolean       | 店員許可要否                                                                    |
| note               | String        | 自由記入欄                                                                      |
| status             | String        | active / removed（GSI: StatusIndex）                                            |
| reportCount        | Number        | 虚偽報告件数                                                                    |
| createdBy          | String        | 投稿者ユーザーID                                                                |
| createdAt          | String        | 作成日時                                                                        |
| updatedAt          | String        | 更新日時                                                                        |

### Reports（虚偽報告）— MVP後

| フィールド    | 型            | 説明                         |
| ------------- | ------------- | ---------------------------- |
| reportId (PK) | String (UUID) | 報告ID                       |
| toiletId      | String (UUID) | トイレID（GSI: ToiletIndex） |
| imageUrl      | String        | 証拠画像URL                  |
| reason        | String        | 報告理由                     |
| status        | String        | accepted / rejected          |
| createdBy     | String        | 報告者ユーザーID             |
| createdAt     | String        | 作成日時                     |

### Devices（プッシュ通知デバイス）— MVP後

| フィールド    | 型     | 説明                |
| ------------- | ------ | ------------------- |
| userId (PK)   | String | ユーザーID          |
| platform (SK) | String | ios / android / web |
| deviceToken   | String | FCM / APNs トークン |

---

## 6. 画面構成（フロントエンド）

### MVP

| 画面                | パス         | 概要                                           |
| ------------------- | ------------ | ---------------------------------------------- |
| ログイン / 新規登録 | /auth        | Cognito SDK で認証                             |
| マップ（ホーム）    | /            | トイレをピン表示。「今すぐトイレに行きたい！」ボタン |
| トイレ登録          | /toilets/new | コンビニ情報 + トイレ情報を入力して送信        |
| トイレ詳細          | /toilets/:id | トイレ情報表示                                 |

### MVP後

| 画面                | パス                    | 概要               |
| ------------------- | ----------------------- | ------------------ |
| トイレなし報告      | /toilets/:id/report     | 証拠画像 + 理由入力 |
| ナビゲーション      | /navigation             | ルート提案 + 地図ナビ |
| 通知設定            | /settings/notifications | プッシュ通知 ON/OFF |

---

## 7. API エンドポイント一覧

[Swagger UI で詳細を見る](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/Neptune-Progate-Hackathon-AWS/back/main/openapi.yml)

### MVP

| メソッド | パス                  | 概要                    | 認証 |
| -------- | --------------------- | ----------------------- | ---- |
| POST     | /images/presigned-url | 画像アップロードURL発行 | 要   |
| GET      | /toilets              | トイレ一覧              | 要   |
| POST     | /toilets              | トイレ登録              | 要   |
| GET      | /toilets/:id          | トイレ詳細              | 要   |

### MVP後

| メソッド | パス                    | 概要                 | 認証 |
| -------- | ----------------------- | -------------------- | ---- |
| GET      | /toilets/:id/reports    | 報告一覧             | 要   |
| POST     | /toilets/:id/reports    | トイレなし報告       | 要   |
| POST     | /navigation/route       | ルート提案           | 要   |
| PUT      | /notifications/device   | デバイス登録         | 要   |
| DELETE   | /notifications/device   | デバイス削除         | 要   |
| POST     | /notifications/location | 位置情報送信         | 要   |

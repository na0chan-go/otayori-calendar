# Architecture

## System Overview

```text
User
↓
Vue Frontend
↓
Go API
↓
PostgreSQL
↓
Google Calendar API
↓
Google Calendar
```

## Processing Flow

```text
1. ユーザーがGoogleログインする
2. おたより画像をアップロードする
3. 画像を保存する
4. OCR / AIで予定候補を抽出する
5. 抽出結果をDBにdraftとして保存する
6. ユーザーが内容を確認・編集する
7. Google Calendar APIで予定登録する
8. 登録済みイベントIDをDBに保存する
```

## Design Policy

### 1. AIによる完全自動登録はしない

AI/OCRは誤読する可能性があるため、必ずユーザー確認を挟んでからGoogleカレンダーへ登録する。

### 2. Google Calendar連携を先に実装する

OCRやAI抽出よりも、OAuthとCalendar API連携の方が技術的な不確実性が高いため、最初に手入力予定をGoogleカレンダーへ登録できる状態を作る。

### 3. 抽出結果はdraftとして保存する

AI抽出結果は確定データではなく、ユーザー確認前の候補として扱う。

### 4. 失敗状態を保持する

Calendar API登録に失敗した場合も、候補データを失わずfailedとして保存し、再実行できる設計にする。

## Components

### Frontend

- Googleログイン導線
- おたより画像アップロード画面
- OCR結果確認画面
- 予定候補一覧画面
- 予定編集画面
- Googleカレンダー登録画面

### Backend

- Google OAuth callback
- Google token management
- Letter upload API
- OCR / AI extraction API
- Extracted event management API
- Google Calendar registration API

### Database

- users
- google_tokens
- letters
- extracted_events

## Backend Dependency Direction

バックエンドは既存APIの挙動を維持しながら、以下の依存方向へ段階的に移行する。

```text
handler
  ↓
usecase
  ↓
domain

infrastructure
  └─ implements ports used by usecase
```

- `domain` はEcho、GORM、Google APIへ依存しない。
- `usecase` は処理フローを担当し、DBや外部APIはport/interface経由で利用する。
- `handler` はHTTP入力、認証ユーザー取得、レスポンス変換を担当する。
- `infrastructure` はGORM、Google Calendar API、Gemini、画像保存を担当する。

### Current Dependencies

現在は `internal/handler` が以下を直接扱っている。

- Echoのrequest/response
- GORMによる所有権確認・保存
- Google tokenの復号・更新
- Google Calendar APIの登録・存在確認
- 予定候補の登録可否と状態遷移

特に `extracted_event.go` と `manual_event.go` はHTTP処理と外部連携が混在しているため、予定登録から段階的に分離する。

### Target Directories

```text
backend/internal/
  domain/          エンティティ、状態遷移、ドメインルール
  usecase/         登録、抽出、同期、削除などの処理フロー
  port/            repository・外部サービスのinterface
  infrastructure/  GORM、Google Calendar、Gemini、画像保存の実装
  handler/         HTTP入力、認証ユーザー取得、レスポンス変換
```

### Planned Ports

- `ExtractedEventRepository`: 所有権を検証した予定候補取得、保存、一覧取得
- `ManualEventRepository`: 手入力予定の取得、保存、一覧取得
- `GoogleTokenRepository`: token取得、更新
- `CalendarGateway`: 予定登録、存在確認
- `LetterRepository`: おたより取得、保存、削除
- `ImageStorage`: 画像保存、取得、削除
- `EventExtractor`: AIによる予定候補抽出

### Migration Stages

1. 予定候補の登録可否・登録結果の状態遷移をdomainへ分離する。
2. 予定候補登録フローをusecaseへ移し、repositoryとCalendar gatewayをinterface化する。
3. 手入力予定登録・Calendar同期をusecaseへ移す。
4. 予定候補の確認・除外・復元をusecaseへ移す。
5. AI抽出とおたより管理をusecaseへ移す。

各段階で既存APIパス・レスポンス形式・状態遷移を維持し、小さなPRとして検証する。

### Current Migration Status

- 予定候補の登録可否・登録結果の状態遷移は `internal/domain/extractedevent` へ分離済み。
- 予定候補登録フローは `internal/usecase/extractedevent` へ分離済み。
- 登録フローが利用するCalendar gatewayとregistration repositoryは `internal/port/extractedevent` で定義済み。
- 予定候補の確認・除外・復元の状態遷移と保存フローはdomain/usecaseへ分離済み。
- 予定候補の編集可否・入力検証・更新後状態と保存フローはdomain/usecaseへ分離済み。
- AI抽出の入力選択・候補生成・トランザクション保存フローはusecase/portへ分離済み。
- AI抽出候補の構造・入力検証・簡易OCR抽出ルールは `internal/domain/extractedevent` へ分離済み。
- 外部AI出力のJSON解析とGORMモデル変換は移行用adapterとしてhandlerに残っている。
- おたより削除の所有権確認・画像隔離・DB削除・画像削除フローは `internal/usecase/letter` と `internal/port/letter` へ分離済み。
- GORMとGoogle Calendar APIのadapterは移行中のためhandler内に残っており、次段階でinfrastructureへ移す。

## Error Handling

| 想定する失敗 | 対策 |
|---|---|
| OCRが日付を誤読する | ユーザー確認を必須にする |
| AIが不要な予定を抽出する | ignoredステータスを用意する |
| Calendar API登録に失敗する | failedステータスを保存する |
| 同じ予定を2回登録する | google_calendar_event_idで重複防止する |
| 時刻が不明 | 終日予定 + 説明欄に原文保存 |
| 曜日と日付が矛盾 | 警告表示する |
| トークン期限切れ | refresh_tokenで再取得する |
| 個人情報がログに出る | ログ出力ルールを設ける |

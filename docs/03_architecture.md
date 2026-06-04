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

# おたよりカレンダー / Otayori Calendar

保育園のおたより画像から予定候補を抽出し、ユーザー確認後にGoogleカレンダーへ登録するWebアプリです。

## Overview

保育園のおたよりには、行事予定、提出期限、持ち物、注意事項など、家庭で管理すべき情報が散らばっています。
紙やPDFで届くことが多く、予定をGoogleカレンダーへ手入力する手間や、提出期限・持ち物の見落としが発生しやすい課題があります。

おたよりカレンダーは、画像アップロード、OCR/AIによる予定候補抽出、ユーザー確認、Googleカレンダー登録までを一連の流れで行うことを目的としたアプリです。

## Problem

- おたよりの予定をカレンダーへ手入力する必要がある
- 提出期限を見落としやすい
- 持ち物を前日に確認し忘れやすい
- 夫婦間で保育園情報を共有しづらい
- 過去のおたよりを探しにくい

## Features

### MVP

- Googleアカウントによるログイン
- おたより画像のアップロード
- OCR / AIによる予定候補の抽出
- 抽出結果の確認・編集
- Googleカレンダーへの予定登録
- 登録済み予定の一覧表示
- 原本画像と抽出結果の管理

### Out of Scope for MVP

- iPhone純正カレンダー連携
- Android純正カレンダー連携
- ICSファイル出力
- LINE通知
- 家族共有
- 複数園対応
- 完全自動登録

## Tech Stack

### Frontend

- Vue 3
- TypeScript
- Vite
- Element Plus

### Backend

- Go
- Echo
- GORM

### Database

- PostgreSQL

### External APIs

- Google OAuth
- Google Calendar API
- Gemini API

### Infrastructure

- Docker
- Docker Compose

## Architecture

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

### AIによる完全自動登録はしない

OCR/AIは誤読する可能性があるため、抽出結果をそのままGoogleカレンダーへ登録しません。
抽出結果は予定候補として保存し、ユーザー確認後に登録します。

### Google Calendar連携を先に実装する

このアプリの最終価値は、予定候補をGoogleカレンダーへ登録できることです。
そのため、OCR/AIより先にGoogle OAuthとCalendar API連携を実装します。

### 抽出結果はdraftとして保存する

AI抽出結果は確定データではなく、ユーザー確認前の候補として扱います。
状態管理により、未確認・確認済み・登録済み・除外・失敗を区別します。

## Documentation

- [Concept](docs/01_concept.md)
- [Requirements](docs/02_requirements.md)
- [Architecture](docs/03_architecture.md)
- [Database Design](docs/04_database.md)
- [Google Calendar Integration](docs/05_google_calendar.md)
- [AI Extraction](docs/06_ai_extraction.md)
- [Security](docs/07_security.md)
- [Roadmap](docs/08_roadmap.md)

## Portfolio Message

このアプリは、家庭内の実課題を題材にしながら、以下の技術要素を説明できるポートフォリオとして設計します。

- Google OAuth
- Google Calendar API
- 外部API連携
- AI/OCR活用
- Goバックエンド設計
- DB状態管理
- 誤登録防止設計
- 個人情報を扱うセキュリティ設計

## Development Order

1. Google OAuthログイン
2. 手入力予定のGoogleカレンダー登録
3. 登録済みevent_idの保存
4. おたより画像アップロード
5. OCR / AI抽出
6. 予定候補の確認・編集
7. Googleカレンダー登録
8. README / docs / demoの整備

## Local Development

### Prerequisites

- Go 1.25.8+
- Node.js 22+
- Docker / Docker Compose
- Google Cloud OAuth Client

### Google OAuth Setup

Google Cloud ConsoleでOAuth Clientを作成し、以下を設定します。

- Authorized JavaScript origins: `http://localhost:5173`
- Authorized redirect URIs: `http://localhost:8080/auth/google/callback`
- Scopes: `userinfo.email`, `userinfo.profile`, `calendar.events`

### Backend

```bash
docker compose up -d postgres
cd backend
cp .env.example .env
go mod tidy
go run ./cmd/api
```

`.env` の `GOOGLE_CLIENT_ID` と `GOOGLE_CLIENT_SECRET` には、Google Cloudで発行した値を設定してください。

### Frontend

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

ブラウザで `http://localhost:5173` を開き、「Googleでログイン」からOAuthログインを開始できます。

### Auth API

| Method | Path | Description |
|---|---|---|
| GET | `/auth/google/login` | Google OAuthログインを開始する |
| GET | `/auth/google/callback` | Google OAuth callbackを受け取り、ユーザーとトークンを保存する |
| GET | `/api/me` | ログイン中のアプリ内ユーザーを返す |
| POST | `/api/manual-events` | 手入力予定をGoogleカレンダーへ登録し、event IDを保存する |
| POST | `/auth/logout` | セッションCookieを削除する |

## License

MIT

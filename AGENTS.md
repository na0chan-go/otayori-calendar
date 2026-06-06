# AGENTS.md

## Project Overview

保育園のおたより画像からAIで予定候補を抽出し、保護者が内容を確認してからGoogleカレンダーへ登録するWebアプリです。

AI/OCRによる誤登録を防ぎながら、おたよりの予定を手入力する負担と見落としを減らすことを目的とします。

作業時は [省トークン作業ガイド](docs/10_token_efficient_workflow.md) を参照し、issue開始・PR・マージ時は [Issue Workflow](docs/09_issue_workflow.md) に従います。

## Tech Stack

- Frontend: Vue 3, TypeScript, Vite
- Backend: Go, Echo, GORM
- Database: PostgreSQL
- Test用DB: SQLite
- External APIs: Google OAuth, Google Calendar API, Gemini API
- Local infrastructure: Docker Compose

## Architecture Rules

- AI抽出結果は `draft` として保存し、ユーザー確認前にGoogleカレンダーへ自動登録しない。
- Googleカレンダー登録前に、ユーザーが予定候補を確認・編集できる状態を保つ。
- API登録失敗時はデータを失わず、`failed` 状態として保持する。
- Google Calendar Event IDを保存し、重複登録を防止する。
- おたより・予定候補などのユーザーデータは、必ずログインユーザーの所有権を検証する。
- フロントとバックエンドで、状態ごとの操作可否を一致させる。
- APIレスポンスでは内部保存パスや秘密情報を返さない。

## Directory Structure

- `frontend/src/`: Vueフロントエンドの画面・スタイル
- `backend/cmd/api/`: APIサーバーのエントリーポイント
- `backend/internal/handler/`: HTTP API、認証、外部API連携処理
- `backend/internal/model/`: GORMモデル
- `backend/internal/service/`: AI抽出などのサービス処理
- `backend/internal/config/`: 環境変数・設定読み込み
- `backend/internal/database/`: DB接続・migration実行
- `backend/migrations/`: PostgreSQL migration
- `backend/storage/`: ローカルのおたより画像保存先
- `docs/`: 要件、設計、セキュリティ、作業フロー

## Commands

PostgreSQLを起動する。

```bash
docker compose up -d postgres
```

バックエンドを起動する。

```bash
cd backend
go run ./cmd/api
```

フロントエンドを起動する。

```bash
cd frontend
npm run dev
```

バックエンドテストを実行する。

```bash
cd backend
go test ./...
```

フロントエンドの型チェック・ビルドを実行する。

```bash
cd frontend
npm run build
```

差分の空白エラーを確認する。

```bash
git diff --check
```

専用のlintコマンドは現在未設定です。Goコード編集時は `gofmt` を実行してください。

## Testing Policy

- バックエンド変更後は `go test ./...` を実行する。
- フロントエンド変更後は `npm run build` を実行する。
- マージ前に `git diff --check` を実行する。
- バックエンド変更後の手動確認前は、必ずバックエンドを再起動する。
- フロントエンド変更は、デスクトップとスマホ幅で主要操作を確認する。
- 認証・所有権・状態遷移・エラー時の挙動を重点的に確認する。
- ユーザーの手動確認完了後にコミット・PR・マージへ進む。

## Coding Style

- Goコードは `gofmt` に従う。
- Goのhandlerでは、認証・所有権検証・入力検証・永続化・レスポンスを明確に分ける。
- エラー時は適切なHTTPステータスと、既存形式に沿ったメッセージを返す。
- TypeScriptでは既存の型を再利用し、`any` の追加を避ける。
- Vueでは既存のComposition API構成と状態管理方法に合わせる。
- CSSは既存のCSS変数・コンポーネントクラス・レスポンシブ方針を再利用する。
- コメントはコードだけでは意図が分かりにくい箇所に限定する。
- コミットメッセージは日本語のprefix付きにする。例: `feat: ...`, `fix: ...`, `docs: ...`

## Security

- `.env` をコミットしない。
- Google OAuthのclient secretをコードに直書きしない。
- tokenやrefresh tokenの扱いに注意し、ログ・レスポンス・テスト出力へ含めない。
- 実データ・個人情報をテストコードに含めない。
- access tokenとrefresh tokenは安全に保存し、最小権限のOAuthスコープを使う。
- OCR全文、子どもの名前、保育園名、画像URL、予定詳細をログに出さない。
- 画像・おたより・予定候補へのアクセスでは、必ずログインユーザーの所有権を検証する。
- 外部AI APIへ送信する情報を必要最小限にする。

## Do Not

- 勝手に大規模リファクタリングしない。
- 既存APIのレスポンス形式を無断で変えない。
- UIライブラリを勝手に追加しない。
- 不要な依存パッケージを追加しない。
- 関係ない変更やユーザーの未コミット変更を削除・巻き戻ししない。
- `git reset --hard` や `git checkout --` などの破壊的コマンドを使わない。
- 登録済み予定をGoogleカレンダーと同期せずにローカルだけ変更しない。
- ユーザー確認なしで個人情報を含む実データを削除・外部送信しない。

## Documentation

仕様変更時は、変更内容に応じて以下を更新する。

- 要件・状態・操作ルール: `docs/02_requirements.md`
- システム構成・設計方針: `docs/03_architecture.md`
- DB・status・migration: `docs/04_database.md`
- Google OAuth・Calendar連携: `docs/05_google_calendar.md`
- AI抽出仕様: `docs/06_ai_extraction.md`
- 認証・個人情報・token: `docs/07_security.md`
- 開発優先度・今後の計画: `docs/08_roadmap.md`
- issue対応フロー: `docs/09_issue_workflow.md`
- AIエージェントの省トークン運用: `docs/10_token_efficient_workflow.md`
- API一覧・起動方法・主要機能: `README.md`

# Token Efficient Workflow

AIエージェントが品質を落とさず、不要なコンテキスト読み込みと会話量を減らすための作業ガイドです。

## 基本方針

- 最初からリポジトリ全体を読まない。
- issueの完了条件を基準に、必要なコードとdocsだけを読む。
- 一度確認した事実は短く要約して保持し、同じコマンドや説明を繰り返さない。
- 中間報告と完了報告は、判断や確認に必要な情報だけを書く。
- 詳細調査は、テスト失敗・仕様不明・影響範囲不明の場合にだけ広げる。

## 作業開始時の最小確認

```bash
git status -sb
rg --files
```

対象issueを確認後、キーワード検索で関連箇所を絞る。

```bash
rg -n "<機能名|APIパス|状態名>" frontend backend docs
```

ファイル全体ではなく、まず該当箇所の周辺だけを読む。

```bash
sed -n '<start>,<end>p' <file>
```

## Issue別の参照表

| 作業内容 | 最初に読むもの |
|---|---|
| フロントUI・UX | `frontend/src/App.vue`, `frontend/src/style.css`, `docs/02_requirements.md` |
| API・handler | 対象の `backend/internal/handler/*.go`, 対応する `*_test.go`, `docs/03_architecture.md` |
| DB・migration | `backend/migrations/`, `backend/internal/model/`, `docs/04_database.md` |
| Google OAuth・Calendar | 対象handler/service, `docs/05_google_calendar.md`, `docs/07_security.md` |
| AI抽出 | `backend/internal/handler/extraction.go`, 関連service, `docs/06_ai_extraction.md` |
| 認証・所有権 | `backend/internal/handler/auth.go`, 対象handler, `docs/07_security.md` |
| issue開始・PR・マージ | `docs/09_issue_workflow.md` |

参照表以外のdocsは、関連する仕様が見つからない場合にだけ読む。

## 調査の止めどき

以下を説明できたら実装へ進む。

- 現在の挙動
- 期待する挙動
- 変更するファイル
- 既存機能への主な影響
- 確認方法

同じ結論を補強するだけの追加検索は行わない。

## 実装中の進め方

- issueの完了条件を満たす最小差分を優先する。
- 関係ないリファクタリングを混ぜない。
- 既存の関数・状態・CSSルールを再利用する。
- 新しく見つけた別問題は、現在のissueを妨げない限り別issueへ切り出す。
- 大きなファイルを再読する代わりに、変更箇所と呼び出し元を `rg` で確認する。

## 確認コマンド

変更範囲に応じて必要な確認だけを先に行い、マージ前に基本確認を一度まとめて行う。

```bash
cd backend && go test ./...
cd frontend && npm run build
git diff --check
```

バックエンド変更後は、手動確認前に必ずバックエンドを再起動する。

## コミュニケーション

中間報告は原則1〜2文で、以下のどれかを伝える。

- 何を調べているか
- 何が分かったか
- 次に何を変更・確認するか
- ユーザー確認が必要な理由

完了報告は、変更結果・確認結果・残るリスクだけを簡潔に伝える。

## コンテキスト引き継ぎ

長い作業では、次の情報だけを短く保持する。

- 対象issueと現在のブランチ
- 実装済み内容
- 変更ファイル
- 実行済みテスト
- 手動確認の結果
- 次の作業
- 未解決事項

過去の会話全文やコマンド出力全文は引き継がない。


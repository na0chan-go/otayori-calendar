# Issue Workflow

GitHub issue に取り掛かってから、PR をマージして issue をクローズするまでの標準手順です。

調査・読み込み・報告を必要最小限にする方法は、[Token Efficient Workflow](10_token_efficient_workflow.md) を参照する。

## 目的

- issue ごとの作業範囲を明確にする
- 実装、確認、PR、マージの流れを毎回そろえる
- チェックリスト更新や自己レビューを忘れないようにする

## 1. Issue の確認

1. 対象 issue の本文、チェックリスト、完了条件と、[Issue別の参照表](10_token_efficient_workflow.md#issue別の参照表) にある関連ドキュメントを読む。
2. 既存の open issue や直近の PR と重複していないか確認する。
3. 実装範囲が曖昧な場合は、影響が大きい選択肢だけユーザーに確認する。
4. 実装方針が決まったら、専用ブランチを作る。

```bash
git switch main
git pull --ff-only
git switch -c codex/issue-<number>-<short-description>
```

## 2. 実装

1. まず関連コードを読む。
2. issue の完了条件を満たす最小単位で実装する。
3. 既存の設計、命名、UI トーンに合わせる。
4. 既存データで起きる不整合も見つけた場合は、再発しないようコード側で補正する。
5. 必要に応じて README や docs も更新する。

## 3. ローカル確認

実装後は、最低限以下を確認する。

```bash
cd backend
go test ./...
```

```bash
cd frontend
npm run build
```

バックエンド変更がある場合は、実装後に必ずバックエンドを再起動する。
古い `go run` プロセスが残っていると、画面からは新しい API が呼ばれているように見えても、実際には古いコードが動くことがある。

```bash
lsof -nP -iTCP:8080 -sTCP:LISTEN
kill <backend-pid>
cd backend
go run ./cmd/api
```

フロント変更がある場合は、ローカル画面でも手動確認する。
自動テストと build が通っていても、実際の画面操作で状態更新、ボタンの disabled、メッセージ表示、既存データの扱いを確認する。

```bash
docker compose up -d postgres
cd backend
# バックエンドが未起動なら起動する
go run ./cmd/api
```

```bash
cd frontend
npm run dev
```

ブラウザで `http://localhost:5173` を開き、issue の完了条件に沿って動作確認する。
確認後は、ユーザーにもローカル確認を依頼し、問題ないことを確認してからコミットや PR 作成に進む。

## 4. 自己レビュー

PR 作成前または PR 作成後に、次の観点で自己レビューする。

- issue のチェックリストをすべて満たしているか
- 既存機能を壊していないか
- 認証やユーザー所有データの境界が守られているか
- エラー時の表示や状態更新が不自然ではないか
- 一部失敗や既存データ不整合など、現実に起きそうなケースを扱えているか
- テストや build が通っているか

問題があればコミット前に修正する。

## 5. コミット

作業範囲だけをステージする。関係ない変更が混ざっている場合は含めない。

```bash
git status -sb
git diff --stat
git add <changed-files>
git commit -m "feat: 日本語の変更概要"
```

コミットメッセージは prefix 付きの日本語を基本にする。

- `feat: ...`
- `fix: ...`
- `docs: ...`
- `test: ...`
- `refactor: ...`

## 6. PR 作成

ブランチを push して、対象 issue に紐づく PR を作る。

```bash
git push -u origin $(git branch --show-current)
```

PR 本文には最低限以下を書く。

- `Closes #<issue-number>`
- 変更内容
- ユーザーや開発上の影響
- 不具合修正なら原因
- 確認したコマンド
- ローカル動作確認の有無

PR は、ユーザーから明示がない限り draft ではなく open で作る運用にする。

## 7. PR 上の自己レビュー

自分の PR は GitHub 上で approve できないため、PR コメントとして自己レビューを残す。

コメントには以下を書く。

- 重点的に確認した箇所
- テストと build の結果
- ローカル動作確認の結果
- 残っている懸念があればその内容

## 8. Issue チェックリスト更新

PR の内容が issue のチェックリストを満たしている場合、issue 本文の該当項目を `[x]` に更新する。

チェックリスト更新は、PR マージ前に行う。

## 9. マージ

PR が mergeable で、テストや自己レビューに問題がなければマージする。

```bash
gh pr view <pr-number> --json mergeable,mergeStateStatus,state,isDraft,statusCheckRollup
```

基本は squash merge を使う。

マージ後、`Closes #<issue-number>` により issue が close されたことを確認する。

```bash
gh issue view <issue-number> --json state,stateReason,url
```

## 10. ローカル同期

マージ後はローカルの `main` を最新にする。

```bash
git switch main
git pull --ff-only
git status -sb
```

必要なら作業用ブランチは削除する。

```bash
git branch -d codex/issue-<number>-<short-description>
```

## 完了報告

最後にユーザーへ簡潔に報告する。

- PR URL
- merge commit
- 確認したコマンド
- issue が close されたこと
- ローカル `main` を同期したこと

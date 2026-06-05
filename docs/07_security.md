# Security

## 扱う情報

本アプリでは以下の個人情報を扱う可能性がある。

- 子どもの名前
- 保育園名
- クラス名
- 行事予定
- 写真
- 連絡事項
- Googleアカウント情報
- Google Calendar APIトークン

## Security Requirements

- ユーザーは自分のおたよりのみ閲覧できる
- Google OAuthトークンは暗号化して保存する
- アップロード画像は認証済みユーザーのみアクセスできる
- 不要になった画像を削除できる
- AI APIに送信する情報を最小化する
- ログに個人情報やトークンを出力しない

## Token Handling

- access_tokenは期限付きで扱う
- refresh_tokenは暗号化して保存する
- トークンをログに出さない
- カレンダー登録権限は最小スコープにする

## Access Control

- すべてのlettersはuser_idに紐づける
- 画像表示APIは `letters.user_id` とログインユーザーIDを照合する
- すべてのextracted_eventsはletter_id経由でユーザー所有権を検証する
- 他ユーザーの画像・OCR結果・予定候補にはアクセスできないようにする

## Logging Policy

ログに出してよい情報と出してはいけない情報を分ける。

### ログに出してよい情報

- 処理成功/失敗
- エラー種別
- リクエストID
- ユーザーIDの内部ID

### ログに出してはいけない情報

- access_token
- refresh_token
- OCR全文
- 子どもの名前
- 保育園名
- 画像URLの署名付きURL
- Googleカレンダー予定の詳細本文

## Privacy Policy for MVP

MVPでは個人利用を前提とし、外部公開時には以下を整備する。

- プライバシーポリシー
- 利用規約
- データ削除機能
- Google OAuth審査対応
- 外部AI APIへのデータ送信に関する説明

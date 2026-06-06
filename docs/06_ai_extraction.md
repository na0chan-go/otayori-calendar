# AI Extraction

## Purpose

OCR結果から、Googleカレンダーに登録できる予定候補を抽出する。

## Provider

外部AI APIには Gemini API を使用し、既定モデルは `gemini-2.5-flash` とする。

採用理由:

- テキストと画像の両方を入力できる
- 構造化JSON出力を指定できる
- Google Cloud / Google Calendar連携と同じGoogle系サービスで構成できる

## Extraction Flow

1. OCRテキストが入力されている場合は、OCRテキストだけをGemini APIへ送る
2. OCRテキストが空の場合は、アップロード済み画像をGemini APIへ送る
3. Gemini APIには `application/json` と既存の出力スキーマを指定する
4. APIレスポンスを既存バリデーションで再検証する
5. 検証済みの予定候補だけをdraft保存する

`GEMINI_API_KEY` が未設定の場合は、既存のOCRテキスト簡易抽出を利用する。
外部APIに失敗した場合は予定候補やおたよりデータを変更せず、画面にエラーを表示する。

## Input Example

```text
6月12日（金）身体測定を行います。
朝は薄着で登園してください。

6月18日（木）はお弁当の日です。
水筒、レジャーシートを持たせてください。
```

## Output Schema

AIの出力はJSONに固定する。

```json
{
  "events": [
    {
      "title": "身体測定",
      "date": "2026-06-12",
      "start_time": null,
      "end_time": null,
      "is_all_day": true,
      "location": "保育園",
      "description": "朝は薄着で登園してください。",
      "confidence": 0.86,
      "source_text": "6月12日（金）身体測定を行います。朝は薄着で登園してください。"
    }
  ]
}
```

## Extraction Rules

- 日付があるものを予定候補とする
- 時間がない場合は終日予定とする
- 「午前中」「登園時」など曖昧な時間はdescriptionに残す
- 持ち物はdescriptionに含める
- 提出期限は予定として抽出する
- confidenceが低いものは登録候補にするが、警告表示する
- source_textには抽出根拠となる原文を保存する

## Date Handling

- 年が書かれていない場合は、発行日または現在日から推定する
- 和暦が書かれている場合は西暦へ変換する
- 曜日と日付が矛盾している場合は日付を優先し、UIで警告する

## Safety Policy

AI抽出結果は必ずユーザーが確認する。
AI出力をそのままGoogleカレンダーに登録しない。
OCR全文、画像データ、Gemini APIのレスポンス本文はログに出さない。
OCRテキストが入力されている場合は画像を送信せず、外部APIへ送る情報を最小化する。

## Prompt Policy

AIに自由文で返させず、JSON形式で返させる。
不明な値は推測しすぎず、nullとして返す。

```text
あなたは保育園のおたよりから予定候補を抽出するアシスタントです。
次のOCRテキストから、日付がある予定候補だけを抽出してください。
出力は必ずJSONのみとし、説明文やMarkdownを含めないでください。
不明な値はnullにしてください。

JSON schema:
{
  "events": [
    {
      "title": "string",
      "date": "YYYY-MM-DD",
      "start_time": "HH:MM|null",
      "end_time": "HH:MM|null",
      "is_all_day": true,
      "location": "string",
      "description": "string",
      "confidence": 0.0,
      "source_text": "string"
    }
  ]
}
```

## Validation

API側ではAI出力をそのまま保存せず、以下を検証する。

- JSONとしてパースできること
- eventsが配列であること
- titleが空でないこと
- dateが日付として妥当であること
- confidenceが0から1の範囲であること
- start_time / end_time が指定される場合は時刻形式として妥当であること

## Configuration

```env
GEMINI_API_KEY=
GEMINI_MODEL=gemini-2.5-flash
```

- `GEMINI_API_KEY`: Google AI Studioなどで発行したGemini APIキー
- `GEMINI_MODEL`: 使用するGeminiモデル。未指定時は `gemini-2.5-flash`

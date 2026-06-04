# AI Extraction

## Purpose

OCR結果から、Googleカレンダーに登録できる予定候補を抽出する。

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

## Prompt Policy

AIに自由文で返させず、JSON形式で返させる。
不明な値は推測しすぎず、nullとして返す。

## Validation

API側ではAI出力をそのまま保存せず、以下を検証する。

- JSONとしてパースできること
- eventsが配列であること
- titleが空でないこと
- dateが日付として妥当であること
- confidenceが0から1の範囲であること
- start_time / end_time が指定される場合は時刻形式として妥当であること

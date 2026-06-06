# Google Calendar Integration

## OAuth Scope

MVPでは、予定作成・編集に必要な最小限のスコープを使う。

```text
https://www.googleapis.com/auth/calendar.events
```

## Calendar ID

MVPではログインユーザーのメインカレンダーを対象にする。

```text
primary
```

将来的には、ユーザーが登録先カレンダーを選択できるようにする。

## All-day Event

日付のみ抽出された場合は終日予定として登録する。

```json
{
  "summary": "身体測定",
  "start": {
    "date": "2026-06-12",
    "timeZone": "Asia/Tokyo"
  },
  "end": {
    "date": "2026-06-13",
    "timeZone": "Asia/Tokyo"
  }
}
```

終日予定の場合、Google Calendar APIでは `end.date` に翌日を指定する。

## Timed Event

開始時刻と終了時刻がある場合は時刻付き予定として登録する。

```json
{
  "summary": "保護者会",
  "location": "保育園",
  "description": "持ち物：筆記用具",
  "start": {
    "dateTime": "2026-06-12T15:00:00+09:00",
    "timeZone": "Asia/Tokyo"
  },
  "end": {
    "dateTime": "2026-06-12T16:00:00+09:00",
    "timeZone": "Asia/Tokyo"
  }
}
```

## Duplicate Prevention

Google Calendar APIで登録後、返却されたevent_idを保存する。

```text
google_calendar_event_id
```

この値が存在する予定は再登録しない。

## Registration Rules

| 抽出結果 | 登録方法 |
|---|---|
| 日付のみ | 終日予定 |
| 時間あり | 時刻付き予定 |
| 午前中 | 終日予定 + descriptionに原文保存 |
| 登園時 | 終日予定 + descriptionに原文保存 |
| 提出期限 | 終日予定 |
| 持ち物 | belongingsへ保存し、登録時にdescriptionへ「持ち物: ...」として反映 |
| 提出期限 | submission_deadlineへ保存し、登録時にdescriptionへ「提出期限: ...」として反映 |

## Failure Handling

- 登録成功時は `status = registered` にする
- 登録失敗時は `status = failed` にする
- 失敗理由はログまたは別カラムで管理する
- failedの予定は再登録できるようにする
- Googleカレンダー上で削除済みの場合は `status = deleted` にする
- deletedの予定は重複ではないため、再登録できるようにする

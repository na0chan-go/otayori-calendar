# Database Design

## users

Googleアカウントとアプリ内ユーザーを紐づける。

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY,
  google_user_id TEXT UNIQUE NOT NULL,
  email TEXT NOT NULL,
  name TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

## google_tokens

Google Calendar API用のトークンを保存する。

```sql
CREATE TABLE google_tokens (
  user_id UUID PRIMARY KEY REFERENCES users(id),
  access_token TEXT NOT NULL,
  refresh_token TEXT,
  expiry TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

## letters

アップロードされたおたより画像とOCR結果を保存する。

```sql
CREATE TABLE letters (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  title TEXT,
  image_path TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  file_size BIGINT NOT NULL,
  ocr_text TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

## extracted_events

AIによって抽出された予定候補を保存する。

```sql
CREATE TABLE extracted_events (
  id UUID PRIMARY KEY,
  letter_id UUID NOT NULL REFERENCES letters(id),
  title TEXT NOT NULL,
  event_date DATE NOT NULL,
  start_time TIME,
  end_time TIME,
  is_all_day BOOLEAN NOT NULL DEFAULT true,
  location TEXT,
  description TEXT,
  confidence NUMERIC(3,2),
  source_text TEXT,
  google_calendar_event_id TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

## status

| status | 意味 |
|---|---|
| draft | AI抽出直後 |
| confirmed | ユーザー確認済み |
| registered | Googleカレンダー登録済み |
| ignored | ユーザーが除外 |
| failed | 登録失敗 |
| deleted | Googleカレンダー上で削除済み |

## Indexes

```sql
CREATE INDEX idx_letters_user_id ON letters(user_id);
CREATE INDEX idx_extracted_events_letter_id ON extracted_events(letter_id);
CREATE INDEX idx_extracted_events_status ON extracted_events(status);
CREATE INDEX idx_extracted_events_event_date ON extracted_events(event_date);
CREATE INDEX idx_extracted_events_google_calendar_event_id ON extracted_events(google_calendar_event_id);
```

## Notes

- `google_calendar_event_id` が存在する予定は再登録しない。
- `source_text` にはAIが予定候補と判断した根拠となる原文を保存する。
- `confidence` はAI抽出結果の確信度として扱い、低い場合はUIで警告表示する。
- token類は平文保存せず、実装時には暗号化を検討する。
- `letters.image_path` は内部保存パスであり、APIレスポンスでは直接返さない。

## manual_events

OCR/AI抽出より先に、手入力した予定をGoogle Calendarへ登録するための予定データを保存する。

```sql
CREATE TABLE manual_events (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  event_date DATE NOT NULL,
  start_at TIMESTAMPTZ,
  end_at TIMESTAMPTZ,
  is_all_day BOOLEAN NOT NULL DEFAULT true,
  location TEXT,
  description TEXT,
  time_zone TEXT NOT NULL DEFAULT 'Asia/Tokyo',
  google_calendar_event_id TEXT,
  status TEXT NOT NULL DEFAULT 'registered',
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

- `google_calendar_event_id` にはGoogle Calendar APIで作成されたevent IDを保存する。
- 終日予定は `event_date` と `is_all_day` で表現する。
- 時刻付き予定は `start_at` / `end_at` に保存する。
- Googleカレンダー上で手動削除された予定は `status = deleted` として扱う。

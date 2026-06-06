export type ViewName = 'home' | 'letters' | 'candidates' | 'calendar'

export type User = {
  id: string
  email: string
  name: string
  created_at: string
}

export type Letter = {
  id: string
  title: string
  mime_type: string
  file_size: number
  image_url: string
  created_at: string
  object_url?: string
}

export type ExtractedEvent = {
  id: string
  letter_id: string
  title: string
  event_date: string
  start_time: string | null
  end_time: string | null
  is_all_day: boolean
  location: string
  description: string
  confidence: number
  source_text: string
  status: string
}

export type ExtractedEventDraft = {
  title: string
  event_date: string
  start_time: string
  end_time: string
  is_all_day: boolean
  location: string
  description: string
}

export type BulkExtractedEventResult = {
  id: string
  status: 'success' | 'failed'
  message?: string
  event?: ExtractedEvent
}

export type BulkExtractedEventsResponse = {
  events: ExtractedEvent[]
  results: BulkExtractedEventResult[]
  summary: {
    success: number
    failed: number
  }
}

export type CalendarEvent = {
  id: string
  source_type: 'manual' | 'extracted'
  title: string
  event_date: string
  start_at: string | null
  end_at: string | null
  is_all_day: boolean
  location: string
  description: string
  time_zone: string
  google_calendar_event_id: string
  status: 'registered' | 'failed' | 'deleted'
  created_at: string
  updated_at: string
}

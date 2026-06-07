import { computed, ref, type Ref } from 'vue'
import type { CalendarEvent } from '../types'
import { apiBaseUrl } from './api'
import { toUserErrorMessage } from '../utils/requestError'

export function useCalendarEvents(errorMessage: Ref<string>) {
  const calendarEvents = ref<CalendarEvent[]>([])
  const calendarMessage = ref('')
  const eventMessage = ref('')
  const retryingCalendarEventId = ref('')
  const savingEvent = ref(false)
  const showManualEventForm = ref(false)
  const manualEvent = ref(emptyManualEvent())

  const attentionCalendarCount = computed(
    () => calendarEvents.value.filter((event) => event.status === 'failed' || event.status === 'deleted').length,
  )
  const hasUnsavedManualEvent = computed(() =>
    Object.entries(manualEvent.value).some(([key, value]) => key !== 'is_all_day' && value !== ''),
  )

  async function loadCalendarEvents() {
    const response = await fetch(`${apiBaseUrl}/api/calendar-events`, { credentials: 'include' })
    if (!response.ok) throw new Error('登録済み予定一覧を取得できませんでした')

    const body = (await response.json()) as { events: CalendarEvent[] }
    calendarEvents.value = body.events
  }

  async function retryCalendarEvent(event: CalendarEvent) {
    retryingCalendarEventId.value = event.id
    calendarMessage.value = ''
    errorMessage.value = ''

    try {
      const response = await fetch(`${apiBaseUrl}/api/manual-events/${event.id}/retry`, {
        method: 'POST',
        credentials: 'include',
      })
      if (!response.ok) {
        const body = (await response.json().catch(() => null)) as { message?: string } | null
        throw new Error(body?.message ?? '予定の再登録に失敗しました')
      }

      calendarMessage.value = '失敗していた予定をGoogleカレンダーへ登録しました'
      await loadCalendarEvents()
    } catch (error) {
      errorMessage.value = toUserErrorMessage(error, '予定の再登録でエラーが発生しました')
    } finally {
      retryingCalendarEventId.value = ''
    }
  }

  async function createManualEvent() {
    savingEvent.value = true
    eventMessage.value = ''
    errorMessage.value = ''

    try {
      const response = await fetch(`${apiBaseUrl}/api/manual-events`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(manualEvent.value),
      })
      if (!response.ok) {
        const body = (await response.json().catch(() => null)) as { message?: string } | null
        throw new Error(body?.message ?? '予定を登録できませんでした')
      }

      eventMessage.value = 'Googleカレンダーに登録しました'
      await loadCalendarEvents()
      manualEvent.value = emptyManualEvent()
    } catch (error) {
      errorMessage.value = toUserErrorMessage(error, '予定登録でエラーが発生しました')
    } finally {
      savingEvent.value = false
    }
  }

  function resetCalendarEvents() {
    calendarEvents.value = []
    calendarMessage.value = ''
    eventMessage.value = ''
  }

  function resetManualEvent() {
    manualEvent.value = emptyManualEvent()
  }

  return {
    attentionCalendarCount,
    calendarEvents,
    calendarMessage,
    calendarStatusLabel,
    canRetryCalendarEvent,
    createManualEvent,
    eventMessage,
    formatCalendarEventTime,
    hasUnsavedManualEvent,
    loadCalendarEvents,
    manualEvent,
    resetCalendarEvents,
    resetManualEvent,
    retryCalendarEvent,
    retryingCalendarEventId,
    savingEvent,
    showManualEventForm,
  }
}

function emptyManualEvent() {
  return {
    title: '',
    event_date: '',
    is_all_day: true,
    start_time: '',
    end_time: '',
    location: '',
    description: '',
  }
}

function calendarStatusLabel(status: string) {
  const labels: Record<string, string> = {
    registered: '登録済み',
    failed: '登録失敗',
    deleted: 'カレンダーから削除済み',
  }
  return labels[status] ?? status
}

function canRetryCalendarEvent(event: CalendarEvent) {
  return (event.status === 'failed' || event.status === 'deleted') && event.source_type === 'manual'
}

function formatCalendarEventTime(event: CalendarEvent) {
  if (event.is_all_day) return '終日'
  const options = { hour: '2-digit', minute: '2-digit' } as const
  const start = event.start_at ? new Date(event.start_at).toLocaleTimeString('ja-JP', options) : '未設定'
  const end = event.end_at ? new Date(event.end_at).toLocaleTimeString('ja-JP', options) : '未設定'
  return `${start} - ${end}`
}

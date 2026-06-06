import { computed, onMounted, ref } from 'vue'
import type {
  BulkExtractedEventsResponse,
  CalendarEvent,
  ExtractedEvent,
  ExtractedEventDraft,
  Letter,
  ViewName,
  User,
} from '../types'

export function useOtayoriCalendar() {
const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'
const user = ref<User | null>(null)
const loading = ref(true)
const errorMessage = ref('')
const eventMessage = ref('')
const savingEvent = ref(false)
const letterMessage = ref('')
const uploadingLetter = ref(false)
const deletingLetterId = ref('')
const extractingLetterId = ref('')
const savingCandidateId = ref('')
const registeringCandidateId = ref('')
const selectedCandidateIds = ref<string[]>([])
const bulkCandidateAction = ref('')
const candidateMessage = ref('')
const calendarMessage = ref('')
const retryingCalendarEventId = ref('')
const letters = ref<Letter[]>([])
const extractedEvents = ref<ExtractedEvent[]>([])
const calendarEvents = ref<CalendarEvent[]>([])
const eventDrafts = ref<Record<string, ExtractedEventDraft>>({})
const ocrTextByLetter = ref<Record<string, string>>({})
const letterTitle = ref('')
const letterImage = ref<File | null>(null)
const activeView = ref<ViewName>('home')
const showManualEventForm = ref(false)
const manualEvent = ref({
  title: '',
  event_date: '',
  is_all_day: true,
  start_time: '',
  end_time: '',
  location: '',
  description: '',
})

const pendingCandidateCount = computed(
  () => extractedEvents.value.filter((event) => event.status === 'draft').length,
)
const readyCandidateCount = computed(
  () => extractedEvents.value.filter((event) => canRegisterExtractedEvent(event)).length,
)
const attentionCalendarCount = computed(
  () => calendarEvents.value.filter((event) => event.status === 'failed' || event.status === 'deleted').length,
)

const extractedStatusLabels: Record<string, string> = {
  draft: '未確認',
  confirmed: '確認済み',
  ignored: '除外済み',
  registered: '登録済み',
  failed: '登録失敗',
  deleted: '削除済み',
}

const calendarStatusLabels: Record<string, string> = {
  registered: '登録済み',
  failed: '登録失敗',
  deleted: 'カレンダーから削除済み',
}

function extractedStatusLabel(status: string) {
  return extractedStatusLabels[status] ?? status
}

function calendarStatusLabel(status: string) {
  return calendarStatusLabels[status] ?? status
}

function switchView(view: ViewName) {
  activeView.value = view
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

async function loadMe() {
  loading.value = true
  errorMessage.value = ''

  try {
    const response = await fetch(`${apiBaseUrl}/api/me`, {
      credentials: 'include',
    })

    if (response.status === 401) {
      user.value = null
      return
    }
    if (!response.ok) {
      throw new Error('ユーザー情報を取得できませんでした')
    }

    const body = (await response.json()) as { user: User }
    user.value = body.user
    await Promise.all([loadLetters(), loadExtractedEvents(), loadCalendarEvents()])
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '予期しないエラーが発生しました'
  } finally {
    loading.value = false
  }
}

function loginWithGoogle() {
  window.location.href = `${apiBaseUrl}/auth/google/login`
}

async function logout() {
  await fetch(`${apiBaseUrl}/auth/logout`, {
    method: 'POST',
    credentials: 'include',
  })
  user.value = null
  clearLetterObjectUrls()
  letters.value = []
  extractedEvents.value = []
  calendarEvents.value = []
  eventDrafts.value = {}
  selectedCandidateIds.value = []
  ocrTextByLetter.value = {}
}

function onLetterImageChange(event: Event) {
  const input = event.target as HTMLInputElement
  letterImage.value = input.files?.[0] ?? null
}

async function uploadLetter() {
  if (!letterImage.value) {
    errorMessage.value = '画像を選択してください'
    return
  }

  uploadingLetter.value = true
  letterMessage.value = ''
  errorMessage.value = ''

  try {
    const formData = new FormData()
    formData.append('image', letterImage.value)
    formData.append('title', letterTitle.value)

    const response = await fetch(`${apiBaseUrl}/api/letters`, {
      method: 'POST',
      credentials: 'include',
      body: formData,
    })
    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as { message?: string } | null
      throw new Error(body?.message ?? 'おたより画像をアップロードできませんでした')
    }

    letterTitle.value = ''
    letterImage.value = null
    letterMessage.value = 'おたより画像をアップロードしました'
    await loadLetters()
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : 'おたより画像のアップロードでエラーが発生しました'
  } finally {
    uploadingLetter.value = false
  }
}

async function extractEvents(letter: Letter) {
  const ocrText = ocrTextByLetter.value[letter.id]?.trim()

  extractingLetterId.value = letter.id
  letterMessage.value = ''
  errorMessage.value = ''

  try {
    const response = await fetch(`${apiBaseUrl}/api/letters/${letter.id}/extract-events`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ ocr_text: ocrText }),
    })
    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as { message?: string } | null
      throw new Error(body?.message ?? '予定候補を抽出できませんでした')
    }

    const body = (await response.json()) as { events: ExtractedEvent[] }
    mergeExtractedEvents(body.events)
    letterMessage.value = 'AIで予定候補を抽出し、draftとして保存しました'
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : '予定候補の抽出でエラーが発生しました'
  } finally {
    extractingLetterId.value = ''
  }
}

async function deleteLetter(letter: Letter) {
  const title = letter.title || '無題のおたより'
  const confirmed = window.confirm(
    `「${title}」を削除しますか？\n\n画像と紐づく予定候補は削除されます。Googleカレンダーへ登録済みの予定は削除されません。`,
  )
  if (!confirmed) return

  deletingLetterId.value = letter.id
  letterMessage.value = ''
  errorMessage.value = ''

  try {
    const response = await fetch(`${apiBaseUrl}/api/letters/${letter.id}`, {
      method: 'DELETE',
      credentials: 'include',
    })
    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as { message?: string } | null
      throw new Error(body?.message ?? 'おたよりを削除できませんでした')
    }

    if (letter.object_url) URL.revokeObjectURL(letter.object_url)
    delete ocrTextByLetter.value[letter.id]
    letterMessage.value = 'おたより画像と紐づく予定候補を削除しました'
    await Promise.all([loadLetters(), loadExtractedEvents(), loadCalendarEvents()])
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'おたよりの削除でエラーが発生しました'
  } finally {
    deletingLetterId.value = ''
  }
}

async function loadExtractedEvents() {
  const response = await fetch(`${apiBaseUrl}/api/extracted-events`, {
    credentials: 'include',
  })
  if (!response.ok) {
    throw new Error('予定候補一覧を取得できませんでした')
  }

  const body = (await response.json()) as { events: ExtractedEvent[] }
  extractedEvents.value = body.events
  syncEventDrafts(body.events)
  pruneSelectedCandidateIds()
}

function mergeExtractedEvents(events: ExtractedEvent[]) {
  const eventMap = new Map(extractedEvents.value.map((event) => [event.id, event]))
  events.forEach((event) => eventMap.set(event.id, event))
  extractedEvents.value = Array.from(eventMap.values()).sort((a, b) =>
    toDateInput(a.event_date).localeCompare(toDateInput(b.event_date)),
  )
  syncEventDrafts(extractedEvents.value)
  pruneSelectedCandidateIds()
}

function syncEventDrafts(events: ExtractedEvent[]) {
  const drafts = { ...eventDrafts.value }
  events.forEach((event) => {
    drafts[event.id] = toEventDraft(event)
  })
  eventDrafts.value = drafts
}

function toEventDraft(event: ExtractedEvent): ExtractedEventDraft {
  return {
    title: event.title,
    event_date: toDateInput(event.event_date),
    start_time: toTimeInput(event.start_time),
    end_time: toTimeInput(event.end_time),
    is_all_day: event.is_all_day,
    location: event.location ?? '',
    description: event.description ?? '',
  }
}

function toDateInput(value: string) {
  if (/^\d{4}-\d{2}-\d{2}/.test(value)) return value.slice(0, 10)
  return new Date(value).toISOString().slice(0, 10)
}

function toTimeInput(value: string | null) {
  if (!value) return ''
  return value.slice(0, 5)
}

async function saveExtractedEvent(event: ExtractedEvent) {
  if (!canEditExtractedEvent(event)) {
    candidateMessage.value =
      event.status === 'ignored'
        ? '編集するには、先に除外を取り消してください'
        : '登録済み予定はGoogleカレンダーとの不整合を防ぐため編集できません'
    return
  }

  await updateExtractedEvent(event, '予定候補を保存しました')
}

async function restoreIgnoredExtractedEvent(event: ExtractedEvent) {
  await updateExtractedEvent(event, '除外を取り消し、確認済みに戻しました')
}

async function updateExtractedEvent(event: ExtractedEvent, successMessage: string) {
  const draft = eventDrafts.value[event.id]
  if (!draft) return

  savingCandidateId.value = event.id
  candidateMessage.value = ''
  errorMessage.value = ''

  try {
    const response = await fetch(`${apiBaseUrl}/api/extracted-events/${event.id}`, {
      method: 'PATCH',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        ...draft,
        start_time: draft.is_all_day ? '' : draft.start_time,
        end_time: draft.is_all_day ? '' : draft.end_time,
      }),
    })
    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as { message?: string } | null
      throw new Error(body?.message ?? '予定候補を更新できませんでした')
    }

    const body = (await response.json()) as { event: ExtractedEvent }
    replaceExtractedEvent(body.event)
    candidateMessage.value = successMessage
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : '予定候補の更新でエラーが発生しました'
  } finally {
    savingCandidateId.value = ''
  }
}

async function ignoreExtractedEvent(event: ExtractedEvent) {
  savingCandidateId.value = event.id
  candidateMessage.value = ''
  errorMessage.value = ''

  try {
    const response = await fetch(`${apiBaseUrl}/api/extracted-events/${event.id}/ignore`, {
      method: 'POST',
      credentials: 'include',
    })
    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as { message?: string } | null
      throw new Error(body?.message ?? '予定候補を除外できませんでした')
    }

    const body = (await response.json()) as { event: ExtractedEvent }
    replaceExtractedEvent(body.event)
    candidateMessage.value = '予定候補を除外しました'
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : '予定候補の除外でエラーが発生しました'
  } finally {
    savingCandidateId.value = ''
  }
}

async function registerExtractedEvent(event: ExtractedEvent) {
  registeringCandidateId.value = event.id
  candidateMessage.value = ''
  errorMessage.value = ''

  try {
    const response = await fetch(`${apiBaseUrl}/api/extracted-events/${event.id}/register`, {
      method: 'POST',
      credentials: 'include',
    })
    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as { message?: string } | null
      throw new Error(body?.message ?? '予定候補をGoogleカレンダーへ登録できませんでした')
    }

    const body = (await response.json()) as { event: ExtractedEvent }
    replaceExtractedEvent(body.event)
    candidateMessage.value = '予定候補をGoogleカレンダーへ登録しました'
    await loadCalendarEvents()
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : '予定候補のGoogleカレンダー登録でエラーが発生しました'
  } finally {
    registeringCandidateId.value = ''
  }
}

async function bulkConfirmExtractedEvents() {
  await runBulkExtractedEventAction('confirm')
}

async function bulkIgnoreExtractedEvents() {
  await runBulkExtractedEventAction('ignore')
}

async function bulkRegisterExtractedEvents() {
  if (!canBulkRegisterSelectedEvents()) {
    candidateMessage.value = '一括登録するには、選択中の予定候補を先に一括確認してください'
    return
  }
  await runBulkExtractedEventAction('register')
}

async function runBulkExtractedEventAction(action: 'confirm' | 'ignore' | 'register') {
  const ids = [...selectedCandidateIds.value]
  if (ids.length === 0) {
    candidateMessage.value = '一括操作する予定候補を選択してください'
    return
  }

  const actionLabels = {
    confirm: '確認',
    ignore: '除外',
    register: 'Googleカレンダー登録',
  } as const

  bulkCandidateAction.value = action
  candidateMessage.value = ''
  errorMessage.value = ''

  try {
    const response = await fetch(`${apiBaseUrl}/api/extracted-events/bulk-${action}`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ ids }),
    })
    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as { message?: string } | null
      throw new Error(body?.message ?? `予定候補の一括${actionLabels[action]}に失敗しました`)
    }

    const body = (await response.json()) as BulkExtractedEventsResponse
    mergeExtractedEvents(body.events)
    selectedCandidateIds.value = selectedCandidateIds.value.filter((id) =>
      body.results.some((result) => result.id === id && result.status === 'failed'),
    )
    candidateMessage.value = `一括${actionLabels[action]}: 成功 ${body.summary.success}件 / 失敗 ${body.summary.failed}件`

    if (action === 'register') {
      await loadCalendarEvents()
    }
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : `予定候補の一括${actionLabels[action]}でエラーが発生しました`
  } finally {
    bulkCandidateAction.value = ''
  }
}

function canRegisterExtractedEvent(event: ExtractedEvent) {
  return ['confirmed', 'failed', 'deleted'].includes(event.status)
}

function selectedExtractedEvents() {
  const selectedIds = new Set(selectedCandidateIds.value)
  return extractedEvents.value.filter((event) => selectedIds.has(event.id))
}

function canBulkRegisterSelectedEvents() {
  const selectedEvents = selectedExtractedEvents()
  return selectedEvents.length > 0 && selectedEvents.every(canRegisterExtractedEvent)
}

function hasUnregisterableSelectedEvents() {
  return selectedCandidateIds.value.length > 0 && !canBulkRegisterSelectedEvents()
}

function canEditExtractedEvent(event: ExtractedEvent) {
  return event.status !== 'registered' && event.status !== 'ignored'
}

function canSelectExtractedEvent(event: ExtractedEvent) {
  return event.status !== 'registered' && event.status !== 'ignored'
}

function selectableExtractedEvents() {
  return extractedEvents.value.filter(canSelectExtractedEvent)
}

function allSelectableCandidatesSelected() {
  const selectableIds = selectableExtractedEvents().map((event) => event.id)
  return selectableIds.length > 0 && selectableIds.every((id) => selectedCandidateIds.value.includes(id))
}

function toggleAllSelectableCandidates(event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  selectedCandidateIds.value = checked ? selectableExtractedEvents().map((candidate) => candidate.id) : []
}

function pruneSelectedCandidateIds() {
  const selectableIds = new Set(selectableExtractedEvents().map((event) => event.id))
  selectedCandidateIds.value = selectedCandidateIds.value.filter((id) => selectableIds.has(id))
}

function replaceExtractedEvent(nextEvent: ExtractedEvent) {
  extractedEvents.value = extractedEvents.value.map((event) =>
    event.id === nextEvent.id ? nextEvent : event,
  )
  eventDrafts.value = {
    ...eventDrafts.value,
    [nextEvent.id]: toEventDraft(nextEvent),
  }
  pruneSelectedCandidateIds()
}

async function loadCalendarEvents() {
  const response = await fetch(`${apiBaseUrl}/api/calendar-events`, {
    credentials: 'include',
  })
  if (!response.ok) {
    throw new Error('登録済み予定一覧を取得できませんでした')
  }

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
    errorMessage.value = error instanceof Error ? error.message : '予定の再登録でエラーが発生しました'
  } finally {
    retryingCalendarEventId.value = ''
  }
}

function canRetryCalendarEvent(event: CalendarEvent) {
  return (event.status === 'failed' || event.status === 'deleted') && event.source_type === 'manual'
}

function formatCalendarEventTime(event: CalendarEvent) {
  if (event.is_all_day) return '終日'
  const start = event.start_at ? new Date(event.start_at).toLocaleTimeString('ja-JP', timeFormatOptions) : '未設定'
  const end = event.end_at ? new Date(event.end_at).toLocaleTimeString('ja-JP', timeFormatOptions) : '未設定'
  return `${start} - ${end}`
}

const timeFormatOptions = {
  hour: '2-digit',
  minute: '2-digit',
} as const

async function loadLetters() {
  const response = await fetch(`${apiBaseUrl}/api/letters`, {
    credentials: 'include',
  })
  if (!response.ok) {
    throw new Error('おたより一覧を取得できませんでした')
  }

  const body = (await response.json()) as { letters: Letter[] }
  clearLetterObjectUrls()
  letters.value = await Promise.all(
    body.letters.map(async (letter) => ({
      ...letter,
      object_url: await loadLetterImage(letter.image_url),
    })),
  )
}

async function loadLetterImage(imageUrl: string) {
  const response = await fetch(`${apiBaseUrl}${imageUrl}`, {
    credentials: 'include',
  })
  if (!response.ok) {
    throw new Error('おたより画像を取得できませんでした')
  }

  return URL.createObjectURL(await response.blob())
}

function clearLetterObjectUrls() {
  letters.value.forEach((letter) => {
    if (letter.object_url) URL.revokeObjectURL(letter.object_url)
  })
}

async function createManualEvent() {
  savingEvent.value = true
  eventMessage.value = ''
  errorMessage.value = ''

  try {
    const response = await fetch(`${apiBaseUrl}/api/manual-events`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(manualEvent.value),
    })

    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as { message?: string } | null
      throw new Error(body?.message ?? '予定を登録できませんでした')
    }

    eventMessage.value = 'Googleカレンダーに登録しました'
    await loadCalendarEvents()
    manualEvent.value = {
      title: '',
      event_date: '',
      is_all_day: true,
      start_time: '',
      end_time: '',
      location: '',
      description: '',
    }
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '予定登録でエラーが発生しました'
  } finally {
    savingEvent.value = false
  }
}

onMounted(loadMe)

  return {
    activeView,
    allSelectableCandidatesSelected,
    attentionCalendarCount,
    bulkCandidateAction,
    bulkConfirmExtractedEvents,
    bulkIgnoreExtractedEvents,
    bulkRegisterExtractedEvents,
    calendarEvents,
    calendarMessage,
    calendarStatusLabel,
    canBulkRegisterSelectedEvents,
    canEditExtractedEvent,
    canRegisterExtractedEvent,
    canRetryCalendarEvent,
    canSelectExtractedEvent,
    candidateMessage,
    createManualEvent,
    deleteLetter,
    deletingLetterId,
    errorMessage,
    eventDrafts,
    eventMessage,
    extractEvents,
    extractedEvents,
    extractedStatusLabel,
    extractingLetterId,
    formatCalendarEventTime,
    hasUnregisterableSelectedEvents,
    ignoreExtractedEvent,
    letterImage,
    letterMessage,
    letterTitle,
    letters,
    loading,
    loginWithGoogle,
    logout,
    manualEvent,
    ocrTextByLetter,
    onLetterImageChange,
    pendingCandidateCount,
    readyCandidateCount,
    registerExtractedEvent,
    registeringCandidateId,
    restoreIgnoredExtractedEvent,
    retryCalendarEvent,
    retryingCalendarEventId,
    saveExtractedEvent,
    savingCandidateId,
    savingEvent,
    selectableExtractedEvents,
    selectedCandidateIds,
    showManualEventForm,
    switchView,
    toggleAllSelectableCandidates,
    uploadingLetter,
    uploadLetter,
    user,
  }
}

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

type User = {
  id: string
  email: string
  name: string
  created_at: string
}

type Letter = {
  id: string
  title: string
  mime_type: string
  file_size: number
  image_url: string
  created_at: string
  object_url?: string
}

type ExtractedEvent = {
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

type ExtractedEventDraft = {
  title: string
  event_date: string
  start_time: string
  end_time: string
  is_all_day: boolean
  location: string
  description: string
}

type BulkExtractedEventResult = {
  id: string
  status: 'success' | 'failed'
  message?: string
  event?: ExtractedEvent
}

type BulkExtractedEventsResponse = {
  events: ExtractedEvent[]
  results: BulkExtractedEventResult[]
  summary: {
    success: number
    failed: number
  }
}

type CalendarEvent = {
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
const activeView = ref<'home' | 'letters' | 'candidates' | 'calendar'>('home')
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

function switchView(view: 'home' | 'letters' | 'candidates' | 'calendar') {
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
</script>

<template>
  <main class="page-shell">
    <div v-if="loading" class="loading-screen">
      <span class="loading-dot"></span>
      <p>カレンダーを準備しています</p>
    </div>

    <section v-else-if="!user" class="login-shell">
      <div class="login-mark">OC</div>
      <p class="eyebrow">Otayori Calendar</p>
      <h1>家族の予定に、<br />見落とさない安心を。</h1>
      <p class="lead">園から届くおたよりを読み取り、大切な予定をGoogleカレンダーへまとめます。</p>
      <button class="primary-button login-button" type="button" @click="loginWithGoogle">Googleでログイン</button>
      <p class="hint">Googleカレンダーへの登録に必要な権限だけを使用します。</p>
    </section>

    <div v-else class="app-layout">
      <header class="app-header">
        <a class="brand" href="#top" aria-label="トップへ">
          <span class="brand-mark">OC</span>
          <span><strong>おたよりカレンダー</strong><small>家族の予定を、ひとつに</small></span>
        </a>
        <div class="account">
          <span class="account-avatar">{{ (user.name || user.email).slice(0, 1) }}</span>
          <span class="account-copy"><strong>{{ user.name || 'ログイン中' }}</strong><small>{{ user.email }}</small></span>
          <button class="text-button" type="button" @click="logout">ログアウト</button>
        </div>
      </header>

      <div id="top" class="content-shell">
        <section v-if="activeView === 'home'" class="welcome-panel">
          <div>
            <p class="eyebrow">Family dashboard</p>
            <h1>{{ user.name || 'こんにちは' }}さん、<br />今日も予定を整えましょう。</h1>
            <p class="lead">おたよりを追加して、確認が必要な予定だけをすばやく片付けられます。</p>
          </div>
          <button class="primary-button" type="button" @click="switchView('letters')">おたよりを追加</button>
        </section>

        <section v-if="activeView === 'home'" class="summary-grid" aria-label="現在の状況">
          <button class="summary-card mint" type="button" @click="switchView('candidates')">
            <span>確認待ち</span><strong>{{ pendingCandidateCount }}</strong><small>予定候補</small>
          </button>
          <button class="summary-card lime" type="button" @click="switchView('candidates')">
            <span>登録できる予定</span><strong>{{ readyCandidateCount }}</strong><small>確認済み・再登録</small>
          </button>
          <button class="summary-card sand" type="button" @click="switchView('calendar')">
            <span>要対応</span><strong>{{ attentionCalendarCount }}</strong><small>失敗・削除済み</small>
          </button>
          <button class="summary-card white" type="button" @click="switchView('letters')">
            <span>おたより</span><strong>{{ letters.length }}</strong><small>アップロード済み</small>
          </button>
        </section>

        <section v-if="activeView === 'home'" class="next-actions">
          <div class="section-heading">
            <div><p class="section-kicker">Next actions</p><h2>次にやること</h2></div>
            <p>いま対応が必要な操作だけを表示しています。</p>
          </div>
          <button v-if="pendingCandidateCount > 0" class="next-action-card" type="button" @click="switchView('candidates')">
            <span class="next-action-number">{{ pendingCandidateCount }}</span>
            <span><strong>未確認の予定候補があります</strong><small>内容を確認してカレンダーへ登録しましょう</small></span>
            <span class="next-action-arrow">→</span>
          </button>
          <button v-if="attentionCalendarCount > 0" class="next-action-card attention" type="button" @click="switchView('calendar')">
            <span class="next-action-number">{{ attentionCalendarCount }}</span>
            <span><strong>カレンダー予定に確認が必要です</strong><small>失敗・削除済みの予定を確認しましょう</small></span>
            <span class="next-action-arrow">→</span>
          </button>
          <button class="next-action-card" type="button" @click="switchView('letters')">
            <span class="next-action-number">＋</span>
            <span><strong>新しいおたよりを追加</strong><small>園から届いた画像を予定に変換します</small></span>
            <span class="next-action-arrow">→</span>
          </button>
        </section>

        <p v-if="errorMessage" class="notice error-notice">{{ errorMessage }}</p>
        <p v-if="letterMessage" class="notice success-notice">{{ letterMessage }}</p>

        <section v-if="activeView === 'letters'" id="upload" class="workspace-section view-section">
          <div class="section-heading">
            <div><p class="section-kicker">Step 1</p><h2>おたよりを追加</h2></div>
            <p>写真を選ぶだけで、AIが予定候補を探します。</p>
          </div>
          <form class="surface form-grid upload-form" @submit.prevent="uploadLetter">
            <label>おたよりの名前<input v-model="letterTitle" type="text" placeholder="例：6月のえんだより" /></label>
            <label class="file-field">画像を選択<input accept="image/jpeg,image/png,image/webp" required type="file" @change="onLetterImageChange" /><span>{{ letterImage?.name || 'JPEG・PNG・WebP' }}</span></label>
            <button class="primary-button" :disabled="uploadingLetter" type="submit">{{ uploadingLetter ? 'アップロード中...' : 'アップロードする' }}</button>
          </form>
        </section>

        <section v-if="activeView === 'letters'" id="letters" class="workspace-section">
          <div class="section-heading">
            <div><p class="section-kicker">Letters</p><h2>アップロード済み</h2></div>
            <p>{{ letters.length }}件のおたよりがあります。</p>
          </div>
          <div v-if="letters.length === 0" class="empty-state">まだおたよりはありません。最初の1枚を追加しましょう。</div>
          <div class="letter-grid">
            <article v-for="letter in letters" :key="letter.id" class="surface letter-card">
              <img v-if="letter.object_url" :src="letter.object_url" :alt="letter.title || 'おたより画像'" />
              <div class="letter-summary">
                <p class="section-kicker">Uploaded</p><h3>{{ letter.title || '無題のおたより' }}</h3>
                <p>{{ new Date(letter.created_at).toLocaleString('ja-JP') }}</p>
              </div>
              <div class="ocr-panel">
                <label>補足テキスト（任意）<textarea v-model="ocrTextByLetter[letter.id]" rows="3" placeholder="画像が読みづらい場合だけ入力してください"></textarea></label>
                <p class="helper-text">空欄なら画像から予定を読み取ります。</p>
                <div class="button-row">
                  <button class="primary-button" :disabled="extractingLetterId === letter.id" type="button" @click="extractEvents(letter)">{{ extractingLetterId === letter.id ? '予定を探しています...' : '予定候補を見つける' }}</button>
                  <button class="danger-button" :disabled="deletingLetterId === letter.id || extractingLetterId === letter.id" type="button" @click="deleteLetter(letter)">{{ deletingLetterId === letter.id ? '削除中...' : '削除' }}</button>
                </div>
              </div>
            </article>
          </div>
        </section>

        <section v-if="activeView === 'candidates'" id="candidates" class="workspace-section view-section">
          <div class="section-heading">
            <div><p class="section-kicker">Step 2</p><h2>予定候補を確認</h2></div>
            <p>内容を確認してから、Googleカレンダーへ登録します。</p>
          </div>
          <div v-if="extractedEvents.length === 0" class="empty-state">おたよりから予定候補を見つけると、ここに表示されます。</div>
        <div v-if="extractedEvents.length > 0" class="bulk-toolbar">
          <label class="checkbox-label bulk-select-label">
            <input
              :checked="allSelectableCandidatesSelected()"
              :disabled="selectableExtractedEvents().length === 0 || bulkCandidateAction !== ''"
              type="checkbox"
              @change="toggleAllSelectableCandidates"
            />
            まとめて選択
          </label>
          <p><strong>{{ selectedCandidateIds.length }}件</strong> 選択中</p>
          <div class="bulk-actions">
            <button class="secondary-button" :disabled="selectedCandidateIds.length === 0 || bulkCandidateAction !== ''" type="button" @click="bulkConfirmExtractedEvents">{{ bulkCandidateAction === 'confirm' ? '確認中...' : '確認済みにする' }}</button>
            <button class="secondary-button" :disabled="selectedCandidateIds.length === 0 || bulkCandidateAction !== ''" type="button" @click="bulkIgnoreExtractedEvents">{{ bulkCandidateAction === 'ignore' ? '除外中...' : '除外する' }}</button>
            <button class="primary-button" :disabled="!canBulkRegisterSelectedEvents() || bulkCandidateAction !== ''" type="button" @click="bulkRegisterExtractedEvents">{{ bulkCandidateAction === 'register' ? '登録中...' : 'カレンダーへ登録' }}</button>
          </div>
          <p v-if="hasUnregisterableSelectedEvents()" class="bulk-guidance">
            一括登録するには、選択中の予定候補を先に一括確認してください。
          </p>
        </div>
        <article
          v-for="event in extractedEvents"
          :key="event.id"
          class="surface candidate-card"
          :class="[`status-${event.status}`, { ignored: event.status === 'ignored' }]"
        >
          <div class="candidate-heading">
            <div class="candidate-title-row">
              <label class="candidate-select">
                <input
                  v-model="selectedCandidateIds"
                  :disabled="!canSelectExtractedEvent(event) || bulkCandidateAction !== ''"
                  :value="event.id"
                  type="checkbox"
                />
                <span>選択</span>
              </label>
              <p class="status-chip">{{ extractedStatusLabel(event.status) }}</p>
              <h3>{{ event.title }}</h3>
            </div>
            <p v-if="event.confidence < 0.7" class="warning-chip">読み取り要確認</p>
          </div>

          <form v-if="eventDrafts[event.id]" class="candidate-form" @submit.prevent="saveExtractedEvent(event)">
            <p v-if="event.status === 'registered'" class="candidate-lock-message">
              Googleカレンダー登録済みのため、この画面では編集できません。
            </p>
            <p v-if="event.status === 'ignored'" class="candidate-lock-message">
              除外済みです。編集を再開する場合は、先に除外を取り消してください。
            </p>
            <fieldset class="candidate-fields" :disabled="!canEditExtractedEvent(event)">
              <label>
                予定名
                <input v-model="eventDrafts[event.id].title" required type="text" />
              </label>
              <label>
                日付
                <input v-model="eventDrafts[event.id].event_date" required type="date" />
              </label>
              <label class="checkbox-label">
                <input v-model="eventDrafts[event.id].is_all_day" type="checkbox" />
                終日予定
              </label>
              <div v-if="!eventDrafts[event.id].is_all_day" class="time-grid">
                <label>
                  開始
                  <input v-model="eventDrafts[event.id].start_time" required type="time" />
                </label>
                <label>
                  終了
                  <input v-model="eventDrafts[event.id].end_time" required type="time" />
                </label>
              </div>
              <label>
                場所
                <input v-model="eventDrafts[event.id].location" type="text" placeholder="保育園" />
              </label>
              <label>
                説明
                <textarea v-model="eventDrafts[event.id].description" rows="3"></textarea>
              </label>
            </fieldset>
            <div class="candidate-actions">
              <button
                v-if="event.status !== 'ignored'"
                class="primary-button"
                :disabled="savingCandidateId === event.id || !canEditExtractedEvent(event)"
                type="submit"
              >
                {{ savingCandidateId === event.id ? '保存中...' : '保存' }}
              </button>
              <button
                v-if="event.status === 'ignored'"
                class="secondary-button"
                :disabled="savingCandidateId === event.id"
                type="button"
                @click="restoreIgnoredExtractedEvent(event)"
              >
                {{ savingCandidateId === event.id ? '取消中...' : '除外を取り消す' }}
              </button>
              <button
                v-if="canRegisterExtractedEvent(event)"
                class="primary-button"
                :disabled="registeringCandidateId === event.id"
                type="button"
                @click="registerExtractedEvent(event)"
              >
                {{ registeringCandidateId === event.id ? '登録中...' : 'Googleカレンダーに登録' }}
              </button>
              <button
                class="secondary-button"
                :disabled="
                  savingCandidateId === event.id ||
                  registeringCandidateId === event.id ||
                  event.status === 'ignored' ||
                  event.status === 'registered'
                "
                type="button"
                @click="ignoreExtractedEvent(event)"
              >
                除外する
              </button>
            </div>
          </form>

          <div class="source-box">
            <p>読み取り確度 {{ Math.round(event.confidence * 100) }}%</p>
            <p>{{ event.source_text || '元テキストはありません。' }}</p>
          </div>
        </article>
        <p v-if="candidateMessage" class="notice success-notice">{{ candidateMessage }}</p>
      </section>

        <section v-if="activeView === 'calendar'" id="calendar" class="workspace-section view-section">
          <div class="section-heading">
            <div><p class="section-kicker">Calendar</p><h2>カレンダー登録状況</h2></div>
            <p>登録後の予定と、対応が必要な予定を確認できます。</p>
          </div>
          <div v-if="calendarEvents.length === 0" class="empty-state">Googleカレンダーへ登録した予定はまだありません。</div>
          <div class="calendar-grid">
        <article
          v-for="event in calendarEvents"
          :key="`${event.source_type}-${event.id}`"
          class="surface registered-event-card"
          :class="{ failed: event.status === 'failed', deleted: event.status === 'deleted' }"
        >
          <div class="candidate-heading">
            <div>
              <p class="status-chip">{{ calendarStatusLabel(event.status) }}</p>
              <h3>{{ event.title }}</h3>
            </div>
            <p class="source-type-chip">{{ event.source_type === 'manual' ? '手入力' : 'おたより候補' }}</p>
          </div>
          <p>{{ new Date(event.event_date).toLocaleDateString('ja-JP') }} / {{ formatCalendarEventTime(event) }}</p>
          <p v-if="event.location">{{ event.location }}</p>
          <p v-if="event.description">{{ event.description }}</p>
          <details v-if="event.google_calendar_event_id" class="technical-details">
            <summary>登録情報を表示</summary>
            <p class="source-text">Google Event ID: {{ event.google_calendar_event_id }}</p>
          </details>
          <p v-if="event.status === 'deleted'" class="error">
            Googleカレンダー上で削除されています。
          </p>
          <p v-else-if="!event.google_calendar_event_id" class="error">Googleカレンダー登録に失敗しています。</p>
          <button
            v-if="canRetryCalendarEvent(event)"
            class="secondary-button"
            :disabled="retryingCalendarEventId === event.id"
            type="button"
            @click="retryCalendarEvent(event)"
          >
            {{ retryingCalendarEventId === event.id ? '再実行中...' : '再登録する' }}
          </button>
        </article>
          </div>
          <p v-if="calendarMessage" class="notice success-notice">{{ calendarMessage }}</p>
      </section>

        <section v-if="activeView === 'calendar'" id="manual" class="workspace-section manual-section">
          <button class="manual-toggle" type="button" @click="showManualEventForm = !showManualEventForm">
            <span><strong>予定を手入力</strong><small>おたよりにない予定を追加</small></span>
            <span>{{ showManualEventForm ? '閉じる' : '＋ 追加する' }}</span>
          </button>
          <form v-if="showManualEventForm" class="surface form-grid" @submit.prevent="createManualEvent">
            <label>予定名<input v-model="manualEvent.title" required type="text" placeholder="例：身体測定" /></label>
            <label>日付<input v-model="manualEvent.event_date" required type="date" /></label>
            <label class="checkbox-label"><input v-model="manualEvent.is_all_day" type="checkbox" />終日予定として登録する</label>
            <div v-if="!manualEvent.is_all_day" class="time-grid">
              <label>開始<input v-model="manualEvent.start_time" required type="time" /></label>
              <label>終了<input v-model="manualEvent.end_time" required type="time" /></label>
            </div>
            <label>場所<input v-model="manualEvent.location" type="text" placeholder="例：保育園" /></label>
            <label>メモ<textarea v-model="manualEvent.description" rows="3" placeholder="持ち物や注意事項"></textarea></label>
            <button class="primary-button" :disabled="savingEvent" type="submit">{{ savingEvent ? '登録中...' : 'Googleカレンダーへ登録' }}</button>
            <p v-if="eventMessage" class="notice success-notice">{{ eventMessage }}</p>
          </form>
        </section>
      </div>

      <nav class="bottom-nav" aria-label="画面切り替え">
        <button :class="{ active: activeView === 'home' }" type="button" @click="switchView('home')"><span>⌂</span><small>ホーム</small></button>
        <button :class="{ active: activeView === 'letters' }" type="button" @click="switchView('letters')"><span>▧</span><small>おたより</small></button>
        <button :class="{ active: activeView === 'candidates' }" type="button" @click="switchView('candidates')"><span>✓</span><small>予定候補</small><b v-if="pendingCandidateCount">{{ pendingCandidateCount }}</b></button>
        <button :class="{ active: activeView === 'calendar' }" type="button" @click="switchView('calendar')"><span>□</span><small>カレンダー</small><b v-if="attentionCalendarCount">{{ attentionCalendarCount }}</b></button>
      </nav>
    </div>
  </main>
</template>

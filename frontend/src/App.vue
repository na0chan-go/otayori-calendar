<script setup lang="ts">
import { onMounted, ref } from 'vue'

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
const manualEvent = ref({
  title: '',
  event_date: '',
  is_all_day: true,
  start_time: '',
  end_time: '',
  location: '',
  description: '',
})

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
    candidateMessage.value = '予定候補を保存しました'
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

function canSelectExtractedEvent(event: ExtractedEvent) {
  return event.status !== 'registered'
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
    <section class="hero-card">
      <p class="eyebrow">Otayori Calendar</p>
      <h1>おたよりの予定を、Googleカレンダーへ。</h1>
      <p class="lead">
        まずはGoogleアカウントでログインして、カレンダー連携の準備をします。
      </p>

      <div v-if="loading" class="panel">ログイン状態を確認しています...</div>

      <div v-else-if="user" class="panel signed-in">
        <p class="label">ログイン中</p>
        <h2>{{ user.name || user.email }}</h2>
        <p>{{ user.email }}</p>
        <button class="ghost-button" type="button" @click="logout">ログアウト</button>
      </div>

      <form v-if="user" class="event-form" @submit.prevent="uploadLetter">
        <p class="label">おたより画像</p>
        <label>
          タイトル
          <input v-model="letterTitle" type="text" placeholder="6月のおたより" />
        </label>
        <label>
          画像
          <input accept="image/jpeg,image/png,image/webp" required type="file" @change="onLetterImageChange" />
        </label>
        <button class="google-button" :disabled="uploadingLetter" type="submit">
          {{ uploadingLetter ? 'アップロード中...' : '画像をアップロード' }}
        </button>
        <p v-if="letterMessage" class="success">{{ letterMessage }}</p>
      </form>

      <section v-if="user" class="letters-section">
        <p class="label">アップロード済み</p>
        <div v-if="letters.length === 0" class="empty-state">まだおたより画像はありません。</div>
        <article v-for="letter in letters" :key="letter.id" class="letter-card">
          <img v-if="letter.object_url" :src="letter.object_url" :alt="letter.title || 'おたより画像'" />
          <div>
            <h3>{{ letter.title || '無題のおたより' }}</h3>
            <p>{{ new Date(letter.created_at).toLocaleString('ja-JP') }}</p>
          </div>
          <div class="ocr-panel">
            <label>
              OCRテキスト（任意）
              <textarea
                v-model="ocrTextByLetter[letter.id]"
                rows="4"
                placeholder="空欄の場合はアップロード画像からAI抽出します。"
              ></textarea>
            </label>
            <p class="source-text">OCRテキストを入力すると、画像ではなくテキストをAIへ送信します。</p>
            <button
              class="ghost-button"
              :disabled="extractingLetterId === letter.id"
              type="button"
              @click="extractEvents(letter)"
            >
              {{ extractingLetterId === letter.id ? 'AI抽出中...' : 'AIで予定候補を抽出' }}
            </button>
          </div>
        </article>
      </section>

      <section v-if="user && extractedEvents.length > 0" class="letters-section">
        <p class="label">予定候補の確認</p>
        <div class="bulk-toolbar">
          <label class="checkbox-label bulk-select-label">
            <input
              :checked="allSelectableCandidatesSelected()"
              :disabled="selectableExtractedEvents().length === 0 || bulkCandidateAction !== ''"
              type="checkbox"
              @change="toggleAllSelectableCandidates"
            />
            まとめて選択
          </label>
          <p>{{ selectedCandidateIds.length }}件選択中</p>
          <div class="bulk-actions">
            <button
              class="ghost-button"
              :disabled="selectedCandidateIds.length === 0 || bulkCandidateAction !== ''"
              type="button"
              @click="bulkConfirmExtractedEvents"
            >
              {{ bulkCandidateAction === 'confirm' ? '確認中...' : '一括確認' }}
            </button>
            <button
              class="ghost-button"
              :disabled="selectedCandidateIds.length === 0 || bulkCandidateAction !== ''"
              type="button"
              @click="bulkIgnoreExtractedEvents"
            >
              {{ bulkCandidateAction === 'ignore' ? '除外中...' : '一括除外' }}
            </button>
            <button
              class="google-button"
              :disabled="selectedCandidateIds.length === 0 || bulkCandidateAction !== ''"
              type="button"
              @click="bulkRegisterExtractedEvents"
            >
              {{ bulkCandidateAction === 'register' ? '登録中...' : '一括登録' }}
            </button>
          </div>
        </div>
        <article
          v-for="event in extractedEvents"
          :key="event.id"
          class="candidate-card"
          :class="{ ignored: event.status === 'ignored' }"
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
              <p class="status-chip">{{ event.status }}</p>
              <h3>{{ event.title }}</h3>
            </div>
            <p v-if="event.confidence < 0.7" class="warning-chip">要確認</p>
          </div>

          <form v-if="eventDrafts[event.id]" class="candidate-form" @submit.prevent="saveExtractedEvent(event)">
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
            <div class="candidate-actions">
              <button class="google-button" :disabled="savingCandidateId === event.id" type="submit">
                {{ savingCandidateId === event.id ? '保存中...' : '保存' }}
              </button>
              <button
                v-if="canRegisterExtractedEvent(event)"
                class="google-button"
                :disabled="registeringCandidateId === event.id"
                type="button"
                @click="registerExtractedEvent(event)"
              >
                {{ registeringCandidateId === event.id ? '登録中...' : 'Googleカレンダーに登録' }}
              </button>
              <button
                class="ghost-button"
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
            <p>confidence: {{ event.confidence.toFixed(2) }}</p>
            <p>{{ event.source_text || '元テキストはありません。' }}</p>
          </div>
        </article>
        <p v-if="candidateMessage" class="success">{{ candidateMessage }}</p>
      </section>

      <section v-if="user" class="letters-section">
        <p class="label">登録済み・失敗予定</p>
        <div v-if="calendarEvents.length === 0" class="empty-state">
          まだGoogleカレンダー登録済みの予定はありません。
        </div>
        <article
          v-for="event in calendarEvents"
          :key="`${event.source_type}-${event.id}`"
          class="registered-event-card"
          :class="{ failed: event.status === 'failed', deleted: event.status === 'deleted' }"
        >
          <div class="candidate-heading">
            <div>
              <p class="status-chip">{{ event.status }}</p>
              <h3>{{ event.title }}</h3>
            </div>
            <p class="source-type-chip">{{ event.source_type === 'manual' ? '手入力' : 'おたより候補' }}</p>
          </div>
          <p>{{ new Date(event.event_date).toLocaleDateString('ja-JP') }} / {{ formatCalendarEventTime(event) }}</p>
          <p v-if="event.location">{{ event.location }}</p>
          <p v-if="event.description">{{ event.description }}</p>
          <p v-if="event.google_calendar_event_id" class="source-text">
            Google Event ID: {{ event.google_calendar_event_id }}
          </p>
          <p v-if="event.status === 'deleted'" class="error">
            Googleカレンダー上で削除されています。
          </p>
          <p v-else-if="!event.google_calendar_event_id" class="error">Googleカレンダー登録に失敗しています。</p>
          <button
            v-if="canRetryCalendarEvent(event)"
            class="ghost-button"
            :disabled="retryingCalendarEventId === event.id"
            type="button"
            @click="retryCalendarEvent(event)"
          >
            {{ retryingCalendarEventId === event.id ? '再実行中...' : '再登録する' }}
          </button>
        </article>
        <p v-if="calendarMessage" class="success">{{ calendarMessage }}</p>
      </section>

      <form v-if="user" class="event-form" @submit.prevent="createManualEvent">
        <p class="label">手入力予定</p>
        <label>
          予定名
          <input v-model="manualEvent.title" required type="text" placeholder="身体測定" />
        </label>
        <label>
          日付
          <input v-model="manualEvent.event_date" required type="date" />
        </label>
        <label class="checkbox-label">
          <input v-model="manualEvent.is_all_day" type="checkbox" />
          終日予定として登録する
        </label>
        <div v-if="!manualEvent.is_all_day" class="time-grid">
          <label>
            開始
            <input v-model="manualEvent.start_time" required type="time" />
          </label>
          <label>
            終了
            <input v-model="manualEvent.end_time" required type="time" />
          </label>
        </div>
        <label>
          場所
          <input v-model="manualEvent.location" type="text" placeholder="保育園" />
        </label>
        <label>
          メモ
          <textarea v-model="manualEvent.description" rows="3" placeholder="持ち物など"></textarea>
        </label>
        <button class="google-button" :disabled="savingEvent" type="submit">
          {{ savingEvent ? '登録中...' : 'Googleカレンダーに登録' }}
        </button>
        <p v-if="eventMessage" class="success">{{ eventMessage }}</p>
      </form>

      <div v-else class="actions">
        <button class="google-button" type="button" @click="loginWithGoogle">
          Googleでログイン
        </button>
        <p class="hint">Google Calendar API の許可画面へ移動します。</p>
      </div>

      <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
    </section>
  </main>
</template>

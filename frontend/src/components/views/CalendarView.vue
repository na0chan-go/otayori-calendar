<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useOtayoriCalendarContext } from '../../composables/otayoriCalendarContext'
import type { CalendarEvent } from '../../types'

const {
  calendarEvents,
  calendarMessage,
  calendarStatusLabel,
  canRetryCalendarEvent,
  createManualEvent,
  extractedEvents,
  eventMessage,
  focusedCalendarEventKey,
  formatCalendarEventTime,
  manualEvent,
  retryCalendarEvent,
  retryingCalendarEventId,
  resetManualEvent,
  savingEvent,
  showManualEventForm,
  switchView,
} = useOtayoriCalendarContext()

type CalendarFilter = 'upcoming' | 'past' | 'all'
const calendarFilter = ref<CalendarFilter>('upcoming')
const manualFormSubmitted = ref(false)

const filteredCalendarEvents = computed(() => {
  const today = localToday()
  return calendarEvents.value.filter((event) => {
    const date = event.event_date.slice(0, 10)
    if (calendarFilter.value === 'upcoming') return date >= today || event.status !== 'registered'
    if (calendarFilter.value === 'past') return date < today
    return true
  })
})

const calendarGroups = computed(() => {
  const groups = new Map<string, CalendarEvent[]>()
  filteredCalendarEvents.value.forEach((event) => {
    const date = event.event_date.slice(0, 10)
    groups.set(date, [...(groups.get(date) ?? []), event])
  })
  return Array.from(groups, ([date, events]) => ({ date, events }))
    .sort((a, b) => calendarFilter.value === 'past' ? b.date.localeCompare(a.date) : a.date.localeCompare(b.date))
})

watch(focusedCalendarEventKey, async (key) => {
  if (!key) return
  calendarFilter.value = 'all'
  await nextTick()
  document.querySelector(`[data-calendar-key="${key}"]`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}, { immediate: true })

function goToNextCalendarAction() {
  switchView(extractedEvents.value.length > 0 ? 'candidates' : 'letters')
}

const manualEventErrors = computed(() => {
  const errors: Record<string, string> = {}
  if (!manualEvent.value.title.trim()) errors.title = '予定名を入力してください'
  if (!manualEvent.value.event_date) errors.event_date = '日付を選択してください'
  if (!manualEvent.value.is_all_day) {
    if (!manualEvent.value.start_time) errors.start_time = '開始時刻を選択してください'
    if (!manualEvent.value.end_time) errors.end_time = '終了時刻を選択してください'
    if (manualEvent.value.start_time && manualEvent.value.end_time && manualEvent.value.end_time <= manualEvent.value.start_time) {
      errors.end_time = '終了時刻は開始時刻より後にしてください'
    }
  }
  return errors
})

function showManualError(field: string) {
  return manualFormSubmitted.value ? manualEventErrors.value[field] : ''
}

async function submitManualEvent() {
  manualFormSubmitted.value = true
  if (Object.keys(manualEventErrors.value).length > 0) {
    await nextTick()
    const invalidInput = document.querySelector<HTMLInputElement>('.manual-section [aria-invalid="true"]')
    invalidInput?.focus()
    invalidInput?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    return
  }
  await createManualEvent()
  manualFormSubmitted.value = false
}

function toggleManualEventForm() {
  if (showManualEventForm.value && hasManualInput() && !window.confirm('入力途中の予定があります。破棄して閉じますか？')) return
  if (showManualEventForm.value) {
    resetManualEvent()
    manualFormSubmitted.value = false
  }
  showManualEventForm.value = !showManualEventForm.value
}

function hasManualInput() {
  return Object.entries(manualEvent.value).some(([key, value]) => key !== 'is_all_day' && value !== '')
}

function localToday() {
  const now = new Date()
  return new Date(now.getTime() - now.getTimezoneOffset() * 60_000).toISOString().slice(0, 10)
}

function calendarDateHeading(date: string) {
  const today = localToday()
  if (date === today) return `今日 · ${formatDate(date)}`
  return formatDate(date)
}

function formatDate(date: string) {
  return new Date(`${date}T00:00:00`).toLocaleDateString('ja-JP', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'short' })
}
</script>

<template>
  <section id="calendar" class="workspace-section view-section">
    <div class="section-heading">
      <div><p class="section-kicker">Calendar</p><h2>カレンダー登録状況</h2></div>
      <p>登録後の予定と、対応が必要な予定を確認できます。</p>
    </div>
    <div v-if="calendarEvents.length === 0" class="empty-state">
      <h3>登録済みの予定はまだありません</h3>
      <p>予定候補の内容を確認して登録するか、予定を手入力できます。</p>
      <button class="primary-button" type="button" @click="goToNextCalendarAction">
        {{ extractedEvents.length > 0 ? '予定候補を確認する' : 'おたよりを追加する' }}
      </button>
    </div>
    <div v-if="calendarEvents.length > 0" class="candidate-filters surface calendar-filters">
      <div class="filter-options">
        <button :class="{ active: calendarFilter === 'upcoming' }" type="button" @click="calendarFilter = 'upcoming'">今後・要対応</button>
        <button :class="{ active: calendarFilter === 'past' }" type="button" @click="calendarFilter = 'past'">過去</button>
        <button :class="{ active: calendarFilter === 'all' }" type="button" @click="calendarFilter = 'all'">すべて</button>
      </div>
      <p class="filter-result"><strong>{{ filteredCalendarEvents.length }}件</strong> を表示中 / 全{{ calendarEvents.length }}件</p>
    </div>
    <div class="calendar-date-groups">
      <section v-for="group in calendarGroups" :key="group.date" class="calendar-date-group">
        <h3>{{ calendarDateHeading(group.date) }}</h3>
        <div class="calendar-grid">
          <article
            v-for="event in group.events as CalendarEvent[]"
            :key="`${event.source_type}-${event.id}`"
            class="surface registered-event-card"
            :class="{ failed: event.status === 'failed', deleted: event.status === 'deleted', focused: focusedCalendarEventKey === `${event.source_type}-${event.id}` }"
            :data-calendar-key="`${event.source_type}-${event.id}`"
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
        <p v-if="event.status === 'deleted'" class="error">Googleカレンダー上で削除されています。</p>
        <p v-else-if="!event.google_calendar_event_id" class="error">Googleカレンダー登録に失敗しています。</p>
        <button v-if="canRetryCalendarEvent(event)" class="secondary-button" :disabled="retryingCalendarEventId === event.id" type="button" @click="retryCalendarEvent(event)">
          {{ retryingCalendarEventId === event.id ? '再実行中...' : '再登録する' }}
        </button>
          </article>
        </div>
      </section>
    </div>
    <p v-if="calendarMessage" class="notice success-notice">{{ calendarMessage }}</p>
  </section>

  <section id="manual" class="workspace-section manual-section">
    <button class="manual-toggle" type="button" @click="toggleManualEventForm">
      <span><strong>予定を手入力</strong><small>おたよりにない予定を追加</small></span>
      <span>{{ showManualEventForm ? '閉じる' : '＋ 追加する' }}</span>
    </button>
    <form v-if="showManualEventForm" class="surface form-grid" novalidate @submit.prevent="submitManualEvent">
      <label>予定名<input v-model="manualEvent.title" :aria-invalid="!!showManualError('title')" required type="text" placeholder="例：身体測定" /><small v-if="showManualError('title')" class="field-error">{{ showManualError('title') }}</small></label>
      <label>日付<input v-model="manualEvent.event_date" :aria-invalid="!!showManualError('event_date')" required type="date" /><small v-if="showManualError('event_date')" class="field-error">{{ showManualError('event_date') }}</small></label>
      <label class="checkbox-label"><input v-model="manualEvent.is_all_day" type="checkbox" />終日予定として登録する</label>
      <div v-if="!manualEvent.is_all_day" class="time-grid">
        <label>開始<input v-model="manualEvent.start_time" :aria-invalid="!!showManualError('start_time')" required type="time" /><small v-if="showManualError('start_time')" class="field-error">{{ showManualError('start_time') }}</small></label>
        <label>終了<input v-model="manualEvent.end_time" :aria-invalid="!!showManualError('end_time')" required type="time" /><small v-if="showManualError('end_time')" class="field-error">{{ showManualError('end_time') }}</small></label>
      </div>
      <label>場所<input v-model="manualEvent.location" type="text" placeholder="例：保育園" /></label>
      <label>メモ<textarea v-model="manualEvent.description" rows="3" placeholder="持ち物や注意事項"></textarea></label>
      <button class="primary-button" :disabled="savingEvent" type="submit">{{ savingEvent ? '登録中...' : 'Googleカレンダーへ登録' }}</button>
      <p v-if="eventMessage" class="notice success-notice">{{ eventMessage }}</p>
    </form>
  </section>
</template>

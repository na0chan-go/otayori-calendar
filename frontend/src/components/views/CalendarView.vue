<script setup lang="ts">
import { useOtayoriCalendarContext } from '../../composables/otayoriCalendarContext'

const {
  calendarEvents,
  calendarMessage,
  calendarStatusLabel,
  canRetryCalendarEvent,
  createManualEvent,
  eventMessage,
  formatCalendarEventTime,
  manualEvent,
  retryCalendarEvent,
  retryingCalendarEventId,
  savingEvent,
  showManualEventForm,
} = useOtayoriCalendarContext()
</script>

<template>
  <section id="calendar" class="workspace-section view-section">
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
        <p v-if="event.status === 'deleted'" class="error">Googleカレンダー上で削除されています。</p>
        <p v-else-if="!event.google_calendar_event_id" class="error">Googleカレンダー登録に失敗しています。</p>
        <button v-if="canRetryCalendarEvent(event)" class="secondary-button" :disabled="retryingCalendarEventId === event.id" type="button" @click="retryCalendarEvent(event)">
          {{ retryingCalendarEventId === event.id ? '再実行中...' : '再登録する' }}
        </button>
      </article>
    </div>
    <p v-if="calendarMessage" class="notice success-notice">{{ calendarMessage }}</p>
  </section>

  <section id="manual" class="workspace-section manual-section">
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
</template>

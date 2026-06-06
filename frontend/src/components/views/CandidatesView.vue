<script setup lang="ts">
import { useOtayoriCalendarContext } from '../../composables/otayoriCalendarContext'

const {
  allSelectableCandidatesSelected,
  bulkCandidateAction,
  bulkConfirmExtractedEvents,
  bulkIgnoreExtractedEvents,
  bulkRegisterExtractedEvents,
  canBulkRegisterSelectedEvents,
  canEditExtractedEvent,
  canRegisterExtractedEvent,
  canSelectExtractedEvent,
  candidateMessage,
  eventDrafts,
  extractedEvents,
  extractedStatusLabel,
  hasUnregisterableSelectedEvents,
  ignoreExtractedEvent,
  registerExtractedEvent,
  registeringCandidateId,
  restoreIgnoredExtractedEvent,
  saveExtractedEvent,
  savingCandidateId,
  selectableExtractedEvents,
  selectedCandidateIds,
  toggleAllSelectableCandidates,
} = useOtayoriCalendarContext()
</script>

<template>
  <section id="candidates" class="workspace-section view-section">
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
          <label>予定名<input v-model="eventDrafts[event.id].title" required type="text" /></label>
          <label>日付<input v-model="eventDrafts[event.id].event_date" required type="date" /></label>
          <label class="checkbox-label"><input v-model="eventDrafts[event.id].is_all_day" type="checkbox" />終日予定</label>
          <div v-if="!eventDrafts[event.id].is_all_day" class="time-grid">
            <label>開始<input v-model="eventDrafts[event.id].start_time" required type="time" /></label>
            <label>終了<input v-model="eventDrafts[event.id].end_time" required type="time" /></label>
          </div>
          <label>場所<input v-model="eventDrafts[event.id].location" type="text" placeholder="保育園" /></label>
          <label>説明<textarea v-model="eventDrafts[event.id].description" rows="3"></textarea></label>
        </fieldset>
        <div class="candidate-actions">
          <button v-if="event.status !== 'ignored'" class="primary-button" :disabled="savingCandidateId === event.id || !canEditExtractedEvent(event)" type="submit">
            {{ savingCandidateId === event.id ? '保存中...' : '保存' }}
          </button>
          <button v-if="event.status === 'ignored'" class="secondary-button" :disabled="savingCandidateId === event.id" type="button" @click="restoreIgnoredExtractedEvent(event)">
            {{ savingCandidateId === event.id ? '取消中...' : '除外を取り消す' }}
          </button>
          <button v-if="canRegisterExtractedEvent(event)" class="primary-button" :disabled="registeringCandidateId === event.id" type="button" @click="registerExtractedEvent(event)">
            {{ registeringCandidateId === event.id ? '登録中...' : 'Googleカレンダーに登録' }}
          </button>
          <button
            class="secondary-button"
            :disabled="savingCandidateId === event.id || registeringCandidateId === event.id || event.status === 'ignored' || event.status === 'registered'"
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
</template>

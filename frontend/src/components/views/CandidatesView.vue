<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import RegistrationPreview from '../RegistrationPreview.vue'
import { useOtayoriCalendarContext } from '../../composables/otayoriCalendarContext'
import type { ExtractedEvent } from '../../types'

const {
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
  focusedCandidateId,
  hasUnregisterableSelectedEvents,
  ignoreExtractedEvent,
  isCandidateDirty,
  registerExtractedEvent,
  registeringCandidateId,
  resetCandidateDraft,
  restoreIgnoredExtractedEvent,
  saveExtractedEvent,
  savingCandidateId,
  selectedCandidateIds,
  selectedCandidateLetterId,
  switchView,
  undoCandidateAction,
  undoLastCandidateAction,
} = useOtayoriCalendarContext()

type StatusFilter = 'all' | 'draft' | 'confirmed' | 'ignored' | 'registered' | 'attention'
type DateFilter = 'all' | 'upcoming' | 'past'

const statusFilter = ref<StatusFilter>('all')
const dateFilter = ref<DateFilter>('all')
const expandedCandidateIds = ref<string[]>([])
const previewEvents = ref<ExtractedEvent[]>([])
const submittedCandidateIds = ref<string[]>([])

const statusFilters: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'すべて' },
  { value: 'draft', label: '未確認' },
  { value: 'confirmed', label: '確認済み' },
  { value: 'attention', label: '要対応' },
  { value: 'ignored', label: '除外済み' },
  { value: 'registered', label: '登録済み' },
]

const filteredExtractedEvents = computed(() =>
  extractedEvents.value.filter((event) =>
    (!selectedCandidateLetterId.value || event.letter_id === selectedCandidateLetterId.value) &&
    matchesStatusFilter(event) &&
    matchesDateFilter(event),
  ),
)

const filteredSelectableEvents = computed(() => filteredExtractedEvents.value.filter(canSelectExtractedEvent))

const allFilteredCandidatesSelected = computed(() =>
  filteredSelectableEvents.value.length > 0 &&
  filteredSelectableEvents.value.every((event) => selectedCandidateIds.value.includes(event.id)),
)

watch(
  () => filteredSelectableEvents.value.map((event) => event.id),
  (visibleIds) => {
    const visibleIdSet = new Set(visibleIds)
    selectedCandidateIds.value = selectedCandidateIds.value.filter((id) => visibleIdSet.has(id))
  },
)

watch(focusedCandidateId, async (id) => {
  if (!id) return
  statusFilter.value = 'all'
  dateFilter.value = 'all'
  expandedCandidateIds.value = [...new Set([...expandedCandidateIds.value, id])]
  await nextTick()
  document.querySelector(`[data-candidate-id="${id}"]`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}, { immediate: true })

function matchesStatusFilter(event: ExtractedEvent) {
  if (statusFilter.value === 'all') return true
  if (statusFilter.value === 'attention') return event.status === 'failed' || event.status === 'deleted'
  return event.status === statusFilter.value
}

function matchesDateFilter(event: ExtractedEvent) {
  if (dateFilter.value === 'all') return true
  const today = new Date()
  const localToday = new Date(today.getTime() - today.getTimezoneOffset() * 60_000).toISOString().slice(0, 10)
  const eventDate = event.event_date.slice(0, 10)
  return dateFilter.value === 'upcoming' ? eventDate >= localToday : eventDate < localToday
}

function toggleAllFilteredCandidates(event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  const filteredIds = new Set(filteredSelectableEvents.value.map((candidate) => candidate.id))
  selectedCandidateIds.value = checked
    ? [...new Set([...selectedCandidateIds.value, ...filteredIds])]
    : selectedCandidateIds.value.filter((id) => !filteredIds.has(id))
}

function isCandidateExpanded(id: string) {
  return expandedCandidateIds.value.includes(id)
}

function toggleCandidate(id: string) {
  const event = extractedEvents.value.find((candidate) => candidate.id === id)
  if (event && isCandidateExpanded(id) && isCandidateDirty(event)) {
    if (!window.confirm('保存していない変更があります。破棄して閉じますか？')) return
    resetCandidateDraft(event)
  }
  expandedCandidateIds.value = isCandidateExpanded(id)
    ? expandedCandidateIds.value.filter((candidateId) => candidateId !== id)
    : [...expandedCandidateIds.value, id]
}

function candidateErrors(event: ExtractedEvent) {
  const draft = eventDrafts.value[event.id]
  if (!draft) return {}
  const errors: Record<string, string> = {}
  if (!draft.title.trim()) errors.title = '予定名を入力してください'
  if (!draft.event_date) errors.event_date = '日付を選択してください'
  if (!draft.is_all_day) {
    if (!draft.start_time) errors.start_time = '開始時刻を選択してください'
    if (!draft.end_time) errors.end_time = '終了時刻を選択してください'
    if (draft.start_time && draft.end_time && draft.end_time <= draft.start_time) {
      errors.end_time = '終了時刻は開始時刻より後にしてください'
    }
  }
  return errors
}

function showCandidateError(event: ExtractedEvent, field: string) {
  return (isCandidateDirty(event) || submittedCandidateIds.value.includes(event.id)) ? candidateErrors(event)[field] : ''
}

async function submitCandidate(event: ExtractedEvent) {
  submittedCandidateIds.value = [...new Set([...submittedCandidateIds.value, event.id])]
  if (Object.keys(candidateErrors(event)).length > 0) {
    await nextTick()
    const invalidInput = document.querySelector<HTMLInputElement>(`[data-candidate-id="${event.id}"] [aria-invalid="true"]`)
    invalidInput?.focus()
    invalidInput?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    return
  }
  await saveExtractedEvent(event)
  submittedCandidateIds.value = submittedCandidateIds.value.filter((id) => id !== event.id)
}

function openSinglePreview(event: ExtractedEvent) {
  previewEvents.value = [event]
}

function openBulkPreview() {
  if (!canBulkRegisterSelectedEvents()) return
  const selectedIds = new Set(selectedCandidateIds.value)
  previewEvents.value = extractedEvents.value.filter((event) => selectedIds.has(event.id))
}

function closePreview() {
  previewEvents.value = []
}

async function confirmPreview() {
  if (previewEvents.value.length === 1) {
    await registerExtractedEvent(previewEvents.value[0])
  } else {
    await bulkRegisterExtractedEvents()
  }
  closePreview()
}

async function ignorePreviewEvent(event: ExtractedEvent) {
  const ignored = await ignoreExtractedEvent(event)
  if (ignored) previewEvents.value = previewEvents.value.filter((previewEvent) => previewEvent.id !== event.id)
}
</script>

<template>
  <section id="candidates" class="workspace-section view-section">
    <div class="section-heading">
      <div><p class="section-kicker">Step 2</p><h2>予定候補を確認</h2></div>
      <p>内容を確認してから、Googleカレンダーへ登録します。</p>
    </div>
    <div v-if="extractedEvents.length === 0" class="empty-state">
      <h3>予定候補はまだありません</h3>
      <p>おたより画像を追加して、「予定候補を見つける」を押してください。</p>
      <button class="primary-button" type="button" @click="switchView('letters')">おたよりを追加する</button>
    </div>
    <div v-if="selectedCandidateLetterId" class="letter-filter-notice">
      <p>選択したおたよりの候補だけを表示しています。</p>
      <button type="button" @click="selectedCandidateLetterId = ''">すべての候補を表示</button>
    </div>
    <div v-if="extractedEvents.length > 0" class="candidate-filters surface">
      <div class="filter-group">
        <p>状態で絞り込む</p>
        <div class="filter-options">
          <button
            v-for="filter in statusFilters"
            :key="filter.value"
            :class="{ active: statusFilter === filter.value }"
            type="button"
            @click="statusFilter = filter.value"
          >
            {{ filter.label }}
          </button>
        </div>
      </div>
      <div class="filter-group">
        <p>日付で絞り込む</p>
        <div class="filter-options">
          <button :class="{ active: dateFilter === 'all' }" type="button" @click="dateFilter = 'all'">すべて</button>
          <button :class="{ active: dateFilter === 'upcoming' }" type="button" @click="dateFilter = 'upcoming'">今後の予定</button>
          <button :class="{ active: dateFilter === 'past' }" type="button" @click="dateFilter = 'past'">過去の予定</button>
        </div>
      </div>
      <p class="filter-result"><strong>{{ filteredExtractedEvents.length }}件</strong> を表示中 / 全{{ extractedEvents.length }}件</p>
    </div>
    <div v-if="extractedEvents.length > 0" class="bulk-toolbar">
      <label class="checkbox-label bulk-select-label">
        <input
          :checked="allFilteredCandidatesSelected"
          :disabled="filteredSelectableEvents.length === 0 || bulkCandidateAction !== ''"
          type="checkbox"
          @change="toggleAllFilteredCandidates"
        />
        表示中をまとめて選択
      </label>
      <p><strong>{{ selectedCandidateIds.length }}件</strong> 選択中</p>
      <div class="bulk-actions">
        <button class="secondary-button" :disabled="selectedCandidateIds.length === 0 || bulkCandidateAction !== ''" type="button" @click="bulkConfirmExtractedEvents">{{ bulkCandidateAction === 'confirm' ? '確認中...' : '確認済みにする' }}</button>
        <button class="secondary-button" :disabled="selectedCandidateIds.length === 0 || bulkCandidateAction !== ''" type="button" @click="bulkIgnoreExtractedEvents">{{ bulkCandidateAction === 'ignore' ? '除外中...' : '除外する' }}</button>
        <button class="primary-button" :disabled="!canBulkRegisterSelectedEvents() || bulkCandidateAction !== ''" type="button" @click="openBulkPreview">カレンダーへ登録</button>
      </div>
      <p v-if="hasUnregisterableSelectedEvents()" class="bulk-guidance">
        一括登録するには、選択中の予定候補を先に一括確認してください。
      </p>
    </div>
    <article
      v-for="event in filteredExtractedEvents"
      :key="event.id"
      class="surface candidate-card"
      :class="[`status-${event.status}`, { ignored: event.status === 'ignored', focused: focusedCandidateId === event.id }]"
      :data-candidate-id="event.id"
    >
      <div class="candidate-heading candidate-summary">
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
          <div>
            <h3>{{ event.title }}</h3>
            <p class="candidate-date">{{ new Date(event.event_date).toLocaleDateString('ja-JP') }}</p>
            <p v-if="event.belongings" class="candidate-detail-preview">持ち物: {{ event.belongings }}</p>
            <p v-if="event.submission_deadline" class="candidate-detail-preview">
              提出期限: {{ new Date(event.submission_deadline).toLocaleDateString('ja-JP') }}
            </p>
          </div>
        </div>
        <div class="candidate-summary-actions">
          <p v-if="event.confidence < 0.7" class="warning-chip">読み取り要確認</p>
          <p v-if="isCandidateDirty(event)" class="unsaved-chip">未保存</p>
          <button class="secondary-button candidate-expand-button" type="button" @click="toggleCandidate(event.id)">
            {{ isCandidateExpanded(event.id) ? '閉じる' : '内容を確認・編集' }}
          </button>
        </div>
      </div>

      <form v-if="eventDrafts[event.id] && isCandidateExpanded(event.id)" class="candidate-form" novalidate @submit.prevent="submitCandidate(event)">
        <p v-if="event.status === 'registered'" class="candidate-lock-message">
          Googleカレンダー登録済みのため、この画面では編集できません。
        </p>
        <p v-if="event.status === 'ignored'" class="candidate-lock-message">
          除外済みです。編集を再開する場合は、先に除外を取り消してください。
        </p>
        <fieldset class="candidate-fields" :disabled="!canEditExtractedEvent(event)">
          <label>予定名<input v-model="eventDrafts[event.id].title" :aria-invalid="!!showCandidateError(event, 'title')" required type="text" /><small v-if="showCandidateError(event, 'title')" class="field-error">{{ showCandidateError(event, 'title') }}</small></label>
          <label>日付<input v-model="eventDrafts[event.id].event_date" :aria-invalid="!!showCandidateError(event, 'event_date')" required type="date" /><small v-if="showCandidateError(event, 'event_date')" class="field-error">{{ showCandidateError(event, 'event_date') }}</small></label>
          <label class="checkbox-label"><input v-model="eventDrafts[event.id].is_all_day" type="checkbox" />終日予定</label>
          <div v-if="!eventDrafts[event.id].is_all_day" class="time-grid">
            <label>開始<input v-model="eventDrafts[event.id].start_time" :aria-invalid="!!showCandidateError(event, 'start_time')" required type="time" /><small v-if="showCandidateError(event, 'start_time')" class="field-error">{{ showCandidateError(event, 'start_time') }}</small></label>
            <label>終了<input v-model="eventDrafts[event.id].end_time" :aria-invalid="!!showCandidateError(event, 'end_time')" required type="time" /><small v-if="showCandidateError(event, 'end_time')" class="field-error">{{ showCandidateError(event, 'end_time') }}</small></label>
          </div>
          <label>場所<input v-model="eventDrafts[event.id].location" type="text" placeholder="保育園" /></label>
          <div class="candidate-important-fields">
            <label>持ち物<textarea v-model="eventDrafts[event.id].belongings" rows="2" placeholder="水筒、帽子、着替え"></textarea></label>
            <label>提出期限<input v-model="eventDrafts[event.id].submission_deadline" type="date" /></label>
          </div>
          <label>その他の注意事項<textarea v-model="eventDrafts[event.id].description" rows="3"></textarea></label>
        </fieldset>
        <div class="candidate-actions">
          <button v-if="event.status !== 'ignored'" class="primary-button" :disabled="savingCandidateId === event.id || !canEditExtractedEvent(event)" type="submit">
            {{ savingCandidateId === event.id ? '保存中...' : '保存' }}
          </button>
          <button v-if="event.status === 'ignored'" class="secondary-button" :disabled="savingCandidateId === event.id" type="button" @click="restoreIgnoredExtractedEvent(event)">
            {{ savingCandidateId === event.id ? '取消中...' : '除外を取り消す' }}
          </button>
          <button v-if="canRegisterExtractedEvent(event)" class="primary-button" :disabled="registeringCandidateId === event.id" type="button" @click="openSinglePreview(event)">
            Googleカレンダーに登録
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

      <div v-if="isCandidateExpanded(event.id)" class="source-box">
        <p>読み取り確度 {{ Math.round(event.confidence * 100) }}%</p>
        <p>{{ event.source_text || '元テキストはありません。' }}</p>
      </div>
    </article>
    <div v-if="extractedEvents.length > 0 && filteredExtractedEvents.length === 0" class="empty-state">
      条件に一致する予定候補はありません。
    </div>
    <p v-if="candidateMessage" class="notice success-notice">{{ candidateMessage }}</p>
    <div v-if="undoCandidateAction" class="notice undo-notice">
      <p>{{ undoCandidateAction.message }}</p>
      <button :disabled="bulkCandidateAction !== ''" type="button" @click="undoLastCandidateAction">取り消す</button>
    </div>
  </section>
  <RegistrationPreview
    v-if="previewEvents.length > 0"
    :events="previewEvents"
    :registering="registeringCandidateId !== '' || bulkCandidateAction === 'register'"
    @close="closePreview"
    @confirm="confirmPreview"
    @ignore="ignorePreviewEvent"
  />
</template>

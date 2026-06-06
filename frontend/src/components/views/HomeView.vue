<script setup lang="ts">
import { computed } from 'vue'
import { useOtayoriCalendarContext } from '../../composables/otayoriCalendarContext'
import { buildLetterProgress } from '../../utils/letterProgress'

const {
  attentionCalendarCount,
  extractedEvents,
  letters,
  pendingCandidateCount,
  readyCandidateCount,
  switchView,
  user,
} = useOtayoriCalendarContext()

const unfinishedLetterCount = computed(() =>
  letters.value.filter((letter) => {
    const progress = buildLetterProgress(extractedEvents.value.filter((event) => event.letter_id === letter.id))
    return progress.label !== '完了'
  }).length,
)

const upcomingImportantEvents = computed(() => {
  const today = new Date()
  const localToday = new Date(today.getTime() - today.getTimezoneOffset() * 60_000).toISOString().slice(0, 10)
  return extractedEvents.value
    .filter((event) => event.status !== 'ignored' && importantEventDate(event, localToday) !== '')
    .sort((a, b) => importantEventDate(a, localToday).localeCompare(importantEventDate(b, localToday)))
    .slice(0, 3)
})

function importantEventDate(event: (typeof extractedEvents.value)[number], localToday = '') {
  const deadline = event.submission_deadline?.slice(0, 10) ?? ''
  if (deadline && (!localToday || deadline >= localToday)) return deadline
  const eventDate = event.event_date.slice(0, 10)
  if (event.belongings && (!localToday || eventDate >= localToday)) return eventDate
  return ''
}
</script>

<template>
  <section class="welcome-panel">
    <div>
      <p class="eyebrow">Family dashboard</p>
      <h1>{{ user?.name || 'こんにちは' }}さん、<br />今日も予定を整えましょう。</h1>
      <p class="lead">おたよりを追加して、確認が必要な予定だけをすばやく片付けられます。</p>
    </div>
    <button class="primary-button" type="button" @click="switchView('letters')">おたよりを追加</button>
  </section>

  <section class="summary-grid" aria-label="現在の状況">
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

  <section v-if="upcomingImportantEvents.length > 0" class="important-events">
    <div class="section-heading">
      <div><p class="section-kicker">Don't forget</p><h2>直近の持ち物・提出期限</h2></div>
      <button class="text-button" type="button" @click="switchView('candidates')">候補を確認</button>
    </div>
    <div class="important-event-grid">
      <button v-for="event in upcomingImportantEvents" :key="event.id" class="important-event-card" type="button" @click="switchView('candidates')">
        <div class="important-event-meta">
          <span v-if="event.belongings" class="important-event-type belongings">持ち物</span>
          <span v-if="event.submission_deadline" class="important-event-type deadline">提出期限</span>
          <time>{{ importantEventDate(event) }}</time>
        </div>
        <strong>{{ event.title }}</strong>
        <small v-if="event.belongings">{{ event.belongings }}</small>
        <small v-if="event.submission_deadline">この日までに提出</small>
      </button>
    </div>
  </section>

  <section class="next-actions">
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
    <button v-if="unfinishedLetterCount > 0" class="next-action-card" type="button" @click="switchView('letters')">
      <span class="next-action-number">{{ unfinishedLetterCount }}</span>
      <span><strong>対応途中のおたよりがあります</strong><small>おたよりごとの進捗を確認しましょう</small></span>
      <span class="next-action-arrow">→</span>
    </button>
    <button class="next-action-card" type="button" @click="switchView('letters')">
      <span class="next-action-number">＋</span>
      <span><strong>新しいおたよりを追加</strong><small>園から届いた画像を予定に変換します</small></span>
      <span class="next-action-arrow">→</span>
    </button>
  </section>
</template>

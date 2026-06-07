<script setup lang="ts">
import { computed } from 'vue'
import { useOtayoriCalendarContext } from '../../composables/otayoriCalendarContext'
import type { CalendarEvent, ExtractedEvent } from '../../types'
import { buildLetterProgress } from '../../utils/letterProgress'
import ChildrenManager from '../ChildrenManager.vue'

const {
  attentionCalendarCount,
  calendarEvents,
  extractedEvents,
  letters,
  openCalendarEvent,
  openCandidate,
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

type TimelineItem = {
  key: string
  date: string
  title: string
  detail: string
  type: 'event' | 'belongings' | 'deadline'
  event?: ExtractedEvent
  calendarEvent?: CalendarEvent
}

const weeklyTimeline = computed(() => {
  const today = localDate(new Date())
  const weekEnd = addDays(today, 6)
  const items: TimelineItem[] = []
  const registeredCandidateIds = new Set(
    calendarEvents.value.filter((event) => event.source_type === 'extracted').map((event) => event.id),
  )

  extractedEvents.value
    .filter((event) => event.status !== 'ignored')
    .forEach((event) => {
      const eventDate = event.event_date.slice(0, 10)
      if (!registeredCandidateIds.has(event.id) && eventDate >= today && eventDate <= weekEnd) {
        items.push({ key: `event-${event.id}`, date: eventDate, title: event.title, detail: withChild(event, event.location || '予定'), type: 'event', event })
      }
      if (event.belongings && eventDate >= today && eventDate <= weekEnd) {
        items.push({ key: `belongings-${event.id}`, date: eventDate, title: event.title, detail: withChild(event, event.belongings), type: 'belongings', event })
      }
      const deadline = event.submission_deadline?.slice(0, 10)
      if (deadline && deadline >= addDays(today, -7) && deadline <= weekEnd) {
        items.push({ key: `deadline-${event.id}`, date: deadline, title: event.title, detail: withChild(event, 'この日までに提出'), type: 'deadline', event })
      }
    })

  calendarEvents.value
    .filter((event) => event.status === 'registered')
    .forEach((event) => {
      const date = event.event_date.slice(0, 10)
      if (date >= today && date <= weekEnd) {
        items.push({ key: `calendar-${event.source_type}-${event.id}`, date, title: event.title, detail: [event.child_name, event.location || 'Googleカレンダー登録済み'].filter(Boolean).join(' · '), type: 'event', calendarEvent: event })
      }
    })

  return items.sort((a, b) => a.date.localeCompare(b.date) || timelineTypeOrder(a.type) - timelineTypeOrder(b.type))
})

const timelineGroups = computed(() => {
  const groups = new Map<string, TimelineItem[]>()
  weeklyTimeline.value.forEach((item) => groups.set(item.date, [...(groups.get(item.date) ?? []), item]))
  return Array.from(groups, ([date, items]) => ({ date, items }))
})

function openTimelineItem(item: TimelineItem) {
  if (item.calendarEvent) {
    openCalendarEvent(item.calendarEvent.source_type, item.calendarEvent.id)
    return
  }
  if (item.event) openCandidate(item.event.id)
}

function withChild(event: ExtractedEvent, detail: string) {
  const childName = letters.value.find((letter) => letter.id === event.letter_id)?.child_name
  return [childName, detail].filter(Boolean).join(' · ')
}

function localDate(date: Date) {
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 10)
}

function addDays(date: string, days: number) {
  const value = new Date(`${date}T00:00:00`)
  value.setDate(value.getDate() + days)
  return localDate(value)
}

function dateHeading(date: string) {
  const today = localDate(new Date())
  if (date < today) return `期限超過 · ${formatShortDate(date)}`
  if (date === today) return `今日 · ${formatShortDate(date)}`
  if (date === addDays(today, 1)) return `明日 · ${formatShortDate(date)}`
  return formatShortDate(date)
}

function formatShortDate(date: string) {
  return new Date(`${date}T00:00:00`).toLocaleDateString('ja-JP', { month: 'numeric', day: 'numeric', weekday: 'short' })
}

function timelineTypeOrder(type: TimelineItem['type']) {
  return { deadline: 0, belongings: 1, event: 2 }[type]
}

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

  <section class="weekly-timeline">
    <div class="section-heading">
      <div><p class="section-kicker">This week</p><h2>今日から1週間</h2></div>
      <p>予定・持ち物・提出期限を日付順で確認できます。</p>
    </div>
    <div v-if="timelineGroups.length === 0" class="empty-state compact-empty-state">
      <h3>今週の予定はありません</h3>
      <p>新しいおたよりを追加すると、ここに直近の予定が表示されます。</p>
    </div>
    <div v-else class="timeline-groups">
      <section v-for="group in timelineGroups" :key="group.date" class="timeline-group" :class="{ overdue: group.date < localDate(new Date()) }">
        <h3>{{ dateHeading(group.date) }}</h3>
        <button v-for="item in group.items" :key="item.key" class="timeline-item" type="button" @click="openTimelineItem(item)">
          <span class="timeline-type" :class="item.type">{{ { event: '予定', belongings: '持ち物', deadline: '提出期限' }[item.type] }}</span>
          <span><strong>{{ item.title }}</strong><small>{{ item.detail }}</small></span>
          <span class="next-action-arrow" aria-hidden="true">→</span>
        </button>
      </section>
    </div>
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
        <span v-if="letters.find((letter) => letter.id === event.letter_id)?.child_name" class="child-chip" :style="{ '--child-color': letters.find((letter) => letter.id === event.letter_id)?.child_color }">{{ letters.find((letter) => letter.id === event.letter_id)?.child_name }}</span>
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
      <span class="next-action-arrow" aria-hidden="true">→</span>
    </button>
    <button v-if="attentionCalendarCount > 0" class="next-action-card attention" type="button" @click="switchView('calendar')">
      <span class="next-action-number">{{ attentionCalendarCount }}</span>
      <span><strong>カレンダー予定に確認が必要です</strong><small>失敗・削除済みの予定を確認しましょう</small></span>
      <span class="next-action-arrow" aria-hidden="true">→</span>
    </button>
    <button v-if="unfinishedLetterCount > 0" class="next-action-card" type="button" @click="switchView('letters')">
      <span class="next-action-number">{{ unfinishedLetterCount }}</span>
      <span><strong>対応途中のおたよりがあります</strong><small>おたよりごとの進捗を確認しましょう</small></span>
      <span class="next-action-arrow" aria-hidden="true">→</span>
    </button>
    <button class="next-action-card" type="button" @click="switchView('letters')">
      <span class="next-action-number">＋</span>
      <span><strong>新しいおたよりを追加</strong><small>園から届いた画像を予定に変換します</small></span>
      <span class="next-action-arrow" aria-hidden="true">→</span>
    </button>
  </section>

  <ChildrenManager />
</template>

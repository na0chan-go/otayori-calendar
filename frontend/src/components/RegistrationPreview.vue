<script setup lang="ts">
import { computed } from 'vue'
import { useOtayoriCalendarContext } from '../composables/otayoriCalendarContext'
import type { CalendarEvent, ExtractedEvent } from '../types'

const props = defineProps<{ events: ExtractedEvent[]; registering: boolean }>()
const emit = defineEmits<{ close: []; confirm: []; ignore: [event: ExtractedEvent] }>()
const { calendarEvents } = useOtayoriCalendarContext()

const duplicateMatches = computed(() => {
  const matches = new Map<string, string[]>()
  props.events.forEach((event) => {
    const eventMatches: string[] = []

    props.events.forEach((candidate) => {
      if (candidate.id !== event.id && isSimilarEvent(event.title, event.event_date, candidate.title, candidate.event_date)) {
        eventMatches.push(`今回の登録対象: ${candidate.title}`)
      }
    })

    calendarEvents.value
      .filter((calendarEvent) => calendarEvent.status === 'registered')
      .forEach((calendarEvent) => {
        if (isSimilarEvent(event.title, event.event_date, calendarEvent.title, calendarEvent.event_date)) {
          eventMatches.push(`${sourceLabel(calendarEvent)}: ${calendarEvent.title}`)
        }
      })

    matches.set(event.id, [...new Set(eventMatches)])
  })
  return matches
})

function isDuplicate(event: ExtractedEvent) {
  return (duplicateMatches.value.get(event.id)?.length ?? 0) > 0
}

function timeLabel(event: ExtractedEvent) {
  if (event.is_all_day) return '終日'
  return `${event.start_time?.slice(0, 5) || '未設定'} - ${event.end_time?.slice(0, 5) || '未設定'}`
}

function isSimilarEvent(title: string, date: string, otherTitle: string, otherDate: string) {
  if (date.slice(0, 10) !== otherDate.slice(0, 10)) return false
  const normalized = normalizeTitle(title)
  const otherNormalized = normalizeTitle(otherTitle)
  if (normalized.length < 3 || otherNormalized.length < 3) return normalized === otherNormalized
  return normalized === otherNormalized || normalized.includes(otherNormalized) || otherNormalized.includes(normalized)
}

function normalizeTitle(title: string) {
  return title.toLocaleLowerCase('ja-JP').replace(/[\s\u3000・、。,.!！?？「」『』（）()【】[\]_-]/g, '')
}

function sourceLabel(event: CalendarEvent) {
  return event.source_type === 'manual' ? '登録済みの手入力予定' : '登録済みのおたより予定'
}
</script>

<template>
  <div class="preview-backdrop" role="presentation" @click.self="emit('close')">
    <section class="preview-dialog" role="dialog" aria-modal="true" aria-labelledby="preview-title">
      <button class="guide-close" type="button" aria-label="登録プレビューを閉じる" @click="emit('close')">×</button>
      <p class="section-kicker">Final check</p>
      <h2 id="preview-title">Googleカレンダーへ登録しますか？</h2>
      <p class="preview-lead"><strong>{{ events.length }}件</strong>の予定を登録します。内容に誤りがあれば、戻って修正してください。</p>

      <div class="preview-list">
        <article v-for="event in events" :key="event.id" class="preview-event">
          <div>
            <p class="status-chip">{{ event.event_date.slice(0, 10) }}</p>
            <h3>{{ event.title }}</h3>
          </div>
          <dl>
            <div><dt>時間</dt><dd>{{ timeLabel(event) }}</dd></div>
            <div v-if="event.location"><dt>場所</dt><dd>{{ event.location }}</dd></div>
            <div v-if="event.description"><dt>説明</dt><dd>{{ event.description }}</dd></div>
          </dl>
          <div v-if="event.confidence < 0.7 || isDuplicate(event)" class="preview-warnings">
            <p v-if="event.confidence < 0.7">読み取り確度が低いため、内容を再確認してください。</p>
            <template v-if="isDuplicate(event)">
              <p>同じ日付に類似する予定があります。</p>
              <ul>
                <li v-for="match in duplicateMatches.get(event.id)" :key="match">{{ match }}</li>
              </ul>
              <button class="preview-ignore-button" :disabled="registering" type="button" @click="emit('ignore', event)">
                この候補を除外する
              </button>
            </template>
          </div>
        </article>
      </div>

      <div class="preview-actions">
        <button class="secondary-button" :disabled="registering" type="button" @click="emit('close')">戻って修正する</button>
        <button class="primary-button" :disabled="registering" type="button" @click="emit('confirm')">
          {{ registering ? '登録中...' : `${events.length}件を登録する` }}
        </button>
      </div>
    </section>
  </div>
</template>

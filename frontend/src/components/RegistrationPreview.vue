<script setup lang="ts">
import { computed } from 'vue'
import type { ExtractedEvent } from '../types'

const props = defineProps<{ events: ExtractedEvent[]; registering: boolean }>()
const emit = defineEmits<{ close: []; confirm: [] }>()
const duplicateKeys = computed(() => {
  const counts = new Map<string, number>()
  props.events.forEach((event) => {
    const key = `${event.title}|${event.event_date.slice(0, 10)}`
    counts.set(key, (counts.get(key) ?? 0) + 1)
  })
  return new Set([...counts].filter(([, count]) => count > 1).map(([key]) => key))
})

function isDuplicate(event: ExtractedEvent) {
  return duplicateKeys.value.has(`${event.title}|${event.event_date.slice(0, 10)}`)
}

function timeLabel(event: ExtractedEvent) {
  if (event.is_all_day) return '終日'
  return `${event.start_time?.slice(0, 5) || '未設定'} - ${event.end_time?.slice(0, 5) || '未設定'}`
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
            <p v-if="isDuplicate(event)">同じ予定名・日付の候補が登録対象に含まれています。</p>
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

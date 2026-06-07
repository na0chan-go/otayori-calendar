<script setup lang="ts">
import { computed } from 'vue'
import { useOtayoriCalendarContext } from '../composables/otayoriCalendarContext'

const {
  bulkCandidateAction,
  deletingLetterId,
  errorMessage,
  extractedEvents,
  extractingLetterId,
  isOnline,
  letters,
  refreshingData,
  refreshData,
  registeringCandidateId,
  retryingCalendarEventId,
  savingCandidateId,
  savingEvent,
  selectedCandidateIds,
  uploadingLetter,
} = useOtayoriCalendarContext()

const operation = computed(() => {
  if (uploadingLetter.value) return { title: 'おたよりをアップロード中', detail: '画像を安全に保存しています。このままお待ちください。' }
  if (extractingLetterId.value) return { title: '予定候補を抽出中', detail: `${letterTitle(extractingLetterId.value)}からAIが予定を探しています。` }
  if (deletingLetterId.value) return { title: 'おたよりを削除中', detail: `${letterTitle(deletingLetterId.value)}と紐づく候補を整理しています。` }
  if (bulkCandidateAction.value) return { title: bulkTitle(bulkCandidateAction.value), detail: `${selectedCandidateIds.value.length}件を処理しています。完了するまで同じ操作を再実行しないでください。` }
  if (registeringCandidateId.value) return { title: 'Googleカレンダーへ登録中', detail: candidateTitle(registeringCandidateId.value) }
  if (savingCandidateId.value) return { title: '予定候補を更新中', detail: candidateTitle(savingCandidateId.value) }
  if (retryingCalendarEventId.value) return { title: 'Googleカレンダーへ再登録中', detail: '登録結果を確認しています。' }
  if (savingEvent.value) return { title: '手入力予定を登録中', detail: 'Googleカレンダーの登録結果を確認しています。' }
  if (refreshingData.value) return { title: '最新状態を確認中', detail: 'サーバー上の現在の状態を読み込んでいます。' }
  return null
})

function letterTitle(id: string) {
  return `「${letters.value.find((letter) => letter.id === id)?.title || 'おたより'}」`
}

function candidateTitle(id: string) {
  return `「${extractedEvents.value.find((event) => event.id === id)?.title || '予定候補'}」を処理しています。`
}

function bulkTitle(action: string) {
  return { confirm: '予定候補を一括確認中', ignore: '予定候補を一括除外中', register: 'Googleカレンダーへ一括登録中', undo: '操作を取り消し中' }[action] || '予定候補を処理中'
}
</script>

<template>
  <aside v-if="!isOnline || operation || errorMessage" class="operation-status" :class="{ offline: !isOnline, failed: errorMessage && isOnline && !operation }">
    <span class="operation-indicator" :class="{ active: operation }"></span>
    <div>
      <strong>{{ !isOnline ? 'インターネット接続がありません' : operation?.title || '処理を完了できませんでした' }}</strong>
      <small>{{ !isOnline ? '入力内容はそのままです。接続が戻ったら最新状態を確認してください。' : operation?.detail || '再実行する前に、サーバー上の現在の状態を確認してください。' }}</small>
    </div>
    <button v-if="!operation" class="secondary-button" :disabled="!isOnline || refreshingData" type="button" @click="refreshData">
      {{ refreshingData ? '確認中...' : '最新状態を確認' }}
    </button>
  </aside>
</template>

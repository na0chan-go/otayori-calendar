<script setup lang="ts">
import { useOtayoriCalendarContext } from '../../composables/otayoriCalendarContext'

const {
  attentionCalendarCount,
  letters,
  pendingCandidateCount,
  readyCandidateCount,
  switchView,
  user,
} = useOtayoriCalendarContext()
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
    <button class="next-action-card" type="button" @click="switchView('letters')">
      <span class="next-action-number">＋</span>
      <span><strong>新しいおたよりを追加</strong><small>園から届いた画像を予定に変換します</small></span>
      <span class="next-action-arrow">→</span>
    </button>
  </section>
</template>

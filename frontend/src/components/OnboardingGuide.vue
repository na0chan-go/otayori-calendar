<script setup lang="ts">
import { useOtayoriCalendarContext } from '../composables/otayoriCalendarContext'

const { closeOnboardingGuide, switchView } = useOtayoriCalendarContext()

function startWithLetter() {
  closeOnboardingGuide()
  switchView('letters')
}
</script>

<template>
  <div class="guide-backdrop" role="presentation" @click.self="closeOnboardingGuide">
    <section class="guide-dialog" role="dialog" aria-modal="true" aria-labelledby="guide-title">
      <button class="guide-close" type="button" aria-label="使い方ガイドを閉じる" @click="closeOnboardingGuide">×</button>
      <p class="section-kicker">はじめての方へ</p>
      <h2 id="guide-title">おたよりを、3ステップで予定に。</h2>
      <p class="guide-lead">AIが見つけた予定は、確認するまでGoogleカレンダーへ登録されません。</p>

      <ol class="guide-steps">
        <li><span>1</span><div><strong>おたよりを追加</strong><p>園から届いた画像を選び、予定候補を見つけます。</p></div></li>
        <li><span>2</span><div><strong>内容を確認</strong><p>予定名や日付を確認・編集し、確認済みにします。</p></div></li>
        <li><span>3</span><div><strong>カレンダーへ登録</strong><p>「Googleカレンダーに登録」を押した予定だけが登録されます。</p></div></li>
      </ol>

      <div class="guide-statuses">
        <h3>予定候補の状態</h3>
        <dl>
          <div><dt>未確認</dt><dd>AIが見つけた直後。内容の確認が必要です。</dd></div>
          <div><dt>確認済み</dt><dd>内容を確認済み。まだGoogleカレンダーには未登録です。</dd></div>
          <div><dt>除外済み</dt><dd>登録しない予定です。あとから取り消せます。</dd></div>
          <div><dt>登録済み</dt><dd>Googleカレンダーへの登録が完了しています。</dd></div>
        </dl>
      </div>

      <button class="primary-button guide-start" type="button" @click="startWithLetter">おたよりを追加して始める</button>
    </section>
  </div>
</template>

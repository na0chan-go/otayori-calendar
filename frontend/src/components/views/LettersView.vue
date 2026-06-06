<script setup lang="ts">
import { computed } from 'vue'
import ImagePreparation from '../ImagePreparation.vue'
import { useOtayoriCalendarContext } from '../../composables/otayoriCalendarContext'
import type { Letter } from '../../types'
import { buildLetterProgress } from '../../utils/letterProgress'

const {
  deleteLetter,
  deletingLetterId,
  extractEvents,
  extractingLetterId,
  extractedEvents,
  letterImage,
  letterTitle,
  letters,
  ocrTextByLetter,
  onLetterImageChange,
  selectedCandidateLetterId,
  switchView,
  uploadingLetter,
  uploadLetter,
} = useOtayoriCalendarContext()

const progressByLetter = computed(() =>
  Object.fromEntries(
    letters.value.map((letter) => [
      letter.id,
      buildLetterProgress(extractedEvents.value.filter((event) => event.letter_id === letter.id)),
    ]),
  ),
)

function scrollToUpload() {
  document.querySelector('#upload')?.scrollIntoView({ behavior: 'smooth' })
}

function showLetterCandidates(letter: Letter) {
  switchView('candidates')
  selectedCandidateLetterId.value = letter.id
}
</script>

<template>
  <section id="upload" class="workspace-section view-section">
    <div class="section-heading">
      <div><p class="section-kicker">Step 1</p><h2>おたよりを追加</h2></div>
      <p>写真を選ぶだけで、AIが予定候補を探します。</p>
    </div>
    <form class="surface form-grid upload-form" @submit.prevent="uploadLetter">
      <label>おたよりの名前<input v-model="letterTitle" type="text" placeholder="例：6月のえんだより" /></label>
      <label class="file-field">画像を選択・撮影<input accept="image/jpeg,image/png,image/webp" capture="environment" required type="file" @change="onLetterImageChange" /><span>{{ letterImage?.name || 'JPEG・PNG・WebP' }}</span></label>
      <ImagePreparation v-if="letterImage" :file="letterImage" @change="letterImage = $event" />
      <button class="primary-button" :disabled="uploadingLetter" type="submit">{{ uploadingLetter ? 'アップロード中...' : 'アップロードする' }}</button>
    </form>
  </section>

  <section id="letters" class="workspace-section">
    <div class="section-heading">
      <div><p class="section-kicker">Letters</p><h2>アップロード済み</h2></div>
      <p>{{ letters.length }}件のおたよりがあります。</p>
    </div>
    <div v-if="letters.length === 0" class="empty-state">
      <h3>最初のおたよりを追加しましょう</h3>
      <p>園から届いた画像を追加すると、AIが予定候補を見つけます。</p>
      <button class="primary-button" type="button" @click="scrollToUpload">画像を選択する</button>
    </div>
    <div class="letter-grid">
      <article v-for="letter in letters" :key="letter.id" class="surface letter-card">
        <img v-if="letter.object_url" :src="letter.object_url" :alt="letter.title || 'おたより画像'" />
        <div class="letter-summary">
          <div class="letter-progress-heading">
            <p class="section-kicker">Uploaded</p>
            <p class="letter-progress-chip" :class="`tone-${progressByLetter[letter.id].tone}`">
              {{ progressByLetter[letter.id].label }}
            </p>
          </div>
          <h3>{{ letter.title || '無題のおたより' }}</h3>
          <p>{{ new Date(letter.created_at).toLocaleString('ja-JP') }}</p>
          <div v-if="progressByLetter[letter.id].total > 0" class="letter-progress-counts">
            <span>候補 {{ progressByLetter[letter.id].total }}件</span>
            <span v-if="progressByLetter[letter.id].counts.draft">未確認 {{ progressByLetter[letter.id].counts.draft }}</span>
            <span v-if="progressByLetter[letter.id].counts.confirmed">登録準備 {{ progressByLetter[letter.id].counts.confirmed }}</span>
            <span v-if="progressByLetter[letter.id].counts.attention">要対応 {{ progressByLetter[letter.id].counts.attention }}</span>
          </div>
        </div>
        <div class="ocr-panel">
          <label>補足テキスト（任意）<textarea v-model="ocrTextByLetter[letter.id]" rows="3" placeholder="画像が読みづらい場合だけ入力してください"></textarea></label>
          <p class="helper-text">空欄なら画像から予定を読み取ります。</p>
          <div class="button-row">
            <button class="primary-button" :disabled="extractingLetterId === letter.id" type="button" @click="extractEvents(letter)">{{ extractingLetterId === letter.id ? '予定を探しています...' : '予定候補を見つける' }}</button>
            <button v-if="progressByLetter[letter.id].total > 0" class="secondary-button" type="button" @click="showLetterCandidates(letter)">候補を確認</button>
            <button class="danger-button" :disabled="deletingLetterId === letter.id || extractingLetterId === letter.id" type="button" @click="deleteLetter(letter)">{{ deletingLetterId === letter.id ? '削除中...' : '削除' }}</button>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>

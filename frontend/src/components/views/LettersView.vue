<script setup lang="ts">
import { useOtayoriCalendarContext } from '../../composables/otayoriCalendarContext'

const {
  deleteLetter,
  deletingLetterId,
  extractEvents,
  extractingLetterId,
  letterImage,
  letterTitle,
  letters,
  ocrTextByLetter,
  onLetterImageChange,
  uploadingLetter,
  uploadLetter,
} = useOtayoriCalendarContext()
</script>

<template>
  <section id="upload" class="workspace-section view-section">
    <div class="section-heading">
      <div><p class="section-kicker">Step 1</p><h2>おたよりを追加</h2></div>
      <p>写真を選ぶだけで、AIが予定候補を探します。</p>
    </div>
    <form class="surface form-grid upload-form" @submit.prevent="uploadLetter">
      <label>おたよりの名前<input v-model="letterTitle" type="text" placeholder="例：6月のえんだより" /></label>
      <label class="file-field">画像を選択<input accept="image/jpeg,image/png,image/webp" required type="file" @change="onLetterImageChange" /><span>{{ letterImage?.name || 'JPEG・PNG・WebP' }}</span></label>
      <button class="primary-button" :disabled="uploadingLetter" type="submit">{{ uploadingLetter ? 'アップロード中...' : 'アップロードする' }}</button>
    </form>
  </section>

  <section id="letters" class="workspace-section">
    <div class="section-heading">
      <div><p class="section-kicker">Letters</p><h2>アップロード済み</h2></div>
      <p>{{ letters.length }}件のおたよりがあります。</p>
    </div>
    <div v-if="letters.length === 0" class="empty-state">まだおたよりはありません。最初の1枚を追加しましょう。</div>
    <div class="letter-grid">
      <article v-for="letter in letters" :key="letter.id" class="surface letter-card">
        <img v-if="letter.object_url" :src="letter.object_url" :alt="letter.title || 'おたより画像'" />
        <div class="letter-summary">
          <p class="section-kicker">Uploaded</p><h3>{{ letter.title || '無題のおたより' }}</h3>
          <p>{{ new Date(letter.created_at).toLocaleString('ja-JP') }}</p>
        </div>
        <div class="ocr-panel">
          <label>補足テキスト（任意）<textarea v-model="ocrTextByLetter[letter.id]" rows="3" placeholder="画像が読みづらい場合だけ入力してください"></textarea></label>
          <p class="helper-text">空欄なら画像から予定を読み取ります。</p>
          <div class="button-row">
            <button class="primary-button" :disabled="extractingLetterId === letter.id" type="button" @click="extractEvents(letter)">{{ extractingLetterId === letter.id ? '予定を探しています...' : '予定候補を見つける' }}</button>
            <button class="danger-button" :disabled="deletingLetterId === letter.id || extractingLetterId === letter.id" type="button" @click="deleteLetter(letter)">{{ deletingLetterId === letter.id ? '削除中...' : '削除' }}</button>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>

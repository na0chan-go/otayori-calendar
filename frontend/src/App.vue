<script setup lang="ts">
import { onMounted, ref } from 'vue'

type User = {
  id: string
  email: string
  name: string
  created_at: string
}

type Letter = {
  id: string
  title: string
  mime_type: string
  file_size: number
  image_url: string
  created_at: string
  object_url?: string
}

type ExtractedEvent = {
  id: string
  letter_id: string
  title: string
  event_date: string
  start_time: string | null
  end_time: string | null
  is_all_day: boolean
  description: string
  confidence: number
  source_text: string
}

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'
const user = ref<User | null>(null)
const loading = ref(true)
const errorMessage = ref('')
const eventMessage = ref('')
const savingEvent = ref(false)
const letterMessage = ref('')
const uploadingLetter = ref(false)
const extractingLetterId = ref('')
const letters = ref<Letter[]>([])
const extractedEvents = ref<ExtractedEvent[]>([])
const ocrTextByLetter = ref<Record<string, string>>({})
const letterTitle = ref('')
const letterImage = ref<File | null>(null)
const manualEvent = ref({
  title: '',
  event_date: '',
  is_all_day: true,
  start_time: '',
  end_time: '',
  location: '',
  description: '',
})

async function loadMe() {
  loading.value = true
  errorMessage.value = ''

  try {
    const response = await fetch(`${apiBaseUrl}/api/me`, {
      credentials: 'include',
    })

    if (response.status === 401) {
      user.value = null
      return
    }
    if (!response.ok) {
      throw new Error('ユーザー情報を取得できませんでした')
    }

    const body = (await response.json()) as { user: User }
    user.value = body.user
    await loadLetters()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '予期しないエラーが発生しました'
  } finally {
    loading.value = false
  }
}

function loginWithGoogle() {
  window.location.href = `${apiBaseUrl}/auth/google/login`
}

async function logout() {
  await fetch(`${apiBaseUrl}/auth/logout`, {
    method: 'POST',
    credentials: 'include',
  })
  user.value = null
  clearLetterObjectUrls()
  letters.value = []
  extractedEvents.value = []
  ocrTextByLetter.value = {}
}

function onLetterImageChange(event: Event) {
  const input = event.target as HTMLInputElement
  letterImage.value = input.files?.[0] ?? null
}

async function uploadLetter() {
  if (!letterImage.value) {
    errorMessage.value = '画像を選択してください'
    return
  }

  uploadingLetter.value = true
  letterMessage.value = ''
  errorMessage.value = ''

  try {
    const formData = new FormData()
    formData.append('image', letterImage.value)
    formData.append('title', letterTitle.value)

    const response = await fetch(`${apiBaseUrl}/api/letters`, {
      method: 'POST',
      credentials: 'include',
      body: formData,
    })
    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as { message?: string } | null
      throw new Error(body?.message ?? 'おたより画像をアップロードできませんでした')
    }

    letterTitle.value = ''
    letterImage.value = null
    letterMessage.value = 'おたより画像をアップロードしました'
    await loadLetters()
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : 'おたより画像のアップロードでエラーが発生しました'
  } finally {
    uploadingLetter.value = false
  }
}

async function extractEvents(letter: Letter) {
  const ocrText = ocrTextByLetter.value[letter.id]?.trim()
  if (!ocrText) {
    errorMessage.value = 'OCRテキストを入力してください'
    return
  }

  extractingLetterId.value = letter.id
  letterMessage.value = ''
  errorMessage.value = ''

  try {
    const response = await fetch(`${apiBaseUrl}/api/letters/${letter.id}/extract-events`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ ocr_text: ocrText }),
    })
    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as { message?: string } | null
      throw new Error(body?.message ?? '予定候補を抽出できませんでした')
    }

    const body = (await response.json()) as { events: ExtractedEvent[] }
    extractedEvents.value = body.events
    letterMessage.value = '予定候補をdraftとして保存しました'
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : '予定候補の抽出でエラーが発生しました'
  } finally {
    extractingLetterId.value = ''
  }
}

async function loadLetters() {
  const response = await fetch(`${apiBaseUrl}/api/letters`, {
    credentials: 'include',
  })
  if (!response.ok) {
    throw new Error('おたより一覧を取得できませんでした')
  }

  const body = (await response.json()) as { letters: Letter[] }
  clearLetterObjectUrls()
  letters.value = await Promise.all(
    body.letters.map(async (letter) => ({
      ...letter,
      object_url: await loadLetterImage(letter.image_url),
    })),
  )
}

async function loadLetterImage(imageUrl: string) {
  const response = await fetch(`${apiBaseUrl}${imageUrl}`, {
    credentials: 'include',
  })
  if (!response.ok) {
    throw new Error('おたより画像を取得できませんでした')
  }

  return URL.createObjectURL(await response.blob())
}

function clearLetterObjectUrls() {
  letters.value.forEach((letter) => {
    if (letter.object_url) URL.revokeObjectURL(letter.object_url)
  })
}

async function createManualEvent() {
  savingEvent.value = true
  eventMessage.value = ''
  errorMessage.value = ''

  try {
    const response = await fetch(`${apiBaseUrl}/api/manual-events`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(manualEvent.value),
    })

    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as { message?: string } | null
      throw new Error(body?.message ?? '予定を登録できませんでした')
    }

    eventMessage.value = 'Googleカレンダーに登録しました'
    manualEvent.value = {
      title: '',
      event_date: '',
      is_all_day: true,
      start_time: '',
      end_time: '',
      location: '',
      description: '',
    }
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '予定登録でエラーが発生しました'
  } finally {
    savingEvent.value = false
  }
}

onMounted(loadMe)
</script>

<template>
  <main class="page-shell">
    <section class="hero-card">
      <p class="eyebrow">Otayori Calendar</p>
      <h1>おたよりの予定を、Googleカレンダーへ。</h1>
      <p class="lead">
        まずはGoogleアカウントでログインして、カレンダー連携の準備をします。
      </p>

      <div v-if="loading" class="panel">ログイン状態を確認しています...</div>

      <div v-else-if="user" class="panel signed-in">
        <p class="label">ログイン中</p>
        <h2>{{ user.name || user.email }}</h2>
        <p>{{ user.email }}</p>
        <button class="ghost-button" type="button" @click="logout">ログアウト</button>
      </div>

      <form v-if="user" class="event-form" @submit.prevent="uploadLetter">
        <p class="label">おたより画像</p>
        <label>
          タイトル
          <input v-model="letterTitle" type="text" placeholder="6月のおたより" />
        </label>
        <label>
          画像
          <input accept="image/jpeg,image/png,image/webp" required type="file" @change="onLetterImageChange" />
        </label>
        <button class="google-button" :disabled="uploadingLetter" type="submit">
          {{ uploadingLetter ? 'アップロード中...' : '画像をアップロード' }}
        </button>
        <p v-if="letterMessage" class="success">{{ letterMessage }}</p>
      </form>

      <section v-if="user" class="letters-section">
        <p class="label">アップロード済み</p>
        <div v-if="letters.length === 0" class="empty-state">まだおたより画像はありません。</div>
        <article v-for="letter in letters" :key="letter.id" class="letter-card">
          <img v-if="letter.object_url" :src="letter.object_url" :alt="letter.title || 'おたより画像'" />
          <div>
            <h3>{{ letter.title || '無題のおたより' }}</h3>
            <p>{{ new Date(letter.created_at).toLocaleString('ja-JP') }}</p>
          </div>
          <div class="ocr-panel">
            <label>
              OCRテキスト
              <textarea
                v-model="ocrTextByLetter[letter.id]"
                rows="4"
                placeholder="6月12日（金）身体測定を行います。"
              ></textarea>
            </label>
            <button
              class="ghost-button"
              :disabled="extractingLetterId === letter.id"
              type="button"
              @click="extractEvents(letter)"
            >
              {{ extractingLetterId === letter.id ? '抽出中...' : '予定候補を抽出' }}
            </button>
          </div>
        </article>
      </section>

      <section v-if="user && extractedEvents.length > 0" class="letters-section">
        <p class="label">抽出された予定候補</p>
        <article v-for="event in extractedEvents" :key="event.id" class="candidate-card">
          <h3>{{ event.title }}</h3>
          <p>{{ new Date(event.event_date).toLocaleDateString('ja-JP') }}</p>
          <p v-if="event.description">{{ event.description }}</p>
          <p class="source-text">confidence: {{ event.confidence.toFixed(2) }}</p>
          <p class="source-text">{{ event.source_text }}</p>
        </article>
      </section>

      <form v-if="user" class="event-form" @submit.prevent="createManualEvent">
        <p class="label">手入力予定</p>
        <label>
          予定名
          <input v-model="manualEvent.title" required type="text" placeholder="身体測定" />
        </label>
        <label>
          日付
          <input v-model="manualEvent.event_date" required type="date" />
        </label>
        <label class="checkbox-label">
          <input v-model="manualEvent.is_all_day" type="checkbox" />
          終日予定として登録する
        </label>
        <div v-if="!manualEvent.is_all_day" class="time-grid">
          <label>
            開始
            <input v-model="manualEvent.start_time" required type="time" />
          </label>
          <label>
            終了
            <input v-model="manualEvent.end_time" required type="time" />
          </label>
        </div>
        <label>
          場所
          <input v-model="manualEvent.location" type="text" placeholder="保育園" />
        </label>
        <label>
          メモ
          <textarea v-model="manualEvent.description" rows="3" placeholder="持ち物など"></textarea>
        </label>
        <button class="google-button" :disabled="savingEvent" type="submit">
          {{ savingEvent ? '登録中...' : 'Googleカレンダーに登録' }}
        </button>
        <p v-if="eventMessage" class="success">{{ eventMessage }}</p>
      </form>

      <div v-else class="actions">
        <button class="google-button" type="button" @click="loginWithGoogle">
          Googleでログイン
        </button>
        <p class="hint">Google Calendar API の許可画面へ移動します。</p>
      </div>

      <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
    </section>
  </main>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'

type User = {
  id: string
  email: string
  name: string
  created_at: string
}

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'
const user = ref<User | null>(null)
const loading = ref(true)
const errorMessage = ref('')
const eventMessage = ref('')
const savingEvent = ref(false)
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

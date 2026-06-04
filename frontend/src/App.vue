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

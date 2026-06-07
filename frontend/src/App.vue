<script setup lang="ts">
import CalendarView from './components/views/CalendarView.vue'
import CandidatesView from './components/views/CandidatesView.vue'
import HomeView from './components/views/HomeView.vue'
import LettersView from './components/views/LettersView.vue'
import OnboardingGuide from './components/OnboardingGuide.vue'
import OperationStatus from './components/OperationStatus.vue'
import { provideOtayoriCalendar } from './composables/otayoriCalendarContext'
import { useOtayoriCalendar } from './composables/useOtayoriCalendar'

const calendar = useOtayoriCalendar()
provideOtayoriCalendar(calendar)

const {
  activeView,
  attentionCalendarCount,
  errorMessage,
  letterMessage,
  loading,
  loginWithGoogle,
  logout,
  openOnboardingGuide,
  pendingCandidateCount,
  showOnboardingGuide,
  switchView,
  user,
} = calendar
</script>

<template>
  <main class="page-shell">
    <div v-if="loading" class="loading-screen">
      <span class="loading-dot"></span>
      <p>カレンダーを準備しています</p>
    </div>

    <section v-else-if="!user" class="login-shell">
      <div class="login-mark">OC</div>
      <p class="eyebrow">Otayori Calendar</p>
      <h1>家族の予定に、<br />見落とさない安心を。</h1>
      <p class="lead">園から届くおたよりを読み取り、大切な予定をGoogleカレンダーへまとめます。</p>
      <button class="primary-button login-button" type="button" @click="loginWithGoogle">Googleでログイン</button>
      <p class="hint">Googleカレンダーへの登録に必要な権限だけを使用します。</p>
    </section>

    <div v-else class="app-layout">
      <header class="app-header">
        <a class="brand" href="#top" aria-label="トップへ">
          <span class="brand-mark">OC</span>
          <span><strong>おたよりカレンダー</strong><small>家族の予定を、ひとつに</small></span>
        </a>
        <div class="account">
          <span class="account-avatar">{{ (user.name || user.email).slice(0, 1) }}</span>
          <span class="account-copy"><strong>{{ user.name || 'ログイン中' }}</strong><small>{{ user.email }}</small></span>
          <button class="text-button guide-button" type="button" @click="openOnboardingGuide">使い方</button>
          <button class="text-button" type="button" @click="logout">ログアウト</button>
        </div>
      </header>

      <div id="top" class="content-shell">
        <OperationStatus />
        <HomeView v-if="activeView === 'home'" />
        <p v-if="errorMessage" class="notice error-notice">{{ errorMessage }}</p>
        <p v-if="letterMessage" class="notice success-notice">{{ letterMessage }}</p>
        <LettersView v-if="activeView === 'letters'" />
        <CandidatesView v-if="activeView === 'candidates'" />
        <CalendarView v-if="activeView === 'calendar'" />
      </div>

      <nav class="bottom-nav" aria-label="画面切り替え">
        <button :class="{ active: activeView === 'home' }" type="button" @click="switchView('home')"><span>⌂</span><small>ホーム</small></button>
        <button :class="{ active: activeView === 'letters' }" type="button" @click="switchView('letters')"><span>▧</span><small>おたより</small></button>
        <button :class="{ active: activeView === 'candidates' }" type="button" @click="switchView('candidates')"><span>✓</span><small>予定候補</small><b v-if="pendingCandidateCount">{{ pendingCandidateCount }}</b></button>
        <button :class="{ active: activeView === 'calendar' }" type="button" @click="switchView('calendar')"><span>□</span><small>カレンダー</small><b v-if="attentionCalendarCount">{{ attentionCalendarCount }}</b></button>
      </nav>
      <OnboardingGuide v-if="showOnboardingGuide" />
    </div>
  </main>
</template>

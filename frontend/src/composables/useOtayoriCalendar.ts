import { onMounted, ref } from 'vue'
import type { User, ViewName } from '../types'
import { apiBaseUrl } from './api'
import { useCalendarEvents } from './useCalendarEvents'
import { useExtractedEvents } from './useExtractedEvents'
import { useLetters } from './useLetters'

export function useOtayoriCalendar() {
  const activeView = ref<ViewName>('home')
  const errorMessage = ref('')
  const loading = ref(true)
  const showOnboardingGuide = ref(false)
  const user = ref<User | null>(null)

  const calendar = useCalendarEvents(errorMessage)
  const candidates = useExtractedEvents(errorMessage, calendar.loadCalendarEvents)
  const letters = useLetters(errorMessage, {
    candidateCountForLetter: (letterId) =>
      candidates.extractedEvents.value.filter((event) => event.letter_id === letterId).length,
    mergeExtractedEvents: candidates.mergeExtractedEvents,
    refreshRelatedData: async () => {
      await Promise.all([candidates.loadExtractedEvents(), calendar.loadCalendarEvents()])
    },
  })

  function switchView(view: ViewName) {
    if (view === 'candidates') candidates.selectedCandidateLetterId.value = ''
    activeView.value = view
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  async function loadMe() {
    loading.value = true
    errorMessage.value = ''

    try {
      const response = await fetch(`${apiBaseUrl}/api/me`, { credentials: 'include' })
      if (response.status === 401) {
        user.value = null
        return
      }
      if (!response.ok) throw new Error('ユーザー情報を取得できませんでした')

      const body = (await response.json()) as { user: User }
      user.value = body.user
      showOnboardingGuide.value = !localStorage.getItem(onboardingStorageKey(body.user.id))
      await Promise.all([letters.loadLetters(), candidates.loadExtractedEvents(), calendar.loadCalendarEvents()])
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
    await fetch(`${apiBaseUrl}/auth/logout`, { method: 'POST', credentials: 'include' })
    user.value = null
    letters.resetLetters()
    candidates.resetExtractedEvents()
    calendar.resetCalendarEvents()
  }

  function openOnboardingGuide() {
    showOnboardingGuide.value = true
  }

  function closeOnboardingGuide() {
    showOnboardingGuide.value = false
    if (user.value) localStorage.setItem(onboardingStorageKey(user.value.id), 'seen')
  }

  onMounted(loadMe)

  return {
    activeView,
    ...calendar,
    ...candidates,
    errorMessage,
    ...letters,
    loading,
    loginWithGoogle,
    logout,
    closeOnboardingGuide,
    openOnboardingGuide,
    showOnboardingGuide,
    switchView,
    user,
  }
}

function onboardingStorageKey(userId: string) {
  return `otayori-calendar:onboarding:${userId}`
}

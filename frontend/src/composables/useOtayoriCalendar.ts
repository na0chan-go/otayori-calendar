import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { User, ViewName } from '../types'
import { apiBaseUrl } from './api'
import { useCalendarEvents } from './useCalendarEvents'
import { useExtractedEvents } from './useExtractedEvents'
import { useLetters } from './useLetters'
import { toUserErrorMessage } from '../utils/requestError'

export function useOtayoriCalendar() {
  const activeView = ref<ViewName>('home')
  const errorMessage = ref('')
  const focusedCalendarEventKey = ref('')
  const focusedCandidateId = ref('')
  const loading = ref(true)
  const isOnline = ref(navigator.onLine)
  const refreshingData = ref(false)
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
    if (view !== activeView.value && hasUnsavedChanges() && !window.confirm('保存していない変更があります。保存せずに画面を移動しますか？入力内容はこの画面に戻るまで保持されます。')) return
    if (view === 'candidates') candidates.selectedCandidateLetterId.value = ''
    focusedCalendarEventKey.value = ''
    focusedCandidateId.value = ''
    activeView.value = view
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  function hasUnsavedChanges() {
    return candidates.hasUnsavedCandidateChanges.value || calendar.hasUnsavedManualEvent.value
  }

  function warnBeforeUnload(event: BeforeUnloadEvent) {
    if (!hasUnsavedChanges()) return
    event.preventDefault()
  }

  function openCandidate(id: string) {
    candidates.selectedCandidateLetterId.value = ''
    focusedCandidateId.value = id
    activeView.value = 'candidates'
  }

  function openCalendarEvent(sourceType: string, id: string) {
    focusedCalendarEventKey.value = `${sourceType}-${id}`
    activeView.value = 'calendar'
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
      errorMessage.value = toUserErrorMessage(error, '予期しないエラーが発生しました')
    } finally {
      loading.value = false
    }
  }

  async function refreshData() {
    if (!user.value || refreshingData.value) return
    refreshingData.value = true
    errorMessage.value = ''
    try {
      await Promise.all([letters.loadLetters(), candidates.loadExtractedEvents(), calendar.loadCalendarEvents()])
    } catch (error) {
      errorMessage.value = toUserErrorMessage(error, '最新状態を確認できませんでした')
    } finally {
      refreshingData.value = false
    }
  }

  function updateOnlineStatus() {
    isOnline.value = navigator.onLine
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

  onMounted(() => {
    loadMe()
    window.addEventListener('beforeunload', warnBeforeUnload)
    window.addEventListener('online', updateOnlineStatus)
    window.addEventListener('offline', updateOnlineStatus)
  })
  onBeforeUnmount(() => {
    window.removeEventListener('beforeunload', warnBeforeUnload)
    window.removeEventListener('online', updateOnlineStatus)
    window.removeEventListener('offline', updateOnlineStatus)
  })

  return {
    activeView,
    ...calendar,
    ...candidates,
    errorMessage,
    focusedCalendarEventKey,
    focusedCandidateId,
    ...letters,
    loading,
    isOnline,
    loginWithGoogle,
    logout,
    closeOnboardingGuide,
    openOnboardingGuide,
    openCalendarEvent,
    openCandidate,
    showOnboardingGuide,
    refreshData,
    refreshingData,
    switchView,
    user,
  }
}

function onboardingStorageKey(userId: string) {
  return `otayori-calendar:onboarding:${userId}`
}

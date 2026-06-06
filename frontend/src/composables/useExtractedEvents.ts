import { computed, ref, type Ref } from 'vue'
import type {
  BulkExtractedEventsResponse,
  ExtractedEvent,
  ExtractedEventDraft,
  UndoCandidateAction,
  UndoStatusRestore,
} from '../types'
import { apiBaseUrl } from './api'

export function useExtractedEvents(errorMessage: Ref<string>, refreshCalendarEvents: () => Promise<void>) {
  const bulkCandidateAction = ref('')
  const candidateMessage = ref('')
  const eventDrafts = ref<Record<string, ExtractedEventDraft>>({})
  const extractedEvents = ref<ExtractedEvent[]>([])
  const registeringCandidateId = ref('')
  const savingCandidateId = ref('')
  const selectedCandidateIds = ref<string[]>([])
  const selectedCandidateLetterId = ref('')
  const undoCandidateAction = ref<UndoCandidateAction | null>(null)
  let undoTimer: ReturnType<typeof setTimeout> | undefined

  const pendingCandidateCount = computed(
    () => extractedEvents.value.filter((event) => event.status === 'draft').length,
  )
  const readyCandidateCount = computed(
    () => extractedEvents.value.filter((event) => canRegisterExtractedEvent(event)).length,
  )

  async function loadExtractedEvents() {
    const response = await fetch(`${apiBaseUrl}/api/extracted-events`, { credentials: 'include' })
    if (!response.ok) throw new Error('予定候補一覧を取得できませんでした')

    const body = (await response.json()) as { events: ExtractedEvent[] }
    extractedEvents.value = body.events
    syncEventDrafts(body.events)
    pruneSelectedCandidateIds()
  }

  function mergeExtractedEvents(events: ExtractedEvent[]) {
    const eventMap = new Map(extractedEvents.value.map((event) => [event.id, event]))
    events.forEach((event) => eventMap.set(event.id, event))
    extractedEvents.value = Array.from(eventMap.values()).sort((a, b) =>
      toDateInput(a.event_date).localeCompare(toDateInput(b.event_date)),
    )
    syncEventDrafts(extractedEvents.value)
    pruneSelectedCandidateIds()
  }

  async function saveExtractedEvent(event: ExtractedEvent) {
    if (!canEditExtractedEvent(event)) {
      candidateMessage.value =
        event.status === 'ignored'
          ? '編集するには、先に除外を取り消してください'
          : '登録済み予定はGoogleカレンダーとの不整合を防ぐため編集できません'
      return
    }
    await updateExtractedEvent(event, '予定候補を保存しました')
  }

  async function restoreIgnoredExtractedEvent(event: ExtractedEvent) {
    await updateExtractedEvent(event, '除外を取り消し、確認済みに戻しました')
  }

  async function updateExtractedEvent(event: ExtractedEvent, successMessage: string) {
    const draft = eventDrafts.value[event.id]
    if (!draft) return

    savingCandidateId.value = event.id
    candidateMessage.value = ''
    errorMessage.value = ''

    try {
      const response = await fetch(`${apiBaseUrl}/api/extracted-events/${event.id}`, {
        method: 'PATCH',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...draft,
          start_time: draft.is_all_day ? '' : draft.start_time,
          end_time: draft.is_all_day ? '' : draft.end_time,
        }),
      })
      if (!response.ok) {
        const body = (await response.json().catch(() => null)) as { message?: string } | null
        throw new Error(body?.message ?? '予定候補を更新できませんでした')
      }

      const body = (await response.json()) as { event: ExtractedEvent }
      replaceExtractedEvent(body.event)
      candidateMessage.value = successMessage
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : '予定候補の更新でエラーが発生しました'
    } finally {
      savingCandidateId.value = ''
    }
  }

  async function ignoreExtractedEvent(event: ExtractedEvent) {
    const previousStatus = event.status
    savingCandidateId.value = event.id
    candidateMessage.value = ''
    errorMessage.value = ''

    try {
      const response = await fetch(`${apiBaseUrl}/api/extracted-events/${event.id}/ignore`, {
        method: 'POST',
        credentials: 'include',
      })
      if (!response.ok) {
        const body = (await response.json().catch(() => null)) as { message?: string } | null
        throw new Error(body?.message ?? '予定候補を除外できませんでした')
      }

      const body = (await response.json()) as { event: ExtractedEvent }
      replaceExtractedEvent(body.event)
      candidateMessage.value = '予定候補を除外しました'
      setUndoCandidateAction('1件の除外を取り消し、元の状態へ戻します', [
        createUndoRestore(event.id, body.event.status, previousStatus),
      ])
      return true
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : '予定候補の除外でエラーが発生しました'
      return false
    } finally {
      savingCandidateId.value = ''
    }
  }

  async function registerExtractedEvent(event: ExtractedEvent) {
    registeringCandidateId.value = event.id
    candidateMessage.value = ''
    errorMessage.value = ''

    try {
      const response = await fetch(`${apiBaseUrl}/api/extracted-events/${event.id}/register`, {
        method: 'POST',
        credentials: 'include',
      })
      if (!response.ok) {
        const body = (await response.json().catch(() => null)) as { message?: string } | null
        throw new Error(body?.message ?? '予定候補をGoogleカレンダーへ登録できませんでした')
      }

      const body = (await response.json()) as { event: ExtractedEvent }
      replaceExtractedEvent(body.event)
      candidateMessage.value = '予定候補をGoogleカレンダーへ登録しました'
      await refreshCalendarEvents()
    } catch (error) {
      errorMessage.value =
        error instanceof Error ? error.message : '予定候補のGoogleカレンダー登録でエラーが発生しました'
    } finally {
      registeringCandidateId.value = ''
    }
  }

  async function bulkConfirmExtractedEvents() {
    await runBulkExtractedEventAction('confirm')
  }

  async function bulkIgnoreExtractedEvents() {
    await runBulkExtractedEventAction('ignore')
  }

  async function bulkRegisterExtractedEvents() {
    if (!canBulkRegisterSelectedEvents()) {
      candidateMessage.value = '一括登録するには、選択中の予定候補を先に一括確認してください'
      return
    }
    await runBulkExtractedEventAction('register')
  }

  async function runBulkExtractedEventAction(action: 'confirm' | 'ignore' | 'register') {
    const ids = [...selectedCandidateIds.value]
    if (ids.length === 0) {
      candidateMessage.value = '一括操作する予定候補を選択してください'
      return
    }

    const actionLabels = { confirm: '確認', ignore: '除外', register: 'Googleカレンダー登録' } as const
    const previousStatuses = new Map(selectedExtractedEvents().map((event) => [event.id, event.status]))
    bulkCandidateAction.value = action
    candidateMessage.value = ''
    errorMessage.value = ''

    try {
      const response = await fetch(`${apiBaseUrl}/api/extracted-events/bulk-${action}`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ids }),
      })
      if (!response.ok) {
        const body = (await response.json().catch(() => null)) as { message?: string } | null
        throw new Error(body?.message ?? `予定候補の一括${actionLabels[action]}に失敗しました`)
      }

      const body = (await response.json()) as BulkExtractedEventsResponse
      mergeExtractedEvents(body.events)
      selectedCandidateIds.value = selectedCandidateIds.value.filter((id) =>
        body.results.some((result) => result.id === id && result.status === 'failed'),
      )
      candidateMessage.value = `一括${actionLabels[action]}: 成功 ${body.summary.success}件 / 失敗 ${body.summary.failed}件`
      if (action !== 'register') {
        const restores = body.results.flatMap((result) => {
          const previousStatus = previousStatuses.get(result.id)
          const currentStatus = result.event?.status
          if (result.status !== 'success' || !previousStatus || !currentStatus || previousStatus === currentStatus) return []
          const restore = createUndoRestore(result.id, currentStatus, previousStatus)
          return restore ? [restore] : []
        })
        setUndoCandidateAction(
          `${restores.length}件の一括${actionLabels[action]}を取り消し、元の状態へ戻します`,
          restores,
        )
      }
      if (action === 'register') await refreshCalendarEvents()
    } catch (error) {
      errorMessage.value =
        error instanceof Error ? error.message : `予定候補の一括${actionLabels[action]}でエラーが発生しました`
    } finally {
      bulkCandidateAction.value = ''
    }
  }

  async function undoLastCandidateAction() {
    const action = undoCandidateAction.value
    if (!action) return

    clearUndoCandidateAction()
    candidateMessage.value = ''
    errorMessage.value = ''
    bulkCandidateAction.value = 'undo'

    try {
      const response = await fetch(`${apiBaseUrl}/api/extracted-events/restore-statuses`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ events: action.restores }),
      })
      if (!response.ok) {
        const body = (await response.json().catch(() => null)) as { message?: string } | null
        throw new Error(body?.message ?? '取り消しに失敗しました')
      }

      const body = (await response.json()) as BulkExtractedEventsResponse
      mergeExtractedEvents(body.events)
      candidateMessage.value =
        body.summary.failed === 0
          ? `${body.summary.success}件の操作を取り消しました`
          : `取り消し: 成功 ${body.summary.success}件 / 失敗 ${body.summary.failed}件。現在の状態を確認してください`
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : '取り消しでエラーが発生しました'
      await loadExtractedEvents().catch(() => undefined)
    } finally {
      bulkCandidateAction.value = ''
    }
  }

  function setUndoCandidateAction(message: string, restores: (UndoStatusRestore | null)[]) {
    const safeRestores = restores.filter((restore): restore is UndoStatusRestore => restore !== null)
    clearUndoCandidateAction()
    if (safeRestores.length === 0) return

    undoCandidateAction.value = { message, restores: safeRestores }
    undoTimer = setTimeout(clearUndoCandidateAction, 8000)
  }

  function clearUndoCandidateAction() {
    if (undoTimer) clearTimeout(undoTimer)
    undoTimer = undefined
    undoCandidateAction.value = null
  }

  function selectedExtractedEvents() {
    const selectedIds = new Set(selectedCandidateIds.value)
    return extractedEvents.value.filter((event) => selectedIds.has(event.id))
  }

  function canBulkRegisterSelectedEvents() {
    const events = selectedExtractedEvents()
    return events.length > 0 && events.every(canRegisterExtractedEvent)
  }

  function hasUnregisterableSelectedEvents() {
    return selectedCandidateIds.value.length > 0 && !canBulkRegisterSelectedEvents()
  }

  function selectableExtractedEvents() {
    return extractedEvents.value.filter(canSelectExtractedEvent)
  }

  function allSelectableCandidatesSelected() {
    const ids = selectableExtractedEvents().map((event) => event.id)
    return ids.length > 0 && ids.every((id) => selectedCandidateIds.value.includes(id))
  }

  function toggleAllSelectableCandidates(event: Event) {
    const checked = (event.target as HTMLInputElement).checked
    selectedCandidateIds.value = checked ? selectableExtractedEvents().map((candidate) => candidate.id) : []
  }

  function pruneSelectedCandidateIds() {
    const selectableIds = new Set(selectableExtractedEvents().map((event) => event.id))
    selectedCandidateIds.value = selectedCandidateIds.value.filter((id) => selectableIds.has(id))
  }

  function replaceExtractedEvent(nextEvent: ExtractedEvent) {
    extractedEvents.value = extractedEvents.value.map((event) => event.id === nextEvent.id ? nextEvent : event)
    eventDrafts.value = { ...eventDrafts.value, [nextEvent.id]: toEventDraft(nextEvent) }
    pruneSelectedCandidateIds()
  }

  function syncEventDrafts(events: ExtractedEvent[]) {
    const drafts = { ...eventDrafts.value }
    events.forEach((event) => {
      drafts[event.id] = toEventDraft(event)
    })
    eventDrafts.value = drafts
  }

  function resetExtractedEvents() {
    extractedEvents.value = []
    eventDrafts.value = {}
    selectedCandidateIds.value = []
    candidateMessage.value = ''
    clearUndoCandidateAction()
    selectedCandidateLetterId.value = ''
  }

  return {
    allSelectableCandidatesSelected,
    bulkCandidateAction,
    bulkConfirmExtractedEvents,
    bulkIgnoreExtractedEvents,
    bulkRegisterExtractedEvents,
    canBulkRegisterSelectedEvents,
    canEditExtractedEvent,
    canRegisterExtractedEvent,
    canSelectExtractedEvent,
    candidateMessage,
    eventDrafts,
    extractedEvents,
    extractedStatusLabel,
    hasUnregisterableSelectedEvents,
    ignoreExtractedEvent,
    loadExtractedEvents,
    mergeExtractedEvents,
    pendingCandidateCount,
    readyCandidateCount,
    registerExtractedEvent,
    registeringCandidateId,
    resetExtractedEvents,
    restoreIgnoredExtractedEvent,
    saveExtractedEvent,
    savingCandidateId,
    selectableExtractedEvents,
    selectedCandidateIds,
    selectedCandidateLetterId,
    toggleAllSelectableCandidates,
    undoCandidateAction,
    undoLastCandidateAction,
  }
}

function createUndoRestore(id: string, expectedStatus: string, status: string): UndoStatusRestore | null {
  const safeStatuses = ['draft', 'confirmed', 'ignored']
  if (!safeStatuses.includes(expectedStatus) || !safeStatuses.includes(status)) return null
  return { id, expected_status: expectedStatus, status }
}

function toEventDraft(event: ExtractedEvent): ExtractedEventDraft {
  return {
    title: event.title,
    event_date: toDateInput(event.event_date),
    start_time: event.start_time?.slice(0, 5) ?? '',
    end_time: event.end_time?.slice(0, 5) ?? '',
    is_all_day: event.is_all_day,
    location: event.location ?? '',
    description: event.description ?? '',
    belongings: event.belongings ?? '',
    submission_deadline: event.submission_deadline?.slice(0, 10) ?? '',
  }
}

function toDateInput(value: string) {
  if (/^\d{4}-\d{2}-\d{2}/.test(value)) return value.slice(0, 10)
  return new Date(value).toISOString().slice(0, 10)
}

function extractedStatusLabel(status: string) {
  const labels: Record<string, string> = {
    draft: '未確認',
    confirmed: '確認済み',
    ignored: '除外済み',
    registered: '登録済み',
    failed: '登録失敗',
    deleted: '削除済み',
  }
  return labels[status] ?? status
}

function canRegisterExtractedEvent(event: ExtractedEvent) {
  return ['confirmed', 'failed', 'deleted'].includes(event.status)
}

function canEditExtractedEvent(event: ExtractedEvent) {
  return event.status !== 'registered' && event.status !== 'ignored'
}

function canSelectExtractedEvent(event: ExtractedEvent) {
  return event.status !== 'registered' && event.status !== 'ignored'
}

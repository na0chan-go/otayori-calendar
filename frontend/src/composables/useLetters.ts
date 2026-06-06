import { ref, type Ref } from 'vue'
import type { ExtractedEvent, Letter } from '../types'
import { apiBaseUrl } from './api'

type LetterDependencies = {
  candidateCountForLetter: (letterId: string) => number
  mergeExtractedEvents: (events: ExtractedEvent[]) => void
  refreshRelatedData: () => Promise<void>
}

export function useLetters(errorMessage: Ref<string>, dependencies: LetterDependencies) {
  const deletingLetterId = ref('')
  const extractingLetterId = ref('')
  const letterImage = ref<File | null>(null)
  const letterMessage = ref('')
  const letterTitle = ref('')
  const letters = ref<Letter[]>([])
  const ocrTextByLetter = ref<Record<string, string>>({})
  const uploadingLetter = ref(false)

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
    if (
      dependencies.candidateCountForLetter(letter.id) > 0 &&
      !window.confirm('このおたよりには予定候補があります。再抽出すると候補が追加される可能性があります。続けますか？')
    ) {
      return
    }
    const ocrText = ocrTextByLetter.value[letter.id]?.trim()
    extractingLetterId.value = letter.id
    letterMessage.value = ''
    errorMessage.value = ''

    try {
      const response = await fetch(`${apiBaseUrl}/api/letters/${letter.id}/extract-events`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ocr_text: ocrText }),
      })
      if (!response.ok) {
        const body = (await response.json().catch(() => null)) as { message?: string } | null
        throw new Error(body?.message ?? '予定候補を抽出できませんでした')
      }

      const body = (await response.json()) as { events: ExtractedEvent[] }
      dependencies.mergeExtractedEvents(body.events)
      letterMessage.value = 'AIで予定候補を抽出し、draftとして保存しました'
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : '予定候補の抽出でエラーが発生しました'
    } finally {
      extractingLetterId.value = ''
    }
  }

  async function deleteLetter(letter: Letter) {
    const title = letter.title || '無題のおたより'
    const confirmed = window.confirm(
      `「${title}」を削除しますか？\n\n画像と紐づく予定候補は削除されます。Googleカレンダーへ登録済みの予定は削除されません。`,
    )
    if (!confirmed) return

    deletingLetterId.value = letter.id
    letterMessage.value = ''
    errorMessage.value = ''

    try {
      const response = await fetch(`${apiBaseUrl}/api/letters/${letter.id}`, {
        method: 'DELETE',
        credentials: 'include',
      })
      if (!response.ok) {
        const body = (await response.json().catch(() => null)) as { message?: string } | null
        throw new Error(body?.message ?? 'おたよりを削除できませんでした')
      }

      if (letter.object_url) URL.revokeObjectURL(letter.object_url)
      delete ocrTextByLetter.value[letter.id]
      letterMessage.value = 'おたより画像と紐づく予定候補を削除しました'
      await Promise.all([loadLetters(), dependencies.refreshRelatedData()])
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : 'おたよりの削除でエラーが発生しました'
    } finally {
      deletingLetterId.value = ''
    }
  }

  async function loadLetters() {
    const response = await fetch(`${apiBaseUrl}/api/letters`, { credentials: 'include' })
    if (!response.ok) throw new Error('おたより一覧を取得できませんでした')

    const body = (await response.json()) as { letters: Letter[] }
    clearLetterObjectUrls()
    letters.value = await Promise.all(
      body.letters.map(async (letter) => ({ ...letter, object_url: await loadLetterImage(letter.image_url) })),
    )
  }

  async function loadLetterImage(imageUrl: string) {
    const response = await fetch(`${apiBaseUrl}${imageUrl}`, { credentials: 'include' })
    if (!response.ok) throw new Error('おたより画像を取得できませんでした')
    return URL.createObjectURL(await response.blob())
  }

  function clearLetterObjectUrls() {
    letters.value.forEach((letter) => {
      if (letter.object_url) URL.revokeObjectURL(letter.object_url)
    })
  }

  function resetLetters() {
    clearLetterObjectUrls()
    letters.value = []
    ocrTextByLetter.value = {}
    letterMessage.value = ''
  }

  return {
    deleteLetter,
    deletingLetterId,
    extractEvents,
    extractingLetterId,
    letterImage,
    letterMessage,
    letterTitle,
    letters,
    loadLetters,
    ocrTextByLetter,
    onLetterImageChange,
    resetLetters,
    uploadingLetter,
    uploadLetter,
  }
}

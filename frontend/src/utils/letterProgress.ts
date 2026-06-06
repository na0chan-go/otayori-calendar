import type { ExtractedEvent, LetterProgress } from '../types'

export function buildLetterProgress(events: ExtractedEvent[]): LetterProgress {
  const counts = {
    draft: events.filter((event) => event.status === 'draft').length,
    confirmed: events.filter((event) => event.status === 'confirmed').length,
    registered: events.filter((event) => event.status === 'registered').length,
    ignored: events.filter((event) => event.status === 'ignored').length,
    attention: events.filter((event) => event.status === 'failed' || event.status === 'deleted').length,
  }

  if (events.length === 0) return { label: '未抽出', tone: 'neutral', total: 0, counts }
  if (counts.attention > 0) return { label: '要対応', tone: 'attention', total: events.length, counts }
  if (counts.draft > 0) return { label: '確認待ち', tone: 'waiting', total: events.length, counts }
  if (counts.confirmed > 0) return { label: '登録準備完了', tone: 'ready', total: events.length, counts }
  return { label: '完了', tone: 'complete', total: events.length, counts }
}

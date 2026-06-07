export function toUserErrorMessage(error: unknown, fallback: string) {
  if (!navigator.onLine) return 'オフラインです。接続が戻ったら最新状態を確認してください'
  if (error instanceof TypeError) return '通信に失敗しました。最新状態を確認してから、必要な操作だけ再実行してください'
  return error instanceof Error ? error.message : fallback
}

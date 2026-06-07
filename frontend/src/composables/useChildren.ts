import { ref, type Ref } from 'vue'
import type { Child } from '../types'
import { toUserErrorMessage } from '../utils/requestError'
import { apiBaseUrl } from './api'

export function useChildren(errorMessage: Ref<string>) {
  const children = ref<Child[]>([])
  const childDrafts = ref<Record<string, { name: string; color: string }>>({})
  const childMessage = ref('')
  const newChild = ref({ name: '', color: '#8fcfb0' })
  const savingChildId = ref('')

  async function loadChildren() {
    const response = await fetch(`${apiBaseUrl}/api/children`, { credentials: 'include' })
    if (!response.ok) throw new Error('子ども一覧を取得できませんでした')
    const body = (await response.json()) as { children: Child[] }
    children.value = body.children
    childDrafts.value = Object.fromEntries(body.children.map((child) => [child.id, { name: child.name, color: child.color }]))
  }

  async function createChild() {
    savingChildId.value = 'new'
    childMessage.value = ''
    errorMessage.value = ''
    try {
      const response = await fetch(`${apiBaseUrl}/api/children`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newChild.value),
      })
      if (!response.ok) {
        const body = (await response.json().catch(() => null)) as { message?: string } | null
        throw new Error(body?.message ?? '子どもを登録できませんでした')
      }
      newChild.value = { name: '', color: '#8fcfb0' }
      childMessage.value = '子どもを登録しました'
      await loadChildren()
    } catch (error) {
      errorMessage.value = toUserErrorMessage(error, '子どもの登録でエラーが発生しました')
    } finally {
      savingChildId.value = ''
    }
  }

  async function saveChild(child: Child) {
    savingChildId.value = child.id
    childMessage.value = ''
    errorMessage.value = ''
    try {
      const response = await fetch(`${apiBaseUrl}/api/children/${child.id}`, {
        method: 'PATCH',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(childDrafts.value[child.id]),
      })
      if (!response.ok) {
        const body = (await response.json().catch(() => null)) as { message?: string } | null
        throw new Error(body?.message ?? '子どもの情報を保存できませんでした')
      }
      childMessage.value = '子どもの情報を保存しました'
      await loadChildren()
    } catch (error) {
      errorMessage.value = toUserErrorMessage(error, '子どもの更新でエラーが発生しました')
    } finally {
      savingChildId.value = ''
    }
  }

  function resetChildren() {
    children.value = []
    childDrafts.value = {}
    childMessage.value = ''
  }

  return { childDrafts, childMessage, children, createChild, loadChildren, newChild, resetChildren, saveChild, savingChildId }
}

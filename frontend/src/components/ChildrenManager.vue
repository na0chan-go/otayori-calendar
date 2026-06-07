<script setup lang="ts">
import { useOtayoriCalendarContext } from '../composables/otayoriCalendarContext'

const { childDrafts, childMessage, children, createChild, newChild, saveChild, savingChildId } = useOtayoriCalendarContext()
</script>

<template>
  <section class="children-manager">
    <div class="section-heading">
      <div><p class="section-kicker">Family</p><h2>子どもの設定</h2></div>
      <p>兄弟姉妹の予定を色で見分けられます。設定しなくても利用できます。</p>
    </div>
    <form class="surface child-form" @submit.prevent="createChild">
      <label>表示名<input v-model="newChild.name" required type="text" placeholder="例：なお" /></label>
      <label>識別色<input v-model="newChild.color" type="color" /></label>
      <button class="primary-button" :disabled="savingChildId !== '' || !newChild.name.trim()" type="submit">子どもを追加</button>
    </form>
    <div class="child-list">
      <form v-for="child in children" :key="child.id" class="surface child-form" @submit.prevent="saveChild(child)">
        <span class="child-color" :style="{ background: childDrafts[child.id].color }"></span>
        <label>表示名<input v-model="childDrafts[child.id].name" required type="text" /></label>
        <label>識別色<input v-model="childDrafts[child.id].color" type="color" /></label>
        <button class="secondary-button" :disabled="savingChildId !== ''" type="submit">{{ savingChildId === child.id ? '保存中...' : '保存' }}</button>
      </form>
    </div>
    <p v-if="childMessage" class="notice success-notice">{{ childMessage }}</p>
  </section>
</template>

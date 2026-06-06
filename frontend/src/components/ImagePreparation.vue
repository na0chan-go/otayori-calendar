<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps<{ file: File }>()
const emit = defineEmits<{ change: [file: File] }>()

const previewUrl = ref('')
const trimming = ref({ top: 0, right: 0, bottom: 0, left: 0 })
let originalFile = props.file
let internalFile: File | null = null

watch(
  () => props.file,
  (file) => {
    if (file !== internalFile) originalFile = file
    internalFile = null
    updatePreview(file)
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
})

function updatePreview(file: File) {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  previewUrl.value = URL.createObjectURL(file)
}

async function rotateImage() {
  await transformImage(props.file, (source, canvas, context) => {
    canvas.width = source.height
    canvas.height = source.width
    context.translate(canvas.width / 2, canvas.height / 2)
    context.rotate(Math.PI / 2)
    context.drawImage(source, -source.width / 2, -source.height / 2)
  })
}

async function cropImage() {
  const { top, right, bottom, left } = trimming.value
  await transformImage(props.file, (source, canvas, context) => {
    const sourceX = source.width * (left / 100)
    const sourceY = source.height * (top / 100)
    const sourceWidth = source.width * ((100 - left - right) / 100)
    const sourceHeight = source.height * ((100 - top - bottom) / 100)
    canvas.width = Math.max(1, Math.round(sourceWidth))
    canvas.height = Math.max(1, Math.round(sourceHeight))
    context.drawImage(source, sourceX, sourceY, sourceWidth, sourceHeight, 0, 0, canvas.width, canvas.height)
  })
  trimming.value = { top: 0, right: 0, bottom: 0, left: 0 }
}

function resetImage() {
  trimming.value = { top: 0, right: 0, bottom: 0, left: 0 }
  emitFile(originalFile)
}

async function transformImage(file: File, draw: (source: HTMLImageElement, canvas: HTMLCanvasElement, context: CanvasRenderingContext2D) => void) {
  const source = await loadImage(file)
  const canvas = document.createElement('canvas')
  const context = canvas.getContext('2d')
  if (!context) return
  draw(source, canvas, context)
  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.92))
  if (!blob) return
  emitFile(new File([blob], replaceExtension(file.name, 'jpg'), { type: 'image/jpeg' }))
}

function emitFile(file: File) {
  internalFile = file
  emit('change', file)
}

function loadImage(file: File) {
  return new Promise<HTMLImageElement>((resolve, reject) => {
    const image = new Image()
    const url = URL.createObjectURL(file)
    image.onload = () => {
      URL.revokeObjectURL(url)
      resolve(image)
    }
    image.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('画像を読み込めませんでした'))
    }
    image.src = url
  })
}

function replaceExtension(name: string, extension: string) {
  return `${name.replace(/\.[^.]+$/, '')}.${extension}`
}
</script>

<template>
  <section class="image-preparation">
    <div class="image-preview-frame">
      <img :src="previewUrl" alt="アップロード前のおたよりプレビュー" />
    </div>
    <div class="image-preparation-actions">
      <button class="secondary-button" type="button" @click="rotateImage">右へ90度回転</button>
      <button class="text-button" type="button" @click="resetImage">元の画像に戻す</button>
    </div>
    <details class="crop-controls">
      <summary>余白を切り抜く</summary>
      <p class="helper-text">各辺から取り除く量を調整してから適用してください。</p>
      <div class="crop-slider-grid">
        <label v-for="side in ['top', 'right', 'bottom', 'left'] as const" :key="side">
          {{ { top: '上', right: '右', bottom: '下', left: '左' }[side] }} {{ trimming[side] }}%
          <input v-model.number="trimming[side]" max="40" min="0" step="1" type="range" />
        </label>
      </div>
      <button
        class="secondary-button"
        :disabled="trimming.top + trimming.bottom >= 90 || trimming.left + trimming.right >= 90"
        type="button"
        @click="cropImage"
      >
        切り抜きを適用
      </button>
    </details>
  </section>
</template>

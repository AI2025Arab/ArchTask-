<!-- ArchTask Addition - DO NOT MERGE INTO VIKUNJA CORE -->
<!-- BUG-03 Fixed: Uses Pinia auth store instead of localStorage for JWT token. -->
<!-- BUG-04 Fixed: Task creation is now done server-side via /api/v1/archtask/* endpoints. -->
<!-- BUG-05 Fixed: API paths updated from /api/v1/ai/* → /api/v1/archtask/input/*. -->
<template>
  <div class="ai-capture-panel">
    <!-- Usage limit banner -->
    <div v-if="usageStats" class="usage-banner" :class="{ 'usage-low': usageStats.remaining_free <= 10 }">
      <span>🤖 المهام المجانية المتبقية هذا الشهر: <strong>{{ usageStats.remaining_free }}</strong> / {{ freeLimit }}</span>
    </div>

    <div v-if="!isProcessing" class="capture-buttons">
      <h2>أضف مهامك الآن</h2>
      <div class="actions-row">
        <button
          class="ai-btn voice-btn"
          @click="toggleRecording"
          :class="{ recording: isRecording }"
          :disabled="!projectId"
          :title="projectId ? '' : 'يجب تحديد مشروع أولاً'"
        >
          <span v-if="!isRecording">🎤 تسجيل صوتي ({{ countdown }} ث)</span>
          <span v-else>🔴 جاري التسجيل... ({{ countdown }} ث) — اضغط للإيقاف</span>
        </button>

        <button class="ai-btn text-btn" @click="showTextInput = !showTextInput">
          📝 {{ showTextInput ? 'إخفاء' : 'كتابة نص' }}
        </button>

        <label class="ai-btn image-btn" for="image-upload-archtask">
          🖼️ رفع صورة/مخطط
          <input
            id="image-upload-archtask"
            type="file"
            accept="image/png, image/jpeg, application/pdf"
            @change="onImageUpload"
            style="display: none;"
          />
        </label>
      </div>

      <div v-if="showTextInput" class="text-input-row">
        <textarea
          v-model="textPrompt"
          placeholder="اكتب مهامك أو البريف هنا بالعربي أو الإنجليزي..."
          rows="4"
        ></textarea>
        <div class="text-actions">
          <button class="submit-btn" @click="sendText" :disabled="!textPrompt.trim() || !projectId">
            إرسال للذكاء الاصطناعي
          </button>
          <button class="cancel-btn" @click="showTextInput = false; textPrompt = ''">إلغاء</button>
        </div>
      </div>
    </div>

    <div v-else class="processing-panel">
      <div class="spinner"></div>
      <span>⚙️ الذكاء الاصطناعي يولّد مهامك المعمارية...</span>
    </div>

    <!-- Result Toast -->
    <transition name="fade">
      <div v-if="toast.show" class="toast" :class="toast.type">
        {{ toast.message }}
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onBeforeUnmount, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getToken } from '@/helpers/auth'

const route = useRoute()

const emit = defineEmits<{
  (e: 'tasks-generated', tasks: unknown[]): void
}>()

// ─── State ──────────────────────────────────────────────────────────────
const isRecording = ref(false)
const isProcessing = ref(false)
const showTextInput = ref(false)
const textPrompt = ref('')
const countdown = ref(30)
const freeLimit = 50

interface UsageStats {
  remaining_free: number
  total_this_month: number
  can_use_ai: boolean
}
const usageStats = ref<UsageStats | null>(null)

const toast = ref({ show: false, message: '', type: 'success' as 'success' | 'error' })

let mediaRecorder: MediaRecorder | null = null
let audioChunks: Blob[] = []
let timer: ReturnType<typeof setInterval> | null = null
let toastTimer: ReturnType<typeof setTimeout> | null = null

// ─── Computed ────────────────────────────────────────────────────────────
const projectId = computed<string | null>(() => {
  const p = route.params.projectId ?? route.params.id ?? route.query.projectId
  return p ? String(p) : null
})

// BUG-03 Fixed: Use the canonical getToken() helper instead of localStorage directly
const authToken = computed<string>(() => getToken() ?? '')


// ─── Helpers ─────────────────────────────────────────────────────────────
const showToast = (message: string, type: 'success' | 'error' = 'success') => {
  if (toastTimer) clearTimeout(toastTimer)
  toast.value = { show: true, message, type }
  toastTimer = setTimeout(() => { toast.value.show = false }, 4000)
}

const apiHeaders = computed(() => ({
  'Content-Type': 'application/json',
  'Authorization': `Bearer ${authToken.value}`,
}))

// ─── Usage Stats ──────────────────────────────────────────────────────────
const fetchUsageStats = async () => {
  try {
    const res = await fetch('/api/v1/archtask/usage', {
      headers: { Authorization: `Bearer ${authToken.value}` },
    })
    if (res.ok) usageStats.value = await res.json()
  } catch (e) {
    console.warn('[ArchTask] Could not fetch usage stats:', e)
  }
}

// ─── Voice Recording ──────────────────────────────────────────────────────
const toggleRecording = () => {
  if (isRecording.value) {
    stopRecording()
  } else {
    startRecording()
  }
}

const startRecording = async () => {
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    const mimeType = MediaRecorder.isTypeSupported('audio/mp4') ? 'audio/mp4' : 'audio/webm'
    mediaRecorder = new MediaRecorder(stream, { mimeType })

    mediaRecorder.ondataavailable = (event) => {
      if (event.data.size > 0) audioChunks.push(event.data)
    }

    mediaRecorder.onstop = async () => {
      const audioBlob = new Blob(audioChunks, { type: mimeType })
      await processAudio(audioBlob)
      stream.getTracks().forEach((track) => track.stop())
    }

    audioChunks = []
    mediaRecorder.start()
    isRecording.value = true
    countdown.value = 30

    timer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) stopRecording()
    }, 1000)
  } catch (error) {
    showToast('حدث خطأ في الوصول للمايكروفون. تأكد من منح الإذن.', 'error')
  }
}

const stopRecording = () => {
  if (mediaRecorder && mediaRecorder.state !== 'inactive') {
    mediaRecorder.stop()
  }
  if (timer) clearInterval(timer)
  isRecording.value = false
}

// BUG-04 & BUG-05 Fixed: Send audio to /api/v1/archtask/input/voice (server-side processing)
const processAudio = async (blob: Blob) => {
  if (!projectId.value) {
    showToast('يجب فتح مشروع أولاً قبل التسجيل الصوتي.', 'error')
    return
  }
  isProcessing.value = true
  const formData = new FormData()
  formData.append('audio', blob, 'recording.webm')
  formData.append('project_id', projectId.value)

  try {
    const res = await fetch('/api/v1/archtask/input/voice', {
      method: 'POST',
      headers: { Authorization: `Bearer ${authToken.value}` },
      body: formData,
    })
    await handleApiResponse(res)
  } catch (e) {
    showToast('عذراً، الخدمة مشغولة. جرب مجدداً.', 'error')
  } finally {
    isProcessing.value = false
  }
}

// BUG-04 & BUG-05 Fixed: Send text to /api/v1/archtask/input/text (server-side)
const sendText = async () => {
  if (!projectId.value || !textPrompt.value.trim()) return
  isProcessing.value = true
  try {
    const res = await fetch('/api/v1/archtask/input/text', {
      method: 'POST',
      headers: apiHeaders.value,
      body: JSON.stringify({
        text: textPrompt.value,
        project_id: parseInt(projectId.value),
      }),
    })
    await handleApiResponse(res)
    textPrompt.value = ''
    showTextInput.value = false
  } catch (e) {
    showToast('عذراً، الخدمة مشغولة.', 'error')
  } finally {
    isProcessing.value = false
  }
}

// BUG-04 & BUG-05 Fixed: Send image to /api/v1/archtask/input/image (server-side)
const onImageUpload = async (event: Event) => {
  if (!projectId.value) {
    showToast('يجب فتح مشروع أولاً قبل رفع الصورة.', 'error')
    return
  }
  const target = event.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return
  const file = target.files[0]

  if (file.size > 5 * 1024 * 1024) {
    showToast('حجم الملف يجب ألا يتجاوز 5 ميجا بايت', 'error')
    return
  }

  isProcessing.value = true

  try {
    let base64String = ''
    let finalMimeType = file.type

    if (file.type === 'application/pdf') {
      // PDF to image via PDF.js
      const pdfjsLib = (window as unknown as Record<string, unknown>).pdfjsLib as Record<string, (...args: unknown[]) => unknown> | undefined
      if (pdfjsLib) {
        const arrayBuffer = await file.arrayBuffer()
        const pdf = await (pdfjsLib.getDocument(arrayBuffer) as Promise<unknown> as Promise<{ getPage: (n: number) => Promise<{ getViewport: (o: object) => { height: number; width: number }; render: (o: object) => { promise: Promise<void> } }> }>).catch(() => null)
        if (pdf) {
          const page = await pdf.getPage(1)
          const viewport = page.getViewport({ scale: 1.5 })
          const canvas = document.createElement('canvas')
          canvas.height = viewport.height
          canvas.width = viewport.width
          await page.render({ canvasContext: canvas.getContext('2d'), viewport }).promise
          base64String = canvas.toDataURL('image/jpeg', 0.8).split(',')[1]
          finalMimeType = 'image/jpeg'
        }
      }
    } else {
      base64String = await new Promise<string>((resolve) => {
        const reader = new FileReader()
        reader.onload = () => resolve((reader.result as string).split(',')[1])
        reader.readAsDataURL(file)
      })
    }

    if (!base64String) {
      showToast('لم يمكن قراءة الملف. جرب صيغة PNG أو JPEG.', 'error')
      return
    }

    const res = await fetch('/api/v1/archtask/input/image', {
      method: 'POST',
      headers: apiHeaders.value,
      body: JSON.stringify({
        image: base64String,
        mimeType: finalMimeType,
        project_id: parseInt(projectId.value as string),
      }),
    })
    await handleApiResponse(res)
  } catch (e) {
    showToast('خطأ في معالجة الملف/الصورة.', 'error')
  } finally {
    isProcessing.value = false
    // Reset file input
    const input = document.getElementById('image-upload-archtask') as HTMLInputElement
    if (input) input.value = ''
  }
}

// ─── Unified response handler ─────────────────────────────────────────────
const handleApiResponse = async (res: Response) => {
  if (res.status === 402) {
    showToast('انتهت المهام المجانية لهذا الشهر. يرجى الترقية.', 'error')
    return
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'خطأ غير معروف' }))
    showToast(err.error ?? 'حدث خطأ في الخدمة.', 'error')
    return
  }

  const data = await res.json()
  const created = data.created ?? 0
  const remaining = data.remaining_free_ops ?? '—'

  showToast(`✅ تم إنشاء ${created} مهمة معمارية! (متبقي: ${remaining} عملية مجانية)`, 'success')
  emit('tasks-generated', data.tasks ?? [])
  await fetchUsageStats()
}

// ─── Lifecycle ────────────────────────────────────────────────────────────
onMounted(fetchUsageStats)

onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
  if (toastTimer) clearTimeout(toastTimer)
  if (mediaRecorder && mediaRecorder.state !== 'inactive') mediaRecorder.stop()
})
</script>

<style scoped>
.ai-capture-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 28px 30px;
  background: linear-gradient(135deg, #1e1e2d 0%, #2d2b42 100%);
  color: white;
  border-radius: 16px;
  margin-bottom: 20px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
  font-family: inherit;
  gap: 16px;
  position: relative;
}

.usage-banner {
  align-self: stretch;
  text-align: center;
  font-size: 0.85rem;
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  color: #aaa;
}
.usage-banner.usage-low {
  background: rgba(255, 65, 108, 0.15);
  color: #ff8fa3;
}

.capture-buttons {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  width: 100%;
}
.capture-buttons h2 {
  font-size: 1.4rem;
  font-weight: 700;
  margin: 0;
}

.actions-row {
  display: flex;
  gap: 12px;
  justify-content: center;
  flex-wrap: wrap;
}

.ai-btn {
  padding: 14px 22px;
  border: none;
  border-radius: 12px;
  cursor: pointer;
  font-size: 0.95rem;
  font-weight: 600;
  transition: all 0.3s ease;
  background: rgba(255, 255, 255, 0.1);
  color: white;
  display: flex;
  align-items: center;
  gap: 8px;
}
.ai-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.ai-btn:not(:disabled):hover {
  background: rgba(255, 255, 255, 0.2);
  transform: translateY(-2px);
}

.voice-btn { background: linear-gradient(135deg, #ff416c, #ff4b2b); }
.voice-btn.recording { animation: pulse 1.5s infinite; }
.text-btn { background: linear-gradient(135deg, #4facfe, #00f2fe); }
.image-btn { background: linear-gradient(135deg, #a18cd1, #fbc2eb); }

@keyframes pulse {
  0%   { box-shadow: 0 0 0 0 rgba(255, 65, 108, 0.7); }
  70%  { box-shadow: 0 0 0 15px rgba(255, 65, 108, 0); }
  100% { box-shadow: 0 0 0 0 rgba(255, 65, 108, 0); }
}

.text-input-row {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
}
textarea {
  width: 100%;
  padding: 14px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.15);
  background: rgba(255, 255, 255, 0.06);
  color: white;
  resize: vertical;
  font-family: inherit;
  font-size: 0.95rem;
  line-height: 1.6;
}
textarea::placeholder { color: rgba(255,255,255,0.4); }

.text-actions {
  display: flex;
  gap: 10px;
}
.submit-btn {
  flex: 1;
  padding: 12px;
  background: linear-gradient(135deg, #43e97b, #38f9d7);
  color: #1a1a2e;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  font-weight: 700;
  font-size: 0.95rem;
  transition: opacity 0.2s;
}
.submit-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.cancel-btn {
  padding: 12px 20px;
  background: rgba(255,255,255,0.1);
  color: white;
  border: none;
  border-radius: 10px;
  cursor: pointer;
}

.processing-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  font-size: 1.1rem;
  font-weight: 600;
  padding: 20px 0;
}
.spinner {
  width: 40px;
  height: 40px;
  border: 4px solid rgba(255, 255, 255, 0.2);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.toast {
  position: fixed;
  bottom: 30px;
  left: 50%;
  transform: translateX(-50%);
  padding: 14px 24px;
  border-radius: 12px;
  font-weight: 600;
  z-index: 9999;
  max-width: 400px;
  text-align: center;
  box-shadow: 0 8px 24px rgba(0,0,0,0.3);
}
.toast.success { background: linear-gradient(135deg, #43e97b, #38f9d7); color: #1a1a2e; }
.toast.error   { background: linear-gradient(135deg, #ff416c, #ff4b2b); color: white; }

.fade-enter-active, .fade-leave-active { transition: opacity 0.5s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>

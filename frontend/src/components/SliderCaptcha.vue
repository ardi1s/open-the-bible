<template>
  <div class="captcha-container">
    <div class="slider-track" ref="track">
      <div class="slider-bg" :style="{ width: position + '%' }"></div>
      <div class="slider-btn" :style="{ left: position + '%' }"
        @mousedown="startDrag" @touchstart.prevent="startDrag">
        ➤
      </div>
    </div>
    <p class="captcha-text">{{ verified ? '✅ 验证通过' : '→ 请按住滑块拖动到最右侧' }}</p>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const emit = defineEmits(['verify'])
const verified = ref(false)
const position = ref(0)
const track = ref(null)

function startDrag(e) {
  if (verified.value) return
  const clientX = e.touches ? e.touches[0].clientX : e.clientX
  const trackWidth = track.value.offsetWidth
  const btnWidth = 40

  function onMove(ev) {
    const x = (ev.touches ? ev.touches[0].clientX : ev.clientX) - clientX
    position.value = Math.max(0, Math.min(100, (x / (trackWidth - btnWidth)) * 100))
  }

  function onEnd() {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onEnd)
    document.removeEventListener('touchmove', onMove)
    document.removeEventListener('touchend', onEnd)
    if (position.value > 85) {
      position.value = 100
      verified.value = true
      emit('verify')
    } else {
      position.value = 0
    }
  }

  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onEnd)
  document.addEventListener('touchmove', onMove, { passive: false })
  document.addEventListener('touchend', onEnd)
}
</script>

<style scoped>
.captcha-container { margin: 12px 0; }
.slider-track { position: relative; height: 40px; background: #eee; border-radius: 20px; overflow: hidden; }
.slider-bg { position: absolute; height: 100%; background: linear-gradient(90deg, #a0d911, #52c41a); border-radius: 20px 0 0 20px; transition: width 0.1s; }
.slider-btn { position: absolute; top: 0; width: 40px; height: 40px; background: #fff; border: 2px solid #ddd; border-radius: 50%; display: flex; align-items: center; justify-content: center; cursor: grab; font-size: 16px; transform: translateX(-20px); user-select: none; touch-action: none; z-index: 2; }
.captcha-text { text-align: center; font-size: 12px; color: #999; margin-top: 6px; }
</style>

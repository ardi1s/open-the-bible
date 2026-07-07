<template>
  <div class="auth-page">
    <div class="auth-card">
      <h1 class="brand">📝 注册</h1>
      <p class="slogan">注册后需等待管理员审核通过</p>

      <el-input v-model="phone" placeholder="请输入手机号" maxlength="11" size="large" class="input" />
      <div class="sms-row">
        <el-input v-model="smsCode" placeholder="验证码" size="large" style="flex:1" />
        <el-button :disabled="countdown > 0" @click="sendSMS" size="large" class="sms-btn">
          {{ countdown > 0 ? countdown + 's' : '获取验证码' }}
        </el-button>
      </div>
      <SliderCaptcha v-if="showCaptcha" @verify="onCaptchaVerified" />
      <el-input v-model="inviteCode" placeholder="邀请码（选填）" size="large" class="input" />

      <el-button class="submit-btn" size="large" @click="handleRegister" :loading="loading">注册</el-button>
      <p class="back-link"><router-link to="/login">← 返回登录</router-link></p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'
import { ElMessage } from 'element-plus'
import SliderCaptcha from '../components/SliderCaptcha.vue'

const router = useRouter()
const userStore = useUserStore()
const phone = ref(''); const smsCode = ref(''); const inviteCode = ref('')
const countdown = ref(0); const showCaptcha = ref(false); const loading = ref(false)

function sendSMS() {
  if (!/^1\d{10}$/.test(phone.value)) return ElMessage.warning('请输入正确的手机号')
  showCaptcha.value = true
}
function onCaptchaVerified() {
  countdown.value = 60
  const t = setInterval(() => { countdown.value--; if (countdown.value <= 0) clearInterval(t) }, 1000)
  showCaptcha.value = false
  ElMessage.success('验证码已发送（Mock：输入 1234）')
}
async function handleRegister() {
  if (!phone.value || !smsCode.value) return ElMessage.warning('请输入手机号和验证码')
  loading.value = true
  try {
    await userStore.doRegister(phone.value, smsCode.value, inviteCode.value)
    ElMessage.success('注册成功！请等待管理员审核')
    router.push('/login')
  } catch (e) { ElMessage.error(e.message || '注册失败') }
  loading.value = false
}
</script>

<style scoped>
.auth-page { display: flex; align-items: center; justify-content: center; min-height: 100vh; padding: 20px; background: #fff; }
.auth-card { width: 100%; max-width: 360px; }
.brand { text-align: center; font-size: 28px; font-weight: 800; margin-bottom: 4px; }
.slogan { text-align: center; font-size: 13px; color: var(--text3); margin-bottom: 28px; }
.input { margin-bottom: 12px; }
.sms-row { display: flex; gap: 10px; margin-bottom: 16px; }
.sms-btn { white-space: nowrap; border-radius: 8px; }
.submit-btn { width: 100%; height: 46px; font-size: 16px; background: var(--red); border-color: var(--red); color: #fff; border-radius: 8px; margin-top: 8px; }
.back-link { text-align: center; margin-top: 20px; font-size: 13px; }
.back-link a { color: var(--red); }
</style>

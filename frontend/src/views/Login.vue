<template>
  <div class="auth-page">
    <div class="auth-card">
      <h1 class="brand">小 X</h1>
      <p class="slogan">标记我的生活</p>
      <el-input v-model="phone" placeholder="手机号" maxlength="11" size="large" class="input" />
      <div class="sms-row">
        <el-input v-model="smsCode" placeholder="验证码" size="large" style="flex:1" />
        <el-button :disabled="countdown > 0" @click="sendSMS" size="large"> {{ countdown > 0 ? countdown + 's' : '获取验证码' }} </el-button>
      </div>
      <SliderCaptcha v-if="showCaptcha" @verify="onCaptchaVerified" />
      <el-button class="submit-btn" size="large" @click="handleLogin" :loading="loading">登录</el-button>
      <div class="links">
        <router-link to="/register">注册新账号</router-link>
        <router-link to="/admin/login" class="admin-entry">管理员入口</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'
import { ElMessage } from 'element-plus'
import SliderCaptcha from '../components/SliderCaptcha.vue'

const router = useRouter(); const userStore = useUserStore()
const phone = ref(''); const smsCode = ref(''); const countdown = ref(0); const showCaptcha = ref(false); const loading = ref(false)

function sendSMS() { if (!/^1\d{10}$/.test(phone.value)) return ElMessage.warning('请输入正确手机号'); showCaptcha.value = true }
function onCaptchaVerified() { countdown.value = 60; const t = setInterval(() => { countdown.value--; if (countdown.value <= 0) clearInterval(t) }, 1000); showCaptcha.value = false; ElMessage.success('验证码已发送，Mock：1234') }
async function handleLogin() {
  if (!phone.value || !smsCode.value) return ElMessage.warning('请输入手机号和验证码')
  loading.value = true
  try {
    await userStore.doLogin(phone.value, smsCode.value)
    ElMessage.success('登录成功')
    router.push(userStore.role === 'admin' ? '/admin/dashboard' : '/feed')
  } catch (e) { ElMessage.error(e.message || '登录失败') }
  loading.value = false
}
</script>

<style scoped>
.auth-page { display: flex; justify-content: center; min-height: 100vh; padding: 60px 20px; background: var(--bg); }
.auth-card { width: 100%; max-width: 380px; background: #fff; border-radius: 12px; padding: 40px 28px; box-shadow: 0 2px 12px rgba(0,0,0,.05); }
.brand { text-align: center; font-size: 36px; font-weight: 800; background: linear-gradient(135deg, var(--red), #ee5a24); -webkit-background-clip: text; -webkit-text-fill-color: transparent; margin-bottom: 4px; }
.slogan { text-align: center; font-size: 13px; color: var(--text3); margin-bottom: 28px; }
.input { margin-bottom: 12px; }
.sms-row { display: flex; gap: 10px; margin-bottom: 16px; }
.submit-btn { width: 100%; height: 46px; font-size: 16px; background: var(--red); border-color: var(--red); color: #fff; border-radius: 8px; margin-top: 8px; }
.submit-btn:hover { background: var(--red-light); }
.links { display: flex; justify-content: space-between; margin-top: 20px; font-size: 13px; }
.links a { color: var(--red); }
.admin-entry { color: #bbb !important; }
</style>

<template>
  <div class="admin-login-page">
    <div class="admin-card">
      <h1>🔐 管理员登录</h1>
      <p class="sub">请使用管理员账号登录后台</p>
      <el-input v-model="phone" placeholder="管理员账号" size="large" class="input" />
      <el-input v-model="smsCode" placeholder="密码" type="password" size="large" class="input" show-password />
      <el-button class="submit-btn" size="large" @click="handleLogin" :loading="loading">登录</el-button>
      <p class="back"><router-link to="/login">← 普通用户登录</router-link></p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../../stores/user'
import { ElMessage } from 'element-plus'

const router = useRouter()
const userStore = useUserStore()
const phone = ref('')
const smsCode = ref('')
const loading = ref(false)

async function handleLogin() {
  if (!phone.value || !smsCode.value) return ElMessage.warning('请输入账号和密码')
  loading.value = true
  try {
    // 管理员登录：admin / admin
    await userStore.doLogin(phone.value, smsCode.value)
    if (userStore.role !== 'admin') throw new Error('非管理员账号')
    ElMessage.success('登录成功')
    router.push('/admin/dashboard')
  } catch (e) {
    ElMessage.error(e.message || '登录失败')
  }
  loading.value = false
}
</script>

<style scoped>
.admin-login-page { display: flex; align-items: center; justify-content: center; min-height: 100vh; background: #f5f5f5; }
.admin-card { width: 360px; background: #fff; border-radius: 12px; padding: 36px 28px; box-shadow: 0 4px 24px rgba(0,0,0,.06); }
.admin-card h1 { text-align: center; font-size: 22px; margin-bottom: 6px; }
.sub { text-align: center; font-size: 13px; color: #999; margin-bottom: 28px; }
.input { margin-bottom: 14px; }
.submit-btn { width: 100%; height: 46px; font-size: 16px; background: #333; border-color: #333; color: #fff; border-radius: 8px; }
.submit-btn:hover { background: #555; border-color: #555; }
.back { text-align: center; margin-top: 18px; font-size: 13px; }
.back a { color: var(--red); }
</style>

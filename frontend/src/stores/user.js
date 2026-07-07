import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getUser } from '../api'

// 统一头像生成——同一个人始终同头像
export function avatarUrl(id) {
  return `https://api.dicebear.com/7.x/thumbs/svg?seed=u${id}`
}

function seedAdmin() {
  let users = JSON.parse(localStorage.getItem('mock_users') || '[]')
  if (!users.find(u => u.role === 'admin')) {
    users.unshift({ id: 10000, phone: 'admin', username: '管理员', status: 'approved', role: 'admin', createdAt: Date.now() })
    localStorage.setItem('mock_users', JSON.stringify(users))
  }
}
seedAdmin()

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const userId = ref(parseInt(localStorage.getItem('userId') || '1'))
  const role = ref(localStorage.getItem('role') || 'user')
  const profile = ref(null)

  const isLoggedIn = computed(() => !!token.value)

  async function fetchProfile(uid) {
    const mockUsers = JSON.parse(localStorage.getItem('mock_users') || '[]')
    const u = mockUsers.find(u => u.id === (uid || userId.value))
    if (u) profile.value = { id: u.id, username: u.username, avatar: avatarUrl(u.id) }
  }

  async function doLogin(phone, code) {
    if (phone === 'admin' && code === 'admin') {
      token.value = 'mock-token-admin'; userId.value = 10000; role.value = 'admin'
      localStorage.setItem('token', token.value); localStorage.setItem('userId', userId.value); localStorage.setItem('role', 'admin')
      profile.value = { id: 10000, username: '管理员', avatar: avatarUrl(10000) }
      return
    }
    const mockUsers = JSON.parse(localStorage.getItem('mock_users') || '[]')
    const user = mockUsers.find(u => u.phone === phone)
    if (!user || user.status !== 'approved') throw new Error(user ? (user.status === 'pending' ? '账户审核中' : '已被拒绝') : '用户不存在，请先注册')
    token.value = 'mock-token-' + phone; userId.value = user.id; role.value = user.role || 'user'
    localStorage.setItem('token', token.value); localStorage.setItem('userId', userId.value); localStorage.setItem('role', role.value)
    await fetchProfile(userId.value)
  }

  async function doRegister(phone, code, inviteCode) {
    const mockUsers = JSON.parse(localStorage.getItem('mock_users') || '[]')
    if (mockUsers.find(u => u.phone === phone)) throw new Error('该手机号已注册')
    mockUsers.push({ id: mockUsers.length + 1, phone, username: '用户' + phone.slice(-4), status: 'pending', role: 'user', createdAt: Date.now(), inviteCode })
    localStorage.setItem('mock_users', JSON.stringify(mockUsers))
  }

  function logout() {
    token.value = ''; userId.value = 0; profile.value = null
    localStorage.removeItem('token'); localStorage.removeItem('userId'); localStorage.removeItem('role')
  }

  return { token, userId, role, profile, isLoggedIn, fetchProfile, doLogin, doRegister, logout }
})

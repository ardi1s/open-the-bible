<template>
  <div class="profile-page">
    <div class="profile-inner">
      <div v-if="isMe" class="profile-card">
        <img class="av-big" :src="avatarUrl(userStore.userId)" />
        <h2>{{ profile?.username || '用户' + userStore.userId }}</h2>
        <p class="bio">{{ profile?.bio || '还没有个人简介 ✨' }}</p>
        <div class="stats">
          <div class="stat"><span class="num">{{ notes.length }}</span><span class="label">笔记</span></div>
          <div class="stat"><span class="num">0</span><span class="label">关注</span></div>
          <div class="stat"><span class="num">0</span><span class="label">粉丝</span></div>
        </div>
        <el-button class="logout-btn" @click="handleLogout" plain>退出登录</el-button>
      </div>

      <div v-else-if="viewUserId" class="profile-card">
        <img class="av-big" :src="avatarUrl(viewUserId)" />
        <h2>{{ authorName }}</h2>
        <p class="bio">TA 还没有个人简介 ✨</p>
        <div class="stats">
          <div class="stat"><span class="num">{{ notes.length }}</span><span class="label">笔记</span></div>
          <div class="stat"><span class="num">0</span><span class="label">关注</span></div>
          <div class="stat"><span class="num">0</span><span class="label">粉丝</span></div>
        </div>
        <div style="display:flex;gap:12px;justify-content:center;margin-top:8px">
          <el-button type="danger" size="small" round @click="doFollow" :loading="followLoading">{{ following ? '已关注' : '+ 关注' }}</el-button>
        </div>
      </div>

      <div v-else class="profile-login">
        <img class="av-big" :src="avatarUrl(0)" />
        <p>登录后查看个人主页</p>
        <el-button type="danger" round size="large" @click="$router.push('/login')">登录 / 注册</el-button>
      </div>

      <div class="notes-section" v-if="notes.length">
        <h3>📝 {{ isMe ? '我的笔记' : 'TA 的笔记' }} ({{ notes.length }})</h3>
        <div class="notes-grid">
          <NoteCard v-for="n in notes" :key="n.note_id || n.id" :note="n" />
        </div>
      </div>
      <div class="notes-section" v-else-if="viewUserId || isMe">
        <p style="text-align:center;color:#ccc;padding:40px">还没有发布过笔记</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useUserStore, avatarUrl } from '../stores/user'
import { getUserNotes, isFollowing, follow, unfollow } from '../api'
import NoteCard from '../components/NoteCard.vue'
import { ElMessage } from 'element-plus'

const route = useRoute(); const userStore = useUserStore()
const profile = computed(() => userStore.profile)
const viewUserId = computed(() => parseInt(route.params.id) || 0)
const isMe = computed(() => viewUserId.value === userStore.userId || (!route.params.id && userStore.isLoggedIn))
const notes = ref([])
const following = ref(false)
const authorName = ref('')
const followLoading = ref(false)

onMounted(async () => {
  const uid = isMe.value ? userStore.userId : viewUserId.value
  if (!uid && !userStore.isLoggedIn) return

  // 拉笔记列表
  try {
    const res = await getUserNotes(uid)
    const list = res?.data?.notes || res?.notes || []
    notes.value = list
    if (list.length > 0) authorName.value = '用户' + uid // mock, 后续接入真实用户名
  } catch {}

  // 检查关注状态
  if (!isMe.value && userStore.isLoggedIn) {
    try {
      const r = await isFollowing(userStore.userId, viewUserId.value)
      following.value = r?.data?.following || false
    } catch {}
  }
})

async function doFollow() {
  followLoading.value = true
  try {
    if (following.value) {
      await unfollow(userStore.userId, viewUserId.value)
      following.value = false
    } else {
      await follow(userStore.userId, viewUserId.value)
      following.value = true
      ElMessage.success('已关注')
    }
  } catch {} finally { followLoading.value = false }
}

function handleLogout() { userStore.logout(); location.reload() }
</script>

<style scoped>
.profile-page { background: var(--bg); min-height: calc(100vh - var(--header-h)); }
.profile-inner { max-width: 660px; margin: 0 auto; padding: 40px 24px; text-align: center; }
.av-big { width: 96px; height: 96px; border-radius: 50%; border: 3px solid var(--red-light); }
h2 { font-size: 22px; margin: 14px 0 4px; }
.bio { font-size: 14px; color: var(--text3); margin-bottom: 24px; }
.stats { display: flex; justify-content: center; gap: 48px; margin-bottom: 20px; }
.stat { display: flex; flex-direction: column; align-items: center; gap: 4px; }
.num { font-size: 22px; font-weight: 700; }
.label { font-size: 12px; color: var(--text3); }
.logout-btn { width: 200px; }
.profile-login { padding-top: 60px; }
.profile-login p { font-size: 15px; color: var(--text2); margin: 16px 0; }
.notes-section { margin-top: 32px; text-align: left; }
.notes-section h3 { font-size: 16px; margin-bottom: 14px; }
.notes-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--card-gap); }
@media (max-width: 600px) { .notes-grid { grid-template-columns: repeat(2, 1fr); } }
</style>

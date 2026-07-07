<template>
  <div class="note-shell" v-if="note">
    <div class="image-zone">
      <span class="close-fab" @click="$router.back()">✕</span>
      <div class="img-viewer">
        <img v-for="(u,i) in imageList" :key="i" :src="u" />
        <div v-if="!imageList.length" class="no-img">📝 暂无图片</div>
      </div>
    </div>

    <div class="info-panel">
      <div class="author-row" style="cursor:pointer" @click="goAuthor">
        <img class="av" :src="authorAv" />
        <div style="flex:1;min-width:0">
          <p class="author-name">{{ authorName }}</p>
          <p class="author-sub">0 粉丝 · 0 关注</p>
        </div>
        <el-button size="small" round @click.stop="toggleFollow" :type="following ? 'default' : 'danger'" plain>{{ following ? '已关注' : '+ 关注' }}</el-button>
      </div>

      <h1 class="note-title">{{ noteTitle }}</h1>
      <p class="note-body">{{ noteContent }}</p>
      <div class="tags" v-if="tagList.length">
        <span v-for="t in tagList" :key="t" class="tag">#{{ t }}</span>
      </div>
      <p class="note-time">{{ formatTime(note.created_at || note.CreatedAt) }}</p>

      <div class="actions-row">
        <div class="act" @click="requireAuth(toggleLike)" :class="{ on: isLiked }">
          <span class="act-icon">{{ isLiked ? '❤️' : '🤍' }}</span>
          <span class="act-num">{{ likeCount }}</span>
        </div>
        <div class="act" @click="requireAuth(toggleCollect)" :class="{ on: isCollected }">
          <span class="act-icon">{{ isCollected ? '⭐' : '☆' }}</span>
          <span class="act-num">{{ collectCount }}</span>
        </div>
        <div class="act" @click="requireAuth(openComment)">
          <span class="act-icon">💬</span>
          <span class="act-num">{{ commentCount }}</span>
        </div>
      </div>

      <div class="comments" v-if="comments.length">
        <h4>评论 {{ commentCount }}</h4>
        <div class="comment-item" v-for="c in comments" :key="c.id" style="cursor:pointer" @click="goUser(c.user_id)">
          <img class="c-av" :src="avatarUrl(c.user_id)" />
          <div>
            <p class="c-user">{{ c.username }} <span class="c-time">{{ formatTime(c.created_at) }}</span></p>
            <p class="c-text">{{ c.content }}</p>
          </div>
        </div>
      </div>
    </div>

    <el-dialog v-model="showComment" title="写评论" width="400px">
      <el-input v-model="commentText" type="textarea" :rows="3" placeholder="友善评论～" />
      <template #footer><el-button type="danger" @click="doComment" :disabled="!commentText.trim()">发布</el-button></template>
    </el-dialog>
    <el-dialog v-model="showLoginHint" title="需要登录" width="340px">
      <p style="text-align:center;padding:12px 0">登录后才能互动哦～</p>
      <el-button type="danger" style="width:100%" @click="$router.push('/login');showLoginHint=false">去登录</el-button>
    </el-dialog>
  </div>
  <div v-else class="loading-page">加载中...</div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore, avatarUrl } from '../stores/user'
import { getNoteDetail, getNoteInteractions, likeNote, unlikeNote, collectNote, uncollectNote, commentOnNote, follow, unfollow, isFollowing } from '../api'
import { ElMessage } from 'element-plus'

const route = useRoute(); const router = useRouter(); const userStore = useUserStore()
const noteId = parseInt(route.params.id); const note = ref(null)
const isLiked=ref(false);const isCollected=ref(false);const following=ref(false)
const likeCount=ref(0);const collectCount=ref(0);const commentCount=ref(0)
const comments=ref([]);const showComment=ref(false);const commentText=ref('');const showLoginHint=ref(false)

const authorId = computed(() => note.value?.user_id || note.value?.UserId || 0)
const authorAv = computed(() => avatarUrl(authorId.value))
const authorName = computed(() => note.value?.author_name || note.value?.Username || '用户')
const noteTitle = computed(() => note.value?.title || note.value?.Title || '')
const noteContent = computed(() => note.value?.content || note.value?.Content || '')
const imageList = computed(() => {
  const imgs = note.value?.image_urls || note.value?.ImageUrls || []
  return Array.isArray(imgs) ? imgs : (typeof imgs === 'string' ? imgs.split(',').filter(Boolean) : [])
})
const tagList = computed(() => {
  const tags = note.value?.tags || note.value?.Tags || []
  return Array.isArray(tags) ? tags : (typeof tags === 'string' ? tags.split(',').filter(Boolean) : [])
})

function goAuthor() { router.push(`/profile/${authorId.value}`) }
function goUser(uid) { if (uid) router.push(`/profile/${uid}`) }

onMounted(async () => {
  try { const r = await getNoteDetail(noteId); note.value = r.data } catch { note.value = { title: '加载失败', content: '' } }
  await fetchInteractions()
  // 检查是否已关注该作者
  if (userStore.isLoggedIn && authorId.value) {
    try { const r = await isFollowing(userStore.userId, authorId.value); following.value = r?.data?.following ?? false } catch {}
  }
})
async function fetchInteractions() {
  try { const d=(await getNoteInteractions(noteId, userStore.userId)).data; isLiked.value=d.is_liked;isCollected.value=d.is_collected;likeCount.value=d.like_count;collectCount.value=d.collect_count;commentCount.value=d.comment_count;comments.value=d.comments||[] } catch {}
}
function requireAuth(fn) { if(!userStore.isLoggedIn){showLoginHint.value=true;return} fn() }
async function toggleLike(){try{if(isLiked.value){await unlikeNote(noteId,userStore.userId);isLiked.value=false;likeCount.value--}else{await likeNote(noteId,userStore.userId);isLiked.value=true;likeCount.value++}}catch{}}
async function toggleCollect(){try{if(isCollected.value){await uncollectNote(noteId,userStore.userId);isCollected.value=false;collectCount.value--}else{await collectNote(noteId,userStore.userId);isCollected.value=true;collectCount.value++}}catch{}}
function openComment(){showComment.value=true}
async function doComment(){try{await commentOnNote(noteId,userStore.userId,commentText.value);commentCount.value++;commentText.value='';showComment.value=false;ElMessage.success('评论成功')}catch{}}
async function toggleFollow(){try{const uid=note.value?.user_id||note.value?.UserId;if(following.value){await unfollow(userStore.userId,uid);following.value=false}else{await follow(userStore.userId,uid);following.value=true}}catch{}}
const formatTime=ts=>ts?new Date(Number(ts)*1000).toLocaleDateString('zh-CN'):''
</script>

<style scoped>
.note-shell{display:flex;min-height:calc(100vh - var(--header-h))}
.image-zone{flex:1;background:#1a1a1a;display:flex;align-items:center;justify-content:center;position:relative;min-width:0}
.close-fab{position:absolute;top:14px;left:18px;font-size:26px;color:#fff;cursor:pointer;z-index:10;opacity:.7}
.close-fab:hover{opacity:1}
.img-viewer{display:flex;flex-direction:column;align-items:center;gap:12px;padding:40px 60px;max-height:calc(100vh - var(--header-h));overflow-y:auto}
.img-viewer img{max-width:100%;max-height:72vh;object-fit:contain;border-radius:4px}
.no-img{color:#555;font-size:32px;padding:60px}
.info-panel{width:428px;min-width:428px;background:#fff;overflow-y:auto;padding:24px 28px;border-left:1px solid var(--border)}
.author-row{display:flex;align-items:center;gap:12px;margin-bottom:20px}
.av{width:44px;height:44px;border-radius:50%}
.author-name{font-size:16px;font-weight:700}
.author-name:hover{color:var(--red)}
.author-sub{font-size:12px;color:var(--text3);margin-top:2px}
.note-title{font-size:20px;font-weight:700;margin-bottom:12px;line-height:1.4}
.note-body{font-size:15px;line-height:1.8;color:var(--text2);white-space:pre-wrap}
.tags{margin:14px 0;display:flex;gap:8px;flex-wrap:wrap}
.tag{font-size:13px;color:#3b82f6;background:#eff6ff;padding:3px 12px;border-radius:10px}
.note-time{font-size:12px;color:var(--text3);margin:8px 0 16px}
.actions-row{display:flex;gap:32px;padding:16px 0;border-top:1px solid var(--border);margin-top:8px}
.act{display:flex;align-items:center;gap:6px;cursor:pointer;user-select:none}
.act-icon{font-size:20px}
.act.on .act-num{color:var(--red)}
.act-num{font-size:14px;color:var(--text2)}
.comments{margin-top:20px}
.comments h4{font-size:14px;margin-bottom:12px}
.comment-item{display:flex;gap:10px;margin-bottom:14px}
.c-av{width:30px;height:30px;border-radius:50%;flex-shrink:0}
.c-user{font-size:13px;font-weight:600}
.c-time{font-weight:400;color:var(--text3);font-size:11px;margin-left:6px}
.c-text{font-size:14px;margin-top:3px}
.loading-page{text-align:center;padding:120px 0;color:#999}
@media(max-width:860px){
.note-shell{flex-direction:column}
.image-zone{max-height:55vh}
.img-viewer{padding:20px}
.info-panel{width:100%;min-width:unset;border-left:none}
}
</style>

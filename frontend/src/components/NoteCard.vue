<template>
  <div class="card" @click="goNote">
    <div class="card-img" v-if="imageList.length">
      <img :src="imageList[0]" />
      <span class="img-count" v-if="imageList.length > 1">{{ imageList.length }}图</span>
    </div>
    <div class="card-img card-img--placeholder" v-else>
      <span style="font-size:32px">📝</span>
    </div>
    <h3 class="title">{{ note.title }}</h3>
    <div class="meta">
      <div class="author" @click.stop="goAuthor">
        <img class="av" :src="avUrl" />
        <span class="name">{{ authorName }}</span>
      </div>
      <span class="likes">❤️ {{ note.like_count || 0 }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { avatarUrl } from '../stores/user'

const props = defineProps({ note: Object })
const router = useRouter()
const authorId = computed(() => props.note.author_id || props.note.user_id || 0)
const authorName = computed(() => props.note.author_name || '用户')
const avUrl = computed(() => avatarUrl(authorId.value))
const imageList = computed(() => {
  const imgs = props.note.image_urls || []
  return Array.isArray(imgs) ? imgs : (typeof imgs === 'string' ? imgs.split(',').filter(Boolean) : [])
})

function goNote() { router.push(`/note/${props.note.note_id || props.note.id}`) }
function goAuthor() { router.push(`/profile/${authorId.value}`) }
</script>

<style scoped>
.card { cursor: pointer; border-radius: 8px; overflow: hidden; background: #fff; transition: transform .15s; }
.card:hover { transform: translateY(-2px); }
.card:active { opacity: .95; }
.card-img { position: relative; width: 100%; background: #f0f0f0; overflow: hidden; }
.card-img img { width: 100%; display: block; object-fit: cover; }
.card-img--placeholder { display: flex; align-items: center; justify-content: center; padding: 40% 0; }
.img-count { position: absolute; bottom: 6px; right: 6px; background: rgba(0,0,0,.45); color: #fff; font-size: 11px; padding: 2px 8px; border-radius: 10px; }
.title { font-size: 13px; font-weight: 500; line-height: 1.5; padding: 8px 10px 0; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; color: var(--text1); }
.meta { display: flex; justify-content: space-between; align-items: center; padding: 6px 10px 10px; }
.author { display: flex; align-items: center; gap: 6px; min-width: 0; cursor: pointer; }
.author:hover .name { color: var(--red); }
.av { width: 18px; height: 18px; border-radius: 50%; flex-shrink: 0; }
.name { font-size: 12px; color: var(--text3); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; transition: color .15s; }
.likes { font-size: 12px; color: var(--text3); flex-shrink: 0; }
</style>

<template>
  <div class="feed-page">
    <div class="tabs-row">
      <span :class="{ active: feedTab === 'recommend' }" @click="switchTab('recommend')">推荐</span>
      <span :class="{ active: feedTab === 'following' }" @click="switchTab('following')">关注</span>
      <span v-for="t in tags" :key="t" :class="{ active: feedTab === t }" @click="switchTab(t)">{{ t }}</span>
    </div>

    <div class="feed-container" v-if="!loading">
      <div class="waterfall" v-if="displayItems.length">
        <div class="col" v-for="c in count" :key="c">
          <NoteCard v-for="item in colItems(c - 1)" :key="item.note_id || item.id" :note="item" />
        </div>
      </div>
      <div v-else class="empty-state">
        <p style="font-size:48px">📭</p>
        <p>{{ feedTab === 'following' ? '还没有关注任何人，去发现页看看吧～' : '暂无匹配内容' }}</p>
      </div>
    </div>
    <div v-if="loading" class="loading-state">加载中...</div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useFeedStore } from '../stores/feed'
import { useUserStore } from '../stores/user'
import NoteCard from '../components/NoteCard.vue'

const feedStore = useFeedStore()
const userStore = useUserStore()
const feedTab = ref('recommend')
const { items, loading } = storeToRefs(feedStore)
const tags = ['穿搭', '美食', '旅行', '美妆', '家居', '健身', '摄影', '知识', '宠物']

const winW = ref(window.innerWidth)
const count = computed(() => winW.value >= 1400 ? 6 : winW.value >= 1100 ? 5 : winW.value >= 768 ? 4 : 2)

// 标签过滤：检查笔记 tags 里是否包含选中的标签
const displayItems = computed(() => {
  if (!tags.includes(feedTab.value)) return items.value || []
  return (items.value || []).filter(item => {
    const itemTags = item.tags || item.Tags || []
    const tlist = Array.isArray(itemTags) ? itemTags : (typeof itemTags === 'string' ? itemTags.split(',') : [])
    return tlist.some(t => {
      const ts = String(t || '').trim().toLowerCase()
      return ts.includes(feedTab.value.toLowerCase())
    })
  })
})

function colItems(i) { return displayItems.value.filter((_, idx) => idx % count.value === i) }

function switchTab(t) {
  feedTab.value = t
  if (t === 'following') feedStore.fetchFollowing(userStore.userId)
  else feedStore.fetchRecommend(userStore.userId)
}

function onResize() { winW.value = window.innerWidth }
onMounted(() => { feedStore.fetchRecommend(userStore.userId); window.addEventListener('resize', onResize) })
onUnmounted(() => window.removeEventListener('resize', onResize))
</script>

<style scoped>
.feed-page { background: var(--bg); padding-top: 8px; }
.tabs-row { max-width: var(--max-width); margin: 0 auto; display: flex; overflow-x: auto; padding: 8px 24px; background: #fff; position: sticky; top: var(--header-h); z-index: 50; border-bottom: 1px solid var(--border); gap: 0; }
.tabs-row::-webkit-scrollbar { display: none; }
.tabs-row span { padding: 8px 16px; font-size: 14px; color: var(--text2); cursor: pointer; white-space: nowrap; flex-shrink: 0; border-radius: 4px; }
.tabs-row span.active { color: var(--text1); font-weight: 700; }
.feed-container { max-width: var(--max-width); margin: 0 auto; padding: var(--card-gap); }
.waterfall { display: flex; gap: var(--card-gap); }
.col { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: var(--card-gap); }
.empty-state { text-align: center; padding: 80px 0; color: #bbb; }
.empty-state p { margin-top: 12px; font-size: 14px; }
.loading-state { text-align: center; padding: 120px 0; color: #ccc; }
</style>

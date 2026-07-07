<template>
  <div class="discover-page">
    <div class="section-title">🔥 热门笔记</div>
    <div class="feed-container" v-if="!loading">
      <div class="waterfall">
        <div class="col" v-for="c in count" :key="c">
          <NoteCard v-for="item in colItems(c - 1)" :key="item.note_id || item.id" :note="item" />
        </div>
      </div>
    </div>
    <div v-if="loading" class="loading-state">加载中...</div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getHotNotes } from '../api'
import NoteCard from '../components/NoteCard.vue'

const hotList = ref([])
const loading = ref(true)
const count = computed(() => window.innerWidth >= 1400 ? 6 : window.innerWidth >= 1100 ? 5 : window.innerWidth >= 768 ? 4 : 2)

function colItems(i) { return hotList.value.filter((_, idx) => idx % count.value === i) }

onMounted(async () => {
  try { const res = await getHotNotes(30); hotList.value = res.data?.items || [] }
  catch {
    hotList.value = Array.from({ length: 12 }, (_, i) => ({
      note_id: i + 1,
      title: ['春日穿搭','美食探店','旅行攻略','护肤分享','家居改造','健身打卡','读书笔记','咖啡教程','Python入门','摄影构图','通勤穿搭','手冲咖啡'][i],
      author_name: ['Lisa','小美','旅行达人','护肤Kitty','阿杰','Jason','书虫','咖啡师Leo','码农日记','摄影师张','穿搭博主C','咖啡控'][i],
      image_urls: [`https://api.dicebear.com/7.x/shapes/svg?seed=n${i}`],
      like_count: Math.floor(Math.random() * 2000),
    }))
  }
  loading.value = false
})
</script>

<style scoped>
.discover-page { background: var(--bg); padding-top: 4px; }
.section-title { max-width: var(--max-width); margin: 0 auto; font-size: 16px; font-weight: 700; padding: 16px 24px 4px; }
.feed-container { max-width: var(--max-width); margin: 0 auto; padding: var(--card-gap); }
.waterfall { display: flex; gap: var(--card-gap); }
.col { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: var(--card-gap); }
.loading-state { text-align: center; padding: 120px 0; color: #ccc; }
</style>

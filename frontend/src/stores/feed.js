import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getFeed, getFollowing } from '../api'

export const useFeedStore = defineStore('feed', () => {
  const items = ref([])
  const loading = ref(false)

  async function fetchRecommend(userId, page = 1) {
    loading.value = true
    try { const res = await getFeed(userId, page); items.value = res?.data?.items || res?.items || [] } catch { items.value = [] }
    loading.value = false
  }

  async function fetchFollowing(userId) {
    loading.value = true
    try {
      const [feedRes, fRes] = await Promise.all([
        getFeed(userId, 1),
        getFollowing(userId, 1, 200),
      ])
      const all = feedRes?.data?.items || feedRes?.items || []
      const followeeIds = (fRes?.data?.users || []).map(u => u.id)
      items.value = all.filter(n => followeeIds.includes(n.author_id || n.user_id))
    } catch { items.value = [] }
    loading.value = false
  }

  return { items, loading, fetchRecommend, fetchFollowing }
})

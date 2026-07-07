<template>
  <div>
    <h2 style="margin-bottom:20px">标签分析</h2>
    <el-input v-model="tag" placeholder="输入标签名查询" style="width:240px;margin-bottom:16px" @keyup.enter="search" />
    <el-button type="primary" @click="search">查询</el-button>
    <el-card v-if="result" style="margin-top:16px">
      <p><strong>标签：</strong>{{ result.tag }}</p>
      <p><strong>笔记数：</strong>{{ result.note_count }}</p>
      <p><strong>总点赞：</strong>{{ result.total_likes }}</p>
      <p><strong>总收藏：</strong>{{ result.total_collects }}</p>
      <p><strong>总评论：</strong>{{ result.total_comments }}</p>
      <p><strong>互动率：</strong>{{ result.engagement_rate?.toFixed(2) }}</p>
    </el-card>
    <h3 style="margin-top:24px">热门标签 Top 10</h3>
    <el-table :data="topTags" style="width:100%;margin-top:12px">
      <el-table-column prop="tag" label="标签" />
      <el-table-column prop="note_count" label="笔记数" />
      <el-table-column label="互动率">
        <template #default="{ row }">{{ row.engagement_rate?.toFixed(2) }}</template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getTagAnalytics, getTopTags } from '../../api'

const tag = ref('')
const result = ref(null)
const topTags = ref([])

async function search() {
  if (!tag.value) return
  try {
    const res = await getTagAnalytics(tag.value)
    result.value = res.data
  } catch {}
}

onMounted(async () => {
  try {
    const res = await getTopTags(10)
    topTags.value = res.data?.tags || []
  } catch {}
})
</script>

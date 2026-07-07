<template>
  <div>
    <h2 style="margin-bottom:20px">数据看板</h2>
    <el-row :gutter="16">
      <el-col :span="6"><el-card><p style="color:#999;font-size:13px">总用户数</p><p style="font-size:28px;font-weight:700">{{ stats.users }}</p></el-card></el-col>
      <el-col :span="6"><el-card><p style="color:#999;font-size:13px">总笔记数</p><p style="font-size:28px;font-weight:700">{{ stats.notes }}</p></el-card></el-col>
      <el-col :span="6"><el-card><p style="color:#999;font-size:13px">今日互动</p><p style="font-size:28px;font-weight:700">{{ stats.interactions }}</p></el-card></el-col>
      <el-col :span="6"><el-card><p style="color:#999;font-size:13px">审核中</p><p style="font-size:28px;font-weight:700;color:#e6a23c">{{ stats.pending }}</p></el-card></el-col>
    </el-row>

    <h3 style="margin:24px 0 12px">📈 热门标签互动率</h3>
    <div ref="tagChart" style="width:100%;height:300px"></div>

    <h3 style="margin:24px 0 12px">📉 笔记粉丝增长（最近 7 天）</h3>
    <div ref="growthChart" style="width:100%;height:300px"></div>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import * as echarts from 'echarts'

const stats = ref({
  users: JSON.parse(localStorage.getItem('mock_users') || '[]').length,
  notes: JSON.parse(localStorage.getItem('mock_notes') || '[]').length,
  interactions: 128,
  pending: JSON.parse(localStorage.getItem('mock_users') || '[]').filter(u => u.status === 'pending').length,
})
const tagChart = ref(null)
const growthChart = ref(null)

onMounted(async () => {
  await nextTick()
  if (tagChart.value) {
    const c1 = echarts.init(tagChart.value)
    c1.setOption({
      tooltip: {},
      xAxis: { type: 'category', data: ['穿搭', '美食', '旅行', '美妆', '健身', '家居'] },
      yAxis: { type: 'value' },
      series: [{ type: 'bar', data: [2.8, 1.5, 1.2, 3.1, 0.9, 1.8], itemStyle: { color: '#ff6b6b' } }],
    })
  }
  if (growthChart.value) {
    const c2 = echarts.init(growthChart.value)
    const hours = Array.from({ length: 7 }, (_, i) => `${24 * (6 - i)}h 前`)
    c2.setOption({
      tooltip: {}, legend: { data: ['笔记1', '笔记2'] },
      xAxis: { type: 'category', data: hours },
      yAxis: { type: 'value' },
      series: [
        { name: '笔记1', type: 'line', data: [0, 2, 3, 5, 7, 12, 18], smooth: true },
        { name: '笔记2', type: 'line', data: [0, 1, 1, 2, 3, 5, 8], smooth: true },
      ],
    })
  }
})
</script>

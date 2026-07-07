<template>
  <div>
    <h2 style="margin-bottom:20px">定时发布管理</h2>
    <el-button type="primary" @click="showDialog = true" style="margin-bottom:16px">+ 新建定时任务</el-button>
    <el-table :data="tasks" style="width:100%">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="title" label="标题" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'done' ? 'success' : row.status === 'failed' ? 'danger' : 'warning'">
            {{ row.status === 'done' ? '已执行' : row.status === 'failed' ? '失败' : '待执行' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="定时" width="160">
        <template #default="{ row }">{{ new Date(row.schedule_time * 1000).toLocaleString() }}</template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="showDialog" title="创建定时发布任务" width="90%">
      <el-input v-model="form.title" placeholder="标题" style="margin-bottom:10px" />
      <el-input v-model="form.content" type="textarea" :rows="3" placeholder="正文" style="margin-bottom:10px" />
      <el-input v-model="form.tags" placeholder="标签（逗号分隔）" style="margin-bottom:10px" />
      <el-date-picker v-model="form.scheduleTime" type="datetime" placeholder="选择发布时间" style="width:100%" />
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" @click="createTask">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { schedulePost } from '../../api'
import { ElMessage } from 'element-plus'

const tasks = ref([])
const showDialog = ref(false)
const form = ref({ title: '', content: '', tags: '', scheduleTime: null })

async function createTask() {
  if (!form.value.scheduleTime) return ElMessage.warning('请选择发布时间')
  try {
    await schedulePost({
      user_id: 1,
      title: form.value.title,
      content: form.value.content,
      tags: form.value.tags ? form.value.tags.split(',') : [],
      schedule_time: Math.floor(new Date(form.value.scheduleTime).getTime() / 1000),
    })
    ElMessage.success('定时任务已创建')
    showDialog.value = false
    tasks.value.push({ id: tasks.value.length + 1, title: form.value.title, status: 'pending', schedule_time: Math.floor(new Date(form.value.scheduleTime).getTime() / 1000) })
  } catch {}
}
</script>

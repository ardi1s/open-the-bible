<template>
  <div>
    <h2 style="margin-bottom:20px">用户审核管理</h2>
    <el-table :data="users" style="width:100%">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="username" label="用户名" />
      <el-table-column prop="phone" label="手机号" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'approved' ? 'success' : row.status === 'rejected' ? 'danger' : 'warning'">
            {{ row.status === 'approved' ? '已通过' : row.status === 'rejected' ? '已拒绝' : '待审核' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" label="注册时间" width="160">
        <template #default="{ row }">{{ new Date(row.createdAt).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="操作" width="200">
        <template #default="{ row }">
          <el-button v-if="row.status === 'pending'" type="success" size="small" @click="approve(row.id)">通过</el-button>
          <el-button v-if="row.status === 'pending'" type="danger" size="small" @click="reject(row.id)">拒绝</el-button>
          <span v-else style="color:#999">—</span>
        </template>
      </el-table-column>
    </el-table>
    <div v-if="!users.length" style="text-align:center;padding:60px;color:#ccc">暂无注册用户</div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

const users = ref([])

function loadUsers() {
  users.value = JSON.parse(localStorage.getItem('mock_users') || '[]')
}

function approve(id) {
  const all = JSON.parse(localStorage.getItem('mock_users') || '[]')
  const u = all.find(u => u.id === id)
  if (u) u.status = 'approved'
  localStorage.setItem('mock_users', JSON.stringify(all))
  ElMessage.success('已通过审核')
  loadUsers()
}

function reject(id) {
  const all = JSON.parse(localStorage.getItem('mock_users') || '[]')
  const u = all.find(u => u.id === id)
  if (u) u.status = 'rejected'
  localStorage.setItem('mock_users', JSON.stringify(all))
  ElMessage.success('已拒绝')
  loadUsers()
}

onMounted(loadUsers)
</script>

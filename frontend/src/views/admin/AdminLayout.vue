<template>
  <div class="admin">
    <el-container>
      <el-header class="admin-header">
        <span class="admin-title">管理后台</span>
        <el-button size="small" @click="goHome" type="default">回主站</el-button>
      </el-header>
      <el-container>
        <el-aside width="180px" class="admin-sidebar">
          <el-menu :default-active="$route.path" router>
            <el-menu-item index="/admin/dashboard">📊 数据看板</el-menu-item>
            <el-menu-item index="/admin/users">👥 用户审核</el-menu-item>
            <el-menu-item index="/admin/scheduled">⏰ 定时发布</el-menu-item>
            <el-menu-item index="/admin/tags">🏷️ 标签分析</el-menu-item>
          </el-menu>
        </el-aside>
        <el-main><router-view /></el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
const router = useRouter()
function goHome() { router.push('/feed') }
</script>

<style scoped>
.admin { max-width: 100% !important; }
.admin-header { background: #fff; border-bottom: 1px solid #eee; display: flex; align-items: center; justify-content: space-between; padding: 0 20px; height: 56px; }
.admin-title { font-weight: 700; font-size: 16px; }
.admin-sidebar { background: #fafafa; border-right: 1px solid #eee; min-height: calc(100vh - 56px); padding: 12px 0; }
.el-menu { border-right: none; }
</style>

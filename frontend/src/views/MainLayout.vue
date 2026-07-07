<template>
  <div class="app-shell">
    <!-- 顶部导航 —— 仿小红书 PC 风格，居中 -->
    <header class="topbar">
      <div class="topbar-inner">
        <span class="logo" @click="$router.push('/feed')">小 X</span>
        <div class="nav-tabs">
          <router-link to="/feed" class="nav-item" :class="{ active: $route.path === '/feed' }">首页</router-link>
          <router-link to="/discover" class="nav-item" :class="{ active: $route.path === '/discover' }">发现</router-link>
          <router-link to="/profile" class="nav-item" :class="{ active: $route.path === '/profile' }">我的</router-link>
        </div>
        <div class="topbar-actions">
          <template v-if="!userStore.isLoggedIn">
            <router-link to="/login" class="login-btn">登录</router-link>
          </template>
          <template v-else>
            <router-link to="/admin/dashboard" class="admin-link">后台</router-link>
            <span class="logout-link" @click="doLogout">退出</span>
          </template>
        </div>
      </div>
    </header>

    <!-- 内容区域 -->
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { useUserStore } from '../stores/user'
const userStore = useUserStore()
function doLogout() { userStore.logout(); window.location.reload() }
</script>

<style scoped>
.app-shell { min-height: 100vh; background: var(--bg); }
.topbar { position: sticky; top: 0; z-index: 100; background: #fff; border-bottom: 1px solid var(--border); height: var(--header-h); }
.topbar-inner { max-width: var(--max-width); margin: 0 auto; display: flex; align-items: center; height: 100%; padding: 0 24px; }
.logo { font-size: 22px; font-weight: 800; background: linear-gradient(135deg, var(--red), #ee5a24); -webkit-background-clip: text; -webkit-text-fill-color: transparent; cursor: pointer; margin-right: 32px; }
.nav-tabs { display: flex; gap: 28px; }
.nav-item { font-size: 15px; color: var(--text2); padding: 4px 0; border-bottom: 2px solid transparent; transition: all .2s; }
.nav-item:hover, .nav-item.active { color: var(--text1); border-bottom-color: var(--text1); }
.topbar-actions { margin-left: auto; display: flex; align-items: center; gap: 16px; font-size: 14px; }
.login-btn { color: var(--red); font-weight: 500; }
.admin-link { color: var(--text3); }
.logout-link { color: var(--text3); cursor: pointer; }
.logout-link:hover { color: var(--red); }
.main-content { min-height: calc(100vh - var(--header-h)); }
</style>

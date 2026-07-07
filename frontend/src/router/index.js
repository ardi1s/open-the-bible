import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  // ── 带顶栏 + 底栏的页面 ──
  {
    path: '/',
    component: () => import('../views/MainLayout.vue'),
    redirect: '/feed',
    children: [
      { path: 'feed', name: 'Feed', component: () => import('../views/Feed.vue'), meta: { title: '首页' } },
      { path: 'discover', name: 'Discover', component: () => import('../views/Discover.vue'), meta: { title: '发现' } },
      { path: 'profile/:id?', name: 'Profile', component: () => import('../views/Profile.vue'), meta: { title: '我的' } },
      // 笔记详情也包进 MainLayout，有顶栏可回首页
      { path: 'note/:id', name: 'NoteDetail', component: () => import('../views/NoteDetail.vue'), meta: { title: '笔记' } },
    ],
  },
  // ── 登录 / 注册（不需要顶栏底栏）──
  { path: '/login', name: 'Login', component: () => import('../views/Login.vue') },
  { path: '/register', name: 'Register', component: () => import('../views/Register.vue') },
  { path: '/admin/login', name: 'AdminLogin', component: () => import('../views/admin/AdminLogin.vue') },
  // ── 管理后台 ──
  {
    path: '/admin',
    component: () => import('../views/admin/AdminLayout.vue'),
    redirect: '/admin/dashboard',
    meta: { requiresAuth: true, role: 'admin' },
    children: [
      { path: 'dashboard', name: 'AdminDashboard', component: () => import('../views/admin/Dashboard.vue'), meta: { title: '数据看板' } },
      { path: 'users', name: 'UserApproval', component: () => import('../views/admin/UserApproval.vue'), meta: { title: '用户审核' } },
      { path: 'scheduled', name: 'ScheduledPosts', component: () => import('../views/admin/ScheduledPosts.vue'), meta: { title: '定时发布' } },
      { path: 'tags', name: 'TagAnalyticsView', component: () => import('../views/admin/TagAnalytics.vue'), meta: { title: '标签分析' } },
    ],
  },
]

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  const role = localStorage.getItem('role')

  if (to.matched.some(r => r.meta.requiresAuth && r.meta.role === 'admin')) {
    if (!token || role !== 'admin') {
      return next('/admin/login')
    }
  }

  next()
})

export default router

import axios from 'axios'
import { ElMessage } from 'element-plus'

const api = axios.create({ baseURL: '/api', timeout: 10000 })

api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  res => res.data,
  err => {
    const msg = err.response?.data?.error || err.message || '网络错误'
    if (err.response?.status !== 404) ElMessage.error(msg)
    return Promise.reject(err)
  }
)

// ── 用户 ──
export const getUser = (id) => api.get(`/user/${id}`)
export const follow = (userId, id, sourceNoteId = 0) => api.post(`/user/${id}/follow`, { user_id: userId, source_note_id: sourceNoteId })
export const unfollow = (userId, id) => api.delete(`/user/${id}/follow`, { data: { user_id: userId } })
export const getFollowers = (id, page = 1, size = 20) => api.get(`/user/${id}/followers`, { params: { page, size } })
export const getFollowing = (id, page = 1, size = 20) => api.get(`/user/${id}/following`, { params: { page, size } })
export const isFollowing = (userId, targetId) => api.get(`/user/${userId}/is-following`, { params: { target_id: targetId } })

// ── 笔记 ──
export const createNote = (data) => api.post('/notes', data)
export const getNoteDetail = (id) => api.get(`/notes/${id}`)
export const getUserNotes = (userId, page = 1, size = 30) => api.get(`/user/${userId}/notes`, { params: { page, size } })

// ── 互动 ──
export const likeNote = (noteId, userId) => api.post(`/notes/${noteId}/like`, { user_id: userId })
export const unlikeNote = (noteId, userId) => api.delete(`/notes/${noteId}/like`, { data: { user_id: userId } })
export const collectNote = (noteId, userId) => api.post(`/notes/${noteId}/collect`, { user_id: userId })
export const uncollectNote = (noteId, userId) => api.delete(`/notes/${noteId}/collect`, { data: { user_id: userId } })
export const commentOnNote = (noteId, userId, content) => api.post(`/notes/${noteId}/comment`, { user_id: userId, content })
export const deleteComment = (commentId, userId) => api.delete(`/comments/${commentId}`, { data: { user_id: userId } })
export const getNoteInteractions = (noteId, userId) => api.get(`/notes/${noteId}/interactions`, { params: { user_id: userId } })

// ── Feed ──
export const getFeed = (userId, page = 1, size = 10) => api.get('/feed', { params: { user_id: userId, page, size } })

// ── Rank ──
export const getHotNotes = (count = 20) => api.get('/rank/hot', { params: { count } })

// ── Agent ──
export const getSuggestions = (userId) => api.get('/agent/suggestions', { params: { user_id: userId } })
export const getTagAnalytics = (tag) => api.get('/agent/tag/analytics', { params: { tag } })
export const getTopTags = (limit = 10) => api.get('/agent/top-tags', { params: { limit } })
export const schedulePost = (data) => api.post('/agent/schedule-post', data)

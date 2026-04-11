<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-copy">
        <span class="auth-eyebrow">Neko Auth</span>
        <h1>输入访问 token 进入工作台</h1>
        <p>后端会校验你输入的访问 token，校验通过后签发认证密钥，后续页面接口都会自动带上这个密钥。</p>
      </div>

      <el-form @submit.prevent="handleLogin">
        <el-form-item label="访问 token">
          <el-input v-model="accessToken" type="password" show-password placeholder="请输入访问 token" @keyup.enter="handleLogin" />
        </el-form-item>
        <el-button type="primary" :loading="submitting" class="auth-submit" @click="handleLogin">登录</el-button>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { setStoredAuthKey } from '@/utils/auth'

const router = useRouter()
const route = useRoute()

const accessToken = ref('')
const submitting = ref(false)

document.title = '认证登录'

async function handleLogin() {
  const token = String(accessToken.value || '').trim()
  if (!token) {
    ElMessage.warning('请输入访问 token')
    return
  }
  submitting.value = true
  try {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ accessToken: token }),
    })
    const data = await res.json()
    if (!data.success) {
      ElMessage.error(data.message || '认证失败')
      return
    }
    setStoredAuthKey(data.data?.authKey)
    ElMessage.success('认证成功')
    const redirect = typeof route.query.redirect === 'string' && route.query.redirect ? route.query.redirect : '/'
    router.replace(redirect)
  } catch {
    ElMessage.error('认证失败')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.auth-page {
  min-height: 100vh;
  padding: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background:
    radial-gradient(circle at top left, rgba(82, 155, 228, 0.12), transparent 32%),
    radial-gradient(circle at bottom right, rgba(239, 188, 125, 0.16), transparent 30%),
    var(--el-bg-color-page);
}

.auth-card {
  width: min(460px, 100%);
  padding: 28px;
  border-radius: 24px;
  background: color-mix(in srgb, var(--el-bg-color-overlay) 92%, white);
  box-shadow: 0 20px 48px rgba(51, 71, 91, 0.12);
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.auth-copy {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.auth-eyebrow {
  display: inline-flex;
  align-self: flex-start;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(84, 150, 215, 0.1);
  color: #5b88bb;
  font-size: 12px;
  font-weight: 700;
}

.auth-copy h1 {
  margin: 0;
  font-size: 28px;
}

.auth-copy p {
  margin: 0;
  color: var(--el-text-color-secondary);
  line-height: 1.7;
}

.auth-submit {
  width: 100%;
}
</style>
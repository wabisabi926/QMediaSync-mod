<script setup lang="ts">
import { reactive, shallowRef, useTemplateRef } from 'vue'
import { Key, Lock, User } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'

export interface LoginSubmitPayload {
  username: string
  password: string
  totp_code: string
  rememberMe: boolean
}

defineProps<{
  loading: boolean
}>()

const emit = defineEmits<{
  submit: [payload: LoginSubmitPayload]
}>()

const loginFormRef = useTemplateRef<FormInstance>('loginFormRef')
const validating = shallowRef(false)
const loginForm = reactive({
  username: '',
  password: '',
  totpCode: '',
  rememberMe: false,
})

const loginRules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 20, message: '用户名长度在 2 到 20 个字符', trigger: 'blur' },
  ],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

const handleSubmit = async () => {
  if (!loginFormRef.value || validating.value) return
  validating.value = true
  try {
    const valid = await loginFormRef.value.validate()
    if (!valid) return
    emit('submit', {
      username: loginForm.username,
      password: loginForm.password,
      totp_code: loginForm.totpCode,
      rememberMe: loginForm.rememberMe,
    })
  } finally {
    validating.value = false
  }
}
</script>

<template>
  <el-form
    ref="loginFormRef"
    :model="loginForm"
    :rules="loginRules"
    class="login-form"
    autocomplete="on"
    @submit.prevent="handleSubmit"
  >
    <el-form-item prop="username">
      <el-input
        v-model="loginForm.username"
        size="large"
        name="username"
        autocomplete="username"
        placeholder="请输入用户名…"
        :prefix-icon="User"
        :disabled="loading"
      />
    </el-form-item>

    <el-form-item prop="password">
      <el-input
        v-model="loginForm.password"
        type="password"
        size="large"
        name="password"
        autocomplete="current-password"
        placeholder="请输入密码…"
        :prefix-icon="Lock"
        show-password
        :disabled="loading"
      />
    </el-form-item>

    <el-form-item prop="totpCode">
      <el-input
        v-model="loginForm.totpCode"
        size="large"
        name="one-time-code"
        autocomplete="one-time-code"
        placeholder="动态验证码"
        :prefix-icon="Key"
        :disabled="loading"
        maxlength="6"
        inputmode="numeric"
      />
    </el-form-item>

    <el-form-item>
      <el-checkbox v-model="loginForm.rememberMe" :disabled="loading"> 保持登录状态 </el-checkbox>
    </el-form-item>

    <el-form-item>
      <el-button
        type="primary"
        size="large"
        class="login-button"
        native-type="submit"
        :loading="loading || validating"
      >
        {{ loading || validating ? '登录中…' : '登录' }}
      </el-button>
    </el-form-item>
  </el-form>
</template>

<style scoped>
.login-form {
  width: 100%;
}

.login-form .el-form-item {
  margin-bottom: 24px;
}

.login-button {
  width: 100%;
  height: 44px;
  font-size: 16px;
  font-weight: 500;
}

@media (max-width: 768px) {
  .login-form .el-form-item {
    margin-bottom: 20px;
  }

  .login-button {
    height: 40px;
    font-size: 15px;
  }
}

@media (max-width: 480px) {
  .login-button {
    height: 42px;
  }
}
</style>

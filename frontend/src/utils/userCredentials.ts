export const userCredentialLimits = {
  username: {
    min: 3,
    max: 20,
  },
  password: {
    min: 6,
  },
} as const

export interface CredentialTextRule {
  validator: (value: string) => string | undefined
}

type ElementRuleCallback = (error?: Error) => void

export const getTextLength = (value: string): number => Array.from(value).length

export const createUsernameRule = (label: string): CredentialTextRule => ({
  validator(value: string): string | undefined {
    const trimmed = value.trim()
    if (!trimmed) {
      return `请输入${label}`
    }
    const length = getTextLength(trimmed)
    if (
      length < userCredentialLimits.username.min ||
      length > userCredentialLimits.username.max
    ) {
      return `${label}长度必须在 ${userCredentialLimits.username.min} 到 ${userCredentialLimits.username.max} 个字符之间`
    }
    return undefined
  },
})

export const createPasswordRule = (label: string): CredentialTextRule => ({
  validator(value: string): string | undefined {
    if (!value) {
      return `请输入${label}`
    }
    if (getTextLength(value) < userCredentialLimits.password.min) {
      return `${label}长度至少 ${userCredentialLimits.password.min} 个字符`
    }
    return undefined
  },
})

export const validateUsername = (username: string, label = '用户名') =>
  createUsernameRule(label).validator(username)

export const validatePassword = (password: string, label = '密码') =>
  createPasswordRule(label).validator(password)

export const createElementCredentialRule = (rule: CredentialTextRule) => ({
  validator(_rule: unknown, value: string | undefined, callback: ElementRuleCallback) {
    const message = rule.validator(value ?? '')
    if (message) {
      callback(new Error(message))
      return
    }
    callback()
  },
  trigger: 'blur',
})

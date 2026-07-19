import { describe, expect, it } from 'vitest'

import {
  createLoginPasswordRule,
  createLoginUsernameRule,
  createPasswordRule,
  createUsernameRule,
  getTextLength,
  userCredentialLimits,
} from '@/utils/userCredentials'

describe('userCredentials', () => {
  it('用户名规则与后端保持 3 到 20 个字符', () => {
    const rule = createUsernameRule('用户名')

    expect(rule.validator('ab')).toBe('用户名长度必须在 3 到 20 个字符之间')
    expect(rule.validator('abc')).toBeUndefined()
    expect(rule.validator('admin123')).toBeUndefined()
    expect(rule.validator('管理员')).toBe('用户名只能包含英文和数字')
    expect(rule.validator('admin_user')).toBe('用户名只能包含英文和数字')
    expect(rule.validator('abcdefghijklmnopqrst')).toBeUndefined()
    expect(rule.validator('abcdefghijklmnopqrstu')).toBe('用户名长度必须在 3 到 20 个字符之间')
    expect(rule.validator('   ')).toBe('请输入用户名')
    expect(userCredentialLimits.username.min).toBe(3)
    expect(userCredentialLimits.username.max).toBe(20)
  })

  it('密码规则与后端保持至少 6 个字符', () => {
    const rule = createPasswordRule('密码')

    expect(rule.validator('12345')).toBe('密码长度至少 6 个字符')
    expect(rule.validator('123456')).toBe('密码不能是纯数字或纯字母')
    expect(rule.validator('secret')).toBe('密码不能是纯数字或纯字母')
    expect(rule.validator('12345!')).toBeUndefined()
    expect(rule.validator('secret1')).toBeUndefined()
    expect(rule.validator('')).toBe('请输入密码')
    expect(userCredentialLimits.password.min).toBe(6)
  })

  it('登录规则只校验非空以兼容旧账号', () => {
    const usernameRule = createLoginUsernameRule('用户名')
    const passwordRule = createLoginPasswordRule('密码')

    expect(usernameRule.validator('ab')).toBeUndefined()
    expect(usernameRule.validator('abcdefghijklmnopqrst')).toBeUndefined()
    expect(usernameRule.validator('abcdefghijklmnopqrstu')).toBe('用户名长度不能超过 20 个字符')
    expect(usernameRule.validator('   ')).toBe('请输入用户名')
    expect(passwordRule.validator('12345')).toBeUndefined()
    expect(passwordRule.validator('')).toBe('请输入密码')
  })

  it('按用户可见字符计数', () => {
    expect(getTextLength('管理员')).toBe(3)
  })
})

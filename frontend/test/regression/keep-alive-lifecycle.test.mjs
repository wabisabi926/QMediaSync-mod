import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'
import { test } from 'vitest'

test('keep-alive 页面在激活、停用和卸载时正确收敛资源', () => {
  const __dirname = dirname(fileURLToPath(import.meta.url))
  const frontendRoot = resolve(__dirname, '../..')

  const readSource = (relativePath) => readFileSync(resolve(frontendRoot, relativePath), 'utf8')

  const assertImportsIcons = (source, icons, messagePrefix) => {
    for (const icon of icons) {
      assert.match(
        source,
        new RegExp(
          `import\\s*{[\\s\\S]*?\\b${icon}\\b[\\s\\S]*?}\\s*from\\s*['"]@element-plus/icons-vue['"]`,
        ),
        `${messagePrefix} should import ${icon} from @element-plus/icons-vue`,
      )
    }
  }

  const getFunctionBody = (source, functionName) => {
    const start = source.indexOf(`export function ${functionName}`)
    assert.notEqual(start, -1, `${functionName} should exist`)

    const bodyStart = source.indexOf('{', start)
    assert.notEqual(bodyStart, -1, `${functionName} should have a body`)

    let depth = 0
    for (let index = bodyStart; index < source.length; index += 1) {
      if (source[index] === '{') depth += 1
      if (source[index] === '}') depth -= 1
      if (depth === 0) {
        return source.slice(bodyStart, index + 1)
      }
    }

    throw new Error(`${functionName} body should close`)
  }

  const getLocalFunctionBody = (source, functionName) => {
    const patterns = [
      new RegExp(`const\\s+${functionName}\\s*=\\s*async\\s*\\([^)]*\\)\\s*=>\\s*{`),
      new RegExp(`const\\s+${functionName}\\s*=\\s*\\([^)]*\\)\\s*=>\\s*{`),
      new RegExp(`async\\s+function\\s+${functionName}\\s*\\([^)]*\\)\\s*{`),
      new RegExp(`function\\s+${functionName}\\s*\\([^)]*\\)\\s*{`),
    ]

    let bodyStart = -1
    for (const pattern of patterns) {
      const match = pattern.exec(source)
      if (match) {
        bodyStart = match.index + match[0].lastIndexOf('{')
        break
      }
    }

    assert.notEqual(bodyStart, -1, `${functionName} should have a body`)

    let depth = 0
    for (let index = bodyStart; index < source.length; index += 1) {
      if (source[index] === '{') depth += 1
      if (source[index] === '}') depth -= 1
      if (depth === 0) {
        return source.slice(bodyStart, index + 1)
      }
    }

    throw new Error(`${functionName} body should close`)
  }

  const assertUsesHook = (source, hookName, message) => {
    assert.match(source, new RegExp(`\\b${hookName}\\s*\\(`), message)
  }

  const assertLoadUsesRequestGate = (source, functionName, messagePrefix) => {
    const body = getLocalFunctionBody(source, functionName)

    assert.match(
      body,
      /const\s+\w+\s*=\s*\w+RequestGate\.next\s*\(\s*\)/,
      `${messagePrefix} should create a request id with next()`,
    )
    assert.match(
      body,
      /\.isCurrent\s*\(\s*\w+\s*\)/,
      `${messagePrefix} should check isCurrent() before using a response`,
    )
  }

  const assertInvalidatesRequestGates = (body, messagePrefix) => {
    assert.match(
      body,
      /\w+RequestGate\.invalidate\s*\(\s*\)/,
      `${messagePrefix} should invalidate in-flight requests`,
    )
  }

  const realtimeEventsSource = readSource('src/composables/useRealtimeEvents.ts')
  const useRealtimeEventBody = getFunctionBody(realtimeEventsSource, 'useRealtimeEvent')

  const loginFormSource = readSource('src/components/auth/LoginForm.vue')
  assertImportsIcons(loginFormSource, ['User', 'Lock'], 'LoginForm.vue')
  assert.match(
    loginFormSource,
    /:prefix-icon\s*=\s*"User"/,
    'LoginForm.vue should bind User as the username prefix icon component',
  )
  assert.match(
    loginFormSource,
    /:prefix-icon\s*=\s*"Lock"/,
    'LoginForm.vue should bind Lock as the password prefix icon component',
  )
  assert.doesNotMatch(
    loginFormSource,
    /(?<!:)\bprefix-icon\s*=\s*"User"/,
    'LoginForm.vue should not use a string User prefix icon',
  )
  assert.doesNotMatch(
    loginFormSource,
    /(?<!:)\bprefix-icon\s*=\s*"Lock"/,
    'LoginForm.vue should not use a string Lock prefix icon',
  )

  assert.match(
    realtimeEventsSource,
    /from 'vue'/,
    'useRealtimeEvents should import lifecycle hooks from Vue',
  )
  assert.match(
    realtimeEventsSource,
    /\bonActivated\b/,
    'useRealtimeEvent should register onActivated for keep-alive pages',
  )
  assert.match(
    realtimeEventsSource,
    /\bonDeactivated\b/,
    'useRealtimeEvent should register onDeactivated for keep-alive pages',
  )
  assert.match(
    useRealtimeEventBody,
    /let\s+unsubscribe/,
    'useRealtimeEvent should keep unsubscribe state so subscriptions are idempotent',
  )
  assert.match(
    useRealtimeEventBody,
    /unsubscribe\s*=\s*on\s*\(/,
    'useRealtimeEvent should subscribe through the shared on() helper',
  )
  assertUsesHook(
    useRealtimeEventBody,
    'onActivated',
    'useRealtimeEvent should resubscribe when a cached page is activated',
  )
  assertUsesHook(
    useRealtimeEventBody,
    'onDeactivated',
    'useRealtimeEvent should unsubscribe when a cached page is deactivated',
  )
  assertUsesHook(
    useRealtimeEventBody,
    'onBeforeUnmount',
    'useRealtimeEvent should still clean up on final unmount',
  )

  for (const queuePage of [
    'src/components/AppUploadQueue.vue',
    'src/components/AppDownloadQueue.vue',
  ]) {
    const source = readSource(queuePage)
    const deactivateBody = getLocalFunctionBody(source, 'deactivateQueuePage')
    const unmountedBody =
      source.match(/onUnmounted\s*\(\s*\(\s*\)\s*=>\s*{[\s\S]*?}\s*\)/)?.[0] || ''

    assertImportsIcons(source, ['WarningFilled'], queuePage)
    assertUsesHook(source, 'onActivated', `${queuePage} should refresh/start polling on activation`)
    assertUsesHook(source, 'onDeactivated', `${queuePage} should stop polling on deactivation`)
    assert.match(
      source,
      /createActiveRequestGate/,
      `${queuePage} should use active request gates for cached page requests`,
    )
    assertLoadUsesRequestGate(source, 'loadQueueData', `${queuePage} loadQueueData`)
    assertLoadUsesRequestGate(source, 'loadQueueStatus', `${queuePage} loadQueueStatus`)
    assert.match(
      source,
      /let\s+isPageActive\s*=\s*false/,
      `${queuePage} should track active state to avoid duplicate initial refreshes`,
    )
    assert.match(
      source,
      /const\s+activateQueuePage\s*=[\s\S]*?isPageActive[\s\S]*?loadQueueData\(\)[\s\S]*?startAutoRefresh\(\)/,
      `${queuePage} should refresh and start polling from a shared activation path`,
    )
    assert.match(
      source,
      /const\s+deactivateQueuePage\s*=[\s\S]*?isPageActive[\s\S]*?stopAutoRefresh\(\)/,
      `${queuePage} should stop polling from a shared deactivation path`,
    )
    assertInvalidatesRequestGates(deactivateBody, `${queuePage} deactivateQueuePage`)
    assert.match(
      source,
      /onDeactivated\s*\(\s*deactivateQueuePage\s*\)/,
      `${queuePage} should stop auto refresh from onDeactivated`,
    )
    assert.match(
      source,
      /onUnmounted\s*\([\s\S]*?stopAutoRefresh\(\)[\s\S]*?\)/,
      `${queuePage} should keep onUnmounted cleanup as a fallback`,
    )
    assertInvalidatesRequestGates(unmountedBody, `${queuePage} onUnmounted`)
  }

  const assertRecordPageLifecycle = ({
    page,
    loadFunction,
    activateFunction,
    deactivateFunction,
  }) => {
    const source = readSource(page)
    const deactivateBody = getLocalFunctionBody(source, deactivateFunction)
    const unmountedBody =
      source.match(/onUnmounted\s*\(\s*\(\s*\)\s*=>\s*{[\s\S]*?}\s*\)/)?.[0] || ''

    assert.match(
      source,
      /createActiveRequestGate/,
      `${page} should use an active request gate for cached page requests`,
    )
    assertLoadUsesRequestGate(source, loadFunction, `${page} ${loadFunction}`)
    assert.match(
      source,
      new RegExp(
        `const\\s+${activateFunction}\\s*=\\s*\\(\\)\\s*=>\\s*{[\\s\\S]*?if\\s*\\(\\s*isPageActive\\s*\\)[\\s\\S]*?return[\\s\\S]*?isPageActive\\s*=\\s*true[\\s\\S]*?${loadFunction}\\s*\\(\\s*\\)`,
      ),
      `${page} should refresh records from the shared activation path`,
    )
    assertUsesHook(source, 'onActivated', `${page} should refresh records on activation`)
    assertUsesHook(
      source,
      'onDeactivated',
      `${page} should mark records page inactive on deactivation`,
    )
    assert.match(
      source,
      /let\s+isPageActive\s*=\s*false/,
      `${page} should track active state to avoid duplicate initial refreshes`,
    )
    assert.match(
      source,
      new RegExp(
        `const\\s+${deactivateFunction}\\s*=\\s*\\(\\)\\s*=>\\s*{[\\s\\S]*?if\\s*\\(\\s*!isPageActive\\s*\\)[\\s\\S]*?return[\\s\\S]*?isPageActive\\s*=\\s*false`,
      ),
      `${page} should mark the page inactive from a shared deactivation path`,
    )
    assertInvalidatesRequestGates(deactivateBody, `${page} ${deactivateFunction}`)
    assert.match(
      source,
      new RegExp(`onMounted\\s*\\(\\s*${activateFunction}\\s*\\)`),
      `${page} should use the shared activation path from onMounted`,
    )
    assert.match(
      source,
      new RegExp(`onActivated\\s*\\(\\s*${activateFunction}\\s*\\)`),
      `${page} should use the shared activation path from onActivated`,
    )
    assert.match(
      source,
      new RegExp(`onDeactivated\\s*\\(\\s*${deactivateFunction}\\s*\\)`),
      `${page} should use the shared deactivation path from onDeactivated`,
    )
    assertInvalidatesRequestGates(unmountedBody, `${page} onUnmounted`)
  }

  assertRecordPageLifecycle({
    page: 'src/components/AppSyncRecords.vue',
    loadFunction: 'loadSyncRecords',
    activateFunction: 'activateSyncRecordsPage',
    deactivateFunction: 'deactivateSyncRecordsPage',
  })

  const syncDirectoryFormSource = readSource('src/components/AppSyncDirectoryForm.vue')
  assertImportsIcons(syncDirectoryFormSource, ['ArrowLeft'], 'AppSyncDirectoryForm.vue')
})

import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { test } from 'vitest'

test('队列页在刷新、查询切换和停用时保持页面状态一致', () => {
  const __dirname = dirname(fileURLToPath(import.meta.url))
  const frontendRoot = resolve(__dirname, '../..')

  const readSource = (relativePath) => readFileSync(resolve(frontendRoot, relativePath), 'utf8')

  const getLocalFunctionBody = (source, functionName) => {
    const patterns = [
      new RegExp(`const\\s+${functionName}\\s*=\\s*async\\s*\\([^)]*\\)\\s*=>\\s*{`),
      new RegExp(`const\\s+${functionName}\\s*=\\s*\\([^)]*\\)\\s*=>\\s*{`),
    ]

    let start = -1
    for (const pattern of patterns) {
      const match = pattern.exec(source)
      if (match) {
        start = match.index
        break
      }
    }

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

  const extractBalancedBlock = (source, bodyStart, label) => {
    assert.notEqual(bodyStart, -1, `${label} should have a body`)

    let depth = 0
    for (let index = bodyStart; index < source.length; index += 1) {
      if (source[index] === '{') depth += 1
      if (source[index] === '}') depth -= 1
      if (depth === 0) {
        return {
          body: source.slice(bodyStart, index + 1),
          end: index + 1,
        }
      }
    }

    throw new Error(`${label} body should close`)
  }

  const getActivatedCallbackBodyContaining = (source, needle) => {
    let searchFrom = 0

    while (searchFrom < source.length) {
      const activatedIndex = source.indexOf('onActivated', searchFrom)
      assert.notEqual(activatedIndex, -1, `onActivated callback containing ${needle} should exist`)

      const bodyStart = source.indexOf('{', activatedIndex)
      const { body, end } = extractBalancedBlock(source, bodyStart, 'onActivated callback')

      if (body.includes(needle)) {
        return body
      }

      searchFrom = end
    }

    throw new Error(`onActivated callback containing ${needle} should exist`)
  }

  const countMatches = (source, pattern) => source.match(pattern)?.length ?? 0
  const escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

  const assertImportsNamed = (source, names, moduleName, messagePrefix) => {
    for (const name of names) {
      assert.match(
        source,
        new RegExp(
          `import\\s*{[\\s\\S]*?\\b${name}\\b[\\s\\S]*?}\\s*from\\s*['"]${moduleName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}['"]`,
        ),
        `${messagePrefix} should import ${name} from ${moduleName}`,
      )
    }
  }

  const assertComputedPageState = (source, pageKey, messagePrefix) => {
    assert.match(
      source,
      new RegExp(
        `const\\s+pageStateStore\\s*=\\s*usePageStateStore\\s*\\(\\s*\\)[\\s\\S]*?const\\s+pageState\\s*=\\s*pageStateStore\\.getPageState\\s*\\(\\s*['"]${pageKey}['"]`,
      ),
      `${messagePrefix} should create page state with ${pageKey}`,
    )
    assert.match(
      source,
      /const\s+currentPage\s*=\s*computed\s*\(\s*{[\s\S]*?get:\s*\(\)\s*=>\s*pageState\.currentPage[\s\S]*?set:\s*\(\s*value\s*\)\s*=>\s*pageStateStore\.setPagination/,
      `${messagePrefix} currentPage should be a computed page-state proxy`,
    )
    assert.match(
      source,
      /const\s+pageSize\s*=\s*computed\s*\(\s*{[\s\S]*?get:\s*\(\)\s*=>\s*pageState\.pageSize[\s\S]*?set:\s*\(\s*value\s*\)\s*=>\s*pageStateStore\.setPagination/,
      `${messagePrefix} pageSize should be a computed page-state proxy`,
    )
    assert.match(
      source,
      /const\s+statusFilter\s*=\s*computed\s*\(\s*{[\s\S]*?get:\s*\(\)\s*=>\s*Number\s*\(\s*pageState\.filters\.status\s*\?\?\s*-1\s*\)[\s\S]*?set:\s*\(\s*value\s*\)\s*=>\s*pageStateStore\.setFilter/,
      `${messagePrefix} statusFilter should be a computed page-state proxy`,
    )
  }

  const assertTablesUseControlledExpansion = (source, rowType, messagePrefix) => {
    assert.equal(
      countMatches(
        source,
        new RegExp(
          `:row-key\\s*=\\s*["']\\(row:\\s*${rowType}\\)\\s*=>\\s*String\\(row\\.id\\)["']`,
          'g',
        ),
      ),
      2,
      `${messagePrefix} should set row-key on both tables`,
    )
    assert.equal(
      countMatches(source, /:expand-row-keys\s*=\s*["']pageState\.expandedRowKeys["']/g),
      2,
      `${messagePrefix} should bind expanded row keys on both tables`,
    )
    assert.equal(
      countMatches(source, /@expand-change\s*=\s*["']handleExpandChange["']/g),
      2,
      `${messagePrefix} should handle expand changes on both tables`,
    )
    assert.equal(
      countMatches(source, /v-loading\s*=\s*["']initialLoading\s*\|\|\s*queryLoading["']/g),
      2,
      `${messagePrefix} should show table loading during initial loads and query switches`,
    )
    assert.equal(
      countMatches(source, /v-if\s*=\s*["']isMobileView["']/g),
      1,
      `${messagePrefix} should only mount the mobile table on mobile`,
    )
    assert.equal(
      countMatches(source, /<el-table\s*\n\s*v-else/g),
      1,
      `${messagePrefix} should mount the desktop table as the alternative branch`,
    )
    assert.doesNotMatch(
      source,
      new RegExp(`<el-table\\s+v-if="isMobileView"[\\s\\S]*?class="hidden-md-and-up"`),
      `${messagePrefix} mobile table should not use a mismatched Element Plus breakpoint class`,
    )
    assert.doesNotMatch(
      source,
      new RegExp(`<el-table\\s+v-else[\\s\\S]*?class="hidden-md-and-down"`),
      `${messagePrefix} desktop table should not use a mismatched Element Plus breakpoint class`,
    )
  }

  const assertLoadQueueData = (
    source,
    pageKey,
    counterField,
    failureMessage,
    legacyFailureMessage,
    messagePrefix,
  ) => {
    const body = getLocalFunctionBody(source, 'loadQueueData')

    assert.match(body, /runRefresh\s*\(/, `${messagePrefix} loadQueueData should use runRefresh`)
    assert.match(
      body,
      /queueDataRequestGate\.next\s*\(\s*\)/,
      `${messagePrefix} loadQueueData should keep request gate ids`,
    )
    assert.match(
      body,
      /queueDataRequestGate\.isCurrent\s*\(/,
      `${messagePrefix} loadQueueData should ignore stale responses`,
    )
    assert.match(
      body,
      /mergeStableList\s*\(\s*queueData\.value\s*,\s*rows\s*,\s*\(\s*row\s*\)\s*=>\s*row\.id\s*\)/,
      `${messagePrefix} should merge rows by stable id`,
    )
    assert.match(
      body,
      new RegExp(
        `pageStateStore\\.setExpandedRowKeys\\s*\\(\\s*['"]${pageKey}['"][\\s\\S]*?retainExistingKeys\\s*\\(\\s*pageState\\.expandedRowKeys\\s*,\\s*queueData\\.value\\s*,\\s*\\(\\s*row\\s*\\)\\s*=>\\s*row\\.id\\s*\\)`,
      ),
      `${messagePrefix} should retain only expanded rows still present after refresh`,
    )
    assert.match(
      body,
      new RegExp(`${counterField}\\.value\\s*=\\s*response\\.data\\.data\\.${counterField}`),
      `${messagePrefix} should update ${counterField} count from the response`,
    )
    assert.doesNotMatch(
      body,
      new RegExp(
        `ElMessage\\.error\\s*\\(\\s*['"]${escapeRegExp(legacyFailureMessage)}['"]\\s*\\)`,
      ),
      `${messagePrefix} loadQueueData should not use the legacy failure message`,
    )
    assert.equal(
      countMatches(
        body,
        new RegExp(
          `ElMessage\\.error\\s*\\(\\s*['"]${escapeRegExp(failureMessage)}['"]\\s*\\)`,
          'g',
        ),
      ),
      2,
      `${messagePrefix} loadQueueData should use the specified failure message for response and catch failures`,
    )
    assert.match(
      body,
      new RegExp(
        `catch\\s*\\([^)]*\\)\\s*{[\\s\\S]*?ElMessage\\.error\\s*\\(\\s*['"]${escapeRegExp(failureMessage)}['"]\\s*\\)`,
      ),
      `${messagePrefix} loadQueueData catch branch should use the specified failure message`,
    )
  }

  const assertLoadQueueDataCoalescesInFlightChanges = (source, messagePrefix) => {
    const body = getLocalFunctionBody(source, 'loadQueueData')
    const requestIdIndex = body.indexOf('queueDataRequestGate.next()')
    const isRefreshingIndex = body.indexOf('isRefreshing.value')
    const runRefreshIndex = body.indexOf('runRefresh')

    assert.match(
      source,
      /const\s+{\s*initialLoading\s*,\s*backgroundRefreshing\s*,\s*isRefreshing\s*,\s*runRefresh\s*}\s*=\s*useBackgroundRefresh\s*\(\s*\)/,
      `${messagePrefix} should read isRefreshing from useBackgroundRefresh to coalesce refreshes`,
    )
    assert.match(
      source,
      /const\s+pendingQueueDataRefresh\s*=\s*ref\s*\(\s*false\s*\)/,
      `${messagePrefix} should keep a pending queue refresh flag`,
    )
    assert.match(
      body,
      /if\s*\(\s*!\s*isPageActive\s*\)\s*{\s*return\s*}/,
      `${messagePrefix} loadQueueData should skip inactive pages before creating requests`,
    )
    assert.notEqual(requestIdIndex, -1, `${messagePrefix} loadQueueData should create a request id`)
    assert.notEqual(
      isRefreshingIndex,
      -1,
      `${messagePrefix} loadQueueData should check for an in-flight refresh`,
    )
    assert.notEqual(runRefreshIndex, -1, `${messagePrefix} loadQueueData should use runRefresh`)
    assert.ok(
      requestIdIndex < isRefreshingIndex && isRefreshingIndex < runRefreshIndex,
      `${messagePrefix} loadQueueData should invalidate stale responses before coalescing in-flight refreshes`,
    )
    assert.match(
      body,
      /if\s*\(\s*isRefreshing\.value\s*\)\s*{[\s\S]*?pendingQueueDataRefresh\.value\s*=\s*true[\s\S]*?return\s*}/,
      `${messagePrefix} loadQueueData should record one pending refresh while a request is in flight`,
    )
    assert.match(
      body,
      /finally\s*{[\s\S]*?if\s*\(\s*pendingQueueDataRefresh\.value\s*&&\s*isPageActive\s*\)\s*{[\s\S]*?pendingQueueDataRefresh\.value\s*=\s*false[\s\S]*?await\s+loadQueueData\s*\(\s*\)/,
      `${messagePrefix} loadQueueData should replay one pending refresh after the current refresh finishes`,
    )
  }

  const assertActivationRepair = (source, pageKey, messagePrefix) => {
    const body = getActivatedCallbackBodyContaining(source, 'pruneExpandedRowKeys')
    const guardIndex = body.search(/if\s*\(\s*queueData\.value\.length\s*>\s*0\s*\)\s*{/)
    assert.notEqual(
      guardIndex,
      -1,
      `${messagePrefix} onActivated prune should be guarded by non-empty queue data`,
    )

    const guardBodyStart = body.indexOf('{', guardIndex)
    const { body: guardedBody, end: guardEnd } = extractBalancedBlock(
      body,
      guardBodyStart,
      `${messagePrefix} onActivated queue data guard`,
    )

    assert.match(
      guardedBody,
      new RegExp(
        `pruneExpandedRowKeys\\s*\\(\\s*['"]${pageKey}['"]\\s*,\\s*queueData\\.value\\.map\\s*\\(\\s*\\(\\s*row\\s*\\)\\s*=>\\s*String\\s*\\(\\s*row\\.id\\s*\\)\\s*\\)\\s*,?\\s*\\)`,
      ),
      `${messagePrefix} should prune expanded keys on activation only when queue data is non-empty`,
    )

    const resizeIndex = body.search(
      /nextTick\s*\(\s*\(\s*\)\s*=>\s*{[\s\S]*?window\.dispatchEvent\s*\(\s*new Event\s*\(\s*['"]resize['"]\s*\)\s*\)/,
    )
    assert.ok(
      resizeIndex > guardEnd,
      `${messagePrefix} should dispatch resize after activation outside the non-empty queue data guard`,
    )
    assert.match(
      source,
      /nextTick\s*\(\s*\(\s*\)\s*=>\s*{[\s\S]*?window\.dispatchEvent\s*\(\s*new Event\s*\(\s*['"]resize['"]\s*\)\s*\)/,
      `${messagePrefix} should dispatch resize after activation`,
    )
  }

  const assertQueueMutationsUseContext = (source, functionNames, messagePrefix) => {
    assert.match(
      source,
      /const\s+queueMutationContextVersion\s*=\s*ref\s*\(\s*0\s*\)/,
      `${messagePrefix} should version queue mutation operations`,
    )
    assert.match(
      source,
      /const\s+activeQueueMutationContext\s*=\s*ref\s*<\s*QueueMutationContextSnapshot\s*\|\s*null\s*>\s*\(\s*null\s*\)/,
      `${messagePrefix} should track the active queue mutation context`,
    )
    assert.match(
      getLocalFunctionBody(source, 'isQueueMutationContextCurrent'),
      /isPageActive[\s\S]*?activeQueueMutationContext\.value[\s\S]*?snapshot\.contextVersion\s*===\s*queueMutationContextVersion\.value/,
      `${messagePrefix} queue mutations should require active page and current mutation version`,
    )
    assert.match(
      getLocalFunctionBody(source, 'invalidateQueueMutationContext'),
      /queueMutationContextVersion\.value\s*\+=\s*1[\s\S]*?activeQueueMutationContext\.value\s*=\s*null/,
      `${messagePrefix} should invalidate queue mutations by version`,
    )
    assert.match(
      getLocalFunctionBody(source, 'deactivateQueuePage'),
      /invalidateQueueMutationContext\s*\(\s*\)/,
      `${messagePrefix} deactivation should invalidate pending queue mutations`,
    )
    assert.match(
      source,
      /onUnmounted\s*\(\s*\(\s*\)\s*=>\s*{[\s\S]*?invalidateQueueMutationContext\s*\(\s*\)/,
      `${messagePrefix} unmount should invalidate pending queue mutations`,
    )

    for (const functionName of functionNames) {
      const body = getLocalFunctionBody(source, functionName)
      assert.match(
        body,
        /const\s+operationContext\s*=\s*startQueueMutationContext\s*\(\s*\)/,
        `${messagePrefix} ${functionName} should start a mutation context`,
      )
      assert.match(
        body,
        /await[\s\S]*?if\s*\(\s*!isQueueMutationContextCurrent\s*\(\s*operationContext\s*\)\s*\)\s*{[\s\S]*?return[\s\S]*?}/,
        `${messagePrefix} ${functionName} should re-check context after awaited work`,
      )
      assert.match(
        body,
        /catch\s*(?:\([^)]*\))?\s*{[\s\S]*?if\s*\(\s*!isQueueMutationContextCurrent\s*\(\s*operationContext\s*\)\s*\)\s*{[\s\S]*?return[\s\S]*?}/,
        `${messagePrefix} ${functionName} catch should ignore stale mutation responses`,
      )
      assert.match(
        body,
        /finally\s*{[\s\S]*?if\s*\(\s*isQueueMutationContextCurrent\s*\(\s*operationContext\s*\)\s*\)/,
        `${messagePrefix} ${functionName} finally should only finish the current mutation context`,
      )
    }
  }

  for (const queuePage of [
    {
      path: 'src/components/AppUploadQueue.vue',
      key: 'upload-queue',
      rowType: 'UploadTask',
      counterField: 'uploading',
      failureMessage: '获取上传队列数据失败',
      legacyFailureMessage: '加载上传队列数据失败',
      mutationFunctions: [
        'clearQueue',
        'clearSuccessAndFailedTasks',
        'retryAllFailedTasks',
        'pauseAllTasks',
        'resumeAllTasks',
      ],
    },
    {
      path: 'src/components/AppDownloadQueue.vue',
      key: 'download-queue',
      rowType: 'DownloadTask',
      counterField: 'downloading',
      failureMessage: '获取下载队列数据失败',
      legacyFailureMessage: '加载下载队列数据失败',
      mutationFunctions: [
        'clearQueue',
        'clearSuccessAndFailedTasks',
        'pauseAllTasks',
        'resumeAllTasks',
      ],
    },
  ]) {
    const source = readSource(queuePage.path)
    const messagePrefix = queuePage.path

    assertImportsNamed(source, ['computed', 'nextTick', 'onActivated'], 'vue', messagePrefix)
    assertImportsNamed(source, ['usePageStateStore'], '@/stores/pageState', messagePrefix)
    assertImportsNamed(
      source,
      ['mergeStableList', 'retainExistingKeys'],
      '@/composables/useStableList',
      messagePrefix,
    )
    assertImportsNamed(
      source,
      ['useBackgroundRefresh'],
      '@/composables/useBackgroundRefresh',
      messagePrefix,
    )

    assertComputedPageState(source, queuePage.key, messagePrefix)
    assertLoadQueueData(
      source,
      queuePage.key,
      queuePage.counterField,
      queuePage.failureMessage,
      queuePage.legacyFailureMessage,
      messagePrefix,
    )
    assertLoadQueueDataCoalescesInFlightChanges(source, messagePrefix)
    assertTablesUseControlledExpansion(source, queuePage.rowType, messagePrefix)
    assert.match(
      source,
      /<el-button[\s\S]*?type="info"[\s\S]*?@click="refreshQueue"[\s\S]*?:loading="backgroundRefreshing"/,
      `${messagePrefix} refresh button should show background refresh state`,
    )
    assert.match(
      source,
      new RegExp(
        `const\\s+handleExpandChange\\s*=\\s*\\(\\s*row:\\s*${queuePage.rowType}\\s*,\\s*expandedRows:\\s*${queuePage.rowType}\\[\\]\\s*\\)\\s*=>\\s*{[\\s\\S]*?pageStateStore\\.setExpandedRowKeys\\s*\\(\\s*['"]${queuePage.key}['"][\\s\\S]*?expandedRows\\.map\\s*\\(\\s*\\(\\s*item\\s*\\)\\s*=>\\s*String\\s*\\(\\s*item\\.id\\s*\\)\\s*\\)`,
      ),
      `${messagePrefix} should persist controlled expanded rows`,
    )
    assertActivationRepair(source, queuePage.key, messagePrefix)
    assertQueueMutationsUseContext(source, queuePage.mutationFunctions, messagePrefix)
  }
})

import assert from 'node:assert/strict'
import { join, resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import ts from 'typescript'
import { afterEach, test } from 'vitest'

const cleanups = []

afterEach(async () => {
  while (cleanups.length > 0) {
    await cleanups.pop()()
  }
})

async function loadHistoryModule() {
  const cacheRoot = resolve('node_modules/.cache')
  await mkdir(cacheRoot, { recursive: true })
  const outdir = await mkdtemp(join(cacheRoot, 'qms-router-history-'))
  const outfile = join(outdir, 'history.mjs')

  cleanups.push(() => rm(outdir, { recursive: true, force: true }))

  const source = await readFile(resolve('src/router/history.ts'), 'utf8')
  const result = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.ESNext,
      target: ts.ScriptTarget.ES2022,
    },
  })
  await writeFile(outfile, result.outputText)

  return import(`${pathToFileURL(outfile).href}?t=${Date.now()}`)
}

function installDocumentMock(documentMock) {
  const hasDocument = Object.hasOwn(globalThis, 'document')
  const originalDocument = globalThis.document

  Object.defineProperty(globalThis, 'document', {
    value: documentMock,
    configurable: true,
    writable: true,
  })

  cleanups.push(() => {
    if (hasDocument) {
      Object.defineProperty(globalThis, 'document', {
        value: originalDocument,
        configurable: true,
        writable: true,
      })
      return
    }

    Reflect.deleteProperty(globalThis, 'document')
  })
}

function installWindowMock(windowMock) {
  const hasWindow = Object.hasOwn(globalThis, 'window')
  const originalWindow = globalThis.window

  Object.defineProperty(globalThis, 'window', {
    value: windowMock,
    configurable: true,
    writable: true,
  })

  cleanups.push(() => {
    if (hasWindow) {
      Object.defineProperty(globalThis, 'window', {
        value: originalWindow,
        configurable: true,
        writable: true,
      })
      return
    }

    Reflect.deleteProperty(globalThis, 'window')
  })
}

function installGlobalMock(name, value) {
  const hasValue = Object.hasOwn(globalThis, name)
  const originalValue = globalThis[name]

  Object.defineProperty(globalThis, name, {
    value,
    configurable: true,
    writable: true,
  })

  cleanups.push(() => {
    if (hasValue) {
      Object.defineProperty(globalThis, name, {
        value: originalValue,
        configurable: true,
        writable: true,
      })
      return
    }

    Reflect.deleteProperty(globalThis, name)
  })
}

test('runWithoutHiddenScrollListeners skips visibilitychange listeners while creating history', async () => {
  const { runWithoutHiddenScrollListeners } = await loadHistoryModule()
  const addedEvents = []
  const documentMock = {
    addEventListener(type) {
      addedEvents.push(type)
      assert.equal(this, documentMock)
    },
  }

  installDocumentMock(documentMock)

  const result = runWithoutHiddenScrollListeners(() => {
    document.addEventListener('visibilitychange', () => {})
    document.addEventListener('click', () => {})
    return 'history-created'
  })

  assert.equal(result, 'history-created')
  assert.deepEqual(addedEvents, ['click'])
  assert.equal(document.addEventListener, documentMock.addEventListener)
})

test('runWithoutHiddenScrollListeners skips pagehide listeners while creating history', async () => {
  const { runWithoutHiddenScrollListeners } = await loadHistoryModule()
  const addedEvents = []
  const windowMock = {
    addEventListener(type) {
      addedEvents.push(type)
      assert.equal(this, windowMock)
    },
  }

  installWindowMock(windowMock)

  const result = runWithoutHiddenScrollListeners(() => {
    window.addEventListener('pagehide', () => {})
    window.addEventListener('popstate', () => {})
    return 'history-created'
  })

  assert.equal(result, 'history-created')
  assert.deepEqual(addedEvents, ['popstate'])
  assert.equal(window.addEventListener, windowMock.addEventListener)
})

test('runWithoutHiddenScrollListeners restores patched listeners when factory throws', async () => {
  const { runWithoutHiddenScrollListeners } = await loadHistoryModule()
  const documentMock = {
    addEventListener() {},
  }
  const windowMock = {
    addEventListener() {},
  }

  installDocumentMock(documentMock)
  installWindowMock(windowMock)

  assert.throws(
    () =>
      runWithoutHiddenScrollListeners(() => {
        throw new Error('factory failed')
      }),
    /factory failed/,
  )
  assert.equal(document.addEventListener, documentMock.addEventListener)
  assert.equal(window.addEventListener, windowMock.addEventListener)
})

test('createQMediaSyncHashHistory keeps popstate and skips hidden scroll listeners', async () => {
  const { createQMediaSyncHashHistory } = await loadHistoryModule()
  const documentEvents = []
  const windowEvents = []
  const locationMock = {
    protocol: 'http:',
    host: 'example.local',
    pathname: '/',
    search: '',
    hash: '#/',
  }
  const historyMock = {
    state: null,
    length: 1,
    replaceState(state) {
      this.state = state
    },
    pushState(state) {
      this.state = state
    },
    go() {},
  }
  const documentMock = {
    querySelector() {
      return null
    },
    addEventListener(type) {
      documentEvents.push(type)
    },
    removeEventListener() {},
  }
  const windowMock = {
    history: historyMock,
    location: locationMock,
    addEventListener(type) {
      windowEvents.push(type)
    },
    removeEventListener() {},
  }

  installDocumentMock(documentMock)
  installWindowMock(windowMock)
  installGlobalMock('location', locationMock)
  installGlobalMock('history', historyMock)

  createQMediaSyncHashHistory()

  assert.deepEqual(documentEvents, [])
  assert.deepEqual(windowEvents, ['popstate'])
})

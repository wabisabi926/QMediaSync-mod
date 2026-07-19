import { readdirSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const assetsDir = resolve(process.argv[2] ?? 'dist/assets')
const assetNames = readdirSync(assetsDir)
const chartChunk = assetNames.find((name) => /^vendor-charts-.+\.js$/.test(name))

if (!chartChunk) {
  // eslint-disable-next-line no-console
  console.log('未生成独立图表分包，无 vendor-charts 循环依赖。')
  process.exit(0)
}

const chartSource = readFileSync(resolve(assetsDir, chartChunk), 'utf8')
const importedChunks = [...chartSource.matchAll(/from["']\.\/([^"']+)["']/g)].map(
  ([, chunkName]) => chunkName,
)

for (const importedChunk of importedChunks) {
  const importedPath = resolve(assetsDir, importedChunk)
  const importedSource = readFileSync(importedPath, 'utf8')

  if (importedSource.includes(`./${chartChunk}`)) {
    throw new Error(`检测到循环依赖：${chartChunk} ↔ ${importedChunk}`)
  }
}

// eslint-disable-next-line no-console
console.log('vendor-charts 不存在循环依赖。')

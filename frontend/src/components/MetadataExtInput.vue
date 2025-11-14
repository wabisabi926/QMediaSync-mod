<template>
  <div class="metadata-ext-input-container">
    <div class="ext-tags-container">
      <el-tag v-for="(tag, index) in tags" :key="index" closable @close="removeTag(index)" class="ext-tag">
        {{ tag }}
      </el-tag>
      <el-button v-if="!showInput" @click="showInput = true" size="small" type="primary" plain class="add-tag-btn">
        + 添加
      </el-button>
    </div>
    <el-input v-model="inputValue" :placeholder="placeholder" @keyup.enter="handleEnter" class="ext-input-full"
      size="small" v-if="showInput">
      <template #append>
        <el-button @click="handleEnter" size="small" type="primary" :disabled="!inputValue.trim()">添加</el-button>
      </template>
    </el-input>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Ref } from 'vue'

// 定义组件属性
interface Props {
  modelValue: string[]
  placeholder?: string
  autoAddDot?: boolean
}

// 定义事件发射
const emit = defineEmits<{
  (e: 'update:modelValue', value: string[]): void
}>()

// 设置默认值
const props = withDefaults(defineProps<Props>(), {
  modelValue: () => [],
  placeholder: '输入扩展名后按回车添加，如：jpg 或 jpg,png,gif',
  autoAddDot: true
});

// 输入框值
const inputValue: Ref<string> = ref('')

// 是否显示输入框
const showInput: Ref<boolean> = ref(false)

// 标签值
const tags: Ref<string[]> = ref([...props.modelValue])

// 监听外部传入的值变化
watch(
  () => props.modelValue,
  (newVal) => {
    tags.value = [...newVal]
  },
  { deep: true }
)

// 处理回车事件
const handleEnter = () => {
  if (!inputValue.value.trim()) {
    showInput.value = false
    return
  }

  // 分割输入的扩展名（支持逗号或分号分隔）
  const exts = inputValue.value
    .split(/[,;]+/)
    .map(ext => ext.trim())
    .filter(ext => ext.length > 0)

  // 处理每个扩展名
  const newTags = [...tags.value]
  exts.forEach(ext => {
    // 根据属性控制是否自动添加点号
    const formattedExt = props.autoAddDot && !ext.startsWith('.') ? `.${ext}` : ext

    // 避免重复添加
    if (!newTags.includes(formattedExt)) {
      newTags.push(formattedExt)
    }
  })

  // 更新标签
  tags.value = newTags
  emit('update:modelValue', newTags)

  // 清空输入框并隐藏输入框
  inputValue.value = ''
  showInput.value = false
}

// 删除标签
const removeTag = (index: number) => {
  tags.value.splice(index, 1)
  emit('update:modelValue', tags.value)
}

// // 失去焦点时隐藏输入框
// const handleBlur = () => {
//   if (!inputValue.value.trim()) {
//     showInput.value = false
//   }
// }

// 在输入框上添加失去焦点事件监听
watch(showInput, (newVal) => {
  if (newVal) {
    // 延迟聚焦到输入框
    setTimeout(() => {
      const input = document.querySelector('.ext-input-full input') as HTMLInputElement | null
      if (input) {
        input.focus()
      }
    }, 0)
  }
})
</script>

<style scoped>
.metadata-ext-input-container {
  width: 100%;
}

.ext-tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  min-height: 32px;
}

.ext-tag {
  margin-bottom: 0;
}

.ext-input-full {
  width: 100%;
  margin-top: 8px;
}

.add-tag-btn {
  margin-left: 0;
}
</style>

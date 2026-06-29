<script setup lang="ts">
import { computed, type Component } from 'vue'

const props = withDefaults(
  defineProps<{
    icon: Component
    label: string
    type?: 'primary' | 'success' | 'warning' | 'danger' | 'info' | ''
    size?: 'small' | 'default' | 'large'
    loading?: boolean
    disabled?: boolean
    isMobile?: boolean
    mobileIconOnly?: boolean
  }>(),
  {
    type: '',
    size: 'default',
    loading: false,
    disabled: false,
    isMobile: false,
    mobileIconOnly: true,
  },
)

const emit = defineEmits<{
  click: [event: MouseEvent]
}>()

const iconOnly = computed(() => props.isMobile && props.mobileIconOnly)

function handleClick(event: MouseEvent) {
  emit('click', event)
}
</script>

<template>
  <el-button
    :type="type || undefined"
    :size="size"
    :loading="loading"
    :disabled="disabled"
    :aria-label="iconOnly ? label : undefined"
    class="responsive-icon-button"
    :class="{ 'responsive-icon-button--icon-only': iconOnly }"
    @click="handleClick"
  >
    <el-icon aria-hidden="true">
      <component :is="icon" />
    </el-icon>
    <span v-if="!iconOnly" class="responsive-icon-button__label">{{ label }}</span>
  </el-button>
</template>

<style scoped>
.responsive-icon-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.responsive-icon-button--icon-only {
  width: 36px;
  height: 36px;
  padding: 0;
}
</style>

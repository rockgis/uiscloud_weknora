<template>
  <div class="model-selector">
    <t-select
      :value="selectedModelId"
      @change="handleModelChange"
      :placeholder="placeholderText"
      :disabled="disabled"
      :loading="loading"
      filterable
      style="width: 100%;"
    >
      <t-option
        v-for="model in models"
        :key="model.id"
        :value="model.id"
        :label="model.name"
      >
        <div class="model-option">
          <t-icon name="check-circle-filled" class="model-icon" />
          <span class="model-name">{{ model.name }}</span>
          <t-tag v-if="model.is_builtin" size="small" theme="primary">내장</t-tag>
          <t-tag v-if="model.is_default" size="small" theme="success">{{ $t('model.defaultTag') }}</t-tag>
        </div>
      </t-option>
      
      <t-option
        v-if="!disabled"
        value="__add_model__"
        class="add-model-option"
      >
        <div class="model-option add">
          <t-icon name="add" class="add-icon" />
          <span class="model-name">{{ $t('model.addModelInSettings') }}</span>
        </div>
      </t-option>
    </t-select>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { listModels, type ModelConfig } from '@/api/model'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'

interface Props {
  modelType: 'KnowledgeQA' | 'Embedding' | 'Rerank' | 'VLLM'
  selectedModelId?: string
  disabled?: boolean
  placeholder?: string
  allModels?: ModelConfig[]
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  placeholder: ''
})

const emit = defineEmits<{
  'update:selectedModelId': [value: string]
  'add-model': []
}>()

const models = ref<ModelConfig[]>([])
const loading = ref(false)
const { t } = useI18n()

const placeholderText = computed(() => {
  return props.placeholder || t('model.selectModelPlaceholder')
})

watch(() => props.allModels, (newModels) => {
  if (newModels && Array.isArray(newModels)) {
    models.value = newModels.filter(m => m.type === props.modelType)
  }
}, { immediate: true })

const selectedModel = computed(() => {
  if (!props.selectedModelId) return null
  return models.value.find(m => m.id === props.selectedModelId)
})

const loadModels = async () => {
  if (props.allModels) {
    return
  }
  
  loading.value = true
  try {
    const result = await listModels()
    if (result && Array.isArray(result)) {
      models.value = result.filter(m => m.type === props.modelType)
    } else {
      models.value = []
    }
  } catch (error) {
    console.error(t('model.loadFailed'), error)
    MessagePlugin.error(t('model.loadFailed'))
    models.value = []
  } finally {
    loading.value = false
  }
}

const handleModelChange = (value: string) => {
  if (value === '__add_model__') {
    emit('add-model')
    return
  }
  emit('update:selectedModelId', value)
}

defineExpose({
  refresh: loadModels
})

onMounted(() => {
  if (!props.allModels) {
    loadModels()
  }
})
</script>

<style lang="less" scoped>
.model-selector {
  width: 100%;
}

.model-option {
  display: flex;
  align-items: center;
  gap: 8px;
  
  .model-icon {
    font-size: 14px;
    color: #07C05F;
  }
  
  .add-icon {
    font-size: 14px;
    color: #07C05F;
  }
  
  .model-name {
    flex: 1;
    font-size: 13px;
  }
  
  &.add {
    .model-name {
      color: #07C05F;
      font-weight: 500;
    }
  }
}
</style>


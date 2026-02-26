<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="dialogVisible" class="model-editor-overlay" @click.self="handleCancel">
        <div class="model-editor-modal">
          <button class="close-btn" @click="handleCancel" :aria-label="$t('common.close')">
            <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
              <path d="M15 5L5 15M5 5L15 15" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            </svg>
          </button>

          <div class="modal-header">
            <h2 class="modal-title">{{ isEdit ? $t('model.editor.editTitle') : $t('model.editor.addTitle') }}</h2>
            <p class="modal-desc">{{ getModalDescription() }}</p>
          </div>

          <div class="modal-body">
            <t-form ref="formRef" :data="formData" :rules="rules" layout="vertical">
        <div class="form-item">
          <label class="form-label required">{{ $t('model.editor.sourceLabel') }}</label>
          <t-radio-group v-model="formData.source">
            <t-radio
              value="local"
              :disabled="ollamaServiceStatus === false"
            >
              {{ $t('model.editor.sourceLocal') }}
            </t-radio>
            <t-radio value="remote">{{ $t('model.editor.sourceRemote') }}</t-radio>
          </t-radio-group>

          <div v-if="ollamaServiceStatus === false" class="ollama-unavailable-tip">
            <t-icon name="error-circle-filled" class="tip-icon" />
            <span class="tip-text">{{ $t('model.editor.ollamaUnavailable') }}</span>
            <t-button
              variant="text"
              size="small"
              theme="primary"
              @click="goToOllamaSettings"
              class="tip-link"
            >
              {{ $t('model.editor.goToOllamaSettings') }}
            </t-button>
          </div>
        </div>

        <div v-if="formData.source === 'local'" class="form-item">
          <label class="form-label required">{{ $t('model.modelName') }}</label>
          <div class="model-select-row">
            <t-select
              v-model="formData.modelName"
              :loading="loadingOllamaModels"
              :class="{ 'downloading': downloading }"
              :style="downloading ? `--progress: ${downloadProgress}%` : ''"
              filterable
              :filter="handleModelFilter"
              :placeholder="$t('model.searchPlaceholder')"
              @focus="loadOllamaModels"
              @visible-change="handleDropdownVisibleChange"
            >
              <t-option
                v-for="model in filteredOllamaModels"
                :key="model.name"
                :value="model.name"
                :label="model.name"
              >
                <div class="model-option">
                  <t-icon name="check-circle-filled" class="downloaded-icon" />
                  <span class="model-name">{{ model.name }}</span>
                  <span class="model-size">{{ formatModelSize(model.size) }}</span>
                </div>
              </t-option>
              
              <t-option
                v-if="showDownloadOption"
                :value="`__download__${searchKeyword}`"
                :label="$t('model.editor.downloadLabel', { keyword: searchKeyword })"
                class="download-option"
              >
                <div class="model-option download">
                  <t-icon name="download" class="download-icon" />
                  <span class="model-name">{{ $t('model.editor.downloadLabel', { keyword: searchKeyword }) }}</span>
                </div>
              </t-option>
              
              <template v-if="downloading" #suffix>
                <div class="download-suffix">
                  <t-icon name="loading" class="spinning" />
                  <span class="progress-text">{{ downloadProgress.toFixed(1) }}%</span>
                </div>
              </template>
            </t-select>
            
            <t-button
              variant="text"
              size="small"
              :loading="loadingOllamaModels"
              @click="refreshOllamaModels"
              class="refresh-btn"
            >
              <t-icon name="refresh" />
              {{ $t('model.editor.refreshList') }}
            </t-button>
          </div>
        </div>

        <div v-else class="form-item">
          <label class="form-label required">{{ $t('model.modelName') }}</label>
          <t-input 
            v-model="formData.modelName" 
            :placeholder="getModelNamePlaceholder()"
          />
        </div>

        <template v-if="formData.source === 'remote'">
          <div class="form-item">
            <label class="form-label required">{{ $t('model.editor.baseUrlLabel') }}</label>
            <t-input 
              v-model="formData.baseUrl" 
              :placeholder="getBaseUrlPlaceholder()"
            />
          </div>

          <div class="form-item">
            <label class="form-label">{{ $t('model.editor.apiKeyOptional') }}</label>
            <t-input 
              v-model="formData.apiKey" 
              type="password"
              :placeholder="$t('model.editor.apiKeyPlaceholder')"
            />
          </div>

          <div class="form-item">
            <label class="form-label">{{ $t('model.editor.connectionTest') }}</label>
            <div class="api-test-section">
              <t-button 
                variant="outline" 
                @click="checkRemoteAPI"
                :loading="checking"
                :disabled="!formData.modelName || !formData.baseUrl"
              >
                <template #icon>
                  <t-icon 
                    v-if="!checking && remoteChecked && remoteAvailable"
                    name="check-circle-filled" 
                    class="status-icon available"
                  />
                  <t-icon 
                    v-else-if="!checking && remoteChecked && !remoteAvailable"
                    name="close-circle-filled" 
                    class="status-icon unavailable"
                  />
                </template>
                {{ checking ? $t('model.editor.testing') : $t('model.editor.testConnection') }}
              </t-button>
              <span v-if="remoteChecked" :class="['test-message', remoteAvailable ? 'success' : 'error']">
                {{ remoteMessage }}
              </span>
            </div>
          </div>
        </template>

        <div v-if="modelType === 'embedding'" class="form-item">
          <label class="form-label">{{ $t('model.editor.dimensionLabel') }}</label>
          <div class="dimension-control">
            <t-input 
              v-model.number="formData.dimension" 
              type="number"
            :min="128"
            :max="4096"
            :placeholder="$t('model.editor.dimensionPlaceholder')"
              :disabled="formData.source === 'local' && checking"
            />
            <t-button 
              v-if="formData.source === 'local' && formData.modelName"
              variant="outline"
              size="small"
              :loading="checking"
              @click="checkOllamaDimension"
              class="dimension-check-btn"
            >
              <t-icon name="refresh" />
              {{ $t('model.editor.checkDimension') }}
            </t-button>
          </div>
          <p v-if="dimensionChecked && dimensionMessage" class="dimension-hint" :class="{ success: dimensionSuccess }">
            {{ dimensionMessage }}
          </p>
        </div>

      </t-form>
          </div>

          <div class="modal-footer">
            <t-button theme="default" variant="outline" @click="handleCancel">
              {{ $t('common.cancel') }}
            </t-button>
            <t-button theme="primary" @click="handleConfirm" :loading="saving">
              {{ $t('common.save') }}
            </t-button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, computed, onUnmounted, nextTick } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { checkOllamaModels, checkRemoteModel, testEmbeddingModel, checkRerankModel, listOllamaModels, downloadOllamaModel, getDownloadProgress, checkOllamaStatus, type OllamaModelInfo } from '@/api/initialization'
import { useI18n } from 'vue-i18n'
import { useUIStore } from '@/stores/ui'

interface ModelFormData {
  id: string
  name: string
  source: 'local' | 'remote'
  modelName: string
  baseUrl?: string
  apiKey?: string
  dimension?: number
  interfaceType?: 'ollama' | 'openai'
  isDefault: boolean
}

interface Props {
  visible: boolean
  modelType: 'chat' | 'embedding' | 'rerank' | 'vllm'
  modelData?: ModelFormData | null
}

const { t } = useI18n()
const uiStore = useUIStore()

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  modelData: null
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'confirm': [data: ModelFormData]
}>()

const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})

const isEdit = computed(() => !!props.modelData)

const formRef = ref()
const saving = ref(false)
const modelChecked = ref(false)
const modelAvailable = ref(false)
const checking = ref(false)
const remoteChecked = ref(false)
const remoteAvailable = ref(false)
const remoteMessage = ref('')
const dimensionChecked = ref(false)
const dimensionSuccess = ref(false)
const dimensionMessage = ref('')

const ollamaModelList = ref<OllamaModelInfo[]>([])
const loadingOllamaModels = ref(false)
const searchKeyword = ref('')
const downloading = ref(false)
const downloadProgress = ref(0)
const currentDownloadModel = ref('')
let downloadInterval: any = null

const ollamaServiceStatus = ref<boolean | null>(null)
const checkingOllamaStatus = ref(false)

const formData = ref<ModelFormData>({
  id: '',
  name: '',
  source: 'local',
  modelName: '',
  baseUrl: '',
  apiKey: '',
  dimension: undefined,
  interfaceType: 'ollama',
  isDefault: false
})

const rules = computed(() => ({
  modelName: [
    { required: true, message: t('model.editor.validation.modelNameRequired') },
    { 
      validator: (val: string) => {
        if (!val || !val.trim()) {
          return { result: false, message: t('model.editor.validation.modelNameEmpty') }
        }
        if (val.trim().length > 100) {
          return { result: false, message: t('model.editor.validation.modelNameMax') }
        }
        return { result: true }
      },
      trigger: 'blur'
    }
  ],
  baseUrl: [
    { 
      required: true, 
      message: t('model.editor.validation.baseUrlRequired'),
      trigger: 'blur'
    },
    {
      validator: (val: string) => {
        if (!val || !val.trim()) {
          return { result: false, message: t('model.editor.validation.baseUrlEmpty') }
        }
        try {
          new URL(val.trim())
          return { result: true }
        } catch {
          return { result: false, message: t('model.editor.validation.baseUrlInvalid') }
        }
      },
      trigger: 'blur'
    }
  ]
}))

const getModalDescription = () => {
  const key = `model.editor.description.${props.modelType}` as const
  return t(key) || t('model.editor.description.default')
}

const getModelNamePlaceholder = () => {
  if (props.modelType === 'vllm') {
    return formData.value.source === 'local'
      ? t('model.editor.modelNamePlaceholder.localVllm')
      : t('model.editor.modelNamePlaceholder.remoteVllm')
  }
  return formData.value.source === 'local'
    ? t('model.editor.modelNamePlaceholder.local')
    : t('model.editor.modelNamePlaceholder.remote')
}

const getBaseUrlPlaceholder = () => {
  return props.modelType === 'vllm'
    ? t('model.editor.baseUrlPlaceholderVllm')
    : t('model.editor.baseUrlPlaceholder')
}

const checkOllamaServiceStatus = async () => {
  console.log('Ollama 서비스 상태 확인 시작...')
  checkingOllamaStatus.value = true
  try {
    const result = await checkOllamaStatus()
    ollamaServiceStatus.value = result.available
    console.log('Ollama 서비스 상태 확인 완료:', result.available)
  } catch (error) {
    console.error('Ollama 서비스 상태 확인 실패:', error)
    ollamaServiceStatus.value = false
  } finally {
    checkingOllamaStatus.value = false
  }
}

const goToOllamaSettings = async () => {
  console.log('Ollama 설정으로 이동 버튼 클릭')
  emit('update:visible', false)
  
  if (uiStore.showSettingsModal) {
    uiStore.closeSettings()
    await nextTick()
  }
  
  console.log('uiStore.openSettings 호출')
  uiStore.openSettings('ollama')
  console.log('uiStore.openSettings 호출 완료')
}

watch(() => props.visible, (val) => {
  if (val) {
    document.body.style.overflow = 'hidden'

    checkOllamaServiceStatus()

    if (props.modelData) {
      formData.value = { ...props.modelData }
    } else {
      resetForm()
    }
  } else {
    document.body.style.overflow = ''
  }
})

const resetForm = () => {
  formData.value = {
    id: generateId(),
    name: '',
    source: 'local',
    modelName: '',
    baseUrl: '',
    apiKey: '',
    dimension: undefined,
    interfaceType: undefined,
    isDefault: false
  }
  modelChecked.value = false
  modelAvailable.value = false
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
}


const generateId = () => {
  return `model_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
}

const filteredOllamaModels = computed(() => {
  if (!searchKeyword.value) return ollamaModelList.value
  return ollamaModelList.value.filter(model => 
    model.name.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

const showDownloadOption = computed(() => {
  if (!searchKeyword.value.trim()) return false
  const exists = ollamaModelList.value.some(model => 
    model.name.toLowerCase() === searchKeyword.value.toLowerCase()
  )
  return !exists
})

const handleModelFilter = (filterWords: string) => {
  searchKeyword.value = filterWords
  return true
}

const loadOllamaModels = async () => {
  if (formData.value.source !== 'local') return
  
  loadingOllamaModels.value = true
  try {
    const models = await listOllamaModels()
    ollamaModelList.value = models
  } catch (error) {
    console.error(t('model.editor.loadModelListFailed'), error)
    MessagePlugin.error(t('model.editor.loadModelListFailed'))
  } finally {
    loadingOllamaModels.value = false
  }
}

const refreshOllamaModels = async () => {
  ollamaModelList.value = []
  await loadOllamaModels()
  MessagePlugin.success(t('model.editor.listRefreshed'))
}

const handleDropdownVisibleChange = (visible: boolean) => {
  if (!visible) {
    searchKeyword.value = ''
  }
}

const formatModelSize = (bytes: number): string => {
  if (!bytes || bytes === 0) return ''
  const gb = bytes / (1024 * 1024 * 1024)
  return gb >= 1 ? `${gb.toFixed(1)} GB` : `${(bytes / (1024 * 1024)).toFixed(0)} MB`
}

const checkModelStatus = async () => {
  if (!formData.value.modelName || formData.value.source !== 'local') {
    return
  }
  
  try {
    const result = await checkOllamaModels([formData.value.modelName])
    modelChecked.value = true
    modelAvailable.value = result.models[formData.value.modelName] || false
  } catch (error) {
    console.error('모델 상태 확인 실패:', error)
    modelChecked.value = false
    modelAvailable.value = false
  }
}

const checkOllamaDimension = async () => {
  if (!formData.value.modelName || formData.value.source !== 'local' || props.modelType !== 'embedding') {
    return
  }
  
  checking.value = true
  dimensionChecked.value = false
  dimensionMessage.value = ''
  
  try {
    const result = await testEmbeddingModel({
      source: 'local',
      modelName: formData.value.modelName,
      dimension: formData.value.dimension
    })
    
    dimensionChecked.value = true
    dimensionSuccess.value = result.available || false
    
    if (result.available && result.dimension) {
      formData.value.dimension = result.dimension
      dimensionMessage.value = t('model.editor.dimensionDetected', { value: result.dimension })
      MessagePlugin.success(dimensionMessage.value)
    } else {
      dimensionMessage.value = result.message || t('model.editor.dimensionFailed')
      MessagePlugin.warning(dimensionMessage.value)
    }
  } catch (error: any) {
    console.error('Ollama 모델 차원 감지 실패:', error)
    dimensionChecked.value = true
    dimensionSuccess.value = false
    dimensionMessage.value = error.message || t('model.editor.dimensionFailed')
    MessagePlugin.error(dimensionMessage.value)
  } finally {
    checking.value = false
  }
}

const checkRemoteAPI = async () => {
  if (!formData.value.modelName || !formData.value.baseUrl) {
    MessagePlugin.warning(t('model.editor.fillModelAndUrl'))
    return
  }
  
  checking.value = true
  remoteChecked.value = false
  remoteMessage.value = ''
  
  try {
    let result: any
    
    switch (props.modelType) {
      case 'chat':
        result = await checkRemoteModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl,
          apiKey: formData.value.apiKey || ''
        })
        break
        
      case 'embedding':
        result = await testEmbeddingModel({
          source: 'remote',
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl,
          apiKey: formData.value.apiKey || '',
          dimension: formData.value.dimension
        })
        if (result.available && result.dimension) {
          formData.value.dimension = result.dimension
        MessagePlugin.info(t('model.editor.remoteDimensionDetected', { value: result.dimension }))
        }
        break
        
      case 'rerank':
        result = await checkRerankModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl,
          apiKey: formData.value.apiKey || ''
        })
        break
        
      case 'vllm':
        result = await checkRemoteModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl,
          apiKey: formData.value.apiKey || ''
        })
        break
        
      default:
        MessagePlugin.error(t('model.editor.unsupportedModelType'))
        return
    }
    
    remoteChecked.value = true
    remoteAvailable.value = result.available || false
    remoteMessage.value = result.message || (result.available ? t('model.editor.connectionSuccess') : t('model.editor.connectionFailed'))
    
    if (result.available) {
      MessagePlugin.success(remoteMessage.value)
    } else {
      MessagePlugin.error(remoteMessage.value)
    }
  } catch (error: any) {
    console.error('Remote API 검증 실패:', error)
    remoteChecked.value = true
    remoteAvailable.value = false
    remoteMessage.value = error.message || t('model.editor.connectionConfigError')
    MessagePlugin.error(remoteMessage.value)
  } finally {
    checking.value = false
  }
}

const handleConfirm = async () => {
  try {
    if (!formData.value.modelName || !formData.value.modelName.trim()) {
      MessagePlugin.warning(t('model.editor.validation.modelNameRequired'))
      return
    }
    
    if (formData.value.modelName.trim().length > 100) {
      MessagePlugin.warning(t('model.editor.validation.modelNameMax'))
      return
    }
    
    if (formData.value.source === 'remote') {
      if (!formData.value.baseUrl || !formData.value.baseUrl.trim()) {
        MessagePlugin.warning(t('model.editor.remoteBaseUrlRequired'))
        return
      }
      
      try {
        new URL(formData.value.baseUrl.trim())
      } catch {
        MessagePlugin.warning(t('model.editor.validation.baseUrlInvalid'))
        return
      }
    }
    
    await formRef.value?.validate()
    saving.value = true
    
    if (!formData.value.id) {
      formData.value.id = generateId()
    }
    
    emit('confirm', { ...formData.value })
    dialogVisible.value = false
  } catch (error) {
    console.error('폼 유효성 검사 실패:', error)
  } finally {
    saving.value = false
  }
}

watch(() => formData.value.modelName, async (newValue, oldValue) => {
  if (!newValue) return
  
  if (newValue.startsWith('__download__')) {
  const modelName = newValue.replace('__download__', '')
  
  formData.value.modelName = ''
  
  await startDownload(modelName)
    return
  }
  
  if (props.modelType === 'embedding' && 
      formData.value.source === 'local' && 
      newValue !== oldValue && 
      oldValue !== '') {
    MessagePlugin.info(t('model.editor.dimensionHint'))
  }
})

const startDownload = async (modelName: string) => {
  downloading.value = true
  downloadProgress.value = 0
  currentDownloadModel.value = modelName
  
  try {
    const result = await downloadOllamaModel(modelName)
    const taskId = result.taskId
    
    MessagePlugin.success(t('model.editor.downloadStarted', { name: modelName }))
    
    downloadInterval = setInterval(async () => {
      try {
        const progress = await getDownloadProgress(taskId)
        downloadProgress.value = progress.progress
        
        if (progress.status === 'completed') {
          clearInterval(downloadInterval)
          downloadInterval = null
          downloading.value = false
          
          MessagePlugin.success(t('model.editor.downloadCompleted', { name: modelName }))
          
          await loadOllamaModels()
          
          formData.value.modelName = modelName
          
          downloadProgress.value = 0
          currentDownloadModel.value = ''
          
        } else if (progress.status === 'failed') {
          clearInterval(downloadInterval)
          downloadInterval = null
          downloading.value = false
          MessagePlugin.error(progress.message || t('model.editor.downloadFailed', { name: modelName }))
          downloadProgress.value = 0
          currentDownloadModel.value = ''
        }
      } catch (error) {
        console.error('다운로드 진행률 조회 실패:', error)
      }
    }, 1000)
    
  } catch (error: any) {
    downloading.value = false
    downloadProgress.value = 0
    currentDownloadModel.value = ''
    MessagePlugin.error(error.message || t('model.editor.downloadStartFailed'))
  }
}

onUnmounted(() => {
  if (downloadInterval) {
    clearInterval(downloadInterval)
  }
})

watch(() => formData.value.source, () => {
  modelChecked.value = false
  modelAvailable.value = false
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
  
  searchKeyword.value = ''
  if (downloadInterval) {
    clearInterval(downloadInterval)
    downloadInterval = null
  }
  downloading.value = false
  downloadProgress.value = 0
  currentDownloadModel.value = ''
})

watch(() => formData.value.modelName, () => {
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
})

const handleCancel = () => {
  dialogVisible.value = false
}
</script>

<style lang="less" scoped>
.model-editor-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1200;
  backdrop-filter: blur(4px);
  overflow: hidden;
  padding: 20px;
}

.model-editor-modal {
  position: relative;
  width: 100%;
  max-width: 560px;
  max-height: 90vh;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 6px 28px rgba(15, 23, 42, 0.08);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.close-btn {
  position: absolute;
  top: 16px;
  right: 16px;
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #666666;
  transition: all 0.15s ease;
  z-index: 10;

  &:hover {
    background: #f5f5f5;
    color: #333333;
  }
}

.modal-header {
  padding: 24px 24px 16px;
  border-bottom: 1px solid #e5e7eb;
  flex-shrink: 0;
}

.modal-title {
  margin: 0 0 6px 0;
  font-size: 18px;
  font-weight: 600;
  color: #333333;
}

.modal-desc {
  margin: 0;
  font-size: 13px;
  color: #666666;
  line-height: 1.5;
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  background: #ffffff;

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-track {
    background: #f5f5f5;
    border-radius: 3px;
  }

  &::-webkit-scrollbar-thumb {
    background: #d0d0d0;
    border-radius: 3px;
    transition: background 0.15s;

    &:hover {
      background: #b0b0b0;
    }
  }

  :deep(.t-form) {
    .t-form-item {
      display: none;
    }
  }
}

.form-item {
  margin-bottom: 20px;

  &:last-child {
    margin-bottom: 0;
  }
}

.form-label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #333333;

  &.required::after {
    content: '*';
    color: #f56c6c;
    margin-left: 4px;
    font-weight: 600;
  }
}

:deep(.t-input),
:deep(.t-select),
:deep(.t-textarea),
:deep(.t-input-number) {
  width: 100%;
  font-size: 13px;

  .t-input__inner,
  .t-input__wrap,
  input,
  textarea {
    font-size: 13px;
    border-radius: 6px;
    border-color: #d9d9d9;
    transition: all 0.15s ease;
  }

  &:hover .t-input__inner,
  &:hover .t-input__wrap,
  &:hover input,
  &:hover textarea {
    border-color: #b3b3b3;
  }

  &.t-is-focused .t-input__inner,
  &.t-is-focused .t-input__wrap,
  &.t-is-focused input,
  &.t-is-focused textarea {
    border-color: #07C05F;
    box-shadow: 0 0 0 2px rgba(7, 192, 95, 0.1);
  }
}

:deep(.t-radio-group) {
  display: flex;
  gap: 24px;

  .t-radio {
    margin-right: 0;
    font-size: 13px;

    &:hover {
      .t-radio__label {
        color: #07C05F;
      }
    }
  }

  .t-radio__label {
    font-size: 13px;
    color: #333333;
    transition: color 0.15s ease;
  }

  .t-radio__input:checked + .t-radio__label {
    color: #07C05F;
    font-weight: 500;
  }
}

:deep(.t-checkbox) {
  font-size: 13px;

  .t-checkbox__label {
    font-size: 13px;
    color: #333333;
  }
}

.modal-footer {
  padding: 16px 24px;
  border-top: 1px solid #e5e7eb;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  flex-shrink: 0;
  background: #fafafa;

  :deep(.t-button) {
    min-width: 80px;
    height: 36px;
    font-weight: 500;
    font-size: 14px;
    border-radius: 6px;
    transition: all 0.15s ease;

    &.t-button--theme-primary {
      background: #07C05F;
      border-color: #07C05F;

      &:hover {
        background: #06b04d;
        border-color: #06b04d;
      }

      &:active {
        background: #059642;
        border-color: #059642;
      }
    }

    &.t-button--variant-outline {
      color: #666666;
      border-color: #d9d9d9;

      &:hover {
        border-color: #07C05F;
        color: #07C05F;
        background: rgba(7, 192, 95, 0.04);
      }
    }
  }
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;

  .model-editor-modal {
    transition: transform 0.2s ease, opacity 0.2s ease;
  }
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;

  .model-editor-modal {
    transform: scale(0.95);
    opacity: 0;
  }
}

.api-test-section {
  display: flex;
  align-items: center;
  gap: 12px;

  .test-message {
    font-size: 13px;
    line-height: 1.5;
    flex: 1;

    &.success {
      color: #059669;
    }

    &.error {
      color: #f56c6c;
    }
  }

  :deep(.t-button) {
    min-width: 88px;
    height: 32px;
    font-size: 13px;
    border-radius: 6px;
    flex-shrink: 0;
  }

  .status-icon {
    font-size: 16px;
    flex-shrink: 0;

    &.available {
      color: #07C05F;
    }

    &.unavailable {
      color: #f56c6c;
    }
  }
}

.model-option {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 4px 0;
  
  .downloaded-icon {
    font-size: 14px;
    color: #07C05F;
    flex-shrink: 0;
  }
  
  .download-icon {
    font-size: 14px;
    color: #07C05F;
    flex-shrink: 0;
  }
  
  .model-name {
    flex: 1;
    font-size: 13px;
    color: #333333;
  }
  
  .model-size {
    font-size: 12px;
    color: #999999;
    margin-left: auto;
  }
  
  &.download {
    .model-name {
      color: #07C05F;
      font-weight: 500;
    }
  }
}

.download-suffix {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0 4px;
  
  .spinning {
    animation: spin 1s linear infinite;
    font-size: 14px;
    color: #07C05F;
  }
  
  .progress-text {
    font-size: 12px;
    font-weight: 500;
    color: #07C05F;
  }
}

:deep(.t-select.downloading) {
  .t-input {
    position: relative;
    overflow: hidden;
    
    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 0;
      bottom: 0;
      width: var(--progress, 0%);
      background: linear-gradient(90deg, rgba(7, 192, 95, 0.08), rgba(7, 192, 95, 0.15));
      transition: width 0.3s ease;
      z-index: 0;
      border-radius: 5px 0 0 5px;
    }
    
    .t-input__inner,
    input {
      position: relative;
      z-index: 1;
      background: transparent !important;
    }
  }
}

.model-select-row {
  display: flex;
  align-items: center;
  gap: 8px;

  .t-select {
    flex: 1;
  }

  :deep(.t-button) {
    height: 32px;
    font-size: 13px;
    border-radius: 6px;
    flex-shrink: 0;
  }
}

.refresh-btn {
  margin-top: 0;
  font-size: 13px;
  color: #666666;
  flex-shrink: 0;

  &:hover {
    color: #07C05F;
    background: rgba(7, 192, 95, 0.04);
  }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.dimension-control {
  display: flex;
  align-items: center;
  gap: 8px;

  :deep(.t-input) {
    flex: 1;
  }
}

.dimension-check-btn {
  flex-shrink: 0;
  font-size: 12px;
}

.dimension-hint {
  margin: 8px 0 0 0;
  font-size: 13px;
  line-height: 1.5;
  color: #e34d59;

  &.success {
    color: #07C05F;
  }
}

.ollama-unavailable-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 10px 12px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 6px;
  font-size: 13px;

  .tip-icon {
    color: #f56c6c;
    font-size: 16px;
    flex-shrink: 0;
    margin-right: 2px;
  }

  .tip-text {
    color: #dc2626;
    flex: 1;
    line-height: 1.5;
  }

  :deep(.tip-link) {
    color: #07C05F;
    font-size: 13px;
    font-weight: 500;
    padding: 4px 6px 4px 10px !important;
    min-height: auto !important;
    height: auto !important;
    line-height: 1.4 !important;
    text-decoration: none;
    white-space: nowrap;
    display: inline-flex !important;
    align-items: center !important;
    gap: 1px;
    border-radius: 4px;
    transition: all 0.2s ease;

    &:hover {
      background: rgba(7, 192, 95, 0.08) !important;
      color: #05a04f !important;
    }

    &:active {
      background: rgba(7, 192, 95, 0.12) !important;
    }

    .t-icon {
      font-size: 14px !important;
      margin: 0 !important;
      line-height: 1 !important;
      display: inline-flex !important;
      align-items: center !important;
    }
  }
}
</style>


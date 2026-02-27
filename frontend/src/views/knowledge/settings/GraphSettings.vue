<template>
  <div class="graph-settings">
    <div class="section-header">
      <h2>{{ t('graphSettings.title') }}</h2>
      <p class="section-description">{{ t('graphSettings.description') }}</p>
      
      <!-- Warning message when graph database is not enabled -->
      <t-alert
        v-if="!isGraphDatabaseEnabled"
        theme="warning"
        style="margin-top: 16px;"
      >
        <template #message>
          <div>{{ t('graphSettings.disabledWarning') }}</div>
          <t-link class="graph-guide-link" theme="primary" @click="handleOpenGraphGuide">
            {{ t('graphSettings.howToEnable') }}
          </t-link>
        </template>
      </t-alert>
    </div>

    <div v-if="isGraphDatabaseEnabled" class="settings-group">
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ t('graphSettings.enableLabel') }}</label>
          <p class="desc">{{ t('graphSettings.enableDescription') }}</p>
        </div>
        <div class="setting-control">
          <t-switch
            v-model="localGraphExtract.enabled"
            @change="handleEnabledChange"
          />
        </div>
      </div>

      <div v-if="localGraphExtract.enabled" class="setting-row vertical">
        <div class="setting-info">
          <label>{{ t('graphSettings.tagsLabel') }}</label>
          <p class="desc">{{ t('graphSettings.tagsDescription') }}</p>
        </div>
        <div class="setting-control full-width">
          <div class="tags-control-group">
            <t-button
              theme="default"
              size="medium"
              :disabled="!modelStatus.llm.available"
              :loading="tagFabring"
              @click="handleFabriTag"
              class="gen-tags-btn"
            >
              {{ t('graphSettings.generateRandomTags') }}
            </t-button>
            <t-select
              v-model="localGraphExtract.tags"
              multiple
              :placeholder="t('graphSettings.tagsPlaceholder')"
              clearable
              creatable
              filterable
              @change="handleTagsChange"
              style="flex: 1; min-width: 400px;"
            />
          </div>
          <div v-if="!modelStatus.llm.available" class="control-tip">
            <t-icon name="info-circle" class="tip-icon" />
            <span>{{ t('graphSettings.completeModelConfig') }}</span>
          </div>
        </div>
      </div>

      <div v-if="localGraphExtract.enabled" class="setting-row vertical">
        <div class="setting-info">
          <label>{{ t('graphSettings.sampleTextLabel') }}</label>
          <p class="desc">{{ t('graphSettings.sampleTextDescription') }}</p>
        </div>
        <div class="setting-control full-width">
          <div class="text-control-group">
            <t-button
              theme="default"
              size="medium"
              :disabled="!modelStatus.llm.available"
              :loading="textFabring"
              @click="handleFabriText"
              class="gen-text-btn"
            >
              {{ t('graphSettings.generateRandomText') }}
            </t-button>
            <t-textarea
              v-model="localGraphExtract.text"
              :placeholder="t('graphSettings.sampleTextPlaceholder')"
              :autosize="{ minRows: 6, maxRows: 12 }"
              show-word-limit
              maxlength="5000"
              @change="handleTextChange"
              style="width: 100%;"
            />
          </div>
          <div v-if="!modelStatus.llm.available" class="control-tip">
            <t-icon name="info-circle" class="tip-icon" />
            <span>{{ t('graphSettings.completeModelConfig') }}</span>
          </div>
        </div>
      </div>

      <div v-if="localGraphExtract.enabled && localGraphExtract.nodes.length > 0" class="setting-row vertical">
        <div class="setting-info">
          <label>{{ t('graphSettings.entityListLabel') }}</label>
          <p class="desc">{{ t('graphSettings.entityListDescription') }}</p>
        </div>
        <div class="setting-control full-width">
          <div class="node-list">
            <div v-for="(node, nodeIndex) in localGraphExtract.nodes" :key="nodeIndex" class="node-item">
              <div class="node-header">
                <t-icon name="user" class="node-icon" />
                <t-input
                  v-model="node.name"
                  :placeholder="t('graphSettings.nodeNamePlaceholder')"
                  @change="handleNodesChange"
                  class="node-name-input"
                />
                <t-button
                  theme="default"
                  size="small"
                  @click="removeNode(nodeIndex)"
                >
                  <t-icon name="delete" />
                </t-button>
              </div>
              <div class="node-attributes">
                <div v-for="(attribute, attrIndex) in node.attributes" :key="attrIndex" class="attribute-item">
                  <t-input
                    v-model="node.attributes[attrIndex]"
                    :placeholder="t('graphSettings.attributePlaceholder')"
                    @change="handleNodesChange"
                    class="attribute-input"
                  />
                  <t-button
                    theme="default"
                    size="small"
                    @click="removeAttribute(nodeIndex, attrIndex)"
                  >
                    <t-icon name="close" />
                  </t-button>
                </div>
                <t-button
                  theme="default"
                  size="small"
                  @click="addAttribute(nodeIndex)"
                  class="add-attr-btn"
                >
                  {{ t('graphSettings.addAttribute') }}
                </t-button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="localGraphExtract.enabled" class="setting-row">
        <div class="setting-info">
          <label>{{ t('graphSettings.manageEntitiesLabel') }}</label>
          <p class="desc">{{ t('graphSettings.manageEntitiesDescription') }}</p>
        </div>
        <div class="setting-control">
          <t-button
            theme="primary"
            @click="addNode"
          >
            {{ t('graphSettings.addEntity') }}
          </t-button>
        </div>
      </div>

      <div v-if="localGraphExtract.enabled && localGraphExtract.relations.length > 0" class="setting-row vertical">
        <div class="setting-info">
          <label>{{ t('graphSettings.relationListLabel') }}</label>
          <p class="desc">{{ t('graphSettings.relationListDescription') }}</p>
        </div>
        <div class="setting-control full-width">
          <div class="relation-list">
            <div v-for="(relation, index) in localGraphExtract.relations" :key="index" class="relation-item">
              <t-select
                v-model="relation.node1"
                :placeholder="t('graphSettings.selectEntity')"
                @change="handleRelationsChange"
                class="relation-select"
              >
                <t-option
                  v-for="node in localGraphExtract.nodes"
                  :key="node.name"
                  :value="node.name"
                  :label="node.name"
                />
              </t-select>
              <t-icon name="arrow-right" class="relation-arrow" />
              <t-select
                v-model="relation.type"
                :placeholder="t('graphSettings.selectRelationType')"
                clearable
                creatable
                filterable
                @change="handleRelationsChange"
                class="relation-select"
              >
                <t-option
                  v-for="tag in localGraphExtract.tags"
                  :key="tag"
                  :value="tag"
                  :label="tag"
                />
              </t-select>
              <t-icon name="arrow-right" class="relation-arrow" />
              <t-select
                v-model="relation.node2"
                :placeholder="t('graphSettings.selectEntity')"
                @change="handleRelationsChange"
                class="relation-select"
              >
                <t-option
                  v-for="node in localGraphExtract.nodes"
                  :key="node.name"
                  :value="node.name"
                  :label="node.name"
                />
              </t-select>
              <t-button
                theme="default"
                size="small"
                @click="removeRelation(index)"
              >
                <t-icon name="delete" />
              </t-button>
            </div>
          </div>
        </div>
      </div>

      <div v-if="localGraphExtract.enabled" class="setting-row">
        <div class="setting-info">
          <label>{{ t('graphSettings.manageRelationsLabel') }}</label>
          <p class="desc">{{ t('graphSettings.manageRelationsDescription') }}</p>
        </div>
        <div class="setting-control">
          <t-button
            theme="primary"
            @click="addRelation"
          >
            {{ t('graphSettings.addRelation') }}
          </t-button>
        </div>
      </div>

      <div v-if="localGraphExtract.enabled" class="setting-row">
        <div class="setting-info">
          <label>{{ t('graphSettings.extractActionsLabel') }}</label>
          <p class="desc">{{ t('graphSettings.extractActionsDescription') }}</p>
        </div>
        <div class="setting-control">
          <div class="action-buttons">
            <t-button
              theme="primary"
              :disabled="!modelStatus.llm.available || !localGraphExtract.text"
              :loading="extracting"
              @click="handleExtract"
            >
              {{ extracting ? t('graphSettings.extracting') : t('graphSettings.startExtraction') }}
            </t-button>
            <t-button
              theme="default"
              @click="defaultExtractExample"
            >
              {{ t('graphSettings.defaultExample') }}
            </t-button>
            <t-button
              theme="default"
              @click="clearExtractExample"
            >
              {{ t('graphSettings.clearExample') }}
            </t-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, computed } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { extractTextRelations, fabriText, fabriTag, type Node, type Relation, type LLMConfig } from '@/api/initialization'
import { listModels } from '@/api/model'
import { getSystemInfo } from '@/api/system'

const { t } = useI18n()

interface GraphExtractConfig {
  enabled: boolean
  text: string
  tags: string[]
  nodes: Node[]
  relations: Relation[]
}

interface Props {
  graphExtract: GraphExtractConfig
  allModels?: any[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:graphExtract': [value: GraphExtractConfig]
}>()

const localGraphExtract = ref<GraphExtractConfig>({
  ...props.graphExtract,
  nodes: props.graphExtract.nodes || [],
  relations: props.graphExtract.relations || []
})

const tagFabring = ref(false)
const textFabring = ref(false)
const extracting = ref(false)

const modelStatus = ref({
  llm: {
    available: false,
    config: null as LLMConfig | null
  }
})

const systemInfo = ref<any>(null)

const isGraphDatabaseEnabled = computed(() => {
  return systemInfo.value?.graph_database_engine && systemInfo.value.graph_database_engine !== '비활성화'
})

// Watch for prop changes
watch(() => props.graphExtract, (newVal) => {
  localGraphExtract.value = {
    ...newVal,
    nodes: newVal.nodes || [],
    relations: newVal.relations || []
  }
}, { deep: true })

const handleConfigChange = () => {
  emit('update:graphExtract', localGraphExtract.value)
}

const handleEnabledChange = () => {
  if (!localGraphExtract.value.enabled) {
    localGraphExtract.value.text = ''
    localGraphExtract.value.tags = []
    localGraphExtract.value.nodes = []
    localGraphExtract.value.relations = []
  }
  handleConfigChange()
}

const handleTagsChange = () => {
  handleConfigChange()
}

const handleTextChange = () => {
  handleConfigChange()
}

const handleNodesChange = () => {
  handleConfigChange()
}

const handleRelationsChange = () => {
  handleConfigChange()
}

const addNode = () => {
  if (!localGraphExtract.value.nodes) {
    localGraphExtract.value.nodes = []
  }
  localGraphExtract.value.nodes.push({
    name: '',
    attributes: []
  })
  handleNodesChange()
}

const removeNode = (index: number) => {
  localGraphExtract.value.nodes.splice(index, 1)
  handleNodesChange()
}

const addAttribute = (nodeIndex: number) => {
  localGraphExtract.value.nodes[nodeIndex].attributes.push('')
  handleNodesChange()
}

const removeAttribute = (nodeIndex: number, attrIndex: number) => {
  localGraphExtract.value.nodes[nodeIndex].attributes.splice(attrIndex, 1)
  handleNodesChange()
}

const addRelation = () => {
  if (!localGraphExtract.value.relations) {
    localGraphExtract.value.relations = []
  }
  localGraphExtract.value.relations.push({
    node1: '',
    node2: '',
    type: ''
  })
  handleRelationsChange()
}

const removeRelation = (index: number) => {
  localGraphExtract.value.relations.splice(index, 1)
  handleRelationsChange()
}

const handleFabriTag = async () => {
  if (!modelStatus.value.llm.available || !modelStatus.value.llm.config) {
    MessagePlugin.warning(t('graphSettings.completeModelConfig'))
    return
  }
  
  tagFabring.value = true
  try {
    const response = await fabriTag({
      llm_config: modelStatus.value.llm.config
    })
    localGraphExtract.value.tags = response.tags || []
    handleTagsChange()
    MessagePlugin.success(t('graphSettings.tagsGenerated'))
  } catch (error: any) {
    console.error('Failed to generate tags:', error)
    MessagePlugin.error(t('graphSettings.tagsGenerateFailed'))
  } finally {
    tagFabring.value = false
  }
}

const handleFabriText = async () => {
  if (!modelStatus.value.llm.available || !modelStatus.value.llm.config) {
    MessagePlugin.warning(t('graphSettings.completeModelConfig'))
    return
  }
  
  textFabring.value = true
  try {
    const response = await fabriText({
      tags: localGraphExtract.value.tags,
      llm_config: modelStatus.value.llm.config
    })
    localGraphExtract.value.text = response.text || ''
    handleTextChange()
    MessagePlugin.success(t('graphSettings.textGenerated'))
  } catch (error: any) {
    console.error('Failed to generate text:', error)
    MessagePlugin.error(t('graphSettings.textGenerateFailed'))
  } finally {
    textFabring.value = false
  }
}

const handleExtract = async () => {
  if (!modelStatus.value.llm.available || !modelStatus.value.llm.config) {
    MessagePlugin.warning(t('graphSettings.completeModelConfig'))
    return
  }
  
  if (!localGraphExtract.value.text) {
    MessagePlugin.warning(t('graphSettings.pleaseInputText'))
    return
  }
  
  extracting.value = true
  try {
    const response = await extractTextRelations({
      text: localGraphExtract.value.text,
      tags: localGraphExtract.value.tags,
      llm_config: modelStatus.value.llm.config
    })
    localGraphExtract.value.nodes = response.nodes || []
    localGraphExtract.value.relations = response.relations || []
    handleNodesChange()
    MessagePlugin.success(t('graphSettings.extractSuccess'))
  } catch (error: any) {
    console.error('Failed to extract relations:', error)
    MessagePlugin.error(t('graphSettings.extractFailed'))
  } finally {
    extracting.value = false
  }
}

const defaultExtractExample = () => {
  localGraphExtract.value.text = `인공지능(AI)은 인간의 학습, 추론, 문제 해결 능력을 컴퓨터 시스템으로 구현하는 기술입니다. 딥러닝은 AI의 핵심 기술 중 하나로, 인공 신경망을 기반으로 대규모 데이터를 학습합니다. GPT는 OpenAI가 개발한 대형 언어 모델(LLM)로, 자연어 처리 분야에서 혁신적인 성과를 이루었습니다. 트랜스포머 아키텍처는 2017년 구글이 발표한 논문에서 처음 소개되었으며, 현대 NLP 모델의 기반이 되었습니다.`
  localGraphExtract.value.tags = ['개발자', '별칭']
  localGraphExtract.value.nodes = [
    {name: 'AI', attributes: ['인공지능', '기계학습의 상위 개념', '다양한 산업에 활용']},
    {name: '딥러닝', attributes: ['AI의 핵심 기술', '인공 신경망 기반']},
    {name: 'GPT', attributes: ['OpenAI 개발', '대형 언어 모델']},
    {name: '트랜스포머', attributes: ['2017년 구글 발표', '현대 NLP 모델의 기반']}
  ]
  localGraphExtract.value.relations = [
    {node1: 'AI', node2: '딥러닝', type: '포함'},
    {node1: 'GPT', node2: 'OpenAI', type: '개발자'},
    {node1: 'GPT', node2: '트랜스포머', type: '기반 기술'}
  ]
  handleNodesChange()
  MessagePlugin.success(t('graphSettings.exampleLoaded'))
}

const clearExtractExample = () => {
  localGraphExtract.value.text = ''
  localGraphExtract.value.tags = []
  localGraphExtract.value.nodes = []
  localGraphExtract.value.relations = []
  handleNodesChange()
  MessagePlugin.success(t('graphSettings.exampleCleared'))
}

const loadModelStatus = async () => {
  try {
    const models = await listModels()
    
    const llmModels = models.filter((m: any) => m.type === 'KnowledgeQA')
    if (llmModels.length > 0) {
      const llmModel = llmModels[0]
      modelStatus.value.llm.available = true
      modelStatus.value.llm.config = {
        source: llmModel.source,
        model_name: llmModel.name,
        base_url: llmModel.parameters?.base_url || '',
        api_key: llmModel.parameters?.api_key || ''
      }
    }
  } catch (error: any) {
    console.error('Failed to load model status:', error)
  }
}

const loadSystemInfo = async () => {
  try {
    const response = await getSystemInfo()
    systemInfo.value = response.data
  } catch (error: any) {
    console.error('Failed to load system info:', error)
  }
}

const graphGuideUrl =
  import.meta.env.VITE_KG_GUIDE_URL ||
  'https://github.com/rockgis/uiscloud_weknora/blob/main/docs/KnowledgeGraph.md'

// Open guide documentation to show how to enable graph database
const handleOpenGraphGuide = () => {
  window.open(graphGuideUrl, '_blank', 'noopener')
}

onMounted(async () => {
  await Promise.all([
    loadModelStatus(),
    loadSystemInfo()
  ])
})
</script>

<style lang="less" scoped>
.graph-settings {
  width: 100%;
}

.section-header {
  margin-bottom: 32px;

  h2 {
    font-size: 20px;
    font-weight: 600;
    color: #333333;
    margin: 0 0 8px 0;
  }

  .section-description {
    font-size: 14px;
    color: #666666;
    margin: 0;
    line-height: 1.5;
  }
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 20px 0;
  border-bottom: 1px solid #e5e7eb;

  &:last-child {
    border-bottom: none;
  }

  &.vertical {
    flex-direction: column;
    gap: 12px;

    .setting-control {
      width: 100%;
      max-width: 100%;
    }
  }
}

.setting-info {
  flex: 1;
  max-width: 65%;
  padding-right: 24px;

  label {
    font-size: 15px;
    font-weight: 500;
    color: #333333;
    display: block;
    margin-bottom: 4px;
  }

  .desc {
    font-size: 13px;
    color: #666666;
    margin: 0;
    line-height: 1.5;
  }
}

.setting-control {
  flex-shrink: 0;
  min-width: 280px;
  display: flex;
  justify-content: flex-end;
  align-items: center;

  &.full-width {
    width: 100%;
    max-width: 100%;
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
}

.tags-control-group,
.text-control-group {
  display: flex;
  gap: 12px;
  width: 100%;
  align-items: flex-start;
}

.text-control-group {
  flex-direction: column;
}

.control-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #666666;

  .tip-icon {
    color: #0052d9;
  }
}

.node-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
}

.node-item {
  background: #f8fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 16px;
}

.node-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;

  .node-icon {
    font-size: 20px;
    color: #0052d9;
  }

  .node-name-input {
    flex: 1;
  }
}

.node-attributes {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-left: 32px;
}

.attribute-item {
  display: flex;
  gap: 8px;
  align-items: center;

  .attribute-input {
    flex: 1;
  }
}

.add-attr-btn {
  align-self: flex-start;
}

.relation-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}

.relation-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #f8fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;

  .relation-select {
    flex: 1;
    min-width: 150px;
  }

  .relation-arrow {
    color: #666666;
    font-size: 16px;
  }
}

.action-buttons {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
</style>
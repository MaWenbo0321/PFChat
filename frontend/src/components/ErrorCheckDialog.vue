<template>
  <el-dialog
      v-model="visible"
      :title="$t('chat.errorDetectedTitle')"
      width="650px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      @close="handleCancel"
  >
    <div class="error-check-dialog">
      <!-- 错误类型提示 -->
      <el-alert
          :title="$t('chat.errorDetected')"
          type="warning"
          :closable="false"
          show-icon
          class="error-alert"
      >
        <template #default>
          <div class="error-type">
            <strong>{{ $t('chat.errorType') }}:</strong>
            <el-tag
                :type="errorData.error_type === '语言语用失误' ? 'danger' : 'warning'"
                size="large"
                style="margin-left: 10px;"
            >
              {{ errorData.error_type === '语言语用失误'
                ? $t('grammar.errorType1')
                : $t('grammar.errorType2') }}
            </el-tag>
          </div>
        </template>
      </el-alert>

      <!-- 原始消息 -->
      <div class="section">
        <div class="section-header">
          <el-icon color="#f56c6c"><WarningFilled /></el-icon>
          <span>{{ $t('chat.originalMessage') }}</span>
        </div>
        <div class="original-message">
          {{ errorData.original_content }}
        </div>
      </div>

      <!-- 建议修改 -->
      <div class="section" v-if="errorData.suggestion">
        <div class="section-header">
          <el-icon color="#67c23a"><Check /></el-icon>
          <span>{{ $t('chat.suggestion') }}</span>
        </div>
        <div class="suggestion-box">
          <div class="suggestion-text">{{ errorData.suggestion }}</div>
          <el-button
              size="small"
              type="primary"
              text
              @click="applySuggestion"
              :icon="CopyDocument"
          >
            {{ $t('chat.applySuggestion') }}
          </el-button>
        </div>
      </div>

      <!-- 错误说明 -->
      <div class="section">
        <div class="section-header">
          <el-icon color="#409eff"><InfoFilled /></el-icon>
          <span>{{ $t('chat.explanation') }}</span>
        </div>
        <div class="explanation-text">
          {{ errorData.explanation }}
        </div>
      </div>

      <!-- 可编辑的消息框 -->
      <div class="section">
        <div class="section-header">
          <el-icon color="#909399"><Edit /></el-icon>
          <span>{{ $t('chat.editMessage') }}</span>
        </div>
        <el-input
            v-model="editedContent"
            type="textarea"
            :rows="4"
            :placeholder="$t('chat.editPlaceholder')"
            maxlength="500"
            show-word-limit
        />
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleCancel">
          {{ $t('common.cancel') }}
        </el-button>
        <el-button
            type="warning"
            @click="sendOriginal"
        >
          {{ $t('chat.sendOriginal') }}
        </el-button>
        <el-button
            type="primary"
            @click="sendEdited"
            :disabled="!editedContent.trim()"
        >
          {{ $t('chat.sendEdited') }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { WarningFilled, Check, InfoFilled, Edit, CopyDocument } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  errorData: {
    type: Object,
    default: () => ({
      original_content: '',
      suggestion: '',
      explanation: '',
      error_type: ''
    })
  }
})

const emit = defineEmits(['update:modelValue', 'send-original', 'send-edited', 'cancel'])

const visible = ref(false)
const editedContent = ref('')

watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val) {
    // 对话框打开时，初始化编辑内容为原始消息
    editedContent.value = props.errorData.original_content
  }
})

watch(visible, (val) => {
  emit('update:modelValue', val)
})

// 应用建议
const applySuggestion = () => {
  editedContent.value = props.errorData.suggestion
}

// 发送原始消息
const sendOriginal = () => {
  emit('send-original', props.errorData.original_content)
  visible.value = false
}

// 发送编辑后的消息
const sendEdited = () => {
  if (editedContent.value.trim()) {
    emit('send-edited', editedContent.value.trim())
    visible.value = false
  }
}

// 取消
const handleCancel = () => {
  emit('cancel')
  visible.value = false
}
</script>

<style scoped>
.error-check-dialog {
  padding: 10px 0;
}

.error-alert {
  margin-bottom: 20px;
}

.error-type {
  display: flex;
  align-items: center;
}

.section {
  margin-bottom: 20px;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  margin-bottom: 10px;
  color: #606266;
  font-size: 14px;
}

.original-message {
  padding: 12px;
  background: #fef0f0;
  border-left: 3px solid #f56c6c;
  border-radius: 4px;
  color: #606266;
  line-height: 1.6;
  word-break: break-word;
}

.suggestion-box {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 10px;
  padding: 12px;
  background: #f0f9ff;
  border-left: 3px solid #67c23a;
  border-radius: 4px;
}

.suggestion-text {
  flex: 1;
  color: #606266;
  line-height: 1.6;
  word-break: break-word;
}

.explanation-text {
  padding: 12px;
  background: #f4f4f5;
  border-left: 3px solid #409eff;
  border-radius: 4px;
  color: #606266;
  line-height: 1.6;
  word-break: break-word;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

:deep(.el-dialog__body) {
  padding: 20px 25px;
}

:deep(.el-textarea__inner) {
  font-family: inherit;
  line-height: 1.6;
}
</style>
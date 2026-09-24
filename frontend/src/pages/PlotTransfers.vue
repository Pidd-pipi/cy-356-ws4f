<template>
  <div class="page-card">
    <div style="display: flex; justify-content: space-between; align-items: center">
      <h3 class="page-title">地块转交申请{{ isAdmin ? '（管理员审核）' : '（我的申请）' }}</h3>
      <el-select v-model="statusFilter" placeholder="全部状态" clearable style="width: 140px" @change="fetch">
        <el-option v-for="(m, k) in TransferStatusMeta" :key="k" :label="m.label" :value="k" />
      </el-select>
    </div>

    <DataTable :data="store.requests" :loading="store.loading" :total="store.total" :page-size="pagination.size.value" :current-page="pagination.page.value" @update:current-page="onPage">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column label="地块" min-width="130">
        <template #default="{ row }">{{ row.plot?.name }}（{{ row.plot?.code }}）</template>
      </el-table-column>
      <el-table-column v-if="isAdmin" label="申请人" width="110">
        <template #default="{ row }">{{ row.applicant?.nickname || row.applicant?.username || '-' }}</template>
      </el-table-column>
      <el-table-column prop="reason" label="申请原因" min-width="180" show-overflow-tooltip />
      <el-table-column label="状态" width="100">
        <template #default="{ row }"><StatusBadge :value="row.status" :meta-map="TransferStatusMeta" /></template>
      </el-table-column>
      <el-table-column label="处理意见" min-width="160" show-overflow-tooltip>
        <template #default="{ row }">{{ row.review_comment || '-' }}</template>
      </el-table-column>
      <el-table-column label="申请时间" width="160">
        <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="230">
        <template #default="{ row }">
          <el-button size="small" @click="openDetail(row)">详情</el-button>
          <template v-if="row.status === 'pending'">
            <template v-if="isAdmin">
              <el-button type="success" size="small" @click="openReview(row, 'approve')">核准</el-button>
              <el-button type="danger" size="small" @click="openReview(row, 'reject')">驳回</el-button>
            </template>
            <el-button v-else type="warning" size="small" @click="cancel(row)">撤回</el-button>
          </template>
        </template>
      </el-table-column>
    </DataTable>

    <el-dialog v-model="detailVisible" title="转交申请详情" width="520px">
      <el-descriptions v-if="current" :column="1" border>
        <el-descriptions-item label="地块">{{ current.plot?.name }}（{{ current.plot?.code }}）</el-descriptions-item>
        <el-descriptions-item label="申请人">{{ current.applicant?.nickname || current.applicant?.username }}</el-descriptions-item>
        <el-descriptions-item label="申请原因">{{ current.reason }}</el-descriptions-item>
        <el-descriptions-item label="当前状态">
          <StatusBadge :value="current.status" :meta-map="TransferStatusMeta" />
        </el-descriptions-item>
        <el-descriptions-item label="处理意见">{{ current.review_comment || '-' }}</el-descriptions-item>
        <el-descriptions-item label="处理人">{{ current.reviewer?.nickname || current.reviewer?.username || '-' }}</el-descriptions-item>
        <el-descriptions-item label="申请时间">{{ formatDateTime(current.created_at) }}</el-descriptions-item>
        <el-descriptions-item label="处理时间">{{ current.reviewed_at || '-' }}</el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="reviewVisible" :title="reviewAction === 'approve' ? '核准转交申请' : '驳回转交申请'" width="480px">
      <el-alert v-if="reviewAction === 'approve'" type="warning" :closable="false" style="margin-bottom: 12px"
        title="核准后原认养关系将被清空，地块立即回到共享池" />
      <el-form label-width="90px">
        <el-form-item :label="reviewAction === 'approve' ? '处理意见' : '驳回意见'" required>
          <el-input v-model="reviewComment" type="textarea" :rows="3"
            :placeholder="reviewAction === 'approve' ? '可填写核准说明（可选）' : '请写明驳回原因（必填）'" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="reviewVisible = false">取消</el-button>
        <el-button :type="reviewAction === 'approve' ? 'success' : 'danger'" :loading="reviewing" @click="submitReview">
          {{ reviewAction === 'approve' ? '确认核准' : '确认驳回' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { usePlotTransferStore } from '@/stores/plotTransfer'
import type { TransferRequest } from '@/api/plotTransfer'
import { useAuth } from '@/hooks/useAuth'
import { usePagination } from '@/hooks/usePagination'
import DataTable from '@/components/DataTable.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { TransferStatusMeta } from '@/constants'
import { formatDateTime } from '@/utils/format'

const store = usePlotTransferStore()
const pagination = usePagination()
const { isAdmin } = useAuth()

const statusFilter = ref('')
const detailVisible = ref(false)
const current = ref<TransferRequest | null>(null)
const reviewVisible = ref(false)
const reviewAction = ref<'approve' | 'reject'>('approve')
const reviewComment = ref('')
const reviewing = ref(false)
const reviewTarget = ref<TransferRequest | null>(null)

async function fetch() {
  await store.fetchRequests({
    page: pagination.page.value,
    page_size: pagination.size.value,
    status: statusFilter.value || undefined
  })
}

function onPage(page: number) {
  pagination.page.value = page
  fetch()
}

function openDetail(row: TransferRequest) {
  current.value = row
  detailVisible.value = true
}

function openReview(row: TransferRequest, action: 'approve' | 'reject') {
  reviewTarget.value = row
  reviewAction.value = action
  reviewComment.value = ''
  reviewVisible.value = true
}

async function submitReview() {
  if (!reviewTarget.value) return
  if (reviewAction.value === 'reject' && reviewComment.value.trim().length < 2) {
    ElMessage.warning('驳回申请必须写明处理意见')
    return
  }
  reviewing.value = true
  try {
    if (reviewAction.value === 'approve') {
      await store.approve(reviewTarget.value.id, reviewComment.value.trim())
      ElMessage.success('已核准，地块回到共享池')
    } else {
      await store.reject(reviewTarget.value.id, reviewComment.value.trim())
      ElMessage.success('已驳回')
    }
    reviewVisible.value = false
    await fetch()
  } finally {
    reviewing.value = false
  }
}

async function cancel(row: TransferRequest) {
  try {
    await ElMessageBox.confirm(`确认撤回地块 ${row.plot?.name} 的转交申请吗？撤回后地块恢复申请前状态。`, '撤回确认', { type: 'warning' })
  } catch {
    return
  }
  await store.cancel(row.id)
  ElMessage.success('转交申请已撤回')
  await fetch()
}

onMounted(fetch)
</script>

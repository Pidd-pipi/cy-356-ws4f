<template>
  <div class="page-card">
    <h3 class="page-title">地块转交申请处理（管理员）</h3>

    <el-radio-group v-model="statusFilter" style="margin-bottom: 12px" @change="onFilterChange">
      <el-radio-button label="">全部</el-radio-button>
      <el-radio-button label="pending">待处理</el-radio-button>
      <el-radio-button label="approved">已核准</el-radio-button>
      <el-radio-button label="rejected">已驳回</el-radio-button>
      <el-radio-button label="withdrawn">已撤回</el-radio-button>
    </el-radio-group>

    <DataTable :data="store.applications" :loading="store.loading" :total="store.total" :page-size="pagination.size.value" :current-page="pagination.page.value" @update:current-page="onPage">
      <el-table-column prop="id" label="单号" width="80" />
      <el-table-column label="地块" min-width="160">
        <template #default="{ row }">
          <router-link :to="`/plots`">{{ row.plot?.code }} {{ row.plot?.name }}</router-link>
        </template>
      </el-table-column>
      <el-table-column label="申请人" width="130">
        <template #default="{ row }">{{ row.applicant?.nickname || row.applicant?.username }}</template>
      </el-table-column>
      <el-table-column prop="reason" label="申请原因" min-width="200" show-overflow-tooltip />
      <el-table-column label="状态" width="100">
        <template #default="{ row }"><StatusBadge :value="row.status" :meta-map="TransferStatusMeta" /></template>
      </el-table-column>
      <el-table-column prop="review_comment" label="处理意见" min-width="180" show-overflow-tooltip>
        <template #default="{ row }">{{ row.review_comment || '-' }}</template>
      </el-table-column>
      <el-table-column label="申请时间" width="170">
        <template #default="{ row }">{{ row.created_at }}</template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <template v-if="row.status === 'pending'">
            <el-button type="success" size="small" @click="approve(row)">核准</el-button>
            <el-button type="danger" size="small" @click="openReject(row)">驳回</el-button>
          </template>
          <span v-else class="done-text">已处理</span>
        </template>
      </el-table-column>
    </DataTable>

    <!-- 驳回必须写明意见 -->
    <el-dialog v-model="rejectVisible" title="驳回转交申请" width="500px">
      <el-form label-width="90px">
        <el-form-item label="申请单号">{{ rejectForm.id }}</el-form-item>
        <el-form-item label="申请原因">{{ rejectForm.reason }}</el-form-item>
        <el-form-item label="处理意见" required>
          <el-input v-model="rejectForm.comment" type="textarea" :rows="4" maxlength="512" show-word-limit
            placeholder="请写明驳回意见，将展示给申请人" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="rejectVisible = false">取消</el-button>
        <el-button type="danger" :loading="processing" @click="submitReject">确认驳回</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useTransferStore } from '@/stores/plotTransfer'
import { usePagination } from '@/hooks/usePagination'
import DataTable from '@/components/DataTable.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { TransferStatusMeta } from '@/constants'
import type { TransferApplicationWithPlot } from '@/api/plotTransfer'

const store = useTransferStore()
const pagination = usePagination()
const statusFilter = ref('pending')

const rejectVisible = ref(false)
const processing = ref(false)
const rejectForm = reactive({ id: 0, reason: '', comment: '' })

async function fetch() {
  await store.fetchApplications({
    page: pagination.page.value,
    page_size: pagination.size.value,
    status: statusFilter.value || undefined
  })
}

function onPage(page: number) {
  pagination.page.value = page
  fetch()
}

function onFilterChange() {
  pagination.page.value = 1
  fetch()
}

// 核准：同意后清空原认养关系，地块回到共享池
async function approve(row: TransferApplicationWithPlot) {
  let comment = ''
  try {
    const { value } = await ElMessageBox.prompt(
      `核准后将解除 ${row.applicant?.nickname || row.applicant?.username} 对地块 ${row.plot?.code || row.plot_id} 的认养关系，地块回到共享池。可填写核准意见：`,
      '核准转交申请',
      { confirmButtonText: '确认核准', cancelButtonText: '取消', inputType: 'textarea', inputValue: '' }
    )
    comment = value || ''
  } catch {
    return
  }
  try {
    await store.review(row.id, true, comment)
    ElMessage.success('已核准，地块已回到共享池')
  } catch {
    // 与撤回并发时后端返回"已经处理"，拦截器已提示，刷新列表即可
  }
  await fetch()
}

function openReject(row: TransferApplicationWithPlot) {
  rejectForm.id = row.id
  rejectForm.reason = row.reason
  rejectForm.comment = ''
  rejectVisible.value = true
}

async function submitReject() {
  if (!rejectForm.comment.trim()) {
    ElMessage.warning('驳回必须填写处理意见')
    return
  }
  processing.value = true
  try {
    await store.review(rejectForm.id, false, rejectForm.comment.trim())
    ElMessage.success('已驳回，原认养关系保留')
    rejectVisible.value = false
    await fetch()
  } finally {
    processing.value = false
  }
}

onMounted(fetch)
</script>

<style scoped>
.done-text { color: #909399; font-size: 12px; }
</style>

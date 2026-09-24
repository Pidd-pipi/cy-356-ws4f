<template>
  <div class="page-card">
    <div style="display: flex; justify-content: space-between; align-items: center">
      <h3 class="page-title">菜园地块认养（GIS 分布）</h3>
      <div>
        <el-button v-if="isAdmin" type="primary" @click="openCreate">+ 新增地块</el-button>
      </div>
    </div>

    <el-card shadow="never" style="margin-bottom: 16px">
      <template #header>地块分布图（按经纬度示意）</template>
      <svg :viewBox="viewBoxStr" class="plot-map" xmlns="http://www.w3.org/2000/svg">
        <rect x="0" y="0" :width="mapW" :height="mapH" fill="#e8f5e9" stroke="#a5d6a7" />
        <line v-for="i in 4" :key="'h' + i" :x1="0" :y1="(mapH / 5) * i" :x2="mapW" :y2="(mapH / 5) * i" stroke="#c8e6c9" stroke-dasharray="4 4" />
        <line v-for="i in 4" :key="'v' + i" :x1="(mapW / 5) * i" :y1="0" :x2="(mapW / 5) * i" :y2="mapH" stroke="#c8e6c9" stroke-dasharray="4 4" />
        <g v-for="p in store.plots" :key="p.id">
          <circle :cx="mapX(p)" :cy="mapY(p)" r="10" :fill="colorOf(p.status)" stroke="#fff" stroke-width="2" />
          <text :x="mapX(p)" :y="mapY(p) - 14" text-anchor="middle" font-size="10" fill="#333">{{ p.code }}</text>
          <title>{{ p.name }}｜{{ PlotStatusMeta[p.status]?.label }}</title>
        </g>
      </svg>
    </el-card>

    <DataTable :data="store.plots" :loading="store.loading" :total="store.total" :page-size="pagination.size.value" :current-page="pagination.page.value" @update:current-page="onPage">
      <el-table-column type="expand">
        <template #default="{ row }">
          <div class="transfer-expand">
            <template v-if="row.latest_transfer">
              <el-descriptions :column="3" border size="small" title="最近一次转交申请">
                <el-descriptions-item label="申请状态">
                  <StatusBadge :value="row.latest_transfer.status" :meta-map="TransferStatusMeta" />
                </el-descriptions-item>
                <el-descriptions-item label="申请人">{{ row.latest_transfer.applicant?.nickname || row.latest_transfer.applicant?.username }}</el-descriptions-item>
                <el-descriptions-item label="申请时间">{{ row.latest_transfer.created_at }}</el-descriptions-item>
                <el-descriptions-item label="申请原因" :span="3">{{ row.latest_transfer.reason }}</el-descriptions-item>
                <el-descriptions-item label="处理意见" :span="2">{{ row.latest_transfer.review_comment || '（暂无）' }}</el-descriptions-item>
                <el-descriptions-item label="处理时间">{{ row.latest_transfer.reviewed_at || '（待处理）' }}</el-descriptions-item>
              </el-descriptions>
            </template>
            <EmptyState description="该地块暂无转交申请记录" />
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="code" label="编号" width="90" />
      <el-table-column prop="name" label="地块名称" min-width="140" />
      <el-table-column label="面积" width="90">
        <template #default="{ row }">{{ formatArea(row.area) }}</template>
      </el-table-column>
      <el-table-column label="土壤" width="90">
        <template #default="{ row }">{{ SoilTypeText[row.soil_type] }}</template>
      </el-table-column>
      <el-table-column label="日照" width="90">
        <template #default="{ row }">{{ SunlightText[row.sunlight] }}</template>
      </el-table-column>
      <el-table-column label="状态" width="110">
        <template #default="{ row }"><StatusBadge :value="row.status" :meta-map="PlotStatusMeta" /></template>
      </el-table-column>
      <el-table-column label="认养人" width="120">
        <template #default="{ row }">{{ row.adopter?.nickname || row.adopter?.username || '-' }}</template>
      </el-table-column>
      <el-table-column label="转交申请" width="100">
        <template #default="{ row }">
          <StatusBadge v-if="row.latest_transfer" :value="row.latest_transfer.status" :meta-map="TransferStatusMeta" />
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openDetail(row)">详情</el-button>
          <el-button v-if="row.status === 'available'" type="success" size="small" @click="adopt(row)">认养</el-button>
          <el-button v-if="canRequestTransfer(row)" type="warning" size="small" @click="openTransfer(row)">申请转交</el-button>
          <el-button v-if="canWithdraw(row)" type="info" size="small" @click="withdraw(row)">撤回申请</el-button>
          <el-button v-if="isAdmin && row.status === 'harvested'" type="danger" size="small" @click="release(row)">强制释放</el-button>
        </template>
      </el-table-column>
    </DataTable>

    <el-dialog v-model="createVisible" title="新增地块（管理员）" width="520px">
      <el-form :model="createForm" label-width="90px">
        <el-form-item label="地块名称"><el-input v-model="createForm.name" /></el-form-item>
        <el-form-item label="地块编号"><el-input v-model="createForm.code" placeholder="如 P-007" /></el-form-item>
        <el-form-item label="面积(m²)"><el-input-number v-model="createForm.area" :min="1" /></el-form-item>
        <el-form-item label="土壤类型">
          <el-select v-model="createForm.soil_type">
            <el-option v-for="(t, k) in SoilTypeText" :key="k" :label="t" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item label="日照条件">
          <el-select v-model="createForm.sunlight">
            <el-option v-for="(t, k) in SunlightText" :key="k" :label="t" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item label="纬度"><el-input-number v-model="createForm.latitude" :precision="4" :step="0.001" /></el-form-item>
        <el-form-item label="经度"><el-input-number v-model="createForm.longitude" :precision="4" :step="0.001" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="createForm.description" type="textarea" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="submitCreate">保存</el-button>
      </template>
    </el-dialog>

    <!-- 转交申请提交弹窗：认养人填写接管理由 -->
    <el-dialog v-model="transferVisible" title="提交转交申请" width="520px">
      <el-alert type="warning" :closable="false" show-icon style="margin-bottom: 12px"
        title="提交后地块进入「转交审核中」，其他居民暂不可认养；管理员核准后才会解除你的认养关系。核准前你可以随时撤回。" />
      <el-form label-width="90px">
        <el-form-item label="地块">
          <span>{{ transferForm.plotName }}（{{ transferForm.plotCode }}）</span>
        </el-form-item>
        <el-form-item label="接管理由" required>
          <el-input v-model="transferForm.reason" type="textarea" :rows="4" maxlength="512" show-word-limit
            placeholder="请填写转交/释放理由，例如：农忙季无暇打理、希望退回共享池由他人接管" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="transferVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitTransfer">提交申请</el-button>
      </template>
    </el-dialog>

    <!-- 地块详情抽屉：保留申请原因、处理意见与当前状态 -->
    <el-drawer v-model="detailVisible" title="地块详情" size="480px">
      <template v-if="detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="编号">{{ detail.code }}</el-descriptions-item>
          <el-descriptions-item label="名称">{{ detail.name }}</el-descriptions-item>
          <el-descriptions-item label="面积">{{ formatArea(detail.area) }}</el-descriptions-item>
          <el-descriptions-item label="土壤类型">{{ SoilTypeText[detail.soil_type] }}</el-descriptions-item>
          <el-descriptions-item label="日照条件">{{ SunlightText[detail.sunlight] }}</el-descriptions-item>
          <el-descriptions-item label="当前状态">
            <StatusBadge :value="detail.status" :meta-map="PlotStatusMeta" />
          </el-descriptions-item>
          <el-descriptions-item label="认养人">{{ detail.adopter?.nickname || detail.adopter?.username || '（共享池）' }}</el-descriptions-item>
          <el-descriptions-item label="描述">{{ detail.description || '（暂无）' }}</el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">转交申请信息</el-divider>
        <template v-if="detail.latest_transfer">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="申请状态">
              <StatusBadge :value="detail.latest_transfer.status" :meta-map="TransferStatusMeta" />
            </el-descriptions-item>
            <el-descriptions-item label="申请人">{{ detail.latest_transfer.applicant?.nickname || detail.latest_transfer.applicant?.username }}</el-descriptions-item>
            <el-descriptions-item label="申请时间">{{ detail.latest_transfer.created_at }}</el-descriptions-item>
            <el-descriptions-item label="申请原因">{{ detail.latest_transfer.reason }}</el-descriptions-item>
            <el-descriptions-item label="处理意见">{{ detail.latest_transfer.review_comment || '（待管理员处理）' }}</el-descriptions-item>
            <el-descriptions-item label="处理人">{{ detail.latest_transfer.reviewer?.nickname || detail.latest_transfer.reviewer?.username || '（待处理）' }}</el-descriptions-item>
            <el-descriptions-item label="处理时间">{{ detail.latest_transfer.reviewed_at || '（待处理）' }}</el-descriptions-item>
          </el-descriptions>
          <div v-if="canWithdraw(detail)" style="margin-top: 12px; text-align: right">
            <el-button type="info" @click="withdraw(detail)">撤回申请</el-button>
          </div>
        </template>
        <EmptyState description="该地块暂无转交申请记录" />
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { usePlotStore } from '@/stores/plot'
import { createPlot, releasePlot, getPlot, type Plot } from '@/api/plot'
import { submitTransfer as apiSubmitTransfer, withdrawTransfer } from '@/api/plotTransfer'
import { useAuth } from '@/hooks/useAuth'
import { usePagination } from '@/hooks/usePagination'
import DataTable from '@/components/DataTable.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import EmptyState from '@/components/EmptyState.vue'
import { PlotStatusMeta, TransferStatusMeta, SoilTypeText, SunlightText } from '@/constants'
import { formatArea, clamp } from '@/utils/format'

const store = usePlotStore()
const pagination = usePagination()
const { user, isAdmin } = useAuth()

const createVisible = ref(false)
const creating = ref(false)
const createForm = reactive({ name: '', code: '', area: 10, soil_type: 'loam', sunlight: 'full', latitude: 31.2304, longitude: 121.4737, description: '' })

// 转交申请弹窗
const transferVisible = ref(false)
const submitting = ref(false)
const transferForm = reactive({ plotId: 0, plotName: '', plotCode: '', reason: '' })

// 地块详情抽屉
const detailVisible = ref(false)
const detail = ref<Plot | null>(null)

const mapW = 600
const mapH = 360
const viewBoxStr = `0 0 ${mapW} ${mapH}`

function mapX(p: Plot) {
  return clamp(((p.longitude - 121.47) / 0.01) * 1000 + mapW / 2, 20, mapW - 20)
}
function mapY(p: Plot) {
  return clamp(((31.24 - p.latitude) / 0.02) * 1000 + mapH / 2, 20, mapH - 20)
}
function colorOf(status: string) {
  if (status === 'available') return '#67c23a'
  if (status === 'adopted') return '#e6a23c'
  if (status === 'pending_transfer') return '#f56c6c'
  return '#909399'
}

function onPage(page: number) {
  pagination.page.value = page
  fetch()
}

async function fetch() {
  await store.fetchPlots({ page: pagination.page.value, page_size: pagination.size.value })
}

// 本人名下且处于可申请状态（已认养/待释放），且没有待处理申请
function canRequestTransfer(row: Plot) {
  return !!user.value && row.adopter_id === user.value.id
    && (row.status === 'adopted' || row.status === 'harvested')
    && row.latest_transfer?.status !== 'pending'
}

// 本人待处理申请可撤回
function canWithdraw(row: Plot) {
  return !!user.value && row.adopter_id === user.value.id
    && row.status === 'pending_transfer'
    && row.latest_transfer?.status === 'pending'
}

async function adopt(row: Plot) {
  try {
    await ElMessageBox.confirm(`确认认养地块 ${row.name}（${row.code}）吗？`, '认养确认', { type: 'success' })
  } catch {
    return
  }
  await store.adopt(row.id)
  ElMessage.success('认养成功，开始你的都市农夫之旅')
}

// 管理员强制释放（认养人请走转交申请）
async function release(row: Plot) {
  try {
    await ElMessageBox.confirm(`管理员强制释放地块 ${row.name} 吗？释放后将重新回到共享池。`, '强制释放确认', { type: 'warning' })
  } catch {
    return
  }
  await releasePlot(row.id)
  ElMessage.success('地块已释放')
  await fetch()
}

function openCreate() {
  createVisible.value = true
}

async function submitCreate() {
  creating.value = true
  try {
    await createPlot({ ...createForm })
    ElMessage.success('地块创建成功')
    createVisible.value = false
    await fetch()
  } finally {
    creating.value = false
  }
}

function openTransfer(row: Plot) {
  transferForm.plotId = row.id
  transferForm.plotName = row.name
  transferForm.plotCode = row.code
  transferForm.reason = ''
  transferVisible.value = true
}

async function submitTransfer() {
  if (!transferForm.reason.trim()) {
    ElMessage.warning('请填写接管理由')
    return
  }
  submitting.value = true
  try {
    await apiSubmitTransfer(transferForm.plotId, transferForm.reason.trim())
    ElMessage.success('转交申请已提交，等待管理员核准')
    transferVisible.value = false
    await fetch()
  } finally {
    submitting.value = false
  }
}

async function withdraw(row: Plot) {
  if (!row.latest_transfer) return
  try {
    await ElMessageBox.confirm(`确认撤回转交申请（地块 ${row.code}）吗？撤回后地块恢复原状态。`, '撤回确认', { type: 'warning' })
  } catch {
    return
  }
  try {
    await withdrawTransfer(row.latest_transfer.id)
    ElMessage.success('转交申请已撤回')
    await fetch()
    if (detailVisible.value) await openDetail(row)
  } catch {
    // 错误提示已由 request 拦截器统一弹出（申请被先核准时会提示"已经处理"）
    detailVisible.value = false
    await fetch()
  }
}

async function openDetail(row: Plot) {
  detailVisible.value = true
  detail.value = row
  try {
    detail.value = await getPlot(row.id)
  } catch {
    // 拉取失败时保留列表行数据
  }
}

onMounted(fetch)
</script>

<style scoped>
.plot-map { width: 100%; height: 360px; border-radius: 8px; }
.transfer-expand { padding: 8px 24px; }
</style>

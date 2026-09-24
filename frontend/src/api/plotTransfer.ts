import { get, post } from '@/utils/request'
import type { UserInfo } from './auth'

// 转交申请（认养人发起，管理员核准后地块回到共享池）
export interface TransferApplication {
  id: number
  plot_id: number
  applicant_id: number
  applicant: UserInfo | null
  reason: string
  status: string
  reviewer_id: number | null
  reviewer: UserInfo | null
  review_comment: string
  reviewed_at: string
  created_at: string
}

export interface TransferApplicationWithPlot extends TransferApplication {
  plot?: { id: number; name: string; code: string } | null
}

// 认养人在地块列表提交转交申请
export function submitTransfer(plotId: number, reason: string): Promise<TransferApplication> {
  return post(`/plots/${plotId}/transfers`, { reason })
}

// 认养人撤回自己的待处理申请
export function withdrawTransfer(id: number): Promise<TransferApplication> {
  return post(`/plot-transfers/${id}/withdraw`)
}

// 管理员核准/驳回（驳回必须带意见）
export function reviewTransfer(id: number, approve: boolean, comment: string): Promise<TransferApplication> {
  return post(`/plot-transfers/${id}/review`, { approve, comment })
}

// 管理员分页查询申请列表
export function listTransfers(params?: Record<string, any>): Promise<{ list: TransferApplicationWithPlot[]; total: number; page: number; page_size: number }> {
  return get('/plot-transfers', { params })
}

// 查询单条申请详情
export function getTransfer(id: number): Promise<TransferApplicationWithPlot> {
  return get(`/plot-transfers/${id}`)
}

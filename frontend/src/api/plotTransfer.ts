import { get, post } from '@/utils/request'
import type { Plot } from './plot'
import type { UserInfo } from './auth'

// 地块转交申请（列表与详情保留申请原因、处理意见及当前状态）
export interface TransferRequest {
  id: number
  plot_id: number
  plot: Plot | null
  applicant_id: number
  applicant: UserInfo | null
  reason: string
  status: string
  prev_plot_status: string
  review_comment: string
  reviewer_id: number | null
  reviewer: UserInfo | null
  reviewed_at: string
  created_at: string
}

export function submitTransferRequest(plotId: number, reason: string): Promise<TransferRequest> {
  return post(`/plots/${plotId}/transfer-requests`, { reason })
}

export function listTransferRequests(params?: Record<string, any>): Promise<{ list: TransferRequest[]; total: number; page: number; page_size: number }> {
  return get('/transfer-requests', { params })
}

export function getTransferRequest(id: number): Promise<TransferRequest> {
  return get(`/transfer-requests/${id}`)
}

export function approveTransferRequest(id: number, reviewComment: string): Promise<TransferRequest> {
  return post(`/transfer-requests/${id}/approve`, { review_comment: reviewComment })
}

export function rejectTransferRequest(id: number, reviewComment: string): Promise<TransferRequest> {
  return post(`/transfer-requests/${id}/reject`, { review_comment: reviewComment })
}

export function cancelTransferRequest(id: number): Promise<TransferRequest> {
  return post(`/transfer-requests/${id}/cancel`)
}

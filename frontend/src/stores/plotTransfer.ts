import { defineStore } from 'pinia'
import {
  approveTransferRequest,
  cancelTransferRequest,
  listTransferRequests,
  rejectTransferRequest,
  type TransferRequest
} from '@/api/plotTransfer'

interface TransferState {
  requests: TransferRequest[]
  total: number
  loading: boolean
}

export const usePlotTransferStore = defineStore('plotTransfer', {
  state: (): TransferState => ({ requests: [], total: 0, loading: false }),
  actions: {
    async fetchRequests(params?: Record<string, any>) {
      this.loading = true
      try {
        const data = await listTransferRequests(params)
        this.requests = data.list
        this.total = data.total
      } finally {
        this.loading = false
      }
    },
    async approve(id: number, reviewComment: string) {
      await approveTransferRequest(id, reviewComment)
    },
    async reject(id: number, reviewComment: string) {
      await rejectTransferRequest(id, reviewComment)
    },
    async cancel(id: number) {
      await cancelTransferRequest(id)
    }
  }
})

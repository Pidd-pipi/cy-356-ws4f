import { defineStore } from 'pinia'
import {
  listTransfers,
  reviewTransfer,
  submitTransfer,
  withdrawTransfer,
  type TransferApplicationWithPlot
} from '@/api/plotTransfer'

interface TransferState {
  applications: TransferApplicationWithPlot[]
  total: number
  loading: boolean
}

export const useTransferStore = defineStore('plotTransfer', {
  state: (): TransferState => ({ applications: [], total: 0, loading: false }),
  actions: {
    async fetchApplications(params?: Record<string, any>) {
      this.loading = true
      try {
        const data = await listTransfers(params)
        this.applications = data.list
        this.total = data.total
      } finally {
        this.loading = false
      }
    },
    async submit(plotId: number, reason: string) {
      return submitTransfer(plotId, reason)
    },
    async withdraw(id: number) {
      return withdrawTransfer(id)
    },
    async review(id: number, approve: boolean, comment: string) {
      return reviewTransfer(id, approve, comment)
    }
  }
})

import type { CreateScanJobInput, HealthStatus, ScanJob, ScanJobPage, ScanJobStatus } from '../types/scan-job'
import { api } from './api'

const jobsPath = '/api/v1/scan-jobs'

export function fetchHealth(): Promise<HealthStatus> {
  return api<HealthStatus>('/api/v1/health')
}

export function fetchScanJobs(filter: {
  status?: ScanJobStatus | ''
  page: number
  pageSize?: number
}): Promise<ScanJobPage> {
  const params = new URLSearchParams()
  params.set('page', String(filter.page))
  params.set('page_size', String(filter.pageSize ?? 20))
  if (filter.status) {
    params.set('status', filter.status)
  }
  return api<ScanJobPage>(`${jobsPath}?${params.toString()}`)
}

export function fetchScanJob(id: string): Promise<ScanJob> {
  return api<ScanJob>(`${jobsPath}/${id}`)
}

export function createScanJob(input: CreateScanJobInput): Promise<ScanJob> {
  return api<ScanJob>(jobsPath, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function cancelScanJob(id: string): Promise<ScanJob> {
  return api<ScanJob>(`${jobsPath}/${id}/cancel`, { method: 'POST' })
}

export function deleteScanJob(id: string): Promise<void> {
  return api<void>(`${jobsPath}/${id}`, { method: 'DELETE' })
}

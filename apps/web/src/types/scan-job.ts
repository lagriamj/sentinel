export const SCAN_JOB_STATUSES = [
  'pending',
  'queued',
  'running',
  'completed',
  'failed',
  'cancelled',
] as const

export type ScanJobStatus = (typeof SCAN_JOB_STATUSES)[number]

export type ScanTarget = {
  id: string
  url: string
  scheme: string
  host: string
  port: number
  authorized: boolean
}

export type ScanJob = {
  id: string
  name: string
  status: ScanJobStatus
  target: ScanTarget
  error_message?: string
  started_at?: string
  finished_at?: string
  created_at: string
  updated_at: string
}

export type ScanJobPage = {
  data: ScanJob[]
  page: number
  page_size: number
  total: number
}

export type CreateScanJobInput = {
  name: string
  target_id: string
}

export type HealthStatus = {
  status: string
}

export function isScanJobStatus(value: string): value is ScanJobStatus {
  return SCAN_JOB_STATUSES.some((status) => status === value)
}

export function canCancel(status: ScanJobStatus): boolean {
  return status === 'pending' || status === 'queued' || status === 'running'
}

export function canDelete(status: ScanJobStatus): boolean {
  return status === 'pending' || status === 'completed' || status === 'failed' || status === 'cancelled'
}

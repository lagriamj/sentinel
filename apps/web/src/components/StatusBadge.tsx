import { Badge } from '@/components/ui/badge'
import type { ScanJobStatus } from '../types/scan-job'

const labels: Record<ScanJobStatus, string> = {
  pending: 'Pending',
  queued: 'Queued',
  running: 'Running',
  completed: 'Completed',
  failed: 'Failed',
  cancelled: 'Cancelled',
}

const tones: Record<ScanJobStatus, string> = {
  pending: 'border-border bg-muted text-foreground',
  queued: 'border-sky-800 bg-sky-950 text-sky-200',
  running: 'border-amber-800 bg-amber-950 text-amber-200',
  completed: 'border-emerald-800 bg-emerald-950 text-emerald-200',
  failed: 'border-red-800 bg-red-950 text-red-200',
  cancelled: 'border-border bg-muted/40 text-muted-foreground',
}

export function statusLabel(status: ScanJobStatus): string {
  return labels[status]
}

export function StatusBadge({ status }: { status: ScanJobStatus }) {
  return (
    <Badge variant="outline" className={tones[status]}>
      {labels[status]}
    </Badge>
  )
}

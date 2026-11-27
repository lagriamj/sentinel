import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { EmptyState } from '../../components/EmptyState'
import { LoadingState } from '../../components/LoadingState'
import { StatusBadge } from '../../components/StatusBadge'
import { formatDateTime } from '../../lib/format'
import { fetchScanJobs } from '../../services/scan-jobs'
import { ApiError } from '../../services/api'
import { SCAN_JOB_STATUSES, isScanJobStatus, type ScanJobStatus } from '../../types/scan-job'

const pageSize = 20
const allStatuses = 'all'

export function ScanJobListPage() {
  const [params, setParams] = useSearchParams()
  const rawStatus = params.get('status') ?? ''
  const status: ScanJobStatus | '' = isScanJobStatus(rawStatus) ? rawStatus : ''
  const rawPage = Number(params.get('page'))
  const page = Number.isInteger(rawPage) && rawPage >= 1 && rawPage <= 10000 ? rawPage : 1

  const jobs = useQuery({
    queryKey: ['scan-jobs', { status, page, pageSize }],
    queryFn: () => fetchScanJobs({ status, page, pageSize }),
    placeholderData: keepPreviousData,
  })

  function updateFilter(nextStatus: string, nextPage: number) {
    const next = new URLSearchParams()
    if (isScanJobStatus(nextStatus)) {
      next.set('status', nextStatus)
    }
    if (nextPage > 1) {
      next.set('page', String(nextPage))
    }
    setParams(next)
  }

  const total = jobs.data?.total ?? 0
  const visible = jobs.data?.data.length ?? 0
  const from = total === 0 ? 0 : (page - 1) * pageSize + 1
  const to = total === 0 ? 0 : from + visible - 1
  const hasNext = page * pageSize < total

  return (
    <div className="mx-auto max-w-6xl">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Scan jobs</h1>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
            Jobs use a target registered on a project. A saved job stays pending until a scanner is connected.
          </p>
        </div>
        <Button asChild>
          <Link to="/scan-jobs/new">New scan job</Link>
        </Button>
      </div>

      <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2">
          <Label htmlFor="job-status">Status</Label>
          <Select value={status || allStatuses} onValueChange={(value) => updateFilter(value === allStatuses ? '' : value, 1)}>
            <SelectTrigger id="job-status" className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={allStatuses}>All</SelectItem>
              {SCAN_JOB_STATUSES.map((item) => (
                <SelectItem key={item} value={item}>
                  {item}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Button type="button" variant="outline" onClick={() => void jobs.refetch()} disabled={jobs.isFetching}>
          Refresh
        </Button>
      </div>

      <div className="mt-4">
        {jobs.isPending ? <LoadingState label="Loading scan jobs" /> : null}
        {jobs.isError ? (
          <Alert variant="destructive">
            <AlertDescription>{jobs.error instanceof ApiError ? jobs.error.message : 'Scan jobs could not be loaded.'}</AlertDescription>
          </Alert>
        ) : null}
        {jobs.data && jobs.data.data.length === 0 ? (
          <EmptyState title="No scan jobs">
            {status ? 'No jobs match this status.' : 'Register a target on a project, then create a job.'}
          </EmptyState>
        ) : null}
        {jobs.data && jobs.data.data.length > 0 ? (
          <>
            <div className="rounded-xl ring-1 ring-foreground/10">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Target</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {jobs.data.data.map((job) => (
                    <TableRow key={job.id}>
                      <TableCell>
                        <Link to={`/scan-jobs/${job.id}`} className="font-medium text-primary underline-offset-4 hover:underline">
                          {job.name}
                        </Link>
                      </TableCell>
                      <TableCell className="max-w-xs truncate text-muted-foreground">{job.target.url}</TableCell>
                      <TableCell>
                        <StatusBadge status={job.status} />
                      </TableCell>
                      <TableCell className="text-muted-foreground">{formatDateTime(job.created_at)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
              <p>
                {from}–{to} of {total}
              </p>
              <div className="flex gap-2">
                <Button type="button" variant="outline" disabled={page <= 1} onClick={() => updateFilter(status, page - 1)}>
                  Previous
                </Button>
                <Button type="button" variant="outline" disabled={!hasNext} onClick={() => updateFilter(status, page + 1)}>
                  Next
                </Button>
              </div>
            </div>
          </>
        ) : null}
      </div>
    </div>
  )
}

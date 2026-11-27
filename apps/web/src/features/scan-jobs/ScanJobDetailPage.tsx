import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { LoadingState } from '../../components/LoadingState'
import { StatusBadge } from '../../components/StatusBadge'
import { formatDateTime, targetEndpoint } from '../../lib/format'
import { ApiError } from '../../services/api'
import { cancelScanJob, deleteScanJob, fetchScanJob } from '../../services/scan-jobs'
import { canCancel, canDelete } from '../../types/scan-job'

export function ScanJobDetailPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [confirming, setConfirming] = useState<'cancel' | 'delete' | null>(null)

  const job = useQuery({
    queryKey: ['scan-jobs', id],
    queryFn: () => fetchScanJob(id),
    enabled: id !== '',
  })

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: ['scan-jobs'] })
  }

  const cancel = useMutation({
    mutationFn: () => cancelScanJob(id),
    onSuccess: () => {
      setConfirming(null)
      refresh()
    },
  })

  const remove = useMutation({
    mutationFn: () => deleteScanJob(id),
    onSuccess: () => {
      refresh()
      void navigate('/scan-jobs')
    },
  })

  if (job.isPending) {
    return <LoadingState label="Loading scan job" />
  }
  if (job.isError || !job.data) {
    const message = job.error instanceof ApiError ? job.error.message : 'Scan job could not be loaded.'
    return (
      <div className="mx-auto max-w-3xl">
        <Alert variant="destructive">
          <AlertDescription>{message}</AlertDescription>
        </Alert>
        <Button variant="link" className="mt-2 px-0" asChild>
          <Link to="/scan-jobs">Back to scan jobs</Link>
        </Button>
      </div>
    )
  }

  const record = job.data
  const actionError = cancel.error || remove.error
  const actionMessage = actionError instanceof ApiError ? actionError.message : actionError ? 'The job could not be updated.' : ''

  return (
    <div className="mx-auto max-w-3xl">
      <Button variant="link" className="px-0" asChild>
        <Link to="/scan-jobs">Back to scan jobs</Link>
      </Button>
      <div className="mt-3 flex flex-wrap items-center gap-3">
        <h1 className="text-2xl font-semibold">{record.name}</h1>
        <StatusBadge status={record.status} />
      </div>
      <p className="mt-2 font-mono text-xs text-muted-foreground">{record.id}</p>

      <Card className="mt-6">
        <CardContent>
          <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Target" value={record.target.url} />
            <Field label="Endpoint" value={targetEndpoint(record.target.host, record.target.port)} />
            <Field label="Scheme" value={record.target.scheme} />
            <Field label="Authorization" value={record.target.authorized ? 'Confirmed' : 'Missing'} />
            <Field label="Created" value={formatDateTime(record.created_at)} />
            <Field label="Updated" value={formatDateTime(record.updated_at)} />
            <Field label="Started" value={formatDateTime(record.started_at)} />
            <Field label="Finished" value={formatDateTime(record.finished_at)} />
          </dl>
        </CardContent>
      </Card>

      {record.error_message ? (
        <Alert variant="destructive" className="mt-4">
          <AlertDescription>{record.error_message}</AlertDescription>
        </Alert>
      ) : null}
      {actionMessage ? (
        <Alert variant="destructive" className="mt-4">
          <AlertDescription>{actionMessage}</AlertDescription>
        </Alert>
      ) : null}

      <div className="mt-6 flex flex-wrap gap-2">
        {canCancel(record.status) ? (
          <Button type="button" variant="outline" onClick={() => setConfirming('cancel')}>
            Cancel job
          </Button>
        ) : null}
        {canDelete(record.status) ? (
          <Button type="button" variant="destructive" onClick={() => setConfirming('delete')}>
            Delete job
          </Button>
        ) : null}
      </div>

      {confirming ? (
        <Card className="mt-4">
          <CardContent className="grid gap-3">
            <p>{confirming === 'cancel' ? 'Cancel this job?' : 'Delete this job and its stored results?'}</p>
            <div className="flex gap-2">
              <Button
                type="button"
                variant={confirming === 'delete' ? 'destructive' : 'default'}
                disabled={cancel.isPending || remove.isPending}
                onClick={() => (confirming === 'cancel' ? cancel.mutate() : remove.mutate())}
              >
                {confirming === 'cancel' ? 'Confirm cancel' : 'Confirm delete'}
              </Button>
              <Button type="button" variant="outline" onClick={() => setConfirming(null)}>
                Keep job
              </Button>
            </div>
          </CardContent>
        </Card>
      ) : null}
    </div>
  )
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs tracking-wide text-muted-foreground uppercase">{label}</dt>
      <dd className="mt-1 break-all text-sm">{value}</dd>
    </div>
  )
}

import { useMutation, useQuery } from '@tanstack/react-query'
import { type FormEvent, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { EmptyState } from '../../components/EmptyState'
import { LoadingState } from '../../components/LoadingState'
import { ApiError } from '../../services/api'
import { fetchProjects, fetchTargets } from '../../services/projects'
import { createScanJob } from '../../services/scan-jobs'

const maxNameLength = 200

export function CreateScanJobPage() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const requestedProject = params.get('project') ?? ''
  const [name, setName] = useState('')
  const [projectID, setProjectID] = useState(requestedProject)
  const [targetID, setTargetID] = useState('')
  const [formError, setFormError] = useState('')

  const projects = useQuery({
    queryKey: ['projects'],
    queryFn: () => fetchProjects(1),
  })
  const selectedProject = projectID || (projects.data?.data.length === 1 ? projects.data.data[0].id : '')
  const targets = useQuery({
    queryKey: ['project-targets', selectedProject],
    queryFn: () => fetchTargets(selectedProject),
    enabled: selectedProject !== '',
  })
  const selectedTarget = targetID || (targets.data?.data.length === 1 ? targets.data.data[0].id : '')

  const create = useMutation({
    mutationFn: createScanJob,
    onSuccess: (job) => {
      void navigate(`/scan-jobs/${job.id}`)
    },
  })

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const trimmed = name.trim()
    if (trimmed === '') {
      setFormError('Name is required.')
      return
    }
    if (trimmed.length > maxNameLength) {
      setFormError('Name must be 200 characters or fewer.')
      return
    }
    if (selectedProject === '') {
      setFormError('Project is required.')
      return
    }
    if (selectedTarget === '') {
      setFormError('Target is required.')
      return
    }
    setFormError('')
    create.mutate({ name: trimmed, target_id: selectedTarget })
  }

  if (projects.isPending) {
    return <LoadingState label="Loading projects" />
  }
  if (projects.isError) {
    return (
      <Alert variant="destructive">
        <AlertDescription>{projects.error instanceof ApiError ? projects.error.message : 'Projects could not be loaded.'}</AlertDescription>
      </Alert>
    )
  }
  if (projects.data.data.length === 0) {
    return (
      <div className="mx-auto max-w-xl">
        <EmptyState title="No projects">
          <Link to="/projects/new" className="text-primary underline-offset-4 hover:underline">
            Create a project
          </Link>{' '}
          before recording a scan job.
        </EmptyState>
      </div>
    )
  }

  const message = formError || (create.error instanceof ApiError ? create.error.message : create.error ? 'The job could not be saved.' : '')

  return (
    <div className="mx-auto max-w-xl">
      <Button variant="link" className="px-0" asChild>
        <Link to="/scan-jobs">Back to scan jobs</Link>
      </Button>
      <h1 className="mt-3 text-2xl font-semibold">New scan job</h1>
      <p className="mt-1 text-sm text-muted-foreground">Choose a target that was registered for a project.</p>
      <form className="mt-6 space-y-4" onSubmit={onSubmit} noValidate>
        <div className="grid gap-2">
          <Label htmlFor="job-name">Name</Label>
          <Input id="job-name" value={name} maxLength={maxNameLength} onChange={(event) => setName(event.target.value)} autoComplete="off" />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="job-project">Project</Label>
          <select
            id="job-project"
            value={selectedProject}
            onChange={(event) => {
              setProjectID(event.target.value)
              setTargetID('')
            }}
            className="h-8 rounded-lg border border-input bg-transparent px-2.5 text-sm dark:bg-input/30"
          >
            <option value="">Select a project</option>
            {projects.data.data.map((project) => (
              <option key={project.id} value={project.id}>
                {project.name}
              </option>
            ))}
          </select>
        </div>
        <div className="grid gap-2">
          <Label htmlFor="job-target">Target</Label>
          <select
            id="job-target"
            value={selectedTarget}
            onChange={(event) => setTargetID(event.target.value)}
            className="h-8 rounded-lg border border-input bg-transparent px-2.5 text-sm dark:bg-input/30"
            disabled={selectedProject === '' || targets.isPending}
          >
            <option value="">Select a target</option>
            {targets.data?.data.map((target) => (
              <option key={target.id} value={target.id}>
                {target.url}
              </option>
            ))}
          </select>
          {selectedProject !== '' && targets.data && targets.data.data.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              This project has no targets.{' '}
              <Link to={`/projects/${selectedProject}`} className="text-primary underline-offset-4 hover:underline">
                Register one
              </Link>
              .
            </p>
          ) : null}
        </div>
        {message ? (
          <Alert variant="destructive">
            <AlertDescription>{message}</AlertDescription>
          </Alert>
        ) : null}
        <div className="flex gap-2">
          <Button type="submit" disabled={create.isPending}>
            {create.isPending ? 'Saving' : 'Save job'}
          </Button>
          <Button variant="ghost" asChild>
            <Link to="/scan-jobs">Cancel</Link>
          </Button>
        </div>
      </form>
    </div>
  )
}

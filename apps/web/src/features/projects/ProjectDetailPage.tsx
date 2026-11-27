import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { type FormEvent, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { EmptyState } from '../../components/EmptyState'
import { LoadingState } from '../../components/LoadingState'
import { ApiError } from '../../services/api'
import { addMember, deleteProject, deleteTarget, fetchMembers, fetchProject, fetchTargets, registerTarget, removeMember } from '../../services/projects'

export function ProjectDetailPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const client = useQueryClient()
  const [removeError, setRemoveError] = useState('')
  const project = useQuery({
    queryKey: ['project', id],
    queryFn: () => fetchProject(id),
  })
  const members = useQuery({
    queryKey: ['project-members', id],
    queryFn: () => fetchMembers(id),
    enabled: project.isSuccess,
  })
  const targets = useQuery({
    queryKey: ['project-targets', id],
    queryFn: () => fetchTargets(id),
    enabled: project.isSuccess,
  })

  const remove = useMutation({
    mutationFn: () => deleteProject(id),
    onSuccess: () => {
      void navigate('/projects')
    },
  })

  if (project.isPending) {
    return <LoadingState label="Loading project" />
  }
  if (project.isError) {
    return (
      <Alert variant="destructive">
        <AlertDescription>{project.error instanceof ApiError ? project.error.message : 'The project could not be loaded.'}</AlertDescription>
      </Alert>
    )
  }

  const isOwner = project.data.role === 'owner'
  const message = remove.error instanceof ApiError ? remove.error.message : remove.error ? 'The project could not be deleted.' : ''

  return (
    <div className="mx-auto max-w-4xl">
      <Button variant="link" className="px-0" asChild>
        <Link to="/projects">Back to projects</Link>
      </Button>
      <div className="mt-3 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">{project.data.name}</h1>
          <p className="mt-1 text-sm capitalize text-muted-foreground">{project.data.role}</p>
          {project.data.description ? <p className="mt-3 max-w-2xl text-sm">{project.data.description}</p> : null}
        </div>
        <Button asChild>
          <Link to={`/scan-jobs/new?project=${project.data.id}`}>New scan job</Link>
        </Button>
      </div>

      <section className="mt-8">
        <h2 className="text-lg font-medium">Members</h2>
        {members.isPending ? <LoadingState label="Loading members" /> : null}
        {members.isError ? (
          <Alert variant="destructive" className="mt-3">
            <AlertDescription>Members could not be loaded.</AlertDescription>
          </Alert>
        ) : null}
        {members.data ? (
          <ul className="mt-3 divide-y rounded-xl ring-1 ring-foreground/10">
            {members.data.data.map((member) => (
              <li key={member.user_id} className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
                <span>
                  {member.email}
                  <span className="ml-2 capitalize text-muted-foreground">{member.role}</span>
                </span>
                {isOwner ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setRemoveError('')
                      void removeMember(id, member.user_id)
                        .then(() => client.invalidateQueries({ queryKey: ['project-members', id] }))
                        .catch((err: unknown) => setRemoveError(err instanceof ApiError ? err.message : 'The member could not be removed.'))
                    }}
                  >
                    Remove
                  </Button>
                ) : null}
              </li>
            ))}
          </ul>
        ) : null}
        {removeError ? (
          <Alert variant="destructive" className="mt-3">
            <AlertDescription>{removeError}</AlertDescription>
          </Alert>
        ) : null}
        {isOwner ? <AddMemberForm projectID={id} /> : null}
      </section>

      <section className="mt-8">
        <h2 className="text-lg font-medium">Targets</h2>
        <p className="mt-1 text-sm text-muted-foreground">Only a registered, authorized target can be used for a scan job.</p>
        {targets.isPending ? <LoadingState label="Loading targets" /> : null}
        {targets.data && targets.data.data.length === 0 ? (
          <div className="mt-3">
            <EmptyState title="No targets">Register an http or https address you are allowed to assess.</EmptyState>
          </div>
        ) : null}
        {targets.data && targets.data.data.length > 0 ? (
          <ul className="mt-3 divide-y rounded-xl ring-1 ring-foreground/10">
            {targets.data.data.map((target) => (
              <li key={target.id} className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
                <span className="min-w-0 truncate">{target.url}</span>
                {isOwner ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => void deleteTarget(id, target.id).then(() => client.invalidateQueries({ queryKey: ['project-targets', id] }))}
                  >
                    Remove
                  </Button>
                ) : null}
              </li>
            ))}
          </ul>
        ) : null}
        <RegisterTargetForm projectID={id} />
      </section>

      {isOwner ? (
        <div className="mt-8">
          {message ? (
            <Alert variant="destructive" className="mb-3">
              <AlertDescription>{message}</AlertDescription>
            </Alert>
          ) : null}
          <Button type="button" variant="outline" onClick={() => remove.mutate()} disabled={remove.isPending}>
            Delete project
          </Button>
        </div>
      ) : null}
    </div>
  )
}

function AddMemberForm({ projectID }: { projectID: string }) {
  const client = useQueryClient()
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<'member' | 'owner'>('member')
  const [formError, setFormError] = useState('')
  const add = useMutation({
    mutationFn: () => addMember(projectID, { email, role }),
    onSuccess: () => {
      setEmail('')
      void client.invalidateQueries({ queryKey: ['project-members', projectID] })
    },
  })

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (email.trim() === '') {
      setFormError('Email is required.')
      return
    }
    setFormError('')
    add.mutate()
  }

  const message = formError || (add.error instanceof ApiError ? add.error.message : '')

  return (
    <form className="mt-4 grid gap-3 sm:grid-cols-[1fr_9rem_auto] sm:items-end" onSubmit={onSubmit} noValidate>
      <div className="grid gap-2">
        <Label htmlFor="member-email">Email</Label>
        <Input id="member-email" type="email" value={email} onChange={(event) => setEmail(event.target.value)} autoComplete="off" />
      </div>
      <div className="grid gap-2">
        <Label htmlFor="member-role">Role</Label>
        <select
          id="member-role"
          value={role}
          onChange={(event) => setRole(event.target.value === 'owner' ? 'owner' : 'member')}
          className="h-8 rounded-lg border border-input bg-transparent px-2.5 text-sm dark:bg-input/30"
        >
          <option value="member">Member</option>
          <option value="owner">Owner</option>
        </select>
      </div>
      <Button type="submit" disabled={add.isPending}>
        Add member
      </Button>
      {message ? (
        <Alert variant="destructive" className="sm:col-span-3">
          <AlertDescription>{message}</AlertDescription>
        </Alert>
      ) : null}
    </form>
  )
}

function RegisterTargetForm({ projectID }: { projectID: string }) {
  const client = useQueryClient()
  const [url, setURL] = useState('')
  const [authorized, setAuthorized] = useState(false)
  const [formError, setFormError] = useState('')
  const register = useMutation({
    mutationFn: () => registerTarget(projectID, { url: url.trim(), authorized: true }),
    onSuccess: () => {
      setURL('')
      setAuthorized(false)
      void client.invalidateQueries({ queryKey: ['project-targets', projectID] })
    },
  })

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (url.trim() === '') {
      setFormError('Target URL is required.')
      return
    }
    if (!authorized) {
      setFormError('Authorization confirmation is required.')
      return
    }
    setFormError('')
    register.mutate()
  }

  const message = formError || (register.error instanceof ApiError ? register.error.message : '')

  return (
    <form className="mt-4 space-y-4" onSubmit={onSubmit} noValidate>
      <div className="grid gap-2">
        <Label htmlFor="target-url">Target URL</Label>
        <Input id="target-url" value={url} placeholder="https://example.com" onChange={(event) => setURL(event.target.value)} autoComplete="off" spellCheck={false} />
      </div>
      <div className="flex items-start gap-2">
        <Checkbox id="authorized" checked={authorized} onCheckedChange={(checked) => setAuthorized(checked === true)} />
        <Label htmlFor="authorized" className="leading-snug">
          I am authorized to assess this target.
        </Label>
      </div>
      {message ? (
        <Alert variant="destructive">
          <AlertDescription>{message}</AlertDescription>
        </Alert>
      ) : null}
      <Button type="submit" disabled={register.isPending}>
        {register.isPending ? 'Saving' : 'Register target'}
      </Button>
    </form>
  )
}

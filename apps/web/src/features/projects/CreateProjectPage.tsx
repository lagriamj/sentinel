import { useMutation } from '@tanstack/react-query'
import { type FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ApiError } from '../../services/api'
import { createProject } from '../../services/projects'

const maxNameLength = 200

export function CreateProjectPage() {
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [formError, setFormError] = useState('')
  const create = useMutation({
    mutationFn: createProject,
    onSuccess: (project) => {
      void navigate(`/projects/${project.id}`)
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
    setFormError('')
    create.mutate({ name: trimmed, description: description.trim() })
  }

  const message = formError || (create.error instanceof ApiError ? create.error.message : create.error ? 'The project could not be saved.' : '')

  return (
    <div className="mx-auto max-w-xl">
      <Button variant="link" className="px-0" asChild>
        <Link to="/projects">Back to projects</Link>
      </Button>
      <h1 className="mt-3 text-2xl font-semibold">New project</h1>
      <p className="mt-1 text-sm text-muted-foreground">A project holds the targets its members are allowed to assess.</p>
      <form className="mt-6 space-y-4" onSubmit={onSubmit} noValidate>
        <div className="grid gap-2">
          <Label htmlFor="project-name">Name</Label>
          <Input id="project-name" value={name} maxLength={maxNameLength} onChange={(event) => setName(event.target.value)} autoComplete="off" />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="project-description">Description</Label>
          <textarea
            id="project-description"
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            className="min-h-24 w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 dark:bg-input/30"
          />
        </div>
        {message ? (
          <Alert variant="destructive">
            <AlertDescription>{message}</AlertDescription>
          </Alert>
        ) : null}
        <div className="flex gap-2">
          <Button type="submit" disabled={create.isPending}>
            {create.isPending ? 'Saving' : 'Save project'}
          </Button>
          <Button variant="ghost" asChild>
            <Link to="/projects">Cancel</Link>
          </Button>
        </div>
      </form>
    </div>
  )
}

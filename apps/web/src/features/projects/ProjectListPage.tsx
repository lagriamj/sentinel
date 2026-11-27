import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { EmptyState } from '../../components/EmptyState'
import { LoadingState } from '../../components/LoadingState'
import { formatDateTime } from '../../lib/format'
import { ApiError } from '../../services/api'
import { fetchProjects } from '../../services/projects'

export function ProjectListPage() {
  const projects = useQuery({
    queryKey: ['projects'],
    queryFn: () => fetchProjects(1),
  })

  return (
    <div className="mx-auto max-w-6xl">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Projects</h1>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
            Group the systems you are allowed to assess. Register a target before creating a scan job.
          </p>
        </div>
        <Button asChild>
          <Link to="/projects/new">New project</Link>
        </Button>
      </div>
      <div className="mt-6">
        {projects.isPending ? <LoadingState label="Loading projects" /> : null}
        {projects.isError ? (
          <Alert variant="destructive">
            <AlertDescription>{projects.error instanceof ApiError ? projects.error.message : 'Projects could not be loaded.'}</AlertDescription>
          </Alert>
        ) : null}
        {projects.data && projects.data.data.length === 0 ? (
          <EmptyState title="No projects">Create a project for the application or API you are allowed to assess.</EmptyState>
        ) : null}
        {projects.data && projects.data.data.length > 0 ? (
          <div className="rounded-xl ring-1 ring-foreground/10">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {projects.data.data.map((project) => (
                  <TableRow key={project.id}>
                    <TableCell>
                      <Link to={`/projects/${project.id}`} className="font-medium text-primary underline-offset-4 hover:underline">
                        {project.name}
                      </Link>
                      {project.description ? <p className="mt-1 max-w-md truncate text-muted-foreground">{project.description}</p> : null}
                    </TableCell>
                    <TableCell className="capitalize text-muted-foreground">{project.role}</TableCell>
                    <TableCell className="text-muted-foreground">{formatDateTime(project.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : null}
      </div>
    </div>
  )
}

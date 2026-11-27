import type { Project, ProjectMember, ProjectPage, RegisteredTarget, TargetPage } from '../types/project'
import { api } from './api'

const projectsPath = '/api/v1/projects'

export function fetchProjects(page = 1): Promise<ProjectPage> {
  return api<ProjectPage>(`${projectsPath}?page=${page}&page_size=20`)
}

export function fetchProject(id: string): Promise<Project> {
  return api<Project>(`${projectsPath}/${id}`)
}

export function createProject(input: { name: string; description: string }): Promise<Project> {
  return api<Project>(projectsPath, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function deleteProject(id: string): Promise<void> {
  return api<void>(`${projectsPath}/${id}`, { method: 'DELETE' })
}

export function fetchMembers(projectID: string): Promise<{ data: ProjectMember[] }> {
  return api<{ data: ProjectMember[] }>(`${projectsPath}/${projectID}/members`)
}

export function addMember(projectID: string, input: { email: string; role: 'owner' | 'member' }): Promise<ProjectMember> {
  return api<ProjectMember>(`${projectsPath}/${projectID}/members`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function removeMember(projectID: string, userID: string): Promise<void> {
  return api<void>(`${projectsPath}/${projectID}/members/${userID}`, { method: 'DELETE' })
}

export function fetchTargets(projectID: string): Promise<TargetPage> {
  return api<TargetPage>(`${projectsPath}/${projectID}/targets?page=1&page_size=100`)
}

export function registerTarget(projectID: string, input: { url: string; authorized: boolean }): Promise<RegisteredTarget> {
  return api<RegisteredTarget>(`${projectsPath}/${projectID}/targets`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function deleteTarget(projectID: string, targetID: string): Promise<void> {
  return api<void>(`${projectsPath}/${projectID}/targets/${targetID}`, { method: 'DELETE' })
}

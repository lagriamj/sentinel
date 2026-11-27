export type ProjectRole = 'owner' | 'member'

export type Project = {
  id: string
  name: string
  description: string
  role: ProjectRole
  created_at: string
  updated_at: string
}

export type ProjectPage = {
  data: Project[]
  page: number
  page_size: number
  total: number
}

export type ProjectMember = {
  user_id: string
  email: string
  role: ProjectRole
  created_at: string
}

export type RegisteredTarget = {
  id: string
  project_id: string
  url: string
  scheme: string
  host: string
  port: number
  authorized: boolean
  created_at: string
  updated_at: string
}

export type TargetPage = {
  data: RegisteredTarget[]
  page: number
  page_size: number
  total: number
}

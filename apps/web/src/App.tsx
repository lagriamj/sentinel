import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { LoginPage } from './features/auth/LoginPage'
import { RegisterPage } from './features/auth/RegisterPage'
import { RequireSession } from './features/auth/RequireSession'
import { SessionProvider } from './features/auth/session'
import { CreateProjectPage } from './features/projects/CreateProjectPage'
import { ProjectDetailPage } from './features/projects/ProjectDetailPage'
import { ProjectListPage } from './features/projects/ProjectListPage'
import { CreateScanJobPage } from './features/scan-jobs/CreateScanJobPage'
import { NotFoundPage } from './features/scan-jobs/NotFoundPage'
import { ScanJobDetailPage } from './features/scan-jobs/ScanJobDetailPage'
import { ScanJobListPage } from './features/scan-jobs/ScanJobListPage'
import { AppShell } from './layouts/AppShell'
import { AuthLayout } from './layouts/AuthLayout'

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        refetchOnWindowFocus: false,
        retry: 1,
      },
    },
  })
}

export function App() {
  const [client] = useState(createQueryClient)

  return (
    <QueryClientProvider client={client}>
      <SessionProvider>
        <BrowserRouter>
          <Routes>
            <Route element={<AuthLayout />}>
              <Route path="login" element={<LoginPage />} />
              <Route path="register" element={<RegisterPage />} />
            </Route>
            <Route element={<AppShell />}>
              <Route element={<RequireSession />}>
                <Route index element={<Navigate to="/projects" replace />} />
                <Route path="projects" element={<ProjectListPage />} />
                <Route path="projects/new" element={<CreateProjectPage />} />
                <Route path="projects/:id" element={<ProjectDetailPage />} />
                <Route path="scan-jobs" element={<ScanJobListPage />} />
                <Route path="scan-jobs/new" element={<CreateScanJobPage />} />
                <Route path="scan-jobs/:id" element={<ScanJobDetailPage />} />
              </Route>
              <Route path="*" element={<NotFoundPage />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </SessionProvider>
    </QueryClientProvider>
  )
}

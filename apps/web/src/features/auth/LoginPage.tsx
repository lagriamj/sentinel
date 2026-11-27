import { type FormEvent, useState } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ApiError } from '../../services/api'
import { loginAccount } from '../../services/auth'
import { AuthCard } from './AuthCard'
import { useSession } from './session'

export function LoginPage() {
  const navigate = useNavigate()
  const { ready, account, setAccount } = useSession()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)

  if (ready && account) {
    return <Navigate to="/projects" replace />
  }

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (email.trim() === '' || password === '') {
      setError('Email and password are required.')
      return
    }
    setError('')
    setPending(true)
    try {
      const user = await loginAccount(email, password)
      setAccount(user)
      void navigate('/projects')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Sign-in failed.')
      setPending(false)
    }
  }

  return (
    <AuthCard
      title="Sign in"
      description="Use the account that owns the scan jobs."
      footer={
        <>
          No account yet?{' '}
          <Link to="/register" className="text-foreground underline-offset-4 hover:underline">
            Create one
          </Link>
        </>
      }
    >
      <form className="space-y-4" onSubmit={onSubmit} noValidate>
        <div className="grid gap-2">
          <Label htmlFor="email">Email</Label>
          <Input
            id="email"
            className="h-9"
            type="email"
            autoComplete="username"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="password">Password</Label>
          <Input
            id="password"
            className="h-9"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
        </div>
        {error ? (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}
        <Button type="submit" className="w-full" size="lg" disabled={pending}>
          {pending ? 'Signing in' : 'Sign in'}
        </Button>
      </form>
    </AuthCard>
  )
}

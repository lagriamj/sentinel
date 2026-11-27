import { type FormEvent, useState } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ApiError } from '../../services/api'
import { registerAccount } from '../../services/auth'
import { AuthCard } from './AuthCard'
import { useSession } from './session'

export function RegisterPage() {
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
    if (email.trim() === '') {
      setError('Email is required.')
      return
    }
    if (password.length < 12) {
      setError('Password must be 12 characters or more.')
      return
    }
    setError('')
    setPending(true)
    try {
      const user = await registerAccount(email, password)
      setAccount(user)
      void navigate('/projects')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'The account could not be created.')
      setPending(false)
    }
  }

  return (
    <AuthCard
      title="Create account"
      description="This account will own the scan jobs you record."
      footer={
        <>
          Already registered?{' '}
          <Link to="/login" className="text-foreground underline-offset-4 hover:underline">
            Sign in
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
            autoComplete="new-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
          <p className="text-xs text-muted-foreground">At least 12 characters.</p>
        </div>
        {error ? (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}
        <Button type="submit" className="w-full" size="lg" disabled={pending}>
          {pending ? 'Creating account' : 'Create account'}
        </Button>
      </form>
    </AuthCard>
  )
}

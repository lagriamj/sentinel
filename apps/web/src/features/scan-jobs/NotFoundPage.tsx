import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'

export function NotFoundPage() {
  return (
    <div className="mx-auto max-w-lg">
      <h1 className="text-xl font-semibold">Page not found</h1>
      <p className="mt-2 text-sm text-muted-foreground">That address is not part of the console.</p>
      <Button variant="link" className="mt-2 px-0" asChild>
        <Link to="/scan-jobs">Back to scan jobs</Link>
      </Button>
    </div>
  )
}

import type { ReactNode } from 'react'
import { Card, CardContent, CardDescription, CardFooter, CardHeader } from '@/components/ui/card'

type AuthCardProps = {
  title: string
  description: string
  children: ReactNode
  footer: ReactNode
}

export function AuthCard({ title, description, children, footer }: AuthCardProps) {
  return (
    <Card>
      <CardHeader className="border-b">
        <h1 className="text-xl font-medium tracking-tight">{title}</h1>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>{children}</CardContent>
      <CardFooter>
        <p className="text-sm text-muted-foreground">{footer}</p>
      </CardFooter>
    </Card>
  )
}

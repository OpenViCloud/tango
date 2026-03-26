import { Link } from "@tanstack/react-router"
import { AlertTriangle, ArrowLeft, RefreshCw } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"

type AdminRouteErrorProps = {
  error: unknown
  reset: () => void
}

function getErrorMessage(error: unknown) {
  if (error instanceof Error && error.message.trim() !== "") {
    return error.message
  }

  if (typeof error === "string" && error.trim() !== "") {
    return error
  }

  return "An unexpected error occurred while loading this page."
}

export function AdminRouteError({ error, reset }: AdminRouteErrorProps) {
  const isDev = import.meta.env.DEV
  const errorMessage = getErrorMessage(error)
  const stack =
    error instanceof Error && error.stack?.trim() ? error.stack : null

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/30 px-6 py-10">
      <Card className="w-full max-w-2xl border-destructive/20 shadow-sm">
        <CardHeader className="gap-4">
          <div className="flex size-12 items-center justify-center rounded-2xl bg-destructive/10 text-destructive">
            <AlertTriangle />
          </div>
          <div className="flex flex-col gap-2">
            <CardTitle className="text-2xl">Could not load page</CardTitle>
            <CardDescription className="max-w-xl text-sm leading-6">
              The page hit an unexpected problem. You can retry the route or go
              back to a safer screen.
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent className="flex flex-col gap-5">
          <div className="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
            {isDev
              ? errorMessage
              : "Something went wrong while rendering this page."}
          </div>

          <div className="flex flex-wrap gap-3">
            <Button onClick={reset}>
              <RefreshCw data-icon="inline-start" />
              Retry
            </Button>
            <Button asChild variant="outline">
              <Link to="/dashboard">
                <ArrowLeft data-icon="inline-start" />
                Back to Dashboard
              </Link>
            </Button>
          </div>

          {isDev && (
            <>
              <Separator />
              <div className="flex flex-col gap-3">
                <h2 className="text-sm font-semibold tracking-tight text-foreground">
                  Technical details
                </h2>
                <pre className="overflow-x-auto rounded-xl border bg-background px-4 py-3 text-xs leading-6 text-muted-foreground">
                  {stack ?? errorMessage}
                </pre>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

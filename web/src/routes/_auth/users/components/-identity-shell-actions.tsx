import { ThemeToggle } from "@/components/theme-toggle"

type IdentityShellActionsProps = {
  primaryAction?: React.ReactNode
}

export function IdentityShellActions({
  primaryAction,
}: IdentityShellActionsProps) {
  return (
    <>
      <ThemeToggle />
      {primaryAction}
    </>
  )
}

import type { ReactNode } from "react"

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { cn } from "@/lib/utils"

type SectionCardProps = {
  icon?: ReactNode
  title?: string
  description?: string
  headerRight?: ReactNode
  children: ReactNode
  className?: string
  contentClassName?: string
}

export function SectionCard({
  icon,
  title,
  description,
  headerRight,
  children,
  className,
  contentClassName,
}: SectionCardProps) {
  return (
    <Card className={cn("border-none bg-card shadow-panel", className)}>
      <CardHeader className="gap-4">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-center gap-3">
            {icon ? (
              <div className="flex size-11 items-center justify-center rounded-xl bg-primary/10 text-primary">
                {icon}
              </div>
            ) : null}
            <div className="flex flex-col gap-1">
              <CardTitle>{title}</CardTitle>
              {description ? (
                <CardDescription>{description}</CardDescription>
              ) : null}
            </div>
          </div>

          {headerRight ? <div className="shrink-0">{headerRight}</div> : null}
        </div>
      </CardHeader>
      <CardContent className={contentClassName}>{children}</CardContent>
    </Card>
  )
}

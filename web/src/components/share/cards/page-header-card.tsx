import type { ReactNode } from "react"

import { Badge } from "@/components/ui/badge"
import { CardDescription, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

type PageHeaderCardProps = {
  icon?: ReactNode
  title: string
  description?: string
  titleMeta?: ReactNode
  headerRight?: ReactNode
  className?: string
}

export function PageHeaderCard({
  icon,
  title,
  description,
  titleMeta,
  headerRight,
  className,
}: PageHeaderCardProps) {
  return (
    <div
      className={cn(
        "flex flex-col gap-4 md:flex-row md:items-center md:justify-between",
        className
      )}
    >
      <div className="flex flex-col gap-3">
        <div className="flex items-center gap-3">
          {icon ? (
            <div className="flex size-11 items-center justify-center rounded-xl bg-primary text-white">
              {icon}
            </div>
          ) : null}
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-3">
              <CardTitle>{title}</CardTitle>
              {titleMeta ? (
                <Badge variant="secondary" className="rounded-md px-2 py-0.5">
                  {titleMeta}
                </Badge>
              ) : null}
            </div>
            {description ? (
              <CardDescription>{description}</CardDescription>
            ) : null}
          </div>
        </div>
      </div>

      {headerRight ? <div className="shrink-0">{headerRight}</div> : null}
    </div>
  )
}

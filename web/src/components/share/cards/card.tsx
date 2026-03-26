import {
  Card as ShadcnCard,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { cx } from "class-variance-authority"
import type { PropsWithChildren, ReactNode } from "react"
type CardProps = {
  title?: string
  description?: string
  footerComponent?: ReactNode
  headerRigthComponent?: ReactNode
  className?: string
  headerComponent?: ReactNode
  style?: "border" | "background"
} & PropsWithChildren

const Card = ({
  children,
  title,
  description,
  footerComponent,
  className,
  headerRigthComponent,
  headerComponent,
  style = "background",
}: CardProps) => {
  const isHasHeader = title || description
  return (
    <ShadcnCard
      className={cx(
        !isHasHeader ? "pt-8" : "",
        className,

        style === "border"
          ? "border border-border bg-card"
          : "border-none bg-card shadow-panel"
      )}
    >
      {headerComponent && <CardHeader>{headerComponent}</CardHeader>}

      {isHasHeader && (
        <CardHeader>
          <CardTitle>
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-medium">{title}</h2>
              {headerRigthComponent}
            </div>
          </CardTitle>
          {description && <CardDescription>{description}</CardDescription>}
        </CardHeader>
      )}

      <CardContent>{children}</CardContent>
      {footerComponent && (
        <CardFooter>
          <p>Card Footer</p>
        </CardFooter>
      )}
    </ShadcnCard>
  )
}

export default Card

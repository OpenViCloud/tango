import { cn } from "@/lib/utils"

const sidebarItems = [
  "General",
  "Environment Variables",
  "Servers",
  "Persistent Storage",
  "Import Backups",
  "Webhooks",
  "Resource Limits",
  "Resource Operations",
  "Metrics",
  "Tags",
  "Danger Zone",
]

interface ConfigSidebarProps {
  active: string
  onSelect: (item: string) => void
  className?: string
}

export function ConfigSidebar({
  active,
  onSelect,
  className,
}: ConfigSidebarProps) {
  return (
    <nav className={cn("flex flex-col gap-0.5", className)}>
      {sidebarItems.map((item) => (
        <button
          key={item}
          onClick={() => onSelect(item)}
          className={cn(
            "rounded-md px-3 py-2 text-left text-sm transition-colors",
            active === item
              ? "bg-secondary font-medium text-accent"
              : "text-muted-foreground hover:bg-secondary/50 hover:text-foreground"
          )}
        >
          {item}
        </button>
      ))}
    </nav>
  )
}

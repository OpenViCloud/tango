import { useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "@tanstack/react-router"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { FolderIcon } from "lucide-react"

import type { CreateProjectModel, ProjectModel } from "@/@types/models"
import { createProjectSchema } from "@/@types/models/project"
import { useGetProjectList, useCreateProject } from "@/hooks/api/use-project"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetFooter,
} from "@/components/ui/sheet"
import { actionIcons, appIcons } from "@/lib/icons"

const CreateIcon = actionIcons.create
const ProjectsIcon = appIcons.projects

// ── Project card ───────────────────────────────────────────────────────────────

function ProjectCard({
  project,
  onClick,
}: {
  project: ProjectModel
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex flex-col gap-3 rounded-xl border bg-card p-4 text-left transition-shadow hover:border-primary/40 hover:shadow-sm focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
    >
      <div className="flex items-start gap-3">
        <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <FolderIcon className="size-5" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium">{project.name}</p>
          {project.description ? (
            <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">
              {project.description}
            </p>
          ) : null}
        </div>
      </div>
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <span>
          {project.environments.length}{" "}
          {project.environments.length === 1 ? "environment" : "environments"}
        </span>
      </div>
    </button>
  )
}

// ── Create project sheet ───────────────────────────────────────────────────────

function CreateProjectSheet({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const { t } = useTranslation()
  const createMutation = useCreateProject()

  const form = useForm<CreateProjectModel>({
    resolver: zodResolver(createProjectSchema),
    defaultValues: { name: "", description: "" },
  })

  const handleClose = (v: boolean) => {
    if (!v) form.reset()
    onOpenChange(v)
  }

  const onSubmit = form.handleSubmit((values) => {
    createMutation.mutate(values, {
      onSuccess: () => {
        toast.success(t("projects.resource.created"))
        handleClose(false)
      },
    })
  })

  const busy = createMutation.isPending

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className="flex flex-col sm:max-w-md">
        <SheetHeader className="border-b pb-4">
          <SheetTitle>{t("projects.createTitle")}</SheetTitle>
        </SheetHeader>

        <form onSubmit={onSubmit} className="flex flex-1 flex-col gap-5 py-4">
          <div className="flex flex-col gap-1.5 px-4">
            <Label htmlFor="proj-name">{t("projects.nameLabel")}</Label>
            <Input
              id="proj-name"
              placeholder={t("projects.namePlaceholder")}
              disabled={busy}
              {...form.register("name")}
            />
            {form.formState.errors.name && (
              <p className="text-xs text-destructive">
                {form.formState.errors.name.message}
              </p>
            )}
          </div>

          <div className="flex flex-col gap-1.5 px-4">
            <Label htmlFor="proj-desc">{t("projects.descriptionLabel")}</Label>
            <Input
              id="proj-desc"
              placeholder={t("projects.descriptionPlaceholder")}
              disabled={busy}
              {...form.register("description")}
            />
          </div>

          <SheetFooter className="mt-auto gap-2 border-t pt-4">
            <Button
              type="button"
              variant="outline"
              disabled={busy}
              onClick={() => handleClose(false)}
            >
              {t("projects.cancel")}
            </Button>
            <Button type="submit" disabled={busy}>
              {busy ? t("projects.creating") : t("projects.create")}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

// ── Main page ──────────────────────────────────────────────────────────────────

export function ProjectsPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [sheetOpen, setSheetOpen] = useState(false)

  const { data: projects, isLoading } = useGetProjectList()

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<ProjectsIcon />}
        title={t("projects.page.title")}
        description={t("projects.page.description")}
        headerRight={
          <Button size="sm" onClick={() => setSheetOpen(true)}>
            <CreateIcon data-icon="inline-start" />
            {t("projects.new")}
          </Button>
        }
      />

      <SectionCard>
        {isLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-28 w-full rounded-xl" />
            ))}
          </div>
        ) : !projects || projects.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("projects.empty")}</p>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {projects.map((project) => (
              <ProjectCard
                key={project.id}
                project={project}
                onClick={() =>
                  navigate({
                    to: "/projects/$projectId",
                    params: { projectId: project.id },
                  })
                }
              />
            ))}
          </div>
        )}
      </SectionCard>

      <CreateProjectSheet open={sheetOpen} onOpenChange={setSheetOpen} />
    </div>
  )
}

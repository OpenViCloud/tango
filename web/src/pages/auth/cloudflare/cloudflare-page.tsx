import { useEffect, useState } from "react"
import { toast } from "sonner"
import { CloudIcon, KeyRoundIcon, PencilIcon, PlusIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import type {
  CloudflareConnectionModel,
  CreateCloudflareConnectionModel,
  UpdateCloudflareConnectionModel,
} from "@/@types/models"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import {
  useCreateCloudflareConnection,
  useGetCloudflareConnections,
  useUpdateCloudflareConnection,
} from "@/hooks/api/use-cloudflare"
import { getErrorMessage } from "@/lib/get-error-message"
import { appIcons } from "@/lib/icons"

const CloudflareIcon = appIcons.cloudflare

type FormDraft = {
  display_name: string
  account_id: string
  zone_id: string
  api_token: string
}

function createDraft(connection?: CloudflareConnectionModel | null): FormDraft {
  return {
    display_name: connection?.display_name ?? "",
    account_id: connection?.account_id ?? "",
    zone_id: connection?.zone_id ?? "",
    api_token: "",
  }
}

export function CloudflarePage() {
  const { t } = useTranslation()
  const { data: connections = [], isLoading } = useGetCloudflareConnections()
  const [sheetOpen, setSheetOpen] = useState(false)
  const [editing, setEditing] = useState<CloudflareConnectionModel | null>(null)

  const openCreate = () => {
    setEditing(null)
    setSheetOpen(true)
  }

  const openEdit = (connection: CloudflareConnectionModel) => {
    setEditing(connection)
    setSheetOpen(true)
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<CloudflareIcon />}
        title={t("cloudflare.page.title")}
        description={t("cloudflare.page.description")}
        headerRight={
          <Button onClick={openCreate}>
            <PlusIcon data-icon="inline-start" />
            {t("cloudflare.actions.add")}
          </Button>
        }
      />

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {isLoading
          ? Array.from({ length: 3 }).map((_, index) => (
              <div
                key={index}
                className="flex flex-col gap-4 rounded-2xl border border-border/70 bg-card/80 p-5"
              >
                <Skeleton className="h-5 w-40" />
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-20 w-full" />
              </div>
            ))
          : connections.map((connection) => (
              <ConnectionCard
                key={connection.id}
                connection={connection}
                onEdit={() => openEdit(connection)}
              />
            ))}
      </div>

      {!isLoading && connections.length === 0 ? (
        <div className="rounded-3xl border border-dashed border-border/70 bg-card/60 px-6 py-10 text-center">
          <div className="mx-auto flex size-14 items-center justify-center rounded-2xl bg-muted text-muted-foreground">
            <CloudIcon />
          </div>
          <p className="mt-4 text-base font-semibold">{t("cloudflare.empty.title")}</p>
          <p className="mt-2 text-sm text-muted-foreground">
            {t("cloudflare.empty.description")}
          </p>
          <Button className="mt-5" onClick={openCreate}>
            <PlusIcon data-icon="inline-start" />
            {t("cloudflare.actions.add")}
          </Button>
        </div>
      ) : null}

      <ConnectionSheet
        open={sheetOpen}
        connection={editing}
        onOpenChange={setSheetOpen}
      />
    </div>
  )
}

function ConnectionCard({
  connection,
  onEdit,
}: {
  connection: CloudflareConnectionModel
  onEdit: () => void
}) {
  const { t } = useTranslation()

  return (
    <div className="rounded-2xl border border-border/70 bg-card/80 p-5 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <div className="flex size-11 items-center justify-center rounded-2xl bg-foreground text-background">
            <CloudIcon />
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <p className="font-semibold text-sm">{connection.display_name}</p>
              <Badge variant="secondary">Cloudflare</Badge>
            </div>
            <p className="text-sm text-muted-foreground">{connection.zone_id}</p>
          </div>
        </div>

        <Badge
          variant={connection.status === "active" ? "default" : "outline"}
          className="capitalize"
        >
          {connection.status}
        </Badge>
      </div>

      <div className="mt-5 grid gap-3 text-sm text-muted-foreground">
        <div className="rounded-xl bg-muted/60 px-3 py-2">
          <p className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground/80">
            {t("cloudflare.card.accountId")}
          </p>
          <p className="mt-1 font-mono text-xs text-foreground">{connection.account_id}</p>
        </div>
        <div className="rounded-xl bg-muted/60 px-3 py-2">
          <p className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground/80">
            {t("cloudflare.card.zoneId")}
          </p>
          <p className="mt-1 font-mono text-xs text-foreground">{connection.zone_id}</p>
        </div>
      </div>

      <div className="mt-4 flex items-center justify-between gap-3 border-t pt-4 text-xs text-muted-foreground">
        <div className="flex items-center gap-2">
          <KeyRoundIcon className="size-3.5" />
          <span>
            {connection.has_api_token
              ? t("cloudflare.card.tokenStored")
              : t("cloudflare.card.tokenMissing")}
          </span>
        </div>
        <Button variant="outline" size="sm" onClick={onEdit}>
          <PencilIcon data-icon="inline-start" />
          {t("cloudflare.actions.edit")}
        </Button>
      </div>
    </div>
  )
}

function ConnectionSheet({
  open,
  connection,
  onOpenChange,
}: {
  open: boolean
  connection: CloudflareConnectionModel | null
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const createMutation = useCreateCloudflareConnection()
  const updateMutation = useUpdateCloudflareConnection()
  const [draft, setDraft] = useState<FormDraft>(createDraft(connection))

  const isEditing = Boolean(connection)
  const isPending = createMutation.isPending || updateMutation.isPending

  useEffect(() => {
    if (open) {
      setDraft(createDraft(connection))
    }
  }, [connection, open])

  const close = (nextOpen: boolean) => {
    if (!nextOpen) {
      setDraft(createDraft(connection))
    }
    onOpenChange(nextOpen)
  }

  const handleSubmit = () => {
    if (isEditing && connection) {
      const payload: UpdateCloudflareConnectionModel = {
        display_name: draft.display_name,
        account_id: draft.account_id,
        zone_id: draft.zone_id,
      }
      if (draft.api_token.trim()) {
        payload.api_token = draft.api_token.trim()
      }
      updateMutation.mutate(
        { id: connection.id, payload },
        {
          onSuccess: () => {
            toast.success(t("cloudflare.toast.updated"))
            close(false)
          },
          onError: (error) => {
            toast.error(getErrorMessage(error))
          },
        }
      )
      return
    }

    const payload: CreateCloudflareConnectionModel = {
      display_name: draft.display_name,
      account_id: draft.account_id,
      zone_id: draft.zone_id,
      api_token: draft.api_token,
    }
    createMutation.mutate(payload, {
      onSuccess: () => {
        toast.success(t("cloudflare.toast.created"))
        close(false)
      },
      onError: (error) => {
        toast.error(getErrorMessage(error))
      },
    })
  }

  const formValid =
    draft.display_name.trim() &&
    draft.account_id.trim() &&
    draft.zone_id.trim() &&
    (isEditing || draft.api_token.trim())

  return (
    <Sheet open={open} onOpenChange={close}>
      <SheetContent className="flex flex-col sm:max-w-xl">
        <SheetHeader className="border-b pb-4">
          <SheetTitle>
            {isEditing ? t("cloudflare.sheet.editTitle") : t("cloudflare.sheet.createTitle")}
          </SheetTitle>
          <SheetDescription>{t("cloudflare.sheet.description")}</SheetDescription>
        </SheetHeader>

        <div className="flex-1 py-6">
          <FieldGroup>
            <Field>
              <FieldLabel>{t("cloudflare.form.displayName")}</FieldLabel>
              <Input
                value={draft.display_name}
                placeholder="Production account"
                onChange={(event) =>
                  setDraft((current) => ({
                    ...current,
                    display_name: event.target.value,
                  }))
                }
                disabled={isPending}
              />
            </Field>

            <Field>
              <FieldLabel>{t("cloudflare.form.accountId")}</FieldLabel>
              <Input
                className="font-mono"
                value={draft.account_id}
                placeholder="cloudflare-account-id"
                onChange={(event) =>
                  setDraft((current) => ({
                    ...current,
                    account_id: event.target.value,
                  }))
                }
                disabled={isPending}
              />
            </Field>

            <Field>
              <FieldLabel>{t("cloudflare.form.zoneId")}</FieldLabel>
              <Input
                className="font-mono"
                value={draft.zone_id}
                placeholder="cloudflare-zone-id"
                onChange={(event) =>
                  setDraft((current) => ({
                    ...current,
                    zone_id: event.target.value,
                  }))
                }
                disabled={isPending}
              />
            </Field>

            <Field>
              <FieldLabel>{t("cloudflare.form.apiToken")}</FieldLabel>
              <Input
                type="password"
                value={draft.api_token}
                placeholder={
                  isEditing ? t("cloudflare.form.apiTokenOptional") : "cf-api-token"
                }
                onChange={(event) =>
                  setDraft((current) => ({
                    ...current,
                    api_token: event.target.value,
                  }))
                }
                disabled={isPending}
              />
              <FieldDescription>
                {isEditing
                  ? t("cloudflare.form.apiTokenEditHint")
                  : t("cloudflare.form.apiTokenHint")}
              </FieldDescription>
            </Field>
          </FieldGroup>
        </div>

        <SheetFooter className="border-t pt-4">
          <Button variant="outline" onClick={() => close(false)} disabled={isPending}>
            {t("cloudflare.actions.cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={isPending || !formValid}>
            {isEditing ? t("cloudflare.actions.update") : t("cloudflare.actions.create")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}

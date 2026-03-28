import { useEffect, useState } from "react"
import { toast } from "sonner"
import {
  ArrowLeftIcon,
  ExternalLinkIcon,
  FolderGit2Icon,
  GithubIcon,
  KeyIcon,
  PlusIcon,
  ShieldCheckIcon,
} from "lucide-react"
import { useTranslation } from "react-i18next"

import type {
  BeginGitHubAppManifestResponseModel,
  SourceConnectionModel,
} from "@/@types/models"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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
  useBeginGitHubAppManifest,
  useConnectPAT,
  useGetSourceList,
} from "@/hooks/api/use-source"
import { getErrorMessage } from "@/lib/get-error-message"

function submitGitHubManifest(result: BeginGitHubAppManifestResponseModel) {
  const form = document.createElement("form")
  form.method = "post"
  form.action = result.create_url

  const input = document.createElement("input")
  input.type = "hidden"
  input.name = "manifest"
  input.value = JSON.stringify(result.manifest)

  form.appendChild(input)
  document.body.appendChild(form)
  form.submit()
}

function SourceCard({ source }: { source: SourceConnectionModel }) {
  const { t } = useTranslation()
  const connectionType =
    typeof source.metadata.connection_type === "string"
      ? source.metadata.connection_type
      : null
  const accountType =
    typeof source.metadata.account_type === "string"
      ? source.metadata.account_type
      : null

  const isPAT = connectionType === "github_pat"

  return (
    <div className="rounded-2xl border border-border/70 bg-card/80 p-5 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <div className="flex size-11 items-center justify-center rounded-2xl bg-foreground text-background">
            {isPAT ? <KeyIcon className="size-5" /> : <GithubIcon className="size-5" />}
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <p className="font-semibold text-sm">{source.display_name}</p>
              <Badge variant="secondary" className="capitalize">
                {source.provider}
              </Badge>
            </div>
            <p className="text-sm text-muted-foreground">
              {source.account_identifier}
            </p>
          </div>
        </div>

        <Badge
          variant={source.status === "active" ? "default" : "outline"}
          className="capitalize"
        >
          {source.status}
        </Badge>
      </div>

      <div className="mt-5 grid gap-3 text-sm text-muted-foreground sm:grid-cols-2">
        <div className="rounded-xl bg-muted/60 px-3 py-2">
          <p className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground/80">
            {t("sources.card.connectionType")}
          </p>
          <p className="mt-1 font-medium text-foreground">
            {isPAT ? t("sources.card.githubPat") : t("sources.card.githubApp")}
          </p>
        </div>
        <div className="rounded-xl bg-muted/60 px-3 py-2">
          <p className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground/80">
            {t("sources.card.accountType")}
          </p>
          <p className="mt-1 font-medium text-foreground">
            {accountType || t("sources.card.unknown")}
          </p>
        </div>
      </div>

      <div className="mt-4 flex items-center justify-between gap-3 border-t pt-4 text-xs text-muted-foreground">
        <span>
          {t("sources.card.connectedOn", { date: source.created_at })}
        </span>
        <span>#{source.external_id}</span>
      </div>
    </div>
  )
}

type AddSourceMode = "select" | "pat"

function AddSourceSheet({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const beginManifest = useBeginGitHubAppManifest()
  const connectPAT = useConnectPAT()

  const [mode, setMode] = useState<AddSourceMode>("select")
  const [patToken, setPatToken] = useState("")
  const [patDisplayName, setPatDisplayName] = useState("")

  const handleClose = (v: boolean) => {
    if (!v) {
      setMode("select")
      setPatToken("")
      setPatDisplayName("")
    }
    onOpenChange(v)
  }

  const handleConnectGitHub = () => {
    beginManifest.mutate(
      {
        app_name: `tango-${window.location.hostname || "local"}`,
        redirect_to: `${window.location.origin}/sources`,
      },
      {
        onSuccess: (result) => {
          submitGitHubManifest(result)
        },
        onError: (error) => {
          toast.error(getErrorMessage(error))
        },
      }
    )
  }

  const handleConnectPAT = () => {
    connectPAT.mutate(
      { token: patToken, display_name: patDisplayName || undefined },
      {
        onSuccess: () => {
          toast.success(t("sources.pat.connected"))
          handleClose(false)
        },
        onError: (error) => {
          toast.error(getErrorMessage(error))
        },
      }
    )
  }

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className="flex flex-col sm:max-w-xl">
        <SheetHeader className="border-b pb-4">
          <SheetTitle>{t("sources.add.title")}</SheetTitle>
          <SheetDescription>{t("sources.add.description")}</SheetDescription>
        </SheetHeader>

        {mode === "select" ? (
          <>
            <div className="flex flex-1 flex-col gap-4 py-6">
              {/* GitHub App option */}
              <button
                type="button"
                onClick={handleConnectGitHub}
                disabled={beginManifest.isPending}
                className="group rounded-3xl border border-border bg-[linear-gradient(135deg,hsl(var(--foreground))_0%,hsl(var(--foreground))_55%,hsl(var(--primary))_100%)] p-[1px] text-left transition-transform hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-70"
              >
                <div className="rounded-[calc(theme(borderRadius.3xl)-1px)] bg-card px-5 py-5">
                  <div className="flex items-start justify-between gap-4">
                    <div className="space-y-3">
                      <div className="flex items-center gap-3">
                        <div className="flex size-11 items-center justify-center rounded-2xl bg-foreground text-background">
                          <GithubIcon className="size-5" />
                        </div>
                        <div>
                          <p className="font-semibold text-base">
                            {t("sources.providers.githubApp")}
                          </p>
                          <p className="text-sm text-muted-foreground">
                            {t("sources.providers.githubAppDescription")}
                          </p>
                        </div>
                      </div>

                      <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                        <span className="rounded-full bg-muted px-2.5 py-1">
                          {t("sources.providers.installFlow")}
                        </span>
                        <span className="rounded-full bg-muted px-2.5 py-1">
                          {t("sources.providers.repoAccess")}
                        </span>
                        <span className="rounded-full bg-muted px-2.5 py-1">
                          {t("sources.providers.noTokenPaste")}
                        </span>
                      </div>
                    </div>

                    <ExternalLinkIcon className="size-4 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
                  </div>
                </div>
              </button>

              {/* PAT option */}
              <button
                type="button"
                onClick={() => setMode("pat")}
                className="group rounded-3xl border border-border bg-[linear-gradient(135deg,hsl(var(--muted-foreground))_0%,hsl(var(--muted-foreground))_55%,hsl(var(--foreground))_100%)] p-[1px] text-left transition-transform hover:-translate-y-0.5"
              >
                <div className="rounded-[calc(theme(borderRadius.3xl)-1px)] bg-card px-5 py-5">
                  <div className="flex items-start justify-between gap-4">
                    <div className="space-y-3">
                      <div className="flex items-center gap-3">
                        <div className="flex size-11 items-center justify-center rounded-2xl bg-muted text-foreground">
                          <KeyIcon className="size-5" />
                        </div>
                        <div>
                          <p className="font-semibold text-base">
                            {t("sources.providers.pat")}
                          </p>
                          <p className="text-sm text-muted-foreground">
                            {t("sources.providers.patDescription")}
                          </p>
                        </div>
                      </div>

                      <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                        <span className="rounded-full bg-muted px-2.5 py-1">
                          {t("sources.providers.patDirectToken")}
                        </span>
                        <span className="rounded-full bg-muted px-2.5 py-1">
                          {t("sources.providers.patPrivate")}
                        </span>
                        <span className="rounded-full bg-muted px-2.5 py-1">
                          {t("sources.providers.patNoOAuth")}
                        </span>
                      </div>
                    </div>

                    <ArrowLeftIcon className="size-4 rotate-180 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
                  </div>
                </div>
              </button>
            </div>

            <SheetFooter className="border-t pt-4">
              <Button variant="outline" onClick={() => handleClose(false)}>
                {t("sources.add.cancel")}
              </Button>
            </SheetFooter>
          </>
        ) : (
          <>
            <div className="flex flex-1 flex-col gap-5 py-6">
              <div className="flex items-center gap-3 rounded-2xl border bg-muted/40 px-4 py-3">
                <div className="flex size-9 items-center justify-center rounded-xl bg-muted text-foreground">
                  <KeyIcon className="size-4" />
                </div>
                <div>
                  <p className="font-semibold text-sm">{t("sources.providers.pat")}</p>
                  <p className="text-xs text-muted-foreground">
                    {t("sources.providers.patDescription")}
                  </p>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="pat-token">{t("sources.pat.tokenLabel")}</Label>
                <Input
                  id="pat-token"
                  type="password"
                  placeholder={t("sources.pat.tokenPlaceholder")}
                  value={patToken}
                  onChange={(e) => setPatToken(e.target.value)}
                  autoComplete="off"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="pat-display-name">
                  {t("sources.pat.displayNameLabel")}
                </Label>
                <Input
                  id="pat-display-name"
                  placeholder={t("sources.pat.displayNamePlaceholder")}
                  value={patDisplayName}
                  onChange={(e) => setPatDisplayName(e.target.value)}
                />
              </div>
            </div>

            <SheetFooter className="flex-row justify-between border-t pt-4">
              <Button
                variant="ghost"
                onClick={() => setMode("select")}
                disabled={connectPAT.isPending}
              >
                <ArrowLeftIcon className="mr-1 size-4" />
                {t("sources.pat.back")}
              </Button>
              <Button
                onClick={handleConnectPAT}
                disabled={!patToken.trim() || connectPAT.isPending}
              >
                {connectPAT.isPending
                  ? t("sources.pat.connecting")
                  : t("sources.pat.connect")}
              </Button>
            </SheetFooter>
          </>
        )}
      </SheetContent>
    </Sheet>
  )
}

export function SourcesPage() {
  const { t } = useTranslation()
  const [sheetOpen, setSheetOpen] = useState(false)
  const { data, isLoading } = useGetSourceList()

  useEffect(() => {
    const url = new URL(window.location.href)
    if (url.searchParams.get("github_connected") !== "1") {
      return
    }

    toast.success(t("sources.githubConnected"))
    url.searchParams.delete("github_connected")
    window.history.replaceState({}, "", url.toString())
  }, [t])

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<FolderGit2Icon className="size-5" />}
        title={t("sources.page.title")}
        description={t("sources.page.description")}
        titleMeta={`${data?.length ?? 0}`}
        headerRight={
          <Button size="sm" onClick={() => setSheetOpen(true)}>
            <PlusIcon data-icon="inline-start" />
            {t("sources.add.action")}
          </Button>
        }
      />

      <SectionCard
        icon={<ShieldCheckIcon className="size-5" />}
        title={t("sources.connected.title")}
        description={t("sources.connected.description")}
      >
        {isLoading ? (
          <div className="grid gap-4 lg:grid-cols-2">
            {Array.from({ length: 2 }).map((_, index) => (
              <Skeleton key={index} className="h-48 rounded-2xl" />
            ))}
          </div>
        ) : !data || data.length === 0 ? (
          <div className="rounded-3xl border border-dashed bg-muted/40 px-6 py-10 text-center">
            <p className="font-medium">{t("sources.empty.title")}</p>
            <p className="mt-2 text-sm text-muted-foreground">
              {t("sources.empty.description")}
            </p>
            <Button className="mt-5" onClick={() => setSheetOpen(true)}>
              {t("sources.empty.cta")}
            </Button>
          </div>
        ) : (
          <div className="grid gap-4 lg:grid-cols-2">
            {data.map((source) => (
              <SourceCard key={source.id} source={source} />
            ))}
          </div>
        )}
      </SectionCard>

      <AddSourceSheet open={sheetOpen} onOpenChange={setSheetOpen} />
    </div>
  )
}

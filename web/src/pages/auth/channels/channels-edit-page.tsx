import { Link, useNavigate } from "@tanstack/react-router"

import { type CreateChannelModel, type UpdateChannelModel } from "@/@types/models"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import {
  useGetChannelById,
  useGetChannelQRCode,
  useUpdateChannel,
} from "@/hooks/api/use-channel"
import { actionIcons, appIcons } from "@/lib/icons"
import { ChannelForm } from "@/routes/_auth/channels/components/-channel-form"
import { useTranslation } from "react-i18next"

type ChannelsEditPageProps = {
  channelId: string
}

export default function ChannelsEditPage({ channelId }: ChannelsEditPageProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data: channel, isLoading, isError } = useGetChannelById(channelId)
  const { data: qrCode, isLoading: isLoadingQRCode } = useGetChannelQRCode(
    channel?.kind === "whatsapp" ? channelId : ""
  )
  const updateChannelMutation = useUpdateChannel()
  const ChannelsIcon = appIcons.channels
  const BackIcon = actionIcons.back

  const handleSubmit = async (
    values: CreateChannelModel | UpdateChannelModel,
    _submitAction: "save" | "saveAndContinue"
  ) => {
    try {
      await updateChannelMutation.mutateAsync({
        channelId,
        payload: values as UpdateChannelModel,
      })
      navigate({ to: "/channels" })
    } catch {
      // Global mutation toast handles server-side failures.
    }
  }

  return (
    <>
      <PageHeaderCard
        icon={<ChannelsIcon />}
        title={t("channels.editTitle")}
        description={t("channels.editDescription")}
        headerRight={
          <Button asChild variant="outline">
            <Link to="/channels">
              <BackIcon data-icon="inline-start" />
              {t("channels.actions.backToChannels")}
            </Link>
          </Button>
        }
      />

      {isLoading ? (
        <Card>
          <CardContent className="flex flex-col gap-3 pt-6">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-32 w-full" />
          </CardContent>
        </Card>
      ) : isError || !channel ? (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {t("channels.errors.loadChannel")}
        </div>
      ) : (
        <div className="flex flex-col gap-6">
          <SectionCard
            icon={<ChannelsIcon />}
            title={t("channels.editPageTitle")}
            description={t("channels.editDescription")}
          >
            <ChannelForm
              mode="update"
              initialValues={channel}
              pending={updateChannelMutation.isPending}
              onSubmit={handleSubmit}
            />
          </SectionCard>

          {channel.kind === "whatsapp" ? (
            <SectionCard
              title={t("channels.qr.title")}
              description={t("channels.qr.description")}
            >
              {isLoadingQRCode ? (
                <div className="flex flex-col gap-3">
                  <Skeleton className="h-5 w-40" />
                  <Skeleton className="size-64 rounded-2xl" />
                </div>
              ) : qrCode?.qr_code ? (
                <div className="flex flex-col gap-4">
                  <div className="overflow-hidden rounded-2xl border bg-white p-4 shadow-sm">
                    <img
                      alt={t("channels.qr.alt")}
                      className="size-64 rounded-lg object-contain"
                      src={resolveQRCodeSource(qrCode.qr_code)}
                    />
                  </div>
                  <p className="text-sm text-muted-foreground">
                    {t("channels.qr.help")}
                  </p>
                </div>
              ) : (
                <div className="rounded-xl border border-dashed px-4 py-6 text-sm text-muted-foreground">
                  {t("channels.qr.empty")}
                </div>
              )}
            </SectionCard>
          ) : null}
        </div>
      )}
    </>
  )
}

function resolveQRCodeSource(qrCode: string) {
  if (
    qrCode.startsWith("data:") ||
    qrCode.startsWith("http://") ||
    qrCode.startsWith("https://") ||
    qrCode.startsWith("/")
  ) {
    return qrCode
  }

  return `data:image/png;base64,${qrCode}`
}

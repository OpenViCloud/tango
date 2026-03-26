import { useNavigate } from "@tanstack/react-router"

import { type CreateChannelModel } from "@/@types/models"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Button } from "@/components/ui/button"
import { useCreateChannel } from "@/hooks/api/use-channel"
import { actionIcons, appIcons } from "@/lib/icons"
import { ChannelForm } from "@/routes/_auth/channels/components/-channel-form"
import { useTranslation } from "react-i18next"

export default function ChannelsCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const createChannelMutation = useCreateChannel()
  const ChannelsIcon = appIcons.channels
  const BackIcon = actionIcons.back

  const handleSubmit = async (
    values: CreateChannelModel,
    submitAction: "save" | "saveAndContinue"
  ) => {
    try {
      const channel = await createChannelMutation.mutateAsync(values)

      if (submitAction === "saveAndContinue") {
        navigate({
          to: "/channels/$channelId/edit",
          params: { channelId: channel.id },
        })
        return
      }

      navigate({ to: "/channels" })
    } catch {
      // Global mutation toast handles server-side failures.
    }
  }

  return (
    <>
      <PageHeaderCard
        icon={<ChannelsIcon />}
        title={t("channels.createTitle")}
        description={t("channels.createDescription")}
        headerRight={
          <Button
            type="button"
            variant="outline"
            onClick={() => navigate({ to: "/channels" })}
          >
            <BackIcon data-icon="inline-start" />
            {t("channels.actions.backToChannels")}
          </Button>
        }
      />

      <SectionCard contentClassName="flex flex-col gap-6">
        <p className="text-sm text-muted-foreground">
          {t("channels.createHelper")}
        </p>

        <ChannelForm
          mode="create"
          pending={createChannelMutation.isPending}
          showSaveAndContinue
          onSubmit={handleSubmit}
        />
      </SectionCard>
    </>
  )
}

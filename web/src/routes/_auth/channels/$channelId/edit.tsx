import { createFileRoute } from "@tanstack/react-router"

import ChannelsEditPage from "@/pages/auth/channels/channels-edit-page"

export const Route = createFileRoute("/_auth/channels/$channelId/edit")({
  component: ChannelEditRoute,
})

function ChannelEditRoute() {
  const { channelId } = Route.useParams()

  return <ChannelsEditPage channelId={channelId} />
}

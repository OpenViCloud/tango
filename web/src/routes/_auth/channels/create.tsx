import { createFileRoute } from "@tanstack/react-router"

import ChannelsCreatePage from "@/pages/auth/channels/channels-create-page"

export const Route = createFileRoute("/_auth/channels/create")({
  component: ChannelsCreatePage,
})

import { createFileRoute } from "@tanstack/react-router"

import ChannelsListPage from "@/pages/auth/channels/channels-list-page"

export const Route = createFileRoute("/_auth/channels/")({
  component: ChannelsListPage,
})

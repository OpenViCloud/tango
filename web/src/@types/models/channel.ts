import { z } from "zod"

import type { BaseRequestModel } from "./common"

const jsonObjectSchema = z.record(z.string(), z.unknown())

export const channelKindSchema = z.enum([
  "discord",
  "telegram",
  "whatsapp",
  "slack",
  "web",
])

export const channelStatusSchema = z.enum(["pending", "active", "disabled"])

export const createChannelSchema = z.object({
  name: z.string().min(1, "validation.required"),
  kind: channelKindSchema,
  credentials: jsonObjectSchema.default({}),
  settings: jsonObjectSchema,
})

export type CreateChannelModel = z.infer<typeof createChannelSchema>

export const updateChannelSchema = createChannelSchema.extend({
  replace_credentials: z.boolean().default(false),
})

export type UpdateChannelModel = z.infer<typeof updateChannelSchema>

export const channelSchema = z.object({
  id: z.string(),
  name: z.string(),
  kind: channelKindSchema,
  status: channelStatusSchema,
  has_credentials: z.boolean(),
  settings: jsonObjectSchema,
  created_at: z.string(),
  updated_at: z.string(),
})

export type ChannelModel = z.infer<typeof channelSchema>

export const channelQRCodeSchema = z.object({
  id: z.string(),
  qr_code: z.string(),
})

export type ChannelQRCodeModel = z.infer<typeof channelQRCodeSchema>

export const testChannelConnectionSchema = z.object({
  kind: channelKindSchema,
  credentials: jsonObjectSchema.default({}),
  settings: jsonObjectSchema.default({}),
})

export type TestChannelConnectionModel = z.infer<
  typeof testChannelConnectionSchema
>

export type GetChannelRequestModel = BaseRequestModel

export const discordRuntimeSchema = z.object({
  channel: z.string(),
  running: z.boolean(),
  token_configured: z.boolean(),
  require_mention: z.boolean(),
  enable_typing: z.boolean(),
  enable_message_content_intent: z.boolean(),
  allowed_user_ids: z.array(z.string()),
})

export type DiscordRuntimeModel = z.infer<typeof discordRuntimeSchema>

export const discordRuntimeRequestSchema = z.object({
  token: z.string().min(1, "validation.required"),
  require_mention: z.boolean(),
  enable_typing: z.boolean(),
  enable_message_content_intent: z.boolean(),
  allowed_user_ids: z.array(z.string()),
})

export type DiscordRuntimeRequestModel = z.infer<
  typeof discordRuntimeRequestSchema
>

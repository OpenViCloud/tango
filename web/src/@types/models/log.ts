import { z } from "zod"

export const logLevelSchema = z.enum(["", "error", "warn", "info", "debug"])

export const getRuntimeLogsRequestSchema = z.object({
  lines: z.number().int().positive().max(1000).default(200),
  traceId: z.string().default(""),
  date: z.string().default(""),
  level: logLevelSchema.default(""),
  contains: z.string().default(""),
})

export type GetRuntimeLogsRequestModel = z.infer<
  typeof getRuntimeLogsRequestSchema
>

export const runtimeLogsSchema = z.object({
  file_path: z.string(),
  lines: z.array(z.string()),
  items: z.array(
    z.object({
      raw: z.string(),
      time: z.string().optional(),
      level: z.string().optional(),
      message: z.string().optional(),
      trace_id: z.string().optional(),
      method: z.string().optional(),
      path: z.string().optional(),
      status: z.number().optional(),
      query: z.string().optional(),
      client_ip: z.string().optional(),
      fields: z.record(z.string(), z.string()).optional(),
    })
  ),
})

export type RuntimeLogsModel = z.infer<typeof runtimeLogsSchema>

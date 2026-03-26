import { z } from "zod"
import type { BaseRequestModel } from "./common"

export type BuildJobStatus =
  | "queued"
  | "cloning"
  | "detecting"
  | "generating"
  | "building"
  | "done"
  | "failed"
  | "canceled"

export type BuildJobModel = {
  id: string
  status: BuildJobStatus
  git_url: string
  git_branch: string
  image_tag: string
  logs: string
  error_msg?: string
  started_at?: string
  finished_at?: string
  created_at: string
  updated_at: string
}

export const createBuildJobSchema = z.object({
  git_url: z.string().min(1, "Git URL is required"),
  git_branch: z.string().optional(),
  image_tag: z.string().min(1, "Image tag is required"),
})

export type CreateBuildJobModel = z.infer<typeof createBuildJobSchema>

export type GetBuildJobListRequestModel = BaseRequestModel & {
  status?: string
}

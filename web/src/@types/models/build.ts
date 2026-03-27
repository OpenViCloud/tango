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
  source_type: "git" | "upload"
  build_mode: "auto" | "dockerfile"
  git_url: string
  git_branch: string
  archive_name?: string
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
  build_mode: z.enum(["auto", "dockerfile"]),
  image_tag: z.string().min(1, "Image tag is required"),
})

export const uploadBuildJobSchema = z.object({
  image_tag: z.string().min(1, "Image tag is required"),
  build_mode: z.enum(["auto", "dockerfile"]),
})

export type CreateBuildJobModel = z.infer<typeof createBuildJobSchema>
export type UploadBuildJobModel = z.infer<typeof uploadBuildJobSchema>

export type GetBuildJobListRequestModel = BaseRequestModel & {
  status?: string
}

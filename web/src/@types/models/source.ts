export type SourceConnectionModel = {
  id: string
  source_provider_id: string
  provider: string
  display_name: string
  account_identifier: string
  external_id: string
  status: string
  expires_at?: string
  last_used_at?: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export type GitRepositoryModel = {
  owner: string
  name: string
  full_name: string
  clone_url: string
  private: boolean
  default_ref: string
}

export type GitBranchModel = {
  name: string
  is_default: boolean
}

export type CreateGitHubAppManifestModel = {
  app_name: string
  redirect_to?: string
}

export type GitHubAppManifestModel = {
  name: string
  url: string
  redirect_url: string
  callback_urls?: string[]
  setup_url: string
  hook_attributes: {
    url: string
    active: boolean
  }
  public: boolean
  default_events?: string[]
  default_permissions: Record<string, string>
}

export type BeginGitHubAppManifestResponseModel = {
  create_url: string
  manifest: GitHubAppManifestModel
}

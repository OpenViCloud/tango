export type CloudflareConnectionModel = {
  id: string
  display_name: string
  account_id: string
  zone_id: string
  status: string
  has_api_token: boolean
  created_at: string
  updated_at: string
}

export type CreateCloudflareConnectionModel = {
  display_name: string
  account_id: string
  zone_id: string
  api_token: string
}

export type UpdateCloudflareConnectionModel = {
  display_name: string
  account_id: string
  zone_id: string
  api_token?: string
}

export type BaseRequestModel = {
  pageIndex?: number
  pageSize?: number
  searchText?: string
  orderBy?: string
  ascending?: boolean
}

export type ApiResponse<T> = {
  success: boolean
  traceId: string
  data: T
}

export type PagedResponse<T> = {
  items: T[]
  pageIndex: number
  pageSize: number
  totalItems: number
  totalPage: number
}

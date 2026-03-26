import type { ApiResponse } from "@/@types/models/common"

export const unwrapApiResponse = <T>(response: ApiResponse<T>): T => response.data

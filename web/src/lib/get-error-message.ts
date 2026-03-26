import axios from "axios"
import i18n from "@/i18n"

type ApiFieldError = {
  code?: string
  message?: string
}

type ApiErrorBody = {
  code?: string
  message?: string
  details?: Record<string, ApiFieldError[]>
}

type ApiErrorEnvelope = {
  error?: ApiErrorBody | string
}

export function getErrorMessage(error: unknown) {
  if (axios.isAxiosError(error)) {
    const apiError = extractApiError(error.response?.data)

    const localizedByCode = translateApiErrorCode(apiError?.code)
    if (localizedByCode) {
      return localizedByCode
    }

    if (apiError?.message && apiError.message.trim() !== "") {
      return apiError.message
    }

    const detailMessage = firstDetailMessage(apiError?.details)
    if (detailMessage) {
      return detailMessage
    }

    if (
      typeof error.response?.data?.error === "string" &&
      error.response.data.error.trim() !== ""
    ) {
      return error.response.data.error
    }

    if (typeof error.message === "string" && error.message.trim() !== "") {
      return error.message
    }
  }

  if (error instanceof Error && error.message.trim() !== "") {
    return error.message
  }

  if (typeof error === "string" && error.trim() !== "") {
    return error
  }

  return "Something went wrong."
}

function extractApiError(data: unknown): ApiErrorBody | undefined {
  if (!data || typeof data !== "object") {
    return undefined
  }
  const envelope = data as ApiErrorEnvelope
  if (!envelope.error || typeof envelope.error === "string") {
    return undefined
  }
  return envelope.error
}

function translateApiErrorCode(code: string | undefined): string | undefined {
  if (typeof code !== "string" || code.trim() === "") {
    return undefined
  }

  const normalizedCode = code.trim().toLowerCase()
  const key = `errors.codes.${normalizedCode}`

  if (!i18n.exists(key)) {
    return undefined
  }

  const translated = i18n.t(key)
  if (typeof translated !== "string" || translated.trim() === "") {
    return undefined
  }

  return translated
}

function firstDetailMessage(
  details: Record<string, ApiFieldError[]> | undefined
): string | undefined {
  if (!details) {
    return undefined
  }
  for (const fieldErrors of Object.values(details)) {
    for (const fieldError of fieldErrors) {
      if (typeof fieldError?.message === "string" && fieldError.message.trim()) {
        return fieldError.message
      }
    }
  }
  return undefined
}

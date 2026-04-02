import type { ResourceModel, ResourceTemplateModel } from "@/@types/models"

type ResourceVisual = {
  iconUrl?: string
  abbr: string
  color: string
}

const TYPE_COLORS: Record<string, string> = {
  db: "#336791",
  app: "#009639",
  service: "#FF6600",
}

function normalizeImageName(value?: string) {
  const trimmed = value?.trim().toLowerCase()
  if (!trimmed) return ""
  const withoutTag = trimmed.split(":")[0] ?? trimmed
  const parts = withoutTag.split("/").filter(Boolean)
  return parts[parts.length - 1] ?? withoutTag
}

function fallbackAbbr(resource: ResourceModel) {
  return resource.name.slice(0, 2).toUpperCase()
}

export function resolveResourceVisual(
  resource: ResourceModel,
  templates: ResourceTemplateModel[] = []
): ResourceVisual {
  const resourceImage = resource.image.trim().toLowerCase()
  const normalizedResourceImage = normalizeImageName(resource.image)

  const matchedTemplate =
    templates.find((template) => template.image.trim().toLowerCase() === resourceImage) ??
    templates.find(
      (template) => normalizeImageName(template.image) === normalizedResourceImage
    ) ??
    templates.find((template) => template.id.trim().toLowerCase() === normalizedResourceImage)

  if (matchedTemplate) {
    return {
      iconUrl: matchedTemplate.icon_url || undefined,
      abbr: matchedTemplate.abbr,
      color: matchedTemplate.color,
    }
  }

  return {
    abbr: fallbackAbbr(resource),
    color: TYPE_COLORS[resource.type] ?? "#6b7280",
  }
}

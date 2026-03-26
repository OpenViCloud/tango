import { CheckIcon, LanguagesIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

const languages = [
  { code: "en", label: "English", flagSrc: "/images/flags/en.webp" },
  { code: "vi", label: "Tiếng Việt", flagSrc: "/images/flags/vn.webp" },
] as const

export function LanguageSwitcher() {
  const { i18n } = useTranslation()
  const currentLanguage =
    languages.find((language) => language.code === i18n.language) ??
    languages[0]

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant="outline" size="sm" className="gap-2">
          <LanguagesIcon className="size-4" />
          <span className="flex items-center gap-2">
            <img
              src={currentLanguage.flagSrc}
              alt={currentLanguage.label}
              className="h-4 w-4 rounded-full object-cover"
              onError={(event) => {
                event.currentTarget.style.display = "none"
              }}
            />
            {currentLanguage.code.toUpperCase()}
          </span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-40">
        {languages.map((language) => (
          <DropdownMenuItem
            key={language.code}
            className="flex items-center justify-between gap-3"
            onClick={() => void i18n.changeLanguage(language.code)}
          >
            <span className="flex items-center gap-2">
              <img
                src={language.flagSrc}
                alt={language.label}
                className="h-4 w-4 rounded-full object-cover"
                onError={(event) => {
                  event.currentTarget.style.display = "none"
                }}
              />
              {language.label}
            </span>
            {i18n.language === language.code ? (
              <CheckIcon className="size-4 text-primary" />
            ) : null}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

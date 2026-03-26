import { useMemo, useState } from "react"
import { CheckIcon, ChevronsUpDownIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { cn } from "@/lib/utils"
import { useTranslation } from "react-i18next"

type AutocompleteOption = {
  value: string
  label: string
}

type AutocompleteFieldProps = {
  id: string
  label: string
  value?: string
  options: AutocompleteOption[]
  placeholder: string
  emptyLabel: string
  description?: string
  error?: string
  disabled?: boolean
  onChange: (value: string) => void
}

export function AutocompleteField({
  id,
  label,
  value,
  options,
  placeholder,
  emptyLabel,
  description,
  error,
  disabled = false,
  onChange,
}: AutocompleteFieldProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")

  const filteredOptions = useMemo(() => {
    const normalized = query.trim().toLowerCase()
    if (!normalized) {
      return options
    }

    return options.filter((option) =>
      option.label.toLowerCase().includes(normalized)
    )
  }, [options, query])

  const selectedLabel = options.find((option) => option.value === value)?.label

  return (
    <Field data-invalid={Boolean(error)}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            id={id}
            type="button"
            variant="outline"
            className={cn(
              "w-full justify-between font-normal",
              !selectedLabel && "text-muted-foreground"
            )}
            disabled={disabled}
            aria-invalid={Boolean(error)}
          >
            <span className="truncate">{selectedLabel || placeholder}</span>
            <ChevronsUpDownIcon data-icon="inline-end" />
          </Button>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-(--radix-popover-trigger-width) p-2">
          <div className="flex flex-col gap-2">
            <Input
              autoFocus
              placeholder={placeholder}
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <div className="max-h-64 overflow-y-auto">
              {filteredOptions.length === 0 ? (
                <div className="px-2 py-3 text-sm text-muted-foreground">
                  {emptyLabel}
                </div>
              ) : (
                <div className="flex flex-col gap-1">
                  {filteredOptions.map((option) => {
                    const isSelected = option.value === value

                    return (
                      <button
                        className={cn(
                          "flex w-full items-center justify-between rounded-lg px-2 py-2 text-left text-sm hover:bg-accent hover:text-accent-foreground",
                          isSelected && "bg-accent text-accent-foreground"
                        )}
                        key={option.value}
                        onClick={() => {
                          onChange(option.value)
                          setOpen(false)
                          setQuery("")
                        }}
                        type="button"
                      >
                        <span className="truncate">{option.label}</span>
                        {isSelected ? <CheckIcon className="size-4" /> : null}
                      </button>
                    )
                  })}
                </div>
              )}
            </div>
          </div>
        </PopoverContent>
      </Popover>

      {description ? <FieldDescription>{description}</FieldDescription> : null}
      {error ? <FieldError errors={[{ message: t(error) }]} /> : null}
    </Field>
  )
}

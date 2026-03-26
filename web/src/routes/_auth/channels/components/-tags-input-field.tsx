import { useState, type KeyboardEvent } from "react"
import {
  Controller,
  type Control,
  type FieldPath,
  type FieldValues,
} from "react-hook-form"
import { XIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

type TagsInputFieldProps<T extends FieldValues> = {
  name: FieldPath<T>
  control: Control<T>
  label: string
  placeholder?: string
  description?: string
}

export function TagsInputField<T extends FieldValues>({
  name,
  control,
  label,
  placeholder,
  description,
}: TagsInputFieldProps<T>) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState("")

  return (
    <Controller
      name={name}
      control={control}
      render={({ field, fieldState }) => {
        const values = Array.isArray(field.value)
          ? (field.value as string[])
          : []

        const commitDraft = () => {
          const nextValue = draft.trim()
          if (!nextValue || values.includes(nextValue)) {
            setDraft("")
            return
          }

          field.onChange([...values, nextValue])
          setDraft("")
        }

        const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
          if (event.key === "Enter" || event.key === ",") {
            event.preventDefault()
            commitDraft()
          }

          if (
            event.key === "Backspace" &&
            draft === "" &&
            values.length > 0
          ) {
            field.onChange(values.slice(0, -1))
          }
        }

        return (
          <Field data-invalid={fieldState.invalid}>
            <FieldLabel htmlFor={String(name)}>{label}</FieldLabel>
            <div
              className={cn(
                "flex min-h-9 w-full flex-wrap items-center gap-2 rounded-lg border border-input bg-background px-3 py-2 focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50",
                fieldState.invalid ? "border-destructive" : ""
              )}
            >
              {values.map((value) => (
                <Badge
                  className="gap-1 rounded-lg pr-1"
                  key={value}
                  variant="secondary"
                >
                  <span className="max-w-40 truncate">{value}</span>
                  <button
                    className="rounded-sm p-0.5 hover:bg-foreground/10"
                    onClick={() =>
                      field.onChange(values.filter((item) => item !== value))
                    }
                    type="button"
                  >
                    <XIcon className="size-3" />
                  </button>
                </Badge>
              ))}

              <Input
                className="h-7 min-w-40 flex-1 self-center border-0 px-0 py-0 text-sm leading-5 shadow-none caret-foreground focus-visible:border-0 focus-visible:ring-0"
                id={String(name)}
                placeholder={placeholder}
                value={draft}
                onBlur={commitDraft}
                onChange={(event) => setDraft(event.target.value)}
                onKeyDown={handleKeyDown}
              />
            </div>

            {description ? <FieldDescription>{description}</FieldDescription> : null}

            {fieldState.invalid ? (
              <FieldError
                errors={[
                  fieldState.error?.message
                    ? {
                        ...fieldState.error,
                        message: t(fieldState.error.message),
                      }
                    : fieldState.error,
                ]}
              />
            ) : null}
          </Field>
        )
      }}
    />
  )
}

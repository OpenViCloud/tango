import type { ReactNode } from "react"
import {
  Controller,
  type Control,
  type FieldPath,
  type FieldValues,
} from "react-hook-form"
import { useTranslation } from "react-i18next"

import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"

type ControlledFieldProps<T extends FieldValues> = {
  name: FieldPath<T>
  control: Control<T>
  label: string
  placeholder?: string
  description?: ReactNode
  type?: string
  multiline?: boolean
  rows?: number
  disabled?: boolean
}

export function ControlledField<T extends FieldValues>({
  name,
  control,
  label,
  placeholder,
  description,
  type = "text",
  multiline = false,
  rows = 3,
  disabled = false,
}: ControlledFieldProps<T>) {
  const { t } = useTranslation()

  return (
    <Controller
      name={name}
      control={control}
      render={({ field, fieldState }) => (
        <Field data-invalid={fieldState.invalid}>
          <FieldLabel htmlFor={String(name)}>{label}</FieldLabel>

          {multiline ? (
            <Textarea
              {...field}
              id={String(name)}
              value={(field.value as string | undefined) ?? ""}
              rows={rows}
              disabled={disabled}
              aria-invalid={fieldState.invalid}
              placeholder={placeholder}
            />
          ) : (
            <Input
              {...field}
              id={String(name)}
              type={type}
              value={(field.value as string | undefined) ?? ""}
              disabled={disabled}
              aria-invalid={fieldState.invalid}
              placeholder={placeholder}
            />
          )}

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
      )}
    />
  )
}

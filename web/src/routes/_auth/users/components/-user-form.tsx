import { zodResolver } from "@hookform/resolvers/zod"
import { useEffect } from "react"
import { type Resolver, useForm } from "react-hook-form"
import { z } from "zod"

import type { CreateUserModel } from "@/@types/models"
import { ControlledField } from "@/components/form/controlled-field"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useTranslation } from "react-i18next"

type UserFormValues = CreateUserModel

type UserFormProps = {
  mode: "create" | "update"
  initialValues?: Partial<UserFormValues>
  readOnlyEmail?: string
  status?: string
  pending?: boolean
  schema: z.ZodTypeAny
  showSaveAndContinue?: boolean
  onSubmit: (
    values: UserFormValues,
    submitAction: "save" | "saveAndContinue"
  ) => void
}

const emptyValues: UserFormValues = {
  email: "",
  first_name: "",
  last_name: "",
  password: "",
  phone: "",
  address: "",
}

export function UserForm({
  mode,
  initialValues,
  readOnlyEmail,
  status,
  pending = false,
  schema,
  showSaveAndContinue = false,
  onSubmit,
}: UserFormProps) {
  const { t } = useTranslation()
  const form = useForm<UserFormValues>({
    resolver: zodResolver(
      schema as never
    ) as unknown as Resolver<UserFormValues>,
    defaultValues: {
      ...emptyValues,
      ...initialValues,
    },
  })

  useEffect(() => {
    form.reset({
      ...emptyValues,
      ...initialValues,
    })
  }, [form, initialValues])

  const handleSave = form.handleSubmit((values) => onSubmit(values, "save"))
  const handleSaveAndContinue = form.handleSubmit((values) =>
    onSubmit(values, "saveAndContinue")
  )

  return (
    <form className="flex flex-col gap-6" onSubmit={handleSave}>
      <FieldGroup>
        {mode === "create" ? (
          <>
            <ControlledField
              name="email"
              control={form.control}
              label={t("identity.form.email")}
              type="email"
              placeholder={t("identity.form.emailPlaceholder")}
            />
            <ControlledField
              name="password"
              control={form.control}
              label={t("identity.form.password")}
              type="password"
              placeholder={t("identity.form.passwordPlaceholder")}
              description={t("identity.form.passwordHelp")}
            />
          </>
        ) : (
          <Field>
            <FieldLabel htmlFor="read-only-email">
              {t("identity.form.email")}
            </FieldLabel>
            <Input
              id="read-only-email"
              type="email"
              disabled
              value={readOnlyEmail ?? ""}
            />
            {status ? (
              <FieldDescription>
                {t("identity.form.statusLabel")}: {status}
              </FieldDescription>
            ) : null}
          </Field>
        )}

        <div className="grid gap-5 md:grid-cols-2">
          <ControlledField
            name="first_name"
            control={form.control}
            label={t("identity.form.firstName")}
          />
          <ControlledField
            name="last_name"
            control={form.control}
            label={t("identity.form.lastName")}
          />
        </div>

        <ControlledField
          name="phone"
          control={form.control}
          label={t("identity.form.phone")}
        />

        <ControlledField
          name="address"
          control={form.control}
          label={t("identity.form.address")}
          multiline
        />
      </FieldGroup>

      <div className="flex flex-wrap justify-end gap-3">
        {showSaveAndContinue ? (
          <Button
            type="button"
            variant="outline"
            disabled={pending}
            onClick={() => {
              void handleSaveAndContinue()
            }}
          >
            {t("identity.actions.saveAndContinue")}
          </Button>
        ) : null}

        <Button type="submit" disabled={pending}>
          {t("identity.actions.save")}
        </Button>
      </div>
    </form>
  )
}

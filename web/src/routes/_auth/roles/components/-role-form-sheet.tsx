import { zodResolver } from "@hookform/resolvers/zod"
import { useEffect } from "react"
import { type Resolver, useForm } from "react-hook-form"
import { z } from "zod"

import type { CreateRoleModel, RoleModel } from "@/@types/models"
import { ControlledField } from "@/components/form/controlled-field"
import { Button } from "@/components/ui/button"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { appIcons } from "@/lib/icons"
import { useTranslation } from "react-i18next"

type RoleFormSheetProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  mode: "create" | "update"
  role?: RoleModel | null
  pending?: boolean
  schema: z.ZodTypeAny
  onSubmit: (values: CreateRoleModel) => void | Promise<void>
}

const emptyValues: CreateRoleModel = {
  name: "",
  description: "",
}

export function RoleFormSheet({
  open,
  onOpenChange,
  mode,
  role,
  pending = false,
  schema,
  onSubmit,
}: RoleFormSheetProps) {
  const { t } = useTranslation()
  const RolesIcon = appIcons.roles
  const form = useForm<CreateRoleModel>({
    resolver: zodResolver(schema as never) as unknown as Resolver<CreateRoleModel>,
    defaultValues: role
      ? {
          name: role.name,
          description: role.description,
        }
      : emptyValues,
  })

  useEffect(() => {
    form.reset(
      role
        ? {
            name: role.name,
            description: role.description,
          }
        : emptyValues
    )
  }, [form, role, open])

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-lg">
        <SheetHeader className="gap-4 border-b pb-4">
          <div className="flex items-center gap-3">
            <div className="flex size-11 items-center justify-center rounded-xl bg-primary/10 text-primary">
              <RolesIcon />
            </div>
            <div className="flex flex-col gap-1">
              <SheetTitle>
                {mode === "create"
                  ? t("identity.roles.sheet.createTitle")
                  : t("identity.roles.sheet.editTitle")}
              </SheetTitle>
              <SheetDescription>
                {mode === "create"
                  ? t("identity.roles.sheet.createDescription")
                  : t("identity.roles.sheet.editDescription")}
              </SheetDescription>
            </div>
          </div>
        </SheetHeader>

        <form
          className="flex flex-1 flex-col"
          onSubmit={form.handleSubmit(async (values) => {
            await onSubmit({
              name: values.name.trim(),
              description: values.description.trim(),
            })
          })}
        >
          <div className="flex flex-1 flex-col gap-5 overflow-y-auto px-4 py-4">
            <ControlledField
              name="name"
              control={form.control}
              label={t("identity.roles.form.name")}
              placeholder={t("identity.roles.form.namePlaceholder")}
            />

            <ControlledField
              name="description"
              control={form.control}
              label={t("identity.roles.form.description")}
              placeholder={t("identity.roles.form.descriptionPlaceholder")}
              multiline
              rows={5}
            />
          </div>

          <SheetFooter className="border-t pt-4 sm:flex-row sm:justify-end">
            <Button
              type="button"
              variant="outline"
              disabled={pending}
              onClick={() => onOpenChange(false)}
            >
              {t("identity.roles.sheet.cancel")}
            </Button>
            <Button type="submit" disabled={pending}>
              {t("identity.actions.save")}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

import { useNavigate } from "@tanstack/react-router"

import { createUserSchema, type CreateUserModel } from "@/@types/models"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Button } from "@/components/ui/button"
import { useCreateUser } from "@/hooks/api/use-user"
import { actionIcons, appIcons } from "@/lib/icons"
import { UserForm } from "@/routes/_auth/users/components/-user-form"
import { useTranslation } from "react-i18next"

export default function UsersCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const createUserMutation = useCreateUser()
  const UsersIcon = appIcons.users
  const BackIcon = actionIcons.back

  const handleSubmit = async (
    values: CreateUserModel,
    submitAction: "save" | "saveAndContinue"
  ) => {
    try {
      const user = await createUserMutation.mutateAsync({
        ...values,
        email: values.email.trim(),
        first_name: values.first_name.trim(),
        last_name: values.last_name.trim(),
        phone: values.phone?.trim(),
        address: values.address?.trim(),
      })
      if (submitAction === "saveAndContinue") {
        navigate({ to: "/users/$userId/edit", params: { userId: user.id } })
        return
      }

      navigate({ to: "/users" })
    } catch {
      // Global mutation toast handles server-side failures.
    }
  }

  return (
    <>
      <PageHeaderCard
        icon={<UsersIcon />}
        title={t("identity.createTitle")}
        description={t("identity.createDescription")}
        headerRight={
          <Button
            type="button"
            variant="outline"
            onClick={() => navigate({ to: "/users" })}
          >
            <BackIcon data-icon="inline-start" />
            {t("identity.actions.backToUsers")}
          </Button>
        }
      />

      <SectionCard contentClassName="flex flex-col gap-6">
        <p className="text-sm text-muted-foreground">
          {t("identity.createHelper")}
        </p>

        <UserForm
          mode="create"
          pending={createUserMutation.isPending}
          schema={createUserSchema}
          showSaveAndContinue
          onSubmit={handleSubmit}
        />
      </SectionCard>
    </>
  )
}

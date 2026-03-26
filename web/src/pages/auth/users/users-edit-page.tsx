import { Link, useNavigate } from "@tanstack/react-router"

import { type CreateUserModel, updateUserSchema } from "@/@types/models"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { useGetUserById, useUpdateUser } from "@/hooks/api/use-user"
import { actionIcons, appIcons } from "@/lib/icons"
import { UserForm } from "@/routes/_auth/users/components/-user-form"
import { useTranslation } from "react-i18next"

type UsersEditPageProps = {
  userId: string
}

export default function UsersEditPage({ userId }: UsersEditPageProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data: user, isLoading, isError } = useGetUserById(userId)
  const updateUserMutation = useUpdateUser()
  const UsersIcon = appIcons.users
  const BackIcon = actionIcons.back

  const handleSubmit = async (
    values: CreateUserModel,
    _submitAction: "save" | "saveAndContinue"
  ) => {
    try {
      await updateUserMutation.mutateAsync({
        userId,
        payload: {
          first_name: values.first_name.trim(),
          last_name: values.last_name.trim(),
          phone: values.phone?.trim(),
          address: values.address?.trim(),
        },
      })
      navigate({ to: "/users" })
    } catch {
      // Global mutation toast handles server-side failures.
    }
  }

  return (
    <>
      <PageHeaderCard
        icon={<UsersIcon />}
        title={t("identity.editTitle")}
        description={t("identity.editDescription")}
        headerRight={
          <Button asChild variant="outline">
            <Link to="/users">
              <BackIcon data-icon="inline-start" />
              {t("identity.actions.backToUsers")}
            </Link>
          </Button>
        }
      />

      {isLoading ? (
        <Card>
          <CardContent className="flex flex-col gap-3 pt-6">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-32 w-full" />
          </CardContent>
        </Card>
      ) : isError || !user ? (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {t("identity.errors.loadUser")}
        </div>
      ) : (
        <SectionCard
          icon={<UsersIcon />}
          title={t("identity.editPageTitle")}
          description={t("identity.editDescription")}
        >
          <UserForm
            mode="update"
            initialValues={{
              email: user.email,
              first_name: user.first_name,
              last_name: user.last_name,
              phone: user.phone,
              address: user.address,
            }}
            readOnlyEmail={user.email}
            status={user.status}
            pending={updateUserMutation.isPending}
            schema={updateUserSchema}
            onSubmit={handleSubmit}
          />
        </SectionCard>
      )}
    </>
  )
}

import { Link } from "@tanstack/react-router"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { useChangePassword } from "@/hooks/api/use-auth"
import { useGetCurrentUser } from "@/hooks/api/use-user"
import { actionIcons, appIcons } from "@/lib/icons"
import { useAuthStore } from "@/store/auth"

const AccountIcon = appIcons.users
const EditIcon = actionIcons.edit

function AccountRow({
  label,
  value,
}: {
  label: string
  value: string
}) {
  return (
    <div className="grid gap-1 rounded-xl border border-border/70 bg-background/60 px-4 py-3 sm:grid-cols-[160px_minmax(0,1fr)] sm:items-center sm:gap-4">
      <div className="text-sm text-muted-foreground">{label}</div>
      <div className="min-w-0 break-words text-sm font-medium text-foreground">
        {value}
      </div>
    </div>
  )
}

export function AccountPage() {
  const { t } = useTranslation()
  const { data: user, isLoading, isError } = useGetCurrentUser()
  const changePasswordMutation = useChangePassword()
  const { clearError } = useAuthStore()
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeaderCard
          icon={<AccountIcon />}
          title={t("account.page.title")}
          description={t("account.page.description")}
        />
        <SectionCard>
          <div className="flex flex-col gap-3">
            <Skeleton className="h-14 rounded-xl" />
            <Skeleton className="h-14 rounded-xl" />
            <Skeleton className="h-14 rounded-xl" />
            <Skeleton className="h-32 rounded-xl" />
          </div>
        </SectionCard>
      </div>
    )
  }

  if (isError || !user) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeaderCard
          icon={<AccountIcon />}
          title={t("account.page.title")}
          description={t("account.page.description")}
        />
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {t("account.errors.load")}
        </div>
      </div>
    )
  }

  const displayName =
    [user.first_name, user.last_name].filter(Boolean).join(" ").trim() ||
    user.nickname ||
    user.email
  const passwordsMatch = newPassword.length > 0 && newPassword === confirmPassword
  const canSubmit =
    currentPassword.length >= 6 &&
    newPassword.length >= 6 &&
    confirmPassword.length >= 6 &&
    passwordsMatch

  const handleChangePassword = async () => {
    if (!canSubmit) return

    try {
      await changePasswordMutation.mutateAsync({
        current_password: currentPassword,
        new_password: newPassword,
      })
      clearError()
      toast.success(t("account.security.changed"))
      window.location.href = "/login"
    } catch {
      // Global mutation toast handles request failures.
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<AccountIcon />}
        title={t("account.page.title")}
        description={t("account.page.description")}
        headerRight={
          <Button asChild variant="outline">
            <Link to="/users/$userId/edit" params={{ userId: user.id }}>
              <EditIcon data-icon="inline-start" />
              {t("account.actions.editProfile")}
            </Link>
          </Button>
        }
      />

      <SectionCard
        icon={<AccountIcon />}
        title={t("account.profile.title")}
        description={t("account.profile.description")}
      >
        <div className="flex flex-col gap-3">
          <AccountRow label={t("account.fields.displayName")} value={displayName} />
          <AccountRow label={t("account.fields.email")} value={user.email} />
          <AccountRow
            label={t("account.fields.phone")}
            value={user.phone || t("account.empty")}
          />
          <AccountRow
            label={t("account.fields.address")}
            value={user.address || t("account.empty")}
          />
          <AccountRow label={t("account.fields.status")} value={user.status} />
          <AccountRow label={t("account.fields.createdAt")} value={user.created_at} />
        </div>
      </SectionCard>

      <SectionCard
        title={t("account.security.title")}
        description={t("account.security.description")}
      >
        <div className="flex flex-col gap-5">
          <p className="max-w-2xl text-sm text-muted-foreground">
            {t("account.security.helper")}
          </p>

          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="current-password">
                {t("account.security.currentPassword")}
              </FieldLabel>
              <Input
                id="current-password"
                type="password"
                autoComplete="current-password"
                value={currentPassword}
                onChange={(event) => setCurrentPassword(event.target.value)}
                disabled={changePasswordMutation.isPending}
              />
            </Field>

            <div className="grid gap-5 md:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="new-password">
                  {t("account.security.newPassword")}
                </FieldLabel>
                <Input
                  id="new-password"
                  type="password"
                  autoComplete="new-password"
                  value={newPassword}
                  onChange={(event) => setNewPassword(event.target.value)}
                  disabled={changePasswordMutation.isPending}
                />
                <FieldDescription>
                  {t("account.security.passwordHint")}
                </FieldDescription>
              </Field>

              <Field>
                <FieldLabel htmlFor="confirm-password">
                  {t("account.security.confirmPassword")}
                </FieldLabel>
                <Input
                  id="confirm-password"
                  type="password"
                  autoComplete="new-password"
                  value={confirmPassword}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  disabled={changePasswordMutation.isPending}
                />
                {confirmPassword.length > 0 && !passwordsMatch ? (
                  <FieldDescription className="text-destructive">
                    {t("account.security.passwordMismatch")}
                  </FieldDescription>
                ) : null}
              </Field>
            </div>
          </FieldGroup>

          <div className="flex justify-end">
            <Button
              type="button"
              disabled={!canSubmit || changePasswordMutation.isPending}
              onClick={handleChangePassword}
            >
              {changePasswordMutation.isPending
                ? t("account.security.changing")
                : t("account.actions.changePassword")}
            </Button>
          </div>
        </div>
      </SectionCard>
    </div>
  )
}

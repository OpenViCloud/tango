import { zodResolver } from "@hookform/resolvers/zod"
import { Link, useNavigate } from "@tanstack/react-router"
import { useEffect, useState } from "react"
import { Controller, useForm } from "react-hook-form"
import { useTranslation } from "react-i18next"

import { registerSchema, type RegisterRequestModel } from "@/@types/models"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { authService } from "@/services/api/auth-service"
import { useAuthStore } from "@/store/auth"

export default function RegisterPage() {
  const { t } = useTranslation()
  const { register, isLoading, error, clearError } = useAuthStore()
  const navigate = useNavigate()
  const [closed, setClosed] = useState(false)
  const [checking, setChecking] = useState(true)

  useEffect(() => {
    authService
      .setupStatus()
      .then((status) => {
        if (!status.setup_required) setClosed(true)
      })
      .catch(() => {})
      .finally(() => setChecking(false))
  }, [])

  const form = useForm<RegisterRequestModel>({
    resolver: zodResolver(registerSchema),
    defaultValues: { email: "", first_name: "", last_name: "", password: "" },
  })

  const handleSubmit = async (values: RegisterRequestModel) => {
    clearError()
    try {
      await register(values)
      navigate({ to: "/dashboard" })
    } catch {
      // error state is handled in the auth store
    }
  }

  if (checking) return null

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-muted/30 px-6 py-10">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,hsl(var(--primary)/0.12),transparent_28%),radial-gradient(circle_at_bottom_left,hsl(var(--chart-2)/0.14),transparent_24%)]" />

      <Card className="relative z-10 w-full sm:max-w-md">
        <CardHeader>
          <CardTitle>{t("auth.setupRequired")}</CardTitle>
          <CardDescription>{t("auth.registerDescription")}</CardDescription>
        </CardHeader>
        <CardContent>
          {closed ? (
            <div className="rounded-lg border border-destructive/20 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {t("auth.registrationClosed")}
            </div>
          ) : (
            <form
              id="register-form"
              className="flex flex-col gap-5"
              onSubmit={form.handleSubmit(handleSubmit)}
            >
              <FieldGroup>
                <div className="grid grid-cols-2 gap-4">
                  <Controller
                    name="first_name"
                    control={form.control}
                    render={({ field, fieldState }) => (
                      <Field data-invalid={fieldState.invalid}>
                        <FieldLabel htmlFor="register-first-name">
                          {t("auth.firstName")}
                        </FieldLabel>
                        <Input
                          {...field}
                          id="register-first-name"
                          placeholder={t("auth.firstNamePlaceholder")}
                          autoComplete="given-name"
                          onChange={(e) => { clearError(); field.onChange(e) }}
                        />
                        {fieldState.invalid && (
                          <FieldError errors={[fieldState.error]} />
                        )}
                      </Field>
                    )}
                  />
                  <Controller
                    name="last_name"
                    control={form.control}
                    render={({ field, fieldState }) => (
                      <Field data-invalid={fieldState.invalid}>
                        <FieldLabel htmlFor="register-last-name">
                          {t("auth.lastName")}
                        </FieldLabel>
                        <Input
                          {...field}
                          id="register-last-name"
                          placeholder={t("auth.lastNamePlaceholder")}
                          autoComplete="family-name"
                          onChange={(e) => { clearError(); field.onChange(e) }}
                        />
                        {fieldState.invalid && (
                          <FieldError errors={[fieldState.error]} />
                        )}
                      </Field>
                    )}
                  />
                </div>

                <Controller
                  name="email"
                  control={form.control}
                  render={({ field, fieldState }) => (
                    <Field data-invalid={fieldState.invalid}>
                      <FieldLabel htmlFor="register-email">
                        {t("auth.email")}
                      </FieldLabel>
                      <Input
                        {...field}
                        id="register-email"
                        type="email"
                        placeholder={t("auth.emailPlaceholder")}
                        autoComplete="email"
                        onChange={(e) => { clearError(); field.onChange(e) }}
                      />
                      {fieldState.invalid && (
                        <FieldError errors={[fieldState.error]} />
                      )}
                    </Field>
                  )}
                />

                <Controller
                  name="password"
                  control={form.control}
                  render={({ field, fieldState }) => (
                    <Field data-invalid={fieldState.invalid}>
                      <FieldLabel htmlFor="register-password">
                        {t("auth.password")}
                      </FieldLabel>
                      <Input
                        {...field}
                        id="register-password"
                        type="password"
                        placeholder={t("auth.passwordPlaceholder")}
                        autoComplete="new-password"
                        onChange={(e) => { clearError(); field.onChange(e) }}
                      />
                      <FieldDescription>{t("auth.passwordHelp")}</FieldDescription>
                      {fieldState.invalid && (
                        <FieldError errors={[fieldState.error]} />
                      )}
                    </Field>
                  )}
                />
              </FieldGroup>

              {error ? (
                <div className="rounded-lg border border-destructive/20 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  {error}
                </div>
              ) : null}
            </form>
          )}
        </CardContent>
        <CardFooter className="flex-col items-stretch gap-4">
          {!closed && (
            <Button
              type="submit"
              form="register-form"
              size="lg"
              disabled={isLoading}
            >
              {isLoading ? t("auth.registerInProgress") : t("auth.register")}
            </Button>
          )}
          <p className="text-center text-sm text-muted-foreground">
            {t("auth.alreadyHaveAccount")}{" "}
            <Link
              to="/login"
              className="font-medium text-foreground underline-offset-4 hover:underline"
            >
              {t("auth.signIn")}
            </Link>
          </p>
        </CardFooter>
      </Card>
    </div>
  )
}

import { zodResolver } from "@hookform/resolvers/zod"
import { Link, useNavigate } from "@tanstack/react-router"
import { Controller, useForm } from "react-hook-form"

import { loginSchema, type LoginRequestModel } from "@/@types/models"
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
import { useAuthStore } from "@/store/auth"
import { useTranslation } from "react-i18next"

export default function LoginPage() {
  const { t } = useTranslation()
  const { login, isLoading, error, clearError } = useAuthStore()
  const navigate = useNavigate()

  const form = useForm<LoginRequestModel>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: "",
      password: "",
    },
  })

  const handleSubmit = async (values: LoginRequestModel) => {
    clearError()

    try {
      await login(values.email, values.password)
      navigate({ to: "/dashboard" })
    } catch {
      // error state is handled in the auth store
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-muted/30 px-6 py-10">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,hsl(var(--primary)/0.12),transparent_28%),radial-gradient(circle_at_bottom_left,hsl(var(--chart-2)/0.14),transparent_24%)]" />

      <Card className="relative z-10 w-full sm:max-w-md">
        <CardHeader>
          <CardTitle>{t("auth.login")}</CardTitle>
          <CardDescription>{t("auth.loginDescription")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form
            id="login-form"
            className="flex flex-col gap-5"
            onSubmit={form.handleSubmit(handleSubmit)}
          >
            <FieldGroup>
              <Controller
                name="email"
                control={form.control}
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid}>
                    <FieldLabel htmlFor="login-email">
                      {t("auth.email")}
                    </FieldLabel>
                    <Input
                      {...field}
                      id="login-email"
                      type="email"
                      aria-invalid={fieldState.invalid}
                      placeholder={t("auth.emailPlaceholder")}
                      autoComplete="email"
                      onChange={(event) => {
                        clearError()
                        field.onChange(event)
                      }}
                    />
                    <FieldDescription>{t("auth.emailHelp")}</FieldDescription>
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
                    <FieldLabel htmlFor="login-password">
                      {t("auth.password")}
                    </FieldLabel>
                    <Input
                      {...field}
                      id="login-password"
                      type="password"
                      aria-invalid={fieldState.invalid}
                      placeholder={t("auth.passwordPlaceholder")}
                      autoComplete="current-password"
                      onChange={(event) => {
                        clearError()
                        field.onChange(event)
                      }}
                    />
                    <FieldDescription>
                      {t("auth.passwordHelp")}
                    </FieldDescription>
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
        </CardContent>
        <CardFooter className="flex-col items-stretch gap-4">
          <Button
            type="submit"
            form="login-form"
            size="lg"
            disabled={isLoading}
          >
            {isLoading ? t("auth.loginInProgress") : t("auth.login")}
          </Button>
          <p className="text-center text-sm text-muted-foreground">
            {t("auth.noAccount")}{" "}
            <Link
              to="/"
              className="font-medium text-foreground underline-offset-4 hover:underline"
            >
              {t("auth.contactAdmin")}
            </Link>
          </p>
        </CardFooter>
      </Card>
    </div>
  )
}

import { zodResolver } from "@hookform/resolvers/zod"
import { useEffect } from "react"
import { Controller, type Resolver, useForm } from "react-hook-form"
import { z } from "zod"

import type {
  ChannelModel,
  CreateChannelModel,
  UpdateChannelModel,
} from "@/@types/models"
import { ControlledField } from "@/components/form/controlled-field"
import { TagsInputField } from "@/routes/_auth/channels/components/-tags-input-field"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { useTestChannelConnection } from "@/hooks/api/use-channel"
import { actionIcons } from "@/lib/icons"
import { useTranslation } from "react-i18next"

type ChannelFormValues = {
  name: string
  kind: ChannelModel["kind"]
  require_mention: boolean
  enable_typing: boolean
  allowed_user_ids: string[]
  replace_credentials: boolean
  discord_token: string
  telegram_token: string
  slack_bot_token: string
  slack_app_token: string
}

type ChannelFormProps = {
  mode: "create" | "update"
  initialValues?: Partial<ChannelModel>
  pending?: boolean
  showSaveAndContinue?: boolean
  onSubmit: (
    values: CreateChannelModel | UpdateChannelModel,
    submitAction: "save" | "saveAndContinue"
  ) => void
}

const emptyValues: ChannelFormValues = {
  name: "",
  kind: "telegram",
  require_mention: true,
  enable_typing: true,
  allowed_user_ids: [],
  replace_credentials: false,
  discord_token: "",
  telegram_token: "",
  slack_bot_token: "",
  slack_app_token: "",
}

const channelKindOptions = [
  {
    value: "telegram",
    imageSrc: "/images/channels/telegram.png",
  },
  {
    value: "discord",
    imageSrc: "/images/channels/discord.png",
  },
  {
    value: "whatsapp",
    imageSrc: "/images/channels/whatsapp.png",
  },
  {
    value: "slack",
    imageSrc: "/images/channels/slack.png",
  },
  {
    value: "web",
    imageSrc: "/images/channels/web.png",
  },
] as const satisfies ReadonlyArray<{
  value: ChannelModel["kind"]
  imageSrc?: string
}>

function createChannelFormSchema(mode: ChannelFormProps["mode"]) {
  return z
    .object({
      name: z.string().min(1, "validation.required"),
      kind: z.enum(["discord", "telegram", "whatsapp", "slack", "web"]),
      require_mention: z.boolean(),
      enable_typing: z.boolean(),
      allowed_user_ids: z.array(z.string()),
      replace_credentials: z.boolean(),
      discord_token: z.string(),
      telegram_token: z.string(),
      slack_bot_token: z.string(),
      slack_app_token: z.string(),
    })
    .superRefine((value, ctx) => {
      const shouldRequireCredentials =
        mode === "create" || value.replace_credentials

      if (!shouldRequireCredentials) {
        return
      }

      if (value.kind === "discord" && value.discord_token.trim() === "") {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["discord_token"],
          message: "validation.required",
        })
      }

      if (value.kind === "telegram" && value.telegram_token.trim() === "") {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["telegram_token"],
          message: "validation.required",
        })
      }

      if (value.kind === "slack") {
        if (value.slack_bot_token.trim() === "") {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ["slack_bot_token"],
            message: "validation.required",
          })
        }

        if (value.slack_app_token.trim() === "") {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ["slack_app_token"],
            message: "validation.required",
          })
        }
      }
    })
}

function buildSettingsPayload(values: ChannelFormValues) {
  return {
    require_mention: values.require_mention,
    enable_typing: values.enable_typing,
    allowed_user_ids: values.allowed_user_ids,
  }
}

function buildCredentialsPayload(values: ChannelFormValues) {
  switch (values.kind) {
    case "discord":
      return values.discord_token.trim()
        ? { token: values.discord_token.trim() }
        : {}
    case "telegram":
      return values.telegram_token.trim()
        ? { token: values.telegram_token.trim() }
        : {}
    case "slack":
      return {
        ...(values.slack_bot_token.trim()
          ? { bot_token: values.slack_bot_token.trim() }
          : {}),
        ...(values.slack_app_token.trim()
          ? { app_token: values.slack_app_token.trim() }
          : {}),
      }
    case "whatsapp":
    case "web":
      return {}
    default:
      return {}
  }
}

export function ChannelForm({
  mode,
  initialValues,
  pending = false,
  showSaveAndContinue = false,
  onSubmit,
}: ChannelFormProps) {
  const { t } = useTranslation()
  const testChannelConnectionMutation = useTestChannelConnection()
  const TestIcon = actionIcons.start

  const form = useForm<ChannelFormValues>({
    resolver: zodResolver(
      createChannelFormSchema(mode)
    ) as unknown as Resolver<ChannelFormValues>,
    defaultValues: {
      ...emptyValues,
      ...toFormValues(initialValues),
    },
  })

  useEffect(() => {
    form.reset({
      ...emptyValues,
      ...toFormValues(initialValues),
    })
  }, [form, initialValues])

  const selectedKind = form.watch("kind")
  const discordToken = form.watch("discord_token")
  const telegramToken = form.watch("telegram_token")
  const slackBotToken = form.watch("slack_bot_token")
  const slackAppToken = form.watch("slack_app_token")
  const shouldEditCredentials =
    mode === "create" || form.watch("replace_credentials")
  const canTestConnection = getCanTestConnection({
    discordToken,
    kind: selectedKind,
    slackAppToken,
    slackBotToken,
    telegramToken,
  })

  const submitValues = (
    values: ChannelFormValues,
    submitAction: "save" | "saveAndContinue"
  ) => {
    const payloadBase = {
      name: values.name.trim(),
      kind: values.kind,
      settings: buildSettingsPayload(values),
      credentials: buildCredentialsPayload(values),
    }

    if (mode === "create") {
      onSubmit(payloadBase, submitAction)
      return
    }

    onSubmit(
      {
        ...payloadBase,
        replace_credentials: values.replace_credentials,
      },
      submitAction
    )
  }

  const handleSave = form.handleSubmit((values) => {
    submitValues(values, "save")
  })

  const handleSaveAndContinue = form.handleSubmit((values) => {
    submitValues(values, "saveAndContinue")
  })

  const handleTestConnection = async () => {
    const credentialsFields = getCredentialFieldNames(selectedKind)
    const isValid = await form.trigger([
      "kind",
      "require_mention",
      "enable_typing",
      "allowed_user_ids",
      ...credentialsFields,
    ])

    if (!isValid) {
      return
    }

    const values = form.getValues()

    try {
      await testChannelConnectionMutation.mutateAsync({
        kind: values.kind,
        credentials: buildCredentialsPayload(values),
        settings: buildSettingsPayload(values),
      })
    } catch {
      // Mutation hook shows toast.
    }
  }

  return (
    <form className="flex flex-col gap-6" onSubmit={handleSave}>
      <FieldGroup>
        <ControlledField
          name="name"
          control={form.control}
          label={t("channels.form.name")}
          placeholder={t("channels.form.namePlaceholder")}
        />

        <Controller
          name="kind"
          control={form.control}
          render={({ field, fieldState }) => (
            <Field data-invalid={fieldState.invalid}>
              <FieldLabel>{t("channels.form.kind")}</FieldLabel>
              <Select onValueChange={field.onChange} value={field.value}>
                <SelectTrigger aria-invalid={fieldState.invalid}>
                  <ChannelKindValue value={field.value} />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {channelKindOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        <ChannelKindOption
                          imageSrc={option.imageSrc}
                          label={t(`channels.options.kind.${option.value}`)}
                          value={option.value}
                        />
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              {fieldState.invalid ? (
                <FieldError errors={[fieldState.error]} />
              ) : null}
            </Field>
          )}
        />

        <div className="grid gap-5 md:grid-cols-2">
          <Controller
            name="require_mention"
            control={form.control}
            render={({ field }) => (
              <Field orientation="horizontal">
                <FieldLabel htmlFor="channel-require-mention">
                  <div className="flex flex-col gap-1">
                    <span>{t("channels.form.requireMention")}</span>
                    <FieldDescription>
                      {t("channels.form.requireMentionHelp")}
                    </FieldDescription>
                  </div>
                </FieldLabel>
                <Switch
                  checked={field.value}
                  id="channel-require-mention"
                  onCheckedChange={field.onChange}
                />
              </Field>
            )}
          />

          <Controller
            name="enable_typing"
            control={form.control}
            render={({ field }) => (
              <Field orientation="horizontal">
                <FieldLabel htmlFor="channel-enable-typing">
                  <div className="flex flex-col gap-1">
                    <span>{t("channels.form.enableTyping")}</span>
                    <FieldDescription>
                      {t("channels.form.enableTypingHelp")}
                    </FieldDescription>
                  </div>
                </FieldLabel>
                <Switch
                  checked={field.value}
                  id="channel-enable-typing"
                  onCheckedChange={field.onChange}
                />
              </Field>
            )}
          />
        </div>

        <TagsInputField
          control={form.control}
          description={t("channels.form.allowedUserIdsHelp")}
          label={t("channels.form.allowedUserIds")}
          name="allowed_user_ids"
          placeholder={t("channels.form.allowedUserIdsPlaceholder")}
        />

        {mode === "update" ? (
          <Controller
            name="replace_credentials"
            control={form.control}
            render={({ field }) => (
              <Field>
                <div className="flex items-start gap-3 rounded-lg border px-4 py-3">
                  <Checkbox
                    checked={field.value}
                    onCheckedChange={(checked) =>
                      field.onChange(checked === true)
                    }
                    aria-label={t("channels.form.replaceCredentials")}
                  />
                  <div className="flex flex-col gap-1">
                    <FieldLabel>
                      {t("channels.form.replaceCredentials")}
                    </FieldLabel>
                    <FieldDescription>
                      {t("channels.form.replaceCredentialsHelp")}
                    </FieldDescription>
                  </div>
                </div>
              </Field>
            )}
          />
        ) : null}

        {shouldEditCredentials ? (
          <ChannelCredentialsFields
            control={form.control}
            kind={selectedKind}
            mode={mode}
          />
        ) : (
          <Field>
            <FieldDescription>
              {t("channels.form.credentialsPreserved")}
            </FieldDescription>
          </Field>
        )}
      </FieldGroup>

      <div className="flex flex-wrap justify-end gap-3">
        <Button
          type="button"
          variant="outline"
          disabled={
            pending ||
            testChannelConnectionMutation.isPending ||
            !canTestConnection
          }
          onClick={() => {
            void handleTestConnection()
          }}
        >
          <TestIcon data-icon="inline-start" />
          {testChannelConnectionMutation.isPending
            ? t("channels.actions.testingConnection")
            : t("channels.actions.testConnection")}
        </Button>

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

function ChannelCredentialsFields({
  control,
  kind,
  mode,
}: {
  control: ReturnType<typeof useForm<ChannelFormValues>>["control"]
  kind: ChannelModel["kind"]
  mode: ChannelFormProps["mode"]
}) {
  const { t } = useTranslation()

  if (kind === "telegram") {
    return (
      <ControlledField
        name="telegram_token"
        control={control}
        label={t("channels.form.telegramToken")}
        placeholder={t("channels.form.telegramTokenPlaceholder")}
        description={t("channels.form.telegramTokenHelp")}
        type="password"
      />
    )
  }

  if (kind === "discord") {
    return (
      <ControlledField
        name="discord_token"
        control={control}
        label={t("channels.form.discordToken")}
        placeholder={t("channels.form.discordTokenPlaceholder")}
        description={t("channels.form.discordTokenHelp")}
        type="password"
      />
    )
  }

  if (kind === "slack") {
    return (
      <div className="grid gap-5 md:grid-cols-2">
        <ControlledField
          name="slack_bot_token"
          control={control}
          label={t("channels.form.slackBotToken")}
          placeholder={t("channels.form.slackBotTokenPlaceholder")}
          description={t("channels.form.slackBotTokenHelp")}
          type="password"
        />
        <ControlledField
          name="slack_app_token"
          control={control}
          label={t("channels.form.slackAppToken")}
          placeholder={t("channels.form.slackAppTokenPlaceholder")}
          description={t("channels.form.slackAppTokenHelp")}
          type="password"
        />
      </div>
    )
  }

  if (kind === "whatsapp") {
    return (
      <Field>
        <FieldDescription>
          {t(
            mode === "create"
              ? "channels.form.whatsappCreateHelp"
              : "channels.form.whatsappEditHelp"
          )}
        </FieldDescription>
      </Field>
    )
  }

  return (
    <Field>
      <FieldDescription>
        {t("channels.form.noCredentialsRequired")}
      </FieldDescription>
    </Field>
  )
}

function toFormValues(
  initialValues?: Partial<ChannelModel>
): Partial<ChannelFormValues> {
  if (!initialValues) {
    return {}
  }

  return {
    name: initialValues.name,
    kind: initialValues.kind,
    require_mention: Boolean(initialValues.settings?.require_mention),
    enable_typing: Boolean(initialValues.settings?.enable_typing),
    allowed_user_ids: Array.isArray(initialValues.settings?.allowed_user_ids)
      ? (initialValues.settings.allowed_user_ids as string[])
      : [],
    replace_credentials: false,
  }
}

function getCredentialFieldNames(kind: ChannelModel["kind"]) {
  switch (kind) {
    case "discord":
      return ["discord_token"] as const
    case "telegram":
      return ["telegram_token"] as const
    case "slack":
      return ["slack_bot_token", "slack_app_token"] as const
    default:
      return [] as const
  }
}

function getCanTestConnection({
  kind,
  telegramToken,
  discordToken,
  slackBotToken,
  slackAppToken,
}: {
  kind: ChannelModel["kind"]
  telegramToken: string
  discordToken: string
  slackBotToken: string
  slackAppToken: string
}) {
  switch (kind) {
    case "telegram":
      return telegramToken.trim() !== ""
    case "discord":
      return discordToken.trim() !== ""
    case "slack":
      return slackBotToken.trim() !== "" && slackAppToken.trim() !== ""
    default:
      return false
  }
}

function ChannelKindValue({ value }: { value: ChannelModel["kind"] }) {
  const { t } = useTranslation()
  const option = channelKindOptions.find((item) => item.value === value)

  return (
    <ChannelKindOption
      imageSrc={option?.imageSrc}
      label={t(`channels.options.kind.${value}`)}
      value={value}
    />
  )
}

function ChannelKindOption({
  imageSrc,
  label,
  value,
}: {
  imageSrc?: string
  label: string
  value: ChannelModel["kind"]
}) {
  return (
    <span className="flex items-center gap-2">
      {imageSrc ? (
        <img
          alt=""
          className="size-8 rounded-sm object-contain"
          src={imageSrc}
        />
      ) : (
        <span className="flex size-4 items-center justify-center rounded-sm bg-muted text-[10px] font-semibold text-muted-foreground uppercase">
          {value.slice(0, 1)}
        </span>
      )}
      <span>{label}</span>
    </span>
  )
}

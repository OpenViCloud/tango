"use client"

import * as React from "react"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

type DeleteConfirmDialogProps = {
  trigger?: React.ReactNode
  open?: boolean
  onOpenChange?: (open: boolean) => void
  title?: string
  description: string
  confirmLabel?: string
  cancelLabel?: string
  confirmText?: string
  confirmTextLabel?: string
  confirmTextPlaceholder?: string
  onConfirm: () => void | Promise<void>
}

export function DeleteConfirmDialog({
  trigger,
  open: openProp,
  onOpenChange,
  title = "Delete item",
  description,
  confirmLabel = "Delete",
  cancelLabel = "Cancel",
  confirmText,
  confirmTextLabel = "Type to confirm",
  confirmTextPlaceholder,
  onConfirm,
}: DeleteConfirmDialogProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const [value, setValue] = React.useState("")
  const isControlled = openProp !== undefined
  const open = isControlled ? openProp : internalOpen

  const handleOpenChange = React.useCallback(
    (nextOpen: boolean) => {
      if (!isControlled) {
        setInternalOpen(nextOpen)
      }
      onOpenChange?.(nextOpen)
    },
    [isControlled, onOpenChange]
  )

  React.useEffect(() => {
    if (!open) {
      setValue("")
    }
  }, [open])

  const requiresText = Boolean(confirmText)
  const isConfirmEnabled = !requiresText || value === confirmText

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      {trigger ? <AlertDialogTrigger asChild>{trigger}</AlertDialogTrigger> : null}
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>

        {requiresText ? (
          <div className="flex flex-col gap-2">
            <Label htmlFor="delete-confirm-input">{confirmTextLabel}</Label>
            <Input
              id="delete-confirm-input"
              value={value}
              onChange={(event) => setValue(event.target.value)}
              placeholder={confirmTextPlaceholder ?? confirmText}
              autoComplete="off"
              autoCapitalize="off"
              autoCorrect="off"
              spellCheck={false}
            />
          </div>
        ) : null}

        <AlertDialogFooter>
          <AlertDialogCancel>{cancelLabel}</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm} disabled={!isConfirmEnabled}>
            {confirmLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

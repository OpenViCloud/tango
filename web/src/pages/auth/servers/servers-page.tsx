import { useState } from "react"
import { useForm, type SubmitHandler } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import {
  CopyIcon,
  ServerIcon,
  WifiIcon,
  Trash2Icon,
  PlusIcon,
  CheckCircleIcon,
  XCircleIcon,
  ClockIcon,
  LoaderIcon,
} from "lucide-react"

import type { ServerModel } from "@/@types/models/server"
import {
  createServerSchema,
  type CreateServerModel,
} from "@/@types/models/server"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import {
  useGetSshPublicKey,
  useGetServerList,
  useCreateServer,
  useDeleteServer,
  usePingServer,
} from "@/hooks/api/use-server"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"

export function ServersPage() {
  const { data: sshKey } = useGetSshPublicKey()
  const { data: servers, isLoading } = useGetServerList()

  const copyKey = () => {
    if (sshKey?.public_key) {
      const cmd = `echo "${sshKey.public_key.trim()}" >> ~/.ssh/authorized_keys`
      void navigator.clipboard.writeText(cmd)
      toast.success("Command copied. Paste it on your VPS.")
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<ServerIcon className="size-6" />}
        title="Servers"
        description="Manage VPS nodes for K8s cluster provisioning."
        headerRight={<AddServerDialog />}
      />

      {/* SSH Key card */}
      <SectionCard
        icon={<WifiIcon className="size-5" />}
        title="Tango SSH Public Key"
        description="Add this key to each VPS before testing the connection."
      >
        <div className="flex flex-col gap-3">
          <ol className="flex flex-col gap-2 text-sm text-muted-foreground list-none">
            <li><span className="font-medium text-foreground">1.</span> Copy lệnh bên dưới</li>
            <li><span className="font-medium text-foreground">2.</span> SSH vào VPS bằng credentials của bạn: <code className="rounded bg-muted px-1 py-0.5 text-xs">ssh root@&lt;server-ip&gt;</code></li>
            <li><span className="font-medium text-foreground">3.</span> Paste và chạy lệnh trên VPS</li>
            <li><span className="font-medium text-foreground">4.</span> Quay lại đây → Add Server → Test SSH</li>
          </ol>
          {sshKey ? (
            <div className="flex items-start gap-2">
              <pre className="flex-1 overflow-x-auto rounded-lg border bg-muted/40 p-3 text-xs text-muted-foreground whitespace-pre-wrap break-all">
                {`echo "${sshKey.public_key.trim()}" >> ~/.ssh/authorized_keys`}
              </pre>
              <Button
                size="icon"
                variant="outline"
                onClick={copyKey}
                title="Copy command"
              >
                <CopyIcon className="size-4" />
              </Button>
            </div>
          ) : (
            <Skeleton className="h-12 w-full" />
          )}
        </div>
      </SectionCard>

      {/* Servers table */}
      <SectionCard
        icon={<ServerIcon className="size-5" />}
        title="Servers"
        description="All registered VPS nodes."
        headerRight={<AddServerDialog />}
      >
        {isLoading ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : !servers?.length ? (
          <p className="text-sm text-muted-foreground">
            No servers yet. Add one to get started.
          </p>
        ) : (
          <ServerTable servers={servers} />
        )}
      </SectionCard>
    </div>
  )
}

function ServerTable({ servers }: { servers: ServerModel[] }) {
  const {
    mutate: ping,
    isPending: pinging,
    variables: pingingId,
  } = usePingServer()
  const { mutate: remove } = useDeleteServer()

  return (
    <div className="overflow-x-auto rounded-xl border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs tracking-wide text-muted-foreground uppercase">
            <th className="px-4 py-3">Name</th>
            <th className="px-4 py-3">Public IP</th>
            <th className="px-4 py-3">Private IP</th>
            <th className="px-4 py-3">SSH</th>
            <th className="px-4 py-3">Status</th>
            <th className="px-4 py-3">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {servers.map((server) => (
            <tr key={server.id} className="hover:bg-muted/20">
              <td className="px-4 py-3 font-medium">{server.name}</td>
              <td className="px-4 py-3 font-mono text-xs">
                {server.public_ip}
              </td>
              <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                {server.private_ip || "—"}
              </td>
              <td className="px-4 py-3 text-xs text-muted-foreground">
                {server.ssh_user}:{server.ssh_port}
              </td>
              <td className="px-4 py-3">
                <ServerStatusBadge status={server.status} />
              </td>
              <td className="px-4 py-3">
                <div className="flex items-center gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => ping(server.id)}
                    disabled={pinging && pingingId === server.id}
                  >
                    {pinging && pingingId === server.id ? (
                      <LoaderIcon className="size-3 animate-spin" />
                    ) : (
                      <WifiIcon className="size-3" />
                    )}
                    <span className="ml-1">Test SSH</span>
                  </Button>
                  <Button
                    size="icon"
                    variant="ghost"
                    className="text-destructive hover:text-destructive"
                    onClick={() => remove(server.id)}
                    title="Remove server"
                  >
                    <Trash2Icon className="size-4" />
                  </Button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function ServerStatusBadge({ status }: { status: string }) {
  if (status === "connected") {
    return (
      <Badge variant="default" className="gap-1">
        <CheckCircleIcon className="size-3" /> Connected
      </Badge>
    )
  }
  if (status === "error") {
    return (
      <Badge variant="destructive" className="gap-1">
        <XCircleIcon className="size-3" /> Error
      </Badge>
    )
  }
  return (
    <Badge variant="secondary" className="gap-1">
      <ClockIcon className="size-3" /> Pending
    </Badge>
  )
}

function AddServerDialog() {
  const [open, setOpen] = useState(false)
  const [showPrivateIP, setShowPrivateIP] = useState(false)
  const { mutate: create, isPending } = useCreateServer()

  const form = useForm<CreateServerModel>({
    resolver: zodResolver(createServerSchema),
    defaultValues: { ssh_user: "root", ssh_port: 22 },
  })

  const onSubmit: SubmitHandler<CreateServerModel> = (data) => {
    create(data, {
      onSuccess: () => {
        setOpen(false)
        form.reset()
      },
    })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm">
          <PlusIcon className="size-4" />
          Add Server
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Add Server</DialogTitle>
        </DialogHeader>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="flex flex-col gap-4"
        >
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              placeholder="server-01"
              {...form.register("name")}
            />
            {form.formState.errors.name && (
              <p className="text-xs text-destructive">
                {form.formState.errors.name.message}
              </p>
            )}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="public_ip">Public IP</Label>
            <Input
              id="public_ip"
              placeholder="103.x.x.x"
              {...form.register("public_ip")}
            />
            {form.formState.errors.public_ip && (
              <p className="text-xs text-destructive">
                {form.formState.errors.public_ip.message}
              </p>
            )}
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="has_private_ip"
              checked={showPrivateIP}
              onChange={(e) => setShowPrivateIP(e.target.checked)}
              className="h-4 w-4"
            />
            <Label
              htmlFor="has_private_ip"
              className="cursor-pointer text-sm font-normal"
            >
              VPS has a separate private IP (NAT)
            </Label>
          </div>

          {showPrivateIP && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="private_ip">Private IP</Label>
              <Input
                id="private_ip"
                placeholder="10.x.x.x"
                {...form.register("private_ip")}
              />
            </div>
          )}

          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="ssh_user">SSH User</Label>
              <Input
                id="ssh_user"
                placeholder="root"
                {...form.register("ssh_user")}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="ssh_port">SSH Port</Label>
              <Input
                id="ssh_port"
                type="number"
                placeholder="22"
                {...form.register("ssh_port", { valueAsNumber: true })}
              />
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending ? "Adding..." : "Add Server"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}

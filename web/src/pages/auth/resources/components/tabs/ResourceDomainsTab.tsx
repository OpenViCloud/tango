import { useState } from "react"
import { CheckCircle, Clock, Globe, Plus, RefreshCw, Trash2 } from "lucide-react"
import { toast } from "sonner"

import type { ResourceModel } from "@/@types/models"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  useListResourceDomains,
  useAddResourceDomain,
  useRemoveResourceDomain,
  useVerifyResourceDomain,
} from "@/hooks/api/use-project"

type Props = {
  resource: ResourceModel
}

export function ResourceDomainsTab({ resource }: Props) {
  const [newHost, setNewHost] = useState("")
  const { data: domains = [], isLoading } = useListResourceDomains(resource.id)
  const addMutation = useAddResourceDomain(resource.id)
  const removeMutation = useRemoveResourceDomain(resource.id)
  const verifyMutation = useVerifyResourceDomain(resource.id)

  const handleAdd = () => {
    const host = newHost.trim()
    if (!host) return
    addMutation.mutate(host, {
      onSuccess: () => {
        setNewHost("")
        toast.success(`Domain ${host} added`)
        if (resource.status === "running") {
          toast.info("Restart resource to apply new domain routing")
        }
      },
      onError: () => toast.error("Failed to add domain"),
    })
  }

  const handleVerify = (domainId: string, host: string) => {
    verifyMutation.mutate(domainId, {
      onSuccess: (res) => {
        if (res.verified) {
          toast.success(`${host} verified`)
        } else {
          toast.error(`DNS not pointing to server yet. Resolved: ${res.resolved_ips?.join(", ") || "none"}`)
        }
      },
    })
  }

  const handleRemove = (domainId: string, host: string) => {
    removeMutation.mutate(domainId, {
      onSuccess: () => toast.success(`${host} removed`),
    })
  }

  return (
    <main className="min-w-0 flex-1 p-4 sm:p-6 lg:p-8">
      <div className="flex flex-col gap-4">

        {/* Add custom domain */}
        <div className="rounded-2xl border bg-card p-5">
          <h2 className="mb-1 text-base font-semibold">Add Custom Domain</h2>
          <p className="mb-4 text-sm text-muted-foreground">
            Point your domain's DNS A record to the server IP, then add it here.
          </p>
          <div className="flex gap-2">
            <Input
              placeholder="api.example.com"
              value={newHost}
              onChange={(e) => setNewHost(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleAdd()}
              disabled={addMutation.isPending}
              className="max-w-sm"
            />
            <Button
              onClick={handleAdd}
              disabled={!newHost.trim() || addMutation.isPending}
              size="sm"
            >
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              Add
            </Button>
          </div>
        </div>

        {/* Domain list */}
        <div className="rounded-2xl border bg-card">
          <div className="border-b px-5 py-3">
            <h2 className="text-base font-semibold">Domains</h2>
          </div>

          {isLoading ? (
            <div className="px-5 py-8 text-sm text-muted-foreground">Loading…</div>
          ) : domains.length === 0 ? (
            <div className="px-5 py-8 text-sm text-muted-foreground">No domains configured.</div>
          ) : (
            <ul className="divide-y">
              {domains.map((d) => (
                <li key={d.id} className="flex items-center gap-3 px-5 py-3">
                  <Globe className="h-4 w-4 shrink-0 text-muted-foreground" />

                  <span className="min-w-0 flex-1 truncate font-mono text-sm">{d.host}</span>

                  <div className="flex shrink-0 items-center gap-2">
                    {d.type === "auto" ? (
                      <Badge variant="secondary" className="text-xs">auto</Badge>
                    ) : d.verified ? (
                      <span className="flex items-center gap-1 text-xs text-green-600">
                        <CheckCircle className="h-3.5 w-3.5" />
                        verified
                      </span>
                    ) : (
                      <span className="flex items-center gap-1 text-xs text-yellow-600">
                        <Clock className="h-3.5 w-3.5" />
                        unverified
                      </span>
                    )}

                    {d.type === "custom" && !d.verified && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7"
                        disabled={verifyMutation.isPending}
                        onClick={() => handleVerify(d.id, d.host)}
                        title="Check DNS"
                      >
                        <RefreshCw className="h-3.5 w-3.5" />
                      </Button>
                    )}

                    {d.type === "custom" && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7 text-destructive hover:text-destructive"
                        disabled={removeMutation.isPending}
                        onClick={() => handleRemove(d.id, d.host)}
                        title="Remove"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>

        {resource.status === "running" && domains.length > 0 && (
          <p className="text-xs text-muted-foreground">
            Restart the resource after adding or removing domains to apply routing changes.
          </p>
        )}
      </div>
    </main>
  )
}

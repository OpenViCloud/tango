import { useEffect, useRef, useState } from "react"
import { RefreshCw, TerminalSquare } from "lucide-react"
import { FitAddon } from "@xterm/addon-fit"
import { Terminal } from "@xterm/xterm"
import "@xterm/xterm/css/xterm.css"

import type { ResourceModel } from "@/@types/models"
import { Button } from "@/components/ui/button"

type ResourceTerminalTabProps = {
  resource: ResourceModel
}

type TerminalMessage =
  | { t: "input"; d: string }
  | { t: "resize"; cols: number; rows: number }

export function ResourceTerminalTab({ resource }: ResourceTerminalTabProps) {
  const canConnect = resource.status === "running" && Boolean(resource.container_id)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const resizeObserverRef = useRef<ResizeObserver | null>(null)
  const [connected, setConnected] = useState(false)
  const [closed, setClosed] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [reconnectKey, setReconnectKey] = useState(0)

  useEffect(() => {
    if (!canConnect) return
    if (!containerRef.current) return

    const term = new Terminal({
      convertEol: true,
      cursorBlink: true,
      fontFamily: '"JetBrains Mono", "Fira Code", monospace',
      fontSize: 13,
      theme: {
        background: "#0f0d1a",
        foreground: "#efeef8",
        cursor: "#f8fafc",
        selectionBackground: "rgba(148, 163, 184, 0.28)",
      },
    })
    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(containerRef.current)
    fitAddon.fit()
    term.focus()

    termRef.current = term
    fitRef.current = fitAddon

    const proto = window.location.protocol === "https:" ? "wss" : "ws"
    const ws = new WebSocket(
      `${proto}://${window.location.host}/api/ws/resources/${resource.id}/terminal`
    )
    ws.binaryType = "arraybuffer"
    wsRef.current = ws

    const sendMessage = (message: TerminalMessage) => {
      if (ws.readyState !== WebSocket.OPEN) return
      ws.send(JSON.stringify(message))
    }

    const syncSize = () => {
      fitAddon.fit()
      sendMessage({
        t: "resize",
        cols: term.cols,
        rows: term.rows,
      })
    }

    const inputDisposable = term.onData((data) => {
      sendMessage({ t: "input", d: data })
    })

    resizeObserverRef.current = new ResizeObserver(() => {
      syncSize()
    })
    resizeObserverRef.current.observe(containerRef.current)

    ws.onopen = () => {
      setConnected(true)
      setClosed(false)
      setError(null)
      syncSize()
      term.writeln("\x1b[32mConnected to container terminal.\x1b[0m")
    }

    ws.onmessage = async (event) => {
      if (typeof event.data === "string") {
        try {
          const message = JSON.parse(event.data) as { t?: string; d?: string }
          if (message.t === "error") {
            setError(message.d ?? "Terminal connection failed.")
            term.writeln(`\r\n\x1b[31m${message.d ?? "Terminal connection failed."}\x1b[0m`)
          }
        } catch {
          term.write(event.data)
        }
        return
      }

      if (event.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(event.data))
        return
      }

      if (event.data instanceof Blob) {
        const buffer = await event.data.arrayBuffer()
        term.write(new Uint8Array(buffer))
      }
    }

    ws.onclose = () => {
      setConnected(false)
      setClosed(true)
    }

    ws.onerror = () => {
      setConnected(false)
      setClosed(true)
      setError("Terminal connection failed.")
    }

    return () => {
      resizeObserverRef.current?.disconnect()
      resizeObserverRef.current = null
      inputDisposable.dispose()
      ws.close()
      wsRef.current = null
      fitRef.current = null
      termRef.current = null
      term.dispose()
    }
  }, [canConnect, reconnectKey, resource.id])

  if (!canConnect) {
    return (
      <main className="min-w-0 flex-1 p-4 sm:p-6 lg:p-8">
        <div className="rounded-2xl border bg-card p-6">
          <h2 className="text-xl font-semibold">Terminal</h2>
          <p className="mt-2 text-sm text-muted-foreground">
            Start <span className="font-medium text-foreground">{resource.name}</span> before
            opening an interactive shell.
          </p>
        </div>
      </main>
    )
  }

  const statusText = error
    ? error
    : connected
      ? "Connected"
      : closed
        ? "Disconnected"
        : "Connecting..."

  return (
    <main className="min-w-0 flex-1 p-4 sm:p-6 lg:p-8">
      <div className="rounded-2xl border bg-card p-6">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div className="flex items-center gap-2">
              <TerminalSquare className="h-5 w-5 text-muted-foreground" />
              <h2 className="text-xl font-semibold">Terminal</h2>
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
              Interactive shell attached to{" "}
              <span className="font-medium text-foreground">{resource.name}</span>.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-sm text-muted-foreground">{statusText}</span>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => {
                setError(null)
                setClosed(false)
                setConnected(false)
                setReconnectKey((value) => value + 1)
              }}
            >
              <RefreshCw className="mr-2 h-4 w-4" />
              Reconnect
            </Button>
          </div>
        </div>

        <div className="mt-4 overflow-hidden rounded-xl border bg-[#0f0d1a]">
          <div className="border-b border-white/10 px-4 py-2 font-mono text-xs text-slate-300">
            {resource.container_id}
          </div>
          <div ref={containerRef} className="h-[520px] w-full px-2 py-3" />
        </div>
      </div>
    </main>
  )
}

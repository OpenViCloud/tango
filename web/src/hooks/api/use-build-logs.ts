import { useEffect, useRef, useState } from "react"
import { applyChunk } from "@/lib/terminal"

type WsMsg =
  | { t: "log"; d: string }
  | { t: "done"; status: string }
  | { t: "error"; d: string }

export type BuildLogState = {
  logs: string
  status: string | null  // final status when done
  connected: boolean
}

export function useBuildLogs(jobId: string | null): BuildLogState {
  const [logs, setLogs] = useState("")
  const [status, setStatus] = useState<string | null>(null)
  const [connected, setConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const doneRef = useRef(false)

  useEffect(() => {
    if (!jobId) return

    const proto = window.location.protocol === "https:" ? "wss" : "ws"
    const url = `${proto}://${window.location.host}/api/ws/builds/${jobId}/logs`
    const ws = new WebSocket(url)
    wsRef.current = ws
    doneRef.current = false

    setLogs("")
    setStatus(null)

    ws.onopen = () => setConnected(true)

    ws.onmessage = (e) => {
      const msg: WsMsg = JSON.parse(e.data)
      if (msg.t === "log") {
        setLogs((prev) => applyChunk(prev, msg.d))
      } else if (msg.t === "done") {
        doneRef.current = true
        setStatus(msg.status)
        setConnected(false)
      }
    }

    ws.onclose = () => setConnected(false)
    ws.onerror = () => setConnected(false) // build logs có no error state, chỉ disconnect

    return () => {
      ws.close()
    }
  }, [jobId])

  return { logs, status, connected }
}

import { useCallback, useEffect, useRef, useState } from "react"

type WsMsg =
  | { t: "log"; d: string }
  | { t: "layer"; id: string; status: string; progress: string }
  | { t: "done"; status: string }
  | { t: "error"; d: string }

export type LayerState = {
  status: string
  progress: string
}

export type PullImageLogState = {
  headerLogs: string
  layers: Map<string, LayerState>
  layerOrder: string[]
  footerLogs: string
  done: boolean
  error: string | null
  connected: boolean
  pull: (reference: string) => void
  reset: () => void
}

export function usePullImageLogs(): PullImageLogState {
  const [headerLogs, setHeaderLogs] = useState("")
  const [layers, setLayers] = useState<Map<string, LayerState>>(new Map())
  const [layerOrder, setLayerOrder] = useState<string[]>([])
  const [footerLogs, setFooterLogs] = useState("")
  const [done, setDone] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [connected, setConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const hasLayersRef = useRef(false)
  const doneRef = useRef(false)

  const reset = useCallback(() => {
    wsRef.current?.close()
    setHeaderLogs("")
    setLayers(new Map())
    setLayerOrder([])
    setFooterLogs("")
    setDone(false)
    setError(null)
    setConnected(false)
    hasLayersRef.current = false
  }, [])

  const pull = useCallback((reference: string) => {
    wsRef.current?.close()
    setHeaderLogs("")
    setLayers(new Map())
    setLayerOrder([])
    setFooterLogs("")
    setDone(false)
    setError(null)
    hasLayersRef.current = false
    doneRef.current = false

    const proto = window.location.protocol === "https:" ? "wss" : "ws"
    const url = `${proto}://${window.location.host}/api/ws/docker/images/pull?reference=${encodeURIComponent(reference)}`
    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => setConnected(true)

    ws.onmessage = (e) => {
      const msg: WsMsg = JSON.parse(e.data)
      if (msg.t === "log") {
        if (hasLayersRef.current) {
          setFooterLogs((prev) => prev + msg.d)
        } else {
          setHeaderLogs((prev) => prev + msg.d)
        }
      } else if (msg.t === "layer") {
        hasLayersRef.current = true
        setLayers((prev) => {
          const next = new Map(prev)
          next.set(msg.id, { status: msg.status, progress: msg.progress })
          return next
        })
        setLayerOrder((prev) => (prev.includes(msg.id) ? prev : [...prev, msg.id]))
      } else if (msg.t === "done") {
        doneRef.current = true
        setDone(true)
        setConnected(false)
      } else if (msg.t === "error") {
        setError(msg.d)
        setConnected(false)
      }
    }

    ws.onclose = () => setConnected(false)
    ws.onerror = () => {
      if (!doneRef.current) {
        setError("Connection error")
      }
      setConnected(false)
    }
  }, [])

  useEffect(() => () => wsRef.current?.close(), [])

  return { headerLogs, layers, layerOrder, footerLogs, done, error, connected, pull, reset }
}

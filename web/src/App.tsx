import React, { useState, useEffect } from 'react'
import { TelemetrySnapshot, FrameEvent } from './types'
import { TokenSavingsHero } from './components/TokenSavingsHero'
import { LiveSieveCanvas } from './components/LiveSieveCanvas'
import { SsimDeltaChart } from './components/SsimDeltaChart'
import { BurstTimeline } from './components/BurstTimeline'
import { ThresholdControl } from './components/ThresholdControl'
import { SvgLogo } from './components/SvgLogo'
import { Wifi, WifiOff } from 'lucide-react'

export const App: React.FC = () => {
  const [telemetry, setTelemetry] = useState<TelemetrySnapshot | null>(null)
  const [latestEvent, setLatestEvent] = useState<FrameEvent | null>(null)
  const [ssimHistory, setSsimHistory] = useState<number[]>(() => Array(40).fill(0.995))
  const [eventsTimeline, setEventsTimeline] = useState<FrameEvent[]>([])
  const [threshold, setThreshold] = useState<number>(0.985)
  const [connected, setConnected] = useState<boolean>(false)

  // SSE event subscriber
  useEffect(() => {
    let evtSource: EventSource | null = null

    try {
      evtSource = new EventSource('/api/events/sse')

      evtSource.onopen = () => {
        setConnected(true)
      }

      evtSource.addEventListener('frame_event', (e) => {
        try {
          const parsed: FrameEvent = JSON.parse(e.data)
          setLatestEvent(parsed)

          setSsimHistory((prev) => {
            const next = [...prev, parsed.ssim]
            if (next.length > 60) next.shift()
            return next
          })

          if (parsed.roi || parsed.trigger_frame?.is_key) {
            setEventsTimeline((prev) => [parsed, ...prev.slice(0, 19)])
          }
        } catch {
          // Ignore JSON errors
        }
      })

      evtSource.onerror = () => {
        setConnected(false)
      }
    } catch {
      setConnected(false)
    }

    return () => {
      if (evtSource) {
        evtSource.close()
      }
    }
  }, [])

  // Telemetry poller
  useEffect(() => {
    const fetchTelemetry = async () => {
      try {
        const res = await fetch('/api/telemetry')
        if (res.ok) {
          const data: TelemetrySnapshot = await res.json()
          setTelemetry(data)
        }
      } catch {
        // Polling retry
      }
    }

    fetchTelemetry()
    const timer = setInterval(fetchTelemetry, 1000)
    return () => clearInterval(timer)
  }, [])

  return (
    <div className="min-h-screen bg-surface-950 text-slate-100 flex flex-col font-sans">
      {/* Header */}
      <header className="border-b border-slate-800 bg-surface-900/90 backdrop-blur sticky top-0 z-50 px-6 py-3.5 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <SvgLogo />
          <div>
            <div className="flex items-center gap-2">
              <h1 className="font-bold text-base tracking-tight text-white">framesieve</h1>
              <span className="px-1.5 py-0.5 rounded text-[10px] font-mono font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                v1.0.0-edge
              </span>
            </div>
            <p className="text-[11px] text-slate-400">
              Perceptual Video Sieve & Dynamic ROI Compressor for Vision-Language Models
            </p>
          </div>
        </div>

        <div className="flex items-center gap-4 text-xs font-mono">
          <div className="flex items-center gap-1.5">
            {connected ? (
              <>
                <Wifi className="w-3.5 h-3.5 text-emerald-400" />
                <span className="text-emerald-400 font-medium">LIVE STREAM CONNECTED</span>
              </>
            ) : (
              <>
                <WifiOff className="w-3.5 h-3.5 text-amber-400" />
                <span className="text-amber-400">STANDBY POLLING</span>
              </>
            )}
          </div>
          <div className="hidden md:block h-4 w-px bg-slate-800"></div>
          <span className="hidden md:inline text-slate-400">
            Port :8686 | Reverse Proxy :8088
          </span>
        </div>
      </header>

      {/* Main Container */}
      <main className="flex-1 p-6 space-y-6 max-w-7xl mx-auto w-full">
        {/* Top Hero KPI Metrics */}
        <TokenSavingsHero snapshot={telemetry} />

        {/* Video Monitor + Side Controls */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2">
            <LiveSieveCanvas latestEvent={latestEvent} />
          </div>
          <div className="space-y-6">
            <ThresholdControl
              currentSSIM={threshold}
              onUpdateThreshold={(val) => setThreshold(val)}
            />
            <SsimDeltaChart history={ssimHistory} threshold={threshold} />
          </div>
        </div>

        {/* Burst Timeline */}
        <BurstTimeline events={eventsTimeline} />
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-800/80 bg-surface-900/50 py-4 px-6 text-center text-xs text-slate-500">
        framesieve edge vision engine — Zero-allocation CPU SIMD image processing in Go
      </footer>
    </div>
  )
}
export default App

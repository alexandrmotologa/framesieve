import React, { useEffect, useRef } from 'react'
import { Activity } from 'lucide-react'

interface Props {
  history: number[]
  threshold: number
}

export const SsimDeltaChart: React.FC<Props> = ({ history, threshold }) => {
  const canvasRef = useRef<HTMLCanvasElement | null>(null)

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const w = canvas.width
    const h = canvas.height

    ctx.clearRect(0, 0, w, h)

    // Background
    ctx.fillStyle = '#0f1117'
    ctx.fillRect(0, 0, w, h)

    // Grid lines
    ctx.strokeStyle = '#1a202c'
    ctx.lineWidth = 1
    for (let y = 0; y < h; y += 30) {
      ctx.beginPath()
      ctx.moveTo(0, y)
      ctx.lineTo(w, y)
      ctx.stroke()
    }

    // Min/Max bounds for SSIM chart: 0.80 to 1.0
    const minVal = 0.80
    const maxVal = 1.0

    const getY = (val: number) => {
      const clamped = Math.max(minVal, Math.min(maxVal, val))
      return h - ((clamped - minVal) / (maxVal - minVal)) * (h - 20) - 10
    }

    // Draw Threshold line
    const threshY = getY(threshold)
    ctx.strokeStyle = 'rgba(239, 68, 68, 0.6)'
    ctx.lineWidth = 1.5
    ctx.setLineDash([4, 4])
    ctx.beginPath()
    ctx.moveTo(0, threshY)
    ctx.lineTo(w, threshY)
    ctx.stroke()
    ctx.setLineDash([])

    // Threshold text
    ctx.fillStyle = '#ef4444'
    ctx.font = '10px monospace'
    ctx.fillText(`Gate Threshold (${threshold})`, w - 140, threshY - 6)

    if (history.length < 2) return

    // Draw SSIM curve
    const stepX = w / (history.length - 1)

    // Gradient fill under curve
    const grad = ctx.createLinearGradient(0, 0, 0, h)
    grad.addColorStop(0, 'rgba(16, 185, 129, 0.35)')
    grad.addColorStop(1, 'rgba(16, 185, 129, 0.0)')

    ctx.beginPath()
    ctx.moveTo(0, h)
    for (let i = 0; i < history.length; i++) {
      const x = i * stepX
      const y = getY(history[i])
      ctx.lineTo(x, y)
    }
    ctx.lineTo(w, h)
    ctx.closePath()
    ctx.fillStyle = grad
    ctx.fill()

    // Stroke line
    ctx.beginPath()
    for (let i = 0; i < history.length; i++) {
      const x = i * stepX
      const y = getY(history[i])
      if (i === 0) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
    }
    ctx.strokeStyle = '#10b981'
    ctx.lineWidth = 2
    ctx.stroke()

    // Draw spike dots for values below threshold
    for (let i = 0; i < history.length; i++) {
      if (history[i] < threshold) {
        const x = i * stepX
        const y = getY(history[i])
        ctx.beginPath()
        ctx.arc(x, y, 4, 0, 2 * Math.PI)
        ctx.fillStyle = '#f59e0b'
        ctx.fill()
        ctx.strokeStyle = '#78350f'
        ctx.lineWidth = 1.5
        ctx.stroke()
      }
    }
  }, [history, threshold])

  return (
    <div className="bg-surface-850/90 border border-slate-800 rounded-xl p-4 shadow-xl">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <Activity className="w-4 h-4 text-emerald-400" />
          <h3 className="text-sm font-semibold text-slate-200">Temporal SSIM Delta Curve</h3>
        </div>
        <span className="text-xs text-slate-400 font-mono">Window: Last 60 Frames</span>
      </div>
      <div className="w-full h-44 bg-surface-950 rounded-lg overflow-hidden border border-slate-800">
        <canvas ref={canvasRef} width={640} height={176} className="w-full h-full" />
      </div>
    </div>
  )
}

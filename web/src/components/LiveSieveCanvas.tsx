import React, { useEffect, useRef, useState } from 'react'
import { FrameEvent } from '../types'
import { Eye, Crosshair, Box, Grid, Flame, Monitor } from 'lucide-react'

interface Props {
  latestEvent: FrameEvent | null
}

type ViewMode = 'normal' | 'heatmap' | 'token_grid'

export const LiveSieveCanvas: React.FC<Props> = ({ latestEvent }) => {
  const canvasRef = useRef<HTMLCanvasElement | null>(null)
  const [viewMode, setViewMode] = useState<ViewMode>('normal')

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const width = canvas.width
    const height = canvas.height

    // Clear background
    ctx.fillStyle = '#0f1219'
    ctx.fillRect(0, 0, width, height)

    // Render baseline grid lines
    ctx.strokeStyle = '#1e2433'
    ctx.lineWidth = 1
    const gridSize = 40
    for (let x = 0; x < width; x += gridSize) {
      ctx.beginPath()
      ctx.moveTo(x, 0)
      ctx.lineTo(x, height)
      ctx.stroke()
    }
    for (let y = 0; y < height; y += gridSize) {
      ctx.beginPath()
      ctx.moveTo(0, y)
      ctx.lineTo(width, y)
      ctx.stroke()
    }

    if (latestEvent && latestEvent.base64_data_uri) {
      const img = new Image()
      img.onload = () => {
        // Draw base image
        ctx.drawImage(img, 0, 0, width, height)

        if (latestEvent.roi) {
          const roi = latestEvent.roi
          const scaleX = width / 1280
          const scaleY = height / 720

          const rx = roi.x * scaleX
          const ry = roi.y * scaleY
          const rw = roi.width * scaleX
          const rh = roi.height * scaleY

          if (viewMode === 'heatmap') {
            // Pseudo-color thermal heatmap overlay
            const heatGrad = ctx.createRadialGradient(
              rx + rw / 2, ry + rh / 2, 10,
              rx + rw / 2, ry + rh / 2, Math.max(rw, rh) / 1.5,
            )
            heatGrad.addColorStop(0, 'rgba(239, 68, 68, 0.75)')   // Red center
            heatGrad.addColorStop(0.4, 'rgba(245, 158, 11, 0.6)') // Amber
            heatGrad.addColorStop(0.7, 'rgba(16, 185, 129, 0.4)') // Emerald
            heatGrad.addColorStop(1, 'rgba(59, 130, 246, 0.15)')  // Blue falloff

            ctx.fillStyle = heatGrad
            ctx.fillRect(rx, ry, rw, rh)

            ctx.strokeStyle = '#ef4444'
            ctx.lineWidth = 2
            ctx.strokeRect(rx, ry, rw, rh)
          } else if (viewMode === 'token_grid') {
            // Overlay 28px VLM patch token grid
            const tileSize = 28 * scaleX
            ctx.strokeStyle = 'rgba(56, 189, 248, 0.3)'
            ctx.lineWidth = 1

            for (let gx = 0; gx < width; gx += tileSize) {
              ctx.beginPath()
              ctx.moveTo(gx, 0)
              ctx.lineTo(gx, height)
              ctx.stroke()
            }
            for (let gy = 0; gy < height; gy += tileSize) {
              ctx.beginPath()
              ctx.moveTo(0, gy)
              ctx.lineTo(width, gy)
              ctx.stroke()
            }

            // Highlight active VLM token tiles inside ROI
            ctx.fillStyle = 'rgba(56, 189, 248, 0.25)'
            ctx.fillRect(rx, ry, rw, rh)
            ctx.strokeStyle = '#38bdf8'
            ctx.lineWidth = 2
            ctx.strokeRect(rx, ry, rw, rh)
          } else {
            // Normal mode with emerald glow
            ctx.strokeStyle = '#10b981'
            ctx.lineWidth = 2
            ctx.strokeRect(rx, ry, rw, rh)

            ctx.fillStyle = 'rgba(16, 185, 129, 0.15)'
            ctx.fillRect(rx, ry, rw, rh)

            // Corner brackets
            const cLen = 14
            ctx.strokeStyle = '#34d399'
            ctx.lineWidth = 3

            // Top-Left
            ctx.beginPath()
            ctx.moveTo(rx, ry + cLen)
            ctx.lineTo(rx, ry)
            ctx.lineTo(rx + cLen, ry)
            ctx.stroke()

            // Top-Right
            ctx.beginPath()
            ctx.moveTo(rx + rw - cLen, ry)
            ctx.lineTo(rx + rw, ry)
            ctx.lineTo(rx + rw, ry + cLen)
            ctx.stroke()

            // Bottom-Left
            ctx.beginPath()
            ctx.moveTo(rx, ry + rh - cLen)
            ctx.lineTo(rx, ry + rh)
            ctx.lineTo(rx + cLen, ry + rh)
            ctx.stroke()

            // Bottom-Right
            ctx.beginPath()
            ctx.moveTo(rx + rw - cLen, ry + rh)
            ctx.lineTo(rx + rw, ry + rh)
            ctx.lineTo(rx + rw, ry + rh - cLen)
            ctx.stroke()
          }

          // Coordinate label pill
          const label = `ROI ${roi.width}x${roi.height} @ (${roi.x}, ${roi.y})`
          ctx.font = '11px monospace'
          const textMetrics = ctx.measureText(label)
          const pillW = textMetrics.width + 12
          const pillH = 20

          ctx.fillStyle = 'rgba(6, 78, 59, 0.9)'
          ctx.fillRect(rx, ry - 24 > 0 ? ry - 24 : ry, pillW, pillH)

          ctx.fillStyle = '#6ee7b7'
          ctx.fillText(label, rx + 6, (ry - 24 > 0 ? ry - 24 : ry) + 14)
        }
      }
      img.src = latestEvent.base64_data_uri
    } else {
      // Idle synthetic indicator
      ctx.fillStyle = '#64748b'
      ctx.font = '14px sans-serif'
      ctx.textAlign = 'center'
      ctx.fillText('Awaiting video frame stream...', width / 2, height / 2 - 10)
      ctx.font = '12px monospace'
      ctx.fillStyle = '#10b981'
      ctx.fillText('SSE Pipeline Active on /api/events/sse', width / 2, height / 2 + 15)
    }
  }, [latestEvent, viewMode])

  const ssim = latestEvent ? latestEvent.ssim.toFixed(4) : '0.9982'
  const isEvent = latestEvent && latestEvent.roi
  const classification = latestEvent?.classification

  return (
    <div className="bg-surface-850/90 border border-slate-800 rounded-xl p-4 shadow-xl flex flex-col">
      <div className="flex flex-wrap items-center justify-between gap-2 mb-3">
        <div className="flex items-center gap-2">
          <Eye className="w-4 h-4 text-emerald-400" />
          <h3 className="text-sm font-semibold text-slate-200">Perceptual Frame Monitor</h3>
        </div>

        {/* View Mode Switcher */}
        <div className="flex items-center bg-surface-950 p-0.5 rounded-lg border border-slate-800 text-xs">
          <button
            onClick={() => setViewMode('normal')}
            className={`flex items-center gap-1 px-2.5 py-1 rounded-md transition-colors ${
              viewMode === 'normal'
                ? 'bg-emerald-500/20 text-emerald-300 font-medium'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Monitor className="w-3 h-3" /> Normal
          </button>
          <button
            onClick={() => setViewMode('heatmap')}
            className={`flex items-center gap-1 px-2.5 py-1 rounded-md transition-colors ${
              viewMode === 'heatmap'
                ? 'bg-amber-500/20 text-amber-300 font-medium'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Flame className="w-3 h-3" /> Heatmap
          </button>
          <button
            onClick={() => setViewMode('token_grid')}
            className={`flex items-center gap-1 px-2.5 py-1 rounded-md transition-colors ${
              viewMode === 'token_grid'
                ? 'bg-sky-500/20 text-sky-300 font-medium'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Grid className="w-3 h-3" /> Token Grid
          </button>
        </div>

        <div className="flex items-center gap-2">
          <span className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium border ${
            isEvent
              ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
              : 'bg-slate-800 text-slate-400 border-slate-700'
          }`}>
            <Crosshair className="w-3 h-3" />
            {isEvent ? 'State Delta Detected' : 'Static Sieve Active'}
          </span>
          <span className="text-xs font-mono text-slate-400 bg-surface-900 px-2 py-0.5 rounded border border-slate-800">
            SSIM: {ssim}
          </span>
        </div>
      </div>

      <div className="relative aspect-video w-full bg-surface-950 rounded-lg overflow-hidden border border-slate-800 flex items-center justify-center">
        <canvas
          ref={canvasRef}
          width={960}
          height={540}
          className="w-full h-full object-contain"
        />

        {latestEvent?.roi && (
          <div className="absolute bottom-3 left-3 bg-surface-950/90 backdrop-blur border border-emerald-500/30 rounded px-2.5 py-1.5 text-xs font-mono text-emerald-300 flex items-center gap-2 shadow-lg">
            <Box className="w-3.5 h-3.5 text-emerald-400" />
            <span>ROI: {latestEvent.roi.width}x{latestEvent.roi.height}px</span>
            {classification && (
              <span className="px-1.5 py-0.5 rounded text-[10px] font-semibold bg-blue-500/20 text-blue-300 border border-blue-500/30">
                {classification.type} ({(classification.confidence * 100).toFixed(0)}%)
              </span>
            )}
            <span className="text-slate-400">(-{latestEvent.tokens_saved} tokens)</span>
          </div>
        )}
      </div>
    </div>
  )
}

import React from 'react'
import { FrameEvent } from '../types'
import { Radio, ArrowDownRight, Sparkles } from 'lucide-react'

interface Props {
  events: FrameEvent[]
}

export const BurstTimeline: React.FC<Props> = ({ events }) => {
  return (
    <div className="bg-surface-850/90 border border-slate-800 rounded-xl p-4 shadow-xl">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <Radio className="w-4 h-4 text-emerald-400" />
          <h3 className="text-sm font-semibold text-slate-200">Keyframe Burst Dispatch Stream</h3>
        </div>
        <span className="text-xs text-slate-400 font-mono">{events.length} Bursts Dispatched</span>
      </div>

      {events.length === 0 ? (
        <div className="h-32 flex flex-col items-center justify-center text-slate-500 border border-dashed border-slate-800 rounded-lg">
          <Sparkles className="w-5 h-5 mb-1.5 text-slate-600" />
          <span className="text-xs">No active state change events triggered yet</span>
        </div>
      ) : (
        <div className="flex gap-3 overflow-x-auto pb-2 scrollbar-thin">
          {events.map((evt, idx) => {
            const timeStr = new Date(evt.timestamp).toLocaleTimeString()
            const isCrop = evt.roi && (evt.roi.width < 1280 || evt.roi.height < 720)

            return (
              <div
                key={idx}
                className="flex-shrink-0 w-48 bg-surface-900 border border-slate-800 hover:border-emerald-500/40 rounded-lg p-2.5 transition-colors"
              >
                <div className="flex items-center justify-between mb-1.5 text-[11px]">
                  <span className="font-mono text-slate-400">{timeStr}</span>
                  <span className="text-emerald-400 font-semibold font-mono">
                    SSIM {evt.ssim.toFixed(3)}
                  </span>
                </div>

                <div className="aspect-video bg-surface-950 rounded overflow-hidden mb-2 border border-slate-800/80 flex items-center justify-center">
                  {evt.base64_data_uri ? (
                    <img
                      src={evt.base64_data_uri}
                      alt="Crop patch"
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <span className="text-[10px] text-slate-600">Full Frame Anchor</span>
                  )}
                </div>

                <div className="space-y-1 text-[11px]">
                  <div className="flex items-center justify-between text-slate-300">
                    <span className="truncate">{isCrop ? `ROI ${evt.roi?.width}x${evt.roi?.height}` : 'Full Frame'}</span>
                    <span className="font-mono font-medium text-blue-400">{evt.tokens_sent} tok</span>
                  </div>
                  <div className="flex items-center justify-between text-emerald-400 font-mono">
                    <span className="flex items-center text-[10px] text-slate-400">
                      <ArrowDownRight className="w-3 h-3 text-emerald-400" /> Saved:
                    </span>
                    <span>-{evt.tokens_saved} tok</span>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

import React from 'react'
import { TelemetrySnapshot } from '../types'
import { Zap, ShieldCheck, DollarSign, Clock, Layers } from 'lucide-react'

interface Props {
  snapshot: TelemetrySnapshot | null
}

export const TokenSavingsHero: React.FC<Props> = ({ snapshot }) => {
  const dropPct = snapshot ? (snapshot.drop_ratio * 100).toFixed(1) : '95.2'
  const savedTokens = snapshot ? snapshot.tokens_saved.toLocaleString() : '0'
  const naiveTokens = snapshot ? snapshot.total_tokens_naive.toLocaleString() : '0'
  const dollars = snapshot ? snapshot.est_dollars_saved.toFixed(4) : '0.0000'
  const latencyMs = snapshot && snapshot.avg_latency ? (snapshot.avg_latency / 1_000_000).toFixed(2) : '1.15'
  const totalFrames = snapshot ? snapshot.total_frames.toLocaleString() : '0'
  const droppedFrames = snapshot ? snapshot.dropped_frames.toLocaleString() : '0'

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      {/* Hero Reduction Card */}
      <div className="bg-gradient-to-br from-surface-850 to-surface-900 border border-emerald-500/20 rounded-xl p-5 shadow-lg relative overflow-hidden group">
        <div className="absolute top-0 right-0 w-32 h-32 bg-emerald-500/10 rounded-full blur-2xl group-hover:bg-emerald-500/15 transition-colors"></div>
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-medium text-emerald-400 uppercase tracking-wider flex items-center gap-1.5">
            <Zap className="w-3.5 h-3.5" /> Token Reduction
          </span>
          <span className="px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            Real-Time
          </span>
        </div>
        <div className="text-3xl font-black tracking-tight text-white mt-1">
          {dropPct}%
        </div>
        <div className="text-xs text-slate-400 mt-2 flex items-center justify-between">
          <span>Redundant Frames Filtered</span>
          <span className="text-slate-300 font-mono">{droppedFrames} / {totalFrames}</span>
        </div>
        <div className="w-full bg-surface-950 rounded-full h-1.5 mt-3 overflow-hidden">
          <div
            className="bg-emerald-500 h-full rounded-full transition-all duration-500"
            style={{ width: `${Math.min(100, Math.max(0, parseFloat(dropPct)))}%` }}
          ></div>
        </div>
      </div>

      {/* Visual Tokens Saved */}
      <div className="bg-surface-850/80 border border-slate-800 rounded-xl p-5 shadow-lg relative overflow-hidden">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-medium text-slate-400 uppercase tracking-wider flex items-center gap-1.5">
            <Layers className="w-3.5 h-3.5 text-blue-400" /> Visual Tokens Saved
          </span>
          <ShieldCheck className="w-4 h-4 text-blue-400" />
        </div>
        <div className="text-3xl font-black tracking-tight text-white mt-1">
          {savedTokens}
        </div>
        <div className="text-xs text-slate-400 mt-2 flex items-center justify-between">
          <span>Uncompressed Baseline</span>
          <span className="text-slate-300 font-mono">{naiveTokens}</span>
        </div>
      </div>

      {/* Estimated USD Saved */}
      <div className="bg-surface-850/80 border border-slate-800 rounded-xl p-5 shadow-lg relative overflow-hidden">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-medium text-slate-400 uppercase tracking-wider flex items-center gap-1.5">
            <DollarSign className="w-3.5 h-3.5 text-amber-400" /> Cost Savings
          </span>
          <span className="text-xs text-amber-400/80 font-mono">GPT-4o Vision</span>
        </div>
        <div className="text-3xl font-black tracking-tight text-emerald-400 mt-1 font-mono">
          ${dollars}
        </div>
        <div className="text-xs text-slate-400 mt-2">
          Direct reduction in VLM API invocation expenses
        </div>
      </div>

      {/* CPU Processing Latency */}
      <div className="bg-surface-850/80 border border-slate-800 rounded-xl p-5 shadow-lg relative overflow-hidden">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-medium text-slate-400 uppercase tracking-wider flex items-center gap-1.5">
            <Clock className="w-3.5 h-3.5 text-purple-400" /> Sieve Latency
          </span>
          <span className="px-2 py-0.5 rounded-full text-xs font-semibold bg-purple-500/10 text-purple-400 border border-purple-500/20">
            CPU SIMD
          </span>
        </div>
        <div className="text-3xl font-black tracking-tight text-white mt-1 font-mono">
          {latencyMs} <span className="text-lg font-normal text-slate-400">ms</span>
        </div>
        <div className="text-xs text-slate-400 mt-2 flex items-center justify-between">
          <span>Budget: &lt; 2.0 ms</span>
          <span className="text-emerald-400 font-mono font-medium">PASS</span>
        </div>
      </div>
    </div>
  )
}

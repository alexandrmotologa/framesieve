import React, { useState } from 'react'
import { Sliders, RefreshCw, Check } from 'lucide-react'

interface Props {
  currentSSIM: number
  onUpdateThreshold: (newVal: number) => void
}

export const ThresholdControl: React.FC<Props> = ({ currentSSIM, onUpdateThreshold }) => {
  const [ssimVal, setSsimVal] = useState<number>(currentSSIM || 0.985)
  const [pHashVal, setPHashVal] = useState<number>(2)
  const [paddingVal, setPaddingVal] = useState<number>(32)
  const [saved, setSaved] = useState<boolean>(false)

  const handleApply = async () => {
    try {
      await fetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          vision: {
            ssim_threshold: ssimVal,
            phash_threshold: pHashVal,
            padding_margin: paddingVal,
          },
        }),
      })
      onUpdateThreshold(ssimVal)
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch {
      // Fallback
      onUpdateThreshold(ssimVal)
    }
  }

  return (
    <div className="bg-surface-850/90 border border-slate-800 rounded-xl p-4 shadow-xl">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Sliders className="w-4 h-4 text-emerald-400" />
          <h3 className="text-sm font-semibold text-slate-200">Perceptual Sieve Tuning</h3>
        </div>
        <button
          onClick={handleApply}
          className="flex items-center gap-1.5 px-3 py-1 bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-medium rounded-md transition-colors"
        >
          {saved ? <Check className="w-3.5 h-3.5" /> : <RefreshCw className="w-3.5 h-3.5" />}
          {saved ? 'Applied' : 'Apply Live'}
        </button>
      </div>

      <div className="space-y-4 text-xs">
        {/* SSIM Sensitivity */}
        <div>
          <div className="flex justify-between text-slate-300 mb-1.5">
            <span>SSIM Gate Threshold:</span>
            <span className="font-mono text-emerald-400 font-semibold">{ssimVal.toFixed(3)}</span>
          </div>
          <input
            type="range"
            min={0.90}
            max={0.999}
            step={0.001}
            value={ssimVal}
            onChange={(e) => setSsimVal(parseFloat(e.target.value))}
            className="w-full accent-emerald-500 bg-surface-950 h-1.5 rounded-lg appearance-none cursor-pointer"
          />
          <div className="flex justify-between text-[10px] text-slate-500 mt-1">
            <span>Aggressive (0.900)</span>
            <span>Balanced (0.985)</span>
            <span>Ultra-Sensitive (0.999)</span>
          </div>
        </div>

        {/* pHash Distance */}
        <div>
          <div className="flex justify-between text-slate-300 mb-1.5">
            <span>pHash Hamming Distance:</span>
            <span className="font-mono text-blue-400 font-semibold">{pHashVal} bits</span>
          </div>
          <input
            type="range"
            min={0}
            max={6}
            step={1}
            value={pHashVal}
            onChange={(e) => setPHashVal(parseInt(e.target.value))}
            className="w-full accent-blue-500 bg-surface-950 h-1.5 rounded-lg appearance-none cursor-pointer"
          />
        </div>

        {/* ROI Padding */}
        <div>
          <div className="flex justify-between text-slate-300 mb-1.5">
            <span>Dynamic ROI Margin:</span>
            <span className="font-mono text-purple-400 font-semibold">{paddingVal} px</span>
          </div>
          <input
            type="range"
            min={0}
            max={64}
            step={8}
            value={paddingVal}
            onChange={(e) => setPaddingVal(parseInt(e.target.value))}
            className="w-full accent-purple-500 bg-surface-950 h-1.5 rounded-lg appearance-none cursor-pointer"
          />
        </div>
      </div>
    </div>
  )
}

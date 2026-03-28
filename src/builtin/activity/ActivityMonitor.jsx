import { useState, useEffect } from 'react'
import { useTelemetry } from '../../core/useTelemetry'

function fmtBytes(b) {
  if (!b || b <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0, v = b
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

function Bar({ value, max, color }) {
  const pct = max > 0 ? Math.min(100, (value / max) * 100) : 0
  return (
    <div className="h-2 bg-neutral-800 rounded-full overflow-hidden">
      <div className={`h-full rounded-full transition-all duration-500 ${color}`} style={{ width: `${pct}%` }} />
    </div>
  )
}

export default function ActivityMonitor() {
  const { stats, connected } = useTelemetry()
  const [history, setHistory] = useState([])
  const [apps, setApps] = useState([])

  // Keep last 60 samples for sparkline
  useEffect(() => {
    if (stats) {
      setHistory(prev => [...prev.slice(-59), { cpu: stats.cpu, mem: stats.mem_percent, t: Date.now() }])
    }
  }, [stats])

  // Poll running apps
  useEffect(() => {
    const poll = () => fetch('/api/apps/running').then(r => r.json()).then(setApps).catch(() => {})
    poll()
    const id = setInterval(poll, 5000)
    return () => clearInterval(id)
  }, [])

  if (!connected) {
    return <div className="p-6 text-neutral-500">Connecting to system telemetry...</div>
  }

  return (
    <div className="p-4 space-y-6 overflow-y-auto h-full">
      <h2 className="text-lg font-medium">Activity Monitor</h2>

      {/* CPU */}
      <div>
        <div className="flex justify-between text-sm mb-1">
          <span className="text-neutral-400">CPU</span>
          <span className="text-neutral-300 font-mono">{Math.round(stats?.cpu || 0)}%</span>
        </div>
        <Bar value={stats?.cpu || 0} max={100} color="bg-blue-500" />
        <Sparkline data={history.map(h => h.cpu)} color="#3b82f6" />
        <div className="flex justify-between text-xs text-neutral-600 mt-1">
          <span>{stats?.num_cpu} cores</span>
          <span>Load: {stats?.load_avg}</span>
        </div>
      </div>

      {/* Memory */}
      <div>
        <div className="flex justify-between text-sm mb-1">
          <span className="text-neutral-400">Memory</span>
          <span className="text-neutral-300 font-mono">{Math.round(stats?.mem_percent || 0)}%</span>
        </div>
        <Bar value={stats?.mem_used || 0} max={stats?.mem_total || 1} color="bg-green-500" />
        <Sparkline data={history.map(h => h.mem)} color="#22c55e" />
        <div className="flex justify-between text-xs text-neutral-600 mt-1">
          <span>{fmtBytes(stats?.mem_used)} used</span>
          <span>{fmtBytes(stats?.mem_total)} total</span>
        </div>
      </div>

      {/* System Info */}
      <div className="grid grid-cols-2 gap-3">
        <InfoCard label="Temperature" value={stats?.temp > 0 ? `${Math.round(stats.temp)}°C` : 'N/A'} />
        <InfoCard label="Battery" value={stats?.battery >= 0 ? `${stats.battery}%${stats.charging ? ' ⚡' : ''}` : 'N/A'} />
        <InfoCard label="Network RX" value={fmtBytes(stats?.net_rx)} />
        <InfoCard label="Network TX" value={fmtBytes(stats?.net_tx)} />
        <InfoCard label="Uptime" value={stats?.uptime || '—'} />
        <InfoCard label="Hostname" value={stats?.hostname || '—'} />
      </div>

      {/* Running Apps */}
      <div>
        <h3 className="text-sm text-neutral-400 mb-2">Running Apps ({apps?.length || 0})</h3>
        {apps?.length === 0 && <p className="text-xs text-neutral-600">No apps running</p>}
        {apps?.map(app => (
          <div key={app.id} className="flex items-center justify-between py-1.5 border-b border-neutral-800/30 text-sm">
            <span className="text-neutral-300">{app.id}</span>
            <div className="flex items-center gap-3 text-xs text-neutral-500">
              <span>Port {app.host_port}</span>
              {app.traffic && <span>RX {fmtBytes(app.traffic.rx_bytes)}</span>}
              {app.traffic && <span>Idle {app.traffic.idle_for}</span>}
              <span className={app.running ? 'text-green-500' : 'text-red-400'}>{app.running ? 'Running' : 'Stopped'}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function InfoCard({ label, value }) {
  return (
    <div className="bg-neutral-900/60 rounded-lg border border-neutral-800/50 px-3 py-2">
      <div className="text-[10px] text-neutral-500 uppercase">{label}</div>
      <div className="text-sm text-neutral-300 font-mono mt-0.5">{value}</div>
    </div>
  )
}

function Sparkline({ data, color }) {
  if (data.length < 2) return null
  const max = Math.max(...data, 1)
  const h = 24
  const w = 200
  const points = data.map((v, i) => `${(i / (data.length - 1)) * w},${h - (v / max) * h}`).join(' ')
  return (
    <svg viewBox={`0 0 ${w} ${h}`} className="w-full mt-1" style={{ height: h }}>
      <polyline points={points} fill="none" stroke={color} strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" opacity="0.6" />
    </svg>
  )
}

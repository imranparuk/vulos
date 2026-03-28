import { useState, useEffect } from 'react'
import { useTelemetry } from './useTelemetry'

function useTime() {
  const [now, setNow] = useState(new Date())
  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(id)
  }, [])
  return now
}

function formatTime(date) {
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function formatDate(date) {
  return date.toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' })
}

function fmtBytes(b) {
  if (!b || b <= 0) return '0'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let v = b
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

export default function LifePulse({ compact = false, className = '' }) {
  const now = useTime()
  const { stats, connected } = useTelemetry()

  // Real data when connected, fallback otherwise
  const cpu = stats ? Math.round(stats.cpu) : null
  const mem = stats ? Math.round(stats.mem_percent) : null
  const battery = stats?.battery >= 0 ? stats.battery : null
  const charging = stats?.charging ?? false
  const temp = stats?.temp > 0 ? Math.round(stats.temp) : null
  const uptime = stats?.uptime || null
  const hostname = stats?.hostname || 'vulos'

  if (compact) {
    return (
      <div className={`flex items-center gap-3 text-xs font-mono ${className}`}>
        <span className="text-neutral-300">{formatTime(now)}</span>
        <span className="text-neutral-600">{formatDate(now)}</span>
        {cpu !== null && <span className="text-neutral-500">CPU {cpu}%</span>}
        {mem !== null && <span className="text-neutral-500">MEM {mem}%</span>}
        <div className="flex items-center gap-1">
          <span className={`w-1.5 h-1.5 rounded-full ${connected ? 'bg-green-500' : 'bg-neutral-600'}`} />
        </div>
        {battery !== null && (
          <span className="text-neutral-500">{battery}%{charging ? '⚡' : ''}</span>
        )}
      </div>
    )
  }

  return (
    <div className={`space-y-3 ${className}`}>
      {/* Clock */}
      <div className="text-center">
        <div className="text-4xl font-light text-neutral-200 font-mono tracking-wider">
          {formatTime(now)}
        </div>
        <div className="text-sm text-neutral-500 mt-1">
          {formatDate(now)}
        </div>
      </div>

      {/* Status cards */}
      <div className="grid grid-cols-2 gap-2">
        {connected && cpu !== null ? (
          <>
            <PulseCard label="CPU" value={`${cpu}%`} sub={stats?.load_avg || `${stats?.num_cpu} cores`} />
            <PulseCard label="Memory" value={`${mem}%`} sub={`${fmtBytes(stats?.mem_used)} / ${fmtBytes(stats?.mem_total)}`} />
            <PulseCard
              label="System"
              value={hostname}
              sub={uptime ? `Up ${uptime}` : 'Online'}
              dot="green"
            />
            <PulseCard
              label={battery !== null ? 'Battery' : 'Temp'}
              value={battery !== null ? `${battery}%${charging ? ' ⚡' : ''}` : temp ? `${temp}°C` : 'N/A'}
              sub={temp && battery !== null ? `${temp}°C` : connected ? 'Connected' : 'Offline'}
            />
          </>
        ) : (
          <>
            <PulseCard label="Next" value="No upcoming events" />
            <PulseCard label="Missions" value="Clear" sub="Nothing running" />
            <PulseCard label="Messages" value="All read" sub="No urgents" />
            <PulseCard label="System" value={connected ? 'Online' : 'Connecting...'} dot={connected ? 'green' : 'yellow'} />
          </>
        )}
      </div>
    </div>
  )
}

function PulseCard({ label, value, sub, dot }) {
  return (
    <div className="bg-neutral-900/60 backdrop-blur-sm rounded-lg border border-neutral-800/50 px-3 py-2.5">
      <div className="flex items-center gap-1.5">
        {dot && <span className={`w-1.5 h-1.5 rounded-full ${
          dot === 'green' ? 'bg-green-500' : dot === 'yellow' ? 'bg-yellow-500' : 'bg-red-500'
        }`} />}
        <span className="text-[10px] text-neutral-500 uppercase tracking-wider">{label}</span>
      </div>
      <div className="text-sm text-neutral-300 mt-0.5 truncate">{value}</div>
      {sub && <div className="text-[10px] text-neutral-600 truncate">{sub}</div>}
    </div>
  )
}

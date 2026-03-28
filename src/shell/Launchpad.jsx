import { useState, createElement, lazy, Suspense } from 'react'
import { useShell } from '../providers/ShellProvider'
import { getApps, searchApps, getAppsByCategory } from '../core/AppRegistry'
import Settings from '../core/Settings'

const Terminal = lazy(() => import('../builtin/terminal/Terminal'))
const ActivityMonitor = lazy(() => import('../builtin/activity/ActivityMonitor'))
const FileManager = lazy(() => import('../builtin/files/FileManager'))
const RemoteBrowser = lazy(() => import('../builtin/webbrowser/RemoteBrowser'))

const categoryLabels = {
  core: 'Core',
  productivity: 'Productivity',
  utilities: 'Utilities',
  media: 'Media',
  developer: 'Developer',
  system: 'System',
}

export default function Launchpad() {
  const { launchpadOpen, setLaunchpad, openWindow } = useShell()
  const [search, setSearch] = useState('')

  if (!launchpadOpen) return null

  const apps = search.trim() ? searchApps(search) : getApps()
  const grouped = search.trim() ? null : getAppsByCategory()

  const launch = async (app) => {
    // Built-in apps render as components, not iframes
    const loading = createElement('div', { className: 'p-4 text-neutral-500' }, 'Loading...')
    const builtins = {
      persona: () => createElement(Settings),
      terminal: () => createElement(Suspense, { fallback: loading }, createElement(Terminal)),
      activity: () => createElement(Suspense, { fallback: loading }, createElement(ActivityMonitor)),
      files: () => createElement(Suspense, { fallback: loading }, createElement(FileManager)),
      browser: () => createElement(Suspense, { fallback: loading }, createElement(RemoteBrowser)),
    }
    if (builtins[app.id]) {
      openWindow({ appId: app.id, title: app.name, icon: app.icon, component: builtins[app.id]() })
      setLaunchpad(false)
      setSearch('')
      return
    }

    // Launch app via backend — returns gateway URL (/app/{appId}/)
    try {
      const res = await fetch('/api/apps/launch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ app_id: app.id, app_port: app.port || 80, command: app.command || '' }),
      })
      if (res.ok) {
        const data = await res.json()
        // Use gateway URL — auth-protected, same origin, same cookie
        const url = data.url || `/app/${app.id}/`
        openWindow({ appId: app.id, title: app.name, url, icon: app.icon })
      } else {
        // Backend not available — use gateway URL anyway (will 404 if not running)
        openWindow({ appId: app.id, title: app.name, url: `/app/${app.id}/`, icon: app.icon })
      }
    } catch {
      openWindow({ appId: app.id, title: app.name, url: `/app/${app.id}/`, icon: app.icon })
    }
    setLaunchpad(false)
    setSearch('')
  }

  return (
    <div
      className="fixed inset-0 z-50 flex flex-col bg-neutral-950/90 backdrop-blur-2xl"
      onClick={(e) => { if (e.target === e.currentTarget) { setLaunchpad(false); setSearch('') } }}
    >
      {/* Search */}
      <div className="flex justify-center pt-12 pb-6 px-6">
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search applications..."
          autoFocus
          className="w-full max-w-md bg-neutral-800/60 border border-neutral-700/50 rounded-xl px-4 py-3 text-sm text-white outline-none placeholder:text-neutral-500 focus:border-neutral-600"
        />
      </div>

      {/* App grid */}
      <div className="flex-1 overflow-y-auto px-6 pb-12">
        <div className="max-w-3xl mx-auto">
          {grouped ? (
            Object.entries(grouped).map(([cat, catApps]) => (
              <div key={cat} className="mb-8">
                <h3 className="text-xs uppercase tracking-wider text-neutral-500 mb-3 px-1">
                  {categoryLabels[cat] || cat}
                </h3>
                <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 gap-4">
                  {catApps.map(app => (
                    <AppTile key={app.id} app={app} onLaunch={launch} />
                  ))}
                </div>
              </div>
            ))
          ) : (
            <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 gap-4">
              {apps.map(app => (
                <AppTile key={app.id} app={app} onLaunch={launch} />
              ))}
            </div>
          )}

          {apps.length === 0 && (
            <div className="text-center text-neutral-600 py-12">
              No applications found
            </div>
          )}
        </div>
      </div>

      {/* Close hint */}
      <div className="text-center pb-6">
        <kbd className="text-[10px] text-neutral-600 border border-neutral-800 rounded px-1.5 py-0.5">esc</kbd>
      </div>
    </div>
  )
}

function AppTile({ app, onLaunch }) {
  return (
    <button
      onClick={() => onLaunch(app)}
      className="flex flex-col items-center gap-2 p-3 rounded-xl hover:bg-neutral-800/50 transition-colors group"
    >
      <div className="w-14 h-14 rounded-2xl bg-neutral-800 border border-neutral-700/50 flex items-center justify-center text-2xl group-hover:border-neutral-600 transition-colors">
        {app.icon}
      </div>
      <span className="text-xs text-neutral-400 text-center truncate w-full">{app.name}</span>
    </button>
  )
}

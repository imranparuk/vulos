import { useState, useEffect } from 'react'

export default function RemoteBrowser() {
  const [nekoUrl, setNekoUrl] = useState(null)
  const [error, setError] = useState(null)

  useEffect(() => {
    // Get neko connection details from vulos server (authenticated)
    fetch('/api/browser/connect')
      .then(r => r.ok ? r.json() : null)
      .then(data => {
        if (data?.url) {
          setNekoUrl(data.url)
        } else {
          setError(data?.error || 'Browser service not running. It starts automatically with the OS.')
        }
      })
      .catch(() => setError('Could not reach server'))
  }, [])

  // On local Cage/WPE kiosk — just open URLs directly
  if (navigator.userAgent.includes('WPE') || navigator.userAgent.includes('Cog')) {
    return <LocalBrowser />
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-full bg-neutral-950 text-neutral-500 text-sm">
        <div className="text-center space-y-2">
          <p>{error}</p>
          <button onClick={() => window.location.reload()} className="btn">Retry</button>
        </div>
      </div>
    )
  }

  if (!nekoUrl) {
    return (
      <div className="flex items-center justify-center h-full bg-neutral-950">
        <span className="text-neutral-600 text-sm flex items-center gap-2">
          <span className="w-4 h-4 border-2 border-neutral-600 border-t-blue-500 rounded-full animate-spin" />
          Connecting to browser...
        </span>
      </div>
    )
  }

  return (
    <iframe
      src={nekoUrl}
      className="w-full h-full border-0"
      allow="autoplay; clipboard-write; clipboard-read; encrypted-media; microphone; camera"
    />
  )
}

function LocalBrowser() {
  const [url, setUrl] = useState('https://google.com')
  return (
    <div className="flex flex-col h-full bg-neutral-950">
      <div className="flex items-center gap-2 px-3 py-2 bg-neutral-900 border-b border-neutral-800/50">
        <input
          value={url} onChange={e => setUrl(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') window.open(url, '_blank') }}
          placeholder="Enter URL..." autoFocus
          className="flex-1 bg-neutral-800/60 border border-neutral-700/50 rounded-lg px-3 py-2 text-sm text-white outline-none"
        />
        <button onClick={() => window.open(url, '_blank')} className="btn">Open</button>
      </div>
      <div className="flex-1 flex items-center justify-center text-neutral-600 text-sm">
        URLs open in the system browser (WebKit)
      </div>
    </div>
  )
}

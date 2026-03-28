import { useState, useEffect, useCallback } from 'react'

export default function FileManager() {
  const [path, setPath] = useState('~')
  const [entries, setEntries] = useState([])
  const [search, setSearch] = useState('')
  const [searchResults, setSearchResults] = useState(null)
  const [loading, setLoading] = useState(false)
  const [preview, setPreview] = useState(null)

  const loadDir = useCallback(async (dir) => {
    setLoading(true)
    setSearchResults(null)
    setPreview(null)
    try {
      const res = await fetch('/api/exec', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ command: `ls -la --color=never "${dir}" 2>/dev/null | tail -n +2` }),
      })
      const data = await res.json()
      const lines = (data.output || '').split('\n').filter(l => l.trim())
      const parsed = lines.map(line => {
        const parts = line.split(/\s+/)
        if (parts.length < 9) return null
        const perms = parts[0]
        const size = parseInt(parts[4]) || 0
        const name = parts.slice(8).join(' ')
        if (name === '.' || name === '..') return null
        return {
          name,
          size,
          perms,
          isDir: perms.startsWith('d'),
          isLink: perms.startsWith('l'),
          modified: `${parts[5]} ${parts[6]} ${parts[7]}`,
        }
      }).filter(Boolean)
      setEntries(parsed)
      setPath(dir)
    } catch {
      setEntries([])
    }
    setLoading(false)
  }, [])

  useEffect(() => { loadDir('~') }, [loadDir])

  const navigate = (name) => {
    const newPath = path === '/' ? `/${name}` : `${path}/${name}`
    loadDir(newPath)
  }

  const goUp = () => {
    const parent = path.split('/').slice(0, -1).join('/') || '/'
    loadDir(parent)
  }

  const semanticSearch = async () => {
    if (!search.trim()) { setSearchResults(null); return }
    try {
      const res = await fetch('/api/recall/search', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: search, top_k: 20 }),
      })
      const results = await res.json()
      setSearchResults(results || [])
    } catch {
      setSearchResults([])
    }
  }

  const previewFile = async (filePath) => {
    try {
      const res = await fetch('/api/exec', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ command: `head -50 "${filePath}" 2>/dev/null` }),
      })
      const data = await res.json()
      setPreview({ path: filePath, content: data.output || '(empty)' })
    } catch {
      setPreview({ path: filePath, content: '(could not read)' })
    }
  }

  const fmtSize = (b) => {
    if (b < 1024) return `${b} B`
    if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`
    if (b < 1024 * 1024 * 1024) return `${(b / (1024 * 1024)).toFixed(1)} MB`
    return `${(b / (1024 * 1024 * 1024)).toFixed(1)} GB`
  }

  return (
    <div className="flex h-full bg-neutral-950 text-neutral-200">
      {/* Main panel */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Path bar */}
        <div className="flex items-center gap-2 px-3 py-2 bg-neutral-900 border-b border-neutral-800/50">
          <button onClick={goUp} className="text-neutral-500 hover:text-white text-sm px-1">←</button>
          <button onClick={() => loadDir('~')} className="text-neutral-500 hover:text-white text-xs">~</button>
          <span className="text-xs text-neutral-400 font-mono truncate flex-1">{path}</span>
          <button onClick={() => loadDir(path)} className="text-xs text-neutral-600 hover:text-neutral-400">↻</button>
        </div>

        {/* Search bar */}
        <div className="flex items-center gap-2 px-3 py-2 border-b border-neutral-800/30">
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && semanticSearch()}
            placeholder="Search files semantically..."
            className="flex-1 bg-neutral-900/60 border border-neutral-800 rounded-lg px-3 py-1.5 text-sm outline-none focus:border-neutral-600"
          />
          <button onClick={semanticSearch} className="text-xs bg-neutral-800 hover:bg-neutral-700 px-2 py-1.5 rounded-lg">Search</button>
        </div>

        {/* File list */}
        <div className="flex-1 overflow-y-auto">
          {loading && <div className="p-4 text-neutral-600 text-sm">Loading...</div>}

          {searchResults ? (
            <>
              <div className="px-3 py-1.5 text-xs text-neutral-600">{searchResults.length} results for "{search}"</div>
              {searchResults.map((r, i) => (
                <div
                  key={i}
                  onClick={() => previewFile(r.metadata?.abs_path || r.metadata?.path)}
                  className="flex items-center gap-3 px-3 py-2 hover:bg-neutral-800/40 cursor-pointer border-b border-neutral-800/20"
                >
                  <span className="text-neutral-600 text-sm">📄</span>
                  <div className="flex-1 min-w-0">
                    <div className="text-sm truncate">{r.metadata?.path}</div>
                    <div className="text-xs text-neutral-600 truncate">{r.content?.slice(0, 80)}</div>
                  </div>
                  <span className="text-[10px] text-neutral-700 shrink-0">{Math.round(r.score * 100)}%</span>
                </div>
              ))}
            </>
          ) : (
            entries.map((e, i) => (
              <div
                key={i}
                onClick={() => e.isDir ? navigate(e.name) : previewFile(`${path}/${e.name}`)}
                className="flex items-center gap-3 px-3 py-1.5 hover:bg-neutral-800/40 cursor-pointer border-b border-neutral-800/20"
              >
                <span className="text-sm w-5 text-center">{e.isDir ? '📁' : e.isLink ? '🔗' : '📄'}</span>
                <span className="text-sm flex-1 truncate">{e.name}</span>
                <span className="text-xs text-neutral-600 w-16 text-right">{e.isDir ? '—' : fmtSize(e.size)}</span>
                <span className="text-xs text-neutral-700 w-28">{e.modified}</span>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Preview panel */}
      {preview && (
        <div className="w-80 border-l border-neutral-800/50 flex flex-col shrink-0">
          <div className="flex items-center justify-between px-3 py-2 bg-neutral-900 border-b border-neutral-800/50">
            <span className="text-xs text-neutral-400 truncate">{preview.path.split('/').pop()}</span>
            <button onClick={() => setPreview(null)} className="text-xs text-neutral-600 hover:text-neutral-400">✕</button>
          </div>
          <pre className="flex-1 overflow-auto p-3 text-xs text-neutral-400 font-mono whitespace-pre-wrap">{preview.content}</pre>
        </div>
      )}
    </div>
  )
}

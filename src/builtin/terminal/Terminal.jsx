import { useEffect, useRef } from 'react'
import { Terminal as XTerm } from 'xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import 'xterm/css/xterm.css'

const WS_URL = `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/api/pty`

export default function Terminal() {
  const containerRef = useRef(null)
  const termRef = useRef(null)
  const wsRef = useRef(null)

  useEffect(() => {
    if (!containerRef.current) return

    const term = new XTerm({
      cursorBlink: true,
      cursorStyle: 'bar',
      fontFamily: "'SF Mono', 'Cascadia Code', 'Fira Code', monospace",
      fontSize: 14,
      lineHeight: 1.2,
      theme: {
        background: '#0a0a0a',
        foreground: '#e5e5e5',
        cursor: '#e5e5e5',
        selectionBackground: '#333333',
        black: '#0a0a0a',
        red: '#ef4444',
        green: '#22c55e',
        yellow: '#eab308',
        blue: '#3b82f6',
        magenta: '#a855f7',
        cyan: '#06b6d4',
        white: '#e5e5e5',
        brightBlack: '#666666',
        brightRed: '#f87171',
        brightGreen: '#4ade80',
        brightYellow: '#facc15',
        brightBlue: '#60a5fa',
        brightMagenta: '#c084fc',
        brightCyan: '#22d3ee',
        brightWhite: '#ffffff',
      },
    })

    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.loadAddon(new WebLinksAddon())

    term.open(containerRef.current)
    fitAddon.fit()
    termRef.current = term

    // Connect WebSocket
    const cols = term.cols
    const rows = term.rows
    const ws = new WebSocket(`${WS_URL}?cols=${cols}&rows=${rows}`)
    ws.binaryType = 'arraybuffer'
    wsRef.current = ws

    ws.onopen = () => {
      term.focus()
    }

    ws.onmessage = (e) => {
      const data = typeof e.data === 'string' ? e.data : new TextDecoder().decode(e.data)
      term.write(data)
    }

    ws.onclose = () => {
      term.write('\r\n\x1b[90m[session ended]\x1b[0m\r\n')
    }

    // Terminal → WebSocket
    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data)
      }
    })

    // Handle resize
    term.onResize(({ cols, rows }) => {
      if (ws.readyState === WebSocket.OPEN) {
        // Send resize control message: \x01{cols},{rows}
        ws.send(new Uint8Array([1, ...new TextEncoder().encode(`${cols},${rows}`)]))
      }
    })

    // Window resize → fit terminal
    const onResize = () => fitAddon.fit()
    window.addEventListener('resize', onResize)
    const resizeObserver = new ResizeObserver(() => fitAddon.fit())
    resizeObserver.observe(containerRef.current)

    return () => {
      window.removeEventListener('resize', onResize)
      resizeObserver.disconnect()
      ws.close()
      term.dispose()
    }
  }, [])

  return (
    <div
      ref={containerRef}
      className="w-full h-full"
      style={{ padding: '4px', background: '#0a0a0a' }}
    />
  )
}

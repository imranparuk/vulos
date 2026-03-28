import { useShell } from '../providers/ShellProvider'

export default function Popout() {
  const { popout, closePopout } = useShell()

  if (!popout) return null

  return (
    <div className="fixed inset-0 z-[100] bg-neutral-950 flex flex-col">
      {/* Minimal top bar */}
      <div className="flex items-center justify-between px-3 py-1.5 bg-neutral-900 border-b border-neutral-800/50 shrink-0">
        <div className="flex items-center gap-2">
          <button
            onClick={closePopout}
            className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs text-neutral-400 hover:text-white hover:bg-neutral-800 transition-colors"
          >
            <svg viewBox="0 0 16 16" className="w-3.5 h-3.5"><path d="M10 3L5 8l5 5" stroke="currentColor" strokeWidth="2" fill="none" /></svg>
            Back
          </button>
          <span className="text-xs text-neutral-600">|</span>
          <span className="text-sm text-neutral-400">{popout.icon} {popout.title}</span>
        </div>
        <div className="flex items-center gap-2">
          <a
            href={popout.url}
            target="_blank"
            rel="noopener"
            className="text-[10px] text-neutral-600 hover:text-neutral-400 transition-colors"
          >
            Open in new tab
          </a>
        </div>
      </div>

      {/* Full-viewport iframe — zero overhead, maximum performance */}
      <iframe
        src={popout.url}
        title={popout.title}
        className="flex-1 w-full border-0"
        sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
      />
    </div>
  )
}

import { useShell } from '../providers/ShellProvider'
import LifePulse from '../core/SystemPulse'
import Portal from '../core/Portal'
import Window from '../shell/Window'
import Dock from '../shell/Dock'
import Launchpad from '../shell/Launchpad'
import Toasts from '../shell/Toasts'

export default function DesktopCanvas() {
  const { windows, allWindows, chatOpen } = useShell()

  return (
    <div className="fixed inset-0 bg-neutral-950 overflow-hidden">
      {/* Wallpaper */}
      <div className="absolute inset-0 bg-gradient-to-br from-neutral-950 via-neutral-900 to-neutral-950 pointer-events-none">
        <div className="absolute inset-0 opacity-[0.03]">
          <div className="absolute top-[20%] left-[30%] w-[500px] h-[500px] rounded-full bg-blue-500 blur-[150px]" />
          <div className="absolute bottom-[25%] right-[20%] w-[400px] h-[400px] rounded-full bg-violet-500 blur-[150px]" />
        </div>
      </div>

      {/* Menu bar */}
      <div className="absolute top-0 left-0 right-0 z-40 h-8 flex items-center justify-between px-4 bg-neutral-900/50 backdrop-blur-md border-b border-neutral-800/30">
        <span className="text-xs font-semibold text-neutral-400">vula</span>
        <LifePulse compact />
      </div>

      {/* Windows area — render ALL windows persistently, hide inactive desktops via CSS */}
      <div className="absolute inset-0 pt-8 pb-16">
        {allWindows.map(win => (
          <Window key={win.id} win={{ ...win, minimized: win.minimized || !win._visible }} />
        ))}

        {/* Empty state */}
        {windows.length === 0 && !chatOpen && (
          <div className="h-full flex flex-col items-center justify-center">
            <LifePulse />
          </div>
        )}
      </div>

      {/* Chat panel — right side */}
      {chatOpen && (
        <div className="absolute top-8 right-0 bottom-16 w-[380px] z-30">
          <Portal />
        </div>
      )}

      {/* Dock */}
      <Dock />

      {/* Launchpad overlay */}
      <Launchpad />

      {/* Toast notifications */}
      <Toasts />
    </div>
  )
}

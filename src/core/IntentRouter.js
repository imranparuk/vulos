import { searchApps } from './AppRegistry'

export function classifyIntent(input) {
  const trimmed = input.trim()
  if (!trimmed) return { type: 'empty' }

  // System command: /settings, /wifi, /backup etc.
  if (trimmed.startsWith('/')) {
    const cmd = trimmed.slice(1).toLowerCase().trim()
    const sys = {
      settings: { action: 'open_persona', label: 'My Persona', url: 'http://persona.vulos' },
      persona: { action: 'open_persona', label: 'My Persona', url: 'http://persona.vulos' },
      wifi: { action: 'open_network', label: 'Network', url: 'http://network.vulos' },
      network: { action: 'open_network', label: 'Network', url: 'http://network.vulos' },
      backup: { action: 'open_vault', label: 'Vault', url: 'http://vault.vulos' },
      vault: { action: 'open_vault', label: 'Vault', url: 'http://vault.vulos' },
      files: { action: 'open_finder', label: 'The Finder', url: 'http://finder.vulos' },
    }
    if (sys[cmd]) return { type: 'system', ...sys[cmd] }
    return { type: 'command', value: cmd }
  }

  // App match
  const matches = searchApps(trimmed)
  if (matches.length === 1) return { type: 'launch_service', service: matches[0] }
  if (matches.length > 1 && matches.length <= 3) return { type: 'service_suggestions', matches }

  // Inline math
  if (/[\d]+\s*[+\-*/÷×%]\s*[\d]/.test(trimmed) || /\d+%\s*(of|on|tax)/i.test(trimmed)) {
    return { type: 'math', value: trimmed }
  }

  // Everything else is a mission for the AI
  return { type: 'mission', value: trimmed }
}

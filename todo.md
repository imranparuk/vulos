# Vula OS — Project Todo

## Completed ✅

Full system built — see git history. Covers shell, AI, system services, auth, storage,
networking, energy, tunnel, build scripts, app store, 3 app services, ONNX embeddings.

---

## AUDIT FINDINGS — Fixed

### CRITICAL: Auth Enforcement ✅
- [x] Middleware now enforces auth — returns 401 on all /api/ and /app/ routes without valid session
- [x] Public endpoint whitelist: /health, /api/auth/providers, /login/*, /callback/*
- [x] Frontend assets (non-api, non-app paths) served without auth (React handles its own gate)

### CRITICAL: Sandbox Security ✅
- [x] Dangerous code validation (blocks subprocess, os.system, eval, exec, fork bombs)
- [x] 100KB code size limit
- [x] 5-minute execution timeout per sandbox script
- [x] Sandbox proxy protected by auth middleware (all /api/ routes)

### HIGH: Dev Mode Bypass ✅
- [x] "Continue without login" only shows in Vite dev mode (import.meta.env.DEV)
- [x] Production builds never show the bypass

### HIGH: AI Viewports ✅ (resolved by auth fix)
- [x] All /api/ endpoints now require session — AI HTML can't call unprotected endpoints

### MEDIUM: AI-Generated Apps Saveable ✅
- [x] Save button (💾) in AI viewport window title bar
- [x] POST /api/ai-apps/save — persists HTML + Python to ~/.vulos/ai-apps/
- [x] GET /api/ai-apps — list all saved AI apps with metadata
- [x] GET /api/ai-apps/{id}/html, /python — retrieve saved code
- [x] DELETE /api/ai-apps/{id} — remove saved app

### NEW: Browser Profiles (Firefox-style isolation) ✅
- [x] Browser profile store (`services/profiles/browser.go`)
- [x] Each profile: own data dir, cookie jar, color, icon
- [x] Default profiles: Personal, Work, Private (auto-created)
- [x] Bind apps to profiles (calculator always uses Work profile)
- [x] Clear data per profile without deleting it
- [x] REST API: CRUD + bind + clear

### NEW: AI OS Control ✅
- [x] AI can include `<os-action>` blocks to control the OS
- [x] Supported: open-app, close-app, notify, energy-mode, exec
- [x] Portal parses actions and calls backend APIs
- [x] System prompt teaches the AI about OS control capabilities
- [x] Backend endpoints: /api/os/open-app, /close-app, /notify, /energy-mode

---

### MEDIUM: Persistence ✅
- [x] Load chat history on Portal mount — restores latest conversation from backend
- [x] Persist window/desktop state to localStorage — auto-save/restore on refresh
- [x] AppRegistry cleaned — removed 6 unimplemented stubs, kept only builtin + installed apps

### LOW: Polish ✅
- [x] Vault/Backup settings UI — status, backup now, sync to device, other devices list
- [x] Recall/Search settings UI — file count, index status, re-index button
- [x] AI Apps gallery UI in Settings — list saved apps, open, delete
- [x] Ad blocker improved — 50+ domains, loads EasyList-format blocklist.txt, class/id matching, nested ad div removal

## All Items Complete

**Files:** 47 Go, 25 frontend, 3 Python apps, 4 build scripts
**Backend services:** 20 | **API endpoints:** 110+

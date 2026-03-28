"""Vula OS — Universal Memory (Notes)
Every thought indexed by Recall. Markdown editor with instant search.
"""
import http.server
import json
import os
import time
import urllib.request

PORT = int(os.environ.get("PORT", os.environ.get("VULOS_PORT", 8080)))
VULOS_API = os.environ.get("VULOS_API", "http://localhost:8080")
DATA_DIR = os.environ.get("NOTES_DIR", os.path.expanduser("~/.vulos/data/notes"))
os.makedirs(DATA_DIR, exist_ok=True)

def list_notes():
    notes = []
    for f in sorted(os.listdir(DATA_DIR), reverse=True):
        if f.endswith(".md"):
            path = os.path.join(DATA_DIR, f)
            with open(path) as fh:
                content = fh.read()
            title = content.split("\n")[0].lstrip("# ").strip() or f
            notes.append({"id": f[:-3], "title": title, "preview": content[:200], "modified": os.path.getmtime(path)})
    return notes

def get_note(note_id):
    path = os.path.join(DATA_DIR, note_id + ".md")
    if not os.path.exists(path): return None
    with open(path) as f: return f.read()

def save_note(note_id, content):
    if not note_id:
        note_id = str(int(time.time() * 1000))
    path = os.path.join(DATA_DIR, note_id + ".md")
    with open(path, "w") as f: f.write(content)
    # Trigger Recall re-index
    try: urllib.request.urlopen(urllib.request.Request(VULOS_API + "/api/recall/index", method="POST"), timeout=2)
    except: pass
    return note_id

def delete_note(note_id):
    path = os.path.join(DATA_DIR, note_id + ".md")
    if os.path.exists(path): os.remove(path)

SHELL_HTML = """<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Universal Memory</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{background:#0a0a0a;color:#e5e5e5;font-family:system-ui;display:flex;height:100vh}
.sidebar{width:240px;border-right:1px solid #222;display:flex;flex-direction:column;flex-shrink:0}
.sidebar .top{padding:8px;border-bottom:1px solid #222;display:flex;gap:4px}
.sidebar input{flex:1;background:#1a1a1a;border:1px solid #333;border-radius:6px;padding:6px 8px;color:#e5e5e5;font-size:13px;outline:none}
.sidebar button{background:#333;border:none;border-radius:6px;padding:6px 10px;color:#e5e5e5;cursor:pointer;font-size:12px}
.sidebar button:hover{background:#444}
.list{flex:1;overflow-y:auto}
.list .item{padding:10px 12px;border-bottom:1px solid #1a1a1a;cursor:pointer;font-size:13px}
.list .item:hover{background:#111}
.list .item.active{background:#1a1a1a;border-left:2px solid #3b82f6}
.list .item .title{font-weight:500}
.list .item .preview{color:#666;font-size:11px;margin-top:2px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.editor{flex:1;display:flex;flex-direction:column}
.editor textarea{flex:1;background:#0a0a0a;color:#e5e5e5;border:none;padding:16px;font-family:'SF Mono',monospace;font-size:14px;line-height:1.6;resize:none;outline:none}
.editor .bar{padding:6px 12px;background:#111;border-top:1px solid #222;display:flex;justify-content:between;font-size:11px;color:#666}
</style>
</head>
<body>
<div class="sidebar">
  <div class="top">
    <input id="search" placeholder="Search..." oninput="searchNotes()">
    <button onclick="newNote()">+</button>
  </div>
  <div class="list" id="list"></div>
</div>
<div class="editor">
  <textarea id="editor" placeholder="Start writing..." oninput="autoSave()"></textarea>
  <div class="bar"><span id="status">Ready</span></div>
</div>
<script>
let notes=[], currentId=null, saveTimer=null;
const list=document.getElementById('list'), editor=document.getElementById('editor'), status=document.getElementById('status');

async function loadNotes(){
  const res=await fetch('/api/notes'); notes=await res.json();
  renderList();
  if(notes.length>0&&!currentId) selectNote(notes[0].id);
}

function renderList(filter){
  const filtered=filter?notes.filter(n=>n.title.toLowerCase().includes(filter)||n.preview.toLowerCase().includes(filter)):notes;
  list.innerHTML=filtered.map(n=>`<div class="item ${n.id===currentId?'active':''}" onclick="selectNote('${n.id}')"><div class="title">${n.title||'Untitled'}</div><div class="preview">${n.preview}</div></div>`).join('');
}

async function selectNote(id){
  currentId=id;
  const res=await fetch('/api/notes/'+id); editor.value=await res.text();
  renderList();
}

function newNote(){
  currentId=null; editor.value='# New Note\\n\\n';
  editor.focus(); autoSave();
}

function autoSave(){
  clearTimeout(saveTimer);
  status.textContent='Saving...';
  saveTimer=setTimeout(async()=>{
    const res=await fetch('/api/notes'+(currentId?'/'+currentId:''),{method:'POST',headers:{'Content-Type':'text/plain'},body:editor.value});
    const data=await res.json();
    if(!currentId) currentId=data.id;
    status.textContent='Saved';
    loadNotes();
  },500);
}

function searchNotes(){
  const q=document.getElementById('search').value.toLowerCase();
  renderList(q);
}

loadNotes();
</script>
</body>
</html>"""


class NotesHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/":
            self.send_html(SHELL_HTML)
        elif self.path == "/api/notes":
            self.send_json(list_notes())
        elif self.path.startswith("/api/notes/"):
            note_id = self.path.split("/api/notes/")[1]
            content = get_note(note_id)
            if content is None:
                self.send_error(404)
            else:
                self.send_response(200)
                self.send_header("Content-Type", "text/plain")
                self.end_headers()
                self.wfile.write(content.encode())
        else:
            self.send_error(404)

    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(length).decode() if length else ""
        if self.path == "/api/notes":
            note_id = save_note(None, body)
            self.send_json({"id": note_id})
        elif self.path.startswith("/api/notes/"):
            note_id = self.path.split("/api/notes/")[1]
            save_note(note_id, body)
            self.send_json({"id": note_id})

    def do_DELETE(self):
        if self.path.startswith("/api/notes/"):
            note_id = self.path.split("/api/notes/")[1]
            delete_note(note_id)
            self.send_json({"status": "deleted"})

    def send_html(self, html):
        self.send_response(200)
        self.send_header("Content-Type", "text/html")
        self.end_headers()
        self.wfile.write(html.encode())

    def send_json(self, data):
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        self.wfile.write(json.dumps(data).encode())

    def log_message(self, format, *args): pass

print(f"[notes] Universal Memory on port {PORT}")
http.server.HTTPServer(("0.0.0.0", PORT), NotesHandler).serve_forever()

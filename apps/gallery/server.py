"""Vula OS — Media Gallery
Smart photo/video organizer. Scans filesystem, shows grid, search via Recall.
"""
import http.server
import json
import mimetypes
import os
import urllib.request

PORT = int(os.environ.get("PORT", os.environ.get("VULOS_PORT", 8080)))
VULOS_API = os.environ.get("VULOS_API", "http://localhost:8080")
MEDIA_DIR = os.environ.get("MEDIA_DIR", os.path.expanduser("~/.vulos/data"))

MEDIA_EXTS = {".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".bmp", ".svg",
              ".mp4", ".mov", ".avi", ".mkv", ".webm"}

def scan_media(root, limit=200):
    media = []
    for dirpath, _, filenames in os.walk(root):
        for f in sorted(filenames, reverse=True):
            ext = os.path.splitext(f)[1].lower()
            if ext not in MEDIA_EXTS:
                continue
            path = os.path.join(dirpath, f)
            rel = os.path.relpath(path, root)
            stat = os.stat(path)
            media.append({
                "name": f,
                "path": rel,
                "size": stat.st_size,
                "modified": stat.st_mtime,
                "type": "video" if ext in {".mp4",".mov",".avi",".mkv",".webm"} else "image",
            })
            if len(media) >= limit:
                return media
    return media

SHELL_HTML = """<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Media Gallery</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{background:#0a0a0a;color:#e5e5e5;font-family:system-ui}
.bar{padding:8px;background:#111;border-bottom:1px solid #222;display:flex;gap:8px}
.bar input{flex:1;background:#1a1a1a;border:1px solid #333;border-radius:8px;padding:8px 12px;color:#e5e5e5;font-size:14px;outline:none}
.bar input:focus{border-color:#555}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(160px,1fr));gap:4px;padding:4px}
.grid .item{aspect-ratio:1;background:#111;border-radius:4px;overflow:hidden;cursor:pointer;position:relative}
.grid .item img{width:100%;height:100%;object-fit:cover}
.grid .item video{width:100%;height:100%;object-fit:cover}
.grid .item .badge{position:absolute;top:4px;right:4px;background:#000a;border-radius:4px;padding:2px 6px;font-size:10px}
.viewer{position:fixed;inset:0;background:#000e;display:flex;align-items:center;justify-content:center;z-index:100;cursor:pointer}
.viewer img,.viewer video{max-width:95vw;max-height:95vh;object-fit:contain}
.count{padding:8px 12px;font-size:12px;color:#666}
</style>
</head>
<body>
<div class="bar">
  <input id="search" placeholder="Search photos & videos..." oninput="search()">
</div>
<div class="count" id="count"></div>
<div class="grid" id="grid"></div>
<div class="viewer" id="viewer" onclick="closeViewer()" style="display:none"></div>
<script>
let allMedia=[];
const grid=document.getElementById('grid'),viewer=document.getElementById('viewer'),count=document.getElementById('count');

async function load(){
  const res=await fetch('/api/media'); allMedia=await res.json();
  render(allMedia);
}

function render(items){
  count.textContent=items.length+' items';
  grid.innerHTML=items.map(m=>{
    if(m.type==='video')
      return `<div class="item" onclick="view('${m.path}','video')"><video src="/media/${m.path}" muted preload="metadata"></video><div class="badge">▶</div></div>`;
    return `<div class="item" onclick="view('${m.path}','image')"><img src="/media/${m.path}" loading="lazy"></div>`;
  }).join('');
}

function view(path,type){
  if(type==='video')
    viewer.innerHTML=`<video src="/media/${path}" controls autoplay style="max-width:95vw;max-height:95vh">`;
  else
    viewer.innerHTML=`<img src="/media/${path}">`;
  viewer.style.display='flex';
}

function closeViewer(){viewer.style.display='none';viewer.innerHTML='';}

async function search(){
  const q=document.getElementById('search').value.trim();
  if(!q){render(allMedia);return;}
  // Use Recall for semantic search
  try{
    const res=await fetch('/api/search?q='+encodeURIComponent(q));
    const results=await res.json();
    render(results);
  }catch{
    // Fallback to filename filter
    render(allMedia.filter(m=>m.name.toLowerCase().includes(q.toLowerCase())));
  }
}

document.addEventListener('keydown',e=>{if(e.key==='Escape')closeViewer();});
load();
</script>
</body>
</html>"""


class GalleryHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/":
            self.send_html(SHELL_HTML)
        elif self.path == "/api/media":
            self.send_json(scan_media(MEDIA_DIR))
        elif self.path.startswith("/api/search?q="):
            self.handle_search()
        elif self.path.startswith("/media/"):
            self.serve_file()
        else:
            self.send_error(404)

    def handle_search(self):
        query = self.path.split("q=", 1)[1]
        query = urllib.request.unquote(query)
        try:
            req = urllib.request.Request(
                VULOS_API + "/api/recall/search",
                data=json.dumps({"query": query, "top_k": 50}).encode(),
                headers={"Content-Type": "application/json"},
            )
            resp = urllib.request.urlopen(req, timeout=5)
            results = json.loads(resp.read())
            # Filter to media files only
            media = []
            for r in results:
                path = r.get("metadata", {}).get("path", "")
                ext = os.path.splitext(path)[1].lower()
                if ext in MEDIA_EXTS:
                    media.append({
                        "name": os.path.basename(path),
                        "path": path,
                        "type": "video" if ext in {".mp4",".mov",".avi",".mkv",".webm"} else "image",
                        "score": r.get("score", 0),
                    })
            self.send_json(media)
        except Exception as e:
            self.send_json([])

    def serve_file(self):
        rel = self.path[len("/media/"):]
        path = os.path.join(MEDIA_DIR, rel)
        path = os.path.realpath(path)
        # Security: ensure within MEDIA_DIR
        if not path.startswith(os.path.realpath(MEDIA_DIR)):
            self.send_error(403)
            return
        if not os.path.isfile(path):
            self.send_error(404)
            return
        mime, _ = mimetypes.guess_type(path)
        self.send_response(200)
        self.send_header("Content-Type", mime or "application/octet-stream")
        self.send_header("Content-Length", str(os.path.getsize(path)))
        self.send_header("Cache-Control", "public, max-age=86400")
        self.end_headers()
        with open(path, "rb") as f:
            while chunk := f.read(65536):
                self.wfile.write(chunk)

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

print(f"[gallery] Media Gallery on port {PORT}")
http.server.HTTPServer(("0.0.0.0", PORT), GalleryHandler).serve_forever()

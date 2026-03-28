"""Vula OS — Smart Browser
Ad-stripping web viewer with AI summarization.
Proxies pages through the server, strips ads/trackers, optionally summarizes.
"""
import http.server
import json
import os
import re
import urllib.request
import urllib.error
from html.parser import HTMLParser

PORT = int(os.environ.get("PORT", os.environ.get("VULOS_PORT", 8080)))
VULOS_API = os.environ.get("VULOS_API", "http://localhost:8080")

# Ad/tracker domain blocklist — loaded from file if available, else defaults
_DEFAULT_AD_DOMAINS = {
    "doubleclick.net", "googlesyndication.com", "googleadservices.com",
    "facebook.net", "fbcdn.net", "analytics.google.com",
    "amazon-adsystem.com", "adnxs.com", "adsrvr.org",
    "criteo.com", "outbrain.com", "taboola.com",
    "scorecardresearch.com", "quantserve.com", "bluekai.com",
    "moatads.com", "2mdn.net", "serving-sys.com",
    "smartadserver.com", "pubmatic.com", "rubiconproject.com",
    "openx.net", "casalemedia.com", "lijit.com",
    "mathtag.com", "turn.com", "nexac.com",
    "demdex.net", "krxd.net", "exelator.com",
    "agkn.com", "rlcdn.com", "bidswitch.net",
    "contextweb.com", "spotxchange.com", "yieldmanager.com",
    "googletagmanager.com", "googletagservices.com",
    "googlesyndication.com", "google-analytics.com",
    "hotjar.com", "fullstory.com", "mouseflow.com",
    "clarity.ms", "newrelic.com", "nr-data.net",
}

def load_blocklist():
    """Load blocklist from EasyList-format file if available."""
    domains = set(_DEFAULT_AD_DOMAINS)
    blocklist_path = os.path.join(os.path.dirname(__file__), "blocklist.txt")
    if os.path.exists(blocklist_path):
        with open(blocklist_path) as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("!") or line.startswith("["):
                    continue
                # EasyList domain rules: ||domain.com^
                if line.startswith("||") and line.endswith("^"):
                    domains.add(line[2:-1])
                # Plain domain lines
                elif "." in line and " " not in line and "/" not in line:
                    domains.add(line)
    return domains

AD_DOMAINS = load_blocklist()

class AdStripper(HTMLParser):
    """Strips ad-related elements from HTML — scripts, iframes, images, divs with ad classes."""
    AD_CLASSES = {"ad", "ads", "advert", "advertisement", "banner-ad", "ad-container", "ad-wrapper",
                  "google-ad", "sponsored", "promoted", "dfp-ad", "ad-slot", "ad-unit"}

    def __init__(self):
        super().__init__()
        self.output = []
        self.skip = False
        self.skip_depth = 0

    def _is_ad(self, tag, attrs):
        attrs_dict = dict(attrs)
        # Check src/href against blocklist
        for attr in ("src", "href", "data-src"):
            val = attrs_dict.get(attr, "")
            if val and any(ad in val for ad in AD_DOMAINS):
                return True
        # Check class names for ad patterns
        classes = attrs_dict.get("class", "").lower().split()
        if any(c in self.AD_CLASSES for c in classes):
            return True
        # Check id for ad patterns
        elem_id = attrs_dict.get("id", "").lower()
        if any(ad in elem_id for ad in ("ad-", "ads-", "advert", "banner-ad", "google_ads")):
            return True
        return False

    def handle_starttag(self, tag, attrs):
        if self.skip:
            self.skip_depth += 1
            return
        if tag in ("script", "iframe", "img", "div", "aside", "section") and self._is_ad(tag, attrs):
            self.skip = True
            self.skip_depth = 1
            return
        attr_str = " ".join(f'{k}="{v}"' for k, v in attrs)
        self.output.append(f"<{tag} {attr_str}>" if attr_str else f"<{tag}>")

    def handle_endtag(self, tag):
        if self.skip:
            self.skip_depth -= 1
            if self.skip_depth <= 0:
                self.skip = False
                self.skip_depth = 0
            return
        if not self.skip:
            self.output.append(f"</{tag}>")

    def handle_data(self, data):
        if not self.skip:
            self.output.append(data)

    def get_output(self):
        return "".join(self.output)


SHELL_HTML = """<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Smart Browser</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { background: #0a0a0a; color: #e5e5e5; font-family: system-ui; }
.bar { display: flex; gap: 8px; padding: 8px; background: #111; border-bottom: 1px solid #222; }
.bar input { flex: 1; background: #1a1a1a; border: 1px solid #333; border-radius: 8px; padding: 8px 12px; color: #e5e5e5; font-size: 14px; outline: none; }
.bar input:focus { border-color: #555; }
.bar button { background: #333; border: none; border-radius: 8px; padding: 8px 16px; color: #e5e5e5; cursor: pointer; font-size: 13px; }
.bar button:hover { background: #444; }
.content { padding: 16px; }
.content iframe { width: 100%; height: calc(100vh - 52px); border: none; background: white; border-radius: 4px; }
.summary { background: #111; border: 1px solid #222; border-radius: 8px; padding: 12px; margin-bottom: 12px; font-size: 14px; line-height: 1.5; }
.summary h3 { color: #888; font-size: 11px; text-transform: uppercase; margin-bottom: 4px; }
</style>
</head>
<body>
<div class="bar">
  <input id="url" type="text" placeholder="Enter URL..." autofocus>
  <button onclick="go()">Go</button>
  <button onclick="summarize()">Summarize</button>
</div>
<div id="summary"></div>
<div class="content"><iframe id="frame"></iframe></div>
<script>
const frame = document.getElementById('frame');
const urlInput = document.getElementById('url');
const summaryDiv = document.getElementById('summary');

urlInput.addEventListener('keydown', e => { if (e.key === 'Enter') go(); });

function go() {
  let url = urlInput.value.trim();
  if (!url) return;
  if (!url.startsWith('http')) url = 'https://' + url;
  frame.src = '/browse?url=' + encodeURIComponent(url);
}

async function summarize() {
  summaryDiv.innerHTML = '<div class="summary"><h3>AI Summary</h3>Loading...</div>';
  const url = urlInput.value.trim();
  try {
    const res = await fetch('/summarize?url=' + encodeURIComponent(url));
    const data = await res.json();
    summaryDiv.innerHTML = '<div class="summary"><h3>AI Summary</h3>' + data.summary + '</div>';
  } catch(e) {
    summaryDiv.innerHTML = '<div class="summary"><h3>Error</h3>Could not summarize.</div>';
  }
}
</script>
</body>
</html>"""


class BrowserHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/" or self.path == "":
            self.send_html(SHELL_HTML)
        elif self.path.startswith("/browse?url="):
            self.handle_browse()
        elif self.path.startswith("/summarize?url="):
            self.handle_summarize()
        else:
            self.send_error(404)

    def handle_browse(self):
        url = self.path.split("url=", 1)[1]
        url = urllib.request.unquote(url)
        try:
            req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0 (compatible; VulaOS)"})
            resp = urllib.request.urlopen(req, timeout=10)
            content = resp.read().decode("utf-8", errors="replace")
            # Strip ads
            stripper = AdStripper()
            stripper.feed(content)
            clean = stripper.get_output()
            self.send_html(clean)
        except Exception as e:
            self.send_html(f"<html><body style='background:#0a0a0a;color:#e5e5e5;padding:20px'><h2>Error</h2><p>{e}</p></body></html>")

    def handle_summarize(self):
        url = self.path.split("url=", 1)[1]
        url = urllib.request.unquote(url)
        try:
            # Fetch page text
            req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0 (compatible; VulaOS)"})
            resp = urllib.request.urlopen(req, timeout=10)
            html = resp.read().decode("utf-8", errors="replace")
            text = re.sub(r"<[^>]+>", " ", html)
            text = re.sub(r"\s+", " ", text).strip()[:3000]

            # Ask AI to summarize
            ai_req = urllib.request.Request(
                VULOS_API + "/api/ai/chat",
                data=json.dumps({"messages": [{"role": "user", "content": f"Summarize this web page in 3 bullet points:\n\n{text}"}], "stream": False}).encode(),
                headers={"Content-Type": "application/json"},
            )
            ai_resp = urllib.request.urlopen(ai_req, timeout=30)
            ai_data = json.loads(ai_resp.read())
            summary = ai_data.get("content", "No summary available.")

            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.end_headers()
            self.wfile.write(json.dumps({"summary": summary}).encode())
        except Exception as e:
            self.send_response(500)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"error": str(e)}).encode())

    def send_html(self, html):
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.end_headers()
        self.wfile.write(html.encode())

    def log_message(self, format, *args):
        pass  # quiet

print(f"[browser] Smart Browser on port {PORT}")
http.server.HTTPServer(("0.0.0.0", PORT), BrowserHandler).serve_forever()

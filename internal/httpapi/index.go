package httpapi

import (
	"net/http"
)

const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<title>历史广播节目单时段归属复核</title>
<style>
body{font-family:system-ui,sans-serif;margin:2rem;background:#0f1419;color:#e6edf3}
h1{color:#58a6ff}section{margin:1.5rem 0;padding:1rem;border:1px solid #30363d;border-radius:8px}
.timeline{display:flex;gap:4px;flex-wrap:wrap;margin-top:1rem}
.slot{padding:8px 12px;border-radius:4px;background:#21262d;border:1px solid #388bfd33;min-width:120px}
.slot.conflict{background:#3d1f1f;border-color:#f85149}
label{display:block;margin:.5rem 0}
input,button{padding:.4rem .8rem;margin-right:.5rem}
button{cursor:pointer;background:#238636;color:#fff;border:none;border-radius:4px}
#error{color:#f85149}
</style>
</head>
<body>
<h1>历史广播节目单时段归属复核台</h1>
<p>选择批次查看校正后时间轴、归属与来源引用（数据来自 /api）。</p>
<label>批次 ID <input id="batchId" type="number" value="1"></label>
<button onclick="loadAll()">加载</button>
<div id="error"></div>
<section><h2>批次</h2><pre id="batch"></pre></section>
<section><h2>时间轴（归属 UTC 窗口）</h2><div id="timeline" class="timeline"></div></section>
<section><h2>来源引用</h2><pre id="citations"></pre></section>
<section><h2>冲突</h2><pre id="conflicts"></pre></section>
<script>
async function api(path){const r=await fetch(path);if(!r.ok)throw new Error(await r.text());return r.json()}
function msToTime(ms){const h=Math.floor(ms/3600000)%24,m=Math.floor((ms%3600000)/60000);return String(h).padStart(2,'0')+':'+String(m).padStart(2,'0')}
async function loadAll(){
  const err=document.getElementById('error');err.textContent='';
  const id=document.getElementById('batchId').value;
  try{
    const batch=await api('/api/batches/'+id);
    document.getElementById('batch').textContent=JSON.stringify(batch,null,2);
    const attrs=await api('/api/batches/'+id+'/attributions');
    const tl=document.getElementById('timeline');tl.innerHTML='';
    attrs.forEach(a=>{
      const d=document.createElement('div');
      d.className='slot'+(a.status.includes('conflict')?' conflict':'');
      d.textContent='条目#'+a.entry_id+' '+msToTime(a.utc_start_ms)+'-'+msToTime(a.utc_end_ms)+' ['+a.status+']';
      tl.appendChild(d);
    });
    const cites=await api('/api/batches/'+id+'/citations');
    document.getElementById('citations').textContent=JSON.stringify(cites,null,2);
    const conflicts=await api('/api/batches/'+id+'/conflicts');
    document.getElementById('conflicts').textContent=JSON.stringify(conflicts,null,2);
  }catch(e){err.textContent=e.message}
}
</script>
</body>
</html>`

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

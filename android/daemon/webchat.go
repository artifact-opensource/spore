package daemon

// Embedded webchat HTML — serves a full chat interface at /
const webchatHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no">
<title>Spore</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
:root{--bg:#0a0a0f;--surface:#12121a;--border:#1e1e2e;--text:#e0e0e0;--dim:#666;--accent:#8b5cf6;--accent2:#a78bfa;--user-bg:#1a1a2e;--bot-bg:#0f0f1a;--input-bg:#16161f;--glow:rgba(139,92,246,0.15);--sidebar-w:260px;--danger:#ef4444}
html,body{height:100%;overflow:hidden;font-family:-apple-system,BlinkMacSystemFont,'SF Pro','Inter',sans-serif;background:var(--bg);color:var(--text)}
.app{display:flex;height:100%}

/* Sidebar */
.sidebar{width:var(--sidebar-w);background:var(--surface);border-right:1px solid var(--border);display:flex;flex-direction:column;transition:transform .2s ease;z-index:20;flex-shrink:0}
.sidebar.hidden{transform:translateX(calc(var(--sidebar-w) * -1));position:absolute;height:100%}
.sidebar-head{padding:12px;border-bottom:1px solid var(--border);display:flex;align-items:center;gap:8px}
.sidebar-head h2{font-size:13px;font-weight:600;color:var(--dim);text-transform:uppercase;letter-spacing:.05em;flex:1}
.btn-new{background:var(--accent);color:#fff;border:none;border-radius:8px;padding:6px 12px;font-size:12px;font-weight:600;cursor:pointer;transition:all .15s;-webkit-tap-highlight-color:transparent}
.btn-new:hover{background:var(--accent2)}
.btn-new:active{transform:scale(.95)}
.session-list{flex:1;overflow-y:auto;padding:6px}
.session-item{padding:10px 12px;border-radius:10px;cursor:pointer;margin-bottom:2px;display:flex;align-items:center;gap:8px;transition:background .1s;position:relative}
.session-item:hover{background:rgba(139,92,246,0.08)}
.session-item.active{background:rgba(139,92,246,0.15);border:1px solid rgba(139,92,246,0.2)}
.session-item .title{flex:1;font-size:13px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.session-item .meta{font-size:10px;color:var(--dim)}
.session-item .del{opacity:0;color:var(--danger);font-size:16px;padding:2px 4px;cursor:pointer;transition:opacity .1s}
.session-item:hover .del{opacity:.6}
.session-item .del:hover{opacity:1}

/* Main chat area */
.main{flex:1;display:flex;flex-direction:column;min-width:0}
.header{padding:12px 16px;border-bottom:1px solid var(--border);display:flex;align-items:center;gap:12px;backdrop-filter:blur(20px);background:rgba(10,10,15,0.9);z-index:10}
.menu-btn{background:none;border:none;color:var(--dim);font-size:20px;cursor:pointer;padding:4px 8px;border-radius:6px;display:none;-webkit-tap-highlight-color:transparent}
.menu-btn:hover{background:rgba(255,255,255,0.05)}
.header .logo{width:32px;height:32px;border-radius:50%;background:linear-gradient(135deg,var(--accent),#6d28d9);display:flex;align-items:center;justify-content:center;font-size:14px;font-weight:700;color:#fff;box-shadow:0 0 20px var(--glow)}
.header h1{font-size:16px;font-weight:600;letter-spacing:-0.02em;flex:1;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.header .status{font-size:11px;color:var(--dim);display:flex;align-items:center;gap:6px;flex-shrink:0}
.header .dot{width:6px;height:6px;border-radius:50%;background:#22c55e;box-shadow:0 0 8px rgba(34,197,94,0.5)}
.messages{flex:1;overflow-y:auto;padding:16px;display:flex;flex-direction:column;gap:12px;scroll-behavior:smooth}
.msg{max-width:85%;padding:10px 14px;border-radius:16px;font-size:14px;line-height:1.55;word-wrap:break-word;animation:fadeIn .2s ease}
.msg.user{align-self:flex-end;background:linear-gradient(135deg,#2d1b69,#1a1145);border:1px solid rgba(139,92,246,0.2);border-bottom-right-radius:4px;color:#e0d4ff}
.msg.bot{align-self:flex-start;background:var(--bot-bg);border:1px solid var(--border);border-bottom-left-radius:4px}
.msg.system{align-self:center;font-size:11px;color:var(--dim);padding:4px 12px;background:transparent;max-width:100%}
.msg pre{background:rgba(0,0,0,0.4);padding:10px 12px;border-radius:8px;overflow-x:auto;font-size:12px;margin:8px 0;font-family:'SF Mono','Fira Code',monospace;border:1px solid var(--border)}
.msg code{font-family:'SF Mono','Fira Code',monospace;font-size:12px;background:rgba(0,0,0,0.3);padding:1px 5px;border-radius:4px}
.msg pre code{background:none;padding:0}
.typing{align-self:flex-start;padding:10px 14px;font-size:13px;color:var(--dim);display:none}
.typing .dots{display:inline-flex;gap:3px}
.typing .dots span{width:5px;height:5px;border-radius:50%;background:var(--dim);animation:bounce .6s infinite alternate}
.typing .dots span:nth-child(2){animation-delay:.15s}
.typing .dots span:nth-child(3){animation-delay:.3s}
.input-bar{padding:12px 16px;border-top:1px solid var(--border);background:rgba(10,10,15,0.95);backdrop-filter:blur(20px)}
.input-wrap{display:flex;gap:8px;align-items:flex-end}
textarea{flex:1;background:var(--input-bg);border:1px solid var(--border);border-radius:20px;padding:10px 16px;color:var(--text);font-size:14px;font-family:inherit;resize:none;outline:none;min-height:40px;max-height:120px;line-height:1.4;transition:border .15s}
textarea:focus{border-color:var(--accent)}
textarea::placeholder{color:var(--dim)}
.send{width:48px;height:48px;border-radius:50%;border:none;background:var(--accent);color:#fff;cursor:pointer;display:flex;align-items:center;justify-content:center;transition:all .15s;flex-shrink:0;-webkit-tap-highlight-color:transparent;touch-action:manipulation}
.send:hover{background:var(--accent2);transform:scale(1.05)}
.send:active{transform:scale(.95);background:var(--accent2)}
.send:disabled{opacity:.3;cursor:default;transform:none}
.send svg{width:20px;height:20px}
.overlay{display:none;position:fixed;inset:0;background:rgba(0,0,0,0.5);z-index:15}
@keyframes fadeIn{from{opacity:0;transform:translateY(6px)}to{opacity:1;transform:translateY(0)}}
@keyframes bounce{to{transform:translateY(-4px);opacity:.4}}
.messages::-webkit-scrollbar{width:4px}
.messages::-webkit-scrollbar-track{background:transparent}
.messages::-webkit-scrollbar-thumb{background:var(--border);border-radius:4px}
.session-list::-webkit-scrollbar{width:3px}
.session-list::-webkit-scrollbar-thumb{background:var(--border);border-radius:3px}
@media(max-width:600px){
  .sidebar{position:absolute;height:100%}
  .sidebar.hidden{transform:translateX(calc(var(--sidebar-w) * -1))}
  .menu-btn{display:block}
  .msg{max-width:92%}
  .messages{padding:12px 10px}
  .overlay.show{display:block}
}
@media(min-width:601px){
  .sidebar.hidden{transform:translateX(0);position:relative}
}
</style>
</head>
<body>
<div class="app">
  <div class="sidebar" id="sidebar">
    <div class="sidebar-head">
      <h2>Sessions</h2>
      <button class="btn-new" onclick="newSession()">+ New</button>
    </div>
    <div class="session-list" id="sessionList"></div>
  </div>
  <div class="overlay" id="overlay" onclick="toggleSidebar()"></div>
  <div class="main">
    <div class="header">
      <button class="menu-btn" id="menuBtn" onclick="toggleSidebar()">☰</button>
      <div class="logo">S</div>
      <h1 id="chatTitle">Spore</h1>
      <div class="status"><span class="dot" id="dot"></span><span id="statusText">connected</span></div>
    </div>
    <div class="messages" id="msgs">
      <div class="msg system">connected to spore — type anything to begin</div>
    </div>
    <div class="typing" id="typing"><div class="dots"><span></span><span></span><span></span></div></div>
    <div class="input-bar">
      <div class="input-wrap">
        <textarea id="input" rows="1" placeholder="say something..." autofocus></textarea>
        <button class="send" id="sendBtn" title="Send">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
        </button>
      </div>
    </div>
  </div>
</div>
<script>
const msgs=document.getElementById('msgs'),input=document.getElementById('input'),sendBtn=document.getElementById('sendBtn'),typing=document.getElementById('typing'),dot=document.getElementById('dot'),statusText=document.getElementById('statusText'),sidebar=document.getElementById('sidebar'),sessionList=document.getElementById('sessionList'),overlay=document.getElementById('overlay'),chatTitle=document.getElementById('chatTitle');
let busy=false,activeSessionId=null;

// --- Session management ---

async function loadSessions(){
  try{
    const r=await fetch('/api/sessions');
    const d=await r.json();
    activeSessionId=d.active||null;
    renderSessions(d.sessions||[]);
    return d;
  }catch(e){console.error('sessions:',e);return {}}
}

function renderSessions(list){
  sessionList.innerHTML='';
  if(list.length===0){
    sessionList.innerHTML='<div style="padding:20px;text-align:center;color:var(--dim);font-size:12px">No sessions yet</div>';
    return;
  }
  list.forEach(s=>{
    const el=document.createElement('div');
    el.className='session-item'+(s.id===activeSessionId?' active':'');
    const ago=timeAgo(new Date(s.updated));
    el.innerHTML='<div style="flex:1;min-width:0"><div class="title">'+esc(s.title)+'</div><div class="meta">'+s.message_count+' msgs · '+ago+'</div></div><span class="del" onclick="event.stopPropagation();delSession(\''+s.id+'\')">×</span>';
    el.onclick=()=>switchSession(s.id);
    sessionList.appendChild(el);
  });
}

async function newSession(){
  try{
    const r=await fetch('/api/sessions',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({})});
    const d=await r.json();
    activeSessionId=d.id;
    msgs.innerHTML='<div class="msg system">new session started</div>';
    chatTitle.textContent='Spore';
    loadSessions();
    closeSidebarMobile();
  }catch(e){console.error('new session:',e)}
}

async function switchSession(id){
  try{
    const r=await fetch('/api/sessions/'+id);
    const d=await r.json();
    activeSessionId=d.id;
    chatTitle.textContent=d.title||'Spore';
    msgs.innerHTML='';
    (d.messages||[]).forEach(m=>{
      if(m.role==='user')addMsg(m.content,'user');
      else if(m.role==='assistant')addMsg(m.content,'bot');
    });
    if(msgs.children.length===0){
      msgs.innerHTML='<div class="msg system">session loaded — continue the conversation</div>';
    }
    msgs.scrollTop=msgs.scrollHeight;
    loadSessions();
    closeSidebarMobile();
  }catch(e){console.error('switch:',e)}
}

async function delSession(id){
  if(!confirm('Delete this session?'))return;
  try{
    await fetch('/api/sessions/'+id,{method:'DELETE'});
    if(id===activeSessionId){
      activeSessionId=null;
      msgs.innerHTML='<div class="msg system">session deleted — start a new one</div>';
      chatTitle.textContent='Spore';
    }
    loadSessions();
  }catch(e){console.error('delete:',e)}
}

function toggleSidebar(){
  sidebar.classList.toggle('hidden');
  overlay.classList.toggle('show');
}
function closeSidebarMobile(){
  if(window.innerWidth<=600){sidebar.classList.add('hidden');overlay.classList.remove('show')}
}

function timeAgo(date){
  const s=Math.floor((Date.now()-date.getTime())/1000);
  if(s<60)return 'now';
  if(s<3600)return Math.floor(s/60)+'m';
  if(s<86400)return Math.floor(s/3600)+'h';
  return Math.floor(s/86400)+'d';
}

// --- Chat ---

function addMsg(text,cls){
  const d=document.createElement('div');
  d.className='msg '+cls;
  if(cls==='bot'){d.innerHTML=renderMd(text)}else{d.textContent=text}
  msgs.appendChild(d);
  msgs.scrollTop=msgs.scrollHeight;
  return d;
}

function renderMd(t){
  t=t.replace(/` + "```" + `([\\s\\S]*?)` + "```" + `/g,(_,c)=>'<pre><code>'+esc(c.replace(/^\\w*\\n/,''))+'</code></pre>');
  t=t.replace(/` + "`" + `([^` + "`" + `]+)` + "`" + `/g,'<code>$1</code>');
  t=t.replace(/\*\*(.+?)\*\*/g,'<strong>$1</strong>');
  t=t.replace(/\*(.+?)\*/g,'<em>$1</em>');
  t=t.replace(/\[([^\]]+)\]\(([^)]+)\)/g,'<a href="$2" target="_blank" style="color:var(--accent2)">$1</a>');
  t=t.replace(/\n/g,'<br>');
  return t;
}
function esc(s){return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')}

async function doSend(){
  const text=input.value.trim();
  if(!text||busy)return;
  busy=true;
  input.value='';
  autoResize();
  sendBtn.disabled=true;
  addMsg(text,'user');
  typing.style.display='block';
  msgs.scrollTop=msgs.scrollHeight;
  try{
    const r=await fetch('/run',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({prompt:text})});
    const d=await r.json();
    typing.style.display='none';
    if(d.error){addMsg('error: '+d.error,'system')}
    else{
      addMsg(d.result||'(no response)','bot');
      // Refresh session list (title may have auto-updated)
      loadSessions();
    }
  }catch(e){
    typing.style.display='none';
    addMsg('connection error: '+e.message,'system');
    dot.style.background='#ef4444';
    statusText.textContent='disconnected';
  }finally{
    busy=false;
    sendBtn.disabled=false;
    input.focus();
  }
}

sendBtn.onclick=function(){doSend()};
sendBtn.addEventListener('touchend',function(e){e.preventDefault();doSend()},{passive:false});
input.onkeydown=e=>{if(e.key==='Enter'&&!e.shiftKey){e.preventDefault();doSend()}};

function autoResize(){input.style.height='auto';input.style.height=Math.min(input.scrollHeight,120)+'px'}
input.oninput=autoResize;

// Health check every 15s
setInterval(async()=>{
  try{const r=await fetch('/health');if(r.ok){dot.style.background='#22c55e';statusText.textContent='connected'}}
  catch(e){dot.style.background='#ef4444';statusText.textContent='disconnected'}
},15000);

// Init: load sessions, auto-resume active, hide sidebar on mobile
if(window.innerWidth<=600)sidebar.classList.add('hidden');
loadSessions().then(()=>{
  if(activeSessionId){
    // Auto-load the active session's messages without requiring a click
    fetch('/api/sessions/'+activeSessionId).then(r=>r.json()).then(d=>{
      if(d.id){
        chatTitle.textContent=d.title||'Spore';
        msgs.innerHTML='';
        (d.messages||[]).forEach(m=>{
          if(m.role==='user')addMsg(m.content,'user');
          else if(m.role==='assistant')addMsg(m.content,'bot');
        });
        if(msgs.children.length===0){
          msgs.innerHTML='<div class="msg system">session resumed — continue the conversation</div>';
        }
        msgs.scrollTop=msgs.scrollHeight;
      }
    }).catch(()=>{});
  }
});
</script>
</body>
</html>` + "`"

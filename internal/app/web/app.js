'use strict';
const $ = selector => document.querySelector(selector);
const escapeText = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
let room = null, info = null, shortlist = null, editing = null, events = null, requestID = 0, busy = false, queue = Promise.resolve();
let likes = new Set(), vetoes = new Set(), traces = [];
const key = () => crypto.randomUUID();
function toast(message) { $('#toast').textContent=message;$('#toast').classList.add('visible');clearTimeout(toast.timer);toast.timer=setTimeout(()=>$('#toast').classList.remove('visible'),4500); }
function bubble(text,user=false){const div=document.createElement('div');div.className='bubble'+(user?' user':'');div.textContent=text;$('#conversation').append(div);while($('#conversation').children.length>8)$('#conversation').firstElementChild.remove();}
async function rpc(method,params,notification=false){
  const body={jsonrpc:'2.0',method,params};if(!notification)body.id=++requestID;
  const response=await fetch('/mcp',{method:'POST',headers:{'Content-Type':'application/json','Accept':'application/json, text/event-stream','MCP-Protocol-Version':'2025-11-25'},body:JSON.stringify(body)});
  if(!response.ok)throw new Error(`MCP HTTP ${response.status}`);
  const text=await response.text();if(!text)return null;const result=JSON.parse(text);if(result.error)throw new Error(result.error.message);return result.result;
}
async function call(name,args={}){
  const started=performance.now();let ok=false;
  try {const result=await rpc('tools/call',{name,arguments:args});if(result.isError)throw new Error((result.content||[]).map(c=>c.text||'').join('\n'));ok=true;return result.structuredContent ?? JSON.parse(result.content.find(c=>c.type==='text').text);}
  finally{traces.unshift(`${new Date().toLocaleTimeString()}  ${name}  ${ok?'OK':'ERROR'}  ${(performance.now()-started).toFixed(0)} ms`);traces=traces.slice(0,30);$('#tool-trace').textContent=traces.join('\n');}
}
function run(action){queue=queue.then(async()=>{busy=true;document.body.setAttribute('aria-busy','true');try{await action();}catch(error){toast(error.message);bubble(error.message);if(room){try{room=await call('get_room',{room_id:room.id});renderRoom();}catch{}}}finally{busy=false;document.body.setAttribute('aria-busy','false');}});return queue;}
function renderRoom(){
  $('#room-title').textContent=room?.name||'Tonight’s company';$('#member-count').textContent=room?.members.length||0;
  $('#add-member').disabled=!room;$('#share-room').disabled=!room;$('#exclude-seen').disabled=!room;$('#exclude-seen').checked=room?.exclude_seen??true;
  $('#members').innerHTML=room?.members.length?room.members.map(m=>`<article class="member"><div class="member-top"><span class="avatar">${escapeText(m.name[0]?.toUpperCase())}</span><div><div class="member-name">${escapeText(m.name)}</div><div class="member-history">${m.profile?'Research profile '+m.profile:'Fresh preferences'}</div></div><button class="member-edit" data-member="${m.id}" aria-label="Edit ${escapeText(m.name)}">↗</button></div><div class="member-tags">${(m.likes||[]).map(g=>`<span class="tag">${escapeText(g)}</span>`).join('')}${(m.vetoes||[]).map(g=>`<span class="tag veto">No ${escapeText(g)}</span>`).join('')}${!m.likes?.length&&!m.vetoes?.length?'<span class="tag">Open to a surprise</span>':''}</div></article>`).join(''):'<p class="muted">Start a room and invite a few different tastes.</p>';
  document.querySelectorAll('.member-edit').forEach(button=>button.onclick=()=>openMember(room.members.find(m=>m.id===button.dataset.member)));
  const saved=room?.choices.at(-1);const changed=saved&&room.revision>saved.from_revision+1;
  $('#receipts').innerHTML=saved?`<div class="eyebrow">YOUR SAVED CHOICE</div><div class="receipt">✓ ${escapeText(saved.title)}<small>Saved from room revision ${saved.from_revision} · survives a restart</small>${changed?'<small class="error">Preferences changed since this choice. Review the new shortlist before watching.</small>':''}</div>`:'';
}
function renderPicks(){
  if(!shortlist)return;
  $('#shortlist-note').textContent=shortlist.message;
  $('#picks').innerHTML=shortlist.candidates.length?shortlist.candidates.map((c,i)=>`<article class="movie"><div class="poster"><div class="number">COMMON GROUND / ${String(i+1).padStart(2,'0')}</div><div class="poster-title">${escapeText(c.movie.title)}</div></div><div class="movie-body"><div class="movie-genres">${c.movie.genres.map(escapeText).join(' · ')}</div>${c.matches.map(m=>`<div class="match" title="${escapeText(m.reason)}"><span>${escapeText(m.name)}</span><meter min="1" max="5" value="${m.estimate}"></meter><span class="value">${m.estimate.toFixed(1)}</span></div>`).join('')}<div class="movie-floor">Nobody below <strong>${c.minimum.toFixed(1)}/5</strong> <span>estimated</span></div><button class="choose" data-movie="${c.movie.id}">Choose this film ↗</button><details class="why"><summary>Why it fits the room</summary><p>${escapeText(c.explanation)}</p>${c.matches.map(m=>`<p>${escapeText(m.name)}: ${escapeText(m.reason)}</p>`).join('')}</details></div></article>`).join(''):`<div class="empty-state"><div class="orbit">◌</div><h3>No common ground yet.</h3><p>${escapeText(shortlist.message)}</p></div>`;
  document.querySelectorAll('.choose').forEach(button=>button.onclick=()=>run(async()=>{room=await call('save_choice',{room_id:room.id,revision:shortlist.room_revision,operation_key:key(),movie_id:Number(button.dataset.movie)});renderRoom();const title=room.choices.at(-1).title;bubble(`Saved ${title}. The choice and its explanation are kept with this room.`);toast('Your movie night is decided.');await refreshShortlist();}));
  $('#provenance').textContent=`${shortlist.considered.toLocaleString()} trained catalog films considered · ${shortlist.vetoed.toLocaleString()} blocked by vetoes · ${shortlist.seen.toLocaleString()} already rated · revision ${shortlist.room_revision}. Match bars are model estimates on a 1–5 scale, not measured satisfaction. No streaming availability is claimed.`;
}
async function refreshShortlist(){if(!room)return;const result=await call('shortlist',{room_id:room.id,limit:6,mode:$('#ranking').value});if(result.room_revision!==room.revision){room=await call('get_room',{room_id:room.id});renderRoom();if(result.room_revision!==room.revision)return;}shortlist=result;renderPicks();}
function attachEvents(){if(events)events.close();if(!room)return;const id=room.id;events=new EventSource('/events/'+encodeURIComponent(id));events.onmessage=event=>{if(!room||room.id!==id||Number(event.data)<=room.revision)return;run(async()=>{room=await call('get_room',{room_id:id});renderRoom();await refreshShortlist();});};}
async function activate(r){room=r;localStorage.setItem('commonreel-room',room.id);history.replaceState(null,'','#room='+room.id);renderRoom();attachEvents();await refreshShortlist();}
async function newRoom(name='Friday movie night'){await activate(await call('create_room',{name}));}
async function memberMutation(member){room=await call('set_member',{room_id:room.id,revision:room.revision,operation_key:key(),member_id:member.id||'',name:member.name,profile:member.profile||0,likes:member.likes||[],vetoes:member.vetoes||[]});renderRoom();}
async function demo(){
  await newRoom('Friday movie night');const profiles=info.profiles.map(p=>p.id);
  for(const [i,member] of [{name:'Alex',likes:['Sci-Fi','Adventure'],vetoes:[]},{name:'Jamie',likes:['Comedy','Romance'],vetoes:['Horror']},{name:'Sam',likes:['Drama'],vetoes:[]}].entries()){await memberMutation({...member,profile:profiles[i]||0});}
  await refreshShortlist();bubble('Three different tastes, one shared shortlist. Jamie’s horror veto is already respected. Compare group average with balance, or ask why the first pick works.');
}
function genreButtons(){
  for(const [selector,set,other] of [['#like-genres',likes,vetoes],['#veto-genres',vetoes,likes]]){
    $(selector).innerHTML=info.genres.map(g=>`<button type="button" data-genre="${escapeText(g)}" class="${set.has(g)?'selected':''}">${escapeText(g)}</button>`).join('');
    $(selector).querySelectorAll('button').forEach(button=>button.onclick=()=>{const g=button.dataset.genre;if(set.has(g))set.delete(g);else{if(set.size>=8){toast('Choose at most eight genres.');return;}set.add(g);other.delete(g);}genreButtons();});
  }
}
function openMember(member=null){if(!room||busy)return;editing=member;likes=new Set(member?.likes||[]);vetoes=new Set(member?.vetoes||[]);$('#member-name').value=member?.name||'';if(member?.profile&&![...$('#member-profile').options].some(o=>Number(o.value)===member.profile)){const option=document.createElement('option');option.value=member.profile;option.textContent=`Existing research profile ${member.profile}`;$('#member-profile').append(option);}$('#member-profile').value=member?.profile||0;$('#member-error').textContent='';genreButtons();$('#member-dialog').showModal();$('#member-name').focus();}
async function command(text){
  bubble(text,true);if(!room){bubble('Start a room first, then I can help everyone find a shared choice.');return;}
  const q=text.toLowerCase().trim();
  if(/why|解释|为什么/.test(q)){await refreshShortlist();const first=shortlist.candidates[0];bubble(first?`${first.movie.title}: ${first.explanation} ${first.matches.map(m=>`${m.name} ${m.estimate.toFixed(1)}/5 (${m.reason})`).join('; ')}.`:shortlist.message);return;}
  if(/no horror|不要恐怖|不看恐怖/.test(q)){if(!room.members.length){bubble('Add a person first so the veto has an owner.');return;}const first=room.members[0];await memberMutation({...first,likes:first.likes.filter(g=>g!=='Horror'),vetoes:[...new Set([...first.vetoes,'Horror'])]});await refreshShortlist();bubble(`Added Horror to ${first.name}’s vetoes and refreshed the room. Everyone else’s preferences stayed in place.`);return;}
  if(/allow seen|rewatch|重看/.test(q)){room=await call('set_policy',{room_id:room.id,revision:room.revision,operation_key:key(),exclude_seen:false});renderRoom();await refreshShortlist();bubble('Previously rated films are allowed now. Every genre veto still applies.');return;}
  if(/recommend|common ground|find|推荐|选片/.test(q)){await refreshShortlist();const c=shortlist.candidates[0];bubble(c?`Start with ${c.movie.title}. The lowest estimated match is ${c.minimum.toFixed(1)}/5 and the average is ${c.average.toFixed(1)}/5. Open “Why it fits the room” to inspect the tradeoff.`:shortlist.message);return;}
  bubble('This is a guided scripted simulation. I understand “recommend”, “no horror”, “allow seen”, and “why first”. Use the member editor for other preferences; an unsupported request never changes the room.');
}
$('#start-demo').onclick=()=>run(demo);
$('#new-room').onclick=()=>run(async()=>{await newRoom();bubble('A fresh room is ready. Add a person and choose a few preferences.');});
$('#share-room').onclick=async()=>{if(!room)return;try{await navigator.clipboard.writeText(location.origin+'/#room='+room.id);toast('Room link copied. Anyone with this local link can edit the room.');}catch{toast('Use the room URL in your address bar to invite another local tab.');}};
$('#add-member').onclick=()=>openMember();$('#close-dialog').onclick=()=>$('#member-dialog').close();
$('#member-form').onsubmit=event=>{event.preventDefault();const member={id:editing?.id||'',name:$('#member-name').value.trim(),profile:Number($('#member-profile').value),likes:[...likes],vetoes:[...vetoes]};run(async()=>{await memberMutation(member);$('#member-dialog').close();await refreshShortlist();});};
$('#exclude-seen').onchange=()=>run(async()=>{room=await call('set_policy',{room_id:room.id,revision:room.revision,operation_key:key(),exclude_seen:$('#exclude-seen').checked});renderRoom();await refreshShortlist();});
$('#ranking').onchange=()=>run(refreshShortlist);
$('#chat-form').onsubmit=event=>{event.preventDefault();const text=$('#chat-input').value.trim();if(!text)return;$('#chat-input').value='';run(()=>command(text));};
document.querySelectorAll('[data-command]').forEach(button=>button.onclick=()=>run(()=>command(button.dataset.command)));
run(async()=>{
  const initialized=await rpc('initialize',{protocolVersion:'2025-11-25',capabilities:{},clientInfo:{name:'CommonReel Alexa+ web simulation',version:'0.1.0'}});
  if(initialized.protocolVersion!=='2025-11-25')throw new Error('Unexpected negotiated MCP protocol version');await rpc('notifications/initialized',{},true);
  info=await call('catalog_info');$('#dataset-status').textContent=info.data.synthetic?'INVENTED DEMO DATA · no real-data claim':`MovieLens 1M · ${info.data.ratings.toLocaleString()} research ratings`;
  for(const p of info.profiles){const option=document.createElement('option');option.value=p.id;option.textContent=`Research profile ${p.id} · ${p.ratings} training ratings`;$('#member-profile').append(option);}
  const fromURL=new URLSearchParams(location.hash.slice(1)).get('room');const id=fromURL||localStorage.getItem('commonreel-room');
  if(id){try{await activate(await call('get_room',{room_id:id}));}catch{localStorage.removeItem('commonreel-room');toast('That room is unavailable. Start a new one.');}}
});

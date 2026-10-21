const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const source = fs.readFileSync(path.join(__dirname, 'web/app.js'), 'utf8');
const start = source.indexOf('async function hydrateProposal(');
const end = source.indexOf('\nfunction render()', start);
const hydrateSource = source.slice(start, end);

function setup(request) {
  const state = {
    t: (source, values = {}) => source.replace(/\{(\w+)\}/g, (_, key) => values[key]),
    request,
    room: { active_candidate: { film_id: 'film:101' } },
    filmMetadata: new Map(),
    metadataRequests: new Set(),
    renders: 0,
    errors: [],
    render() { state.renders++; },
    notice(message, error) { state.errors.push({ message, error }); },
  };
  vm.runInNewContext(hydrateSource, state);
  return state;
}

test('a reopened proposal loads exact film IDs without a title search', async () => {
  const film = { film_id: 'film:101', title: 'Actual title', year: 2001, runtime_minutes: 95 };
  const state = setup(async (url, method, body, capability) => {
    assert.equal(url, '/catalog?ids=film%3A101');
    assert.equal(method, 'GET');
    assert.equal(capability, '');
    return { films: [film] };
  });
  await state.hydrateProposal({ film_id: film.film_id, title: '' });
  assert.equal(state.filmMetadata.get(film.film_id), film);
  assert.equal(state.renders, 1);
  assert.equal(state.metadataRequests.size, 0);
});

function languageEnvironment(saved, browserLanguage = 'en-US') {
  const storage = new Map(saved ? [['commonreel.language', saved]] : []);
  const select = { value: '' }, events = [];
  const document = {
    documentElement: { lang: '' }, title: '',
    querySelector(selector) { return selector === '#language' ? select : null; },
    querySelectorAll() { return [select]; },
    createTextNode(value) { return { textContent: value, isConnected: true }; },
    dispatchEvent(event) { events.push(event.type); },
  };
  const context = {
    navigator: { language: browserLanguage }, document, Event,
    localStorage: { getItem: key => storage.get(key), setItem: (key, value) => storage.set(key, value) },
  };
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, 'web/i18n.js'), 'utf8') + '\nthis.i18n=CommonReelI18n;', context);
  return { i18n: context.i18n, document, storage, select, events };
}

test('language follows the browser initially and persists an explicit supported choice', () => {
  const env = languageEnvironment(undefined, 'zh-TW');
  env.i18n.apply();
  assert.equal(env.document.documentElement.lang, 'zh-CN');
  assert.equal(env.select.value, 'zh-CN');
  env.i18n.setLanguage('en');
  assert.equal(env.document.documentElement.lang, 'en');
  assert.equal(env.storage.get('commonreel.language'), 'en');
  assert.equal(languageEnvironment('en', 'zh-CN').i18n.language, 'en');
  env.i18n.setLanguage('unavailable');
  assert.equal(env.document.documentElement.lang, 'en');
});

test('switching updates bound labels and attributes while preserving values and record text', () => {
  const { i18n } = languageEnvironment('en');
  const label = { isConnected: true, textContent: '', setAttribute(name, value) { this[name] = value; } };
  const input = { ...label, value: 'unsaved 喜剧, Action' };
  const name = '<img onerror=alert(1)> Comedy';
  i18n.attribute(input, 'placeholder', 'Search movie title');
  const text = i18n.node('{name}’s preferences', { name });
  i18n.attribute(label, 'aria-label', 'Record accept for {name}', { name });
  i18n.setLanguage('zh-CN');
  assert.equal(text.textContent, name + ' 的偏好');
  assert.equal(input.placeholder, '搜索电影名称');
  assert.equal(input.value, 'unsaved 喜剧, Action');
  assert.equal(label['aria-label'], '记录 ' + name + ' 接受');
  i18n.setLanguage('en');
  assert.equal(text.textContent, name + '’s preferences');
  assert.equal(input.placeholder, 'Search movie title');
});

test('translated genre input keeps canonical API values and all catalog templates retain their fields', () => {
  const { i18n } = languageEnvironment('zh-CN');
  assert.equal(i18n.canonicalGenre('喜剧'), 'Comedy');
  assert.equal(i18n.canonicalGenre('恐怖'), 'Horror');
  assert.equal(i18n.canonicalGenre('unrecognized'), 'unrecognized');
  assert.equal(i18n.t('{accepted}/{total} accepted', { accepted: 1, total: 2 }), '1/2 已记录接受');
  assert.equal(i18n.t('{reply}{trace}', { reply: 'raw reply', trace: '\ntrace' }), 'raw reply\ntrace');
  for (const [source, translated] of Object.entries(i18n.messages)) {
    const fields = value => [...value.matchAll(/\{(\w+)\}/g)].map(match => match[1]).sort();
    assert.deepEqual(fields(translated), fields(source), source);
  }
});

test('every static interface phrase has a Chinese catalog entry', () => {
  const { i18n } = languageEnvironment('en');
  const html = fs.readFileSync(path.join(__dirname, 'web/index.html'), 'utf8');
  const untranslated = html.replace(/<svg[\s\S]*?<\/svg>/g, '').split(/<[^>]*>/)
    .map(text => text.trim()).filter(text => /[A-Za-z]{2}/.test(text))
    .filter(text => !['Common', 'Reel', 'CommonReel', 'English'].includes(text))
    .filter(text => !Object.hasOwn(i18n.messages, text));
  assert.deepEqual(untranslated, []);
  const catalog = JSON.parse(fs.readFileSync(path.join(__dirname, '../../assets/catalog-v2.json'), 'utf8'));
  for (const genre of new Set(catalog.flatMap(film => film.genres))) {
    assert.ok(Object.hasOwn(i18n.messages, genre), genre);
    assert.equal(i18n.canonicalGenre(i18n.messages[genre]), genre);
  }
});

test('concurrent hydration shares one request and does not redraw a different proposal', async () => {
  let resolve, calls = 0;
  const response = new Promise(done => { resolve = done; });
  const state = setup(() => { calls++; return response; });
  const active = { film_id: 'film:101' };
  const first = state.hydrateProposal(active);
  await state.hydrateProposal(active);
  state.room.active_candidate = { film_id: 'another-film' };
  resolve({ films: [{ film_id: active.film_id, title: 'Actual title' }] });
  await first;
  assert.equal(calls, 1);
  assert.equal(state.renders, 0);
  assert.equal(state.filmMetadata.size, 1);
});

test('lookup errors remain visible and release the request for a later retry', async () => {
  for (const request of [async () => { throw Error('HTTP 503'); }, async () => ({ films: [] })]) {
    const state = setup(request);
    await state.hydrateProposal({ film_id: 'film:101' });
    assert.equal(state.renders, 0);
    assert.equal(state.filmMetadata.size, 0);
    assert.equal(state.metadataRequests.size, 0);
    assert.equal(state.errors.length, 1);
    assert.equal(state.errors[0].error, true);
    assert.match(state.errors[0].message, /HTTP 503|outside the current library/);
  }
});

test('save button, response count and hint agree before, during and after saving', () => {
  const begin = source.indexOf('function syncControls(');
  const finish = source.indexOf('\nfunction setStep(', begin);
  const nodes = new Map();
  const state = {
    t: (source, values = {}) => source.replace(/\{(\w+)\}/g, (_, key) => values[key]),
    setText(node, source, values = {}) { node.textContent = this.t(source, values); },
    I18n: { prune() {} }, invalidatePreviewCards() {},
    room: {
      active_candidate: { film_id: 'film:101' }, decision_version: 4,
      members: [{ id: 'm1' }, { id: 'm2' }], acceptances: {},
    },
    finding: false, ready: () => true, document:{querySelectorAll(){return [];}},
    el: (tag, text) => ({ textContent: text }), icon: () => ({}),
    $(selector) {
      if (!nodes.has(selector)) nodes.set(selector, {
        textContent: '', busy: false,
        getAttribute() { return this.busy ? 'true' : null; },
        replaceChildren(label) { this.textContent = label.textContent; },
      });
      return nodes.get(selector);
    },
  };
  state.setText = state.setText.bind(state);
  vm.runInNewContext(source.slice(begin, finish), state);
  const save = state.$('#save-choice'), hint = state.$('#save-hint');
  state.syncControls();
  assert.equal(save.disabled, true);
  assert.match(hint.textContent, /Waiting for 2/);
  state.room.acceptances = { m1: { response: 'accept' }, m2: { response: 'accept' } };
  state.syncControls();
  assert.equal(save.disabled, false);
  assert.equal(state.$('#response-count').textContent, '2/2 accepted');
  assert.match(hint.textContent, /Ready to save/);
  save.busy = true;
  save.textContent = 'Rechecking supply…';
  state.syncControls();
  assert.equal(save.disabled, true);
  assert.equal(save.textContent, 'Rechecking supply…');
  assert.match(hint.textContent, /Checking current subscription supply/);
  save.busy = false;
  state.room.last_choice = { film_id: 'film:101', decision_version: 4 };
  state.syncControls();
  assert.equal(save.disabled, true);
  assert.equal(save.textContent, 'Choice saved');
  assert.equal(hint.textContent, 'Choice saved for these limits.');
  state.room.decision_version = 5;
  state.room.acceptances = {};
  state.syncControls();
  assert.equal(save.textContent, 'Save group-recorded choice');
  assert.match(hint.textContent, /Waiting for 2/);
});

test('runtime card uses exact tool counts, cached facts and an explicit versioned apply action', async () => {
  const begin=source.indexOf('function runtimePreviewCard('), finish=source.indexOf('\nfunction invalidatePreviewCards(',begin);
  assert.ok(begin>=0 && finish>begin,'runtime preview card implementation');
  class Node { constructor(tag,text='',className=''){this.tag=tag;this.textContent=text;this.className=className;this.children=[];this.dataset={};} append(...nodes){this.children.push(...nodes);} }
  const commands=[],nodes=[],handlers=[];
  const state={roomID:'room-a',room:{room_revision:8,policy:{provider_ids:[8]}},platforms:[[8,'Known Service']],I18n:{language:'en'},
    el(tag,text,cls,values={}){const n=new Node(tag,(text||'').replace(/\{(\w+)\}/g,(_,key)=>values[key]),cls);nodes.push(n);return n;},
    literal:(tag,text,cls)=>new Node(tag,text,cls),action:(node,fn)=>handlers.push({node,fn}),command:async(type,fields)=>commands.push({type,fields}),
    $(selector){return {open:false,close(){}};},setStep(){},setText(){},notice(){},lastList:null};
  vm.runInNewContext(source.slice(begin,finish),state);
  const p={room_revision:8,current_max_runtime_minutes:90,max_runtime_minutes:120,current_feasible_count:0,proposed_feasible_count:8,newly_feasible_count:8,proposed_unknown_count:2,preview_basis:'verified-basis',candidates:[{film:{title:'Real title',year:2020,runtime_minutes:110},availability:{fetched_at:'2026-10-21T00:00:00Z',offers:[{region:'US',monetization:'sub',provider_id:8}]}}]};
  state.runtimePreviewCard(p);
  assert.ok(nodes.some(n=>n.textContent==='0 → 8 eligible films · 8 newly eligible'));
  assert.ok(nodes.some(n=>n.textContent==='2 films have unknown metadata or supply under this limit. They are excluded.'));
  const apply=handlers.find(h=>h.node.textContent==='Apply this runtime');assert.ok(apply);await apply.fn();
  assert.equal(commands[0].type,'apply_runtime_preview');
  assert.equal(commands[0].fields.preview_basis,'verified-basis');assert.equal(commands[0].fields.expected_room_revision,8);assert.equal(commands[0].fields.max_runtime_minutes,120);
  handlers.length=0;state.runtimePreviewCard({...p,newly_feasible_count:0,candidates:[]});assert.equal(handlers.length,0,'no apply action without verified new candidates');
});

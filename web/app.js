const TOKEN = document.querySelector('meta[name="gbavm-session-token"]').content;
const BASE = '/' + TOKEN + '/api';
const $ = id => document.getElementById(id);

const DEFAULT_CLIP = Object.freeze({
  start: '0:00', end: '', speed: 1, fit: 'fit', audio: 'mix', volume: 100,
  loop: false, paletteMode: 'shared', ditherMode: 'ordered'
});
const FPS_VBLANKS = {smooth: 4, balanced: 5, classic: 6, compact: 8};
const FPS_ORDER = ['smooth', 'balanced', 'classic', 'compact'];
const ROM_LIMIT = 32 * 1024 * 1024;
const MIB = 1024 * 1024;

let state = null;
let pollTimer = null;
let uploadBusy = false;
let selectedID = '';
let editScope = 'project';
let projectDefaults = {...DEFAULT_CLIP};
let clipConfigs = {};
let playheads = {};
let lastPreviewKey = '';
let lastThumbKey = '';
let audioURL = '';
let romTitleAuto = true;
let pendingProject = null;
let draggedID = '';
let optimizerProposal = null;
let scopeInitialized = false;
let menuBackgroundID = 'ocean-wave-animated';
let customMenuTheme = null;
let activeMenuTheme = null;
let stopMenuPreview = null;

function headers(extra = {}) { return Object.assign({'X-GBA-Token': TOKEN}, extra); }
async function api(path, options = {}) {
  options.headers = headers(options.headers || {});
  const response = await fetch(BASE + path, options);
  if (!response.ok) {
    let body;
    try { body = await response.json(); } catch { body = {error: await response.text()}; }
    const error = new Error(body.error || 'Request failed');
    error.status = response.status;
    throw error;
  }
  return response.status === 204 ? null : response.json();
}
function show(id) {
  ['welcome', 'loading', 'editor'].forEach(name => $(name).classList.toggle('hidden', name !== id));
  $('topBar').classList.toggle('hidden', id !== 'editor');
}
function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}
function fmt(seconds) {
  seconds = Math.max(0, Number(seconds) || 0);
  const minutes = Math.floor(seconds / 60);
  return minutes + ':' + String(Math.floor(seconds % 60)).padStart(2, '0');
}
function precise(seconds) {
  seconds = Math.max(0, Number(seconds) || 0);
  const minutes = Math.floor(seconds / 60);
  return minutes + ':' + (seconds % 60).toFixed(2).padStart(5, '0');
}
function parseClock(value) {
  if (String(value).trim() === '') return 0;
  const parts = String(value).split(':').map(Number);
  if (parts.some(x => !Number.isFinite(x))) return NaN;
  return parts.length === 3 ? parts[0] * 3600 + parts[1] * 60 + parts[2]
    : parts.length === 2 ? parts[0] * 60 + parts[1] : parts[0];
}
function parsePartDuration(value) {
  const text = String(value ?? '').trim();
  if (text === '' || text === '0') return 0;
  const match = /^(\d+):([0-5]\d)$/.exec(text);
  if (!match) return NaN;
  const seconds = Number(match[1]) * 60 + Number(match[2]);
  return seconds <= 240 * 60 ? seconds : NaN;
}
function partDurationValue(seconds) {
  seconds = Math.max(0, Math.round(Number(seconds) || 0));
  return seconds === 0 ? '0' : Math.floor(seconds / 60) + ':' + String(seconds % 60).padStart(2, '0');
}
function clockValue(seconds) {
  seconds = Math.max(0, Number(seconds) || 0);
  const minutes = Math.floor(seconds / 60);
  const rest = seconds - minutes * 60;
  return minutes + ':' + rest.toFixed(2).padStart(5, '0');
}
function titleFromFilename(name) {
  return sanitizeMenuTitle(name.replace(/\.[^.]+$/, '')).value || 'GBA VIDEO';
}
function sanitizeMenuTitle(value) {
  const upper = String(value || '').toUpperCase();
  const invalid = /[^A-Z0-9 ]/.test(upper);
  const cleaned = upper.replace(/[^A-Z0-9 ]/g, ' ').replace(/\s+/g, ' ').slice(0, 12);
  return {value: cleaned, invalid};
}
function selectedIndex() { return state?.videos?.findIndex(v => v.id === selectedID) ?? -1; }
function selectedVideo() { return state?.videos?.find(v => v.id === selectedID) || state?.videos?.[0]; }
function cloneClip(settings) { return {...DEFAULT_CLIP, ...settings}; }
function effectiveClip(id) {
  const config = clipConfigs[id] || {title: 'GBA VIDEO', useProject: true, ...DEFAULT_CLIP};
  return config.useProject ? {...projectDefaults, title: config.title, useProject: true}
    : {...cloneClip(config), title: config.title, useProject: false};
}
function ensureClipConfigs() {
  if (!state?.videos) return;
  const valid = new Set(state.videos.map(v => v.id));
  for (const id of Object.keys(clipConfigs)) if (!valid.has(id)) delete clipConfigs[id];
  for (const video of state.videos) {
    if (!clipConfigs[video.id]) {
      clipConfigs[video.id] = {title: titleFromFilename(video.name), useProject: true, ...DEFAULT_CLIP};
    }
  }
  if (!selectedID || !valid.has(selectedID)) selectedID = state.videos[0]?.id || '';
}

async function poll() {
  try {
    state = await api('/state');
    render();
  } catch (error) {
    console.error(error);
  }
  clearTimeout(pollTimer);
  pollTimer = setTimeout(poll, 500);
}
function render() {
  if (!state) return;
  if (!state.videos?.length) {
    show('welcome');
    $('clipInfo').textContent = '';
    return;
  }
  if (state.inspectStatus === 'waiting' || state.inspectStatus === 'inspecting') {
    show('loading');
    $('loadingTitle').textContent = state.inspectStatus === 'waiting' ? 'Preparing the app…' : 'Opening videos…';
    $('loadingText').textContent = state.inspectStatus === 'waiting'
      ? (state.engineMessage || 'Preparing FFmpeg') : 'Reading duration, dimensions and audio streams.';
    $('loadingFill').style.width = (state.engineProgress || 0) + '%';
  }
  if (state.engineStatus === 'missing' || state.engineStatus === 'error') {
    show('loading');
    $('engineError').textContent = state.engineMessage;
    $('engineError').classList.remove('hidden');
    $('retryEngine').classList.remove('hidden');
  }
  if (state.inspectStatus === 'error') {
    show('loading');
    $('loadingTitle').textContent = 'Could not open videos';
    $('loadingText').textContent = state.inspectError || 'A video could not be inspected.';
  }
  if (state.inspectStatus === 'ready') {
    show('editor');
    ensureClipConfigs();
    if (pendingProject) applyPendingProject();
    renderClips();
    updateOutputModes();
    refreshScope(!scopeInitialized);
    scopeInitialized = true;
    syncTimeline(false);
    estimate();
  }
  setConvertingState(!!state.converting);
  if (state.converting) {
    $('progressWrap').classList.remove('hidden');
    $('done').classList.add('hidden');
    $('convertError').classList.add('hidden');
    $('progressText').textContent = state.progressMessage;
    $('progressPct').textContent = state.progress + '%';
    $('progressFill').style.width = state.progress + '%';
  } else if (state.result) {
    $('progressWrap').classList.remove('hidden');
    $('progressText').textContent = state.progressMessage || 'Output created successfully';
    $('progressPct').textContent = '100%';
    $('progressFill').style.width = '100%';
    $('done').classList.remove('hidden');
  } else if (state.convertError) {
    $('convertError').textContent = state.convertError;
    $('convertError').classList.remove('hidden');
  }
}
function setConvertingState(busy) {
  const ids = ['preset','start','end','speed','fps','fit','seekSeconds','paletteMode','ditherMode','compression','audio','volume','normalize','limiter','romTitle','outputMode','loop','resume','splitVideo','splitBudget','maxPartDuration','chapterAware','partTitleScreens','resumeLongSplit','useProject','menuTitle','menuBackground','customMenuBackground','menuUIColor','menuOutline','menuOutlineColor'];
  ids.forEach(id => { if ($(id)) $(id).disabled = busy || $(id).dataset.scopeDisabled === '1'; });
  ['convert','optimize','addVideos','moveUp','moveDown','saveProject','openProject'].forEach(id => { if ($(id)) $(id).disabled = busy; });
}

async function chooseNative(append) {
  try {
    const response = await api('/dialog/videos?append=' + (append ? '1' : '0'), {method: 'POST'});
    if (!response?.cancelled) {
      lastPreviewKey = '';
      lastThumbKey = '';
      await poll();
    }
  } catch (error) {
    if (error.status === 501) {
      $('picker').dataset.append = append ? '1' : '0';
      $('picker').click();
    } else alert(error.message);
  }
}
function upload(files, append) {
  if (uploadBusy || !files.length) return;
  uploadBusy = true;
  if (!append) romTitleAuto = true;
  show('loading');
  $('loadingTitle').textContent = 'Loading ' + files.length + ' video' + (files.length === 1 ? '' : 's') + '…';
  $('loadingText').textContent = 'Copying files into the portable workspace.';
  const form = new FormData();
  files.forEach(file => form.append('video', file, file.name));
  const xhr = new XMLHttpRequest();
  xhr.open('POST', BASE + '/upload?append=' + (append ? 1 : 0));
  xhr.setRequestHeader('X-GBA-Token', TOKEN);
  xhr.upload.onprogress = event => { if (event.lengthComputable) $('loadingFill').style.width = Math.round(event.loaded / event.total * 100) + '%'; };
  xhr.onload = () => {
    uploadBusy = false;
    if (xhr.status < 300) { lastPreviewKey = ''; lastThumbKey = ''; $('previewImage').removeAttribute('src'); poll(); }
    else { show('loading'); $('loadingText').textContent = 'Upload failed.'; }
  };
  xhr.onerror = () => { uploadBusy = false; show('loading'); $('loadingText').textContent = 'Upload failed.'; };
  xhr.send(form);
}
$('chooseVideos').onclick = event => { event.stopPropagation(); chooseNative(false); };
$('addVideos').onclick = () => chooseNative(true);
$('welcome').onclick = event => { if (!event.target.closest('button')) chooseNative(false); };
$('welcome').onkeydown = event => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); chooseNative(false); } };
$('picker').onchange = () => { const files = [...$('picker').files]; if (files.length) upload(files, $('picker').dataset.append === '1'); $('picker').value = ''; };
for (const name of ['dragenter','dragover']) document.addEventListener(name, event => { event.preventDefault(); $('welcome').classList.add('drag'); });
for (const name of ['dragleave','drop']) document.addEventListener(name, event => { event.preventDefault(); $('welcome').classList.remove('drag'); });
document.addEventListener('drop', event => { const files = [...(event.dataTransfer?.files || [])]; if (files.length) upload(files, !!state?.videos?.length); });

function renderClips() {
  if (!state?.videos) return;
  const html = state.videos.map((video, index) => {
    const info = video.info;
    const config = clipConfigs[video.id];
    const status = info ? info.width + '×' + info.height + ' • ' + fmt(info.duration) + (info.audioStreams ? ' • audio' : ' • silent') : (video.error || video.status);
    const relink = video.needsRelink ? '<button class="clip-action relink" data-relink="' + index + '" title="Relink source file">↻</button>' : '';
    return '<div class="clip ' + (video.id === selectedID ? 'active' : '') + '" draggable="true" data-id="' + video.id + '">' +
      '<span class="clip-handle" title="Drag to reorder">⋮⋮</span><div class="clip-info"><b>' + (index + 1) + '. ' + escapeHTML(video.name) + '</b><small>' + escapeHTML(status) + '</small>' +
      '<span class="clip-badge ' + (config?.useProject ? '' : 'custom') + '">' + (config?.useProject ? 'Project' : 'Custom') + '</span></div>' + relink +
      '<button class="clip-action remove" data-remove="' + index + '" title="Remove this video">×</button></div>';
  }).join('');
  $('clips').innerHTML = html;
  for (const element of $('clips').querySelectorAll('.clip')) {
    element.onclick = event => {
      if (event.target.closest('button')) return;
      selectedID = element.dataset.id;
      editScope = 'clip';
      refreshScope(true);
      renderClips();
      syncTimeline(true);
    };
    element.ondragstart = event => { draggedID = element.dataset.id; element.classList.add('dragging'); event.dataTransfer.effectAllowed = 'move'; };
    element.ondragend = () => { draggedID = ''; element.classList.remove('dragging'); document.querySelectorAll('.drop-before').forEach(x => x.classList.remove('drop-before')); };
    element.ondragover = event => { event.preventDefault(); if (draggedID && draggedID !== element.dataset.id) element.classList.add('drop-before'); };
    element.ondragleave = () => element.classList.remove('drop-before');
    element.ondrop = async event => {
      event.preventDefault();
      element.classList.remove('drop-before');
      if (!draggedID || draggedID === element.dataset.id) return;
      const ids = state.videos.map(v => v.id);
      const from = ids.indexOf(draggedID), to = ids.indexOf(element.dataset.id);
      ids.splice(to, 0, ids.splice(from, 1)[0]);
      await applyOrder(ids);
    };
  }
  for (const button of $('clips').querySelectorAll('[data-remove]')) button.onclick = event => { event.stopPropagation(); removeVideo(+button.dataset.remove); };
  for (const button of $('clips').querySelectorAll('[data-relink]')) button.onclick = event => { event.stopPropagation(); relinkVideo(+button.dataset.relink); };
  const video = selectedVideo();
  $('clipInfo').textContent = video?.info ? 'Previewing ' + video.name + ' • ' + video.info.fps.toFixed(2) + ' source fps' : (video?.error || '');
  if (romTitleAuto && state.videos[0]) $('romTitle').value = titleFromFilename(state.videos[0].name);
  $('moveUp').disabled = selectedIndex() <= 0 || state.converting;
  $('moveDown').disabled = selectedIndex() < 0 || selectedIndex() >= state.videos.length - 1 || state.converting;
}
async function applyOrder(ids) {
  const map = new Map(state.videos.map(v => [v.id, v]));
  state.videos = ids.map(id => map.get(id));
  renderClips();
  updateOutputModes();
  estimate();
  try { await api('/video/reorder', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({ids})}); }
  catch (error) { alert(error.message); await poll(); }
}
async function moveSelected(delta) {
  const ids = state.videos.map(v => v.id);
  const index = ids.indexOf(selectedID), target = index + delta;
  if (index < 0 || target < 0 || target >= ids.length) return;
  [ids[index], ids[target]] = [ids[target], ids[index]];
  await applyOrder(ids);
}
$('moveUp').onclick = () => moveSelected(-1);
$('moveDown').onclick = () => moveSelected(1);
async function removeVideo(index) {
  try {
    const removedID = state.videos[index]?.id;
    await api('/video/remove', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({index})});
    if (removedID) delete clipConfigs[removedID];
    if (selectedID === removedID) selectedID = state.videos[Math.max(0, index - 1)]?.id || '';
    lastPreviewKey = ''; lastThumbKey = '';
    await poll();
  } catch (error) { alert(error.message); }
}
async function relinkVideo(index) {
  try {
    const response = await api('/video/relink?index=' + index, {method:'POST'});
    if (!response.cancelled) { lastPreviewKey = ''; lastThumbKey = ''; await poll(); }
  } catch (error) { alert(error.message); }
}

function menuStyleSettings() {
  return {uiColor:$('menuUIColor')?.value || 'white', outline:!!$('menuOutline')?.checked, outlineColor:$('menuOutlineColor')?.value || 'black'};
}
function rebuildMenuTheme() {
  if (!window.MenuThemeTools || !$('menuBackground')) return;
  menuBackgroundID = $('menuBackground').value;
  if (menuBackgroundID === 'custom' && customMenuTheme) activeMenuTheme = MenuThemeTools.applyUI(customMenuTheme, menuStyleSettings());
  else if (menuBackgroundID === 'custom') activeMenuTheme = MenuThemeTools.createBuiltinTheme('classic-dark', menuStyleSettings());
  else activeMenuTheme = MenuThemeTools.createBuiltinTheme(menuBackgroundID, menuStyleSettings());
  $('customMenuBackgroundRow').classList.toggle('hidden', menuBackgroundID !== 'custom');
  $('menuOutlineColor').disabled = !$('menuOutline').checked || !!state?.converting;
}
function serializedMenuTheme() {
  rebuildMenuTheme();
  return activeMenuTheme ? MenuThemeTools.serializeTheme(activeMenuTheme) : null;
}
function menuThemeBytes() {
  if (!activeMenuTheme) return 0;
  return 64 + activeMenuTheme.palette.length + activeMenuTheme.frames.reduce((sum, frame) => sum + frame.length, 0);
}
async function loadCustomMenuBackground(file) {
  if (!file) return;
  $('menuBackgroundStatus').textContent = 'Optimizing ' + file.name + '…';
  try {
    customMenuTheme = await MenuThemeTools.decodeCustomFile(file, menuStyleSettings(), fraction => {
      $('menuBackgroundStatus').textContent = 'Optimizing ' + file.name + '… ' + Math.round(fraction * 100) + '%';
    });
    $('menuBackgroundStatus').textContent = customMenuTheme.frames.length > 1
      ? file.name + ' — ' + customMenuTheme.frames.length + ' optimized animation frames'
      : file.name + ' — optimized static background';
    rebuildMenuTheme(); estimate();
  } catch (error) {
    customMenuTheme = null;
    $('menuBackgroundStatus').textContent = 'Could not read this image or GIF: ' + error.message;
  }
}

function updateSplitControls() {
  const single = state.videos.length === 1;
  $('splitVideoRow').classList.toggle('hidden', !single);
  $('longSplitControls').classList.toggle('hidden', !single || !$('splitVideo').checked);
}
function updateOutputModes() {
  const select = $('outputMode');
  const current = select.value;
  const single = state.videos.length === 1;
  if (single) {
    select.innerHTML = '<option value="rom">Single ROM</option>';
    select.value = 'rom';
  } else {
    select.innerHTML = '<option value="playlist">One ROM — play clips in order</option><option value="menu">One ROM — clip menu</option><option value="batch">Separate ROMs in ZIP</option>';
    select.value = ['playlist','menu','batch'].includes(current) ? current : 'playlist';
  }
  $('menuSettingsSection')?.classList.toggle('hidden', !(state.videos.length > 1 && select.value === 'menu'));
  updateSplitControls();
}

function refreshScope(force) {
  $('editProjectDefaults').classList.toggle('active', editScope === 'project');
  $('editSelectedClip').classList.toggle('active', editScope === 'clip');
  $('clipIdentitySection').classList.toggle('hidden', editScope !== 'clip');
  const config = clipConfigs[selectedID];
  if (!config) return;
  const custom = editScope === 'clip' && !config.useProject;
  $('scopeBadge').textContent = editScope === 'project' ? 'Project settings' : (custom ? 'Custom settings' : 'Project settings');
  $('scopeBadge').className = 'scope-badge ' + (custom ? 'custom' : 'project');
  if (!force) return;
  if (editScope === 'clip') {
    $('useProject').checked = config.useProject;
    $('menuTitle').value = config.title;
    updateTitlePreview();
  }
  const source = editScope === 'project' ? projectDefaults : effectiveClip(selectedID);
  for (const key of ['start','end','speed','fit','audio','volume','loop','paletteMode','ditherMode']) {
    const element = $(key);
    if (!element) continue;
    if (element.type === 'checkbox') element.checked = !!source[key]; else element.value = source[key];
    const disabled = editScope === 'clip' && config.useProject;
    element.dataset.scopeDisabled = disabled ? '1' : '0';
    element.disabled = disabled || !!state?.converting;
  }
  for (const element of document.querySelectorAll('.project-control')) element.classList.toggle('scope-muted', editScope === 'clip');
  if (force) { syncTimeline(true); updatePreview(); }
}
$('editProjectDefaults').onclick = () => { editScope = 'project'; refreshScope(true); };
$('editSelectedClip').onclick = () => { editScope = 'clip'; refreshScope(true); };
$('useProject').onchange = () => {
  const config = clipConfigs[selectedID];
  if (!config) return;
  if (!config.useProject && $('useProject').checked) config.useProject = true;
  else if (config.useProject && !$('useProject').checked) Object.assign(config, cloneClip(projectDefaults), {useProject:false, title:config.title});
  refreshScope(true); renderClips(); estimate();
};
function savePerClipField(id) {
  const element = $(id);
  const value = element.type === 'checkbox' ? element.checked : (element.type === 'number' ? Number(element.value) : element.value);
  if (editScope === 'project') projectDefaults[id] = value;
  else {
    const config = clipConfigs[selectedID];
    if (!config || config.useProject) return;
    config[id] = value;
  }
  $('preset').value = 'custom';
  estimate();
  if (['start','end','fit','speed'].includes(id)) { syncTimeline(false); updatePreview(); }
}
for (const id of ['start','end','speed','fit','audio','volume','loop','paletteMode','ditherMode']) $(id).addEventListener('input', () => savePerClipField(id));
for (const id of ['fps','seekSeconds','compression','normalize','limiter','maxPartDuration','chapterAware','partTitleScreens','resumeLongSplit']) $(id).addEventListener('input', () => { $('preset').value = 'custom'; estimate(); });
$('splitVideo').addEventListener('change', () => { updateSplitControls(); $('preset').value = 'custom'; estimate(); });
$('outputMode').onchange = () => { updateOutputModes(); estimate(); };
function updateSplitBudgetLabel() { $('splitBudgetValue').textContent = $('splitBudget').value + ' MiB'; }
$('splitBudget').addEventListener('input', () => { updateSplitBudgetLabel(); $('preset').value = 'custom'; estimate(); });
for (const button of document.querySelectorAll('.split-preset')) button.onclick = () => { $('splitBudget').value = button.dataset.size; updateSplitBudgetLabel(); estimate(); };
updateSplitBudgetLabel();
$('romTitle').addEventListener('input', () => { romTitleAuto = false; });

const GLYPHS = {
  '0':0x7B6F,'1':0x2C97,'2':0x73E7,'3':0x73CF,'4':0x5BC9,'5':0x79CF,'6':0x79EF,'7':0x7292,'8':0x7BEF,'9':0x7BCF,
  A:0x2BED,B:0x6BAE,C:0x7927,D:0x6B6E,E:0x79E7,F:0x79E4,G:0x79AF,H:0x5BED,I:0x7497,J:0x124E,K:0x5D6D,L:0x4927,M:0x5FE9,N:0x5F6D,O:0x7B6F,P:0x7BE4,Q:0x7B7B,R:0x7BED,S:0x79CF,T:0x7492,U:0x5B6F,V:0x5B6A,W:0x5BFD,X:0x5AAD,Y:0x5A92,Z:0x72A7,
  ' ':0
};
function updateTitlePreview(invalid = false) {
  const config = clipConfigs[selectedID];
  if (!config) return;
  const canvas = $('titlePreview'), context = canvas.getContext('2d');
  context.imageSmoothingEnabled = false;
  context.fillStyle = '#000'; context.fillRect(0,0,canvas.width,canvas.height);
  context.fillStyle = '#ffdd00';
  const scale = 4, startX = 8, startY = 6;
  [...config.title.padEnd(12, ' ')].slice(0,12).forEach((char,index) => {
    const bits = GLYPHS[char] || 0;
    for (let row=0; row<5; row++) for (let col=0; col<3; col++) {
      const bit = 14 - (row*3+col);
      if (bits & (1<<bit)) context.fillRect(startX + index*16 + col*scale, startY + row*scale, scale, scale);
    }
  });
  $('titleCount').textContent = config.title.length + '/12';
  const duplicates = Object.entries(clipConfigs).filter(([id,c]) => id !== selectedID && c.title === config.title && config.title).length;
  const warning = invalid ? 'Unsupported characters were replaced.' : duplicates ? 'Another clip uses the same menu title.' : '';
  $('titleWarning').textContent = warning;
  $('titleWarning').className = 'field full validation ' + (warning ? 'warning' : '');
}
$('menuTitle').oninput = () => {
  const result = sanitizeMenuTitle($('menuTitle').value);
  $('menuTitle').value = result.value;
  clipConfigs[selectedID].title = result.value || 'GBA VIDEO';
  updateTitlePreview(result.invalid);
  renderClips();
};
$('resetMenuTitle').onclick = () => {
  const video = selectedVideo();
  if (!video) return;
  clipConfigs[selectedID].title = titleFromFilename(video.name);
  $('menuTitle').value = clipConfigs[selectedID].title;
  updateTitlePreview();
};

function currentTimelineSettings() { return editScope === 'project' ? projectDefaults : effectiveClip(selectedID); }
function syncTimeline(forceThumbs) {
  const video = selectedVideo();
  if (!video?.info) return;
  const settings = currentTimelineSettings();
  const duration = video.info.duration;
  const step = video.info.fps > 0 ? 1 / video.info.fps : 0.04;
  for (const id of ['timelineStart','timelineEnd','timelinePlay']) { $(id).max = duration; $(id).step = step; }
  let start = Math.min(duration, Math.max(0, parseClock(settings.start) || 0));
  let end = settings.end.trim() ? parseClock(settings.end) : duration;
  if (!Number.isFinite(end)) end = duration;
  end = Math.min(duration, Math.max(start + step, end));
  let play = playheads[selectedID];
  if (!Number.isFinite(play)) play = start;
  play = Math.min(end, Math.max(start, play));
  playheads[selectedID] = play;
  $('timelineStart').value = start; $('timelineEnd').value = end; $('timelinePlay').value = play;
  updateTimelineLabels();
  if (forceThumbs) lastThumbKey = '';
  renderTimelineThumbs();
}
function updateTimelineLabels() {
  $('timelineStartText').textContent = precise($('timelineStart').value);
  $('timelineCurrentText').textContent = precise($('timelinePlay').value);
  $('timelineEndText').textContent = precise($('timelineEnd').value);
}
function renderTimelineThumbs() {
  const video = selectedVideo();
  if (!video?.info) return;
  const index = selectedIndex();
  const fit = currentTimelineSettings().fit;
  const key = video.id + '|' + fit + '|' + video.info.duration.toFixed(3);
  if (key === lastThumbKey) return;
  lastThumbKey = key;
  const count = 6;
  $('timelineThumbs').innerHTML = Array.from({length:count}, (_,i) => {
    const time = video.info.duration * (i + .5) / count;
    return '<img alt="" src="' + BASE + '/preview?index=' + index + '&time=' + encodeURIComponent(time) + '&fit=' + fit + '&thumb=' + i + '">';
  }).join('');
}
function updatePreview(time = playheads[selectedID]) {
  const video = selectedVideo();
  if (!video?.info) return;
  const index = selectedIndex();
  const fit = currentTimelineSettings().fit;
  time = Math.min(video.info.duration, Math.max(0, Number(time) || 0));
  playheads[selectedID] = time;
  $('timelinePlay').value = time;
  updateTimelineLabels();
  const key = [video.id,time.toFixed(3),fit].join('|');
  if (key === lastPreviewKey && $('previewImage').src) return;
  lastPreviewKey = key;
  const image = $('previewImage');
  image.onerror = () => image.removeAttribute('src');
  image.src = BASE + '/preview?index=' + index + '&time=' + encodeURIComponent(time) + '&fit=' + fit + '&key=' + encodeURIComponent(key);
}
$('timelinePlay').oninput = () => updatePreview(Number($('timelinePlay').value));
$('timelineStart').oninput = () => {
  let start = Number($('timelineStart').value), end = Number($('timelineEnd').value);
  if (start >= end) { start = Math.max(0, end - Number($('timelineStart').step)); $('timelineStart').value = start; }
  $('start').value = clockValue(start); savePerClipField('start');
  if (Number($('timelinePlay').value) < start) updatePreview(start); else updateTimelineLabels();
};
$('timelineEnd').oninput = () => {
  let start = Number($('timelineStart').value), end = Number($('timelineEnd').value);
  if (end <= start) { end = Math.min(Number($('timelineEnd').max), start + Number($('timelineEnd').step)); $('timelineEnd').value = end; }
  const duration = selectedVideo()?.info?.duration || end;
  $('end').value = Math.abs(end - duration) < Number($('timelineEnd').step) ? '' : clockValue(end);
  savePerClipField('end');
  if (Number($('timelinePlay').value) > end) updatePreview(end); else updateTimelineLabels();
};
$('timeline').onwheel = event => { event.preventDefault(); const step = Number($('timelinePlay').step) || .04; updatePreview(Number($('timelinePlay').value) + (event.deltaY > 0 ? step : -step)); };
$('timeline').onkeydown = event => { if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') { event.preventDefault(); const step = Number($('timelinePlay').step) || .04; updatePreview(Number($('timelinePlay').value) + (event.key === 'ArrowRight' ? step : -step)); } };
$('previewStart').onclick = () => updatePreview(Number($('timelineStart').value));
$('previewEnd').onclick = () => updatePreview(Number($('timelineEnd').value));
$('jumpBegin').onclick = () => updatePreview(0);
$('jumpEnd').onclick = () => updatePreview(selectedVideo()?.info?.duration || 0);

const PRESETS = {
  best:{fps:'smooth',compression:'delta',normalize:true,limiter:true,defaults:{fit:'fit',audio:'mix',paletteMode:'scene',ditherMode:'error'}},
  balanced:{fps:'balanced',compression:'delta',normalize:false,limiter:true,defaults:{fit:'fit',audio:'mix',paletteMode:'shared',ditherMode:'ordered'}},
  long:{fps:'compact',compression:'delta',normalize:false,limiter:true,defaults:{fit:'fit',audio:'mix',paletteMode:'shared',ditherMode:'ordered'}},
  small:{fps:'compact',compression:'delta',normalize:false,limiter:false,defaults:{fit:'fit',audio:'none',paletteMode:'shared',ditherMode:'off'}}
};
$('preset').onchange = () => {
  const preset = PRESETS[$('preset').value];
  if (!preset) return;
  $('fps').value = preset.fps; $('compression').value = preset.compression; $('normalize').checked = preset.normalize; $('limiter').checked = preset.limiter;
  Object.assign(projectDefaults, preset.defaults);
  refreshScope(true); estimate();
};

function globalValues(includeMenuTheme = true) {
  return {
    fps:$('fps').value, seekSeconds:Number($('seekSeconds').value), compression:$('compression').value,
    normalize:$('normalize').checked, limiter:$('limiter').checked, resume:$('resume').checked,
    romTitle:$('romTitle').value, outputMode:$('outputMode').value,
    splitVideo:$('splitVideo').checked, splitBudgetMiB:Number($('splitBudget').value),
    maxPartDuration:$('maxPartDuration').value.trim() || '0',
    chapterAware:$('chapterAware').checked, partTitleScreens:$('partTitleScreens').checked,
    resumeLongSplit:$('resumeLongSplit').checked,
    menuBackground:$('menuBackground')?.value || 'ocean-wave-animated',
    menuUIColor:$('menuUIColor')?.value || 'white',
    menuOutline:!!$('menuOutline')?.checked,
    menuOutlineColor:$('menuOutlineColor')?.value || 'black',
    menuTheme:includeMenuTheme && $('outputMode').value === 'menu' ? serializedMenuTheme() : null
  };
}
function values() {
  const global = globalValues();
  return {
    ...global, ...projectDefaults,
    clips: state.videos.map(video => {
      const config = clipConfigs[video.id];
      return {id:video.id,title:config.title,useProject:config.useProject,start:config.start,end:config.end,speed:Number(config.speed),fit:config.fit,audio:config.audio,volume:Number(config.volume),loop:!!config.loop,paletteMode:config.paletteMode,ditherMode:config.ditherMode};
    })
  };
}
function modelSnapshot() {
  return {global:globalValues(false), defaults:cloneClip(projectDefaults), clips:JSON.parse(JSON.stringify(clipConfigs))};
}
function modelEffective(model, id) {
  const config = model.clips[id];
  return config.useProject ? {...model.defaults, title:config.title} : {...config};
}
function estimateModel(model) {
  if (!state?.videos?.length) return {bytes:0,frames:0,breakdown:{}};
  const vblanks = FPS_VBLANKS[model.global.fps] || 5;
  const fps = 59.727500569606 / vblanks;
  let player = 32768 + state.videos.length * 96 + (model.global.outputMode === 'menu' ? menuThemeBytes() : 0), videoBytes = 0, audioBytes = 0, paletteBytes = 0, indexBytes = 0, frames = 0, sourceDuration = 0;
  for (const video of state.videos) {
    if (!video.info) continue;
    const clip = modelEffective(model, video.id);
    const start = Math.max(0, parseClock(clip.start) || 0);
    let end = clip.end?.trim() ? parseClock(clip.end) : video.info.duration;
    if (!Number.isFinite(end)) end = video.info.duration;
    end = Math.min(video.info.duration, end);
    if (end <= start || !Number.isFinite(Number(clip.speed)) || clip.speed <= 0) return {error:'Check trim settings.'};
    const sourceClipDuration = end - start;
    sourceDuration += sourceClipDuration;
    const displayDuration = sourceClipDuration / Number(clip.speed);
    const frameCount = Math.max(1, Math.ceil(displayDuration * fps));
    frames += frameCount;
    const compressionFactor = model.global.compression === 'delta' ? 0.68 : 1;
    videoBytes += frameCount * 9600 * compressionFactor;
    if (model.global.compression === 'delta') indexBytes += frameCount * 8;
    const palettes = clip.paletteMode === 'scene' ? Math.max(1, Math.ceil(frameCount / 60)) : 1;
    paletteBytes += palettes * 512 + (palettes > 1 ? frameCount * 2 : 0);
    if (clip.audio !== 'none' && video.info.audioStreams) audioBytes += displayDuration * 16384 + frameCount * 4;
  }
  const bytes = Math.ceil(player + videoBytes + audioBytes + paletteBytes + indexBytes);
  let cartridge = 1 << 20;
  while (cartridge < bytes && cartridge < ROM_LIMIT) cartridge *= 2;
  return {bytes, cartridge, frames, fps, sourceDuration, breakdown:{player,video:videoBytes,audio:audioBytes,palettes:paletteBytes,indexes:indexBytes}};
}
function estimate() {
  const model = modelSnapshot();
  const result = estimateModel(model);
  if (result.error) { $('estimate').textContent = result.error; return result; }
  const single = state.videos.length === 1;
  const manualSplit = single && !!model.global.splitVideo;
  const budgetMiB = manualSplit ? Math.max(1, Math.min(32, Number(model.global.splitBudgetMiB) || 31)) : 32;
  const budgetBytes = budgetMiB * MIB;
  const overhead = 32768 + 96 + 512;
  const usable = Math.max(1, budgetBytes - overhead);
  const payload = Math.max(1, result.bytes - overhead);
  let estimatedParts = Math.max(1, Math.ceil(payload / usable));
  const maxPartSeconds = manualSplit ? parsePartDuration(model.global.maxPartDuration) : 0;
  if (!Number.isFinite(maxPartSeconds)) { $('estimate').innerHTML = '<b class="estimate-over">Maximum duration must be 0 or MM:SS, for example 1:05.</b>'; return {...result,error:'invalid maximum part duration'}; }
  if (single && maxPartSeconds > 0) estimatedParts = Math.max(estimatedParts, Math.ceil(result.sourceDuration / maxPartSeconds));
  const automaticSplit = single && estimatedParts > 1;
  $('optimize').classList.toggle('hidden', automaticSplit);
  const over = result.bytes > ROM_LIMIT;
  const headline = automaticSplit
    ? '<b>Estimated output: ' + estimatedParts + ' ROM parts</b>'
    : (over ? '<b class="estimate-over">Estimated data exceeds 32 MiB</b>' : 'Estimated output: <b>1 ROM</b> • Cartridge: <b>' + (result.cartridge/MIB) + ' MiB</b>');
  const splitNote = automaticSplit
    ? '<br>' + (manualSplit ? 'Split target: ' : 'Automatic target: ') + budgetMiB + ' MiB per ROM' + (maxPartSeconds > 0 ? ' • maximum ' + partDurationValue(maxPartSeconds) + ' per part' : '') +
      (manualSplit && model.global.chapterAware && state.videos[0]?.info?.chapters?.length ? ' • ' + state.videos[0].info.chapters.length + ' chapter boundaries found' : '')
    : (manualSplit ? '<br>Split rules are enabled; this video currently fits one part.' : '');
  $('estimate').innerHTML = headline +
    '<br>Estimated data: ' + (result.bytes/MIB).toFixed(2) + ' MiB • ' + result.frames + ' frames • ' + result.fps.toFixed(2) + ' fps' +
    '<br>Video ' + (result.breakdown.video/MIB).toFixed(2) + ' MiB • Audio ' + (result.breakdown.audio/MIB).toFixed(2) + ' MiB • Palettes/indexes ' + ((result.breakdown.palettes+result.breakdown.indexes)/MIB).toFixed(2) + ' MiB' + splitNote;
  result.estimatedParts = estimatedParts;
  result.splitBudgetMiB = budgetMiB;
  return result;
}

function makeCustomInModel(model, id) {
  const config = model.clips[id];
  if (config.useProject) model.clips[id] = {...model.defaults, title:config.title, useProject:false};
  return model.clips[id];
}
function buildOptimizerProposal() {
  const beforeModel = modelSnapshot();
  const model = JSON.parse(JSON.stringify(beforeModel));
  const changes = [];
  let result = estimateModel(model);
  if (result.bytes <= ROM_LIMIT) return {model,changes:['The current project already fits within 32 MiB.'],before:result,after:result,noop:true};
  const before = result;
  if (model.global.compression !== 'delta') {
    model.global.compression = 'delta'; changes.push('Video compression: Uncompressed → Delta + keyframes'); result = estimateModel(model);
  }
  while (result.bytes > ROM_LIMIT) {
    const index = FPS_ORDER.indexOf(model.global.fps);
    if (index < 0 || index === FPS_ORDER.length - 1) break;
    const old = model.global.fps, next = FPS_ORDER[index + 1]; model.global.fps = next;
    changes.push('Frame rate: ' + (59.7275/FPS_VBLANKS[old]).toFixed(2) + ' fps → ' + (59.7275/FPS_VBLANKS[next]).toFixed(2) + ' fps');
    result = estimateModel(model);
  }
  if (result.bytes > ROM_LIMIT) {
    if (model.defaults.paletteMode === 'scene') { model.defaults.paletteMode = 'shared'; changes.push('Project palette: Per-scene → Shared'); }
    for (const video of state.videos) {
      const config = model.clips[video.id];
      if (!config.useProject && config.paletteMode === 'scene') { config.paletteMode = 'shared'; changes.push(video.name + ': Per-scene palette → Shared'); }
    }
    result = estimateModel(model);
  }
  if (result.bytes > ROM_LIMIT) {
    const candidates = state.videos.map(video => {
      const clip = modelEffective(model, video.id);
      const start = parseClock(clip.start) || 0;
      const end = clip.end?.trim() ? parseClock(clip.end) : video.info.duration;
      return {video,bytes:clip.audio === 'none' ? 0 : Math.max(0,(end-start)/clip.speed)*16384};
    }).sort((a,b) => b.bytes-a.bytes);
    for (const candidate of candidates) {
      if (result.bytes <= ROM_LIMIT) break;
      if (!candidate.bytes) continue;
      const config = makeCustomInModel(model, candidate.video.id);
      config.audio = 'none';
      changes.push(candidate.video.name + ': Audio → None');
      result = estimateModel(model);
    }
  }
  if (result.bytes > ROM_LIMIT && state.videos.length) {
    const video = state.videos.find(item => item.id === selectedID) || state.videos[state.videos.length-1];
    const config = makeCustomInModel(model, video.id);
    const start = parseClock(config.start) || 0;
    const originalEnd = config.end?.trim() ? Math.min(parseClock(config.end), video.info.duration) : video.info.duration;
    let low = start + .2, high = originalEnd, best = low;
    for (let i=0;i<24;i++) {
      const mid = (low+high)/2; config.end = clockValue(mid);
      const test = estimateModel(model);
      if (test.bytes <= ROM_LIMIT) { best=mid; low=mid; } else high=mid;
    }
    config.end = clockValue(best);
    changes.push(video.name + ': End time shortened to ' + clockValue(best));
    result = estimateModel(model);
  }
  return {model,changes,before,after:result,noop:false};
}
$('optimize').onclick = () => {
  optimizerProposal = buildOptimizerProposal();
  const list = optimizerProposal.changes.map(change => '<li>' + escapeHTML(change) + '</li>').join('');
  $('optimizerSummary').innerHTML = '<p>Estimated size: <b>' + (optimizerProposal.before.bytes/1048576).toFixed(2) + ' MiB</b> → <b>' + (optimizerProposal.after.bytes/1048576).toFixed(2) + ' MiB</b></p><ul>' + list + '</ul>' +
    (optimizerProposal.after.bytes > ROM_LIMIT ? '<p class="estimate-over">Even the strongest automatic changes may not fit; shorten more clips manually.</p>' : '');
  $('optimizerApply').classList.toggle('hidden', optimizerProposal.noop);
  $('optimizerModal').classList.remove('hidden');
};
$('optimizerCancel').onclick = () => $('optimizerModal').classList.add('hidden');
$('optimizerApply').onclick = () => {
  if (!optimizerProposal) return;
  projectDefaults = cloneClip(optimizerProposal.model.defaults);
  clipConfigs = optimizerProposal.model.clips;
  const global = optimizerProposal.model.global;
  for (const key of ['fps','compression']) $(key).value = global[key];
  for (const key of ['normalize','limiter']) $(key).checked = global[key];
  $('preset').value = 'custom';
  $('optimizerModal').classList.add('hidden');
  refreshScope(true); renderClips(); estimate();
};

function applyPendingProject() {
  const settings = pendingProject;
  pendingProject = null;
  projectDefaults = cloneClip({start:settings.start,end:settings.end,speed:settings.speed,fit:settings.fit || 'fit',audio:settings.audio,volume:settings.volume,loop:settings.loop,paletteMode:settings.paletteMode,ditherMode:settings.ditherMode});
  for (const key of ['fps','compression','outputMode']) if (settings[key]) $(key).value = settings[key];
  $('seekSeconds').value = settings.seekSeconds || 5;
  $('normalize').checked = !!settings.normalize; $('limiter').checked = !!settings.limiter; $('resume').checked = !!settings.resume;
  $('splitVideo').checked = !!settings.splitVideo;
  $('splitBudget').value = settings.splitBudgetMiB || 31;
  $('maxPartDuration').value = settings.maxPartDuration || partDurationValue((Number(settings.maxPartMinutes) || 0) * 60);
  $('chapterAware').checked = settings.chapterAware !== false; $('partTitleScreens').checked = settings.partTitleScreens !== false; $('resumeLongSplit').checked = settings.resumeLongSplit !== false; updateSplitBudgetLabel(); updateSplitControls();
  $('romTitle').value = settings.romTitle || ''; romTitleAuto = false;
  if ($('menuBackground')) {
    $('menuBackground').value = settings.menuBackground || settings.menuTheme?.id || 'ocean-wave-animated';
    $('menuUIColor').value = settings.menuUIColor || 'white';
    $('menuOutline').checked = settings.menuOutline !== false;
    $('menuOutlineColor').value = settings.menuOutlineColor || 'black';
    if (($('menuBackground').value === 'custom') && settings.menuTheme) customMenuTheme = MenuThemeTools.deserializeTheme(settings.menuTheme);
    rebuildMenuTheme();
  }
  clipConfigs = {};
  for (const clip of settings.clips || []) clipConfigs[clip.id] = {title:clip.title || 'GBA VIDEO',useProject:clip.useProject !== false,start:clip.start || '0:00',end:clip.end || '',speed:clip.speed || 1,fit:clip.fit || 'fit',audio:clip.audio || 'mix',volume:Number.isFinite(clip.volume)?clip.volume:100,loop:!!clip.loop,paletteMode:clip.paletteMode || 'shared',ditherMode:clip.ditherMode || 'ordered'};
  ensureClipConfigs(); selectedID = state.videos[0]?.id || ''; editScope = 'project'; lastPreviewKey=''; lastThumbKey='';
}
async function saveProject() {
  try {
    const response = await api('/project/save', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(values())});
    if (!response.cancelled) alert('Project saved:\n' + response.path);
  } catch (error) { alert(error.message); }
}
async function openProject() {
  try {
    const response = await api('/project/open', {method:'POST'});
    if (response.cancelled) return;
    pendingProject = response.settings;
    scopeInitialized = false;
    clipConfigs = {}; selectedID = ''; lastPreviewKey=''; lastThumbKey='';
    await poll();
  } catch (error) { alert(error.message); }
}
$('saveProject').onclick = saveProject;
$('openProject').onclick = openProject;
$('openProjectWelcome').onclick = event => { event.stopPropagation(); openProject(); };

if ($('menuBackground')) {
  rebuildMenuTheme();
  stopMenuPreview = MenuThemeTools.startPreview($('menuPreview'), () => activeMenuTheme, menuStyleSettings);
  $('menuBackground').addEventListener('change', () => { rebuildMenuTheme(); estimate(); });
  $('menuUIColor').addEventListener('change', () => { rebuildMenuTheme(); estimate(); });
  $('menuOutline').addEventListener('change', () => { rebuildMenuTheme(); estimate(); });
  $('menuOutlineColor').addEventListener('change', () => { rebuildMenuTheme(); estimate(); });
  $('customMenuBackground').addEventListener('change', event => loadCustomMenuBackground(event.target.files?.[0]));
  $('clearCustomMenuBackground').onclick = () => { customMenuTheme = null; $('customMenuBackground').value = ''; $('menuBackgroundStatus').textContent = 'Choose a PNG, JPG, WebP or GIF.'; rebuildMenuTheme(); estimate(); };
}

$('audioPreview').onclick = async () => {
  try {
    $('audioPreview').disabled = true;
    const response = await fetch(BASE + '/audio-preview?index=' + selectedIndex(), {method:'POST',headers:headers({'Content-Type':'application/json'}),body:JSON.stringify(values())});
    if (!response.ok) { const body = await response.json(); throw new Error(body.error); }
    const blob = await response.blob(); if (audioURL) URL.revokeObjectURL(audioURL); audioURL = URL.createObjectURL(blob); $('audioPlayer').src = audioURL; await $('audioPlayer').play();
  } catch (error) { alert(error.message); } finally { $('audioPreview').disabled = false; }
};
$('convert').onclick = async () => {
  try { await api('/convert', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(values())}); poll(); }
  catch (error) { $('convertError').textContent = error.message; $('convertError').classList.remove('hidden'); }
};
$('download').onclick = () => { const link = document.createElement('a'); link.href = BASE + '/download'; link.download = state.downloadName || 'GBA_Video_Maker_output'; link.click(); };
$('retryEngine').onclick = () => api('/engine/retry', {method:'POST'});
$('resetTop').onclick = async () => {
  await api('/reset', {method:'POST'}); state=null; selectedID=''; clipConfigs={}; scopeInitialized=false; projectDefaults={...DEFAULT_CLIP}; playheads={}; lastPreviewKey=''; lastThumbKey=''; romTitleAuto=true; $('romTitle').value=''; show('welcome');
};
setInterval(() => fetch(BASE + '/heartbeat', {method:'POST',headers:headers(),keepalive:true}).catch(()=>{}), 5000);
window.addEventListener('pagehide', () => fetch(BASE + '/close-intent', {method:'POST',headers:headers(),keepalive:true}).catch(()=>{}));

poll();

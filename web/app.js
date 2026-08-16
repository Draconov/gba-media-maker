(async () => {
const [GBAText, MenuThemeTools, TitleCardTools] = await Promise.all([
  import('./shared/gba-text.js'),
  import('./shared/menu-themes.js'),
  import('./shared/title-cards.js'),
]);

const TOKEN = document.querySelector('meta[name="gbavm-session-token"]').content;
const APP_BASE = '/' + TOKEN;
const BASE = APP_BASE + '/api';
const $ = id => document.getElementById(id);

const DEFAULT_CLIP = Object.freeze({
  start: '0:00', end: '', speed: 1, fit: 'fit', audio: 'mix', volume: 100,
  loop: false, paletteMode: 'shared', ditherMode: 'ordered', imageSeconds: 5
});
const FPS_VBLANKS = {smooth: 4, balanced: 5, classic: 6, compact: 8};
const FPS_ORDER = ['smooth', 'balanced', 'classic', 'compact'];
const ROM_LIMIT = 32 * 1024 * 1024;
const MIB = 1024 * 1024;
const AUDIO_ARTWORK_PRESETS = Array.from({length:20}, (_,index) => 'preset-' + String(index + 1).padStart(2,'0'));

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
let smartAnalysis = null;
let smartAbort = null;
let scopeInitialized = false;
let menuBackgroundID = 'ocean-wave-animated';
let customMenuTheme = null;
let customMenuSourceFile = null;
let customMenuSourceIsVideo = false;
let customMenuLoadToken = 0;
let activeMenuTheme = null;
let stopMenuPreview = null;
let titleCardProject = null;
let titleCardProjectSource = "";
let titleCardPart = 1;
let titleCardEstimatedParts = 1;
let titleCardPreviewTimer = null;
let titleCardPreviewAbort = null;
let titleCardPreviewPendingKey = "";
let titleCardPreviewDesiredKey = "";
let titleCardSectionSignature = "";
const titleCardPreviewCache = new Map();

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
  return sanitizeMenuTitle(name.replace(/\.[^.]+$/, '')).value || 'GBA MEDIA';
}
function sanitizeMenuTitle(value) {
  const result = GBAText.sanitizeGBAText(value, 12);
  return {value: result.text, invalid: result.unsupported.length > 0, unsupported: result.unsupported};
}
function selectedIndex() { return state?.videos?.findIndex(v => v.id === selectedID) ?? -1; }
function selectedVideo() { return state?.videos?.find(v => v.id === selectedID) || state?.videos?.[0]; }
function mediaKind(video=selectedVideo()){ return video?.info?.kind || 'video'; }
function isGIF(video=selectedVideo()){ return /\.gif$/i.test(video?.name || ''); }
function rawFilenameTitle(name){ return String(name || '').replace(/\.[^.]+$/, '').trim(); }
function defaultMusicTitle(video){ return String(video?.info?.title || rawFilenameTitle(video?.name) || 'UNTITLED').trim().slice(0,28); }
function defaultMusicArtist(video){ return String(video?.info?.artist || '').trim().slice(0,28); }
function normalizeMusicArtworkPreset(value){ return AUDIO_ARTWORK_PRESETS.includes(value) ? value : AUDIO_ARTWORK_PRESETS[0]; }
function automaticMusicArtworkPreset(seed){ let h=2166136261>>>0; for(const ch of String(seed||'')){ h^=ch.codePointAt(0); h=Math.imul(h,16777619)>>>0; } return AUDIO_ARTWORK_PRESETS[h%AUDIO_ARTWORK_PRESETS.length]; }
function musicArtworkPresetURL(value){ return APP_BASE + '/audio-artwork/' + normalizeMusicArtworkPreset(value) + '.png'; }
function musicArtworkPreviewSource(video=selectedVideo()) {
  if (!video || mediaKind(video) !== 'audio') return '';
  const config=clipConfigs[video.id] || {};
  const mode=['default','embedded','custom'].includes(config.musicArtworkMode) ? config.musicArtworkMode : 'embedded';
  const preset=normalizeMusicArtworkPreset(config.musicArtworkPreset);
  if(mode==='custom' && /^data:image\/png;base64,/i.test(config.musicArtworkCustom || '')) return config.musicArtworkCustom;
  if(mode==='default') return musicArtworkPresetURL(preset);
  const index=state?.videos?.findIndex(item=>item.id===video.id) ?? 0;
  return BASE + '/preview?index=' + Math.max(0,index) + '&time=0&fit=fit&artworkFallback=' + encodeURIComponent(preset) + '&artwork=embedded';
}
function titleForVideo(video){ const meta=video?.info?.title; return titleFromFilename((meta && mediaKind(video)==='audio') ? meta : (video?.name || 'GBA MEDIA')); }
function cloneClip(settings) { return {...DEFAULT_CLIP, ...settings}; }
function effectiveClip(id) {
  const config = clipConfigs[id] || {title: 'GBA VIDEO', useProject: true, audioTrack: 0, ...DEFAULT_CLIP};
  const result = config.useProject ? {...projectDefaults, audioTrack: Number(config.audioTrack) || 0, title: config.title, useProject: true}
    : {...cloneClip(config), audioTrack: Number(config.audioTrack) || 0, title: config.title, useProject: false};
  if (isGIF(state?.videos?.find(video => video.id === id))) result.loop = true;
  return result;
}
function ensureClipConfigs() {
  if (!state?.videos) return;
  const valid = new Set(state.videos.map(v => v.id));
  for (const id of Object.keys(clipConfigs)) if (!valid.has(id)) delete clipConfigs[id];
  for (const video of state.videos) {
    if (!clipConfigs[video.id]) {
      clipConfigs[video.id] = {title: titleForVideo(video), useProject: true, audioTrack: 0, musicTitle: defaultMusicTitle(video), musicArtist: defaultMusicArtist(video), musicArtworkMode:'embedded', musicArtworkPreset:automaticMusicArtworkPreset(video.name), musicArtworkCustom:'', musicSeekSeconds:5, ...DEFAULT_CLIP};
    }
    if (clipConfigs[video.id].musicTitle == null) clipConfigs[video.id].musicTitle = defaultMusicTitle(video);
    if (clipConfigs[video.id].musicArtist == null) clipConfigs[video.id].musicArtist = defaultMusicArtist(video);
    if (!['default','embedded','custom'].includes(clipConfigs[video.id].musicArtworkMode)) clipConfigs[video.id].musicArtworkMode = 'embedded';
    clipConfigs[video.id].musicArtworkPreset = normalizeMusicArtworkPreset(clipConfigs[video.id].musicArtworkPreset);
    if (clipConfigs[video.id].musicArtworkCustom == null) clipConfigs[video.id].musicArtworkCustom = '';
    if (![3,5,10,15].includes(Number(clipConfigs[video.id].musicSeekSeconds))) clipConfigs[video.id].musicSeekSeconds = 5;
    if (isGIF(video)) clipConfigs[video.id].loop = true;
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
    $('loadingTitle').textContent = state.inspectStatus === 'waiting' ? 'Preparing the app…' : 'Opening media…';
    $('loadingText').textContent = state.inspectStatus === 'waiting'
      ? (state.engineMessage || 'Preparing FFmpeg') : 'Reading media type, duration, dimensions and audio streams.';
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
    $('loadingTitle').textContent = 'Could not open media';
    $('loadingText').textContent = state.inspectError || 'A media file could not be inspected.';
  }
  if (state.inspectStatus === 'ready') {
    show('editor');
    ensureClipConfigs();
    if (pendingProject) applyPendingProject();
    renderClips();
    updateOutputModes();
    updateExtremeUI();
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
  const ids = ['preset','audioQuality','smartTarget','smartPriority','start','end','speed','fps','fit','seekSeconds','paletteMode','ditherMode','compression','audioTrack','audio','volume','normalize','limiter','romTitle','outputMode','loop','resume','splitVideo','splitBudget','maxPartDuration','chapterAware','partTitleScreens','resumeLongSplit','titleCardUseShared','titleCardPartSelect','titleCardTitle','titleCardSubtitle','titleCardBackground','titleCardFrameOffset','titleCardDarkness','titleCardSolidColor','titleCardTextColor','titleCardSubtitleTextColor','titleCardOutline','titleCardOutlineColor','titleCardSubtitleOutlineColor','titleCardAlignment','titleCardSubtitleAlignment','titleCardTextSize','titleCardSubtitleTextSize','titleCardStartMode','titleCardDuration','titleCardAllowSkip','titleCardFade','useProject','menuTitle','menuBackground','customMenuBackground','customMenuVideoStart','customMenuVideoDuration','menuUIColor','menuSelectionColor','menuOutline','menuOutlineColor','imageSeconds','imageSlideshow','imageFit','musicTitle','musicArtist','musicStart','musicEnd','musicSpeed','musicArtworkMode','musicArtworkCustom','musicSeekSeconds'];
  ids.forEach(id => { if ($(id)) $(id).disabled = busy || $(id).dataset.scopeDisabled === '1'; });
  ['convert','optimize','smartAnalyze','addVideos','moveUp','moveDown','saveProject','openProject'].forEach(id => { if ($(id)) $(id).disabled = busy; });
  for (const button of document.querySelectorAll('.music-artwork-preset')) button.disabled = busy;
  if(!busy && state?.videos?.length) syncMediaAliasControls();
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

function audioTrackLabel(track, fallbackIndex = 0) {
  const number = Number.isInteger(track?.index) ? track.index + 1 : fallbackIndex + 1;
  const details = [];
  const title = String(track?.title || '').trim();
  const language = String(track?.language || '').trim();
  if (title) details.push(title);
  else if (language) details.push(language.toUpperCase());
  if (title && language && !title.toLowerCase().includes(language.toLowerCase())) details.push(language.toUpperCase());
  if (Number(track?.channels) > 0) details.push(Number(track.channels) === 1 ? 'mono' : Number(track.channels) === 2 ? 'stereo' : `${track.channels} ch`);
  if (track?.default) details.push('default');
  return `Track ${number}${details.length ? ' — ' + details.join(' · ') : ''}`;
}
function refreshAudioTrackSelector() {
  const field = $('audioTrackField'), select = $('audioTrack');
  if (!field || !select) return;
  const video = selectedVideo();
  const config = clipConfigs[selectedID];
  const tracks = video?.info?.audioTracks || [];
  const count = Number(video?.info?.audioStreams || tracks.length || 0);
  field.classList.toggle('hidden', count <= 1);
  select.replaceChildren();
  if (count <= 0) {
    const option = document.createElement('option'); option.value='0'; option.textContent='No audio tracks'; select.append(option);
    select.disabled = true;
    if (config) config.audioTrack = 0;
    return;
  }
  for (let index=0; index<count; index++) {
    const option=document.createElement('option');
    option.value=String(index); option.textContent=audioTrackLabel(tracks[index], index); select.append(option);
  }
  let chosen = Number(config?.audioTrack || 0);
  if (!Number.isInteger(chosen) || chosen < 0 || chosen >= count) chosen = 0;
  if (config) config.audioTrack = chosen;
  select.value=String(chosen);
  select.disabled=!!state?.converting;
}

function renderClips() {
  if (!state?.videos) return;
  const kindIcon={video:'🎬',audio:'🎵',image:'🖼'};
  const kindName={video:'Video',audio:'Audio',image:'Image'};
  const html = state.videos.map((video, index) => {
    const info = video.info;
    const config = clipConfigs[video.id];
    const kind=info?.kind || 'video';
    let status=video.error || video.status;
    if(info){
      if(kind==='image') status=`${info.width}×${info.height} • Image`;
      else if(kind==='audio') status=`${fmt(info.duration)} • ${info.audioStreams||1} audio track${(info.audioStreams||1)===1?'':'s'}${info.artist?' • '+info.artist:''}`;
      else status=isGIF(video) ? `${info.width}×${info.height} • ${fmt(info.duration)} • GIF • auto-loop` : `${info.width}×${info.height} • ${fmt(info.duration)}${info.audioStreams?' • audio':' • silent'}`;
    }
    const relink = video.needsRelink ? '<button class="clip-action relink" data-relink="' + index + '" title="Relink source file">↻</button>' : '';
    return '<div class="clip ' + (video.id === selectedID ? 'active' : '') + '" draggable="true" data-id="' + video.id + '">' +
      '<span class="clip-handle" title="Drag to reorder">⋮⋮</span><div class="clip-info"><b>' + (index + 1) + '. <span class="media-kind">'+(kindIcon[kind]||'•')+' '+(kindName[kind]||'Media')+'</span> ' + escapeHTML(video.name) + '</b><small>' + escapeHTML(status) + '</small>' +
      '<span class="clip-badge ' + (config?.useProject ? '' : 'custom') + '">' + (config?.useProject ? 'Project' : 'Custom') + '</span></div>' + relink +
      '<button class="clip-action remove" data-remove="' + index + '" title="Remove this media item">×</button></div>';
  }).join('');
  $('clips').innerHTML = html;
  for (const element of $('clips').querySelectorAll('.clip')) {
    element.onclick = event => {
      if (event.target.closest('button')) return;
      selectedID = element.dataset.id; editScope = 'clip'; refreshScope(true); renderClips(); syncTimeline(true);
    };
    element.ondragstart = event => { draggedID = element.dataset.id; element.classList.add('dragging'); event.dataTransfer.effectAllowed = 'move'; };
    element.ondragend = () => { draggedID = ''; element.classList.remove('dragging'); document.querySelectorAll('.drop-before').forEach(x => x.classList.remove('drop-before')); };
    element.ondragover = event => { event.preventDefault(); if (draggedID && draggedID !== element.dataset.id) element.classList.add('drop-before'); };
    element.ondragleave = () => element.classList.remove('drop-before');
    element.ondrop = async event => { event.preventDefault(); element.classList.remove('drop-before'); if (!draggedID || draggedID === element.dataset.id) return; const ids = state.videos.map(v => v.id); const from = ids.indexOf(draggedID), to = ids.indexOf(element.dataset.id); ids.splice(to, 0, ids.splice(from, 1)[0]); await applyOrder(ids); };
  }
  for (const button of $('clips').querySelectorAll('[data-remove]')) button.onclick = event => { event.stopPropagation(); removeVideo(+button.dataset.remove); };
  for (const button of $('clips').querySelectorAll('[data-relink]')) button.onclick = event => { event.stopPropagation(); relinkVideo(+button.dataset.relink); };
  const media = selectedVideo(),kind=mediaKind(media);
  if(media?.info){
    if(kind==='audio') $('clipInfo').textContent = [media.info.title||media.name,media.info.artist,media.info.album].filter(Boolean).join(' • ');
    else if(kind==='image') $('clipInfo').textContent = `Previewing ${media.name} • native output 240×160 RGB555`;
    else $('clipInfo').textContent = 'Previewing ' + media.name + ' • ' + Number(media.info.fps||0).toFixed(2) + ' source fps';
  } else $('clipInfo').textContent=media?.error||'';
  refreshAudioTrackSelector();
  updateMediaSettingsVisibility();
  if (romTitleAuto && state.videos[0]) $('romTitle').value = titleForVideo(state.videos[0]);
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
  return {uiColor:$('menuUIColor')?.value || '#FFFFFF', selectedColor:$('menuSelectionColor')?.value || '#FFDE00', outline:!!$('menuOutline')?.checked, outlineColor:$('menuOutlineColor')?.value || '#000000'};
}
function updateMenuColorReadouts() {
  for (const [inputID, outputID, fallback] of [
    ['menuUIColor','menuUIColorValue','#FFFFFF'],
    ['menuSelectionColor','menuSelectionColorValue','#FFDE00'],
    ['menuOutlineColor','menuOutlineColorValue','#000000']
  ]) {
    const input=$(inputID), output=$(outputID);
    if (!input || !output) continue;
    const color=MenuThemeTools.describeColor(input.value,fallback);
    output.textContent=`${color.hex} · RGB555 ${color.r},${color.g},${color.b}`;
    input._gbaColorPickerController?.sync();
  }
}
function snapMenuColor(inputID, fallback) {
  const input=$(inputID);
  if (!input) return;
  input.value=MenuThemeTools.quantizeHexColor(input.value,fallback);
  updateMenuColorReadouts();
}
function restoreMenuColors(settings={}) {
  const colors=MenuThemeTools.settingsColours({
    uiColor:settings.menuUIColor ?? settings.menuTheme?.uiColor ?? 'white',
    selectedColor:settings.menuSelectionColor ?? settings.menuTheme?.selectedColor,
    outlineColor:settings.menuOutlineColor ?? settings.menuTheme?.outlineColor ?? 'black'
  });
  $('menuUIColor').value=MenuThemeTools.rgb555ToHex(colors.ui);
  $('menuSelectionColor').value=MenuThemeTools.rgb555ToHex(colors.selected);
  $('menuOutlineColor').value=MenuThemeTools.rgb555ToHex(colors.outline);
  updateMenuColorReadouts();
}
function rebuildMenuTheme() {
  if (!$('menuBackground')) return;
  menuBackgroundID = $('menuBackground').value;
  if (menuBackgroundID === 'custom' && customMenuTheme) activeMenuTheme = MenuThemeTools.applyUI(customMenuTheme, menuStyleSettings());
  else if (menuBackgroundID === 'custom') activeMenuTheme = MenuThemeTools.createBuiltinTheme('classic-dark', menuStyleSettings());
  else activeMenuTheme = MenuThemeTools.createBuiltinTheme(menuBackgroundID, menuStyleSettings());
  $('customMenuBackgroundRow').classList.toggle('hidden', menuBackgroundID !== 'custom');
  $('menuOutlineColor').disabled = !$('menuOutline').checked || !!state?.converting;
  $('menuOutlineColor')._gbaColorPickerController?.sync();
}
function serializedMenuTheme() {
  rebuildMenuTheme();
  return activeMenuTheme ? MenuThemeTools.serializeTheme(activeMenuTheme) : null;
}
function menuThemeBytes() {
  if (!activeMenuTheme) return 0;
  return 64 + activeMenuTheme.palette.length + activeMenuTheme.frames.reduce((sum, frame) => sum + frame.length, 0);
}
function menuBackgroundVideoTiming() {
  const start=parseClock($('customMenuVideoStart')?.value || '0');
  const duration=parseClock($('customMenuVideoDuration')?.value || '0:04');
  if(!Number.isFinite(start)||start<0) throw new Error('Video start must be a valid non-negative time.');
  if(!Number.isFinite(duration)||duration<1||duration>32) throw new Error('Video duration must be between 1 and 32 seconds.');
  return {start,duration};
}
async function decodeDesktopMenuVideo(file,settings,token) {
  const {start,duration}=menuBackgroundVideoTiming();
  $('menuBackgroundStatus').textContent='Uploading and decoding '+file.name+' with FFmpeg…';
  const form=new FormData(); form.append('video',file,file.name);
  const response=await fetch(BASE+'/menu-background/video?start='+encodeURIComponent(start)+'&duration='+encodeURIComponent(duration),{method:'POST',headers:headers(),body:form});
  if(!response.ok) {
    let message='Could not decode the menu background video.';
    try { const body=await response.json(); if(body?.error) message=body.error; } catch { const text=await response.text(); if(text) message=text; }
    throw new Error(message);
  }
  const bytes=new Uint8Array(await response.arrayBuffer());
  if(token!==customMenuLoadToken) return null;
  const frameVBlanks=Number(response.headers.get('X-Menu-Frame-VBlanks'))||12;
  return MenuThemeTools.decodeRGB24Frames(bytes,file.name,frameVBlanks,settings,fraction=>{
    if(token===customMenuLoadToken) $('menuBackgroundStatus').textContent='Optimizing '+file.name+'… '+Math.round(fraction*100)+'%';
  });
}
async function loadCustomMenuBackground(file=customMenuSourceFile) {
  if (!file) return;
  customMenuSourceFile=file;
  customMenuSourceIsVideo=MenuThemeTools.isVideoFile(file);
  $('customMenuVideoTiming')?.classList.toggle('hidden',!customMenuSourceIsVideo);
  const token=++customMenuLoadToken;
  $('menuBackgroundStatus').textContent = (customMenuSourceIsVideo?'Decoding ':'Optimizing ') + file.name + '…';
  try {
    const theme=customMenuSourceIsVideo
      ? await decodeDesktopMenuVideo(file,menuStyleSettings(),token)
      : await MenuThemeTools.decodeCustomFile(file,menuStyleSettings(),fraction=>{ if(token===customMenuLoadToken) $('menuBackgroundStatus').textContent='Optimizing '+file.name+'… '+Math.round(fraction*100)+'%'; });
    if(token!==customMenuLoadToken||!theme) return;
    customMenuTheme=theme;
    $('menuBackgroundStatus').textContent = customMenuTheme.frames.length > 1
      ? file.name + ' — ' + customMenuTheme.frames.length + ' optimized looping frames'
      : file.name + ' — optimized static background';
    rebuildMenuTheme(); estimate();
  } catch (error) {
    if(token!==customMenuLoadToken) return;
    customMenuTheme = null;
    $('menuBackgroundStatus').textContent = 'Could not read this image, GIF or video: ' + error.message;
  }
}


function titleCardSourceName() { return state?.videos?.[0]?.name || "GBA VIDEO"; }
function resetTitleCardPreviewCache() {
  clearTimeout(titleCardPreviewTimer);
  titleCardPreviewTimer = null;
  titleCardPreviewAbort?.abort();
  titleCardPreviewAbort = null;
  titleCardPreviewPendingKey = "";
  titleCardPreviewDesiredKey = "";
  titleCardPreviewCache.clear();
}
function ensureTitleCardProject(force = false) {
  const source = titleCardSourceName();
  if (force || !titleCardProject || titleCardProjectSource !== source) {
    resetTitleCardPreviewCache();
    titleCardSectionSignature = "";
    titleCardProject = TitleCardTools.createTitleCardProject(source);
    titleCardProjectSource = source;
    titleCardPart = 1;
  }
}
function titleCardPartRecord(part, create = false) {
  ensureTitleCardProject();
  if (!titleCardProject) return null;
  if (titleCardProject.useShared) return titleCardProject.shared;
  let record = (titleCardProject.parts || []).find(item => Number(item.part) === Number(part));
  if (!record && create) {
    record = {part:Number(part),settings:JSON.parse(JSON.stringify(titleCardProject.shared))};
    titleCardProject.parts.push(record);
  }
  return record?.settings || titleCardProject.shared;
}
function serializeTitleCards() {
  ensureTitleCardProject();
  const copy = JSON.parse(JSON.stringify(titleCardProject || TitleCardTools.createTitleCardProject(titleCardSourceName())));
  copy.enabled = !!$('partTitleScreens')?.checked;
  copy.useShared = !!$('titleCardUseShared')?.checked;
  return copy;
}
function titleCardColorReadout(inputID, outputID, fallback) {
  const input=$(inputID), output=$(outputID);
  if (!input || !output) return;
  const color=MenuThemeTools.describeColor(input.value,fallback);
  output.textContent=`${color.hex} · RGB555 ${color.r},${color.g},${color.b}`;
  input._gbaColorPickerController?.sync();
}
function updateTitleCardColorReadouts() {
  titleCardColorReadout('titleCardTextColor','titleCardTextColorValue','#FFFFFF');
  titleCardColorReadout('titleCardOutlineColor','titleCardOutlineColorValue','#000000');
  titleCardColorReadout('titleCardSubtitleTextColor','titleCardSubtitleTextColorValue','#FFFFFF');
  titleCardColorReadout('titleCardSubtitleOutlineColor','titleCardSubtitleOutlineColorValue','#000000');
  titleCardColorReadout('titleCardSolidColor','titleCardSolidColorValue','#000000');
}
function updateTitleCardTextWarning() {
  const warning=$('titleCardTextWarning');
  if (!warning) return;
  const subtitleForCheck=$('titleCardSubtitle').value.replaceAll('{part}','1');
  const unsupported=[...new Set([...GBAText.unsupportedGBARunes($('titleCardTitle').value), ...GBAText.unsupportedGBARunes(subtitleForCheck)])];
  warning.textContent=unsupported.length ? `Unsupported GBA characters: ${unsupported.join(' ')}. They will be replaced in the ROM.` : '';
  warning.classList.toggle('hidden', unsupported.length === 0);
}
function rawTitleCardSettings() {
  return {
    title:$('titleCardTitle').value,
    subtitle:$('titleCardSubtitle').value,
    backgroundMode:$('titleCardBackground').value,
    frameOffsetSeconds:Number($('titleCardFrameOffset').value)||0,
    darkness:Number($('titleCardDarkness').value)||0,
    solidColor:$('titleCardSolidColor').value,
    titleTextColor:$('titleCardTextColor').value,
    titleOutlineColor:$('titleCardOutlineColor').value,
    titleAlignment:$('titleCardAlignment').value,
    titleTextSize:$('titleCardTextSize').value,
    subtitleTextColor:$('titleCardSubtitleTextColor').value,
    subtitleOutlineColor:$('titleCardSubtitleOutlineColor').value,
    subtitleAlignment:$('titleCardSubtitleAlignment').value,
    subtitleTextSize:$('titleCardSubtitleTextSize').value,
    drawOutline:$('titleCardOutline').checked,
    startMode:$('titleCardStartMode').value,
    durationSeconds:Number($('titleCardDuration').value)||3,
    allowSkip:$('titleCardAllowSkip').checked,
    fade:$('titleCardFade').checked,
  };
}
function saveTitleCardFields() {
  if (!titleCardProject) ensureTitleCardProject();
  const target = titleCardPartRecord(titleCardPart, true);
  if (!target) return;
  Object.assign(target, rawTitleCardSettings());
  $('titleCardDarknessValue').textContent = `${target.darkness}%`;
  updateTitleCardConditionalFields();
  updateTitleCardColorReadouts();
  updateTitleCardTextWarning();
  renderTitleCardPreview();
}
function loadTitleCardFields() {
  ensureTitleCardProject();
  const settings = titleCardPartRecord(titleCardPart, false) || TitleCardTools.defaultTitleCardSettings(titleCardSourceName());
  $('titleCardTitle').value = settings.title ?? '';
  $('titleCardSubtitle').value = settings.subtitle ?? '';
  $('titleCardBackground').value = settings.backgroundMode || 'part-first-frame';
  $('titleCardFrameOffset').value = Number(settings.frameOffsetSeconds)||0;
  $('titleCardDarkness').value = Number.isFinite(Number(settings.darkness)) ? Number(settings.darkness) : 50;
  $('titleCardDarknessValue').textContent = `${$('titleCardDarkness').value}%`;
  $('titleCardSolidColor').value = settings.solidColor || '#000000';
  $('titleCardTextColor').value = settings.titleTextColor || settings.textColor || '#FFFFFF';
  $('titleCardOutlineColor').value = settings.titleOutlineColor || settings.outlineColor || '#000000';
  $('titleCardSubtitleTextColor').value = settings.subtitleTextColor || settings.textColor || '#FFFFFF';
  $('titleCardSubtitleOutlineColor').value = settings.subtitleOutlineColor || settings.outlineColor || '#000000';
  $('titleCardOutline').checked = settings.drawOutline !== false;
  $('titleCardAlignment').value = settings.titleAlignment || settings.alignment || 'center';
  $('titleCardTextSize').value = ['large','medium','small'].includes(settings.titleTextSize) ? settings.titleTextSize : (['medium','small'].includes(settings.textSize) ? settings.textSize : 'large');
  $('titleCardSubtitleAlignment').value = settings.subtitleAlignment || settings.alignment || 'center';
  $('titleCardSubtitleTextSize').value = ['large','medium','small'].includes(settings.subtitleTextSize) ? settings.subtitleTextSize : (settings.textSize === 'large' ? 'medium' : 'small');
  $('titleCardStartMode').value = settings.startMode === 'timer' ? 'timer' : 'button';
  $('titleCardDuration').value = Number(settings.durationSeconds)||3;
  $('titleCardAllowSkip').checked = settings.allowSkip !== false;
  $('titleCardFade').checked = settings.fade !== false;
  updateTitleCardConditionalFields();
  updateTitleCardColorReadouts();
  updateTitleCardTextWarning();
  renderTitleCardPreview();
}
function updateTitleCardConditionalFields() {
  const background=$('titleCardBackground').value;
  $('titleCardFrameOffsetField').classList.toggle('hidden', background !== 'part-frame');
  $('titleCardSolidColorField').classList.toggle('hidden', background !== 'solid');
  $('titleCardDurationField').classList.toggle('hidden', $('titleCardStartMode').value !== 'timer');
  $('titleCardAllowSkip').closest('label').classList.toggle('scope-muted', $('titleCardStartMode').value !== 'timer');
  const enabled = !!$('partTitleScreens').checked && !state?.converting;
  $('titleCardAllowSkip').disabled = !enabled || $('titleCardStartMode').value !== 'timer';
  $('titleCardOutlineColor').disabled = !enabled || !$('titleCardOutline').checked;
  $('titleCardSubtitleOutlineColor').disabled = !enabled || !$('titleCardOutline').checked;
  $('titleCardOutlineColor')._gbaColorPickerController?.sync();
  $('titleCardSubtitleOutlineColor')._gbaColorPickerController?.sync();
}
function estimatedTitleCardSourceTime(part) {
  const video=state?.videos?.[0];
  if (!video?.info) return 0;
  const clip=effectiveClip(video.id);
  const start=Math.max(0,parseClock(clip.start)||0);
  let end=clip.end?.trim()?parseClock(clip.end):video.info.duration;
  if (!Number.isFinite(end)) end=video.info.duration;
  end=Math.min(video.info.duration,Math.max(start,end));
  const segment=(end-start)/Math.max(1,titleCardEstimatedParts);
  const settings=titleCardPartRecord(part,false)||{};
  const offset=settings.backgroundMode==='part-frame'?Math.max(0,Number(settings.frameOffsetSeconds)||0):0;
  return Math.min(Math.max(start,end-0.04),start+(part-1)*segment+offset);
}
function titleCardPreviewFrameKey(part, fit) {
  const video = state?.videos?.[0];
  const time = estimatedTitleCardSourceTime(part);
  return `${video?.id || "video"}|${time.toFixed(3)}|${fit}`;
}
function drawCurrentTitleCardPreview(source) {
  if (!$('titleCardPreview') || !$('titleCardSection') || $('titleCardSection').classList.contains('hidden')) return;
  const settings = titleCardPartRecord(titleCardPart, false);
  const fit = effectiveClip(state.videos[0].id).fit || 'fit';
  TitleCardTools.renderTitleCardPreview($('titleCardPreview'), source, fit, settings, titleCardPart, titleCardSourceName());
}
function rememberTitleCardPreview(key, image) {
  titleCardPreviewCache.delete(key);
  titleCardPreviewCache.set(key, image);
  while (titleCardPreviewCache.size > 16) titleCardPreviewCache.delete(titleCardPreviewCache.keys().next().value);
}
function imageFromBlob(blob) {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(blob);
    const image = new Image();
    image.onload = () => { URL.revokeObjectURL(url); resolve(image); };
    image.onerror = () => { URL.revokeObjectURL(url); reject(new Error('Could not decode preview frame.')); };
    image.src = url;
  });
}
function renderTitleCardPreview() {
  if (!$('titleCardPreview') || !$('titleCardSection') || $('titleCardSection').classList.contains('hidden')) return;
  ensureTitleCardProject();
  const settings = titleCardPartRecord(titleCardPart, false);
  const fit = effectiveClip(state.videos[0].id).fit || 'fit';
  if (settings?.backgroundMode === 'solid') {
    titleCardPreviewDesiredKey = 'solid';
    clearTimeout(titleCardPreviewTimer);
    titleCardPreviewTimer = null;
    titleCardPreviewAbort?.abort();
    titleCardPreviewAbort = null;
    titleCardPreviewPendingKey = '';
    drawCurrentTitleCardPreview($('titleCardPreview'));
    return;
  }

  const key = titleCardPreviewFrameKey(titleCardPart, fit);
  titleCardPreviewDesiredKey = key;
  const cached = titleCardPreviewCache.get(key);
  if (cached) {
    if (titleCardPreviewPendingKey && titleCardPreviewPendingKey !== key) titleCardPreviewAbort?.abort();
    drawCurrentTitleCardPreview(cached);
    return;
  }
  if (titleCardPreviewPendingKey === key) return;

  clearTimeout(titleCardPreviewTimer);
  if (titleCardPreviewPendingKey && titleCardPreviewPendingKey !== key) titleCardPreviewAbort?.abort();
  titleCardPreviewTimer = setTimeout(async () => {
    titleCardPreviewTimer = null;
    if (titleCardPreviewDesiredKey !== key || titleCardPreviewPendingKey === key) return;
    const controller = new AbortController();
    titleCardPreviewAbort = controller;
    titleCardPreviewPendingKey = key;
    const time = estimatedTitleCardSourceTime(titleCardPart);
    const url = BASE + '/preview?index=0&time=' + encodeURIComponent(time) + '&fit=' + encodeURIComponent(fit);
    try {
      const response = await fetch(url, {signal: controller.signal, cache: 'force-cache'});
      if (!response.ok) throw new Error('Preview request failed.');
      const image = await imageFromBlob(await response.blob());
      rememberTitleCardPreview(key, image);
      if (titleCardPreviewDesiredKey === key) drawCurrentTitleCardPreview(image);
    } catch (error) {
      if (error?.name !== 'AbortError' && titleCardPreviewDesiredKey === key) {
        const fallback = {...titleCardPartRecord(titleCardPart, false), backgroundMode:'solid', solidColor:'#000000'};
        const fitNow = effectiveClip(state.videos[0].id).fit || 'fit';
        TitleCardTools.renderTitleCardPreview($('titleCardPreview'), $('titleCardPreview'), fitNow, fallback, titleCardPart, titleCardSourceName());
      }
    } finally {
      if (titleCardPreviewPendingKey === key) titleCardPreviewPendingKey = '';
      if (titleCardPreviewAbort === controller) titleCardPreviewAbort = null;
    }
  }, 180);
}
function updateTitleCardNavState() {
  $('titleCardPartSelect').value = String(titleCardPart);
  $('titleCardPartLabel').textContent = `of ${titleCardEstimatedParts}`;
  const enabled = !!$('partTitleScreens').checked && !state?.converting;
  $('titleCardPrev').disabled = !enabled || titleCardPart <= 1;
  $('titleCardNext').disabled = !enabled || titleCardPart >= titleCardEstimatedParts;
}
function setTitleCardPart(part, force = false) {
  const nextPart = Math.max(1, Math.min(titleCardEstimatedParts, Number(part) || 1));
  const changed = nextPart !== titleCardPart;
  titleCardPart = nextPart;
  updateTitleCardNavState();
  if (changed || force) loadTitleCardFields();
}
function updateTitleCardSection(estimateResult) {
  const section = $('titleCardSection');
  const wasVisible = !section.classList.contains('hidden');
  const visible = state?.videos?.length === 1 && mediaKind(state.videos[0]) === 'video' && Number(estimateResult?.estimatedParts || 1) > 1;
  section.classList.toggle('hidden', !visible);
  if (!visible) {
    titleCardSectionSignature = '';
    return;
  }
  ensureTitleCardProject();
  const previousParts = titleCardEstimatedParts;
  titleCardEstimatedParts = Math.max(2, Number(estimateResult.estimatedParts) || 2);
  titleCardProject.enabled = $('partTitleScreens').checked;
  titleCardProject.useShared = $('titleCardUseShared').checked;
  if ($('titleCardPartSelect').options.length !== titleCardEstimatedParts) {
    $('titleCardPartSelect').innerHTML = Array.from({length:titleCardEstimatedParts}, (_, index) => `<option value="${index + 1}">Part ${index + 1}</option>`).join('');
  }
  if (titleCardPart > titleCardEstimatedParts) titleCardPart = titleCardEstimatedParts;
  const clip = effectiveClip(state.videos[0].id);
  const signature = `${state.videos[0].id}|${titleCardEstimatedParts}|${clip.start}|${clip.end}|${clip.fit}`;
  const sourceChanged = signature !== titleCardSectionSignature;
  titleCardSectionSignature = signature;
  const enabled = $('partTitleScreens').checked;
  $('titleCardControls').classList.toggle('scope-muted', !enabled);
  for (const control of $('titleCardControls').querySelectorAll('input,select,button')) control.disabled = !enabled || !!state?.converting;
  for (const id of ['titleCardUseShared','titleCardOutline','titleCardAllowSkip','titleCardFade']) $(id).disabled = !enabled || !!state?.converting;
  $('titleCardCopyToAll').classList.toggle('hidden', $('titleCardUseShared').checked);
  if (!wasVisible || previousParts !== titleCardEstimatedParts || sourceChanged) setTitleCardPart(titleCardPart, true);
  else updateTitleCardNavState();
  updateTitleCardConditionalFields();
}

function updateMediaSettingsVisibility(){
  const kind=mediaKind();
  const isVideo=kind==='video',isAudio=kind==='audio',isImage=kind==='image';
  $('qualitySection')?.classList.toggle('hidden',!isVideo);
  $('videoSection')?.classList.toggle('hidden',!isVideo);
  $('musicSection')?.classList.toggle('hidden',!isAudio);
  $('imageSection')?.classList.toggle('hidden',!isImage);
  $('colorSection')?.classList.toggle('hidden',!isVideo);
  $('extremeSection')?.classList.toggle('media-hidden',!isVideo);
  $('smartAnalyze')?.classList.toggle('hidden',!isVideo);
  for(const id of ['fps','compression','paletteMode','ditherMode']) if($(id)){ const off=!isVideo; $(id).dataset.mediaDisabled=off?'1':'0'; $(id).disabled=off||!!state?.converting||$(id).dataset.scopeDisabled==='1'; }
  for(const id of ['start','end','speed']) if($(id)){ const off=!isVideo; $(id).dataset.mediaDisabled=off?'1':'0'; $(id).disabled=off||!!state?.converting||$(id).dataset.scopeDisabled==='1'; }
  if($('fit')) { const off=!isVideo; $('fit').dataset.mediaDisabled=off?'1':'0'; $('fit').disabled=off||!!state?.converting||$('fit').dataset.scopeDisabled==='1'; }
  $('audioSection')?.classList.toggle('hidden',isImage);
  $('timelinePanel')?.classList.toggle('hidden',isImage);
  $('previewEnd')?.classList.toggle('hidden',isImage);
  if(isAudio && $('audio')) $('audio').value=$('audio').value==='none'?'mix':$('audio').value;
  syncMediaAliasControls();
}

function currentSettingsSource(){
  return editScope === 'project' ? projectDefaults : effectiveClip(selectedID);
}
function settingsAreLocked(){
  const config=clipConfigs[selectedID];
  return editScope==='clip' && !!config?.useProject;
}
function syncMediaAliasControls(){
  const source=currentSettingsSource();
  const locked=settingsAreLocked() || !!state?.converting;
  const kind=mediaKind();
  if(kind==='audio'){
    const config=clipConfigs[selectedID];
    if($('musicTitle')) { $('musicTitle').value=config?.musicTitle ?? defaultMusicTitle(selectedVideo()); $('musicTitle').disabled=!!state?.converting; }
    if($('musicArtist')) { $('musicArtist').value=config?.musicArtist ?? defaultMusicArtist(selectedVideo()); $('musicArtist').disabled=!!state?.converting; }
    if($('musicArtworkMode')) {
      const mode=['default','embedded','custom'].includes(config?.musicArtworkMode) ? config.musicArtworkMode : 'embedded';
      const preset=normalizeMusicArtworkPreset(config?.musicArtworkPreset);
      $('musicArtworkMode').value=mode;
      $('musicArtworkMode').disabled=!!state?.converting;
      $('musicArtworkPresetRow')?.classList.toggle('hidden',mode==='custom');
      $('musicArtworkCustomRow')?.classList.toggle('hidden',mode!=='custom');
      for(const button of document.querySelectorAll('.music-artwork-preset')) {
        button.setAttribute('aria-checked',button.dataset.preset===preset?'true':'false');
        button.disabled=!!state?.converting;
      }
      if($('musicArtworkInfo')) $('musicArtworkInfo').textContent = mode==='default'
        ? 'The selected built-in artwork will be stored with this track.'
        : mode==='embedded'
          ? 'Embedded artwork is used when available. If none is present, the selected default preset is used.'
          : 'Your custom image is stored with this track after being cropped to the GBA screen.';
      if($('musicArtworkCustomStatus') && mode==='custom') $('musicArtworkCustomStatus').textContent = config?.musicArtworkCustom ? 'Custom artwork ready — 240×160.' : 'Choose an image. It will be cropped to 240×160 for the GBA.';
    }
    for(const [id,key] of [['musicStart','start'],['musicEnd','end'],['musicSpeed','speed']]) if($(id)){ $(id).value=source[key] ?? ''; $(id).disabled=locked; }
    if($('musicSeekSeconds')) { $('musicSeekSeconds').value=String([3,5,10,15].includes(Number(config?.musicSeekSeconds))?Number(config.musicSeekSeconds):5); $('musicSeekSeconds').disabled=!!state?.converting; }
  }
  if(kind==='image'){
    if($('imageFit')) { $('imageFit').value=source.fit || 'fit'; $('imageFit').disabled=locked; }
    const seconds=Number(source.imageSeconds);
    const enabled=Number.isFinite(seconds) && seconds>0;
    if($('imageSlideshow')) { $('imageSlideshow').checked=enabled; $('imageSlideshow').disabled=locked; }
    if($('imageSeconds')) {
      if(enabled) $('imageSeconds').value=seconds;
      else if(!Number.isFinite(Number($('imageSeconds').value)) || Number($('imageSeconds').value)<=0) $('imageSeconds').value=5;
      $('imageSeconds').disabled=locked || !enabled;
    }
    $('imageSlideshowDuration')?.classList.toggle('scope-muted',!enabled);
  }
}

function updateSplitControls() {
  const single = state.videos.length === 1;
  const isVideo = single && mediaKind(state.videos[0]) === 'video';
  $('splitVideoRow').classList.toggle('hidden', !isVideo);
  $('longSplitControls').classList.toggle('hidden', !isVideo || !$('splitVideo').checked);
  if(!isVideo) $('splitVideo').checked=false;
}

function updateOutputModes() {
  const select = $('outputMode');
  const current = select.value;
  const single = state.videos.length === 1;
  if (single) {
    select.innerHTML = '<option value="rom">Single ROM</option>';
    select.value = 'rom';
    if($('outputModeHint')) $('outputModeHint').textContent='A single item opens directly.';
  } else {
    select.innerHTML = '<option value="menu">One ROM — media menu</option><option value="batch">Separate ROMs in ZIP</option>';
    select.value = current === 'batch' ? 'batch' : 'menu';
    if($('outputModeHint')) $('outputModeHint').textContent='Every collection ROM starts in the media menu, even when all items are the same type.';
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
  refreshAudioTrackSelector();
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
  for (const key of ['start','end','speed','fit','audio','volume','loop','paletteMode','ditherMode','imageSeconds']) {
    const element = $(key);
    if (!element) continue;
    if (element.type === 'checkbox') element.checked = !!source[key]; else element.value = source[key];
    const gifLoop = key === 'loop' && isGIF();
    if (gifLoop) element.checked = true;
    const disabled = (editScope === 'clip' && config.useProject) || gifLoop;
    element.dataset.scopeDisabled = disabled ? '1' : '0';
    element.disabled = disabled || !!state?.converting;
  }
  for (const element of document.querySelectorAll('.project-control')) element.classList.toggle('scope-muted', editScope === 'clip');
  updateMediaSettingsVisibility();
  if (force) { syncTimeline(true); updatePreview(); }
}
$('editProjectDefaults').onclick = () => { editScope = 'project'; refreshScope(true); };
$('editSelectedClip').onclick = () => { editScope = 'clip'; refreshScope(true); };
$('useProject').onchange = () => {
  const config = clipConfigs[selectedID];
  if (!config) return;
  if (!config.useProject && $('useProject').checked) config.useProject = true;
  else if (config.useProject && !$('useProject').checked) { const audioTrack=config.audioTrack,musicTitle=config.musicTitle,musicArtist=config.musicArtist,musicArtworkMode=config.musicArtworkMode,musicArtworkPreset=config.musicArtworkPreset,musicArtworkCustom=config.musicArtworkCustom,musicSeekSeconds=config.musicSeekSeconds; Object.assign(config, cloneClip(projectDefaults), {useProject:false, title:config.title, audioTrack, musicTitle, musicArtist, musicArtworkMode, musicArtworkPreset, musicArtworkCustom, musicSeekSeconds}); }
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
$('audioTrack').addEventListener('change', () => {
  const config=clipConfigs[selectedID]; if (!config) return;
  config.audioTrack=Number($('audioTrack').value) || 0;
  if (audioURL) { URL.revokeObjectURL(audioURL); audioURL=''; $('audioPlayer').removeAttribute('src'); }
});
for (const id of ['start','end','speed','fit','audio','volume','loop','paletteMode','ditherMode','imageSeconds']) $(id).addEventListener('input', () => savePerClipField(id));
function saveAliasField(id,key){
  const element=$(id);
  if(!element || settingsAreLocked()) return;
  const value=element.type==='number'?Number(element.value):element.value;
  if(editScope==='project') projectDefaults[key]=value;
  else { const config=clipConfigs[selectedID]; if(!config || config.useProject) return; config[key]=value; }
  const source=$(key); if(source) source.value=value;
  $('preset').value='custom';
  estimate();
  if(['start','end','fit','speed'].includes(key)){ syncTimeline(false); updatePreview(); }
}
for(const [id,key] of [['musicStart','start'],['musicEnd','end'],['musicSpeed','speed'],['imageFit','fit']]) $(id)?.addEventListener('input',()=>saveAliasField(id,key));
$('musicTitle')?.addEventListener('input',()=>{ const config=clipConfigs[selectedID]; if(config){ config.musicTitle=$('musicTitle').value.slice(0,28); estimate(); } });
$('musicArtist')?.addEventListener('input',()=>{ const config=clipConfigs[selectedID]; if(config){ config.musicArtist=$('musicArtist').value.slice(0,28); estimate(); } });
$('musicSeekSeconds')?.addEventListener('change',()=>{ const config=clipConfigs[selectedID]; if(config){ const value=Number($('musicSeekSeconds').value); config.musicSeekSeconds=[3,5,10,15].includes(value)?value:5; estimate(); } });
function initializeMusicArtworkPresets(){
  const host=$('musicArtworkPresets');
  if(!host || host.children.length) return;
  host.innerHTML=AUDIO_ARTWORK_PRESETS.map((preset,index)=>'<button type="button" class="music-artwork-preset" role="radio" aria-checked="false" data-preset="'+preset+'" title="Preset '+String(index+1).padStart(2,'0')+'"><img alt="Preset '+String(index+1).padStart(2,'0')+'" src="'+musicArtworkPresetURL(preset)+'"><span>'+String(index+1).padStart(2,'0')+'</span></button>').join('');
  for(const button of host.querySelectorAll('.music-artwork-preset')) button.addEventListener('click',()=>{
    const config=clipConfigs[selectedID]; if(!config || mediaKind()!=='audio' || state?.converting) return;
    config.musicArtworkPreset=normalizeMusicArtworkPreset(button.dataset.preset);
    syncMediaAliasControls(); lastPreviewKey=''; lastThumbKey=''; updatePreview(); renderTimelineThumbs(); estimate();
  });
}
async function customArtworkToDataURL(file){
  if(!file) throw new Error('Choose an image first.');
  if(file.size > 32*1024*1024) throw new Error('Custom artwork image is too large.');
  const url=URL.createObjectURL(file);
  try {
    const image=await new Promise((resolve,reject)=>{ const img=new Image(); img.onload=()=>resolve(img); img.onerror=()=>reject(new Error('Could not decode custom artwork image.')); img.src=url; });
    const canvas=document.createElement('canvas'); canvas.width=240; canvas.height=160;
    const ctx=canvas.getContext('2d',{alpha:false});
    ctx.fillStyle='#000'; ctx.fillRect(0,0,240,160);
    const width=image.naturalWidth||image.width,height=image.naturalHeight||image.height;
    if(!width||!height) throw new Error('Custom artwork image has invalid dimensions.');
    const scale=Math.max(240/width,160/height),drawW=width*scale,drawH=height*scale;
    ctx.drawImage(image,(240-drawW)/2,(160-drawH)/2,drawW,drawH);
    return canvas.toDataURL('image/png');
  } finally { URL.revokeObjectURL(url); }
}
$('musicArtworkMode')?.addEventListener('change',()=>{
  const config=clipConfigs[selectedID]; if(!config || mediaKind()!=='audio') return;
  config.musicArtworkMode=$('musicArtworkMode').value;
  syncMediaAliasControls(); lastPreviewKey=''; lastThumbKey=''; updatePreview(); renderTimelineThumbs(); estimate();
});
$('musicArtworkCustom')?.addEventListener('change',async()=>{
  const file=$('musicArtworkCustom').files?.[0]; if(!file) return;
  const config=clipConfigs[selectedID]; if(!config || mediaKind()!=='audio') return;
  try {
    $('musicArtworkCustomStatus').textContent='Preparing '+file.name+'…';
    config.musicArtworkCustom=await customArtworkToDataURL(file);
    config.musicArtworkMode='custom';
    $('musicArtworkCustom').value='';
    syncMediaAliasControls(); lastPreviewKey=''; lastThumbKey=''; updatePreview(); renderTimelineThumbs(); estimate();
  } catch(error) { $('musicArtworkCustomStatus').textContent=error.message; alert(error.message); }
});
$('imageSlideshow')?.addEventListener('change',()=>{
  if(settingsAreLocked()) return;
  const source=currentSettingsSource();
  let seconds=Number(source.imageSeconds);
  if($('imageSlideshow').checked){
    if(!Number.isFinite(seconds)||seconds<=0) seconds=Number($('imageSeconds').dataset.lastSeconds)||5;
  }else{
    if(Number.isFinite(seconds)&&seconds>0) $('imageSeconds').dataset.lastSeconds=String(seconds);
    seconds=0;
  }
  if(editScope==='project') projectDefaults.imageSeconds=seconds;
  else { const config=clipConfigs[selectedID]; if(config&&!config.useProject) config.imageSeconds=seconds; }
  syncMediaAliasControls();
  estimate();
});
$('imageSeconds')?.addEventListener('input',()=>{ const value=Number($('imageSeconds').value); if(value>0) $('imageSeconds').dataset.lastSeconds=String(value); syncMediaAliasControls(); });
for (const id of ['fps','seekSeconds','compression','normalize','limiter','maxPartDuration','chapterAware','partTitleScreens','resumeLongSplit']) $(id).addEventListener('input', () => { $('preset').value = 'custom'; estimate(); });
$('splitVideo').addEventListener('change', () => { updateSplitControls(); $('preset').value = 'custom'; estimate(); });
$('outputMode').onchange = () => { updateOutputModes(); estimate(); };
function updateSplitBudgetLabel() { $('splitBudgetValue').textContent = $('splitBudget').value + ' MiB'; }
$('splitBudget').addEventListener('input', () => { updateSplitBudgetLabel(); $('preset').value = 'custom'; estimate(); });
for (const button of document.querySelectorAll('.split-preset')) button.onclick = () => { $('splitBudget').value = button.dataset.size; updateSplitBudgetLabel(); estimate(); };
updateSplitBudgetLabel();
$('romTitle').addEventListener('input', () => { romTitleAuto = false; });

function updateTitlePreview(invalid = false, unsupported = []) {
  const config = clipConfigs[selectedID];
  if (!config) return;
  const canvas = $('titlePreview'), context = canvas.getContext('2d');
  context.imageSmoothingEnabled = false;
  context.fillStyle = '#000'; context.fillRect(0,0,canvas.width,canvas.height);
  context.fillStyle = '#ffdd00';
  const scale = 4, startX = 8, startY = 6;
  [...config.title.padEnd(12, ' ')].slice(0,12).forEach((char,index) => {
    const bits = GBAText.glyphBits(char);
    for (let row=0; row<5; row++) for (let col=0; col<3; col++) {
      const bit = 14 - (row*3+col);
      if (bits & (1<<bit)) context.fillRect(startX + index*16 + col*scale, startY + row*scale, scale, scale);
    }
  });
  $('titleCount').textContent = GBAText.glyphLength(config.title) + '/12';
  const duplicates = Object.entries(clipConfigs).filter(([id,c]) => id !== selectedID && c.title === config.title && config.title).length;
  const warning = invalid ? `Unsupported GBA characters: ${unsupported.join(' ') || '?'}. They were replaced.` : duplicates ? 'Another clip uses the same menu title.' : '';
  $('titleWarning').textContent = warning;
  $('titleWarning').className = 'field full validation ' + (warning ? 'warning' : '');
}
$('menuTitle').oninput = () => {
  const result = sanitizeMenuTitle($('menuTitle').value);
  $('menuTitle').value = result.value;
  clipConfigs[selectedID].title = result.value || 'GBA VIDEO';
  updateTitlePreview(result.invalid, result.unsupported);
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
  const artSource = mediaKind(video)==='audio' ? musicArtworkPreviewSource(video) : '';
  const artKey = mediaKind(video)==='audio' ? '|' + (clipConfigs[video.id]?.musicArtworkMode || 'embedded') + '|' + normalizeMusicArtworkPreset(clipConfigs[video.id]?.musicArtworkPreset) + '|' + (clipConfigs[video.id]?.musicArtworkCustom || '').length : '';
  const key = video.id + '|' + fit + '|' + video.info.duration.toFixed(3) + artKey;
  if (key === lastThumbKey) return;
  lastThumbKey = key;
  const count = 6;
  $('timelineThumbs').innerHTML = Array.from({length:count}, (_,i) => {
    if (artSource) return '<img alt="" src="' + artSource.replace(/&/g,'&amp;').replace(/"/g,'&quot;') + '">';
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
  const artworkSource = mediaKind(video)==='audio' ? musicArtworkPreviewSource(video) : '';
  const artworkKey = mediaKind(video)==='audio' ? [(clipConfigs[video.id]?.musicArtworkMode||'embedded'),normalizeMusicArtworkPreset(clipConfigs[video.id]?.musicArtworkPreset),(clipConfigs[video.id]?.musicArtworkCustom||'').length].join('|') : '';
  const key = [video.id,time.toFixed(3),fit,artworkKey].join('|');
  if (key === lastPreviewKey && $('previewImage').src) return;
  lastPreviewKey = key;
  const image = $('previewImage');
  image.onerror = () => image.removeAttribute('src');
  image.src = artworkSource || (BASE + '/preview?index=' + index + '&time=' + encodeURIComponent(time) + '&fit=' + fit + '&key=' + encodeURIComponent(key));
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
  best:{fps:'smooth',compression:'delta',normalize:true,limiter:true,audioQuality:'pcm',defaults:{fit:'fit',audio:'mix',paletteMode:'scene',ditherMode:'error'}},
  balanced:{fps:'balanced',compression:'delta',normalize:false,limiter:true,audioQuality:'pcm',defaults:{fit:'fit',audio:'mix',paletteMode:'shared',ditherMode:'ordered'}},
  long:{fps:'compact',compression:'delta',normalize:false,limiter:true,audioQuality:'pcm',defaults:{fit:'fit',audio:'mix',paletteMode:'shared',ditherMode:'ordered'}},
  small:{fps:'compact',compression:'delta',normalize:false,limiter:false,audioQuality:'pcm',defaults:{fit:'fit',audio:'none',paletteMode:'shared',ditherMode:'off'}},
  extreme:{fps:'balanced',compression:'delta',normalize:false,limiter:true,audioQuality:'auto',defaults:{fit:'fit',audio:'mix',paletteMode:'scene',ditherMode:'ordered'}}
};
function updateExtremeUI() {
  const extreme = $('preset').value === 'extreme';
  $('extremeSection')?.classList.toggle('hidden', !extreme);
  $('audioQuality').disabled = !extreme || !!state?.converting;
  if (!extreme) {
    $('audioQuality').value = 'pcm';
    smartAnalysis = null;
    $('smartResults')?.classList.add('hidden');
    if ($('smartStatus')) $('smartStatus').textContent = 'Not analyzed';
  }
}
$('preset').onchange = () => {
  const preset = PRESETS[$('preset').value];
  updateExtremeUI();
  if (!preset) return;
  $('fps').value = preset.fps; $('compression').value = preset.compression; $('normalize').checked = preset.normalize; $('limiter').checked = preset.limiter;
  $('audioQuality').value = preset.audioQuality || 'pcm';
  Object.assign(projectDefaults, preset.defaults);
  refreshScope(true); estimate();
};

function globalValues(includeMenuTheme = true) {
  return {
    preset:$('preset').value, audioQuality:$('audioQuality').value, smartTargetMiB:Number($('smartTarget').value), smartPriority:$('smartPriority').value,
    fps:$('fps').value, seekSeconds:Number($('seekSeconds').value), compression:$('compression').value,
    normalize:$('normalize').checked, limiter:$('limiter').checked, resume:$('resume').checked,
    romTitle:$('romTitle').value, outputMode:$('outputMode').value,
    splitVideo:$('splitVideo').checked, splitBudgetMiB:Number($('splitBudget').value),
    maxPartDuration:$('maxPartDuration').value.trim() || '0',
    chapterAware:$('chapterAware').checked, partTitleScreens:$('partTitleScreens').checked,
    resumeLongSplit:$('resumeLongSplit').checked,
    titleCards:serializeTitleCards(),
    menuBackground:$('menuBackground')?.value || 'ocean-wave-animated',
    menuUIColor:$('menuUIColor')?.value || '#FFFFFF',
    menuSelectionColor:$('menuSelectionColor')?.value || '#FFDE00',
    menuOutline:!!$('menuOutline')?.checked,
    menuOutlineColor:$('menuOutlineColor')?.value || '#000000',
    menuTheme:includeMenuTheme && $('outputMode').value === 'menu' ? serializedMenuTheme() : null
  };
}
function values() {
  const global = globalValues();
  return {
    ...global, ...projectDefaults,
    clips: state.videos.map(video => {
      const config = clipConfigs[video.id];
      return {id:video.id,title:config.title,useProject:config.useProject,start:config.start,end:config.end,speed:Number(config.speed),fit:config.fit,audio:config.audio,audioTrack:Number(config.audioTrack)||0,volume:Number(config.volume),loop:isGIF(video)||!!config.loop,paletteMode:config.paletteMode,ditherMode:config.ditherMode,imageSeconds:Number.isFinite(Number(config.imageSeconds))?Number(config.imageSeconds):5,musicTitle:config.musicTitle||'',musicArtist:config.musicArtist||'',musicArtworkMode:config.musicArtworkMode||'embedded',musicArtworkPreset:normalizeMusicArtworkPreset(config.musicArtworkPreset),musicArtworkCustom:config.musicArtworkCustom||'',musicSeekSeconds:[3,5,10,15].includes(Number(config.musicSeekSeconds))?Number(config.musicSeekSeconds):5};
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
    const info=video.info, kind=info.kind||'video', clip=modelEffective(model,video.id);
    if(kind==='image'){
      videoBytes += 240*160*2;
      frames += 1;
      sourceDuration += Math.max(0,Number(clip.imageSeconds)||0);
      continue;
    }
    const start = Math.max(0, parseClock(clip.start) || 0);
    let end = clip.end?.trim() ? parseClock(clip.end) : info.duration;
    if (!Number.isFinite(end)) end = info.duration;
    end = Math.min(info.duration, end);
    if (end <= start || !Number.isFinite(Number(clip.speed)) || clip.speed <= 0) return {error:'Check trim settings.'};
    const sourceClipDuration = end - start;
    sourceDuration += sourceClipDuration;
    const displayDuration = sourceClipDuration / Number(clip.speed);
    const frameCount = Math.max(1, Math.ceil(displayDuration * fps));
    frames += frameCount;
    if(kind==='audio'){
      videoBytes += 240*160*2 + 44;
      indexBytes += frameCount*4; // audio seek table
    }else{
      const compressionFactor = model.global.compression === 'delta' ? (model.global.preset === 'extreme' ? 0.61 : 0.68) : 1;
      videoBytes += frameCount * 9600 * compressionFactor;
      if (model.global.compression === 'delta') indexBytes += frameCount * 8;
      const palettes = clip.paletteMode === 'scene' ? Math.max(1, Math.ceil(frameCount / 60)) : 1;
      paletteBytes += palettes * 512 + (palettes > 1 ? frameCount * 2 : 0);
    }
    if (clip.audio !== 'none' && info.audioStreams) {
      const requested = model.global.preset === 'extreme' ? model.global.audioQuality : 'pcm';
      const pcmBytes = displayDuration * 16384;
      const targetBytes = Math.max(1, Math.min(32, Number(model.global.smartTargetMiB) || 32)) * MIB;
      const codec = requested === 'auto' && model.global.preset === 'extreme' && pcmBytes > targetBytes / 3 ? 'adpcm' : (requested === 'adpcm' ? 'adpcm' : 'pcm');
      audioBytes += pcmBytes * (codec === 'adpcm' ? 0.505 : 1) + (kind==='video'?frameCount*4:0);
    }
  }
  const bytes = Math.ceil(player + videoBytes + audioBytes + paletteBytes + indexBytes);
  let cartridge = 1 << 20;
  while (cartridge < bytes && cartridge < ROM_LIMIT) cartridge *= 2;
  return {bytes, cartridge, frames, fps, sourceDuration, breakdown:{player,video:videoBytes,audio:audioBytes,palettes:paletteBytes,indexes:indexBytes}};
}
function estimate() {
  const model = modelSnapshot();
  const result = estimateModel(model);
  if (result.error) { $('estimate').textContent = result.error; $('titleCardSection')?.classList.add('hidden'); return result; }
  const single = state.videos.length === 1;
  const singleVideo = single && mediaKind(state.videos[0]) === 'video';
  const manualSplit = singleVideo && !!model.global.splitVideo;
  const budgetMiB = manualSplit ? Math.max(1, Math.min(32, Number(model.global.splitBudgetMiB) || 31)) : 32;
  const budgetBytes = budgetMiB * MIB;
  const overhead = 32768 + 96 + 512;
  const usable = Math.max(1, budgetBytes - overhead);
  const payload = Math.max(1, result.bytes - overhead);
  let estimatedParts = Math.max(1, Math.ceil(payload / usable));
  const maxPartSeconds = manualSplit ? parsePartDuration(model.global.maxPartDuration) : 0;
  if (!Number.isFinite(maxPartSeconds)) { $('estimate').innerHTML = '<b class="estimate-over">Maximum duration must be 0 or MM:SS, for example 1:05.</b>'; return {...result,error:'invalid maximum part duration'}; }
  if (single && maxPartSeconds > 0) estimatedParts = Math.max(estimatedParts, Math.ceil(result.sourceDuration / maxPartSeconds));
  let automaticSplit = singleVideo && estimatedParts > 1;
  if (automaticSplit && $('partTitleScreens')?.checked) {
    for (let pass=0; pass<2; pass++) {
      const withCards=payload + TitleCardTools.TITLE_CARD_BYTES * estimatedParts;
      estimatedParts=Math.max(estimatedParts,Math.ceil(withCards/usable));
    }
    result.bytes += TitleCardTools.TITLE_CARD_BYTES * estimatedParts;
  }
  automaticSplit = singleVideo && estimatedParts > 1;
  const hasNonVideo=state.videos.some(v=>mediaKind(v)!=='video');
  $('optimize').classList.toggle('hidden', automaticSplit || hasNonVideo);
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
    '<br>Visual media ' + (result.breakdown.video/MIB).toFixed(2) + ' MiB • Audio ' + (result.breakdown.audio/MIB).toFixed(2) + ' MiB • Palettes/indexes ' + ((result.breakdown.palettes+result.breakdown.indexes)/MIB).toFixed(2) + ' MiB' + splitNote;
  result.estimatedParts = estimatedParts;
  result.splitBudgetMiB = budgetMiB;
  updateTitleCardSection(result);
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


function formatSmartBytes(bytes) { return (Number(bytes || 0) / MIB).toFixed(2) + ' MiB'; }
function renderSmartAnalysis(result) {
  smartAnalysis = result;
  const cards = [result.recommended, ...(result.alternatives || [])].map((candidate, index) => {
    const cls = index === 0 ? 'smart-result-card recommended' : 'smart-result-card';
    const fit = candidate.fitsTarget ? '<span class="pill">Fits target</span>' : '<span class="estimate-over">Exceeds target</span>';
    return `<article class="${cls}"><h4>${escapeHTML(candidate.label)}</h4><p><b>${formatSmartBytes(candidate.estimatedMinBytes)}–${formatSmartBytes(candidate.estimatedMaxBytes)}</b> ${fit}</p><p>Visual ${candidate.visualQuality}/100 · Motion ${candidate.motionQuality}/100 · Stability ${candidate.temporalStability}/100</p><p>${candidate.fps.toFixed(2)} FPS · ${escapeHTML(candidate.paletteMode)} palette · ${escapeHTML(candidate.ditherMode)} dithering · ${candidate.audioCodec === 'adpcm' ? 'Compact ADPCM' : 'Standard PCM'}</p><p class="tiny">${escapeHTML(candidate.summary || '')}</p><div class="smart-result-actions"><button class="btn compact smart-apply" type="button" data-id="${escapeHTML(candidate.id)}">Apply</button></div></article>`;
  }).join('');
  $('smartResults').innerHTML = `<p class="tiny">Confidence: <b>${escapeHTML(result.confidence)}</b>. Samples: ${(result.samples || []).map(sample => `${escapeHTML(sample.kind)} ${fmt(sample.time)}`).join(' · ')}</p>${cards}`;
  $('smartResults').classList.remove('hidden');
  $('smartStatus').textContent = `Recommended ${result.recommended.label} · ${formatSmartBytes(result.recommended.estimatedBytes)}`;
  for (const button of $('smartResults').querySelectorAll('.smart-apply')) button.onclick = () => applySmartCandidate(button.dataset.id);
}
function applySmartCandidate(id) {
  const candidate = smartAnalysis?.candidates?.find(item => item.id === id) || smartAnalysis?.recommended;
  if (!candidate) return;
  const fpsName = ({4:'smooth',5:'balanced',6:'classic',8:'compact'})[candidate.vblanks] || 'balanced';
  $('preset').value = 'extreme';
  $('fps').value = fpsName;
  $('compression').value = 'delta';
  $('audioQuality').value = candidate.audioCodec || 'pcm';
  projectDefaults.paletteMode = candidate.paletteMode || 'scene';
  projectDefaults.ditherMode = candidate.ditherMode || 'ordered';
  updateExtremeUI();
  refreshScope(true);
  estimate();
  $('smartStatus').textContent = `${candidate.label} settings applied`;
}
async function runSmartAnalysis() {
  if ($('preset').value !== 'extreme') return;
  smartAbort?.abort();
  smartAbort = new AbortController();
  $('smartAnalyze').disabled = true;
  $('smartCancel').classList.remove('hidden');
  $('smartStatus').textContent = 'Scanning representative scenes…';
  try {
    const response = await fetch(BASE + '/smart-analyze', {method:'POST',headers:headers({'Content-Type':'application/json'}),body:JSON.stringify(values()),signal:smartAbort.signal});
    if (!response.ok) { const body = await response.json(); throw new Error(body.error || 'Analysis failed'); }
    renderSmartAnalysis(await response.json());
  } catch (error) {
    if (error.name !== 'AbortError') { $('smartStatus').textContent = error.message; alert(error.message); }
  } finally {
    $('smartAnalyze').disabled = false;
    $('smartCancel').classList.add('hidden');
    smartAbort = null;
  }
}
$('smartAnalyze').onclick = runSmartAnalysis;
$('smartCancel').onclick = () => smartAbort?.abort();
$('smartTarget').addEventListener('change', () => { smartAnalysis = null; $('smartStatus').textContent='Settings changed — analyze again'; });
$('smartPriority').addEventListener('change', () => { smartAnalysis = null; $('smartStatus').textContent='Settings changed — analyze again'; });
$('audioQuality').addEventListener('change', () => { smartAnalysis = null; if ($('preset').value === 'extreme') $('smartStatus').textContent='Audio choice changed — analyze again'; estimate(); });

function applyPendingProject() {
  const settings = pendingProject;
  pendingProject = null;
  projectDefaults = cloneClip({start:settings.start,end:settings.end,speed:settings.speed,fit:settings.fit || 'fit',audio:settings.audio,volume:settings.volume,loop:settings.loop,paletteMode:settings.paletteMode,ditherMode:settings.ditherMode,imageSeconds:Number.isFinite(Number(settings.imageSeconds))?Number(settings.imageSeconds):5});
  for (const key of ['fps','compression','outputMode']) if (settings[key]) $(key).value = settings[key];
  $('preset').value = settings.preset || 'custom';
  $('audioQuality').value = settings.audioQuality || 'pcm';
  $('smartTarget').value = String(settings.smartTargetMiB || 32);
  $('smartPriority').value = settings.smartPriority || 'balanced';
  updateExtremeUI();
  $('seekSeconds').value = settings.seekSeconds || 5;
  $('normalize').checked = !!settings.normalize; $('limiter').checked = !!settings.limiter; $('resume').checked = !!settings.resume;
  $('splitVideo').checked = !!settings.splitVideo;
  $('splitBudget').value = settings.splitBudgetMiB || 31;
  $('maxPartDuration').value = settings.maxPartDuration || partDurationValue((Number(settings.maxPartMinutes) || 0) * 60);
  $('chapterAware').checked = settings.chapterAware !== false; $('partTitleScreens').checked = settings.titleCards?.enabled ?? (settings.partTitleScreens !== false); $('resumeLongSplit').checked = settings.resumeLongSplit !== false;
  titleCardProject = settings.titleCards ? JSON.parse(JSON.stringify(settings.titleCards)) : null;
  titleCardProjectSource = titleCardProject ? titleCardSourceName() : "";
  if (titleCardProject) $('titleCardUseShared').checked = titleCardProject.useShared !== false;
  updateSplitBudgetLabel(); updateSplitControls();
  $('romTitle').value = settings.romTitle || ''; romTitleAuto = false;
  if ($('menuBackground')) {
    $('menuBackground').value = settings.menuBackground || settings.menuTheme?.id || 'ocean-wave-animated';
    restoreMenuColors(settings);
    $('menuOutline').checked = settings.menuOutline !== false;
    customMenuLoadToken++; customMenuSourceFile=null; customMenuSourceIsVideo=false; $('customMenuVideoTiming')?.classList.add('hidden'); $('customMenuBackground').value='';
    customMenuTheme = (($('menuBackground').value === 'custom') && settings.menuTheme) ? MenuThemeTools.deserializeTheme(settings.menuTheme) : null;
    rebuildMenuTheme();
  }
  clipConfigs = {};
  for (const clip of settings.clips || []) clipConfigs[clip.id] = {title:clip.title || 'GBA VIDEO',useProject:clip.useProject !== false,start:clip.start || '0:00',end:clip.end || '',speed:clip.speed || 1,fit:clip.fit || 'fit',audio:clip.audio || 'mix',audioTrack:Number.isInteger(clip.audioTrack)?clip.audioTrack:0,volume:Number.isFinite(clip.volume)?clip.volume:100,loop:!!clip.loop,paletteMode:clip.paletteMode || 'shared',ditherMode:clip.ditherMode || 'ordered',imageSeconds:Number.isFinite(Number(clip.imageSeconds))?Number(clip.imageSeconds):5,musicTitle:clip.musicTitle ?? null,musicArtist:clip.musicArtist ?? null,musicArtworkMode:['default','embedded','custom'].includes(clip.musicArtworkMode)?clip.musicArtworkMode:'embedded',musicArtworkPreset:normalizeMusicArtworkPreset(clip.musicArtworkPreset),musicArtworkCustom:clip.musicArtworkCustom||'',musicSeekSeconds:[3,5,10,15].includes(Number(clip.musicSeekSeconds))?Number(clip.musicSeekSeconds):([3,5,10,15].includes(Number(settings.seekSeconds))?Number(settings.seekSeconds):5)};
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
initializeMusicArtworkPresets();
$('saveProject').onclick = saveProject;
$('openProject').onclick = openProject;
$('openProjectWelcome').onclick = event => { event.stopPropagation(); openProject(); };

if ($('menuBackground')) {
  for (const [inputID,label] of [['menuUIColor','UI text colour'],['menuSelectionColor','Selection colour'],['menuOutlineColor','Outline colour']]) {
    MenuThemeTools.setupGBAColorPicker($(inputID),{label});
  }
  updateMenuColorReadouts();
  rebuildMenuTheme();
  stopMenuPreview = MenuThemeTools.startPreview($('menuPreview'), () => activeMenuTheme, menuStyleSettings);
  $('menuBackground').addEventListener('change', () => { rebuildMenuTheme(); estimate(); });
  for (const [inputID, fallback] of [['menuUIColor','#FFFFFF'],['menuSelectionColor','#FFDE00'],['menuOutlineColor','#000000']]) {
    $(inputID).addEventListener('input', () => { updateMenuColorReadouts(); rebuildMenuTheme(); });
    $(inputID).addEventListener('change', () => { snapMenuColor(inputID,fallback); rebuildMenuTheme(); estimate(); });
  }
  $('menuOutline').addEventListener('change', () => { rebuildMenuTheme(); estimate(); });
  $('customMenuBackground').addEventListener('change', event => loadCustomMenuBackground(event.target.files?.[0]));
  for(const id of ['customMenuVideoStart','customMenuVideoDuration']) $(id)?.addEventListener('change',()=>{ if(customMenuSourceIsVideo&&customMenuSourceFile) loadCustomMenuBackground(customMenuSourceFile); });
  $('clearCustomMenuBackground').onclick = () => { customMenuLoadToken++; customMenuTheme = null; customMenuSourceFile=null; customMenuSourceIsVideo=false; $('customMenuBackground').value = ''; $('customMenuVideoTiming')?.classList.add('hidden'); $('menuBackgroundStatus').textContent = 'Choose a PNG, JPG, WebP, GIF or video.'; rebuildMenuTheme(); estimate(); };
}


if ($('titleCardPreview')) {
  for (const [inputID,label] of [['titleCardTextColor','Title text colour'],['titleCardOutlineColor','Title outline colour'],['titleCardSubtitleTextColor','Subtitle text colour'],['titleCardSubtitleOutlineColor','Subtitle outline colour'],['titleCardSolidColor','Title-card background colour']]) {
    MenuThemeTools.setupGBAColorPicker($(inputID),{label});
  }
  updateTitleCardColorReadouts();
  $('titleCardPrev').onclick=()=>setTitleCardPart(titleCardPart-1);
  $('titleCardNext').onclick=()=>setTitleCardPart(titleCardPart+1);
  $('titleCardPartSelect').onchange=()=>setTitleCardPart(Number($('titleCardPartSelect').value));
  $('titleCardUseShared').onchange=()=>{
    ensureTitleCardProject(); titleCardProject.useShared=$('titleCardUseShared').checked;
    $('titleCardCopyToAll').classList.toggle('hidden',titleCardProject.useShared);
    loadTitleCardFields(); estimate();
  };
  $('titleCardCopyToAll').onclick=()=>{
    const source=JSON.parse(JSON.stringify(titleCardPartRecord(titleCardPart,false)));
    titleCardProject.parts=Array.from({length:titleCardEstimatedParts},(_,index)=>({part:index+1,settings:JSON.parse(JSON.stringify(source))}));
    loadTitleCardFields(); estimate();
  };
  $('partTitleScreens').addEventListener('change',()=>{ ensureTitleCardProject(); titleCardProject.enabled=$('partTitleScreens').checked; estimate(); });
  for (const id of ['titleCardTitle','titleCardSubtitle','titleCardBackground','titleCardFrameOffset','titleCardDarkness','titleCardSolidColor','titleCardTextColor','titleCardSubtitleTextColor','titleCardOutline','titleCardOutlineColor','titleCardSubtitleOutlineColor','titleCardAlignment','titleCardSubtitleAlignment','titleCardTextSize','titleCardSubtitleTextSize','titleCardStartMode','titleCardDuration','titleCardAllowSkip','titleCardFade']) {
    $(id).addEventListener('input',saveTitleCardFields);
    $(id).addEventListener('change',()=>{
      if (['titleCardSolidColor','titleCardTextColor','titleCardSubtitleTextColor','titleCardOutlineColor','titleCardSubtitleOutlineColor'].includes(id)) {
        const fallback = ['titleCardTextColor','titleCardSubtitleTextColor'].includes(id) ? '#FFFFFF' : '#000000';
        $(id).value=MenuThemeTools.quantizeHexColor($(id).value,fallback);
      }
      saveTitleCardFields();
    });
  }
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
  await api('/reset', {method:'POST'}); resetTitleCardPreviewCache(); state=null; selectedID=''; clipConfigs={}; scopeInitialized=false; titleCardProject=null; titleCardProjectSource=''; titleCardPart=1; titleCardSectionSignature=''; projectDefaults={...DEFAULT_CLIP}; playheads={}; lastPreviewKey=''; lastThumbKey=''; if($('musicArtworkCustom')) $('musicArtworkCustom').value=''; romTitleAuto=true; smartAnalysis=null; $('romTitle').value=''; show('welcome');
};
setInterval(() => fetch(BASE + '/heartbeat', {method:'POST',headers:headers(),keepalive:true}).catch(()=>{}), 5000);
window.addEventListener('pagehide', () => fetch(BASE + '/close-intent', {method:'POST',headers:headers(),keepalive:true}).catch(()=>{}));

poll();
})().catch(error => {
  console.error('Desktop UI failed to initialize:', error);
  const target = document.getElementById('engineError') || document.body;
  if (target) {
    target.textContent = 'Could not initialize the desktop interface. ' + (error?.message || error);
    target.classList?.remove('hidden');
  }
});

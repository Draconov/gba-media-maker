const TOKEN = window.GBAVM_SESSION_TOKEN, BASE = '/' + TOKEN + '/api', $ = id => document.getElementById(id);
let state = null, pollTimer = null, uploadBusy = false, selected = 0, previewMode = 'start', audioURL = '', lastPreviewKey = '', romTitleAuto = true;
function headers(x = {}) { return Object.assign({ 'X-GBA-Token': TOKEN }, x); }
function show(id) { ['welcome', 'loading', 'editor'].forEach(x => $(x).classList.toggle('hidden', x !== id)); $('topBar').classList.toggle('hidden', id !== 'editor'); }
async function api(path, opt = {}) { opt.headers = headers(opt.headers || {}); let r = await fetch(BASE + path, opt); if (!r.ok) {
    let x;
    try {
        x = await r.json();
    }
    catch {
        x = { error: await r.text() };
    }
    throw Error(x.error || 'Request failed');
} ; return r.status === 204 ? null : r.json(); }
function fmt(sec) { sec = Math.max(0, +sec || 0); let m = Math.floor(sec / 60), s = Math.floor(sec % 60); return m + ':' + String(s).padStart(2, '0'); }
function parseClock(s) { if (String(s).trim() === '')
    return 0; let a = String(s).split(':').map(Number); if (a.some(x => !isFinite(x)))
    return NaN; return a.length === 3 ? a[0] * 3600 + a[1] * 60 + a[2] : a.length === 2 ? a[0] * 60 + a[1] : a[0]; }
function titleName(n) { return (n.replace(/\.[^.]+$/, '').toUpperCase().replace(/[^A-Z0-9 ]/g, ' ').replace(/\s+/g, ' ').trim() || 'GBA VIDEO').slice(0, 12); }
function choose(append = false) { $('picker').dataset.append = append ? '1' : '0'; $('picker').click(); }
$('welcome').onclick = () => choose(false);
$('welcome').onkeydown = e => { if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault();
    choose(false);
} };
$('addVideos').onclick = () => choose(true);
$('picker').onchange = () => { let f = [...$('picker').files]; if (f.length)
    upload(f, $('picker').dataset.append === '1'); $('picker').value = ''; };
for (const ev of ['dragenter', 'dragover'])
    document.addEventListener(ev, e => { e.preventDefault(); $('welcome').classList.add('drag'); });
for (const ev of ['dragleave', 'drop'])
    document.addEventListener(ev, e => { e.preventDefault(); $('welcome').classList.remove('drag'); });
document.addEventListener('drop', e => { let f = [...(e.dataTransfer?.files || [])]; if (f.length)
    upload(f, state && state.videos?.length); });
function upload(files, append) { if (uploadBusy)
    return; if (!append)
    romTitleAuto = true; uploadBusy = true; show('loading'); $('loadingTitle').textContent = 'Loading ' + files.length + ' video' + (files.length === 1 ? '' : 's') + '…'; $('loadingText').textContent = 'Copying files into the portable workspace.'; let form = new FormData(); files.forEach(f => form.append('video', f, f.name)); let x = new XMLHttpRequest(); x.open('POST', BASE + '/upload?append=' + (append ? 1 : 0)); x.setRequestHeader('X-GBA-Token', TOKEN); x.upload.onprogress = e => { if (e.lengthComputable)
    $('loadingFill').style.width = Math.round(e.loaded / e.total * 100) + '%'; }; x.onload = () => { uploadBusy = false; if (x.status < 300) {
    lastPreviewKey = '';
    $('previewImage').removeAttribute('src');
    poll();
}
else {
    showLoadError('Upload failed');
} }; x.onerror = () => { uploadBusy = false; showLoadError('Upload failed'); }; x.send(form); }
function showLoadError(m) { show('loading'); $('loadingTitle').textContent = 'Could not open videos'; $('loadingText').textContent = m; }
async function poll() { try {
    state = await api('/state');
    render();
}
catch (e) {
    console.error(e);
} clearTimeout(pollTimer); pollTimer = setTimeout(poll, 500); }
function render() {
    if (!state)
        return;
    if (!state.videos || !state.videos.length) {
        show('welcome');
        $('clipInfo').textContent = '';
        return;
    }
    if (state.inspectStatus === 'waiting' || state.inspectStatus === 'inspecting') {
        show('loading');
        $('loadingTitle').textContent = state.inspectStatus === 'waiting' ? 'Preparing the app…' : 'Opening videos…';
        $('loadingText').textContent = state.inspectStatus === 'waiting' ? (state.engineMessage || 'Preparing FFmpeg') : 'Reading duration, dimensions and audio streams.';
        $('loadingFill').style.width = (state.engineProgress || 0) + '%';
    }
    if (state.engineStatus === 'error' || state.engineStatus === 'missing') {
        show('loading');
        $('engineError').textContent = state.engineMessage;
        $('engineError').classList.remove('hidden');
        $('retryEngine').classList.remove('hidden');
    }
    if (state.inspectStatus === 'error') {
        show('loading');
        showLoadError(state.inspectError || 'A video could not be inspected.');
    }
    if (state.inspectStatus === 'ready') {
        show('editor');
        if (selected >= state.videos.length)
            selected = 0;
        renderClips();
        updatePreview();
        estimate();
    }
    let ids = ['preset', 'start', 'end', 'speed', 'fps', 'fit', 'seekSeconds', 'paletteMode', 'ditherMode', 'compression', 'audio', 'volume', 'normalize', 'limiter', 'romTitle', 'outputMode', 'loop', 'resume'];
    ids.forEach(id => $(id).disabled = !!state.converting);
    $('convert').disabled = !!state.converting;
    if (state.converting) {
        $('progressWrap').classList.remove('hidden');
        $('done').classList.add('hidden');
        $('convertError').classList.add('hidden');
        $('progressText').textContent = state.progressMessage;
        $('progressPct').textContent = state.progress + '%';
        $('progressFill').style.width = state.progress + '%';
    }
    else if (state.result) {
        $('progressWrap').classList.remove('hidden');
        $('progressText').textContent = 'Output created successfully';
        $('progressPct').textContent = '100%';
        $('progressFill').style.width = '100%';
        $('done').classList.remove('hidden');
    }
    else if (state.convertError) {
        $('convertError').textContent = state.convertError;
        $('convertError').classList.remove('hidden');
    }
}
async function removeVideo(i) { try {
    await api('/video/remove', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ index: i }) });
    lastPreviewKey = '';
    $('previewImage').removeAttribute('src');
    selected = Math.max(0, Math.min(selected, state.videos.length - 2));
    await poll();
    if (!state?.videos?.length) {
        selected = 0;
        $('clipInfo').textContent = '';
        show('welcome');
    }
}
catch (e) {
    alert(e.message);
} }
function renderClips() { let h = ''; state.videos.forEach((v, i) => { let inf = v.info; h += '<div class="clip ' + (i === selected ? 'active' : '') + '" data-i="' + i + '"><div class="clip-info"><b>' + (i + 1) + '. ' + escapeHTML(v.name) + '</b><small>' + (inf ? (inf.width + '×' + inf.height + ' • ' + fmt(inf.duration) + (inf.audioStreams ? ' • audio' : ' • silent')) : v.status) + '</small></div><button class="clip-remove" type="button" data-remove="' + i + '" title="Remove this video" aria-label="Remove ' + escapeHTML(v.name) + '">×</button></div>'; }); $('clips').innerHTML = h; [...$('clips').querySelectorAll('.clip')].forEach(el => el.onclick = e => { if (e.target.closest('.clip-remove'))
    return; selected = +el.dataset.i; renderClips(); updatePreview(); }); [...$('clips').querySelectorAll('.clip-remove')].forEach(el => el.onclick = e => { e.stopPropagation(); removeVideo(+el.dataset.remove); }); let v = state.videos[selected]; if (v?.info)
    $('clipInfo').textContent = 'Previewing ' + v.name + ' • ' + v.info.fps.toFixed(2) + ' source fps';
else
    $('clipInfo').textContent = ''; if (romTitleAuto && state.videos[0])
    $('romTitle').value = titleName(state.videos[0].name); let modeSel = $('outputMode'), current = modeSel.value; if (state.videos.length === 1) {
    modeSel.innerHTML = '<option value="rom">Single ROM</option>';
    modeSel.value = 'rom';
}
else {
    modeSel.innerHTML = '<option value="playlist">One ROM — play clips in order</option><option value="menu">One ROM — clip menu</option><option value="batch">Separate ROMs in ZIP</option>';
    modeSel.value = ['playlist', 'menu', 'batch'].includes(current) ? current : 'playlist';
} }
function escapeHTML(s) { return String(s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
let previewTimer;
function updatePreview() { if (!state?.videos?.[selected]?.info)
    return; clearTimeout(previewTimer); previewTimer = setTimeout(() => { let v = state.videos[selected], t = previewMode === 'end' ? parseClock($('end').value) : parseClock($('start').value); if (previewMode === 'end' && (!$('end').value.trim() || !isFinite(t)))
    t = v.info.duration; t = Math.min(Math.max(0, t), v.info.duration); let key = [selected, t.toFixed(3), $('fit').value, previewMode].join('|'); if (key === lastPreviewKey && $('previewImage').src)
    return; lastPreviewKey = key; let img = $('previewImage'); img.onerror = () => { img.removeAttribute('src'); }; img.src = BASE + '/preview?index=' + selected + '&time=' + encodeURIComponent(t) + '&fit=' + $('fit').value + '&key=' + encodeURIComponent(key); }, 160); }
$('previewStart').onclick = () => { previewMode = 'start'; lastPreviewKey = ''; updatePreview(); };
$('previewEnd').onclick = () => { previewMode = 'end'; lastPreviewKey = ''; updatePreview(); };
function values() { return { start: $('start').value, end: $('end').value, speed: +$('speed').value, fps: $('fps').value, fit: $('fit').value, audio: $('audio').value, volume: +$('volume').value, loop: $('loop').checked, romTitle: $('romTitle').value, seekSeconds: +$('seekSeconds').value, normalize: $('normalize').checked, limiter: $('limiter').checked, resume: $('resume').checked, compression: $('compression').value, paletteMode: $('paletteMode').value, ditherMode: $('ditherMode').value, outputMode: $('outputMode').value }; }
const presets = { best: { fps: 'smooth', audio: 'mix', paletteMode: 'scene', ditherMode: 'error', compression: 'delta', normalize: true, limiter: true }, balanced: { fps: 'balanced', audio: 'mix', paletteMode: 'shared', ditherMode: 'ordered', compression: 'delta', normalize: false, limiter: true }, long: { fps: 'compact', audio: 'mix', paletteMode: 'shared', ditherMode: 'ordered', compression: 'delta', normalize: false, limiter: true }, small: { fps: 'compact', audio: 'none', paletteMode: 'shared', ditherMode: 'off', compression: 'delta', normalize: false, limiter: false } };
function applyPreset() { let p = presets[$('preset').value]; if (!p)
    return; Object.entries(p).forEach(([k, v]) => { let e = $(k); if (e.type === 'checkbox')
    e.checked = v;
else
    e.value = v; }); estimate(); updatePreview(); }
$('preset').onchange = applyPreset;
function markCustom(e) { if (e.target.id !== 'preset')
    $('preset').value = 'custom'; }
function estimate() { if (!state?.videos?.length)
    return; let v = values(), start = parseClock(v.start), endText = v.end.trim(), vb = { smooth: 4, balanced: 5, classic: 6, compact: 8 }[v.fps], fps = 59.727500569606 / vb, totalFrames = 0, raw = 16384 + state.videos.length * 96, totalDur = 0; for (let x of state.videos) {
    if (!x.info)
        continue;
    let end = endText ? Math.min(parseClock(endText), x.info.duration) : x.info.duration;
    if (!isFinite(start) || !isFinite(end) || end <= start) {
        $('estimate').textContent = 'Check trim settings.';
        return;
    }
    let d = (end - start) / v.speed, f = Math.max(1, Math.ceil(d * fps));
    totalFrames += f;
    totalDur += end - start;
    let pals = v.paletteMode === 'scene' ? Math.ceil(f / 60) : 1;
    raw += pals * 512 + (pals > 1 ? f * 2 : 0) + f * 9600 + (v.compression === 'delta' ? f * 16 : 0);
    if (v.audio !== 'none' && x.info.audioStreams)
        raw += f * 4 + Math.ceil((f * vb / 59.7275) * 16384 / 16) * 16;
} let p = 1048576; while (p < raw)
    p *= 2; let bps = fps * 9600 + (v.audio !== 'none' ? 16384 + fps * 4 : 0), limit = (33554432 - 16384) / bps * v.speed; $('estimate').innerHTML = (raw > 33554432 ? '<b style="color:#ff8d8d">Worst-case estimate exceeds 32 MiB</b>' : 'Estimated cartridge: <b>' + (p / 1048576) + ' MiB</b>') + '<br>Worst-case data: ' + (raw / 1048576).toFixed(2) + ' MiB • ' + totalFrames + ' frames • ' + fps.toFixed(2) + ' fps<br>Approximate single-clip duration limit: ' + fmt(limit) + ' <span class="pill">delta compression may improve it</span>'; }
for (let id of ['start', 'end', 'speed', 'fps', 'fit', 'seekSeconds', 'paletteMode', 'ditherMode', 'compression', 'audio', 'volume', 'normalize', 'limiter'])
    $(id).addEventListener('input', e => { markCustom(e); estimate(); if (['start', 'end', 'fit'].includes(id))
        updatePreview(); });
$('romTitle').addEventListener('input', () => { romTitleAuto = false; });
$('audioPreview').onclick = async () => { try {
    $('audioPreview').disabled = true;
    let r = await fetch(BASE + '/audio-preview?index=' + selected, { method: 'POST', headers: headers({ 'Content-Type': 'application/json' }), body: JSON.stringify(values()) });
    if (!r.ok) {
        let x = await r.json();
        throw Error(x.error);
    }
    let b = await r.blob();
    if (audioURL)
        URL.revokeObjectURL(audioURL);
    audioURL = URL.createObjectURL(b);
    $('audioPlayer').src = audioURL;
    $('audioPlayer').play();
}
catch (e) {
    alert(e.message);
}
finally {
    $('audioPreview').disabled = false;
} };
$('convert').onclick = async () => { try {
    await api('/convert', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(values()) });
    poll();
}
catch (e) {
    $('convertError').textContent = e.message;
    $('convertError').classList.remove('hidden');
} };
$('download').onclick = () => { let a = document.createElement('a'); a.href = BASE + '/download'; a.download = state.downloadName || 'GBA_Video_Maker_output'; a.click(); };
$('retryEngine').onclick = () => api('/engine/retry', { method: 'POST' });
$('resetTop').onclick = async () => { await api('/reset', { method: 'POST' }); state = null; selected = 0; lastPreviewKey = ''; romTitleAuto = true; $('romTitle').value = ''; show('welcome'); };
setInterval(() => fetch(BASE + '/heartbeat', { method: 'POST', headers: headers(), keepalive: true }).catch(() => { }), 5000);
window.addEventListener('pagehide', () => fetch(BASE + '/close-intent', { method: 'POST', headers: headers(), keepalive: true }).catch(() => { }));
poll();

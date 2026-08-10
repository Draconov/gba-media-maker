(function(global){
'use strict';
const W=120,H=80,BYTES=W*H;
function clamp(v,a,b){return Math.max(a,Math.min(b,v));}
function q8(v){return Math.round(clamp(v,0,255)*31/255);}
function rgb555(r,g,b){return q8(r)|(q8(g)<<5)|(q8(b)<<10);}
function hexParts(hex,fallback='#000000'){let h=String(hex||fallback).trim();if(!/^#[0-9a-f]{6}$/i.test(h))h=fallback;return [parseInt(h.slice(1,3),16),parseInt(h.slice(3,5),16),parseInt(h.slice(5,7),16)];}
function rgb555ToHex(v){const r=Math.round((v&31)*255/31),g=Math.round(((v>>5)&31)*255/31),b=Math.round(((v>>10)&31)*255/31);return '#'+[r,g,b].map(x=>x.toString(16).padStart(2,'0')).join('').toUpperCase();}
function quantizeHexColor(hex,fallback='#000000'){const [r,g,b]=hexParts(hex,fallback);return rgb555ToHex(rgb555(r,g,b));}
function describeColor(hex,fallback){const q=quantizeHexColor(hex,fallback),[r8,g8,b8]=hexParts(q,fallback);return{hex:q,r:q8(r8),g:q8(g8),b:q8(b8),value:rgb555(r8,g8,b8)};}
function settingsColours(s={}){return{ui:describeColor(s.uiColor,'#FFFFFF').value,selected:describeColor(s.selectedColor,'#FFDE00').value,outline:describeColor(s.outlineColor,'#000000').value};}
function basePalette(){const p=new Uint8Array(512);const cols=[[6,10,20],[10,25,48],[15,48,86],[18,78,120],[32,119,163],[80,169,201],[160,218,230],[255,255,255],[255,222,0],[0,0,0]];cols.forEach((c,i)=>{const v=rgb555(...c);p[i*2]=v&255;p[i*2+1]=v>>8;});return p;}
function proceduralFrame(kind,phase=0){const f=new Uint8Array(BYTES);for(let y=0;y<H;y++)for(let x=0;x<W;x++){let v=0;if(kind==='classic-dark')v=((x+y)>>4)&1;if(kind.includes('wave')){const wave=Math.sin((x+phase*3)/10)+Math.sin((y+phase)/7);v=clamp(2+Math.floor((y/H)*3+wave),1,6);}f[y*W+x]=v;}return f;}
function applyUI(theme,s){const t={...theme,frames:theme.frames.map(x=>new Uint8Array(x)),palette:new Uint8Array(theme.palette)};const c=settingsColours(s);t.uiColor=c.ui;t.selectedColor=c.selected;t.outline=s?.outline!==false;t.outlineColor=c.outline;return t;}
function createBuiltinTheme(id='ocean-wave-animated',s={}){let count=1,kind='static',frameVBlanks=12;if(id.includes('animated')){count=8;kind='frames';}const frames=[];for(let i=0;i<count;i++)frames.push(proceduralFrame(id,i));return applyUI({id,name:id.replaceAll('-',' '),kind,palette:basePalette(),frames,frameVBlanks,uiColor:0x7fff,selectedColor:0x037f,outline:true,outlineColor:0},s);}
function b64(bytes){let s='';for(let i=0;i<bytes.length;i+=0x8000)s+=String.fromCharCode(...bytes.subarray(i,i+0x8000));return btoa(s);}
function unb64(s){const raw=atob(s||'');const a=new Uint8Array(raw.length);for(let i=0;i<raw.length;i++)a[i]=raw.charCodeAt(i);return a;}
function serializeTheme(t){if(!t)return null;return{id:t.id||'custom',name:t.name||'Custom',kind:t.kind||'static',palette:b64(t.palette),frames:t.frames.map(b64),frameVBlanks:t.frameVBlanks||12,uiColor:t.uiColor||0,selectedColor:t.selectedColor||0,outline:t.outline!==false,outlineColor:t.outlineColor||0,shimmer:t.shimmer||null};}
function deserializeTheme(t){if(!t)return null;return{...t,palette:typeof t.palette==='string'?unb64(t.palette):new Uint8Array(t.palette),frames:(t.frames||[]).map(x=>typeof x==='string'?unb64(x):new Uint8Array(x))};}
function nearestPaletteIndex(r,g,b){const qr=Math.round(r/64),qg=Math.round(g/64),qb=Math.round(b/64);return 1+qr*25+qg*5+qb;}
function quantizedPalette(){const p=basePalette();for(let r=0;r<5;r++)for(let g=0;g<5;g++)for(let b=0;b<5;b++){const i=1+r*25+g*5+b,v=rgb555(r*64,g*64,b*64);p[i*2]=v&255;p[i*2+1]=v>>8;}return p;}
function frameFromImageData(data){const f=new Uint8Array(BYTES);for(let i=0;i<BYTES;i++)f[i]=nearestPaletteIndex(data[i*4],data[i*4+1],data[i*4+2]);return f;}
async function decodeImage(file){const bmp=await createImageBitmap(file);const c=document.createElement('canvas');c.width=W;c.height=H;const x=c.getContext('2d');x.imageSmoothingEnabled=true;const scale=Math.max(W/bmp.width,H/bmp.height),dw=bmp.width*scale,dh=bmp.height*scale;x.drawImage(bmp,(W-dw)/2,(H-dh)/2,dw,dh);return frameFromImageData(x.getImageData(0,0,W,H).data);}
function isVideoFile(file){return !!file&&(String(file.type).startsWith('video/')||/\.(mp4|m4v|mov|webm|mkv|avi|mpg|mpeg)$/i.test(file.name||''));}
async function decodeVideo(file,progress){const url=URL.createObjectURL(file);const v=document.createElement('video');v.muted=true;v.src=url;await new Promise((res,rej)=>{v.onloadedmetadata=res;v.onerror=rej;});const c=document.createElement('canvas');c.width=W;c.height=H;const x=c.getContext('2d'),frames=[];const n=Math.min(8,Math.max(1,Math.ceil(v.duration||1)));for(let i=0;i<n;i++){v.currentTime=(v.duration||1)*i/n;await new Promise(res=>{v.onseeked=res;setTimeout(res,200);});x.drawImage(v,0,0,W,H);frames.push(frameFromImageData(x.getImageData(0,0,W,H).data));progress?.((i+1)/n);}URL.revokeObjectURL(url);return frames;}
async function decodeCustomFile(file,s,progress){const frames=isVideoFile(file)?await decodeVideo(file,progress):[await decodeImage(file)];return applyUI({id:'custom',name:file.name||'Custom',kind:frames.length>1?'frames':'static',palette:quantizedPalette(),frames,frameVBlanks:12},s);}
function decodeRGB24Frames(bytes,name,frameVBlanks,s,progress){const src=bytes instanceof Uint8Array?bytes:new Uint8Array(bytes);const stride=BYTES*3,count=Math.max(1,Math.floor(src.length/stride)),frames=[];for(let n=0;n<count;n++){const f=new Uint8Array(BYTES),off=n*stride;for(let i=0;i<BYTES;i++)f[i]=nearestPaletteIndex(src[off+i*3],src[off+i*3+1],src[off+i*3+2]);frames.push(f);progress?.((n+1)/count);}return applyUI({id:'custom',name:name||'Custom',kind:frames.length>1?'frames':'static',palette:quantizedPalette(),frames,frameVBlanks:frameVBlanks||12},s);}
function paletteRGB(p,i){const v=p[i*2]|(p[i*2+1]<<8);return[(v&31)*255/31,((v>>5)&31)*255/31,((v>>10)&31)*255/31];}
function drawPreview(canvas,t){if(!canvas||!t?.frames?.length)return;const ctx=canvas.getContext('2d'),img=ctx.createImageData(W,H),frame=t.frames[(t._phase||0)%t.frames.length];for(let i=0;i<BYTES;i++){const [r,g,b]=paletteRGB(t.palette,frame[i]);img.data[i*4]=r;img.data[i*4+1]=g;img.data[i*4+2]=b;img.data[i*4+3]=255;}const c=document.createElement('canvas');c.width=W;c.height=H;c.getContext('2d').putImageData(img,0,0);ctx.imageSmoothingEnabled=false;ctx.clearRect(0,0,canvas.width,canvas.height);ctx.drawImage(c,0,0,canvas.width,canvas.height);ctx.fillStyle=rgb555ToHex(t.uiColor);ctx.font='20px monospace';ctx.fillText('SELECT MEDIA',24,40);ctx.fillStyle=rgb555ToHex(t.selectedColor);ctx.fillText('> 01  VIDEO',28,80);ctx.fillStyle=rgb555ToHex(t.uiColor);ctx.fillText('  02  AUDIO',28,110);ctx.fillText('  03  IMAGE',28,140);}
function startPreview(canvas,getTheme){let stopped=false,t=0;function loop(){if(stopped)return;const th=getTheme?.();if(th){th._phase=t++;drawPreview(canvas,th);}setTimeout(loop,180);}loop();return()=>{stopped=true;};}
const COMMON_GBA_COLORS=[
  ['White','#FFFFFF'],['Gray','#848484'],['Black','#000000'],['Red','#FF0000'],['Orange','#FF8400'],
  ['Yellow','#FFDE00'],['Green','#00FF00'],['Cyan','#00FFFF'],['Blue','#0000FF'],['Magenta','#FF00FF']
].map(([name,hex])=>Object.freeze({name,hex:quantizeHexColor(hex,hex)}));
let activeGBAColorPicker=null;
function normalizeHexColor(value){
  let text=String(value??'').trim().replace(/^0x/i,'').replace(/^#/,'');
  if(/^[0-9a-f]{3}$/i.test(text))text=text.split('').map(c=>c+c).join('');
  if(!/^[0-9a-f]{6}$/i.test(text))return null;
  return `#${text.toUpperCase()}`;
}
function rgbToHSV(r,g,b){
  const red=clamp(Number(r)||0,0,255)/255,green=clamp(Number(g)||0,0,255)/255,blue=clamp(Number(b)||0,0,255)/255;
  const maximum=Math.max(red,green,blue),minimum=Math.min(red,green,blue),difference=maximum-minimum;
  let hue=0;
  if(difference){
    if(maximum===red)hue=60*(((green-blue)/difference)%6);
    else if(maximum===green)hue=60*((blue-red)/difference+2);
    else hue=60*((red-green)/difference+4);
  }
  if(hue<0)hue+=360;
  return{h:hue,s:maximum?difference/maximum:0,v:maximum};
}
function hsvToHex(h,s,v){
  const hue=((Number(h)||0)%360+360)%360,saturation=clamp(Number(s)||0,0,1),value=clamp(Number(v)||0,0,1);
  const chroma=value*saturation,section=hue/60,secondary=chroma*(1-Math.abs(section%2-1));
  let red=0,green=0,blue=0;
  if(section<1)[red,green,blue]=[chroma,secondary,0];
  else if(section<2)[red,green,blue]=[secondary,chroma,0];
  else if(section<3)[red,green,blue]=[0,chroma,secondary];
  else if(section<4)[red,green,blue]=[0,secondary,chroma];
  else if(section<5)[red,green,blue]=[secondary,0,chroma];
  else [red,green,blue]=[chroma,0,secondary];
  const match=value-chroma;
  return `#${[red,green,blue].map(channel=>Math.round((channel+match)*255).toString(16).padStart(2,'0')).join('')}`.toUpperCase();
}
function setupGBAColorPicker(input,{label='GBA colour'}={}){
  if(!input)return null;
  if(input._gbaColorPickerController)return input._gbaColorPickerController;
  const control=input.closest('.gba-color-control');
  if(!control||typeof document==='undefined')return null;
  control.classList.add('gba-color-enhanced');

  const trigger=document.createElement('button');
  trigger.type='button';
  trigger.className='gba-color-trigger';
  trigger.id=`${input.id}PickerButton`;
  trigger.setAttribute('aria-haspopup','dialog');
  trigger.setAttribute('aria-expanded','false');
  trigger.innerHTML='<span class="gba-color-trigger-swatch" aria-hidden="true"></span>';
  control.insertBefore(trigger,input);
  for(const externalLabel of document.querySelectorAll(`label[for="${input.id}"]`))externalLabel.htmlFor=trigger.id;
  input.classList.add('gba-native-color-input');
  input.tabIndex=-1;
  input.setAttribute('aria-hidden','true');

  const panel=document.createElement('div');
  panel.className='gba-color-popover';
  panel.id=`${input.id}PickerPanel`;
  panel.hidden=true;
  panel.setAttribute('role','dialog');
  panel.setAttribute('aria-label',`${label} picker`);
  trigger.setAttribute('aria-controls',panel.id);

  const content=document.createElement('div');
  content.className='gba-picker-content';
  const svArea=document.createElement('div');
  svArea.className='gba-sv-area';
  svArea.tabIndex=0;
  svArea.setAttribute('role','slider');
  svArea.setAttribute('aria-label',`${label} saturation and brightness`);
  svArea.setAttribute('aria-valuemin','0');
  svArea.setAttribute('aria-valuemax','100');
  const svThumb=document.createElement('span');
  svThumb.className='gba-sv-thumb';
  svThumb.setAttribute('aria-hidden','true');
  svArea.append(svThumb);
  content.append(svArea);

  const pickerRow=document.createElement('div');
  pickerRow.className='gba-picker-row';
  const eyedropperButton=document.createElement('button');
  eyedropperButton.type='button';
  eyedropperButton.className='gba-eyedropper';
  eyedropperButton.setAttribute('aria-label',`Pick ${label.toLowerCase()} from the screen`);
  eyedropperButton.title='Pick a colour from the screen';
  eyedropperButton.innerHTML='<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18.5 2.5a2.12 2.12 0 0 1 3 3L13 14l-3-3 8.5-8.5Z"/><path d="m9 12-5.2 5.2a2 2 0 0 0-.6 1.4V21h2.4a2 2 0 0 0 1.4-.6L12 15"/><path d="m14.5 6.5 3 3"/></svg>';
  const currentSwatch=document.createElement('span');
  currentSwatch.className='gba-current-swatch';
  currentSwatch.setAttribute('role','img');
  currentSwatch.setAttribute('aria-label',`${label} preview`);
  const hueSlider=document.createElement('input');
  hueSlider.type='range';hueSlider.min='0';hueSlider.max='359';hueSlider.step='1';hueSlider.value='0';
  hueSlider.className='gba-hue-slider';
  hueSlider.setAttribute('aria-label',`${label} hue`);
  hueSlider.title='Hue';
  pickerRow.append(eyedropperButton,currentSwatch,hueSlider);
  content.append(pickerRow);

  const rgbEditor=document.createElement('div');
  rgbEditor.className='gba-rgb-editor';
  const rgbInputs=[];
  for(const channel of ['R','G','B']){
    const channelLabel=document.createElement('label');channelLabel.className='gba-rgb-channel';
    const caption=document.createElement('span');caption.textContent=channel;
    const numberInput=document.createElement('input');
    numberInput.type='number';numberInput.min='0';numberInput.max='255';numberInput.step='1';numberInput.inputMode='numeric';
    numberInput.setAttribute('aria-label',`${label} ${channel}`);
    channelLabel.append(caption,numberInput);rgbEditor.append(channelLabel);rgbInputs.push(numberInput);
  }
  content.append(rgbEditor);

  const hexLabel=document.createElement('label');hexLabel.className='gba-hex-editor';
  const hexCaption=document.createElement('span');hexCaption.textContent='HEX';
  const hexInput=document.createElement('input');
  hexInput.type='text';hexInput.inputMode='text';hexInput.autocomplete='off';hexInput.spellcheck=false;hexInput.maxLength=9;hexInput.placeholder='#RRGGBB';
  hexInput.setAttribute('aria-label',`${label} HEX code`);
  hexLabel.append(hexCaption,hexInput);content.append(hexLabel);

  const quickGrid=document.createElement('div');quickGrid.className='gba-quick-colors';
  const quickButtons=[];
  for(const colour of COMMON_GBA_COLORS){
    const button=document.createElement('button');button.type='button';button.className='gba-quick-color';button.style.background=colour.hex;
    button.title=`${colour.name} — ${colour.hex}`;button.setAttribute('aria-label',`Choose ${colour.name} ${colour.hex}`);button.dataset.hex=colour.hex;
    quickGrid.append(button);quickButtons.push(button);
  }
  content.append(quickGrid);panel.append(content);control.append(panel);

  const swatch=trigger.querySelector('.gba-color-trigger-swatch');
  const EyeDropperClass=typeof window!=='undefined'?window.EyeDropper:null;
  let lastHue=0,saturation=0,brightness=1,draggingSV=false;
  const clamp01=value=>clamp(value,0,1);
  function close(){panel.hidden=true;trigger.setAttribute('aria-expanded','false');if(activeGBAColorPicker===controller)activeGBAColorPicker=null;}
  function open(){
    if(input.disabled)return;
    if(activeGBAColorPicker&&activeGBAColorPicker!==controller)activeGBAColorPicker.close();
    panel.hidden=false;trigger.setAttribute('aria-expanded','true');activeGBAColorPicker=controller;panel.classList.remove('align-right');
    const bounds=panel.getBoundingClientRect();if(bounds.right>window.innerWidth-10)panel.classList.add('align-right');
  }
  function dispatch(kind){input.dispatchEvent(new Event(kind,{bubbles:true}));}
  function setRawHex(hex,commit=false){
    const normalized=normalizeHexColor(hex);if(!normalized)return false;
    input.value=normalized;dispatch('input');if(commit)dispatch('change');return true;
  }
  function applyRGB(commit=false){
    const values=rgbInputs.map(field=>Number(field.value));if(values.some(value=>!Number.isFinite(value)))return;
    const clamped=values.map(value=>clamp(Math.round(value),0,255));
    setRawHex(`#${clamped.map(value=>value.toString(16).padStart(2,'0')).join('')}`,commit);
  }
  function applyHue(commit=false){
    const hue=clamp(Math.round(Number(hueSlider.value)||0),0,359);lastHue=hue;
    setRawHex(hsvToHex(lastHue,saturation>0.01?saturation:1,brightness>0.01?brightness:1),commit);
  }
  function applySV(clientX,clientY,commit=false){
    const bounds=svArea.getBoundingClientRect();if(bounds.width<=0||bounds.height<=0)return;
    saturation=clamp01((clientX-bounds.left)/bounds.width);brightness=clamp01(1-(clientY-bounds.top)/bounds.height);
    setRawHex(hsvToHex(lastHue,saturation,brightness),commit);
  }
  function sync(){
    const colour=describeColor(input.value,'#000000');
    trigger.style.setProperty('--gba-swatch-color',colour.hex);swatch.style.backgroundColor=colour.hex;currentSwatch.style.backgroundColor=colour.hex;
    trigger.setAttribute('aria-label',`${label}: ${colour.hex}. Open colour picker`);currentSwatch.setAttribute('aria-label',`${label} preview ${colour.hex}`);
    const [r8,g8,b8]=hexParts(colour.hex,'#000000');
    for(const [field,value] of rgbInputs.map((field,index)=>[field,[r8,g8,b8][index]]))if(document.activeElement!==field)field.value=String(value);
    const hsv=rgbToHSV(r8,g8,b8);saturation=hsv.s;brightness=hsv.v;if(hsv.s>0.01&&hsv.v>0.01)lastHue=hsv.h;
    const displayedHue=Math.round(lastHue)%360;if(document.activeElement!==hueSlider)hueSlider.value=String(displayedHue);
    const hueColour=`hsl(${displayedHue} 100% 50%)`;hueSlider.style.setProperty('--hue-colour',hueColour);svArea.style.setProperty('--hue-colour',hueColour);
    svArea.style.setProperty('--sv-x',`${saturation*100}%`);svArea.style.setProperty('--sv-y',`${(1-brightness)*100}%`);
    svArea.setAttribute('aria-valuenow',String(Math.round(brightness*100)));svArea.setAttribute('aria-valuetext',`${Math.round(saturation*100)}% saturation, ${Math.round(brightness*100)}% brightness`);
    if(document.activeElement!==hexInput)hexInput.value=colour.hex;hexInput.classList.remove('invalid');
    for(const button of quickButtons)button.setAttribute('aria-pressed',String(button.dataset.hex===colour.hex));
    const disabled=Boolean(input.disabled);trigger.disabled=disabled;hueSlider.disabled=disabled;svArea.tabIndex=disabled?-1:0;svArea.setAttribute('aria-disabled',String(disabled));
    for(const field of rgbInputs)field.disabled=disabled;hexInput.disabled=disabled;for(const button of quickButtons)button.disabled=disabled;
    eyedropperButton.disabled=disabled||typeof EyeDropperClass!=='function';
    eyedropperButton.title=typeof EyeDropperClass==='function'?'Pick a colour from the screen':'Screen eyedropper is not available here';
    if(disabled)close();
  }

  trigger.addEventListener('click',event=>{event.preventDefault();panel.hidden?open():close();});
  svArea.addEventListener('pointerdown',event=>{if(input.disabled)return;draggingSV=true;svArea.setPointerCapture?.(event.pointerId);applySV(event.clientX,event.clientY,false);event.preventDefault();});
  svArea.addEventListener('pointermove',event=>{if(!draggingSV||input.disabled)return;applySV(event.clientX,event.clientY,false);});
  const finishSV=event=>{if(!draggingSV)return;draggingSV=false;applySV(event.clientX,event.clientY,true);svArea.releasePointerCapture?.(event.pointerId);};
  svArea.addEventListener('pointerup',finishSV);
  svArea.addEventListener('pointercancel',event=>{draggingSV=false;svArea.releasePointerCapture?.(event.pointerId);});
  svArea.addEventListener('keydown',event=>{
    if(input.disabled)return;const step=event.shiftKey?0.10:0.02;
    if(event.key==='ArrowLeft')saturation=clamp01(saturation-step);else if(event.key==='ArrowRight')saturation=clamp01(saturation+step);else if(event.key==='ArrowUp')brightness=clamp01(brightness+step);else if(event.key==='ArrowDown')brightness=clamp01(brightness-step);else return;
    event.preventDefault();setRawHex(hsvToHex(lastHue,saturation,brightness),true);
  });
  for(const field of rgbInputs){field.addEventListener('input',()=>applyRGB(false));field.addEventListener('change',()=>applyRGB(true));field.addEventListener('blur',()=>setTimeout(sync,0));field.addEventListener('keydown',event=>{if(event.key==='Enter'){event.preventDefault();applyRGB(true);field.select();}});}
  hexInput.addEventListener('input',()=>{const normalized=normalizeHexColor(hexInput.value);hexInput.classList.toggle('invalid',!normalized);if(normalized)setRawHex(normalized,false);});
  const commitHex=()=>{const normalized=normalizeHexColor(hexInput.value);if(!normalized){hexInput.classList.remove('invalid');sync();return;}setRawHex(normalized,true);sync();};
  hexInput.addEventListener('change',commitHex);hexInput.addEventListener('blur',()=>setTimeout(sync,0));hexInput.addEventListener('keydown',event=>{if(event.key==='Enter'){event.preventDefault();commitHex();hexInput.select();}});
  hueSlider.addEventListener('input',()=>applyHue(false));hueSlider.addEventListener('change',()=>applyHue(true));hueSlider.addEventListener('blur',()=>setTimeout(sync,0));
  eyedropperButton.addEventListener('click',async()=>{if(typeof EyeDropperClass!=='function'||input.disabled)return;try{const result=await new EyeDropperClass().open();if(result?.sRGBHex)setRawHex(result.sRGBHex,true);}catch(error){if(error?.name!=='AbortError')console.warn('Screen eyedropper failed',error);}});
  for(const button of quickButtons)button.addEventListener('click',()=>setRawHex(button.dataset.hex,true));
  input.addEventListener('input',sync);input.addEventListener('change',sync);
  document.addEventListener('pointerdown',event=>{if(!panel.hidden&&!control.contains(event.target))close();});
  document.addEventListener('keydown',event=>{if(event.key==='Escape'&&!panel.hidden){close();trigger.focus();}});
  const controller={open,close,sync,panel,trigger};input._gbaColorPickerController=controller;
  if(typeof MutationObserver!=='undefined')new MutationObserver(sync).observe(input,{attributes:true,attributeFilter:['disabled']});
  sync();return controller;
}
global.MenuThemeTools={describeColor,quantizeHexColor,settingsColours,rgb555ToHex,applyUI,createBuiltinTheme,serializeTheme,deserializeTheme,decodeCustomFile,decodeRGB24Frames,isVideoFile,setupGBAColorPicker,startPreview};
})(window);

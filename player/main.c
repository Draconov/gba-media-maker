typedef unsigned char u8;
typedef unsigned short u16;
typedef unsigned int u32;
typedef signed short s16;

#define REG16(a) (*(volatile u16 *)(a))
#define REG32(a) (*(volatile u32 *)(a))
#define REG_DISPCNT REG16(0x04000000)
#define REG_VCOUNT REG16(0x04000006)
#define REG_SOUNDCNT_L REG16(0x04000080)
#define REG_SOUNDCNT_H REG16(0x04000082)
#define REG_SOUNDCNT_X REG16(0x04000084)
#define REG_SOUNDBIAS REG16(0x04000088)
#define REG_FIFO_A REG32(0x040000A0)
#define REG_DMA1SAD REG32(0x040000BC)
#define REG_DMA1DAD REG32(0x040000C0)
#define REG_DMA1CNT_L REG16(0x040000C4)
#define REG_DMA1CNT_H REG16(0x040000C6)
#define REG_TM0CNT_L REG16(0x04000100)
#define REG_TM0CNT_H REG16(0x04000102)
#define REG_TM2CNT_L REG16(0x04000108)
#define REG_TM2CNT_H REG16(0x0400010A)
#define REG_TM3CNT_L REG16(0x0400010C)
#define REG_TM3CNT_H REG16(0x0400010E)
#define REG_KEYINPUT REG16(0x04000130)
#define REG_BLDCNT REG16(0x04000050)
#define REG_BLDY REG16(0x04000054)
#define PALRAM ((volatile u16 *)0x05000000)
#define VRAM0 ((volatile u16 *)0x06000000)
#define VRAM1 ((volatile u16 *)0x0600A000)
#define SRAM ((volatile u8 *)0x0E000000)
#define ROM_BASE 0x08000000u
#define MODE3 0x0403u
#define MODE4 0x0404u
#define PAGE 0x0010u
#define FORCE_BLANK 0x0080u

#define KEY_A 0x0001u
#define KEY_B 0x0002u
#define KEY_SELECT 0x0004u
#define KEY_START 0x0008u
#define KEY_RIGHT 0x0010u
#define KEY_LEFT 0x0020u
#define KEY_UP 0x0040u
#define KEY_DOWN 0x0080u
#define KEY_R 0x0100u
#define KEY_L 0x0200u

#define GLOBAL_FLAG_RESUME 0x0001u
#define GLOBAL_FLAG_PLAYLIST 0x0002u
#define GLOBAL_FLAG_TITLE_SCREEN 0x0004u
#define CLIP_FLAG_AUDIO 0x0001u
#define CLIP_FLAG_LOOP 0x0002u
#define CLIP_FLAG_COMPRESSED 0x0004u
#define CLIP_FLAG_SCENE_PAL 0x0008u
#define CLIP_FLAG_ADPCM 0x0010u
#define CLIP_FLAG_ADAPTIVE 0x0020u
#define CLIP_FLAG_MEDIA_AUDIO 0x0040u
#define CLIP_FLAG_MEDIA_IMAGE 0x0080u
#define CLIP_FLAG_MEDIA_META 0x0100u
#define MEDIA_META_MAGIC 0x31444D4Du
#define AUDIO_CODEC_PCM 1u
#define AUDIO_CODEC_ADPCM 2u
#define GBV5_MAGIC 0x35564247u
#define MENU_THEME_MAGIC 0x3148544Du
#define TITLE_CARD_MAGIC 0x31444354u
#define TITLE_CARD_FLAG_WAIT_A 1u
#define TITLE_CARD_FLAG_SKIP 2u
#define TITLE_CARD_FLAG_FADE 4u
#define FRAME_W 120u
#define FRAME_H 80u
#define FRAME_BYTES 9600u
#define NATIVE_PIXELS 38400u
#define UI_BLACK 250u
#define UI_DARK 251u
#define UI_WHITE 252u
#define UI_YELLOW 253u
#define UI_RED 254u
#define UI_GREEN 255u
#define SRAM_MAGIC 0x394D4247u /* GBM9 */
#define SRAM_XOR 0xA5A50000u

#define ACTION_NONE 0
#define ACTION_RESTART 1
#define ACTION_SEEK_BACK 2
#define ACTION_SEEK_FORWARD 3
#define ACTION_UI_REFRESH 4
#define ACTION_FRAME_BACK 5
#define ACTION_FRAME_FORWARD 6
#define ACTION_HELP 7
#define ACTION_TOGGLE_PAUSE 8
#define ACTION_PREV_CLIP 9
#define ACTION_NEXT_CLIP 10
#define ACTION_RESUME_PENDING 11
#define PLAY_RESULT_RESTART_CURRENT 0
#define PLAY_RESULT_NEXT_CLIP 1
#define PLAY_RESULT_RETURN_MENU 2
#define PLAY_RESULT_PREV_CLIP 3

enum PlaybackState { PLAYBACK_RUNNING = 0, PLAYBACK_PAUSED = 1, PLAYBACK_RESUME_ARMED = 2 };

struct GlobalMetadata {
 u32 magic; u16 version; u16 flags; u16 clip_count; u16 default_clip;
 u32 clip_table_offset; u32 clip_descriptor_size; u32 title_screen_part;
 char title_screen_name[24]; u32 reserved[4];
};
struct ClipDescriptor {
 u32 frame_count, frame_bytes, video_offset, video_index_offset, audio_offset, audio_size;
 u32 palette_offset, palette_index_offset, seek_table_offset, audio_rate, seek_frame_step;
 u16 vblanks_per_frame, source_width, source_height, flags, seek_seconds, palette_count, keyframe_interval, reserved0;
 char title[12]; u32 raw_video_bytes, stored_video_bytes, audio_codec, audio_sample_count, audio_block_samples, audio_block_bytes;
};

struct MediaMetadata { u32 magic; char artist[20]; char album[20]; };
struct MenuThemeHeader {
 u32 magic; u16 version, kind; u32 palette_offset, frames_offset; u16 frame_count, frame_vblanks, flags, ui_colour, selected_colour, outline_colour;
 u16 shimmer_source_start, shimmer_count, shimmer_target1, shimmer_interval1, shimmer_target2, shimmer_interval2, shimmer_phases, reserved0;
 u32 frame_bytes, data_size, reserved[3];
};
struct TitleCardHeader { u32 magic; u16 version, flags; u32 pixel_bytes, duration_vblanks, reserved[4]; };
struct PlayerUI { int muted, volume_level, hud_mode, hud_last_visible; u16 hud_timer,mute_timer,volume_timer,seek_timer; int seek_direction,seek_hold_direction; u16 seek_hold_counter; int help_combo_latched,hud_combo_latched,clip_combo_latched,pause_button_latched; };
struct PlaybackClock { u32 next_deadline, step_whole, step_remainder, remainder_accum; };
extern const struct GlobalMetadata gba_video_metadata;

static u8 frame_a[FRAME_BYTES], frame_b[FRAME_BYTES];
#define ADPCM_HALF 4096u
static u8 adpcm_pcm[ADPCM_HALF*2u] __attribute__((aligned(4)));
static const u8 *adpcm_stream; static u32 adpcm_start_sample,adpcm_next_switch,adpcm_active_half,adpcm_sample_count; static int adpcm_active;

static void wait_vblank(void){ while(REG_VCOUNT>=160){} while(REG_VCOUNT<160){} }
static u16 keys_down(void){ return (u16)((~REG_KEYINPUT)&0x03FFu); }
static const u8 *rom_ptr(u32 o){ return (const u8 *)(ROM_BASE+o); }
static u16 rd16(const u8*p){ return (u16)(p[0]|((u16)p[1]<<8)); }
static u32 rd32(const u8*p){ return (u32)p[0]|((u32)p[1]<<8)|((u32)p[2]<<16)|((u32)p[3]<<24); }
static void copy8(u8*d,const u8*s,u32 n){ while(n--)*d++=*s++; }
static void copy16(volatile u16*d,const u16*s,u32 n){ while(n--)*d++=*s++; }
static u32 udiv(u32 n,u32 d){ u32 q=0,b=1; if(!d)return 0; while(d<=n && !(d&0x80000000u)){d<<=1;b<<=1;} while(b){ if(n>=d){n-=d;q|=b;} d>>=1;b>>=1;} return q; }
static int clampi(int v,int lo,int hi){ return v<lo?lo:(v>hi?hi:v); }

static u16 glyph_bits(u8 c){
 switch(c){
 case '0':return 0x7B6F;case '1':return 0x2C97;case '2':return 0x73E7;case '3':return 0x73CF;case '4':return 0x5BC9;case '5':return 0x79CF;case '6':return 0x79EF;case '7':return 0x7292;case '8':return 0x7BEF;case '9':return 0x7BCF;
 case 'A':return 0x2BED;case 'B':return 0x6BAE;case 'C':return 0x7927;case 'D':return 0x6B6E;case 'E':return 0x79E7;case 'F':return 0x79E4;case 'G':return 0x79AF;case 'H':return 0x5BED;case 'I':return 0x7497;case 'J':return 0x124E;case 'K':return 0x5D6D;case 'L':return 0x4927;case 'M':return 0x5FE9;case 'N':return 0x5F6D;case 'O':return 0x7B6F;case 'P':return 0x7BE4;case 'Q':return 0x7B7B;case 'R':return 0x7BED;case 'S':return 0x79CF;case 'T':return 0x7492;case 'U':return 0x5B6F;case 'V':return 0x5B6A;case 'W':return 0x5BFD;case 'X':return 0x5AAD;case 'Y':return 0x5A92;case 'Z':return 0x72A7;
 case ':':return 0x0410;case ';':return 0x200A;case '/': return 0x12A4u;case '\\':return 0x4489;case '-':return 0x01C0;case '_':return 0x0007;case '+':return 0x05D0;case '=':return 0x0E38;case '.':return 0x0002;case ',':return 0x000A;case '!':return 0x2492;case '?':return 0x72C2;case '(':return 0x2488;case ')':return 0x1112;case '[':return 0x6926;case ']':return 0x324B;case '&':return 0x2AAE;case '%':return 0x5295;case '#':return 0x5F7D;case '@':return 0x7BE7;case '\'':return 0x2400;case '"':return 0x5A00;case '>':return 0x22A2;case '<':return 0x1144;
 case 0x80:return 0x2BED;case 0x81:return 0x79AE;case 0x82:return 0x6BAE;case 0x83:return 0x7924;case 0x84:return 0x1F24;case 0x85:return 0x2B7D;case 0x86:return 0x79E7;case 0x87:return 0x39A3;case 0x88:return 0x5F3F;case 0x89:return 0x55D5;case 0x8A:return 0x72CF;case 0x8B:return 0x5F6D;case 0x8C:return 0x7497;case 0x8D:return 0x5497;case 0x8E:return 0x557D;case 0x8F:return 0x5D6D;case 0x90:return 0x3B6D;case 0x91:return 0x5FE9;case 0x92:return 0x5BED;case 0x93:return 0x7B6F;case 0x94:return 0x7B6D;case 0x95:return 0x7BE4;case 0x96:return 0x7927;case 0x97:return 0x7492;case 0x98:return 0x5A92;case 0x99:return 0x2F7A;case 0x9A:return 0x5AAD;case 0x9B:return 0x5B79;case 0x9C:return 0x5AC9;case 0x9D:return 0x5B6F;case 0x9E:return 0x5BF9;case 0x9F:return 0x64D3;case 0xA0:return 0x5BAE;case 0xA1:return 0x49AE;case 0xA2:return 0x62CE;case 0xA3:return 0x5F6F;case 0xA4:return 0x3AED;case 0xA5:return 0x5F4A; default:return 0;
 }
}
static void set_ui_palette(void){u32 i;for(i=0;i<256;i++)PALRAM[i]=0;PALRAM[UI_BLACK]=0;PALRAM[UI_DARK]=0x18C6;PALRAM[UI_WHITE]=0x7FFF;PALRAM[UI_YELLOW]=0x037F;PALRAM[UI_RED]=0x001F;PALRAM[UI_GREEN]=0x03E0;}
static void p4(volatile u16*d,u32 x,u32 y,u8 c){volatile u16*r;u16 v;if(x>=120||y>=80)return;r=d+(y*2u)*120u+x;v=(u16)(c|((u16)c<<8));r[0]=v;r[120]=v;}
static void rect4(volatile u16*d,u32 x,u32 y,u32 w,u32 h,u8 c){u32 a,b;for(b=0;b<h;b++)for(a=0;a<w;a++)p4(d,x+a,y+b,c);}
static void char4(volatile u16*d,u32 x,u32 y,u8 c,u8 col){u16 bits=glyph_bits(c);u32 r,k;for(r=0;r<5;r++)for(k=0;k<3;k++)if(bits&(1u<<(14u-(r*3u+k))))p4(d,x+k,y+r,col);}
static void text4n(volatile u16*d,u32 x,u32 y,const char*s,u32 max,u8 col){u32 i=0;while(i<max&&s[i]){char4(d,x,y,(u8)s[i],col);x+=4;i++;}}
static void text4(volatile u16*d,u32 x,u32 y,const char*s,u8 col){text4n(d,x,y,s,64,col);}
static void clear4(volatile u16*d){u32 i;for(i=0;i<19200;i++)d[i]=0;}
static void p3(volatile u16*d,u32 x,u32 y,u16 c){if(x<240&&y<160)d[y*240u+x]=c;}
static void rect3(volatile u16*d,u32 x,u32 y,u32 w,u32 h,u16 c){u32 a,b;for(b=0;b<h;b++)for(a=0;a<w;a++)p3(d,x+a,y+b,c);}
static void char3(volatile u16*d,u32 x,u32 y,u8 c,u16 col){u16 bits=glyph_bits(c);u32 r,k,xx,yy;for(r=0;r<5;r++)for(k=0;k<3;k++)if(bits&(1u<<(14u-(r*3u+k))))for(yy=0;yy<2;yy++)for(xx=0;xx<2;xx++)p3(d,x+k*2u+xx,y+r*2u+yy,col);}
static void text3n(volatile u16*d,u32 x,u32 y,const char*s,u32 max,u16 col){u32 i=0;while(i<max&&s[i]){char3(d,x,y,(u8)s[i],col);x+=8;i++;}}
static u32 seconds_for_frame(u32 f,u16 vb){ return udiv(f*(u32)vb*1000u,59728u); }
static void time5(char*out,u32 sec){u32 m=udiv(sec,60),s=sec-m*60;out[0]=(char)('0'+(m/10)%10);out[1]=(char)('0'+m%10);out[2]=':';out[3]=(char)('0'+s/10);out[4]=(char)('0'+s%10);out[5]=0;}

static const u16 *palette_for_frame(const struct ClipDescriptor*c,u32 f){u32 i=0;if(c->palette_count>1&&c->palette_index_offset){i=rd16(rom_ptr(c->palette_index_offset+f*2));if(i>=c->palette_count)i=0;}return (const u16*)rom_ptr(c->palette_offset+i*512u);}
static void apply_delta(const u8*p,u32 n,u8*d){u32 pos=0,o=0;while(o+4<=n&&pos<FRAME_BYTES){u32 skip=rd16(p+o),run=rd16(p+o+2);o+=4;pos+=skip;if(pos>FRAME_BYTES)pos=FRAME_BYTES;if(run>FRAME_BYTES-pos)run=FRAME_BYTES-pos;if(o+run>n)run=n-o;copy8(d+pos,p+o,run);pos+=run;o+=run;}}
static const u8 *record(const struct ClipDescriptor*c,u32 f){return rom_ptr(c->video_offset+rd32(rom_ptr(c->video_index_offset+f*4)));}
static void load_frame_pixels(const struct ClipDescriptor*c,u32 f,u8*d){u32 b;const u8*r;if(!(c->flags&CLIP_FLAG_COMPRESSED)){copy8(d,rom_ptr(c->video_offset+f*FRAME_BYTES),FRAME_BYTES);return;}b=f;while(b){r=record(c,b);if(rd32(r)==0)break;b--;}r=record(c,b);if(rd32(r)==0&&rd32(r+4)>=FRAME_BYTES)copy8(d,r+8,FRAME_BYTES);else{u32 i;for(i=0;i<FRAME_BYTES;i++)d[i]=0;}while(b<f){b++;r=record(c,b);if(rd32(r)==0)copy8(d,r+8,FRAME_BYTES);else apply_delta(r+8,rd32(r+4),d);}}
static void load_next_pixels(const struct ClipDescriptor*c,u32 f,const u8*cur,u8*d){const u8*r;if(!(c->flags&CLIP_FLAG_COMPRESSED)){copy8(d,rom_ptr(c->video_offset+f*FRAME_BYTES),FRAME_BYTES);return;}copy8(d,cur,FRAME_BYTES);r=record(c,f);if(rd32(r)==0)copy8(d,r+8,FRAME_BYTES);else apply_delta(r+8,rd32(r+4),d);}
static void render_pixels(const u8*s,volatile u16*d){u32 y,x;for(y=0;y<80;y++){volatile u16*r0=d+y*240u,*r1=r0+120;for(x=0;x<120;x++){u16 p=s[y*120+x];p|=p<<8;r0[x]=p;r1[x]=p;}}}
static void copy_palette(const u16*p){u32 i;for(i=0;i<256;i++)PALRAM[i]=p[i];}
static void draw_video_hud(volatile u16*d,u32 f,const struct ClipDescriptor*c,const struct PlayerUI*ui){char a[6],b[6];u32 total,cur,w;if(ui->hud_mode<=0)return;cur=seconds_for_frame(f,c->vblanks_per_frame);total=seconds_for_frame(c->frame_count?c->frame_count-1:0,c->vblanks_per_frame);time5(a,cur);time5(b,total);rect4(d,0,68,120,12,UI_BLACK);text4n(d,3,69,a,5,UI_WHITE);text4(d,24,69,"/",UI_DARK);text4n(d,29,69,b,5,UI_WHITE);rect4(d,4,77,112,2,UI_DARK);w=c->frame_count>1?udiv(f*112u,c->frame_count-1u):112;rect4(d,4,77,w,2,UI_YELLOW);if(ui->muted)text4(d,93,69,"MUTE",UI_RED);}
static void render_frame_with_ui(const u8*p,u32 f,volatile u16*d,const struct ClipDescriptor*c,const struct PlayerUI*ui){render_pixels(p,d);draw_video_hud(d,f,c,ui);}
static void show_rendered_page(u16*page,const u16*pal){wait_vblank();copy_palette(pal);*page^=1;REG_DISPCNT=MODE4|(*page?PAGE:0);}
static void render_and_show(const u8*p,u32 f,u16*page,const struct ClipDescriptor*c,const struct PlayerUI*ui){volatile u16*d=*page?VRAM0:VRAM1;render_frame_with_ui(p,f,d,c,ui);show_rendered_page(page,palette_for_frame(c,f));}

static const int step_table[89]={7,8,9,10,11,12,13,14,16,17,19,21,23,25,28,31,34,37,41,45,50,55,60,66,73,80,88,97,107,118,130,143,157,173,190,209,230,253,279,307,337,371,408,449,494,544,598,658,724,796,876,963,1060,1166,1282,1411,1552,1707,1878,2066,2272,2499,2749,3024,3327,3660,4026,4428,4871,5358,5894,6484,7132,7845,8630,9493,10442,11487,12635,13899,15289,16818,18500,20350,22385,24623,27086,29794,32767};
static const signed char index_table[16]={-1,-1,-1,-1,2,4,6,8,-1,-1,-1,-1,2,4,6,8};
static int ima(u8 code,int*pred,int*idx){int st=step_table[*idx],d=st>>3;if(code&4)d+=st;if(code&2)d+=st>>1;if(code&1)d+=st>>2;if(code&8)*pred-=d;else*pred+=d;*pred=clampi(*pred,-32768,32767);*idx=clampi(*idx+index_table[code&15],0,88);return *pred;}
static void decode_adpcm_range(const u8*audio,u32 start,u32 count,u8*dst){u32 samples=rd32(audio+8),bb=rd32(audio+12),bc=rd32(audio+16),written=0;if(rd32(audio)!=0x31444149u||rd16(audio+6)!=2048){while(written<count)dst[written++]=0;return;}while(written<count&&start<samples){u32 block=start>>11,within=start&2047u,i;const u8*data;int pred,idx;if(block>=bc)break;data=audio+20+block*bb;pred=(int)(s16)rd16(data);idx=clampi(data[2],0,88);if(within==0&&written<count){dst[written++]=(u8)((pred>>8)&255);start++;within=1;}for(i=1;i<2048&&written<count&&start<samples;i++){u32 np=i-1;u8 pack=data[4+(np>>1)],code=(np&1)?(pack>>4):(pack&15);int v=ima(code,&pred,&idx);if(i>=within){dst[written++]=(u8)((v>>8)&255);start++;}}}while(written<count)dst[written++]=0;}
static u16 sound_control(const struct PlayerUI*ui,int reset){u16 v=0x0800;if(!ui->muted&&ui->volume_level){v|=0x0200;if(ui->volume_level>1)v|=0x0004;}if(reset)v|=0x0800;return v;}
static void audio_stop(void){REG_TM0CNT_H=0;REG_DMA1CNT_H=0;REG_SOUNDCNT_H=0x0800;adpcm_active=0;}
static void audio_pause(void){REG_TM0CNT_H=0;}
static void audio_resume(void){REG_TM0CNT_L=0xFC00;REG_TM0CNT_H=0x0080;}
static void playback_timer_stop(void){REG_TM2CNT_H=0;REG_TM3CNT_H=0;}
static void playback_timer_reset(void){playback_timer_stop();REG_TM2CNT_L=0;REG_TM3CNT_L=0;REG_TM3CNT_H=0x0084;REG_TM2CNT_H=0x0083;}
static void playback_timer_pause(void){playback_timer_stop();}
static u32 playback_timer_read(void){u16 a,b,c;do{a=REG_TM3CNT_L;b=REG_TM2CNT_L;c=REG_TM3CNT_L;}while(a!=c);return ((u32)a<<16)|b;}
static u32 seek_value(const struct ClipDescriptor*c,u32 f){u32 v;if(!c->seek_table_offset||f>=c->frame_count)return 0;v=rd32(rom_ptr(c->seek_table_offset+f*4));if(c->audio_codec==AUDIO_CODEC_ADPCM||c->flags&CLIP_FLAG_ADPCM){if(c->audio_sample_count&&v>=c->audio_sample_count)v=c->audio_sample_count-1;return v;}v&=~3u;if(c->audio_size<4)return 0;if(v>c->audio_size-4)v=(c->audio_size-4)&~3u;return v;}
static void audio_dma(const u8*s,int paused,const struct PlayerUI*ui){REG_SOUNDCNT_X=0x0080;REG_SOUNDCNT_L=0;REG_SOUNDBIAS=0x0200;REG_SOUNDCNT_H=sound_control(ui,1);REG_DMA1SAD=(u32)s;REG_DMA1DAD=(u32)&REG_FIFO_A;REG_DMA1CNT_L=4;REG_DMA1CNT_H=0xB640;if(!paused)audio_resume();}
static void audio_start_at(const struct ClipDescriptor*c,u32 v,int paused,const struct PlayerUI*ui){const u8*a=rom_ptr(c->audio_offset);audio_stop();if(c->audio_sample_count&&v>=c->audio_sample_count)v=c->audio_sample_count-1;if(c->audio_codec==AUDIO_CODEC_ADPCM||c->flags&CLIP_FLAG_ADPCM){adpcm_stream=a;adpcm_start_sample=v;adpcm_sample_count=c->audio_sample_count;adpcm_active_half=0;adpcm_next_switch=ADPCM_HALF;decode_adpcm_range(a,v,ADPCM_HALF,adpcm_pcm);decode_adpcm_range(a,v+ADPCM_HALF,ADPCM_HALF,adpcm_pcm+ADPCM_HALF);adpcm_active=1;audio_dma(adpcm_pcm,paused,ui);}else audio_dma(a+(v&~3u),paused,ui);}
static void audio_start_for_frame(const struct ClipDescriptor*c,u32 f,int paused,const struct PlayerUI*ui){audio_start_at(c,seek_value(c,f),paused,ui);}
static void audio_service(void){u32 e,n,r,s,i;if(!adpcm_active)return;e=playback_timer_read();while(e>=adpcm_next_switch){n=adpcm_active_half^1u;REG_DMA1CNT_H=0;REG_DMA1SAD=(u32)(adpcm_pcm+n*ADPCM_HALF);REG_DMA1DAD=(u32)&REG_FIFO_A;REG_DMA1CNT_L=4;REG_DMA1CNT_H=0xB640;adpcm_active_half=n;r=n^1u;s=adpcm_start_sample+adpcm_next_switch+ADPCM_HALF;if(s<adpcm_sample_count)decode_adpcm_range(adpcm_stream,s,ADPCM_HALF,adpcm_pcm+r*ADPCM_HALF);else for(i=0;i<ADPCM_HALF;i++)adpcm_pcm[r*ADPCM_HALF+i]=0;adpcm_next_switch+=ADPCM_HALF;}}
static void audio_apply_state(const struct PlayerUI*ui){REG_SOUNDCNT_H=sound_control(ui,0);}

static void playback_clock_init(struct PlaybackClock*c,u16 vb){c->step_whole=(u32)vb*274u;c->step_remainder=(u32)vb*31u;c->remainder_accum=0;c->next_deadline=0;}
static void playback_clock_advance(struct PlaybackClock*c){c->next_deadline+=c->step_whole;c->remainder_accum+=c->step_remainder;if(c->remainder_accum>=100){c->remainder_accum-=100;c->next_deadline++;}}
static u32 seek_target(u32 f,u32 count,u32 step,int forward){if(step<1)step=1;if(forward){u32 t=f+step;return t<count?t:count-1;}return f>step?f-step:0;}
static int wait_frame_period(u16*prev,u32 deadline,int has_audio,int playlist,enum PlaybackState*state,struct PlayerUI*ui){for(;;){u16 now,pressed;audio_service();if(playback_timer_read()>=deadline)return ACTION_NONE;now=keys_down();pressed=now&~(*prev);*prev=now;if((now&KEY_A)==0)ui->pause_button_latched=0;if((pressed&KEY_A)&&!ui->pause_button_latched){ui->pause_button_latched=1;if(*state==PLAYBACK_RUNNING){playback_timer_pause();if(has_audio)audio_pause();*state=PLAYBACK_PAUSED;}else if(*state==PLAYBACK_PAUSED){*state=PLAYBACK_RESUME_ARMED;return ACTION_RESUME_PENDING;}}if(pressed&KEY_B)return ACTION_RESTART;if(pressed&KEY_START){ui->hud_mode=(ui->hud_mode+1)%3;return ACTION_UI_REFRESH;}if(pressed&KEY_SELECT){ui->muted=!ui->muted;audio_apply_state(ui);return ACTION_UI_REFRESH;}if(pressed&KEY_UP){if(ui->volume_level<2)ui->volume_level++;audio_apply_state(ui);return ACTION_UI_REFRESH;}if(pressed&KEY_DOWN){if(ui->volume_level>0)ui->volume_level--;audio_apply_state(ui);return ACTION_UI_REFRESH;}if(playlist&&(now&KEY_SELECT)&&(pressed&KEY_LEFT))return ACTION_PREV_CLIP;if(playlist&&(now&KEY_SELECT)&&(pressed&KEY_RIGHT))return ACTION_NEXT_CLIP;if(pressed&KEY_L)return ACTION_SEEK_BACK;if(pressed&KEY_R)return ACTION_SEEK_FORWARD;if(*state!=PLAYBACK_RUNNING){if(pressed&KEY_LEFT)return ACTION_FRAME_BACK;if(pressed&KEY_RIGHT)return ACTION_FRAME_FORWARD;}wait_vblank();}}

static void show_help_screen(u16*page,int menu,int playlist){volatile u16*d=*page?VRAM0:VRAM1;clear4(d);set_ui_palette();text4(d,42,2,"CONTROLS",UI_YELLOW);text4(d,3,10,"A PAUSE PLAY",UI_WHITE);text4(d,3,18,menu?"B RETURN MENU":"B RESTART",UI_WHITE);text4(d,3,26,"L R SEEK",UI_WHITE);text4(d,3,34,"START HUD",UI_WHITE);text4(d,3,42,"SELECT MUTE",UI_WHITE);text4(d,3,50,"UP DOWN VOLUME",UI_WHITE);if(playlist)text4(d,3,58,"SELECT+LEFT RIGHT CLIP",UI_WHITE);wait_vblank();*page^=1;REG_DISPCNT=MODE4|(*page?PAGE:0);while(keys_down())wait_vblank();while(!keys_down())wait_vblank();while(keys_down())wait_vblank();}
static int is_menu_mode(const struct GlobalMetadata*m){return m->clip_count>1&&!(m->flags&GLOBAL_FLAG_PLAYLIST);}

static u32 sram_rd(u32 o){return SRAM[o]|((u32)SRAM[o+1]<<8)|((u32)SRAM[o+2]<<16)|((u32)SRAM[o+3]<<24);}
static void sram_wr(u32 o,u32 v){SRAM[o]=(u8)v;SRAM[o+1]=(u8)(v>>8);SRAM[o+2]=(u8)(v>>16);SRAM[o+3]=(u8)(v>>24);}
static void sram_prepare(const struct GlobalMetadata*m){u32 i;if(!(m->flags&GLOBAL_FLAG_RESUME))return;if(sram_rd(0)!=SRAM_MAGIC||sram_rd(4)!=m->clip_count){sram_wr(0,SRAM_MAGIC);sram_wr(4,m->clip_count);sram_wr(8,0);for(i=0;i<m->clip_count&&i<8000;i++)sram_wr(16+i*4,0);}}
static u32 load_menu_selection(const struct GlobalMetadata*m){u32 v;if(!(m->flags&GLOBAL_FLAG_RESUME))return 0;v=sram_rd(8);return v<m->clip_count?v:0;}
static void save_menu_selection(const struct GlobalMetadata*m,u32 v){if(m->flags&GLOBAL_FLAG_RESUME)sram_wr(8,v);}
static void save_position(const struct GlobalMetadata*m,u32 clip,u32 f){if((m->flags&GLOBAL_FLAG_RESUME)&&clip<8000)sram_wr(16+clip*4,(f+1)^SRAM_XOR^clip);}
static void clear_position(const struct GlobalMetadata*m,u32 clip){if((m->flags&GLOBAL_FLAG_RESUME)&&clip<8000)sram_wr(16+clip*4,0);}
static int load_position(const struct GlobalMetadata*m,u32 clip,u32*f){u32 v,d;if(!(m->flags&GLOBAL_FLAG_RESUME)||clip>=8000)return 0;v=sram_rd(16+clip*4);if(!v)return 0;d=v^SRAM_XOR^clip;if(!d)return 0;*f=d-1;return 1;}

static const struct TitleCardHeader *title_card(const struct GlobalMetadata*m){const struct TitleCardHeader*c;if(!m->reserved[1])return 0;c=(const struct TitleCardHeader*)rom_ptr(m->reserved[1]);return c->magic==TITLE_CARD_MAGIC?c:0;}
static void show_title_card(const struct GlobalMetadata*m){const struct TitleCardHeader*c=title_card(m);u32 i,t=0;u16 k;if(!c)return;REG_DISPCNT=FORCE_BLANK;audio_stop();copy16(VRAM0,(const u16*)(c+1),NATIVE_PIXELS);wait_vblank();REG_DISPCNT=MODE3;while(keys_down())wait_vblank();if(c->flags&TITLE_CARD_FLAG_WAIT_A){while(!(keys_down()&KEY_A))wait_vblank();}else while(t<c->duration_vblanks){wait_vblank();t++;k=keys_down();if((c->flags&TITLE_CARD_FLAG_SKIP)&&(k&KEY_A))break;}while(keys_down())wait_vblank();if(c->flags&TITLE_CARD_FLAG_FADE){REG_BLDCNT=0x00FF;for(i=0;i<=16;i++){REG_BLDY=i;wait_vblank();}REG_BLDCNT=REG_BLDY=0;}REG_DISPCNT=FORCE_BLANK;}

static const struct MenuThemeHeader *menu_theme(const struct GlobalMetadata*m){const struct MenuThemeHeader*t;if(!m->reserved[0])return 0;t=(const struct MenuThemeHeader*)rom_ptr(m->reserved[0]);if(t->magic!=MENU_THEME_MAGIC||t->version!=1||t->frame_bytes!=FRAME_BYTES)return 0;return t;}
static void menu_background(volatile u16*d,const struct GlobalMetadata*m,u32 frame){const struct MenuThemeHeader*t=menu_theme(m);u32 i;if(t&&t->frames_offset&&t->frame_count){const u8*s=rom_ptr(t->frames_offset+(frame%t->frame_count)*FRAME_BYTES);render_pixels(s,d);copy_palette((const u16*)rom_ptr(t->palette_offset));PALRAM[UI_WHITE]=t->ui_colour;PALRAM[UI_YELLOW]=t->selected_colour;PALRAM[UI_DARK]=t->outline_colour;}else{clear4(d);set_ui_palette();for(i=0;i<80;i+=8)rect4(d,0,i,120,1,(i&8)?UI_DARK:UI_BLACK);}}
static char media_letter(const struct ClipDescriptor*c){return (c->flags&CLIP_FLAG_MEDIA_AUDIO)?'A':((c->flags&CLIP_FLAG_MEDIA_IMAGE)?'I':'V');}
static u32 menu_total_seconds(const struct GlobalMetadata*m,const struct ClipDescriptor*c){u32 i,t=0;for(i=0;i<m->clip_count;i++){if(c[i].flags&CLIP_FLAG_MEDIA_IMAGE){t+=udiv(c[i].audio_sample_count,1000);}else t+=seconds_for_frame(c[i].frame_count,c[i].vblanks_per_frame);}return t;}
static u32 select_clip_menu(const struct GlobalMetadata*m,const struct ClipDescriptor*c,u32 sel){u16 prev=keys_down();u16 page=1;u32 anim=0,total=menu_total_seconds(m,c);if(m->clip_count<=1)return 0;for(;;){volatile u16*d=page?VRAM0:VRAM1;u32 i,start=(sel/10)*10;char tm[6];menu_background(d,m,anim);text4(d,34,3,"SELECT MEDIA",UI_WHITE);time5(tm,total);text4n(d,96,3,tm,5,UI_DARK);for(i=0;i<10&&start+i<m->clip_count;i++){char tag[4]={'[',media_letter(&c[start+i]),']',0};u32 idx=start+i,y=16+i*6;text4(d,3,y,tag,idx==sel?UI_YELLOW:UI_DARK);text4n(d,18,y,c[idx].title,12,idx==sel?UI_YELLOW:UI_WHITE);}wait_vblank();page^=1;REG_DISPCNT=MODE4|(page?PAGE:0);for(;;){u16 now=keys_down(),p=now&~prev;prev=now;if(p&KEY_UP){sel=sel?sel-1:m->clip_count-1;break;}if(p&KEY_DOWN){sel=(sel+1)%m->clip_count;break;}if(p&KEY_LEFT){sel=sel>=10?sel-10:0;break;}if(p&KEY_RIGHT){sel+10<m->clip_count? (sel+=10):(sel=m->clip_count-1);break;}if(p&KEY_A){save_menu_selection(m,sel);while(keys_down())wait_vblank();return sel;}if((p&KEY_START)&&(p&KEY_SELECT)){show_help_screen(&page,1,0);break;}wait_vblank();anim++;if((anim&15)==0)break;}}}

static const struct MediaMetadata*media_metadata(const struct ClipDescriptor*c){const struct MediaMetadata*m;if(!(c->flags&CLIP_FLAG_MEDIA_META)||!c->video_index_offset)return 0;m=(const struct MediaMetadata*)rom_ptr(c->video_index_offset);return m->magic==MEDIA_META_MAGIC?m:0;}
static void native_overlay(const struct ClipDescriptor*c,u32 frame,int paused,int image,int show){char cur[6],tot[6];u32 sec,total,w;const struct MediaMetadata*meta;if(!show)return;if(image){rect3(VRAM0,0,132,240,28,0);text3n(VRAM0,8,138,c->title,12,0x7FFF);text3n(VRAM0,184,138,"IMAGE",5,0x03FF);return;}rect3(VRAM0,0,108,240,52,0);text3n(VRAM0,8,112,c->title,12,0x7FFF);meta=media_metadata(c);if(meta){text3n(VRAM0,8,124,meta->artist,20,0x5294);text3n(VRAM0,8,136,meta->album,20,0x4210);}sec=seconds_for_frame(frame,c->vblanks_per_frame);total=seconds_for_frame(c->frame_count?c->frame_count-1:0,c->vblanks_per_frame);time5(cur,sec);time5(tot,total);text3n(VRAM0,8,148,cur,5,0x7FFF);text3n(VRAM0,52,148,"/",1,0x4210);text3n(VRAM0,64,148,tot,5,0x7FFF);text3n(VRAM0,184,148,paused?"PAUSE":"PLAY",5,paused?0x03FF:0x03E0);rect3(VRAM0,8,158,224,2,0x2108);w=c->frame_count>1?udiv(frame*224,c->frame_count-1):224;rect3(VRAM0,8,158,w,2,0x03FF);}
static void show_native_art(const struct ClipDescriptor*c,u32 frame,int paused,int image,int show){REG_DISPCNT=FORCE_BLANK;copy16(VRAM0,(const u16*)rom_ptr(c->video_offset),NATIVE_PIXELS);native_overlay(c,frame,paused,image,show);wait_vblank();REG_DISPCNT=MODE3;}

static int play_image(const struct GlobalMetadata*m,const struct ClipDescriptor*c,u32 idx){u16 prev=keys_down();u32 ticks=0,limit=0;int show=1;if(c->audio_sample_count)limit=udiv(c->audio_sample_count*60u,1000u);show_native_art(c,0,0,1,show);while(keys_down())wait_vblank();for(;;){u16 now,p;wait_vblank();ticks++;now=keys_down();p=now&~prev;prev=now;if(p&KEY_B)return is_menu_mode(m)?PLAY_RESULT_RETURN_MENU:PLAY_RESULT_RESTART_CURRENT;if(p&KEY_LEFT)return PLAY_RESULT_PREV_CLIP;if(p&KEY_RIGHT)return PLAY_RESULT_NEXT_CLIP;if(p&KEY_A){show=!show;show_native_art(c,0,0,1,show);}if(limit&&ticks>=limit)return PLAY_RESULT_NEXT_CLIP;save_position(m,idx,0);}}
static int play_audio(const struct GlobalMetadata*m,const struct ClipDescriptor*c,u32 idx,struct PlayerUI*ui){
 u32 f=0,step=c->seek_frame_step?c->seek_frame_step:1,base_sample,paused_sample=0;u16 prev=keys_down();int paused=0,show=1;
 if(load_position(m,idx,&f)&&f>=c->frame_count)f=0;base_sample=seek_value(c,f);show_native_art(c,f,paused,0,show);playback_timer_reset();audio_start_for_frame(c,f,0,ui);while(keys_down())wait_vblank();
 for(;;){u16 now,p;u32 elapsed,abs;wait_vblank();audio_service();now=keys_down();p=now&~prev;prev=now;elapsed=playback_timer_read();abs=base_sample+elapsed;
  if(!paused){while(f+1<c->frame_count&&seek_value(c,f+1)<=abs)f++;if(abs>=c->audio_sample_count){audio_stop();playback_timer_stop();clear_position(m,idx);return (c->flags&CLIP_FLAG_LOOP)?PLAY_RESULT_RESTART_CURRENT:PLAY_RESULT_NEXT_CLIP;}}
  if(p&KEY_A){paused=!paused;if(paused){paused_sample=abs;playback_timer_pause();audio_pause();}else{base_sample=paused_sample;audio_start_at(c,base_sample,0,ui);playback_timer_reset();}show_native_art(c,f,paused,0,show);}
  if(p&KEY_B){audio_stop();playback_timer_stop();save_position(m,idx,f);return is_menu_mode(m)?PLAY_RESULT_RETURN_MENU:PLAY_RESULT_RESTART_CURRENT;}
  if((p&KEY_LEFT)||(p&KEY_L)){f=seek_target(f,c->frame_count,step,0);base_sample=seek_value(c,f);paused_sample=base_sample;audio_start_at(c,base_sample,paused,ui);playback_timer_reset();if(paused)playback_timer_pause();show_native_art(c,f,paused,0,show);save_position(m,idx,f);}
  if((p&KEY_RIGHT)||(p&KEY_R)){f=seek_target(f,c->frame_count,step,1);base_sample=seek_value(c,f);paused_sample=base_sample;audio_start_at(c,base_sample,paused,ui);playback_timer_reset();if(paused)playback_timer_pause();show_native_art(c,f,paused,0,show);save_position(m,idx,f);}
  if(p&KEY_START){show=!show;show_native_art(c,f,paused,0,show);}if(p&KEY_SELECT){ui->muted=!ui->muted;audio_apply_state(ui);}if(p&KEY_UP){if(ui->volume_level<2)ui->volume_level++;audio_apply_state(ui);}if(p&KEY_DOWN){if(ui->volume_level>0)ui->volume_level--;audio_apply_state(ui);}
  if((m->flags&GLOBAL_FLAG_PLAYLIST)&&(now&KEY_SELECT)&&(p&KEY_LEFT)){audio_stop();return PLAY_RESULT_PREV_CLIP;}if((m->flags&GLOBAL_FLAG_PLAYLIST)&&(now&KEY_SELECT)&&(p&KEY_RIGHT)){audio_stop();return PLAY_RESULT_NEXT_CLIP;}if((f&15)==0)native_overlay(c,f,paused,0,show);
 }
}

static int play_video(const struct GlobalMetadata*meta,const struct ClipDescriptor*clip,u32 clip_index,struct PlayerUI*ui){
 u32 frame=0;u8*current=frame_a,*next=frame_b;u16 displayed_page=0,previous_keys=keys_down();enum PlaybackState state=PLAYBACK_RUNNING;struct PlaybackClock clock;int has_audio=(clip->flags&CLIP_FLAG_AUDIO)!=0;int next_frame_valid = 0;int at_end=0;
 if(load_position(meta,clip_index,&frame)&&frame>=clip->frame_count)frame=0;load_frame_pixels(clip,frame,current);REG_DISPCNT=FORCE_BLANK;set_ui_palette();render_and_show(current,frame,&displayed_page,clip,ui);playback_clock_init(&clock,clip->vblanks_per_frame);playback_timer_reset();playback_clock_advance(&clock);if(has_audio)audio_start_for_frame(clip,frame,0,ui);ui->pause_button_latched = (previous_keys & KEY_A) != 0u;
 for(;;){
  if(state!=PLAYBACK_RUNNING){int action=wait_frame_period(&previous_keys,0xFFFFFFFFu,has_audio,(meta->flags&GLOBAL_FLAG_PLAYLIST)!=0,&state,ui);if(action==ACTION_RESTART){playback_timer_stop();audio_stop();save_position(meta,clip_index,frame);return is_menu_mode(meta)?PLAY_RESULT_RETURN_MENU:PLAY_RESULT_RESTART_CURRENT;}if(action==ACTION_PREV_CLIP){playback_timer_stop();audio_stop();save_position(meta,clip_index,frame);return PLAY_RESULT_PREV_CLIP;}if(action==ACTION_NEXT_CLIP){playback_timer_stop();audio_stop();save_position(meta,clip_index,frame);return PLAY_RESULT_NEXT_CLIP;}if(action==ACTION_FRAME_BACK||action==ACTION_FRAME_FORWARD||action==ACTION_SEEK_BACK||action==ACTION_SEEK_FORWARD){u32 target=(action==ACTION_FRAME_BACK)?(frame?frame-1:0):(action==ACTION_FRAME_FORWARD)?(frame+1<clip->frame_count?frame+1:frame):seek_target(frame,clip->frame_count,clip->seek_frame_step,action==ACTION_SEEK_FORWARD);if(target!=frame){load_frame_pixels(clip,target,current);frame=target;render_and_show(current,frame,&displayed_page,clip,ui);next_frame_valid = 0; state = PLAYBACK_PAUSED; save_position(meta,clip_index,frame);}continue;}if(action==ACTION_RESUME_PENDING){u32 resume_frame=frame+1<clip->frame_count?frame+1:frame;if(state != PLAYBACK_RESUME_ARMED) continue;if(frame+1<clip->frame_count && !next_frame_valid) { state = PLAYBACK_PAUSED; continue; }playback_clock_init(&clock,clip->vblanks_per_frame);playback_clock_advance(&clock);if(has_audio) audio_start_for_frame(clip,resume_frame,1,ui);wait_vblank();if(frame+1<clip->frame_count){show_rendered_page(&displayed_page,palette_for_frame(clip,resume_frame));{u8*tmp=current;current=next;next=tmp;}frame=resume_frame;next_frame_valid = 0;}playback_timer_reset();if(has_audio) audio_resume();state = PLAYBACK_RUNNING;previous_keys=keys_down();continue;}if(action==ACTION_UI_REFRESH){render_and_show(current,frame,&displayed_page,clip,ui);next_frame_valid = 0;}continue;}
  {int has_next=frame+1<clip->frame_count;volatile u16*back=displayed_page?VRAM0:VRAM1;int action;if(has_next && !next_frame_valid){load_next_pixels(clip,frame+1,current,next);render_frame_with_ui(next,frame+1,back,clip,ui);next_frame_valid = 1;}action=wait_frame_period(&previous_keys,clock.next_deadline,has_audio,(meta->flags&GLOBAL_FLAG_PLAYLIST)!=0,&state,ui);if(action==ACTION_RESTART){playback_timer_stop();audio_stop();save_position(meta,clip_index,frame);return is_menu_mode(meta)?PLAY_RESULT_RETURN_MENU:PLAY_RESULT_RESTART_CURRENT;}if(action==ACTION_PREV_CLIP){playback_timer_stop();audio_stop();save_position(meta,clip_index,frame);return PLAY_RESULT_PREV_CLIP;}if(action==ACTION_NEXT_CLIP){playback_timer_stop();audio_stop();save_position(meta,clip_index,frame);return PLAY_RESULT_NEXT_CLIP;}if(action==ACTION_RESUME_PENDING)continue;if(action==ACTION_UI_REFRESH){render_and_show(current,frame,&displayed_page,clip,ui);next_frame_valid = 0;continue;}if(action==ACTION_SEEK_BACK||action==ACTION_SEEK_FORWARD){u32 target=seek_target(frame,clip->frame_count,clip->seek_frame_step,action==ACTION_SEEK_FORWARD);load_frame_pixels(clip,target,current);frame=target;render_and_show(current,frame,&displayed_page,clip,ui);next_frame_valid = 0;playback_clock_init(&clock,clip->vblanks_per_frame);playback_timer_reset();playback_clock_advance(&clock);if(has_audio)audio_start_for_frame(clip,frame,0,ui);save_position(meta,clip_index,frame);continue;}if(state!=PLAYBACK_RUNNING)continue;if(has_next){show_rendered_page(&displayed_page,palette_for_frame(clip,frame+1));{u8*tmp=current;current=next;next=tmp;}++frame; next_frame_valid = 0; playback_clock_advance(&clock);if((frame&31)==0)save_position(meta,clip_index,frame);}else{if(at_end||!(clip->flags&CLIP_FLAG_LOOP)){playback_timer_stop();audio_stop();clear_position(meta,clip_index);return (clip->flags&CLIP_FLAG_LOOP)?PLAY_RESULT_RESTART_CURRENT:PLAY_RESULT_NEXT_CLIP;}at_end=1;}}
 }
}

static int play_media(const struct GlobalMetadata*m,const struct ClipDescriptor*c,u32 idx,struct PlayerUI*ui){if(c->flags&CLIP_FLAG_MEDIA_IMAGE)return play_image(m,c,idx);if(c->flags&CLIP_FLAG_MEDIA_AUDIO)return play_audio(m,c,idx,ui);return play_video(m,c,idx,ui);}

void main(void){const struct GlobalMetadata*m=&gba_video_metadata;const struct ClipDescriptor*clips;u32 selected=0;struct PlayerUI ui={0,2,2,2,0,0,0,0,0,0,0,0,0,0,0};if(m->magic!=GBV5_MAGIC||m->version!=5||!m->clip_count)for(;;){}clips=(const struct ClipDescriptor*)rom_ptr(m->clip_table_offset);sram_prepare(m);show_title_card(m);selected=load_menu_selection(m);for(;;){int r;if(is_menu_mode(m))selected=select_clip_menu(m,clips,selected);save_menu_selection(m,selected);r=play_media(m,&clips[selected],selected,&ui);if(r==PLAY_RESULT_RETURN_MENU){continue;}if(r==PLAY_RESULT_PREV_CLIP){selected=selected?selected-1:m->clip_count-1;continue;}if(r==PLAY_RESULT_NEXT_CLIP){selected=(selected+1)%m->clip_count;if(m->clip_count==1&&!(clips[0].flags&CLIP_FLAG_LOOP)){selected=0;}continue;}if(r==PLAY_RESULT_RESTART_CURRENT){continue;}}}

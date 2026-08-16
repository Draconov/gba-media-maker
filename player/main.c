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
#define REG_WAITCNT REG16(0x04000204)
#define PALRAM ((volatile u16 *)0x05000000)
#define OBJ_PALRAM ((volatile u16 *)0x05000200)
#define OBJ_TILE_VRAM ((volatile u16 *)0x06014000)
#define OAM ((volatile u16 *)0x07000000)
#define VRAM0 ((volatile u16 *)0x06000000)
#define VRAM1 ((volatile u16 *)0x0600A000)
#define SRAM ((volatile u8 *)0x0E000000)
#define ROM_BASE 0x08000000u
#define MODE3 0x0403u
#define MODE4 0x0404u
#define MODE4_OBJ (MODE4|0x1040u)
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
#define MEDIA_META_MAGIC_V1 0x31444D4Du
#define MEDIA_META_MAGIC_V2 0x32444D4Du
#define HUD_HOLD_VBLANKS 6u
#define VOLUME_HOLD_VBLANKS 6u
#define SEEK_HOLD_VBLANKS 6u
#define SEEK_REPEAT_VBLANKS 18u
#define AUDIO_HUD_HOLD_VBLANKS 24u
#define AUDIO_VOLUME_HOLD_VBLANKS 24u
#define AUDIO_SEEK_HOLD_VBLANKS 24u
#define AUDIO_CODEC_PCM 1u
#define AUDIO_CODEC_ADPCM 2u
#define GBV5_MAGIC 0x35564247u
#define MENU_THEME_MAGIC 0x3148544Du
#define MENU_THEME_STATIC 0u
#define MENU_THEME_SHIMMER 1u
#define MENU_THEME_FRAMES 2u
#define MENU_THEME_FLAG_OUTLINE 0x0001u
#define MENU_ARROW_BLINK_VBLANKS 24u
#define MENU_ROWS 10u
#define MENU_ARROW_OAM_INDEX 0u
#define MENU_ARROW_TILE_INDEX 512u
#define OBJ_DISABLE 0x0200u
#define OBJ_SIZE_16 0x4000u
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

#include "menu_background_data.h"

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
#define PLAY_RESULT_NEXT_CLIP_DIRECT 4

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

struct MenuThemeHeader {
 u32 magic; u16 version, kind; u32 palette_offset, frames_offset; u16 frame_count, frame_vblanks, flags, ui_colour, selected_colour, outline_colour;
 u16 shimmer_source_start, shimmer_count, shimmer_target1, shimmer_interval1, shimmer_target2, shimmer_interval2, shimmer_phases, reserved0;
 u32 frame_bytes, data_size, reserved[3];
};
struct TitleCardHeader { u32 magic; u16 version, flags; u32 pixel_bytes, duration_vblanks, reserved[4]; };
struct PlayerUI { int muted, volume_level, hud_mode, hud_last_visible,paused_ui; u16 hud_timer,mute_timer,volume_timer,seek_timer; int seek_direction,seek_hold_direction; u16 seek_hold_counter; int help_combo_latched,pause_button_latched,start_pending,select_pending; };
struct PlaybackClock { u32 next_deadline, step_whole, step_remainder, remainder_accum; };
extern const struct GlobalMetadata gba_video_metadata;

static const struct MenuThemeHeader *active_menu_theme;
static int active_menu_outline;

static u8 frame_a[FRAME_BYTES], frame_b[FRAME_BYTES];
#define ADPCM_HALF 4096u
static u8 adpcm_pcm[ADPCM_HALF*2u] __attribute__((aligned(4)));
static const u8 *adpcm_stream; static u32 adpcm_start_sample,adpcm_next_switch,adpcm_active_half,adpcm_sample_count; static int adpcm_active;
static u32 pcm_guard_ticks; static int pcm_guard_active;

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
 case ':':return 0x0410;case ';':return 0x200A;case '/': return 0x12A4u;case '\\':return 0x4489;case '-':return 0x01C0;case '_':return 0x0007;case '+':return 0x05D0;case '=':return 0x0E38;case '.':return 0x0002;case ',':return 0x000A;case '!':return 0x2492;case '?':return 0x72C2;case '(':return 0x2488;case ')':return 0x1112;case '[':return 0x6926;case ']':return 0x324B;case '&':return 0x2AAE;case '%':return 0x5295;case '#':return 0x5F7D;case '@':return 0x7BE7;case '\'':return 0x2400;case '"':return 0x5A00;case '>':return 0x23CA;case '<':return 0x29E2;
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
static void dim3(volatile u16*d,u32 y,u32 h){u32 n=h*240u;d+=y*240u;while(n--){*d=(u16)((*d&0x7BDEu)>>1);d++;}}
static void char3(volatile u16*d,u32 x,u32 y,u8 c,u16 col){u16 bits=glyph_bits(c);u32 r,k,xx,yy;for(r=0;r<5;r++)for(k=0;k<3;k++)if(bits&(1u<<(14u-(r*3u+k))))for(yy=0;yy<2;yy++)for(xx=0;xx<2;xx++)p3(d,x+k*2u+xx,y+r*2u+yy,col);}
static void text3n(volatile u16*d,u32 x,u32 y,const char*s,u32 max,u16 col){u32 i=0;while(i<max&&s[i]){char3(d,x,y,(u8)s[i],col);x+=8;i++;}}
static u32 text_len(const char*s){u32 n=0;while(s&&s[n])n++;return n;}
static void text3(volatile u16*d,u32 x,u32 y,const char*s,u16 col){text3n(d,x,y,s,64,col);}
static void block3(volatile u16*d,u32 x,u32 y,u16 col){rect3(d,x,y,2,2,col);}
static void speaker3(volatile u16*d,u32 x,u32 y,int crossed){
 block3(d,x,y+6,0x7FFF);block3(d,x+2,y+4,0x7FFF);block3(d,x+2,y+6,0x7FFF);block3(d,x+2,y+8,0x7FFF);block3(d,x+4,y+2,0x7FFF);block3(d,x+4,y+4,0x7FFF);block3(d,x+4,y+6,0x7FFF);block3(d,x+4,y+8,0x7FFF);block3(d,x+4,y+10,0x7FFF);
 if(crossed){block3(d,x+10,y+2,0x001F);block3(d,x+18,y+2,0x001F);block3(d,x+12,y+4,0x001F);block3(d,x+16,y+4,0x001F);block3(d,x+14,y+6,0x001F);block3(d,x+12,y+8,0x001F);block3(d,x+16,y+8,0x001F);block3(d,x+10,y+10,0x001F);block3(d,x+18,y+10,0x001F);}
 else{block3(d,x+10,y+6,0x03E0);block3(d,x+10,y+8,0x03E0);block3(d,x+12,y+8,0x03E0);block3(d,x+12,y+10,0x03E0);block3(d,x+14,y+6,0x03E0);block3(d,x+14,y+8,0x03E0);block3(d,x+16,y+4,0x03E0);block3(d,x+16,y+6,0x03E0);block3(d,x+18,y+2,0x03E0);block3(d,x+18,y+4,0x03E0);}
}
static void mute_badge3(volatile u16*d,int muted){rect3(d,214,6,24,14,0);speaker3(d,216,6,muted);}
static void volume_badge3(volatile u16*d,int level){const char*t=level==2?"V100":(level==1?"V50":"V0");u32 tw=text_len(t)*8u-2u,bw=tw+4u,x=238u-bw;rect3(d,x,6,bw,14,0);text3(d,x+2,8,t,0x7FFF);}
static u32 seconds_for_frame(u32 f,u16 vb){ return udiv(f*(u32)vb*1000u,59728u); }
static void time5(char*out,u32 sec){u32 m=udiv(sec,60),s=sec-m*60;out[0]=(char)('0'+(m/10)%10);out[1]=(char)('0'+m%10);out[2]=':';out[3]=(char)('0'+s/10);out[4]=(char)('0'+s%10);out[5]=0;}

static const u16 *palette_for_frame(const struct ClipDescriptor*c,u32 f){u32 i=0;if(c->palette_count>1&&c->palette_index_offset){i=rd16(rom_ptr(c->palette_index_offset+f*2));if(i>=c->palette_count)i=0;}return (const u16*)rom_ptr(c->palette_offset+i*512u);}
static void apply_delta(const u8*p,u32 n,u8*d){u32 pos=0,o=0;while(o+4<=n&&pos<FRAME_BYTES){u32 skip=rd16(p+o),run=rd16(p+o+2);o+=4;pos+=skip;if(pos>FRAME_BYTES)pos=FRAME_BYTES;if(run>FRAME_BYTES-pos)run=FRAME_BYTES-pos;if(o+run>n)run=n-o;copy8(d+pos,p+o,run);pos+=run;o+=run;}}
static const u8 *record(const struct ClipDescriptor*c,u32 f){return rom_ptr(c->video_offset+rd32(rom_ptr(c->video_index_offset+f*4)));}
static void load_frame_pixels(const struct ClipDescriptor*c,u32 f,u8*d){u32 b;const u8*r;if(!(c->flags&CLIP_FLAG_COMPRESSED)){copy8(d,rom_ptr(c->video_offset+f*FRAME_BYTES),FRAME_BYTES);return;}b=f;while(b){r=record(c,b);if(rd32(r)==0)break;b--;}r=record(c,b);if(rd32(r)==0&&rd32(r+4)>=FRAME_BYTES)copy8(d,r+8,FRAME_BYTES);else{u32 i;for(i=0;i<FRAME_BYTES;i++)d[i]=0;}while(b<f){b++;r=record(c,b);if(rd32(r)==0)copy8(d,r+8,FRAME_BYTES);else apply_delta(r+8,rd32(r+4),d);}}
static void load_next_pixels(const struct ClipDescriptor*c,u32 f,const u8*cur,u8*d){const u8*r;if(!(c->flags&CLIP_FLAG_COMPRESSED)){copy8(d,rom_ptr(c->video_offset+f*FRAME_BYTES),FRAME_BYTES);return;}copy8(d,cur,FRAME_BYTES);r=record(c,f);if(rd32(r)==0)copy8(d,r+8,FRAME_BYTES);else apply_delta(r+8,rd32(r+4),d);}
static void render_pixels_rows(const u8*s,volatile u16*d,u32 rows){u32 y,x;if(rows>80u)rows=80u;for(y=0;y<rows;y++){volatile u16*r0=d+y*240u,*r1=r0+120;for(x=0;x<120;x++){u16 p=s[y*120+x];p|=p<<8;r0[x]=p;r1[x]=p;}}}
static void render_pixels(const u8*s,volatile u16*d){render_pixels_rows(s,d,80u);}
static void copy_palette(const u16*p){u32 i;for(i=0;i<256;i++)PALRAM[i]=p[i];}
static void speaker4(volatile u16*d,u32 x,u32 y,int crossed){p4(d,x,y+3,UI_WHITE);p4(d,x+1,y+2,UI_WHITE);p4(d,x+1,y+3,UI_WHITE);p4(d,x+1,y+4,UI_WHITE);p4(d,x+2,y+1,UI_WHITE);p4(d,x+2,y+2,UI_WHITE);p4(d,x+2,y+3,UI_WHITE);p4(d,x+2,y+4,UI_WHITE);p4(d,x+2,y+5,UI_WHITE);if(crossed){p4(d,x+5,y+1,UI_RED);p4(d,x+9,y+1,UI_RED);p4(d,x+6,y+2,UI_RED);p4(d,x+8,y+2,UI_RED);p4(d,x+7,y+3,UI_RED);p4(d,x+6,y+4,UI_RED);p4(d,x+8,y+4,UI_RED);p4(d,x+5,y+5,UI_RED);p4(d,x+9,y+5,UI_RED);}else{p4(d,x+5,y+3,UI_GREEN);p4(d,x+5,y+4,UI_GREEN);p4(d,x+6,y+4,UI_GREEN);p4(d,x+6,y+5,UI_GREEN);p4(d,x+7,y+3,UI_GREEN);p4(d,x+7,y+4,UI_GREEN);p4(d,x+8,y+2,UI_GREEN);p4(d,x+8,y+3,UI_GREEN);p4(d,x+9,y+1,UI_GREEN);p4(d,x+9,y+2,UI_GREEN);}}
static void mute_badge4(volatile u16*d,int muted){rect4(d,107,3,12,7,UI_BLACK);speaker4(d,108,3,muted);}
static void volume_badge4(volatile u16*d,int level){const char*t=level==2?"V100":(level==1?"V50":"V0");u32 n=text_len(t),tw=n*4u-1u,bw=tw+2u,x=119u-bw;rect4(d,x,3,bw,7,UI_BLACK);text4(d,x+1,4,t,UI_WHITE);}
static void frame6(char*out,u32 f){int i;if(f>99999u)f=99999u;out[0]='F';for(i=5;i;i--){out[i]=(char)('0'+f%10u);f=udiv(f,10u);}}
static const u8 loop_icon_rows[6]={92,98,112,7,35,29};
static const u8 seek_arrow_rows[7]={0,4,6,127,127,6,4};
static void bitmap4(volatile u16*d,u32 x,u32 y,const u8*rows,u32 h,int flip){u32 r,k;for(r=0;r<h;r++)for(k=0;k<7;k++)if(rows[r]&(1u<<k))p4(d,x+(flip?6u-k:k),y+r,UI_YELLOW);}
static void seek_badge4(volatile u16*d,int dir,u32 sec){char n[2];u32 digits=sec>=10u?2u:1u,nw=digits==2u?7u:3u,bw=7u+2u+nw+4u,bx=(120u-bw)/2u,cx=bx+2u;if(digits==2u){n[0]=(char)('0'+udiv(sec,10u)%10u);n[1]=(char)('0'+sec%10u);}else n[0]=(char)('0'+sec%10u);rect4(d,bx,32,bw,10,UI_BLACK);if(dir<0){bitmap4(d,cx,33,seek_arrow_rows,7,0);text4n(d,cx+9u,34,n,digits,UI_WHITE);}else{text4n(d,cx,34,n,digits,UI_WHITE);bitmap4(d,cx+nw+2u,33,seek_arrow_rows,7,1);}}
static void draw_video_hud(volatile u16*d,u32 f,const struct ClipDescriptor*c,const struct PlayerUI*ui){char a[6],b[6],fr[6];u32 total,cur,w,x;int mode=ui->hud_mode;if((ui->hud_timer||ui->paused_ui)&&mode<2)mode=2;if(mode<=0)return;cur=seconds_for_frame(f,c->vblanks_per_frame);total=seconds_for_frame(c->frame_count?c->frame_count-1u:0u,c->vblanks_per_frame);time5(a,cur);time5(b,total);x=mode==1?38u:3u;rect4(d,0,mode==1?68u:67u,120,mode==1?8u:13u,UI_BLACK);text4n(d,x,69,a,5,UI_WHITE);text4(d,x+20u,69,"/",UI_WHITE);text4n(d,x+24u,69,b,5,UI_WHITE);if(mode==1)return;frame6(fr,f+1u);text4n(d,51,69,fr,6,UI_WHITE);rect4(d,5,77,110,2,UI_DARK);w=c->frame_count>1?udiv(f*110u,c->frame_count-1u):110u;if(w>110u)w=110u;if(w)rect4(d,5,77,w,2,UI_YELLOW);if(c->flags&CLIP_FLAG_LOOP)bitmap4(d,108,69,loop_icon_rows,6,0);}
static void render_frame_with_ui(const u8*p,u32 f,volatile u16*d,const struct ClipDescriptor*c,const struct PlayerUI*ui){int has_audio=(c->flags&CLIP_FLAG_AUDIO)!=0;int mode=ui->hud_mode;if((ui->hud_timer||ui->paused_ui)&&mode<2)mode=2;render_pixels_rows(p,d,mode==2?67u:80u);draw_video_hud(d,f,c,ui);if(has_audio&&ui->mute_timer)mute_badge4(d,ui->muted);if(has_audio&&ui->volume_timer)volume_badge4(d,ui->volume_level);if(ui->seek_timer&&ui->seek_direction)seek_badge4(d,ui->seek_direction,c->seek_seconds?c->seek_seconds:5u);}
static void show_rendered_page(u16*page,const u16*pal){copy_palette(pal);*page^=1;REG_DISPCNT=MODE4|(*page?PAGE:0);}
static void show_rendered_page_synced(u16*page,const u16*pal){wait_vblank();show_rendered_page(page,pal);}
static void render_and_show(const u8*p,u32 f,u16*page,const struct ClipDescriptor*c,const struct PlayerUI*ui){volatile u16*d=*page?VRAM0:VRAM1;render_frame_with_ui(p,f,d,c,ui);show_rendered_page_synced(page,palette_for_frame(c,f));}

static const int step_table[89]={7,8,9,10,11,12,13,14,16,17,19,21,23,25,28,31,34,37,41,45,50,55,60,66,73,80,88,97,107,118,130,143,157,173,190,209,230,253,279,307,337,371,408,449,494,544,598,658,724,796,876,963,1060,1166,1282,1411,1552,1707,1878,2066,2272,2499,2749,3024,3327,3660,4026,4428,4871,5358,5894,6484,7132,7845,8630,9493,10442,11487,12635,13899,15289,16818,18500,20350,22385,24623,27086,29794,32767};
static const signed char index_table[16]={-1,-1,-1,-1,2,4,6,8,-1,-1,-1,-1,2,4,6,8};
static int ima(u8 code,int*pred,int*idx){int st=step_table[*idx],d=st>>3;if(code&4)d+=st;if(code&2)d+=st>>1;if(code&1)d+=st>>2;if(code&8)*pred-=d;else*pred+=d;*pred=clampi(*pred,-32768,32767);*idx=clampi(*idx+index_table[code&15],0,88);return *pred;}
static void decode_adpcm_range(const u8*audio,u32 start,u32 count,u8*dst){u32 samples=rd32(audio+8),bb=rd32(audio+12),bc=rd32(audio+16),written=0;if(rd32(audio)!=0x31444149u||rd16(audio+6)!=2048){while(written<count)dst[written++]=0;return;}while(written<count&&start<samples){u32 block=start>>11,within=start&2047u,i;const u8*data;int pred,idx;if(block>=bc)break;data=audio+20+block*bb;pred=(int)(s16)rd16(data);idx=clampi(data[2],0,88);if(within==0&&written<count){dst[written++]=(u8)((pred>>8)&255);start++;within=1;}for(i=1;i<2048&&written<count&&start<samples;i++){u32 np=i-1;u8 pack=data[4+(np>>1)],code=(np&1)?(pack>>4):(pack&15);int v=ima(code,&pred,&idx);if(i>=within){dst[written++]=(u8)((v>>8)&255);start++;}}}while(written<count)dst[written++]=0;}
static u16 sound_control(const struct PlayerUI*ui,int reset){u16 v=reset?0x0800u:0u;if(!ui->muted&&ui->volume_level>0)v|=0x0300u;if(ui->volume_level>=2)v|=0x0004u;return v;}
static void audio_stop(void){REG_TM0CNT_H=0;REG_DMA1CNT_H=0;REG_SOUNDCNT_H=0x0800;adpcm_active=0;pcm_guard_active=0;pcm_guard_ticks=0;}
static void audio_pause(void){REG_TM0CNT_H=0;}
static void audio_resume(void){REG_TM0CNT_L=0xFC00;REG_TM0CNT_H=0x0080;}
static void playback_timer_stop(void){REG_TM2CNT_H=0;REG_TM3CNT_H=0;}
static void playback_timer_reset(void){playback_timer_stop();REG_TM2CNT_L=0;REG_TM3CNT_L=0;REG_TM3CNT_H=0x0084;REG_TM2CNT_H=0x0083;}
static void playback_timer_pause(void){playback_timer_stop();}
static u32 playback_timer_read(void){u16 a,b,c;do{a=REG_TM3CNT_L;b=REG_TM2CNT_L;c=REG_TM3CNT_L;}while(a!=c);return ((u32)a<<16)|b;}
static u32 seek_value(const struct ClipDescriptor*c,u32 f){u32 v;if(!c->seek_table_offset||f>=c->frame_count)return 0;v=rd32(rom_ptr(c->seek_table_offset+f*4));if(c->audio_codec==AUDIO_CODEC_ADPCM||c->flags&CLIP_FLAG_ADPCM){if(c->audio_sample_count&&v>=c->audio_sample_count)v=c->audio_sample_count-1;return v;}v&=~3u;if(c->audio_size<4)return 0;if(v>c->audio_size-4)v=(c->audio_size-4)&~3u;return v;}
static void audio_dma(const u8*s,int paused,const struct PlayerUI*ui){REG_SOUNDCNT_X=0x0080;REG_SOUNDCNT_L=0;REG_SOUNDBIAS=0x0200;REG_SOUNDCNT_H=sound_control(ui,1);REG_DMA1SAD=(u32)s;REG_DMA1DAD=(u32)&REG_FIFO_A;REG_DMA1CNT_L=4;REG_DMA1CNT_H=0xB640;if(!paused)audio_resume();}
static void audio_start_at(const struct ClipDescriptor*c,u32 v,int paused,const struct PlayerUI*ui){const u8*a=rom_ptr(c->audio_offset);audio_stop();if(c->audio_sample_count&&v>=c->audio_sample_count)v=c->audio_sample_count-1;if(c->audio_codec==AUDIO_CODEC_ADPCM||c->flags&CLIP_FLAG_ADPCM){adpcm_stream=a;adpcm_start_sample=v;adpcm_sample_count=c->audio_sample_count;adpcm_active_half=0;adpcm_next_switch=ADPCM_HALF;decode_adpcm_range(a,v,ADPCM_HALF,adpcm_pcm);decode_adpcm_range(a,v+ADPCM_HALF,ADPCM_HALF,adpcm_pcm+ADPCM_HALF);adpcm_active=1;audio_dma(adpcm_pcm,paused,ui);}else{u32 start=v&~3u;u32 end=c->audio_sample_count?c->audio_sample_count:c->audio_size;if(end>start){pcm_guard_ticks=end-start;pcm_guard_active=1;}audio_dma(a+start,paused,ui);}}
static void audio_start_for_frame(const struct ClipDescriptor*c,u32 f,int paused,const struct PlayerUI*ui){audio_start_at(c,seek_value(c,f),paused,ui);}
static void audio_service(void){u32 e,n,r,s,i;e=playback_timer_read();if(pcm_guard_active&&e>=pcm_guard_ticks){REG_TM0CNT_H=0;REG_DMA1CNT_H=0;REG_SOUNDCNT_H=0x0800;pcm_guard_active=0;}if(!adpcm_active)return;while(e>=adpcm_next_switch){n=adpcm_active_half^1u;REG_DMA1CNT_H=0;REG_DMA1SAD=(u32)(adpcm_pcm+n*ADPCM_HALF);REG_DMA1DAD=(u32)&REG_FIFO_A;REG_DMA1CNT_L=4;REG_DMA1CNT_H=0xB640;adpcm_active_half=n;r=n^1u;s=adpcm_start_sample+adpcm_next_switch+ADPCM_HALF;if(s<adpcm_sample_count)decode_adpcm_range(adpcm_stream,s,ADPCM_HALF,adpcm_pcm+r*ADPCM_HALF);else for(i=0;i<ADPCM_HALF;i++)adpcm_pcm[r*ADPCM_HALF+i]=0;adpcm_next_switch+=ADPCM_HALF;}}
static void audio_apply_state(const struct PlayerUI*ui){REG_SOUNDCNT_H=sound_control(ui,0);}

static void playback_clock_init(struct PlaybackClock*c,u16 vb){c->step_whole=(u32)vb*274u;c->step_remainder=(u32)vb*31u;c->remainder_accum=0;c->next_deadline=0;}
static void playback_clock_advance(struct PlaybackClock*c){c->next_deadline+=c->step_whole;c->remainder_accum+=c->step_remainder;if(c->remainder_accum>=100){c->remainder_accum-=100;c->next_deadline++;}}
static u32 seek_target(u32 f,u32 count,u32 step,int forward){if(step<1)step=1;if(forward){u32 t=f+step;return t<count?t:count-1;}return f>step?f-step:0;}
static void start_seek_feedback(struct PlayerUI*ui,int dir){ui->seek_direction=dir;ui->seek_timer=SEEK_HOLD_VBLANKS;ui->hud_timer=HUD_HOLD_VBLANKS;}
static void start_audio_seek_feedback(struct PlayerUI*ui,int dir){ui->seek_direction=dir;ui->seek_timer=AUDIO_SEEK_HOLD_VBLANKS;ui->hud_timer=AUDIO_HUD_HOLD_VBLANKS;}
static int tick_ui_timers(struct PlayerUI*ui);
static void cycle_hud(struct PlayerUI*ui){ui->hud_mode++;if(ui->hud_mode>2)ui->hud_mode=0;if(ui->hud_mode>0)ui->hud_last_visible=ui->hud_mode;}
static int held_seek_action(u16 now,u16 pressed,int paused,struct PlayerUI*ui){int dir=0;if((now&KEY_LEFT)&&!(now&KEY_RIGHT))dir=-1;else if((now&KEY_RIGHT)&&!(now&KEY_LEFT))dir=1;if(!dir){ui->seek_hold_direction=0;ui->seek_hold_counter=0;return ACTION_NONE;}if(ui->seek_hold_direction!=dir||(dir<0?(pressed&KEY_LEFT):(pressed&KEY_RIGHT))){ui->seek_hold_direction=dir;ui->seek_hold_counter=SEEK_REPEAT_VBLANKS;return paused?(dir<0?ACTION_FRAME_BACK:ACTION_FRAME_FORWARD):(dir<0?ACTION_SEEK_BACK:ACTION_SEEK_FORWARD);}if(ui->seek_hold_counter>0)ui->seek_hold_counter--;if(ui->seek_hold_counter==0){ui->seek_hold_counter=SEEK_REPEAT_VBLANKS;return paused?(dir<0?ACTION_FRAME_BACK:ACTION_FRAME_FORWARD):(dir<0?ACTION_SEEK_BACK:ACTION_SEEK_FORWARD);}return ACTION_NONE;}
static int common_combo_action(u16 now,u16 pressed,int can_change,int audio_controls,u16 mute_hold,struct PlayerUI*ui){if((now&(KEY_START|KEY_SELECT))==(KEY_START|KEY_SELECT)){ui->start_pending=0;ui->select_pending=0;if(!ui->help_combo_latched){ui->help_combo_latched=1;return ACTION_HELP;}return ACTION_NONE;}ui->help_combo_latched=0;if(can_change&&(pressed&KEY_L)&&!(now&KEY_R))return ACTION_PREV_CLIP;if(can_change&&(pressed&KEY_R)&&!(now&KEY_L))return ACTION_NEXT_CLIP;if((pressed&KEY_START)&&!(now&KEY_SELECT))ui->start_pending=1;if(audio_controls&&(pressed&KEY_SELECT)&&!(now&KEY_START))ui->select_pending=1;if(!audio_controls)ui->select_pending=0;if(ui->start_pending&&!(now&KEY_START)){ui->start_pending=0;cycle_hud(ui);return ACTION_UI_REFRESH;}if(ui->select_pending&&!(now&KEY_SELECT)){ui->select_pending=0;ui->muted=!ui->muted;ui->mute_timer=mute_hold;audio_apply_state(ui);return ACTION_UI_REFRESH;}return ACTION_NONE;}
static int wait_frame_period(u16*prev,u32 deadline,int has_audio,int can_change,enum PlaybackState*state,struct PlayerUI*ui){for(;;){u16 now,pressed;int action;audio_service();now=keys_down();pressed=now&~(*prev);*prev=now;if((now&KEY_A)==0)ui->pause_button_latched=0;action=common_combo_action(now,pressed,can_change,has_audio,HUD_HOLD_VBLANKS,ui);if(action!=ACTION_NONE)return action;if((pressed&KEY_A)&&!ui->pause_button_latched){ui->pause_button_latched=1;if(*state==PLAYBACK_RUNNING){playback_timer_pause();if(has_audio)audio_pause();*state=PLAYBACK_PAUSED;ui->paused_ui=1;return ACTION_UI_REFRESH;}else if(*state==PLAYBACK_PAUSED){ui->paused_ui=0;*state=PLAYBACK_RESUME_ARMED;return ACTION_RESUME_PENDING;}}if(pressed&KEY_B)return ACTION_RESTART;if(has_audio&&(pressed&KEY_UP)){if(ui->volume_level<2)ui->volume_level++;ui->volume_timer=VOLUME_HOLD_VBLANKS;audio_apply_state(ui);return ACTION_UI_REFRESH;}if(has_audio&&(pressed&KEY_DOWN)){if(ui->volume_level>0)ui->volume_level--;ui->volume_timer=VOLUME_HOLD_VBLANKS;audio_apply_state(ui);return ACTION_UI_REFRESH;}action=held_seek_action(now,pressed,*state!=PLAYBACK_RUNNING,ui);if(action!=ACTION_NONE)return action;wait_vblank();tick_ui_timers(ui);if(playback_timer_read()>=deadline)return ACTION_NONE;}}

static void show_help_screen(u16*page,int menu,int multi_media,int can_pause,int can_seek,int audio_controls){volatile u16*d=*page?VRAM0:VRAM1;u32 y=10;clear4(d);set_ui_palette();text4(d,42,2,"CONTROLS",UI_YELLOW);if(can_pause){text4(d,3,y,"A PAUSE RESUME",UI_WHITE);y+=6;}text4(d,3,y,menu?"B RETURN MENU":"B RESTART CLIP",UI_WHITE);y+=6;if(multi_media){text4(d,3,y,"L R PREV NEXT MEDIA",UI_WHITE);y+=6;}if(can_seek){text4(d,3,y,"LEFT RIGHT SEEK STEP",UI_WHITE);y+=6;}if(audio_controls){text4(d,3,y,"UP DOWN VOLUME 0 50 100",UI_WHITE);y+=6;text4(d,3,y,"SELECT MUTE UNMUTE",UI_WHITE);y+=6;}text4(d,3,y,"START CYCLE HUD",UI_WHITE);y+=6;text4(d,3,y,"START+SELECT HELP",UI_WHITE);wait_vblank();*page^=1;REG_DISPCNT=MODE4|(*page?PAGE:0);while(keys_down())wait_vblank();while(!keys_down())wait_vblank();while(keys_down())wait_vblank();}
static int is_menu_mode(const struct GlobalMetadata*m){return m->clip_count>1&&!(m->flags&GLOBAL_FLAG_PLAYLIST);}

static u32 sram_rd(u32 o){return SRAM[o]|((u32)SRAM[o+1]<<8)|((u32)SRAM[o+2]<<16)|((u32)SRAM[o+3]<<24);}
static void sram_wr(u32 o,u32 v){SRAM[o]=(u8)v;SRAM[o+1]=(u8)(v>>8);SRAM[o+2]=(u8)(v>>16);SRAM[o+3]=(u8)(v>>24);}
static void sram_prepare(const struct GlobalMetadata*m){u32 i;if(!(m->flags&GLOBAL_FLAG_RESUME))return;if(sram_rd(0)!=SRAM_MAGIC||sram_rd(4)!=m->clip_count){sram_wr(0,SRAM_MAGIC);sram_wr(4,m->clip_count);sram_wr(8,0);for(i=0;i<m->clip_count&&i<8000;i++)sram_wr(16+i*4,0);}}
static u32 load_menu_selection(const struct GlobalMetadata*m){u32 v;if(!(m->flags&GLOBAL_FLAG_RESUME))return 0;v=sram_rd(8);return v<m->clip_count?v:0;}
static void save_menu_selection(const struct GlobalMetadata*m,u32 v){if(m->flags&GLOBAL_FLAG_RESUME)sram_wr(8,v);}
static void save_position(const struct GlobalMetadata*m,u32 clip,u32 f){if((m->flags&GLOBAL_FLAG_RESUME)&&clip<8000)sram_wr(16+clip*4,(f+1)^SRAM_XOR^clip);}
static void clear_position(const struct GlobalMetadata*m,u32 clip){if((m->flags&GLOBAL_FLAG_RESUME)&&clip<8000)sram_wr(16+clip*4,0);}
static int load_position(const struct GlobalMetadata*m,u32 clip,u32*f){u32 v,d;if(!(m->flags&GLOBAL_FLAG_RESUME)||clip>=8000)return 0;v=sram_rd(16+clip*4);if(!v)return 0;d=v^SRAM_XOR^clip;if(!d)return 0;*f=d-1;return 1;}
static int resume_prompt(u32 seconds){volatile u16*d=VRAM0;char t[6];time5(t,seconds);clear4(d);set_ui_palette();text4(d,22,23,"CONTINUE FROM",UI_YELLOW);text4n(d,38,32,t,5,UI_WHITE);text4(d,28,45,"A CONTINUE",UI_WHITE);text4(d,28,53,"B RESTART",UI_WHITE);REG_DISPCNT=MODE4;while(keys_down())wait_vblank();for(;;){u16 k;wait_vblank();k=keys_down();if(k&KEY_A){while(keys_down())wait_vblank();return 1;}if(k&KEY_B){while(keys_down())wait_vblank();return 0;}}}

static const struct TitleCardHeader *title_card(const struct GlobalMetadata*m){const struct TitleCardHeader*c;if(!m->reserved[1])return 0;c=(const struct TitleCardHeader*)rom_ptr(m->reserved[1]);return c->magic==TITLE_CARD_MAGIC?c:0;}
static void show_title_card(const struct GlobalMetadata*m){const struct TitleCardHeader*c=title_card(m);u32 i,t=0;u16 k;if(!c)return;REG_DISPCNT=FORCE_BLANK;audio_stop();copy16(VRAM0,(const u16*)(c+1),NATIVE_PIXELS);wait_vblank();REG_DISPCNT=MODE3;while(keys_down())wait_vblank();if(c->flags&TITLE_CARD_FLAG_WAIT_A){while(!(keys_down()&KEY_A))wait_vblank();}else while(t<c->duration_vblanks){wait_vblank();t++;k=keys_down();if((c->flags&TITLE_CARD_FLAG_SKIP)&&(k&KEY_A))break;}while(keys_down())wait_vblank();if(c->flags&TITLE_CARD_FLAG_FADE){REG_BLDCNT=0x00FF;for(i=0;i<=16;i++){REG_BLDY=i;wait_vblank();}REG_BLDCNT=REG_BLDY=0;}REG_DISPCNT=FORCE_BLANK;}

static const struct MenuThemeHeader *menu_theme(const struct GlobalMetadata*m){
 const struct MenuThemeHeader*t;if(!m->reserved[0])return 0;t=(const struct MenuThemeHeader*)rom_ptr(m->reserved[0]);
 if(t->magic!=MENU_THEME_MAGIC||t->version!=1u||t->frame_bytes!=FRAME_BYTES||!t->frame_count)return 0;return t;
}
static void set_menu_palette(const struct GlobalMetadata*m){
 active_menu_theme=menu_theme(m);active_menu_outline=0;
 if(active_menu_theme){copy_palette((const u16*)rom_ptr(active_menu_theme->palette_offset));PALRAM[UI_WHITE]=active_menu_theme->ui_colour;PALRAM[UI_YELLOW]=active_menu_theme->selected_colour;PALRAM[UI_DARK]=active_menu_theme->outline_colour;active_menu_outline=(active_menu_theme->flags&MENU_THEME_FLAG_OUTLINE)!=0;}
 else copy_palette(menu_background_palette);
}
#define MENU_SHIMMER_FIRST_COLOUR 17u
#define MENU_SHIMMER_LAST_COLOUR 45u
#define MENU_SHIMMER_LOWER_COPY_BASE 46u
#define MENU_SHIMMER_CREST_COPY_BASE 75u
#define MENU_SHIMMER_PHASES 4u
#define MENU_SHIMMER_LOWER_VBLANKS 12u
#define MENU_SHIMMER_CREST_VBLANKS 30u
static void step_menu_shimmer_range(const u16*palette,u32 source,u32 count,u32 target,u32 phases,u32 phase){u32 i;for(i=0;i<count;i++){u16 c=palette[source+i];if(((source+i+phase)&(phases-1u))==0)c=(u16)(c+0x0420u);PALRAM[target+i]=c;}}
static void step_menu_shimmer(const struct MenuThemeHeader*t,u32 target,u32 phase){if(t)step_menu_shimmer_range((const u16*)rom_ptr(t->palette_offset),t->shimmer_source_start,t->shimmer_count,target,t->shimmer_phases,phase);}
static void step_fallback_lower_shimmer(u32 phase){step_menu_shimmer_range(menu_background_palette,MENU_SHIMMER_FIRST_COLOUR,MENU_SHIMMER_LAST_COLOUR-MENU_SHIMMER_FIRST_COLOUR+1u,MENU_SHIMMER_LOWER_COPY_BASE,MENU_SHIMMER_PHASES,phase);}
static void step_fallback_crest_shimmer(u32 phase){step_menu_shimmer_range(menu_background_palette,MENU_SHIMMER_FIRST_COLOUR,MENU_SHIMMER_LAST_COLOUR-MENU_SHIMMER_FIRST_COLOUR+1u,MENU_SHIMMER_CREST_COPY_BASE,MENU_SHIMMER_PHASES,phase);}
static void draw_menu_char(volatile u16*d,u32 x,u32 y,u8 c,u8 col){u16 bits=glyph_bits(c);u32 r,k;if(active_menu_outline)for(r=0;r<5;r++)for(k=0;k<3;k++)if(bits&(1u<<(14u-(r*3u+k)))){int ox,oy;for(oy=-1;oy<=1;oy++)for(ox=-1;ox<=1;ox++){int px=(int)x+(int)k+ox,py=(int)y+(int)r+oy;if(px>=0&&py>=0&&px<120&&py<80)p4(d,(u32)px,(u32)py,UI_DARK);}}for(r=0;r<5;r++)for(k=0;k<3;k++)if(bits&(1u<<(14u-(r*3u+k))))p4(d,x+k,y+r,col);}
static void draw_menu_text_n(volatile u16*d,u32 x,u32 y,const char*t,u32 n,u8 col){u32 i=0;while(i<n&&t[i]){draw_menu_char(d,x+i*4u,y,(u8)t[i],col);i++;}}
static void draw_menu_text(volatile u16*d,u32 x,u32 y,const char*t,u8 col){u32 n=0;while(t[n])n++;draw_menu_text_n(d,x,y,t,n,col);}
static void menu_arrow_tile_pixel(u32 x,u32 y,u8 color){u32 tile=udiv(y,8u)*2u+udiv(x,8u),ix=x&7u,iy=y&7u,bo=tile*32u+iy*4u+udiv(ix,2u),hi=bo>>1,shift=(bo&1u)*8u;u16 hw=OBJ_TILE_VRAM[hi];u8 v=(u8)(hw>>shift);if(ix&1u)v=(u8)((v&0x0Fu)|(u8)(color<<4));else v=(u8)((v&0xF0u)|color);hw=(u16)((hw&(u16)~(0x00FFu<<shift))|((u16)v<<shift));OBJ_TILE_VRAM[hi]=hw;}
static void menu_arrow_init(void){static const u8 widths[5]={2u,3u,4u,3u,2u};u32 i,r,x,sy;for(i=0;i<128u;i++){OAM[i*4u]=OBJ_DISABLE;OAM[i*4u+1u]=0;OAM[i*4u+2u]=0;OAM[i*4u+3u]=0;}for(i=0;i<64u;i++)OBJ_TILE_VRAM[i]=0;if(active_menu_outline)for(r=0;r<5;r++){sy=3u+r*2u;for(x=0;x<widths[r]*2u;x++){int ox,oy;for(oy=-1;oy<=1;oy++)for(ox=-1;ox<=1;ox++){int px=(int)x+ox,py=(int)sy+oy;if(px>=0&&py>=0&&px<16&&py<16){menu_arrow_tile_pixel((u32)px,(u32)py,2u);if(py+1<16)menu_arrow_tile_pixel((u32)px,(u32)(py+1),2u);}}}}for(r=0;r<5;r++){sy=3u+r*2u;for(x=0;x<widths[r]*2u;x++){menu_arrow_tile_pixel(x,sy,1u);menu_arrow_tile_pixel(x,sy+1u,1u);}}OBJ_PALRAM[0]=0;OBJ_PALRAM[1]=active_menu_theme?active_menu_theme->selected_colour:0x037Fu;OBJ_PALRAM[2]=active_menu_theme?active_menu_theme->outline_colour:0;OAM[0]=OBJ_DISABLE;OAM[1]=OBJ_SIZE_16;OAM[2]=MENU_ARROW_TILE_INDEX;}
static void menu_arrow_set(u32 x,u32 y,int visible){OAM[MENU_ARROW_OAM_INDEX*4u]=(u16)((y&0xFFu)|(visible?0u:OBJ_DISABLE));OAM[MENU_ARROW_OAM_INDEX*4u+1u]=(u16)((x&0x1FFu)|OBJ_SIZE_16);OAM[MENU_ARROW_OAM_INDEX*4u+2u]=MENU_ARROW_TILE_INDEX;}
static void menu_arrow_hide(void){OAM[MENU_ARROW_OAM_INDEX*4u]|=OBJ_DISABLE;}
static void draw_menu_background(volatile u16*d,u32 frame){const u8*p=menu_background_pixels;if(active_menu_theme){if(frame>=active_menu_theme->frame_count)frame=0;p=rom_ptr(active_menu_theme->frames_offset+frame*FRAME_BYTES);}render_pixels(p,d);if(active_menu_outline){rect4(d,0,13,120,1,UI_DARK);rect4(d,0,15,120,1,UI_DARK);}rect4(d,0,14,120,1,UI_WHITE);}
static u32 fixed_text_length(const char*t,u32 max){u32 n=0;while(n<max&&t[n])n++;return n;}
static u32 append_decimal(char*out,u32 pos,u32 v){char tmp[5];u32 n=0,i;if(v>999)v=999;do{tmp[n++]=(char)('0'+v%10u);v=udiv(v,10u);}while(v&&n<4u);for(i=0;i<n;i++)out[pos++]=tmp[n-i-1u];return pos;}
static void make_duration_text(char out[6],u32 sec){u32 min=udiv(sec,60u),rem=sec%60u;if(min>99)min=99;out[0]=(char)('0'+udiv(min,10u));out[1]=(char)('0'+min%10u);out[2]=':';out[3]=(char)('0'+udiv(rem,10u));out[4]=(char)('0'+rem%10u);out[5]=0;}
static void make_clip_position_text(char out[16],u32 count,u32 selected){u32 pos=0,i;const char pfx[]="CLIP ";for(i=0;i<5;i++)out[pos++]=pfx[i];pos=append_decimal(out,pos,selected+1u);out[pos++]='/';pos=append_decimal(out,pos,count);out[pos]=0;}
static void make_total_text(char out[16],u32 sec){u32 pos=0,i;char d[6];const char pfx[]="TOTAL ";make_duration_text(d,sec);for(i=0;i<6;i++)out[pos++]=pfx[i];for(i=0;i<5;i++)out[pos++]=d[i];out[pos]=0;}
static u32 menu_total_seconds(const struct GlobalMetadata*m,const struct ClipDescriptor*c){u32 i,t=0;for(i=0;i<m->clip_count;i++){if(c[i].flags&CLIP_FLAG_MEDIA_IMAGE)t+=udiv(c[i].audio_sample_count,1000u);else t+=seconds_for_frame(c[i].frame_count,c[i].vblanks_per_frame);}return t;}
static u32 menu_column_count(u32 count){if(count<=MENU_ROWS)return 1u;if(count<=MENU_ROWS*2u)return 2u;return 3u;}
static void draw_clip_menu(volatile u16*d,const struct GlobalMetadata*m,const struct ClipDescriptor*c,u32 selected,u32 total,u32 bg){char pos[16],tot[16];u32 cols=menu_column_count(m->clip_count),page_size=cols*MENU_ROWS,page_start=udiv(selected,page_size)*page_size,colw=udiv(120u,cols),maxchars=cols>=3u?8u:12u,col,row;draw_menu_background(d,bg);draw_menu_text(d,36,2,"SELECT MEDIA",UI_WHITE);make_clip_position_text(pos,m->clip_count,selected);make_total_text(tot,total);draw_menu_text(d,2,8,pos,UI_WHITE);draw_menu_text(d,74,8,tot,UI_WHITE);for(col=0;col<cols;col++){u32 x=col*colw;for(row=0;row<MENU_ROWS;row++){u32 idx=page_start+col*MENU_ROWS+row,y=17u+row*6u;if(idx>=m->clip_count)break;draw_menu_text_n(d,x+8u,y,c[idx].title,fixed_text_length(c[idx].title,maxchars),idx==selected?UI_YELLOW:UI_WHITE);}}}
static void menu_arrow_position(u32 selected,u32 count,u32*x,u32*y){u32 cols=menu_column_count(count),page_size=cols*MENU_ROWS,page_start=udiv(selected,page_size)*page_size,rel=selected-page_start,col=udiv(rel,MENU_ROWS),row=rel-col*MENU_ROWS,colw=udiv(120u,cols),textx=(col*colw+8u)*2u;*x=textx-10u;*y=(17u+row*6u)*2u-3u;}
static u32 menu_move_up(u32 selected,u32 count){u32 cols=menu_column_count(count),ps=cols*MENU_ROWS,start=udiv(selected,ps)*ps,col=udiv(selected-start,MENU_ROWS),first=start+col*MENU_ROWS,last=first+MENU_ROWS-1u;if(last>=count)last=count-1u;return selected>first?selected-1u:last;}
static u32 menu_move_down(u32 selected,u32 count){u32 cols=menu_column_count(count),ps=cols*MENU_ROWS,start=udiv(selected,ps)*ps,col=udiv(selected-start,MENU_ROWS),first=start+col*MENU_ROWS,last=first+MENU_ROWS-1u;if(last>=count)last=count-1u;return selected<last?selected+1u:first;}
static void show_menu_help_screen(u16*page){volatile u16*d=*page?VRAM0:VRAM1;clear4(d);set_ui_palette();text4(d,38,4,"MENU CONTROLS",UI_YELLOW);text4(d,3,18,"UP DOWN WITHIN COLUMN",UI_WHITE);text4(d,3,28,"LEFT RIGHT COLUMNS",UI_WHITE);text4(d,3,38,"A PLAY SELECTED MEDIA",UI_WHITE);text4(d,3,54,"START+SELECT HELP",UI_DARK);wait_vblank();*page^=1;REG_DISPCNT=MODE4|(*page?PAGE:0);while(keys_down())wait_vblank();while(!keys_down())wait_vblank();while(keys_down())wait_vblank();}
static u32 select_clip_menu(const struct GlobalMetadata*m,const struct ClipDescriptor*c,u32 initial){u32 selected=initial<m->clip_count?initial:0u,total=menu_total_seconds(m,c),blink=0,lowerc=0,lowerp=0,crestc=0,crestp=0,animc=0,bg=0,ax=0,ay=0;int anim_ready=0,arrow=1,help_latched=0;u16 page=1,prev=keys_down();if(m->clip_count<=1)return 0;REG_DISPCNT=FORCE_BLANK;set_menu_palette(m);menu_arrow_init();for(;;){volatile u16*d=page?VRAM0:VRAM1;draw_clip_menu(d,m,c,selected,total,bg);menu_arrow_position(selected,m->clip_count,&ax,&ay);menu_arrow_set(ax,ay,1);arrow=1;blink=0;wait_vblank();page^=1;REG_DISPCNT=(u16)(MODE4_OBJ|(page?PAGE:0));anim_ready=0;for(;;){u16 now,p;wait_vblank();if(anim_ready){page^=1;REG_DISPCNT=(u16)(MODE4_OBJ|(page?PAGE:0));anim_ready=0;}if(active_menu_theme&&active_menu_theme->kind==MENU_THEME_SHIMMER&&active_menu_theme->shimmer_phases){if(++lowerc>=active_menu_theme->shimmer_interval1){lowerc=0;lowerp=(lowerp+1u)&(active_menu_theme->shimmer_phases-1u);step_menu_shimmer(active_menu_theme,active_menu_theme->shimmer_target1,lowerp);}if(++crestc>=active_menu_theme->shimmer_interval2){crestc=0;crestp=(crestp+1u)&(active_menu_theme->shimmer_phases-1u);step_menu_shimmer(active_menu_theme,active_menu_theme->shimmer_target2,crestp);}}else if(!active_menu_theme){if(++lowerc>=MENU_SHIMMER_LOWER_VBLANKS){lowerc=0;lowerp=(lowerp+1u)&(MENU_SHIMMER_PHASES-1u);step_fallback_lower_shimmer(lowerp);}if(++crestc>=MENU_SHIMMER_CREST_VBLANKS){crestc=0;crestp=(crestp+1u)&(MENU_SHIMMER_PHASES-1u);step_fallback_crest_shimmer(crestp);}}else if(active_menu_theme->kind==MENU_THEME_FRAMES&&active_menu_theme->frame_count>1u&&active_menu_theme->frame_vblanks){if(++animc>=active_menu_theme->frame_vblanks){volatile u16*back;animc=0;bg++;if(bg>=active_menu_theme->frame_count)bg=0;back=page?VRAM0:VRAM1;draw_clip_menu(back,m,c,selected,total,bg);anim_ready=1;}}now=keys_down();p=now&~prev;prev=now;if(++blink>=MENU_ARROW_BLINK_VBLANKS){blink=0;arrow=!arrow;menu_arrow_set(ax,ay,arrow);}if((now&(KEY_START|KEY_SELECT))==(KEY_START|KEY_SELECT)){if(!help_latched){help_latched=1;menu_arrow_hide();show_menu_help_screen(&page);set_menu_palette(m);menu_arrow_init();prev=keys_down();break;}}else help_latched=0;if(p&KEY_UP){menu_arrow_hide();selected=menu_move_up(selected,m->clip_count);save_menu_selection(m,selected);break;}if(p&KEY_DOWN){menu_arrow_hide();selected=menu_move_down(selected,m->clip_count);save_menu_selection(m,selected);break;}if(p&KEY_LEFT){u32 next=selected>=MENU_ROWS?selected-MENU_ROWS:m->clip_count-1u;menu_arrow_hide();selected=next;save_menu_selection(m,selected);break;}if(p&KEY_RIGHT){u32 next=selected+MENU_ROWS<m->clip_count?selected+MENU_ROWS:0u;menu_arrow_hide();selected=next;save_menu_selection(m,selected);break;}if(p&KEY_A){menu_arrow_hide();save_menu_selection(m,selected);while(keys_down())wait_vblank();return selected;}}}}

static const u8*media_metadata(const struct ClipDescriptor*c){const u8*m;if(!(c->flags&CLIP_FLAG_MEDIA_META)||!c->video_index_offset)return 0;m=rom_ptr(c->video_index_offset);return (rd32(m)==MEDIA_META_MAGIC_V1||rd32(m)==MEDIA_META_MAGIC_V2)?m:0;}
static const char*media_title(const struct ClipDescriptor*c){const u8*m=media_metadata(c);if(m&&rd32(m)==MEDIA_META_MAGIC_V2)return (const char*)(m+4);return c->title;}
static u32 media_title_limit(const struct ClipDescriptor*c){const u8*m=media_metadata(c);return (m&&rd32(m)==MEDIA_META_MAGIC_V2)?28u:12u;}
static const char*media_artist(const struct ClipDescriptor*c){const u8*m=media_metadata(c);if(!m)return "";return (const char*)(m+(rd32(m)==MEDIA_META_MAGIC_V2?32:4));}
static u32 media_artist_limit(const struct ClipDescriptor*c){const u8*m=media_metadata(c);if(!m)return 0u;return rd32(m)==MEDIA_META_MAGIC_V2?28u:20u;}
static int tick_ui_timers(struct PlayerUI*ui){int changed=0;if(ui->hud_timer&&--ui->hud_timer==0)changed|=1;if(ui->mute_timer&&--ui->mute_timer==0)changed|=2;if(ui->volume_timer&&--ui->volume_timer==0)changed|=2;if(ui->seek_timer&&--ui->seek_timer==0){ui->seek_direction=0;changed|=2;}return changed;}
static void bitmap3(volatile u16*d,u32 x,u32 y,const u8*rows,u32 h,int flip){u32 r,k;for(r=0;r<h;r++)for(k=0;k<7;k++)if(rows[r]&(1u<<k))block3(d,x+(flip?6u-k:k)*2u,y+r*2u,0x03FF);}
static void seek_badge3(volatile u16*d,int dir,u32 sec){char n[2];u32 digits=sec>=10u?2u:1u,nw=digits==2u?7u:3u,bw=7u+2u+nw+4u,bx=(120u-bw)/2u,cx=bx+2u;if(digits==2u){n[0]=(char)('0'+udiv(sec,10u)%10u);n[1]=(char)('0'+sec%10u);}else n[0]=(char)('0'+sec%10u);rect3(d,bx*2u,64,bw*2u,20,0);if(dir<0){bitmap3(d,cx*2u,66,seek_arrow_rows,7,0);text3n(d,(cx+9u)*2u,68,n,digits,0x7FFF);}else{text3n(d,cx*2u,68,n,digits,0x7FFF);bitmap3(d,(cx+nw+2u)*2u,66,seek_arrow_rows,7,1);}}
static void native_audio_clock(const struct ClipDescriptor*c,u32 frame,const struct PlayerUI*ui){char cur[6];u32 mode=ui->hud_mode,w;if((ui->hud_timer||ui->paused_ui)&&mode<2)mode=2;if(!mode)return;time5(cur,seconds_for_frame(frame,c->vblanks_per_frame));rect3(VRAM0,8,144,38,10,0);text3n(VRAM0,8,144,cur,5,0x7FFF);if(mode==2){rect3(VRAM0,8,156,224,4,0x2108);w=c->frame_count>1?udiv(frame*224u,c->frame_count-1u):224u;rect3(VRAM0,8,156,w,4,0x03FF);}}
static void native_draw_audio_badges(const struct ClipDescriptor*c,const struct PlayerUI*ui){if(ui->mute_timer)mute_badge3(VRAM0,ui->muted);if(ui->volume_timer)volume_badge3(VRAM0,ui->volume_level);if(ui->seek_timer&&ui->seek_direction)seek_badge3(VRAM0,ui->seek_direction,c->seek_seconds?c->seek_seconds:5u);}
static void native_overlay(const struct ClipDescriptor*c,u32 frame,int paused,int image,int show,const struct PlayerUI*ui){char tot[6];u32 total;int mode=show?ui->hud_mode:0;if((ui->hud_timer||ui->paused_ui)&&!image&&mode<2)mode=2;if(mode<=0){if(!image)native_draw_audio_badges(c,ui);return;}if(image){rect3(VRAM0,0,132,240,28,0);text3n(VRAM0,8,138,c->title,12,0x7FFF);text3n(VRAM0,184,138,"IMAGE",5,0x03FF);return;}total=seconds_for_frame(c->frame_count?c->frame_count-1:0,c->vblanks_per_frame);time5(tot,total);if(mode==2){dim3(VRAM0,104,56);text3n(VRAM0,8,108,media_title(c),media_title_limit(c),0x7FFF);text3n(VRAM0,8,122,media_artist(c),media_artist_limit(c),0x5294);text3n(VRAM0,184,144,paused?"PAUSE":"PLAY",5,paused?0x03FF:0x03E0);}else dim3(VRAM0,140,20);native_audio_clock(c,frame,ui);text3n(VRAM0,52,144,"/",1,0x4210);text3n(VRAM0,64,144,tot,5,0x7FFF);native_draw_audio_badges(c,ui);}

static void native_restore_audio_badges(const struct ClipDescriptor*c){const u16*src=(const u16*)rom_ptr(c->video_offset);u32 y;for(y=0;y<18;y++)copy16(VRAM0+(4u+y)*240u+198u,src+(4u+y)*240u+198u,42u);for(y=0;y<24;y++)copy16(VRAM0+(62u+y)*240u+96u,src+(62u+y)*240u+96u,48u);}
static void native_refresh_audio_badges(const struct ClipDescriptor*c,const struct PlayerUI*ui){native_restore_audio_badges(c);native_draw_audio_badges(c,ui);}
static void native_refresh_audio_ui(const struct ClipDescriptor*c,u32 frame,int paused,int show,const struct PlayerUI*ui){const u16*src=(const u16*)rom_ptr(c->video_offset);copy16(VRAM0+104u*240u,src+104u*240u,56u*240u);native_restore_audio_badges(c);native_overlay(c,frame,paused,0,show,ui);}
static void native_refresh_image_ui(const struct ClipDescriptor*c,int show,const struct PlayerUI*ui){const u16*src=(const u16*)rom_ptr(c->video_offset);copy16(VRAM0+132u*240u,src+132u*240u,28u*240u);if(show)native_overlay(c,0,0,1,1,ui);}
static void show_native_art(const struct ClipDescriptor*c,u32 frame,int paused,int image,int show,const struct PlayerUI*ui){REG_DISPCNT=FORCE_BLANK;copy16(VRAM0,(const u16*)rom_ptr(c->video_offset),NATIVE_PIXELS);native_overlay(c,frame,paused,image,show,ui);wait_vblank();REG_DISPCNT=MODE3;}

static int play_image(const struct GlobalMetadata*m,const struct ClipDescriptor*c,u32 idx,struct PlayerUI*ui){
 u16 prev=keys_down(),help_page=1;u32 ticks=0,limit=0;int paused=0;if(c->audio_sample_count)limit=udiv(c->audio_sample_count*60u,1000u);show_native_art(c,0,0,1,ui->hud_mode>0,ui);while(keys_down())wait_vblank();
 for(;;){u16 now,p;int action,oldmode;wait_vblank();now=keys_down();p=now&~prev;prev=now;oldmode=ui->hud_mode;action=common_combo_action(now,p,m->clip_count>1,0,HUD_HOLD_VBLANKS,ui);if(action==ACTION_HELP){show_help_screen(&help_page,is_menu_mode(m),m->clip_count>1,limit>0,0,0);show_native_art(c,0,paused,1,ui->hud_mode>0,ui);continue;}if(action==ACTION_UI_REFRESH){ui->hud_mode=oldmode?0:1;ui->hud_last_visible=1;native_refresh_image_ui(c,ui->hud_mode>0,ui);continue;}if(action==ACTION_PREV_CLIP)return PLAY_RESULT_PREV_CLIP;if(action==ACTION_NEXT_CLIP)return PLAY_RESULT_NEXT_CLIP_DIRECT;if(p&KEY_B)return is_menu_mode(m)?PLAY_RESULT_RETURN_MENU:PLAY_RESULT_RESTART_CURRENT;if(limit&&(p&KEY_A))paused=!paused;if(!paused)ticks++;if(limit&&ticks>=limit)return PLAY_RESULT_NEXT_CLIP;save_position(m,idx,0);}
}
static int play_audio(const struct GlobalMetadata*m,const struct ClipDescriptor*c,u32 idx,struct PlayerUI*ui){
 u32 f=0,step=c->seek_frame_step?c->seek_frame_step:1,base_sample,paused_sample=0,lastsec;u16 prev=keys_down(),help_page=1;int paused=0;
 if(load_position(m,idx,&f)&&f>=c->frame_count)f=0;base_sample=seek_value(c,f);show_native_art(c,f,paused,0,ui->hud_mode>0,ui);lastsec=seconds_for_frame(f,c->vblanks_per_frame);playback_timer_reset();audio_start_for_frame(c,f,0,ui);while(keys_down())wait_vblank();
 for(;;){u16 now,p;u32 elapsed,abs,target,sec;int ui_changed,action,oldmode;wait_vblank();audio_service();ui_changed=tick_ui_timers(ui);now=keys_down();p=now&~prev;prev=now;elapsed=playback_timer_read();abs=base_sample+elapsed;if(ui_changed){if((ui_changed&1)&&ui->hud_mode<2)native_refresh_audio_ui(c,f,paused,ui->hud_mode>0,ui);else native_refresh_audio_badges(c,ui);}
  if(!paused){while(f+1<c->frame_count&&seek_value(c,f+1)<=abs)f++;if(abs>=c->audio_sample_count){audio_stop();playback_timer_stop();clear_position(m,idx);return (c->flags&CLIP_FLAG_LOOP)?PLAY_RESULT_RESTART_CURRENT:PLAY_RESULT_NEXT_CLIP;}}sec=seconds_for_frame(f,c->vblanks_per_frame);if(sec!=lastsec){native_audio_clock(c,f,ui);lastsec=sec;}
  oldmode=ui->hud_mode;action=common_combo_action(now,p,m->clip_count>1,1,AUDIO_HUD_HOLD_VBLANKS,ui);
  if(action==ACTION_HELP){u32 resume_sample=paused?paused_sample:abs;if(!paused){playback_timer_pause();audio_pause();}show_help_screen(&help_page,is_menu_mode(m),m->clip_count>1,1,1,1);if(!paused){base_sample=resume_sample;audio_start_at(c,base_sample,0,ui);playback_timer_reset();}show_native_art(c,f,paused,0,ui->hud_mode>0,ui);lastsec=seconds_for_frame(f,c->vblanks_per_frame);continue;}
  if(action==ACTION_UI_REFRESH){if(ui->hud_mode!=oldmode)native_refresh_audio_ui(c,f,paused,ui->hud_mode>0,ui);else native_refresh_audio_badges(c,ui);continue;}
  if(action==ACTION_PREV_CLIP){audio_stop();playback_timer_stop();save_position(m,idx,f);return PLAY_RESULT_PREV_CLIP;}
  if(action==ACTION_NEXT_CLIP){audio_stop();playback_timer_stop();save_position(m,idx,f);return PLAY_RESULT_NEXT_CLIP_DIRECT;}
  if(p&KEY_A){paused=!paused;ui->paused_ui=paused;if(paused){paused_sample=abs;playback_timer_pause();audio_pause();}else{base_sample=paused_sample;audio_start_at(c,base_sample,0,ui);playback_timer_reset();}native_refresh_audio_ui(c,f,paused,ui->hud_mode>0,ui);}
  if(p&KEY_B){audio_stop();playback_timer_stop();if(is_menu_mode(m)){save_position(m,idx,f);return PLAY_RESULT_RETURN_MENU;}clear_position(m,idx);return PLAY_RESULT_RESTART_CURRENT;}
  action=held_seek_action(now,p,0,ui);if(action==ACTION_SEEK_BACK||action==ACTION_SEEK_FORWARD){u32 amount=step;int forward=(action==ACTION_FRAME_FORWARD||action==ACTION_SEEK_FORWARD);target=seek_target(f,c->frame_count,amount,forward);if(target!=f){if(action==ACTION_SEEK_BACK||action==ACTION_SEEK_FORWARD)start_audio_seek_feedback(ui,forward?1:-1);f=target;base_sample=seek_value(c,f);paused_sample=base_sample;audio_start_at(c,base_sample,paused,ui);playback_timer_reset();if(paused)playback_timer_pause();if(ui->hud_mode<2)native_refresh_audio_ui(c,f,paused,ui->hud_mode>0,ui);else{native_audio_clock(c,f,ui);native_refresh_audio_badges(c,ui);}lastsec=seconds_for_frame(f,c->vblanks_per_frame);save_position(m,idx,f);}continue;}
  if(p&KEY_UP){if(ui->volume_level<2)ui->volume_level++;ui->volume_timer=AUDIO_VOLUME_HOLD_VBLANKS;audio_apply_state(ui);native_refresh_audio_badges(c,ui);}
  if(p&KEY_DOWN){if(ui->volume_level>0)ui->volume_level--;ui->volume_timer=AUDIO_VOLUME_HOLD_VBLANKS;audio_apply_state(ui);native_refresh_audio_badges(c,ui);}
 }
}

static int play_video(const struct GlobalMetadata*meta,const struct ClipDescriptor*clip,u32 clip_index,struct PlayerUI*ui){
 u32 frame=0;u8*current=frame_a,*next=frame_b;u16 displayed_page=0,previous_keys=keys_down();enum PlaybackState state=PLAYBACK_RUNNING;struct PlaybackClock clock;int has_audio=(clip->flags&CLIP_FLAG_AUDIO)!=0;int next_frame_valid=0;int at_end=0;
 if(load_position(meta,clip_index,&frame)&&frame>=clip->frame_count)frame=0;load_frame_pixels(clip,frame,current);REG_DISPCNT=FORCE_BLANK;set_ui_palette();render_and_show(current,frame,&displayed_page,clip,ui);playback_clock_init(&clock,clip->vblanks_per_frame);playback_timer_reset();playback_clock_advance(&clock);if(has_audio)audio_start_for_frame(clip,frame,0,ui);ui->pause_button_latched=(previous_keys&KEY_A)!=0u;
 for(;;){
  if(state!=PLAYBACK_RUNNING){
   int action=wait_frame_period(&previous_keys,0xFFFFFFFFu,has_audio,meta->clip_count>1,&state,ui);
   if(action==ACTION_RESTART){playback_timer_stop();audio_stop();if(is_menu_mode(meta)){save_position(meta,clip_index,frame);return PLAY_RESULT_RETURN_MENU;}clear_position(meta,clip_index);return PLAY_RESULT_RESTART_CURRENT;}
   if(action==ACTION_PREV_CLIP){playback_timer_stop();audio_stop();save_position(meta,clip_index,frame);return PLAY_RESULT_PREV_CLIP;}
   if(action==ACTION_NEXT_CLIP){playback_timer_stop();audio_stop();save_position(meta,clip_index,frame);return PLAY_RESULT_NEXT_CLIP_DIRECT;}
   if(action==ACTION_HELP){show_help_screen(&displayed_page,is_menu_mode(meta),meta->clip_count>1,1,1,has_audio);render_and_show(current,frame,&displayed_page,clip,ui);next_frame_valid=0;previous_keys=keys_down();continue;}
   if(action==ACTION_FRAME_BACK||action==ACTION_FRAME_FORWARD||action==ACTION_SEEK_BACK||action==ACTION_SEEK_FORWARD){u32 target=(action==ACTION_FRAME_BACK)?(frame?frame-1:0):(action==ACTION_FRAME_FORWARD)?(frame+1<clip->frame_count?frame+1:frame):seek_target(frame,clip->frame_count,clip->seek_frame_step,action==ACTION_SEEK_FORWARD);if(target!=frame){if(action==ACTION_SEEK_BACK||action==ACTION_SEEK_FORWARD)start_seek_feedback(ui,action==ACTION_SEEK_FORWARD?1:-1);ui->paused_ui=1;load_frame_pixels(clip,target,current);frame=target;render_and_show(current,frame,&displayed_page,clip,ui);next_frame_valid=0;state=PLAYBACK_PAUSED;save_position(meta,clip_index,frame);}continue;}
   if(action==ACTION_RESUME_PENDING){int advance=frame+1<clip->frame_count&&next_frame_valid;u32 resume_frame=advance?frame+1:frame;if(state!=PLAYBACK_RESUME_ARMED)continue;playback_clock_init(&clock,clip->vblanks_per_frame);playback_clock_advance(&clock);if(has_audio)audio_start_for_frame(clip,resume_frame,1,ui);wait_vblank();if(advance){show_rendered_page(&displayed_page,palette_for_frame(clip,resume_frame));{u8*tmp=current;current=next;next=tmp;}frame=resume_frame;next_frame_valid=0;}playback_timer_reset();if(has_audio)audio_resume();ui->paused_ui=0;state=PLAYBACK_RUNNING;previous_keys=keys_down();continue;}
   if(action==ACTION_UI_REFRESH){render_and_show(current,frame,&displayed_page,clip,ui);next_frame_valid=0;}
   continue;
  }
  {
   int has_next=frame+1<clip->frame_count;volatile u16*back=displayed_page?VRAM0:VRAM1;int action;
   if(has_next&&!next_frame_valid){load_next_pixels(clip,frame+1,current,next);render_frame_with_ui(next,frame+1,back,clip,ui);next_frame_valid=1;}
   action=wait_frame_period(&previous_keys,clock.next_deadline,has_audio,meta->clip_count>1,&state,ui);
   if(action==ACTION_RESTART){playback_timer_stop();audio_stop();if(is_menu_mode(meta)){save_position(meta,clip_index,frame);return PLAY_RESULT_RETURN_MENU;}clear_position(meta,clip_index);return PLAY_RESULT_RESTART_CURRENT;}
   if(action==ACTION_PREV_CLIP){playback_timer_stop();audio_stop();save_position(meta,clip_index,frame);return PLAY_RESULT_PREV_CLIP;}
   if(action==ACTION_NEXT_CLIP){playback_timer_stop();audio_stop();save_position(meta,clip_index,frame);return PLAY_RESULT_NEXT_CLIP_DIRECT;}
   if(action==ACTION_HELP){playback_timer_pause();if(has_audio)audio_pause();show_help_screen(&displayed_page,is_menu_mode(meta),meta->clip_count>1,1,1,has_audio);render_and_show(current,frame,&displayed_page,clip,ui);next_frame_valid=0;playback_clock_init(&clock,clip->vblanks_per_frame);playback_timer_reset();playback_clock_advance(&clock);if(has_audio)audio_start_for_frame(clip,frame,0,ui);previous_keys=keys_down();continue;}
   if(action==ACTION_RESUME_PENDING)continue;
   if(action==ACTION_UI_REFRESH){render_and_show(current,frame,&displayed_page,clip,ui);next_frame_valid=0;continue;}
   if(action==ACTION_SEEK_BACK||action==ACTION_SEEK_FORWARD){u32 target=seek_target(frame,clip->frame_count,clip->seek_frame_step,action==ACTION_SEEK_FORWARD);start_seek_feedback(ui,action==ACTION_SEEK_FORWARD?1:-1);load_frame_pixels(clip,target,current);frame=target;render_and_show(current,frame,&displayed_page,clip,ui);next_frame_valid=0;playback_clock_init(&clock,clip->vblanks_per_frame);playback_timer_reset();playback_clock_advance(&clock);if(has_audio)audio_start_for_frame(clip,frame,0,ui);save_position(meta,clip_index,frame);continue;}
   if(state!=PLAYBACK_RUNNING)continue;
   if(has_next){show_rendered_page(&displayed_page,palette_for_frame(clip,frame+1));{u8*tmp=current;current=next;next=tmp;}++frame;next_frame_valid=0;playback_clock_advance(&clock);if((frame&31)==0)save_position(meta,clip_index,frame);}
   else{if(at_end||!(clip->flags&CLIP_FLAG_LOOP)){playback_timer_stop();audio_stop();clear_position(meta,clip_index);return (clip->flags&CLIP_FLAG_LOOP)?PLAY_RESULT_RESTART_CURRENT:PLAY_RESULT_NEXT_CLIP;}at_end=1;}
  }
 }
}

static int play_media(const struct GlobalMetadata*m,const struct ClipDescriptor*c,u32 idx,struct PlayerUI*ui){ui->paused_ui=0;ui->start_pending=0;ui->select_pending=0;if(c->flags&CLIP_FLAG_MEDIA_IMAGE){ui->hud_mode=0;ui->hud_last_visible=1;ui->hud_timer=ui->mute_timer=ui->volume_timer=ui->seek_timer=0;ui->seek_direction=ui->seek_hold_direction=0;ui->seek_hold_counter=0;return play_image(m,c,idx,ui);}if(c->flags&CLIP_FLAG_MEDIA_AUDIO){ui->hud_mode=2;ui->hud_last_visible=2;return play_audio(m,c,idx,ui);}ui->hud_mode=0;ui->hud_last_visible=2;if(!(c->flags&CLIP_FLAG_AUDIO))ui->mute_timer=ui->volume_timer=0;return play_video(m,c,idx,ui);}

void main(void){const struct GlobalMetadata*m=&gba_video_metadata;const struct ClipDescriptor*clips;u32 selected=0;int menu_pending;struct PlayerUI ui={0};REG_WAITCNT=0x4317;ui.volume_level=2;ui.hud_mode=0;ui.hud_last_visible=2;if(m->magic!=GBV5_MAGIC||m->version!=5||!m->clip_count)for(;;){}clips=(const struct ClipDescriptor*)rom_ptr(m->clip_table_offset);sram_prepare(m);show_title_card(m);selected=load_menu_selection(m);menu_pending=is_menu_mode(m);for(;;){int r;u32 saved=0;if(is_menu_mode(m)&&menu_pending){selected=select_clip_menu(m,clips,selected);menu_pending=0;}save_menu_selection(m,selected);if(!(clips[selected].flags&CLIP_FLAG_MEDIA_IMAGE)&&load_position(m,selected,&saved)){if(saved>0&&saved+1<clips[selected].frame_count){if(!resume_prompt(seconds_for_frame(saved,clips[selected].vblanks_per_frame)))clear_position(m,selected);}else if(saved+1>=clips[selected].frame_count)clear_position(m,selected);}r=play_media(m,&clips[selected],selected,&ui);if(r==PLAY_RESULT_RETURN_MENU){menu_pending=1;continue;}if(r==PLAY_RESULT_PREV_CLIP){selected=selected?selected-1:m->clip_count-1;menu_pending=0;continue;}if(r==PLAY_RESULT_NEXT_CLIP_DIRECT){selected=(selected+1)%m->clip_count;menu_pending=0;continue;}if(r==PLAY_RESULT_NEXT_CLIP){if(is_menu_mode(m)){menu_pending=1;continue;}selected=(selected+1)%m->clip_count;if(m->clip_count==1&&!(clips[0].flags&CLIP_FLAG_LOOP))selected=0;continue;}if(r==PLAY_RESULT_RESTART_CURRENT){menu_pending=0;continue;}}}

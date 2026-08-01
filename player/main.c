typedef unsigned char  u8;
typedef unsigned short u16;
typedef unsigned int   u32;

#define REG16(addr) (*(volatile u16 *)(addr))
#define REG32(addr) (*(volatile u32 *)(addr))

#define REG_DISPCNT      REG16(0x04000000)
#define REG_VCOUNT       REG16(0x04000006)
#define REG_BG2PA        REG16(0x04000020)
#define REG_BG2PB        REG16(0x04000022)
#define REG_BG2PC        REG16(0x04000024)
#define REG_BG2PD        REG16(0x04000026)
#define REG_BG2X         REG32(0x04000028)
#define REG_BG2Y         REG32(0x0400002C)
#define REG_WAITCNT      REG16(0x04000204)
#define REG_IME          REG16(0x04000208)

#define REG_SOUNDCNT_L   REG16(0x04000080)
#define REG_SOUNDCNT_H   REG16(0x04000082)
#define REG_SOUNDCNT_X   REG16(0x04000084)
#define REG_SOUNDBIAS    REG16(0x04000088)
#define REG_FIFO_A       REG32(0x040000A0)
#define REG_DMA1SAD      REG32(0x040000BC)
#define REG_DMA1DAD      REG32(0x040000C0)
#define REG_DMA1CNT_L    REG16(0x040000C4)
#define REG_DMA1CNT_H    REG16(0x040000C6)
#define REG_TM0CNT_L     REG16(0x04000100)
#define REG_TM0CNT_H     REG16(0x04000102)
#define REG_TM2CNT_L     REG16(0x04000108)
#define REG_TM2CNT_H     REG16(0x0400010A)
#define REG_TM3CNT_L     REG16(0x0400010C)
#define REG_TM3CNT_H     REG16(0x0400010E)
#define REG_KEYINPUT     REG16(0x04000130)

#define PALRAM           ((volatile u16 *)0x05000000)
#define VRAM_PAGE0       ((volatile u16 *)0x06000000)
#define VRAM_PAGE1       ((volatile u16 *)0x0600A000)
#define SRAM_BASE        ((volatile u8 *)0x0E000000)
#define ROM_BASE         0x08000000u

#define MODE4_BG2        0x0404
#define PAGE_SELECT      0x0010
#define FORCE_BLANK      0x0080

#define KEY_A            0x0001
#define KEY_B            0x0002
#define KEY_SELECT       0x0004
#define KEY_START        0x0008
#define KEY_RIGHT        0x0010
#define KEY_LEFT         0x0020
#define KEY_UP           0x0040
#define KEY_DOWN         0x0080
#define KEY_R            0x0100
#define KEY_L            0x0200

#define GLOBAL_FLAG_RESUME   0x0001u
#define GLOBAL_FLAG_PLAYLIST 0x0002u
#define CLIP_FLAG_AUDIO      0x0001u
#define CLIP_FLAG_LOOP       0x0002u
#define CLIP_FLAG_COMPRESSED 0x0004u
#define CLIP_FLAG_SCENE_PAL  0x0008u
#define GBV5_MAGIC           0x35564247u

#define FRAME_WIDTH       120u
#define FRAME_HEIGHT      80u
#define FRAME_BYTES       9600u
#define GBA_REFRESH_MILLI 59728u

#define UI_BLACK          250u
#define UI_DARK           251u
#define UI_WHITE          252u
#define UI_YELLOW         253u
#define UI_RED            254u
#define UI_GREEN          255u

#define HUD_HOLD_VBLANKS    24u
#define SEEK_HOLD_VBLANKS   24u
#define VOLUME_HOLD_VBLANKS 36u
#define SEEK_REPEAT_VBLANKS 24u

#define ACTION_NONE          0
#define ACTION_RESTART       1
#define ACTION_SEEK_BACK     2
#define ACTION_SEEK_FORWARD  3
#define ACTION_UI_REFRESH    4
#define ACTION_FRAME_BACK    5
#define ACTION_FRAME_FORWARD 6
#define ACTION_HELP          7
#define ACTION_TOGGLE_PAUSE  8

#define PLAY_RESULT_RESTART_CURRENT 0
#define PLAY_RESULT_NEXT_CLIP       1
#define PLAY_RESULT_RETURN_MENU     2

struct GlobalMetadata {
    u32 magic;
    u16 version;
    u16 flags;
    u16 clip_count;
    u16 default_clip;
    u32 clip_table_offset;
    u32 clip_descriptor_size;
    u32 reserved[11];
};

struct ClipDescriptor {
    u32 frame_count;
    u32 frame_bytes;
    u32 video_offset;
    u32 video_index_offset;
    u32 audio_offset;
    u32 audio_size;
    u32 palette_offset;
    u32 palette_index_offset;
    u32 seek_table_offset;
    u32 audio_rate;
    u32 seek_frame_step;
    u16 vblanks_per_frame;
    u16 source_width;
    u16 source_height;
    u16 flags;
    u16 seek_seconds;
    u16 palette_count;
    u16 keyframe_interval;
    u16 reserved0;
    char title[12];
    u32 uncompressed_video_size;
    u32 compressed_video_size;
    u32 reserved[4];
};

struct PlayerUI {
    int muted;
    int volume_level; /* 0, 1 (50%), 2 (100%) */
    int hud_mode;     /* 0 hidden, 1 time, 2 full */
    int hud_last_visible;
    u16 hud_timer;
    u16 mute_timer;
    u16 volume_timer;
    u16 seek_timer;
    int seek_direction;
    int seek_hold_direction;
    u16 seek_hold_counter;
    int help_combo_latched;
    int hud_combo_latched;
};

struct PlaybackClock {
    u32 next_deadline;
    u32 step_whole;
    u32 step_remainder;
    u32 remainder_accum;
};

extern const struct GlobalMetadata gba_video_metadata;

static u8 frame_a[FRAME_BYTES];
static u8 frame_b[FRAME_BYTES];
static const char sram_type[] __attribute__((used)) = "SRAM_V113";

static void wait_vblank(void)
{
    while (REG_VCOUNT >= 160) { }
    while (REG_VCOUNT < 160) { }
}

static u16 keys_down(void)
{
    return (u16)((~REG_KEYINPUT) & 0x03FFu);
}

static const u8 *rom_ptr(u32 offset)
{
    return (const u8 *)(ROM_BASE + offset);
}

static u16 read16(const u8 *p)
{
    return (u16)((u16)p[0] | ((u16)p[1] << 8));
}

static u32 read32(const u8 *p)
{
    return (u32)p[0] | ((u32)p[1] << 8) | ((u32)p[2] << 16) | ((u32)p[3] << 24);
}

static void copy_bytes(u8 *dst, const u8 *src, u32 count)
{
    u32 i;
    for (i = 0; i < count; ++i) dst[i] = src[i];
}

static void copy_palette(const u16 *palette)
{
    u32 i;
    for (i = 0; i < 256u; ++i) PALRAM[i] = palette[i];
}

static void put_logical_pixel(volatile u16 *dst, u32 x, u32 y, u16 colour)
{
    u16 pair;
    volatile u16 *row0;
    volatile u16 *row1;
    if (x >= FRAME_WIDTH || y >= FRAME_HEIGHT) return;
    pair = (u16)(colour | (colour << 8));
    row0 = dst + (y * 2u) * 120u;
    row1 = row0 + 120u;
    row0[x] = pair;
    row1[x] = pair;
}

static void fill_rect(volatile u16 *dst, u32 x, u32 y, u32 width, u32 height, u16 colour)
{
    u32 yy, xx;
    for (yy = 0; yy < height; ++yy) {
        for (xx = 0; xx < width; ++xx) put_logical_pixel(dst, x + xx, y + yy, colour);
    }
}

/* Compact 3x5 uppercase font. */
static u16 glyph_bits(char c)
{
    switch (c) {
    case '0': return 0x7B6Fu; case '1': return 0x2C97u; case '2': return 0x73E7u;
    case '3': return 0x73CFu; case '4': return 0x5BC9u; case '5': return 0x79CFu;
    case '6': return 0x79EFu; case '7': return 0x7292u; case '8': return 0x7BEFu;
    case '9': return 0x7BCFu;
    case 'A': return 0x2BEDu; case 'B': return 0x6BAEu; case 'C': return 0x7927u;
    case 'D': return 0x6B6Eu; case 'E': return 0x79E7u; case 'F': return 0x79E4u;
    case 'G': return 0x79AFu; case 'H': return 0x5BEDu; case 'I': return 0x7497u;
    case 'J': return 0x124Eu; case 'K': return 0x5D6Du; case 'L': return 0x4927u;
    case 'M': return 0x5FE9u; case 'N': return 0x5F6Du; case 'O': return 0x7B6Fu;
    case 'P': return 0x7BE4u; case 'Q': return 0x7B7Bu; case 'R': return 0x7BEDu;
    case 'S': return 0x79CFu; case 'T': return 0x7492u; case 'U': return 0x5B6Fu;
    case 'V': return 0x5B6Au; case 'W': return 0x5BFDu; case 'X': return 0x5AADu;
    case 'Y': return 0x5A92u; case 'Z': return 0x72A7u;
    case ':': return 0x0410u; case '/': return 0x12A4u; case '-': return 0x01C0u;
    case '+': return 0x05D0u; case '.': return 0x0002u; case '>': return 0x22A2u;
    case '<': return 0x1144u; case '%': return 0x5295u; default: return 0u;
    }
}

static void draw_char(volatile u16 *dst, u32 x, u32 y, char c, u16 colour)
{
    u16 bits = glyph_bits(c);
    u32 row, col;
    for (row = 0; row < 5u; ++row) {
        for (col = 0; col < 3u; ++col) {
            u32 bit = 14u - (row * 3u + col);
            if (bits & (1u << bit)) put_logical_pixel(dst, x + col, y + row, colour);
        }
    }
}

static void draw_text(volatile u16 *dst, u32 x, u32 y, const char *text, u32 length, u16 colour)
{
    u32 i;
    for (i = 0; i < length; ++i) draw_char(dst, x + i * 4u, y, text[i], colour);
}

static u32 text_length(const char *s)
{
    u32 n = 0u;
    while (s[n] != 0) ++n;
    return n;
}

static void draw_text_auto(volatile u16 *dst, u32 x, u32 y, const char *text, u16 colour)
{
    draw_text(dst, x, y, text, text_length(text), colour);
}

static u32 divide_u32(u32 numerator, u32 denominator)
{
    u32 quotient = 0u, remainder = 0u;
    int bit;
    if (denominator == 0u) return 0u;
    for (bit = 31; bit >= 0; --bit) {
        remainder = (remainder << 1) | ((numerator >> (u32)bit) & 1u);
        if (remainder >= denominator) {
            remainder -= denominator;
            quotient |= 1u << (u32)bit;
        }
    }
    return quotient;
}

static u32 seconds_for_frame(u32 frame, u16 vblanks)
{
    u32 milli = frame * (u32)vblanks * 1000u;
    return divide_u32(milli + GBA_REFRESH_MILLI / 2u, GBA_REFRESH_MILLI);
}

static void make_time_text(char out[11], u32 current_seconds, u32 total_seconds)
{
    u32 cm = divide_u32(current_seconds, 60u), cs = current_seconds % 60u;
    u32 tm = divide_u32(total_seconds, 60u), ts = total_seconds % 60u;
    u32 tens;
    if (cm > 99u) cm = 99u;
    if (tm > 99u) tm = 99u;
    tens = divide_u32(cm, 10u); out[0] = (char)('0' + tens); out[1] = (char)('0' + cm - tens * 10u);
    out[2] = ':'; tens = divide_u32(cs, 10u); out[3] = (char)('0' + tens); out[4] = (char)('0' + cs - tens * 10u);
    out[5] = '/'; tens = divide_u32(tm, 10u); out[6] = (char)('0' + tens); out[7] = (char)('0' + tm - tens * 10u);
    out[8] = ':'; tens = divide_u32(ts, 10u); out[9] = (char)('0' + tens); out[10] = (char)('0' + ts - tens * 10u);
}

static void make_frame_text(char out[6], u32 frame)
{
    int i;
    if (frame > 99999u) frame = 99999u;
    out[0] = 'F';
    for (i = 5; i >= 1; --i) { out[i] = (char)('0' + (frame % 10u)); frame = divide_u32(frame, 10u); }
}

static void draw_loop_icon(volatile u16 *dst, u32 x, u32 y)
{
    put_logical_pixel(dst,x+2u,y,UI_YELLOW); put_logical_pixel(dst,x+3u,y,UI_YELLOW); put_logical_pixel(dst,x+4u,y,UI_YELLOW); put_logical_pixel(dst,x+6u,y,UI_YELLOW);
    put_logical_pixel(dst,x+1u,y+1u,UI_YELLOW); put_logical_pixel(dst,x+5u,y+1u,UI_YELLOW); put_logical_pixel(dst,x+6u,y+1u,UI_YELLOW);
    put_logical_pixel(dst,x+4u,y+2u,UI_YELLOW); put_logical_pixel(dst,x+5u,y+2u,UI_YELLOW); put_logical_pixel(dst,x+6u,y+2u,UI_YELLOW);
    put_logical_pixel(dst,x,y+3u,UI_YELLOW); put_logical_pixel(dst,x+1u,y+3u,UI_YELLOW); put_logical_pixel(dst,x+2u,y+3u,UI_YELLOW);
    put_logical_pixel(dst,x,y+4u,UI_YELLOW); put_logical_pixel(dst,x+1u,y+4u,UI_YELLOW); put_logical_pixel(dst,x+5u,y+4u,UI_YELLOW);
    put_logical_pixel(dst,x,y+5u,UI_YELLOW); put_logical_pixel(dst,x+2u,y+5u,UI_YELLOW); put_logical_pixel(dst,x+3u,y+5u,UI_YELLOW); put_logical_pixel(dst,x+4u,y+5u,UI_YELLOW);
}

static void draw_hud(volatile u16 *dst, u32 frame, const struct ClipDescriptor *clip, int mode)
{
    u32 total_seconds;
    char time_text[11];
    char frame_text[6];
    if (mode <= 0) return;
    total_seconds = seconds_for_frame(clip->frame_count > 0u ? clip->frame_count - 1u : 0u, clip->vblanks_per_frame);
    make_time_text(time_text, seconds_for_frame(frame, clip->vblanks_per_frame), total_seconds);
    if (mode == 1) {
        fill_rect(dst, 0u, 68u, 120u, 8u, UI_BLACK);
        draw_text(dst, 38u, 69u, time_text, 11u, UI_WHITE);
        return;
    }
    fill_rect(dst, 0u, 67u, 120u, 13u, UI_BLACK);
    draw_text(dst, 3u, 69u, time_text, 11u, UI_WHITE);
    make_frame_text(frame_text, frame + 1u);
    draw_text(dst, 51u, 69u, frame_text, 6u, UI_WHITE);
    fill_rect(dst, 5u, 77u, 110u, 2u, UI_DARK);
    if (clip->frame_count <= 1u) fill_rect(dst, 5u, 77u, 110u, 2u, UI_YELLOW);
    else {
        u32 filled = divide_u32(frame * 110u, clip->frame_count - 1u);
        if (filled > 110u) filled = 110u;
        if (filled > 0u) fill_rect(dst, 5u, 77u, filled, 2u, UI_YELLOW);
    }
    if (clip->flags & CLIP_FLAG_LOOP) draw_loop_icon(dst, 108u, 69u);
}

static void draw_speaker(volatile u16 *dst, u32 x, u32 y, int crossed)
{
    put_logical_pixel(dst,x,y+3u,UI_WHITE); put_logical_pixel(dst,x+1u,y+2u,UI_WHITE); put_logical_pixel(dst,x+1u,y+3u,UI_WHITE); put_logical_pixel(dst,x+1u,y+4u,UI_WHITE);
    put_logical_pixel(dst,x+2u,y+1u,UI_WHITE); put_logical_pixel(dst,x+2u,y+2u,UI_WHITE); put_logical_pixel(dst,x+2u,y+3u,UI_WHITE); put_logical_pixel(dst,x+2u,y+4u,UI_WHITE); put_logical_pixel(dst,x+2u,y+5u,UI_WHITE);
    if (crossed) {
        put_logical_pixel(dst,x+5u,y+1u,UI_RED); put_logical_pixel(dst,x+9u,y+1u,UI_RED); put_logical_pixel(dst,x+6u,y+2u,UI_RED); put_logical_pixel(dst,x+8u,y+2u,UI_RED); put_logical_pixel(dst,x+7u,y+3u,UI_RED); put_logical_pixel(dst,x+6u,y+4u,UI_RED); put_logical_pixel(dst,x+8u,y+4u,UI_RED); put_logical_pixel(dst,x+5u,y+5u,UI_RED); put_logical_pixel(dst,x+9u,y+5u,UI_RED);
    } else {
        put_logical_pixel(dst,x+5u,y+3u,UI_GREEN); put_logical_pixel(dst,x+5u,y+4u,UI_GREEN); put_logical_pixel(dst,x+6u,y+4u,UI_GREEN); put_logical_pixel(dst,x+6u,y+5u,UI_GREEN); put_logical_pixel(dst,x+7u,y+3u,UI_GREEN); put_logical_pixel(dst,x+7u,y+4u,UI_GREEN); put_logical_pixel(dst,x+8u,y+2u,UI_GREEN); put_logical_pixel(dst,x+8u,y+3u,UI_GREEN); put_logical_pixel(dst,x+9u,y+1u,UI_GREEN); put_logical_pixel(dst,x+9u,y+2u,UI_GREEN);
    }
}

static void draw_mute_badge(volatile u16 *dst, int muted)
{
    fill_rect(dst, 107u, 3u, 12u, 7u, UI_BLACK);
    draw_speaker(dst, 108u, 3u, muted);
}

static void draw_volume_badge(volatile u16 *dst, int level)
{
    const char *text = level == 2 ? "V100" : (level == 1 ? "V50" : "V0");
    u32 text_width = text_length(text) * 4u - 1u;
    u32 box_width = text_width + 2u;
    u32 box_x = 119u - box_width;
    fill_rect(dst, box_x, 3u, box_width, 7u, UI_BLACK);
    draw_text_auto(dst, box_x + 1u, 4u, text, UI_WHITE);
}

static void draw_left_arrow(volatile u16 *dst, u32 x, u32 y)
{
    fill_rect(dst,x,y+3u,1u,2u,UI_YELLOW); fill_rect(dst,x+1u,y+2u,1u,4u,UI_YELLOW); fill_rect(dst,x+2u,y+1u,1u,6u,UI_YELLOW); fill_rect(dst,x+3u,y+3u,4u,2u,UI_YELLOW);
}

static void draw_right_arrow(volatile u16 *dst, u32 x, u32 y)
{
    fill_rect(dst,x,y+3u,4u,2u,UI_YELLOW); fill_rect(dst,x+4u,y+1u,1u,6u,UI_YELLOW); fill_rect(dst,x+5u,y+2u,1u,4u,UI_YELLOW); fill_rect(dst,x+6u,y+3u,1u,2u,UI_YELLOW);
}

static u32 number_digits(u32 v) { return v >= 10u ? 2u : 1u; }

static void draw_small_number(volatile u16 *dst, u32 x, u32 y, u32 value, u16 colour)
{
    if (value >= 10u) {
        draw_char(dst, x, y, (char)('0' + divide_u32(value, 10u) % 10u), colour);
        draw_char(dst, x + 4u, y, (char)('0' + value % 10u), colour);
    } else draw_char(dst, x, y, (char)('0' + value), colour);
}

static void draw_seek_badge(volatile u16 *dst, int direction, u32 seconds)
{
    u32 digits = number_digits(seconds);
    u32 number_width = digits == 2u ? 7u : 3u;
    u32 content_width = 7u + 2u + number_width;
    u32 box_w = content_width + 4u;
    u32 box_x = (120u - box_w) / 2u;
    u32 content_x = box_x + 2u;
    fill_rect(dst, box_x, 32u, box_w, 10u, UI_BLACK);
    if (direction < 0) {
        draw_left_arrow(dst, content_x, 33u);
        draw_small_number(dst, content_x + 9u, 34u, seconds, UI_WHITE);
    } else {
        draw_small_number(dst, content_x, 34u, seconds, UI_WHITE);
        draw_right_arrow(dst, content_x + number_width + 2u, 33u);
    }
}

static const u16 *palette_for_frame(const struct ClipDescriptor *clip, u32 frame)
{
    u32 index = 0u;
    if (clip->palette_count > 1u && clip->palette_index_offset != 0u) {
        const u8 *table = rom_ptr(clip->palette_index_offset);
        index = read16(table + frame * 2u);
        if (index >= clip->palette_count) index = 0u;
    }
    return (const u16 *)rom_ptr(clip->palette_offset + index * 512u);
}

static void apply_delta(const u8 *payload, u32 size, u8 *dst)
{
    u32 pos = 0u, off = 0u;
    while (off + 4u <= size && pos < FRAME_BYTES) {
        u32 skip = read16(payload + off), run = read16(payload + off + 2u);
        off += 4u;
        pos += skip;
        if (pos > FRAME_BYTES) pos = FRAME_BYTES;
        if (run > FRAME_BYTES - pos) run = FRAME_BYTES - pos;
        if (off + run > size) run = size - off;
        copy_bytes(dst + pos, payload + off, run);
        pos += run;
        off += run;
    }
}

static const u8 *compressed_record(const struct ClipDescriptor *clip, u32 frame)
{
    const u8 *index = rom_ptr(clip->video_index_offset);
    return rom_ptr(clip->video_offset + read32(index + frame * 4u));
}

static void load_frame_pixels(const struct ClipDescriptor *clip, u32 frame, u8 *dst)
{
    if (!(clip->flags & CLIP_FLAG_COMPRESSED)) {
        copy_bytes(dst, rom_ptr(clip->video_offset + frame * FRAME_BYTES), FRAME_BYTES);
        return;
    }
    {
        u32 base = frame;
        const u8 *record;
        while (base > 0u) {
            record = compressed_record(clip, base);
            if (read32(record) == 0u) break;
            --base;
        }
        record = compressed_record(clip, base);
        if (read32(record) != 0u || read32(record + 4u) < FRAME_BYTES) {
            u32 i; for (i = 0; i < FRAME_BYTES; ++i) dst[i] = 0u;
        } else copy_bytes(dst, record + 8u, FRAME_BYTES);
        while (base < frame) {
            ++base;
            record = compressed_record(clip, base);
            if (read32(record) == 0u) copy_bytes(dst, record + 8u, FRAME_BYTES);
            else apply_delta(record + 8u, read32(record + 4u), dst);
        }
    }
}

static void load_next_pixels(const struct ClipDescriptor *clip, u32 frame, const u8 *current, u8 *dst)
{
    if (!(clip->flags & CLIP_FLAG_COMPRESSED)) {
        copy_bytes(dst, rom_ptr(clip->video_offset + frame * FRAME_BYTES), FRAME_BYTES);
        return;
    }
    copy_bytes(dst, current, FRAME_BYTES);
    {
        const u8 *record = compressed_record(clip, frame);
        if (read32(record) == 0u) copy_bytes(dst, record + 8u, FRAME_BYTES);
        else apply_delta(record + 8u, read32(record + 4u), dst);
    }
}

static void render_pixels(const u8 *src, volatile u16 *dst)
{
    u32 y, x;
    for (y = 0; y < FRAME_HEIGHT; ++y) {
        volatile u16 *row0 = dst + (y * 2u) * 120u;
        volatile u16 *row1 = row0 + 120u;
        for (x = 0; x < FRAME_WIDTH; ++x) {
            u16 p = src[y * FRAME_WIDTH + x];
            p = (u16)(p | (p << 8));
            row0[x] = p; row1[x] = p;
        }
    }
}

static void render_frame_with_ui(const u8 *pixels, u32 frame, volatile u16 *dst,
                                 const struct ClipDescriptor *clip, const struct PlayerUI *ui)
{
    int hud_mode = ui->hud_mode;
    render_pixels(pixels, dst);
    if (ui->hud_timer != 0u && hud_mode < 2) hud_mode = 2;
    draw_hud(dst, frame, clip, hud_mode);
    if (ui->mute_timer != 0u) draw_mute_badge(dst, ui->muted);
    if (ui->volume_timer != 0u) draw_volume_badge(dst, ui->volume_level);
    if (ui->seek_timer != 0u && ui->seek_direction != 0) draw_seek_badge(dst, ui->seek_direction, clip->seek_seconds ? clip->seek_seconds : 5u);
}

static void show_rendered_page(u16 *displayed_page, const u16 *palette)
{
    copy_palette(palette);
    *displayed_page ^= 1u;
    REG_DISPCNT = *displayed_page ? (MODE4_BG2 | PAGE_SELECT) : MODE4_BG2;
}

static void render_and_show(const u8 *pixels, u32 frame, u16 *displayed_page,
                            const struct ClipDescriptor *clip, const struct PlayerUI *ui)
{
    volatile u16 *back = *displayed_page ? VRAM_PAGE0 : VRAM_PAGE1;
    render_frame_with_ui(pixels, frame, back, clip, ui);
    wait_vblank();
    show_rendered_page(displayed_page, palette_for_frame(clip, frame));
}

static u16 sound_control(const struct PlayerUI *ui, int reset)
{
    u16 v = reset ? 0x0800u : 0u;
    if (!ui->muted && ui->volume_level > 0) v |= 0x0300u;
    if (ui->volume_level >= 2) v |= 0x0004u;
    return v;
}

static void audio_stop(void)
{
    REG_TM0CNT_H = 0; REG_DMA1CNT_H = 0; REG_SOUNDCNT_H = 0x0800;
}

static void audio_apply_state(const struct PlayerUI *ui)
{
    REG_SOUNDCNT_H = sound_control(ui, 0);
}

static void audio_start_at(const u8 *audio, u32 byte_offset, int paused, const struct PlayerUI *ui)
{
    audio_stop();
    REG_SOUNDCNT_X = 0x0080; REG_SOUNDCNT_L = 0; REG_SOUNDBIAS = 0x0200;
    REG_SOUNDCNT_H = sound_control(ui, 1);
    REG_DMA1SAD = (u32)(audio + (byte_offset & ~3u)); REG_DMA1DAD = (u32)&REG_FIFO_A;
    REG_DMA1CNT_L = 4; REG_DMA1CNT_H = 0xB640;
    if (!paused) { REG_TM0CNT_L = 0xFC00; REG_TM0CNT_H = 0x0080; }
}

static void audio_pause(void) { REG_TM0CNT_H = 0; }
static void audio_resume(void) { REG_TM0CNT_L = 0xFC00; REG_TM0CNT_H = 0x0080; }

static u32 audio_offset_for_frame(const struct ClipDescriptor *clip, u32 frame)
{
    u32 offset;
    if (clip->seek_table_offset == 0u || frame >= clip->frame_count) return 0u;
    offset = read32(rom_ptr(clip->seek_table_offset + frame * 4u)) & ~3u;
    if (clip->audio_size < 4u) return 0u;
    if (offset > clip->audio_size - 4u) offset = (clip->audio_size - 4u) & ~3u;
    return offset;
}

/* Timer 2 runs at 16,384 Hz; Timer 3 cascades for a 32-bit playback clock. */
static void playback_timer_stop(void)
{
    REG_TM2CNT_H = 0u;
    REG_TM3CNT_H = 0u;
}

static void playback_timer_reset(void)
{
    playback_timer_stop();
    REG_TM2CNT_L = 0u;
    REG_TM3CNT_L = 0u;
    REG_TM3CNT_H = 0x0084u;
    REG_TM2CNT_H = 0x0083u;
}

static void playback_timer_pause(void)
{
    playback_timer_stop();
}

static void playback_timer_resume(void)
{
    REG_TM3CNT_H = 0x0084u;
    REG_TM2CNT_H = 0x0083u;
}

static u32 playback_timer_read(void)
{
    u16 high1, low, high2;
    do {
        high1 = REG_TM3CNT_L;
        low = REG_TM2CNT_L;
        high2 = REG_TM3CNT_L;
    } while (high1 != high2);
    return ((u32)high1 << 16) | (u32)low;
}

static void playback_clock_init(struct PlaybackClock *clock, u16 vblanks)
{
    u32 numerator = (u32)vblanks * 16384000u;
    clock->step_whole = numerator / GBA_REFRESH_MILLI;
    clock->step_remainder = numerator % GBA_REFRESH_MILLI;
    clock->remainder_accum = 0u;
    clock->next_deadline = 0u;
}

static void playback_clock_advance(struct PlaybackClock *clock)
{
    clock->next_deadline += clock->step_whole;
    clock->remainder_accum += clock->step_remainder;
    if (clock->remainder_accum >= GBA_REFRESH_MILLI) {
        clock->remainder_accum -= GBA_REFRESH_MILLI;
        ++clock->next_deadline;
    }
}

static int tick_ui_timers(struct PlayerUI *ui)
{
    int changed = 0;
    if (ui->hud_timer && --ui->hud_timer == 0u) changed = 1;
    if (ui->mute_timer && --ui->mute_timer == 0u) changed = 1;
    if (ui->volume_timer && --ui->volume_timer == 0u) changed = 1;
    if (ui->seek_timer && --ui->seek_timer == 0u) { ui->seek_direction = 0; changed = 1; }
    return changed;
}

static void cycle_hud(struct PlayerUI *ui)
{
    ui->hud_mode = (ui->hud_mode + 1) % 3;
    if (ui->hud_mode != 0) ui->hud_last_visible = ui->hud_mode;
    ui->hud_timer = 0u;
}

static void toggle_hud_combo(struct PlayerUI *ui)
{
    if (ui->hud_mode == 0) ui->hud_mode = ui->hud_last_visible ? ui->hud_last_visible : 2;
    else { ui->hud_last_visible = ui->hud_mode; ui->hud_mode = 0; }
    ui->hud_timer = 0u;
}

static int poll_action(u16 *previous_keys, int paused, int has_audio, struct PlayerUI *ui)
{
    u16 now = keys_down();
    u16 pressed = (u16)(now & (u16)~(*previous_keys));
    int direction = 0;
    *previous_keys = now;

    if ((now & (KEY_START | KEY_SELECT)) == (KEY_START | KEY_SELECT)) {
        if (!ui->help_combo_latched) { ui->help_combo_latched = 1; return ACTION_HELP; }
        return ACTION_NONE;
    }
    if (ui->help_combo_latched) {
        if ((now & (KEY_START | KEY_SELECT)) == 0u) ui->help_combo_latched = 0;
        else return ACTION_NONE;
    }

    if ((now & (KEY_L | KEY_R)) == (KEY_L | KEY_R)) {
        ui->seek_hold_direction = 0; ui->seek_hold_counter = 0u;
        if (!ui->hud_combo_latched) { ui->hud_combo_latched = 1; toggle_hud_combo(ui); return ACTION_UI_REFRESH; }
        return ACTION_NONE;
    }
    if (ui->hud_combo_latched) {
        if ((now & (KEY_L | KEY_R)) == 0u) ui->hud_combo_latched = 0;
        else return ACTION_NONE;
    }

    if (pressed & KEY_B) return ACTION_RESTART;
    if (pressed & KEY_START) { cycle_hud(ui); return ACTION_UI_REFRESH; }
    if (pressed & KEY_A) return ACTION_TOGGLE_PAUSE;
    if ((pressed & KEY_SELECT) && has_audio) {
        ui->muted = !ui->muted; ui->mute_timer = HUD_HOLD_VBLANKS; audio_apply_state(ui); return ACTION_UI_REFRESH;
    }
    if (pressed & KEY_UP) {
        if (ui->volume_level < 2) ++ui->volume_level;
        ui->volume_timer = VOLUME_HOLD_VBLANKS; if (has_audio) audio_apply_state(ui); return ACTION_UI_REFRESH;
    }
    if (pressed & KEY_DOWN) {
        if (ui->volume_level > 0) --ui->volume_level;
        ui->volume_timer = VOLUME_HOLD_VBLANKS; if (has_audio) audio_apply_state(ui); return ACTION_UI_REFRESH;
    }

    if (paused) {
        if (pressed & KEY_LEFT) return ACTION_FRAME_BACK;
        if (pressed & KEY_RIGHT) return ACTION_FRAME_FORWARD;
        if (now & KEY_L) direction = -1;
        else if (now & KEY_R) direction = 1;
    } else {
        if (now & (KEY_L | KEY_LEFT)) direction = -1;
        else if (now & (KEY_R | KEY_RIGHT)) direction = 1;
    }

    if (direction != 0) {
        u16 direction_keys = direction < 0 ? (KEY_L | (paused ? 0u : KEY_LEFT)) : (KEY_R | (paused ? 0u : KEY_RIGHT));
        if (pressed & direction_keys) {
            ui->seek_hold_direction = direction; ui->seek_hold_counter = 0u;
            return direction < 0 ? ACTION_SEEK_BACK : ACTION_SEEK_FORWARD;
        }
        if (ui->seek_hold_direction != direction) {
            ui->seek_hold_direction = direction; ui->seek_hold_counter = 0u;
        } else {
            ++ui->seek_hold_counter;
            if (ui->seek_hold_counter >= SEEK_REPEAT_VBLANKS) {
                ui->seek_hold_counter = 0u;
                return direction < 0 ? ACTION_SEEK_BACK : ACTION_SEEK_FORWARD;
            }
        }
    } else { ui->seek_hold_direction = 0; ui->seek_hold_counter = 0u; }
    return ACTION_NONE;
}

static int wait_frame_period(u16 *previous_keys, u32 deadline, int has_audio, int *paused,
                             struct PlayerUI *ui)
{
    for (;;) {
        int changed, action;
        wait_vblank();
        changed = tick_ui_timers(ui);
        action = poll_action(previous_keys, *paused, has_audio, ui);
        if (action == ACTION_TOGGLE_PAUSE) {
            *paused = !*paused; ui->hud_timer = HUD_HOLD_VBLANKS;
            if (*paused) {
                playback_timer_pause();
                if (has_audio) audio_pause();
            } else {
                playback_timer_resume();
                if (has_audio) audio_resume();
            }
            return ACTION_UI_REFRESH;
        }
        if (action != ACTION_NONE) return action;
        if (*paused && changed) return ACTION_UI_REFRESH;
        if (!*paused && playback_timer_read() >= deadline) return ACTION_NONE;
    }
}

static u32 seek_target(u32 frame, u32 frame_count, u32 step, int forward)
{
    if (step == 0u) step = 50u;
    if (forward) {
        if (frame >= frame_count - 1u || step >= frame_count - frame) return frame_count - 1u;
        return frame + step;
    }
    return frame <= step ? 0u : frame - step;
}

static void start_seek_feedback(struct PlayerUI *ui, int direction)
{
    ui->seek_direction = direction; ui->seek_timer = SEEK_HOLD_VBLANKS; ui->hud_timer = HUD_HOLD_VBLANKS;
}

static void clear_screen(volatile u16 *dst)
{
    fill_rect(dst, 0u, 0u, 120u, 80u, UI_BLACK);
}

static int is_menu_mode(const struct GlobalMetadata *meta)
{
    return meta->clip_count > 1u && !(meta->flags & GLOBAL_FLAG_PLAYLIST);
}

static void draw_help(volatile u16 *dst, int menu_mode)
{
    clear_screen(dst);
    draw_text_auto(dst, 42u, 2u, "CONTROLS", UI_YELLOW);
    draw_text_auto(dst, 3u, 9u, "A PAUSE PLAY", UI_WHITE);
    draw_text_auto(dst, 3u, 15u, menu_mode ? "B RETURN MENU" : "B RESTART", UI_WHITE);
    draw_text_auto(dst, 3u, 21u, "START HUD MODE", UI_WHITE);
    draw_text_auto(dst, 3u, 27u, "SELECT MUTE", UI_WHITE);
    draw_text_auto(dst, 3u, 33u, "L R SEEK HOLD", UI_WHITE);
    draw_text_auto(dst, 3u, 39u, "LEFT RIGHT SEEK", UI_WHITE);
    draw_text_auto(dst, 3u, 45u, "PAUSED LEFT RIGHT FRAME", UI_WHITE);
    draw_text_auto(dst, 3u, 51u, "UP DOWN VOLUME", UI_WHITE);
    draw_text_auto(dst, 3u, 57u, "L+R HUD SHOW HIDE", UI_WHITE);
    draw_text_auto(dst, 3u, 63u, "START+SELECT HELP", UI_WHITE);
    draw_text_auto(dst, 22u, 72u, "PRESS ANY BUTTON", UI_YELLOW);
}

static void set_ui_palette(void);

static void show_help_screen(u16 *displayed_page, int menu_mode)
{
    volatile u16 *back = *displayed_page ? VRAM_PAGE0 : VRAM_PAGE1;
    u16 now;
    draw_help(back, menu_mode);
    wait_vblank();
    set_ui_palette();
    *displayed_page ^= 1u;
    REG_DISPCNT = *displayed_page ? (MODE4_BG2 | PAGE_SELECT) : MODE4_BG2;
    while (keys_down() != 0u) wait_vblank();
    do { wait_vblank(); now = keys_down(); } while (now == 0u);
    while (keys_down() != 0u) wait_vblank();
}

static void set_ui_palette(void)
{
    u32 i;
    for (i=0;i<256u;++i) PALRAM[i]=0;
    PALRAM[UI_BLACK]=0x0000; PALRAM[UI_DARK]=0x18C6; PALRAM[UI_WHITE]=0x7FFF; PALRAM[UI_YELLOW]=0x037F; PALRAM[UI_RED]=0x001F; PALRAM[UI_GREEN]=0x03E0;
}

static u32 sram_read_u32(u32 off)
{
    return (u32)SRAM_BASE[off] | ((u32)SRAM_BASE[off+1u]<<8) | ((u32)SRAM_BASE[off+2u]<<16) | ((u32)SRAM_BASE[off+3u]<<24);
}

static void sram_write_u32(u32 off, u32 v)
{
    SRAM_BASE[off]=(u8)v; SRAM_BASE[off+1u]=(u8)(v>>8); SRAM_BASE[off+2u]=(u8)(v>>16); SRAM_BASE[off+3u]=(u8)(v>>24);
}

static void save_position(u32 clip, u32 frame)
{
    u32 check = 0x47564238u ^ clip ^ frame;
    sram_write_u32(0u,0x47564238u); sram_write_u32(4u,clip); sram_write_u32(8u,frame); sram_write_u32(12u,check);
}

static void clear_position(void) { sram_write_u32(0u,0u); }

static int load_position(const struct GlobalMetadata *meta, u32 *clip, u32 *frame)
{
    u32 c, f, check;
    if (!(meta->flags & GLOBAL_FLAG_RESUME) || sram_type[0] != 'S') return 0;
    if (sram_read_u32(0u) != 0x47564238u) return 0;
    c=sram_read_u32(4u); f=sram_read_u32(8u); check=sram_read_u32(12u);
    if (check != (0x47564238u ^ c ^ f) || c >= meta->clip_count) return 0;
    *clip=c; *frame=f; return 1;
}

static int resume_prompt(u32 seconds)
{
    volatile u16 *dst = VRAM_PAGE0;
    char time_text[11];
    make_time_text(time_text, seconds, seconds);
    clear_screen(dst); set_ui_palette();
    draw_text_auto(dst, 22u, 23u, "CONTINUE FROM", UI_YELLOW);
    draw_text(dst, 38u, 32u, time_text, 5u, UI_WHITE);
    draw_text_auto(dst, 28u, 45u, "A CONTINUE", UI_WHITE);
    draw_text_auto(dst, 28u, 53u, "B RESTART", UI_WHITE);
    REG_DISPCNT = MODE4_BG2;
    while (keys_down()!=0u) wait_vblank();
    for (;;) {
        u16 p; wait_vblank(); p=keys_down();
        if (p & KEY_A) { while(keys_down()!=0u) wait_vblank(); return 1; }
        if (p & KEY_B) { while(keys_down()!=0u) wait_vblank(); return 0; }
    }
}

static void draw_menu_arrow(volatile u16 *dst, u32 x, u32 y, u16 colour)
{
    /* A clear five-pixel right arrow instead of the tiny font chevron. */
    put_logical_pixel(dst, x + 2u, y + 0u, colour);
    put_logical_pixel(dst, x + 2u, y + 1u, colour);
    put_logical_pixel(dst, x + 3u, y + 1u, colour);
    put_logical_pixel(dst, x + 0u, y + 2u, colour);
    put_logical_pixel(dst, x + 1u, y + 2u, colour);
    put_logical_pixel(dst, x + 2u, y + 2u, colour);
    put_logical_pixel(dst, x + 3u, y + 2u, colour);
    put_logical_pixel(dst, x + 4u, y + 2u, colour);
    put_logical_pixel(dst, x + 2u, y + 3u, colour);
    put_logical_pixel(dst, x + 3u, y + 3u, colour);
    put_logical_pixel(dst, x + 2u, y + 4u, colour);
}

static u32 select_clip_menu(const struct GlobalMetadata *meta, const struct ClipDescriptor *clips, u32 initial_selection)
{
    u32 selected = initial_selection < meta->clip_count ? initial_selection : (meta->default_clip < meta->clip_count ? meta->default_clip : 0u);
    u32 top = 0u;
    u16 prev = keys_down();
    if (meta->clip_count <= 1u) return 0u;
    for (;;) {
        volatile u16 *dst = VRAM_PAGE0;
        u32 row;
        clear_screen(dst); set_ui_palette();
        draw_text_auto(dst, 34u, 2u, "SELECT VIDEO", UI_YELLOW);
        if (selected < top) top=selected;
        if (selected >= top+10u) top=selected-9u;
        for (row=0u; row<10u && top+row<meta->clip_count; ++row) {
            const struct ClipDescriptor *clip=&clips[top+row];
            u16 col=(top+row==selected)?UI_YELLOW:UI_WHITE;
            if (top+row==selected) draw_menu_arrow(dst,1u,10u+row*6u,col);
            draw_text(dst,8u,10u+row*6u,clip->title,12u,col);
        }
        REG_DISPCNT=MODE4_BG2;
        for (;;) {
            u16 now, pressed; wait_vblank(); now=keys_down(); pressed=(u16)(now & (u16)~prev); prev=now;
            if (pressed & KEY_UP) { selected = selected==0u ? meta->clip_count-1u : selected-1u; break; }
            if (pressed & KEY_DOWN) { selected = selected+1u>=meta->clip_count ? 0u : selected+1u; break; }
            if (pressed & KEY_A) { while(keys_down()!=0u) wait_vblank(); return selected; }
        }
    }
}

static int play_clip(const struct GlobalMetadata *meta, const struct ClipDescriptor *clip, u32 clip_index,
                     u32 initial_frame, struct PlayerUI *ui)
{
    const u8 *audio = rom_ptr(clip->audio_offset);
    int has_audio = (clip->flags & CLIP_FLAG_AUDIO) && clip->audio_size && clip->seek_table_offset;
    u32 frame = initial_frame < clip->frame_count ? initial_frame : 0u;
    struct PlaybackClock clock;
    u16 displayed_page = 0u;
    u16 previous_keys;
    int paused = 0, at_end = 0;
    u8 *current = frame_a, *next = frame_b;

    load_frame_pixels(clip, frame, current);
    audio_stop(); REG_DISPCNT=FORCE_BLANK;
    copy_palette(palette_for_frame(clip,frame));
    render_frame_with_ui(current,frame,VRAM_PAGE0,clip,ui);
    wait_vblank(); REG_DISPCNT=MODE4_BG2;
    previous_keys=keys_down();
    playback_clock_init(&clock, clip->vblanks_per_frame);
    playback_timer_reset();
    playback_clock_advance(&clock);
    if (has_audio) audio_start_at(audio,audio_offset_for_frame(clip,frame),0,ui);
    if (meta->flags & GLOBAL_FLAG_RESUME) save_position(clip_index,frame);

    for (;;) {
        if (at_end) {
            int redraw=0, action;
            wait_vblank(); if (tick_ui_timers(ui)) redraw=1;
            action=poll_action(&previous_keys,0,has_audio,ui);
            if (action==ACTION_RESTART) { playback_timer_stop(); audio_stop(); clear_position(); return is_menu_mode(meta) ? PLAY_RESULT_RETURN_MENU : PLAY_RESULT_RESTART_CURRENT; }
            if (action==ACTION_HELP) {
                playback_timer_stop(); audio_stop();
                show_help_screen(&displayed_page, is_menu_mode(meta)); render_and_show(current,frame,&displayed_page,clip,ui); previous_keys=keys_down();
                playback_timer_stop(); audio_stop();
                continue;
            }
            if (action==ACTION_UI_REFRESH) redraw=1;
            if (action==ACTION_SEEK_BACK || action==ACTION_SEEK_FORWARD) {
                u32 target=seek_target(frame,clip->frame_count,clip->seek_frame_step,action==ACTION_SEEK_FORWARD);
                if (target!=frame) {
                    start_seek_feedback(ui,action==ACTION_SEEK_FORWARD?1:-1); load_frame_pixels(clip,target,current); frame=target; at_end=0; paused=0;
                    render_and_show(current,frame,&displayed_page,clip,ui); previous_keys=keys_down();
                    playback_clock_init(&clock, clip->vblanks_per_frame); playback_timer_reset(); playback_clock_advance(&clock);
                    if (has_audio) audio_start_at(audio,audio_offset_for_frame(clip,frame),0,ui);
                    if (meta->flags & GLOBAL_FLAG_RESUME) save_position(clip_index,frame);
                }
                continue;
            }
            if (redraw) { render_and_show(current,frame,&displayed_page,clip,ui); previous_keys=keys_down(); }
            continue;
        }
        {
            int has_next = frame + 1u < clip->frame_count;
            volatile u16 *back = displayed_page ? VRAM_PAGE0 : VRAM_PAGE1;
            int action;
            if (has_next) { load_next_pixels(clip,frame+1u,current,next); render_frame_with_ui(next,frame+1u,back,clip,ui); }
            action=wait_frame_period(&previous_keys,clock.next_deadline,has_audio,&paused,ui);
            if (action==ACTION_RESTART) { playback_timer_stop(); audio_stop(); clear_position(); return is_menu_mode(meta) ? PLAY_RESULT_RETURN_MENU : PLAY_RESULT_RESTART_CURRENT; }
            if (action==ACTION_HELP) {
                playback_timer_pause(); if (has_audio) audio_pause();
                show_help_screen(&displayed_page, is_menu_mode(meta)); render_and_show(current,frame,&displayed_page,clip,ui); previous_keys=keys_down();
                playback_clock_init(&clock, clip->vblanks_per_frame); playback_timer_reset(); playback_clock_advance(&clock);
                if (paused) playback_timer_pause();
                if (has_audio) audio_start_at(audio,audio_offset_for_frame(clip,frame),paused,ui);
                continue;
            }
            if (action==ACTION_UI_REFRESH) { render_and_show(current,frame,&displayed_page,clip,ui); previous_keys=keys_down(); continue; }
            if (action==ACTION_FRAME_BACK || action==ACTION_FRAME_FORWARD) {
                u32 target=frame;
                if (action==ACTION_FRAME_BACK && frame>0u) target=frame-1u;
                if (action==ACTION_FRAME_FORWARD && frame+1u<clip->frame_count) target=frame+1u;
                if (target!=frame) {
                    load_frame_pixels(clip,target,current); frame=target; ui->hud_timer=HUD_HOLD_VBLANKS;
                    render_and_show(current,frame,&displayed_page,clip,ui); previous_keys=keys_down();
                    playback_clock_init(&clock, clip->vblanks_per_frame); playback_timer_reset(); playback_clock_advance(&clock); playback_timer_pause();
                    if (has_audio) audio_start_at(audio,audio_offset_for_frame(clip,frame),1,ui);
                    if (meta->flags & GLOBAL_FLAG_RESUME) save_position(clip_index,frame);
                }
                continue;
            }
            if (action==ACTION_SEEK_BACK || action==ACTION_SEEK_FORWARD) {
                u32 target=seek_target(frame,clip->frame_count,clip->seek_frame_step,action==ACTION_SEEK_FORWARD);
                if (target!=frame) {
                    if (has_audio) audio_stop(); start_seek_feedback(ui,action==ACTION_SEEK_FORWARD?1:-1);
                    load_frame_pixels(clip,target,current); frame=target;
                    render_and_show(current,frame,&displayed_page,clip,ui); previous_keys=keys_down();
                    playback_clock_init(&clock, clip->vblanks_per_frame); playback_timer_reset(); playback_clock_advance(&clock); if (paused) playback_timer_pause();
                    if (has_audio) audio_start_at(audio,audio_offset_for_frame(clip,frame),paused,ui);
                    if (meta->flags & GLOBAL_FLAG_RESUME) save_position(clip_index,frame);
                }
                continue;
            }
            if (has_next) {
                show_rendered_page(&displayed_page,palette_for_frame(clip,frame+1u));
                { u8 *tmp=current; current=next; next=tmp; }
                ++frame; playback_clock_advance(&clock);
                if ((meta->flags & GLOBAL_FLAG_RESUME) && (frame % 10u == 0u)) save_position(clip_index,frame);
            } else {
                playback_timer_stop(); audio_stop();
                if (clip->flags & CLIP_FLAG_LOOP) { clear_position(); return PLAY_RESULT_RESTART_CURRENT; }
                if (is_menu_mode(meta)) {
                    clear_position();
                    return PLAY_RESULT_RETURN_MENU;
                }
                if ((meta->flags & GLOBAL_FLAG_PLAYLIST) && clip_index + 1u < meta->clip_count) {
                    if (meta->flags & GLOBAL_FLAG_RESUME) save_position(clip_index + 1u, 0u); else clear_position();
                    return PLAY_RESULT_NEXT_CLIP;
                }
                at_end=1; if (meta->flags & GLOBAL_FLAG_RESUME) save_position(clip_index,frame);
            }
        }
    }
}

void main(void)
{
    const struct GlobalMetadata *meta=&gba_video_metadata;
    const struct ClipDescriptor *clips;
    struct PlayerUI ui;
    u32 selected=0u, saved_clip=0u, saved_frame=0u;
    int have_resume=0;

    REG_IME=0; REG_WAITCNT=0x4317; REG_DISPCNT=FORCE_BLANK; playback_timer_stop();
    if (meta->magic!=GBV5_MAGIC || meta->version!=5u || meta->clip_count==0u || meta->clip_descriptor_size!=96u) for(;;){}
    clips=(const struct ClipDescriptor *)rom_ptr(meta->clip_table_offset);
    REG_BG2PA=0x0100; REG_BG2PB=0; REG_BG2PC=0; REG_BG2PD=0x0100; REG_BG2X=0; REG_BG2Y=0;
    ui.muted=0; ui.volume_level=2; ui.hud_mode=0; ui.hud_last_visible=2; ui.hud_timer=0; ui.mute_timer=0; ui.volume_timer=0; ui.seek_timer=0; ui.seek_direction=0; ui.seek_hold_direction=0; ui.seek_hold_counter=0; ui.help_combo_latched=0; ui.hud_combo_latched=0;

    have_resume=load_position(meta,&saved_clip,&saved_frame);
    if (have_resume && saved_clip<meta->clip_count && saved_frame>0u && saved_frame+1u<clips[saved_clip].frame_count) {
        selected=saved_clip;
        if (!resume_prompt(seconds_for_frame(saved_frame,clips[saved_clip].vblanks_per_frame))) {
            clear_position(); have_resume=0;
        }
    } else {
        have_resume=0;
        if (meta->flags & GLOBAL_FLAG_PLAYLIST) selected=0u;
        else selected=select_clip_menu(meta,clips,selected);
    }

    for (;;) {
        int result;
        u32 start=(have_resume && selected==saved_clip)?saved_frame:0u;
        have_resume=0;
        result=play_clip(meta,&clips[selected],selected,start,&ui);
        if (result == PLAY_RESULT_NEXT_CLIP && selected + 1u < meta->clip_count) {
            selected++;
        } else if (result == PLAY_RESULT_RETURN_MENU && is_menu_mode(meta)) {
            selected=select_clip_menu(meta,clips,selected);
        }
    }
}

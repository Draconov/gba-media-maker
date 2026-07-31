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
#define REG_KEYINPUT     REG16(0x04000130)

#define PALRAM           ((volatile u16 *)0x05000000)
#define VRAM_PAGE0       ((volatile u16 *)0x06000000)
#define VRAM_PAGE1       ((volatile u16 *)0x0600A000)
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
#define KEY_R            0x0100
#define KEY_L            0x0200

#define FLAG_AUDIO       0x0001u
#define FLAG_LOOP        0x0002u
#define GBV4_MAGIC       0x34564247u

#define FRAME_WIDTH      120u
#define FRAME_HEIGHT     80u
#define FRAME_BYTES      9600u
#define GBA_REFRESH_MILLI 59728u

/* Palette entries 250-255 are reserved by the converter for the player HUD. */
#define UI_BLACK         250u
#define UI_DARK          251u
#define UI_WHITE         252u
#define UI_YELLOW        253u
#define UI_RED           254u
#define UI_GREEN         255u

#define HUD_HOLD_VBLANKS 24u
#define SEEK_HOLD_VBLANKS 24u

#define ACTION_NONE          0
#define ACTION_RESTART       1
#define ACTION_SEEK_BACK     2
#define ACTION_SEEK_FORWARD  3
#define ACTION_UI_REFRESH    4

struct VideoMetadata {
    u32 magic;
    u16 version;
    u16 flags;
    u32 frame_count;
    u32 frame_bytes;
    u32 video_offset;
    u32 audio_offset;
    u32 audio_size;
    u32 palette_offset;
    u32 audio_rate;
    u16 vblanks_per_frame;
    u16 source_width;
    u16 source_height;
    u16 reserved0;
    u32 samples_required;
    u32 seek_table_offset;
    u32 seek_frame_step;
    u32 reserved2;
    u32 reserved3;
};

struct PlayerUI {
    int muted;
    int hud_pinned;
    u16 hud_timer;
    u16 mute_timer;
    u16 seek_timer;
    int seek_direction;
};

extern const struct VideoMetadata gba_video_metadata;

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

static void copy_palette(const u16 *palette)
{
    u32 i;
    for (i = 0; i < 256u; ++i) {
        PALRAM[i] = palette[i];
    }
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
    u32 yy;
    u32 xx;
    for (yy = 0; yy < height; ++yy) {
        for (xx = 0; xx < width; ++xx) {
            put_logical_pixel(dst, x + xx, y + yy, colour);
        }
    }
}

/* 3x5 glyphs, encoded row-major in the low 15 bits. */
static u16 glyph_bits(char c)
{
    switch (c) {
    case '0': return 0x7B6Fu; /* 111 101 101 101 111 */
    case '1': return 0x2C97u; /* 010 110 010 010 111 */
    case '2': return 0x73E7u; /* 111 001 111 100 111 */
    case '3': return 0x73CFu; /* 111 001 111 001 111 */
    case '4': return 0x5BC9u; /* 101 101 111 001 001 */
    case '5': return 0x79CFu; /* 111 100 111 001 111 */
    case '6': return 0x79EFu; /* 111 100 111 101 111 */
    case '7': return 0x7292u; /* 111 001 010 010 010 */
    case '8': return 0x7BEFu; /* 111 101 111 101 111 */
    case '9': return 0x7BCFu; /* 111 101 111 001 111 */
    case ':': return 0x0410u; /* 000 010 000 010 000 */
    case '/': return 0x12A4u; /* 001 001 010 100 100 */
    case 'S': return 0x79CFu;
    default:  return 0u;
    }
}

static void draw_char(volatile u16 *dst, u32 x, u32 y, char c, u16 colour)
{
    u16 bits = glyph_bits(c);
    u32 row;
    u32 col;
    for (row = 0; row < 5u; ++row) {
        for (col = 0; col < 3u; ++col) {
            u32 bit = 14u - (row * 3u + col);
            if (bits & (1u << bit)) {
                put_logical_pixel(dst, x + col, y + row, colour);
            }
        }
    }
}

static void draw_text(volatile u16 *dst, u32 x, u32 y, const char *text, u32 length, u16 colour)
{
    u32 i;
    for (i = 0; i < length; ++i) {
        draw_char(dst, x + i * 4u, y, text[i], colour);
    }
}


static u32 divide_u32(u32 numerator, u32 denominator)
{
    u32 quotient = 0u;
    u32 remainder = 0u;
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
    u32 cm = divide_u32(current_seconds, 60u);
    u32 cs = current_seconds - cm * 60u;
    u32 tm = divide_u32(total_seconds, 60u);
    u32 ts = total_seconds - tm * 60u;
    if (cm > 99u) cm = 99u;
    if (tm > 99u) tm = 99u;
    {
        u32 tens = divide_u32(cm, 10u);
        out[0] = (char)('0' + tens);
        out[1] = (char)('0' + (cm - tens * 10u));
    }
    out[2] = ':';
    {
        u32 tens = divide_u32(cs, 10u);
        out[3] = (char)('0' + tens);
        out[4] = (char)('0' + (cs - tens * 10u));
    }
    out[5] = '/';
    {
        u32 tens = divide_u32(tm, 10u);
        out[6] = (char)('0' + tens);
        out[7] = (char)('0' + (tm - tens * 10u));
    }
    out[8] = ':';
    {
        u32 tens = divide_u32(ts, 10u);
        out[9] = (char)('0' + tens);
        out[10] = (char)('0' + (ts - tens * 10u));
    }
}

static void draw_loop_icon(volatile u16 *dst, u32 x, u32 y)
{
    /* Exact 7x6 repeat icon from the latest provided reference image. */
    put_logical_pixel(dst, x + 2u, y + 0u, UI_YELLOW);
    put_logical_pixel(dst, x + 3u, y + 0u, UI_YELLOW);
    put_logical_pixel(dst, x + 4u, y + 0u, UI_YELLOW);
    put_logical_pixel(dst, x + 6u, y + 0u, UI_YELLOW);

    put_logical_pixel(dst, x + 1u, y + 1u, UI_YELLOW);
    put_logical_pixel(dst, x + 5u, y + 1u, UI_YELLOW);
    put_logical_pixel(dst, x + 6u, y + 1u, UI_YELLOW);

    put_logical_pixel(dst, x + 4u, y + 2u, UI_YELLOW);
    put_logical_pixel(dst, x + 5u, y + 2u, UI_YELLOW);
    put_logical_pixel(dst, x + 6u, y + 2u, UI_YELLOW);

    put_logical_pixel(dst, x + 0u, y + 3u, UI_YELLOW);
    put_logical_pixel(dst, x + 1u, y + 3u, UI_YELLOW);
    put_logical_pixel(dst, x + 2u, y + 3u, UI_YELLOW);

    put_logical_pixel(dst, x + 0u, y + 4u, UI_YELLOW);
    put_logical_pixel(dst, x + 1u, y + 4u, UI_YELLOW);
    put_logical_pixel(dst, x + 5u, y + 4u, UI_YELLOW);

    put_logical_pixel(dst, x + 0u, y + 5u, UI_YELLOW);
    put_logical_pixel(dst, x + 2u, y + 5u, UI_YELLOW);
    put_logical_pixel(dst, x + 3u, y + 5u, UI_YELLOW);
    put_logical_pixel(dst, x + 4u, y + 5u, UI_YELLOW);
}

static void draw_progress_hud(volatile u16 *dst, u32 frame, u32 frame_count, u16 vblanks, int show_loop)
{
    const u32 track_x = 5u;
    const u32 track_width = 110u;
    u32 filled = 0u;
    u32 total_seconds;
    char text[11];

    fill_rect(dst, 0u, 67u, 120u, 13u, UI_BLACK);
    total_seconds = seconds_for_frame(frame_count > 0u ? frame_count - 1u : 0u, vblanks);
    make_time_text(text, seconds_for_frame(frame, vblanks), total_seconds);
    draw_text(dst, 38u, 69u, text, 11u, UI_WHITE);

    fill_rect(dst, track_x, 77u, track_width, 2u, UI_DARK);
    if (frame_count <= 1u) {
        filled = track_width;
    } else {
        filled = divide_u32(frame * track_width, frame_count - 1u);
        if (filled > track_width) filled = track_width;
    }
    if (filled > 0u) fill_rect(dst, track_x, 77u, filled, 2u, UI_YELLOW);
    if (show_loop) {
        draw_loop_icon(dst, 108u, 69u);
    }
}

static void draw_speaker(volatile u16 *dst, u32 x, u32 y, int crossed)
{
    put_logical_pixel(dst, x, y + 3u, UI_WHITE);
    put_logical_pixel(dst, x + 1u, y + 2u, UI_WHITE);
    put_logical_pixel(dst, x + 1u, y + 3u, UI_WHITE);
    put_logical_pixel(dst, x + 1u, y + 4u, UI_WHITE);
    put_logical_pixel(dst, x + 2u, y + 1u, UI_WHITE);
    put_logical_pixel(dst, x + 2u, y + 2u, UI_WHITE);
    put_logical_pixel(dst, x + 2u, y + 3u, UI_WHITE);
    put_logical_pixel(dst, x + 2u, y + 4u, UI_WHITE);
    put_logical_pixel(dst, x + 2u, y + 5u, UI_WHITE);
    if (crossed) {
        put_logical_pixel(dst, x + 5u, y + 1u, UI_RED);
        put_logical_pixel(dst, x + 9u, y + 1u, UI_RED);
        put_logical_pixel(dst, x + 6u, y + 2u, UI_RED);
        put_logical_pixel(dst, x + 8u, y + 2u, UI_RED);
        put_logical_pixel(dst, x + 7u, y + 3u, UI_RED);
        put_logical_pixel(dst, x + 6u, y + 4u, UI_RED);
        put_logical_pixel(dst, x + 8u, y + 4u, UI_RED);
        put_logical_pixel(dst, x + 5u, y + 5u, UI_RED);
        put_logical_pixel(dst, x + 9u, y + 5u, UI_RED);
    } else {
        /* Compact green check mark, matching the requested drawn shape. */
        put_logical_pixel(dst, x + 5u, y + 3u, UI_GREEN);
        put_logical_pixel(dst, x + 5u, y + 4u, UI_GREEN);
        put_logical_pixel(dst, x + 6u, y + 4u, UI_GREEN);
        put_logical_pixel(dst, x + 6u, y + 5u, UI_GREEN);
        put_logical_pixel(dst, x + 7u, y + 3u, UI_GREEN);
        put_logical_pixel(dst, x + 7u, y + 4u, UI_GREEN);
        put_logical_pixel(dst, x + 8u, y + 2u, UI_GREEN);
        put_logical_pixel(dst, x + 8u, y + 3u, UI_GREEN);
        put_logical_pixel(dst, x + 9u, y + 1u, UI_GREEN);
        put_logical_pixel(dst, x + 9u, y + 2u, UI_GREEN);
    }
}

static void draw_mute_badge(volatile u16 *dst, int muted)
{
    fill_rect(dst, 107u, 3u, 12u, 7u, UI_BLACK);
    draw_speaker(dst, 108u, 3u, muted);
}

static void draw_left_arrow(volatile u16 *dst, u32 x, u32 y)
{
    fill_rect(dst, x, y + 3u, 1u, 2u, UI_YELLOW);
    fill_rect(dst, x + 1u, y + 2u, 1u, 4u, UI_YELLOW);
    fill_rect(dst, x + 2u, y + 1u, 1u, 6u, UI_YELLOW);
    fill_rect(dst, x + 3u, y + 3u, 4u, 2u, UI_YELLOW);
}

static void draw_right_arrow(volatile u16 *dst, u32 x, u32 y)
{
    fill_rect(dst, x, y + 3u, 4u, 2u, UI_YELLOW);
    fill_rect(dst, x + 4u, y + 1u, 1u, 6u, UI_YELLOW);
    fill_rect(dst, x + 5u, y + 2u, 1u, 4u, UI_YELLOW);
    fill_rect(dst, x + 6u, y + 3u, 1u, 2u, UI_YELLOW);
}

static void draw_seek_badge(volatile u16 *dst, int direction)
{
    const u32 box_x = 50u;
    const u32 box_y = 32u;
    const u32 box_w = 18u;
    const u32 box_h = 10u;

    fill_rect(dst, box_x, box_y, box_w, box_h, UI_BLACK);
    if (direction < 0) {
        draw_left_arrow(dst, 51u, 33u);
        draw_text(dst, 60u, 34u, "5", 1u, UI_WHITE);
    } else {
        draw_text(dst, 54u, 34u, "5", 1u, UI_WHITE);
        draw_right_arrow(dst, 60u, 33u);
    }
}

static void render_frame(const u8 *video, u32 frame_number, volatile u16 *dst)
{
    const u8 *src = video + frame_number * FRAME_BYTES;
    u32 y;

    for (y = 0; y < FRAME_HEIGHT; ++y) {
        volatile u16 *row0 = dst + (y * 2u) * 120u;
        volatile u16 *row1 = row0 + 120u;
        u32 x;
        for (x = 0; x < FRAME_WIDTH; ++x) {
            u16 p = src[y * FRAME_WIDTH + x];
            p = (u16)(p | (p << 8));
            row0[x] = p;
            row1[x] = p;
        }
    }
}

static void render_frame_with_ui(const u8 *video, u32 frame_number, volatile u16 *dst,
                                 const struct VideoMetadata *meta, const struct PlayerUI *ui,
                                 int paused, int has_audio)
{
    int show_hud;
    render_frame(video, frame_number, dst);
    show_hud = ui->hud_pinned || ui->hud_timer != 0u || paused;
    if (show_hud) {
        draw_progress_hud(dst, frame_number, meta->frame_count, meta->vblanks_per_frame, (meta->flags & FLAG_LOOP) != 0u);
    }
    if (has_audio && ui->mute_timer != 0u) {
        draw_mute_badge(dst, ui->muted);
    }
    if (ui->seek_timer != 0u && ui->seek_direction != 0) {
        draw_seek_badge(dst, ui->seek_direction);
    }
}

static void show_rendered_page(u16 *displayed_page)
{
    *displayed_page ^= 1u;
    REG_DISPCNT = *displayed_page ? (MODE4_BG2 | PAGE_SELECT) : MODE4_BG2;
}

static void render_and_show(const u8 *video, u32 frame_number, u16 *displayed_page,
                            const struct VideoMetadata *meta, const struct PlayerUI *ui,
                            int paused, int has_audio)
{
    volatile u16 *back = *displayed_page ? VRAM_PAGE0 : VRAM_PAGE1;
    render_frame_with_ui(video, frame_number, back, meta, ui, paused, has_audio);
    wait_vblank();
    show_rendered_page(displayed_page);
}

static void audio_stop(void)
{
    REG_TM0CNT_H = 0;
    REG_DMA1CNT_H = 0;
    REG_SOUNDCNT_H = 0x0800;
}

static void audio_set_muted(int muted)
{
    /* Keep Timer 0 and DMA running; only change Direct Sound A routing. */
    REG_SOUNDCNT_H = muted ? 0x0004 : 0x0304;
}

static void audio_start_at(const u8 *audio, u32 byte_offset, int paused, int muted)
{
    audio_stop();
    REG_SOUNDCNT_X = 0x0080;
    REG_SOUNDCNT_L = 0;
    REG_SOUNDBIAS = 0x0200;
    REG_SOUNDCNT_H = muted ? 0x0804 : 0x0B04;

    REG_DMA1SAD = (u32)(audio + (byte_offset & ~3u));
    REG_DMA1DAD = (u32)&REG_FIFO_A;
    REG_DMA1CNT_L = 4;
    REG_DMA1CNT_H = 0xB640;

    if (!paused) {
        REG_TM0CNT_L = 0xFC00;
        REG_TM0CNT_H = 0x0080;
    }
}

static void audio_pause(void)
{
    REG_TM0CNT_H = 0;
}

static void audio_resume(void)
{
    REG_TM0CNT_L = 0xFC00;
    REG_TM0CNT_H = 0x0080;
}

static int tick_ui_timers(struct PlayerUI *ui)
{
    int changed = 0;
    if (ui->hud_timer != 0u) {
        --ui->hud_timer;
        if (ui->hud_timer == 0u) changed = 1;
    }
    if (ui->mute_timer != 0u) {
        --ui->mute_timer;
        if (ui->mute_timer == 0u) changed = 1;
    }
    if (ui->seek_timer != 0u) {
        --ui->seek_timer;
        if (ui->seek_timer == 0u) {
            ui->seek_direction = 0;
            changed = 1;
        }
    }
    return changed;
}

static int wait_frame_period(u16 *previous_keys, u16 vblanks, int has_audio, int *paused,
                             u32 *elapsed, struct PlayerUI *ui)
{
    while (*elapsed < (u32)vblanks) {
        u16 now;
        u16 pressed;
        int timer_changed;
        wait_vblank();
        timer_changed = tick_ui_timers(ui);
        now = keys_down();
        pressed = (u16)(now & (u16)~(*previous_keys));
        *previous_keys = now;

        if (pressed & (KEY_START | KEY_B)) {
            return ACTION_RESTART;
        }
        if (pressed & (KEY_L | KEY_LEFT)) {
            return ACTION_SEEK_BACK;
        }
        if (pressed & (KEY_R | KEY_RIGHT)) {
            return ACTION_SEEK_FORWARD;
        }
        if ((pressed & KEY_SELECT) && has_audio) {
            ui->muted = !ui->muted;
            ui->mute_timer = HUD_HOLD_VBLANKS;
            audio_set_muted(ui->muted);
            return ACTION_UI_REFRESH;
        }
        if (pressed & KEY_UP) {
            ui->hud_pinned = !ui->hud_pinned;
            ui->hud_timer = 0u;
            return ACTION_UI_REFRESH;
        }
        if (pressed & KEY_A) {
            *paused = !*paused;
            ui->hud_timer = HUD_HOLD_VBLANKS;
            if (has_audio) {
                if (*paused) audio_pause();
                else audio_resume();
            }
            return ACTION_UI_REFRESH;
        }
        if (*paused && timer_changed) {
            return ACTION_UI_REFRESH;
        }
        if (!*paused) {
            ++(*elapsed);
        }
    }
    return ACTION_NONE;
}

static u32 seek_target(u32 frame, u32 frame_count, u32 step, int forward)
{
    if (step == 0u) step = 50u;
    if (forward) {
        if (frame >= frame_count - 1u || step >= frame_count - frame) {
            return frame_count - 1u;
        }
        return frame + step;
    }
    if (frame <= step) {
        return 0u;
    }
    return frame - step;
}

static u32 audio_offset_for_frame(const struct VideoMetadata *meta, const u32 *seek_table, u32 frame)
{
    u32 offset;
    if (seek_table == (const u32 *)0 || frame >= meta->frame_count) {
        return 0u;
    }
    offset = seek_table[frame] & ~3u;
    if (meta->audio_size < 4u) {
        return 0u;
    }
    if (offset > meta->audio_size - 4u) {
        offset = (meta->audio_size - 4u) & ~3u;
    }
    return offset;
}

static void start_seek_feedback(struct PlayerUI *ui, int direction)
{
    ui->seek_direction = direction;
    ui->seek_timer = SEEK_HOLD_VBLANKS;
    ui->hud_timer = HUD_HOLD_VBLANKS;
}

void main(void)
{
    const struct VideoMetadata *meta = &gba_video_metadata;
    const u16 *palette;
    const u8 *video;
    const u8 *audio;
    const u32 *seek_table;
    u16 vblanks;
    int has_audio;
    struct PlayerUI ui;

    REG_IME = 0;
    REG_WAITCNT = 0x4317;
    REG_DISPCNT = FORCE_BLANK;

    if (meta->magic != GBV4_MAGIC || meta->version != 4u || meta->frame_count == 0u ||
        meta->frame_bytes != FRAME_BYTES || meta->source_width != FRAME_WIDTH ||
        meta->source_height != FRAME_HEIGHT) {
        for (;;) { }
    }

    palette = (const u16 *)rom_ptr(meta->palette_offset);
    video = rom_ptr(meta->video_offset);
    audio = rom_ptr(meta->audio_offset);
    seek_table = meta->seek_table_offset ? (const u32 *)rom_ptr(meta->seek_table_offset) : (const u32 *)0;
    vblanks = meta->vblanks_per_frame;
    if (vblanks == 0u) vblanks = 6u;
    has_audio = (meta->flags & FLAG_AUDIO) != 0u && meta->audio_size != 0u && seek_table != (const u32 *)0;

    REG_BG2PA = 0x0100;
    REG_BG2PB = 0;
    REG_BG2PC = 0;
    REG_BG2PD = 0x0100;
    REG_BG2X = 0;
    REG_BG2Y = 0;

    copy_palette(palette);
    ui.muted = 0;
    ui.hud_pinned = 0;
    ui.hud_timer = 0u;
    ui.mute_timer = 0u;
    ui.seek_timer = 0u;
    ui.seek_direction = 0;

    for (;;) {
        u32 frame = 0u;
        u32 frame_elapsed = 0u;
        u16 displayed_page = 0u;
        u16 previous_keys;
        int paused = 0;
        int restart = 0;
        int at_end = 0;

        audio_stop();
        REG_DISPCNT = FORCE_BLANK;
        render_frame_with_ui(video, 0u, VRAM_PAGE0, meta, &ui, paused, has_audio);
        wait_vblank();
        REG_DISPCNT = MODE4_BG2;
        previous_keys = keys_down();
        if (has_audio) {
            audio_start_at(audio, audio_offset_for_frame(meta, seek_table, 0u), 0, ui.muted);
        }

        while (!restart) {
            if (at_end) {
                u16 now;
                u16 pressed;
                int redraw = 0;
                wait_vblank();
                if (tick_ui_timers(&ui)) redraw = 1;
                now = keys_down();
                pressed = (u16)(now & (u16)~previous_keys);
                previous_keys = now;

                if (pressed & (KEY_A | KEY_B | KEY_START)) {
                    restart = 1;
                } else if (pressed & (KEY_L | KEY_LEFT)) {
                    u32 target = seek_target(frame, meta->frame_count, meta->seek_frame_step, 0);
                    if (target != frame) {
                        start_seek_feedback(&ui, -1);
                        render_and_show(video, target, &displayed_page, meta, &ui, 0, has_audio);
                        previous_keys = keys_down();
                        frame = target;
                        frame_elapsed = 0u;
                        at_end = 0;
                        paused = 0;
                        if (has_audio) {
                            audio_start_at(audio, audio_offset_for_frame(meta, seek_table, frame), 0, ui.muted);
                        }
                    }
                } else if ((pressed & KEY_SELECT) && has_audio) {
                    ui.muted = !ui.muted;
                    ui.mute_timer = HUD_HOLD_VBLANKS;
                    redraw = 1;
                } else if (pressed & KEY_UP) {
                    ui.hud_pinned = !ui.hud_pinned;
                    ui.hud_timer = 0u;
                    redraw = 1;
                }
                if (redraw && !restart && at_end) {
                    render_and_show(video, frame, &displayed_page, meta, &ui, 0, has_audio);
                    previous_keys = keys_down();
                }
                continue;
            }

            {
                int has_next = frame + 1u < meta->frame_count;
                volatile u16 *back = displayed_page ? VRAM_PAGE0 : VRAM_PAGE1;
                int action;

                if (has_next) {
                    render_frame_with_ui(video, frame + 1u, back, meta, &ui, paused, has_audio);
                }

                action = wait_frame_period(&previous_keys, vblanks, has_audio, &paused, &frame_elapsed, &ui);
                if (action == ACTION_RESTART) {
                    restart = 1;
                    continue;
                }
                if (action == ACTION_UI_REFRESH) {
                    render_and_show(video, frame, &displayed_page, meta, &ui, paused, has_audio);
                    previous_keys = keys_down();
                    continue;
                }
                if (action == ACTION_SEEK_BACK || action == ACTION_SEEK_FORWARD) {
                    u32 target = seek_target(frame, meta->frame_count, meta->seek_frame_step,
                                             action == ACTION_SEEK_FORWARD);
                    if (target != frame) {
                        if (has_audio) audio_stop();
                        start_seek_feedback(&ui, action == ACTION_SEEK_FORWARD ? 1 : -1);
                        render_and_show(video, target, &displayed_page, meta, &ui, paused, has_audio);
                        previous_keys = keys_down();
                        frame = target;
                        frame_elapsed = 0u;
                        if (has_audio) {
                            audio_start_at(audio, audio_offset_for_frame(meta, seek_table, frame), paused, ui.muted);
                        }
                    }
                    continue;
                }

                if (has_next) {
                    show_rendered_page(&displayed_page);
                    ++frame;
                    frame_elapsed = 0u;
                } else {
                    audio_stop();
                    if (meta->flags & FLAG_LOOP) {
                        restart = 1;
                    } else {
                        at_end = 1;
                    }
                }
            }
        }
    }
}

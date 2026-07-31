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
#define KEY_START        0x0008
#define KEY_R            0x0100
#define KEY_L            0x0200

#define FLAG_AUDIO       0x0001u
#define FLAG_LOOP        0x0002u
#define GBV3_MAGIC       0x33564247u

#define FRAME_WIDTH      120u
#define FRAME_HEIGHT     80u
#define FRAME_BYTES      9600u

#define ACTION_NONE          0
#define ACTION_RESTART       1
#define ACTION_SEEK_BACK     2
#define ACTION_SEEK_FORWARD  3

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

static void show_rendered_page(u16 *displayed_page)
{
    *displayed_page ^= 1u;
    REG_DISPCNT = *displayed_page ? (MODE4_BG2 | PAGE_SELECT) : MODE4_BG2;
}

static void render_and_show(const u8 *video, u32 frame_number, u16 *displayed_page)
{
    volatile u16 *back = *displayed_page ? VRAM_PAGE0 : VRAM_PAGE1;
    render_frame(video, frame_number, back);
    wait_vblank();
    show_rendered_page(displayed_page);
}

static void audio_stop(void)
{
    REG_TM0CNT_H = 0;
    REG_DMA1CNT_H = 0;
    REG_SOUNDCNT_H = 0x0800;
}

static void audio_start_at(const u8 *audio, u32 byte_offset, int paused)
{
    audio_stop();
    REG_SOUNDCNT_X = 0x0080;
    REG_SOUNDCNT_L = 0;
    REG_SOUNDBIAS = 0x0200;
    REG_SOUNDCNT_H = 0x0B04;

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

static int wait_frame_period(u16 *previous_keys, u16 vblanks, int has_audio, int *paused)
{
    u32 elapsed = 0;

    while (elapsed < (u32)vblanks) {
        u16 now;
        u16 pressed;
        wait_vblank();
        now = keys_down();
        pressed = (u16)(now & (u16)~(*previous_keys));
        *previous_keys = now;

        if (pressed & (KEY_START | KEY_B)) {
            return ACTION_RESTART;
        }
        if (pressed & KEY_L) {
            return ACTION_SEEK_BACK;
        }
        if (pressed & KEY_R) {
            return ACTION_SEEK_FORWARD;
        }
        if (pressed & KEY_A) {
            *paused = !*paused;
            if (has_audio) {
                if (*paused) audio_pause();
                else audio_resume();
            }
        }
        if (!*paused) {
            ++elapsed;
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

void main(void)
{
    const struct VideoMetadata *meta = &gba_video_metadata;
    const u16 *palette;
    const u8 *video;
    const u8 *audio;
    const u32 *seek_table;
    u16 vblanks;
    int has_audio;

    REG_IME = 0;
    REG_WAITCNT = 0x4317;
    REG_DISPCNT = FORCE_BLANK;

    if (meta->magic != GBV3_MAGIC || meta->version != 3u || meta->frame_count == 0u ||
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

    for (;;) {
        u32 frame = 0u;
        u16 displayed_page = 0u;
        u16 previous_keys;
        int paused = 0;
        int restart = 0;
        int at_end = 0;

        audio_stop();
        REG_DISPCNT = FORCE_BLANK;
        render_frame(video, 0u, VRAM_PAGE0);
        wait_vblank();
        REG_DISPCNT = MODE4_BG2;
        previous_keys = keys_down();
        if (has_audio) {
            audio_start_at(audio, audio_offset_for_frame(meta, seek_table, 0u), 0);
        }

        while (!restart) {
            if (at_end) {
                u16 now;
                u16 pressed;
                wait_vblank();
                now = keys_down();
                pressed = (u16)(now & (u16)~previous_keys);
                previous_keys = now;

                if (pressed & (KEY_A | KEY_B | KEY_START)) {
                    restart = 1;
                } else if (pressed & KEY_L) {
                    u32 target = seek_target(frame, meta->frame_count, meta->seek_frame_step, 0);
                    if (target != frame) {
                        render_and_show(video, target, &displayed_page);
                        previous_keys = keys_down();
                        frame = target;
                        at_end = 0;
                        paused = 0;
                        if (has_audio) {
                            audio_start_at(audio, audio_offset_for_frame(meta, seek_table, frame), 0);
                        }
                    }
                }
                continue;
            }

            {
                int has_next = frame + 1u < meta->frame_count;
                volatile u16 *back = displayed_page ? VRAM_PAGE0 : VRAM_PAGE1;
                int action;

                if (has_next) {
                    render_frame(video, frame + 1u, back);
                }

                action = wait_frame_period(&previous_keys, vblanks, has_audio, &paused);
                if (action == ACTION_RESTART) {
                    restart = 1;
                    continue;
                }
                if (action == ACTION_SEEK_BACK || action == ACTION_SEEK_FORWARD) {
                    u32 target = seek_target(frame, meta->frame_count, meta->seek_frame_step,
                                             action == ACTION_SEEK_FORWARD);
                    if (target != frame) {
                        if (has_audio) audio_stop();
                        render_and_show(video, target, &displayed_page);
                        previous_keys = keys_down();
                        frame = target;
                        if (has_audio) {
                            audio_start_at(audio, audio_offset_for_frame(meta, seek_table, frame), paused);
                        }
                    }
                    continue;
                }

                if (has_next) {
                    show_rendered_page(&displayed_page);
                    ++frame;
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

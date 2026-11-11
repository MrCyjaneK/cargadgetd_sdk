#ifndef CARGADGETD_CG_APP_H
#define CARGADGETD_CG_APP_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/*
 * Plugin ABI: each .so exports C symbols named
 *
 *   cargadgetd_app_<id>_<method>
 *
 * <id> is a C identifier (the app id). The host dlsyms these after
 * discovering <id> from the .so (dynamic symbol *_draw, else the <id> in
 * the filename). When both are present they must be the same id.
 *
 * Library name must be `libcargadgetd_app_<id>.so`, installed in
 * `/usr/lib/cargadgetd/plugins/`. Other names (music.so, reversegearhead.so)
 * are not apps. The helloworld package's id is `hello`.
 *
 *   void        cargadgetd_app_<id>_init(const struct cargadgetd_host *host);
 *   void        cargadgetd_app_<id>_draw(uint32_t *rgba, int w, int h, int stride_bytes);
 *   void        cargadgetd_app_<id>_touch(int x, int y, int action); // 0 up, 1 down, 2 move
 *   void        cargadgetd_app_<id>_pause(void);
 *   void        cargadgetd_app_<id>_resume(void);
 *   void        cargadgetd_app_<id>_stop(void); // optional; host Close / plugin unload
 *   const char *cargadgetd_app_<id>_name(void);
 *   const uint8_t *cargadgetd_app_<id>_icon(int *w, int *h); // packed RGBA8; C.malloc (host copies)
 *   void        cargadgetd_app_<id>_huinput_<event>(int pressed); // optional; 1 down, 0 up
 *   void        cargadgetd_app_<id>_gps_fix(int has_pos, double lat, double lon,
 *                   double acc_m, int has_speed, int kmh, int has_course,
 *                   double course, const char *err); // optional
 *   void        cargadgetd_app_<id>_gps_sat(int gain, const char *kicker,
 *                   const char *title, const char *sub, const char *detail,
 *                   const char *flag); // optional; one lock or drop
 *
 * HU keys are broadcast to every loaded app (opened or not). <event> names:
 *   mic menu home back call endcall key0..key9 star pound
 *   dpad_up dpad_down dpad_left dpad_right dpad_click plus search
 *   playpause stop next prev play pause
 *   dpad_up_left dpad_down_left dpad_up_right dpad_down_right
 *   rotary media navigation tel
 * Unknown scan codes are not called; they go to apps.unhandled_log.
 *
 * Host (passed to init):
 *   play      — optional PCM mix into the shell
 *   frame_ms  — paint + video interval; 0 → 33 (same knob as video.frame_ms)
 *
 * Optional settings. Missing settings_count means the app has none. A count
 * above 0 means the other settings symbols are present. Indexes are that
 * plugin's slots; keep the order stable. The plugin stores values however
 * it wants. set_* returns 0, or nonzero. On nonzero, settings_error() is
 * the message until the next settings call on this plugin, and the stored
 * value is unchanged. Returned strings stay valid until that next call.
 * set_bool takes 0 or 1. Enum tokens are comma-separated and contain no
 * comma. Type is 0 bool, 1 enum, 2 int, 3 string. String get/set are
 * optional; a row of type 3 is skipped when they are missing. The host
 * loader does not dlsym these; a settings UI does. They do not need init
 * or cargadgetd_host.
 * Shells use the same suffixes on cargadgetd_shell_<id>_settings_*.
 *
 *   int         cargadgetd_app_<id>_settings_count(void);
 *   const char *cargadgetd_app_<id>_settings_label(int i);
 *   int         cargadgetd_app_<id>_settings_type(int i);
 *   const char *cargadgetd_app_<id>_settings_enum_values(int i);
 *   int         cargadgetd_app_<id>_settings_get_bool(int i);
 *   const char *cargadgetd_app_<id>_settings_get_enum(int i);
 *   int         cargadgetd_app_<id>_settings_get_int(int i);
 *   int         cargadgetd_app_<id>_settings_set_bool(int i, int v);
 *   int         cargadgetd_app_<id>_settings_set_enum(int i, const char *v);
 *   int         cargadgetd_app_<id>_settings_set_int(int i, int v);
 *   const char *cargadgetd_app_<id>_settings_get_string(int i); // optional
 *   int         cargadgetd_app_<id>_settings_set_string(int i, const char *v);
 *   const char *cargadgetd_app_<id>_settings_error(void);
 *
 * Required: draw. All others optional. name/icon may return NULL.
 */

struct cargadgetd_host {
	void (*play)(const int16_t *pcm, int nframes, int channels, int sample_rate);
	int frame_ms; /* paint + video interval; 0 → 33 */
};

#ifdef CARGADGETD_APP_IMPLEMENT_PLAY
static void cargadgetd_host_play(const struct cargadgetd_host *h,
				 const int16_t *pcm, int nframes, int channels, int sample_rate)
{
	if (h && h->play)
		h->play(pcm, nframes, channels, sample_rate);
}

static int cargadgetd_host_frame_ms(const struct cargadgetd_host *h)
{
	int ms;

	if (!h || h->frame_ms <= 0)
		return 33;
	ms = h->frame_ms;
	if (ms < 16)
		return 16;
	return ms;
}
#endif

#ifdef __cplusplus
}
#endif

#endif

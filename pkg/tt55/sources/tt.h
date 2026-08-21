#ifndef TT55_H
#define TT55_H

#include <stddef.h>
#include <stdint.h>

enum {
	TT55_OK = 0,
	TT55_ERR_ARG = -1,
	TT55_ERR_FMT = -2,
	TT55_ERR_NOCMAP = -3,
	TT55_ERR_NOHMTX = -4
};

typedef struct {
	const uint8_t *data;
	uint64_t len;
	uint32_t cmap_off;
	uint32_t cmap_len;
	uint32_t cmap_sub;
	uint32_t hmtx_off;
	uint32_t hmtx_len;
	uint16_t num_glyphs;
	uint16_t number_of_h_metrics;
	uint16_t units_per_em;
} Tt55_font;

int32_t tt55_open(const uint8_t *data, uint64_t len, Tt55_font *font);
int32_t tt55_glyph(const Tt55_font *font, uint32_t cp, uint16_t *gid);
int32_t tt55_advance(const Tt55_font *font, uint16_t gid, uint16_t *aw);

#endif

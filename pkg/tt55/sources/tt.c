#include "tt.h"

#define TAG_CMAP 0x636D6170u
#define TAG_HEAD 0x68656164u
#define TAG_HHEA 0x68686561u
#define TAG_HMTX 0x686D7478u
#define TAG_MAXP 0x6D617870u
#define HEAD_MAGIC 0x5F0F3CF5u
#define MAX_SFNT_TABLES 64

static int32_t need(const Tt55_font *f, uint32_t off, uint32_t n)
{
	if (f->data == 0) {
		return TT55_ERR_ARG;
	}
	if ((uint64_t)off + (uint64_t)n > f->len) {
		return TT55_ERR_FMT;
	}
	return TT55_OK;
}

static uint16_t be16(const uint8_t *p)
{
	return (uint16_t)(((uint16_t)p[0] << 8) | (uint16_t)p[1]);
}

static uint32_t be32(const uint8_t *p)
{
	return ((uint32_t)p[0] << 24) | ((uint32_t)p[1] << 16) | ((uint32_t)p[2] << 8) | (uint32_t)p[3];
}

static int32_t read16(const Tt55_font *f, uint32_t off, uint16_t *out)
{
	int32_t e = need(f, off, 2);
	if (e != TT55_OK) {
		return e;
	}
	*out = be16(f->data + off);
	return TT55_OK;
}

static int32_t read32(const Tt55_font *f, uint32_t off, uint32_t *out)
{
	int32_t e = need(f, off, 4);
	if (e != TT55_OK) {
		return e;
	}
	*out = be32(f->data + off);
	return TT55_OK;
}

static int32_t find_table(const Tt55_font *f, uint16_t ntab, uint32_t tag, uint32_t *off, uint32_t *tlen)
{
	uint16_t i;
	for (i = 0; i < ntab; i++) {
		uint32_t rec = 12u + (uint32_t)i * 16u;
		uint32_t t;
		int32_t e = read32(f, rec, &t);
		if (e != TT55_OK) {
			return e;
		}
		if (t == tag) {
			e = read32(f, rec + 8, off);
			if (e != TT55_OK) {
				return e;
			}
			e = read32(f, rec + 12, tlen);
			if (e != TT55_OK) {
				return e;
			}
			return TT55_OK;
		}
	}
	return TT55_ERR_FMT;
}

static int32_t pick_cmap(const Tt55_font *f, uint32_t *sub_off)
{
	uint16_t nenc;
	uint16_t i;
	uint32_t best = 0;
	int32_t have = 0;
	int32_t e = read16(f, f->cmap_off + 2, &nenc);
	if (e != TT55_OK) {
		return e;
	}
	for (i = 0; i < nenc; i++) {
		uint32_t rec = f->cmap_off + 4u + (uint32_t)i * 8u;
		uint16_t plat;
		uint16_t enc;
		uint32_t rel;
		uint32_t abs_off;
		uint16_t fmt;
		e = read16(f, rec, &plat);
		if (e != TT55_OK) {
			return e;
		}
		e = read16(f, rec + 2, &enc);
		if (e != TT55_OK) {
			return e;
		}
		e = read32(f, rec + 4, &rel);
		if (e != TT55_OK) {
			return e;
		}
		abs_off = f->cmap_off + rel;
		e = read16(f, abs_off, &fmt);
		if (e != TT55_OK) {
			return e;
		}
		if (fmt != 4 && fmt != 12) {
			continue;
		}
		if (plat == 3 && enc == 10 && fmt == 12) {
			*sub_off = abs_off;
			return TT55_OK;
		}
		if (plat == 3 && enc == 1 && fmt == 4) {
			best = abs_off;
			have = 1;
		} else if (plat == 0 && have == 0) {
			best = abs_off;
			have = 1;
		}
	}
	if (have == 0) {
		return TT55_ERR_NOCMAP;
	}
	*sub_off = best;
	return TT55_OK;
}

int32_t tt55_open(const uint8_t *data, uint64_t len, Tt55_font *font)
{
	uint32_t scaler;
	uint16_t ntab;
	uint32_t head_off;
	uint32_t head_len;
	uint32_t hhea_off;
	uint32_t hhea_len;
	uint32_t maxp_off;
	uint32_t maxp_len;
	uint32_t magic;
	uint32_t sub;
	int32_t e;
	if (data == 0 || font == 0 || len < 12) {
		return TT55_ERR_ARG;
	}
	font->data = data;
	font->len = len;
	e = read32(font, 0, &scaler);
	if (e != TT55_OK) {
		return e;
	}
	if (scaler != 0x00010000u && scaler != 0x74727565u && scaler != 0x4F54544Fu) {
		return TT55_ERR_FMT;
	}
	e = read16(font, 4, &ntab);
	if (e != TT55_OK) {
		return e;
	}
	if (ntab == 0 || ntab > MAX_SFNT_TABLES) {
		return TT55_ERR_FMT;
	}
	e = find_table(font, ntab, TAG_CMAP, &font->cmap_off, &font->cmap_len);
	if (e != TT55_OK) {
		return TT55_ERR_NOCMAP;
	}
	e = find_table(font, ntab, TAG_HEAD, &head_off, &head_len);
	if (e != TT55_OK || head_len < 54) {
		return TT55_ERR_FMT;
	}
	e = find_table(font, ntab, TAG_HHEA, &hhea_off, &hhea_len);
	if (e != TT55_OK || hhea_len < 36) {
		return TT55_ERR_NOHMTX;
	}
	e = find_table(font, ntab, TAG_HMTX, &font->hmtx_off, &font->hmtx_len);
	if (e != TT55_OK) {
		return TT55_ERR_NOHMTX;
	}
	e = find_table(font, ntab, TAG_MAXP, &maxp_off, &maxp_len);
	if (e != TT55_OK || maxp_len < 6) {
		return TT55_ERR_FMT;
	}
	e = read32(font, head_off + 12, &magic);
	if (e != TT55_OK || magic != HEAD_MAGIC) {
		return TT55_ERR_FMT;
	}
	e = read16(font, head_off + 18, &font->units_per_em);
	if (e != TT55_OK) {
		return e;
	}
	e = read16(font, maxp_off + 4, &font->num_glyphs);
	if (e != TT55_OK) {
		return e;
	}
	e = read16(font, hhea_off + 34, &font->number_of_h_metrics);
	if (e != TT55_OK) {
		return e;
	}
	if (font->number_of_h_metrics == 0) {
		return TT55_ERR_NOHMTX;
	}
	e = pick_cmap(font, &sub);
	if (e != TT55_OK) {
		return e;
	}
	font->cmap_sub = sub;
	return TT55_OK;
}

static int32_t glyph_fmt4(const Tt55_font *f, uint32_t cp, uint16_t *gid)
{
	uint16_t segx2;
	uint16_t seg;
	uint16_t i;
	uint32_t sub = f->cmap_sub;
	int32_t e;
	if (cp > 0xFFFFu) {
		*gid = 0;
		return TT55_OK;
	}
	e = read16(f, sub + 6, &segx2);
	if (e != TT55_OK) {
		return e;
	}
	seg = (uint16_t)(segx2 / 2);
	if (seg == 0) {
		return TT55_ERR_FMT;
	}
	for (i = 0; i < seg; i++) {
		uint16_t endc;
		uint16_t startc;
		uint16_t delta;
		uint16_t iro;
		e = read16(f, sub + 14u + (uint32_t)i * 2u, &endc);
		if (e != TT55_OK) {
			return e;
		}
		e = read16(f, sub + 16u + (uint32_t)seg * 2u + (uint32_t)i * 2u, &startc);
		if (e != TT55_OK) {
			return e;
		}
		if (cp < startc || cp > endc) {
			continue;
		}
		e = read16(f, sub + 16u + (uint32_t)seg * 4u + (uint32_t)i * 2u, &delta);
		if (e != TT55_OK) {
			return e;
		}
		e = read16(f, sub + 16u + (uint32_t)seg * 6u + (uint32_t)i * 2u, &iro);
		if (e != TT55_OK) {
			return e;
		}
		if (iro == 0) {
			*gid = (uint16_t)((uint16_t)cp + delta);
			return TT55_OK;
		}
		{
			uint32_t iro_addr = sub + 16u + (uint32_t)seg * 6u + (uint32_t)i * 2u;
			uint32_t gpos = iro_addr + (uint32_t)iro + 2u * ((uint32_t)cp - (uint32_t)startc);
			uint16_t g;
			e = read16(f, gpos, &g);
			if (e != TT55_OK) {
				return e;
			}
			if (g != 0) {
				g = (uint16_t)(g + delta);
			}
			*gid = g;
			return TT55_OK;
		}
	}
	*gid = 0;
	return TT55_OK;
}

static int32_t glyph_fmt12(const Tt55_font *f, uint32_t cp, uint16_t *gid)
{
	uint32_t ng;
	uint32_t i;
	uint32_t sub = f->cmap_sub;
	int32_t e = read32(f, sub + 12, &ng);
	if (e != TT55_OK) {
		return e;
	}
	for (i = 0; i < ng; i++) {
		uint32_t rec = sub + 16u + i * 12u;
		uint32_t start;
		uint32_t end;
		uint32_t sgid;
		e = read32(f, rec, &start);
		if (e != TT55_OK) {
			return e;
		}
		e = read32(f, rec + 4, &end);
		if (e != TT55_OK) {
			return e;
		}
		if (cp < start || cp > end) {
			continue;
		}
		e = read32(f, rec + 8, &sgid);
		if (e != TT55_OK) {
			return e;
		}
		{
			uint32_t g = sgid + (cp - start);
			if (g > 0xFFFFu) {
				return TT55_ERR_FMT;
			}
			*gid = (uint16_t)g;
			return TT55_OK;
		}
	}
	*gid = 0;
	return TT55_OK;
}

int32_t tt55_glyph(const Tt55_font *font, uint32_t cp, uint16_t *gid)
{
	uint16_t fmt;
	int32_t e;
	if (font == 0 || gid == 0 || font->data == 0) {
		return TT55_ERR_ARG;
	}
	e = read16(font, font->cmap_sub, &fmt);
	if (e != TT55_OK) {
		return e;
	}
	if (fmt == 4) {
		return glyph_fmt4(font, cp, gid);
	}
	if (fmt == 12) {
		return glyph_fmt12(font, cp, gid);
	}
	return TT55_ERR_NOCMAP;
}

int32_t tt55_advance(const Tt55_font *font, uint16_t gid, uint16_t *aw)
{
	uint32_t idx;
	int32_t e;
	if (font == 0 || aw == 0 || font->data == 0) {
		return TT55_ERR_ARG;
	}
	if (gid >= font->num_glyphs) {
		return TT55_ERR_FMT;
	}
	idx = gid;
	if (idx >= (uint32_t)font->number_of_h_metrics) {
		idx = (uint32_t)font->number_of_h_metrics - 1u;
	}
	e = read16(font, font->hmtx_off + idx * 4u, aw);
	if (e != TT55_OK) {
		return e;
	}
	return TT55_OK;
}

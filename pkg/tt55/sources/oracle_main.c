#include "tt.h"
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>

int main(int argc, char **argv) {
	if (argc < 2) {
		fprintf(stderr, "Usage: %s <font.ttf>\n", argv[0]);
		return 1;
	}

	FILE *f = fopen(argv[1], "rb");
	if (!f) {
		perror("fopen");
		return 1;
	}

	fseek(f, 0, SEEK_END);
	long sz = ftell(f);
	fseek(f, 0, SEEK_SET);

	uint8_t *buf = (uint8_t *)malloc(sz);
	if (!buf) {
		fclose(f);
		return 1;
	}

	if (fread(buf, 1, sz, f) != (size_t)sz) {
		free(buf);
		fclose(f);
		return 1;
	}
	fclose(f);

	Tt55_font font;
	int32_t err = tt55_open(buf, (uint64_t)sz, &font);
	printf("OPEN:%d units_per_em:%u num_glyphs:%u num_hmetrics:%u cmap_off:%u cmap_len:%u cmap_sub:%u hmtx_off:%u hmtx_len:%u\n",
		err, font.units_per_em, font.num_glyphs, font.number_of_h_metrics,
		font.cmap_off, font.cmap_len, font.cmap_sub, font.hmtx_off, font.hmtx_len);

	if (err != 0) {
		free(buf);
		return 0;
	}

	// Échantillonnage de codepoints (ASCII, Latin-1, Grec, Cyrillique, Math, CJK)
	uint32_t test_cps[] = {
		32, 65, 97, 126, 160, 233, 246, 0x03B1, 0x03C9, 0x0410, 0x044F,
		0x20AC, 0x221E, 0x4E00, 0x9FA5, 0x1F600, 0x1F680, 0xFFFF, 0x10000
	};
	size_t n_cps = sizeof(test_cps) / sizeof(test_cps[0]);

	for (size_t i = 0; i < n_cps; i++) {
		uint32_t cp = test_cps[i];
		uint16_t gid = 0;
		int32_t gerr = tt55_glyph(&font, cp, &gid);
		uint16_t aw = 0;
		int32_t aerr = -999;
		if (gerr == 0) {
			aerr = tt55_advance(&font, gid, &aw);
		}
		printf("CP:%u GID:%u GERR:%d AW:%u AERR:%d\n", cp, gid, gerr, aw, aerr);
	}

	free(buf);
	return 0;
}

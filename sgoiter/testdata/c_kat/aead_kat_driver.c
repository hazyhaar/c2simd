/* Driver KAT monocypher C (gcc -O0).
 * Build depuis c2simd :
 *   gcc -O0 -I spec/c_sources/upstream/monocypher/4.0.2 \
 *       -o /tmp/aead_kat_driver \
 *       sgoiter/testdata/c_kat/aead_kat_driver.c \
 *       spec/c_sources/upstream/monocypher/4.0.2/monocypher.c
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include "monocypher.h"

static void put_hex(const uint8_t *p, size_t n) {
	for (size_t i = 0; i < n; i++) printf("%02x", p[i]);
}

static void fill_key_nonce(uint8_t key[32], uint8_t nonce[24]) {
	for (int i = 0; i < 32; i++) key[i] = (uint8_t)(i + 1);
	for (int i = 0; i < 24; i++) nonce[i] = (uint8_t)(i + 10);
}

static void run_case(const char *name, const uint8_t *ad, size_t ad_size,
		     const uint8_t *pt, size_t pt_size) {
	uint8_t key[32], nonce[24];
	fill_key_nonce(key, nonce);
	uint8_t *ct = (uint8_t *)calloc(1, pt_size ? pt_size : 1);
	uint8_t mac[16];
	crypto_aead_lock(ct, mac, key, nonce, ad, ad_size, pt, pt_size);
	printf("NAME=%s\n", name);
	printf("CT=");
	put_hex(ct, pt_size);
	printf("\nMAC=");
	put_hex(mac, 16);
	printf("\n");
	free(ct);
}

int main(int argc, char **argv) {
	const char *which = argc > 1 ? argv[1] : "all";

	if (!strcmp(which, "pt36_header") || !strcmp(which, "all")) {
		const char *ad = "HEADER";
		const char *pt = "HELLO MONOCYPHER SGOITER AEAD CGO=0!";
		run_case("pt36_header", (const uint8_t *)ad, strlen(ad),
			 (const uint8_t *)pt, strlen(pt));
	}
	if (!strcmp(which, "pt0_ad_empty") || !strcmp(which, "all")) {
		run_case("pt0_ad_empty", NULL, 0, NULL, 0);
	}
	if (!strcmp(which, "pt1_ad_empty") || !strcmp(which, "all")) {
		uint8_t pt[1] = {0x41};
		run_case("pt1_ad_empty", NULL, 0, pt, 1);
	}
	if (!strcmp(which, "pt64_ad_empty") || !strcmp(which, "all")) {
		uint8_t pt[64];
		for (int i = 0; i < 64; i++) pt[i] = (uint8_t)(i % 251);
		run_case("pt64_ad_empty", NULL, 0, pt, 64);
	}
	if (!strcmp(which, "pt65_ad_empty") || !strcmp(which, "all")) {
		uint8_t pt[65];
		for (int i = 0; i < 65; i++) pt[i] = (uint8_t)(i % 251);
		run_case("pt65_ad_empty", NULL, 0, pt, 65);
	}
	if (!strcmp(which, "pt129_ad_empty") || !strcmp(which, "all")) {
		uint8_t pt[129];
		for (int i = 0; i < 129; i++) pt[i] = (uint8_t)(i % 251);
		run_case("pt129_ad_empty", NULL, 0, pt, 129);
	}
	if (!strcmp(which, "pt193_ad_empty") || !strcmp(which, "all")) {
		uint8_t pt[193];
		for (int i = 0; i < 193; i++) pt[i] = (uint8_t)(i % 251);
		run_case("pt193_ad_empty", NULL, 0, pt, 193);
	}
	if (!strcmp(which, "pt1024_header1kb") || !strcmp(which, "all")) {
		const char *ad = "HEADER 1KB";
		uint8_t pt[1024];
		for (int i = 0; i < 1024; i++) pt[i] = (uint8_t)((i * 17 + 3) % 251);
		run_case("pt1024_header1kb", (const uint8_t *)ad, strlen(ad), pt, 1024);
	}
	if (!strcmp(which, "pt4096_ad_empty") || !strcmp(which, "all")) {
		uint8_t *pt = (uint8_t *)malloc(4096);
		for (int i = 0; i < 4096; i++) pt[i] = (uint8_t)(i % 251);
		run_case("pt4096_ad_empty", NULL, 0, pt, 4096);
		free(pt);
	}
	if (!strcmp(which, "chacha20_ietf") || !strcmp(which, "all")) {
		uint8_t key[32], nonce[12], pt[64], ct[64];
		for (int i = 0; i < 32; i++) key[i] = (uint8_t)(i + 1);
		for (int i = 0; i < 12; i++) nonce[i] = (uint8_t)(i + 5);
		for (int i = 0; i < 64; i++) pt[i] = (uint8_t)(i % 251);
		uint32_t next_ctr = crypto_chacha20_ietf(ct, pt, 64, key, nonce, 0x1000);
		printf("NAME=chacha20_ietf\n");
		printf("CT=");
		put_hex(ct, 64);
		printf("\nCTR=%08x\n", next_ctr);
	}
	return 0;
}

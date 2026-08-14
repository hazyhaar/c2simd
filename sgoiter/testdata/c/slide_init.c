#include <stdint.h>
typedef struct { int16_t next_index; int8_t next_digit; uint8_t next_check; } slide_ctx;
void slide_init(slide_ctx *ctx) {
	ctx->next_check = 1;
	ctx->next_index = -1;
	ctx->next_digit = -1;
}

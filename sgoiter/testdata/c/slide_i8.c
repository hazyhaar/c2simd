#include <stdint.h>
typedef struct { int16_t next_index; int8_t next_digit; uint8_t next_check; } slide_ctx;
int8_t set_digit(slide_ctx *c, int v) {
	c->next_digit = (int8_t)(v >> 1);
	return c->next_digit;
}

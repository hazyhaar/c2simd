/* Driver KAT libinjection_sqli C (gcc -O2).
 * Build depuis c2simd :
 *   gcc -O2 -I spec/c_sources/testdata/c_sources/libinjection \
 *       -o /tmp/libinjection_kat_driver \
 *       sgoiter/testdata/c_kat/libinjection_kat_driver.c \
 *       spec/c_sources/testdata/c_sources/libinjection/libinjection_sqli.c
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "libinjection.h"

static void run_case(const char *name, const char *input) {
	char fp[16] = {0};
	size_t len = strlen(input);
	int is_sqli = libinjection_sqli(input, len, fp);
	printf("NAME=%s\n", name);
	printf("RET=%d\n", is_sqli);
	printf("FP=%s\n", fp);
}

int main(int argc, char **argv) {
	const char *which = argc > 1 ? argv[1] : "all";

	struct {
		const char *name;
		const char *input;
	} cases[] = {
		{"sqli_or_1eq1", "' OR '1'='1"},
		{"sqli_union", "1 UNION SELECT 1,2,3--"},
		{"sqli_comment", "SELECT * FROM users WHERE id=1 -- comment"},
		{"benign_select", "SELECT id, name FROM users WHERE age > 21"},
		{"benign_text", "Hello world, this is normal user input."},
		{"sqli_semicolon", "1; DROP TABLE users;--"},
		{NULL, NULL}
	};

	for (int i = 0; cases[i].name != NULL; i++) {
		if (!strcmp(which, cases[i].name) || !strcmp(which, "all")) {
			run_case(cases[i].name, cases[i].input);
		}
	}
	return 0;
}

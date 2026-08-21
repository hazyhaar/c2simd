#ifndef C2_PAINTER_H
#define C2_PAINTER_H

#include <stdint.h>
#include <stddef.h>

typedef struct {
    int x;
    int y;
    int w;
    int h;
} c2_rect_t;

typedef struct {
    uint32_t *pixels;
    int width;
    int height;
    int stride;
    c2_rect_t clip;
} c2_painter_t;

/* Types de nœuds graphiques linéaires aplatis */
enum {
    C2_NODE_RECT = 1,
    C2_NODE_ROUNDED_RECT = 2,
    C2_NODE_CIRCLE = 3,
    C2_NODE_ELLIPSE = 4,
    C2_NODE_LINE = 5,
    C2_NODE_TEXT_GLYPH = 6,
    C2_NODE_MASK = 7,
    C2_NODE_SHADOW = 8
};

/* Nœud graphique linéaire compact - aligné sur 32 octets (2 par ligne de cache L1 de 64B) */
typedef struct {
    uint16_t type;       /* C2_NODE_* */
    uint16_t flags;      /* 0: normal, 1: stroke */
    int32_t  x;          /* Position X absolue */
    int32_t  y;          /* Position Y absolue */
    int32_t  w;          /* Largeur / X1 pour ligne */
    int32_t  h;          /* Hauteur / Y1 pour ligne */
    uint32_t color0;     /* Couleur primaire RGBA */
    uint32_t color1;     /* Couleur secondaire / contour RGBA */
    int32_t  param0;     /* Rayon, épaisseur de trait, ou taille de police */
    int32_t  param1;     /* Paramètre géométrique secondaire (ex. rayon Y) */
} c2_node_t;

/* Scène graphique linéaire double-bufferisée */
typedef struct {
    c2_node_t *nodes;
    int count;
    int capacity;
} c2_scene_t;

/* Helpers mathématiques & couleurs */
int c2_isqrt(int val);
int64_t c2_isqrt64(int64_t val);
uint32_t c2_rgba(uint8_t r, uint8_t g, uint8_t b, uint8_t a);
uint32_t c2_rgba_premul(uint8_t r, uint8_t g, uint8_t b, uint8_t a);
uint32_t c2_blend_pixel(uint32_t dst, uint32_t src);
uint32_t c2_blend_pixel_cov(uint32_t dst, uint32_t src, uint32_t cov);

/* Initialisation et clipping */
void c2_painter_init(c2_painter_t *p, uint32_t *pixels, int width, int height, int stride);
void c2_painter_set_clip(c2_painter_t *p, int x, int y, int w, int h);
void c2_painter_reset_clip(c2_painter_t *p);
void c2_clear(c2_painter_t *p, uint32_t color);

/* Rectangles */
void c2_fill_rect(c2_painter_t *p, int x, int y, int w, int h, uint32_t color);
void c2_stroke_rect(c2_painter_t *p, int x, int y, int w, int h, int stroke_width, uint32_t color);
void c2_fill_rounded_rect(c2_painter_t *p, int x, int y, int w, int h, int radius, uint32_t color);
void c2_stroke_rounded_rect(c2_painter_t *p, int x, int y, int w, int h, int radius, int stroke_width, uint32_t color);

/* Cercles et Ellipses */
void c2_fill_circle(c2_painter_t *p, int cx, int cy, int radius, uint32_t color);
void c2_stroke_circle(c2_painter_t *p, int cx, int cy, int radius, int stroke_width, uint32_t color);
void c2_fill_ellipse(c2_painter_t *p, int cx, int cy, int rx, int ry, uint32_t color);
void c2_stroke_ellipse(c2_painter_t *p, int cx, int cy, int rx, int ry, int stroke_width, uint32_t color);

/* Lignes antialiasées */
void c2_draw_line(c2_painter_t *p, int x0, int y0, int x1, int y1, int stroke_width, uint32_t color);

/* Blitting masques et images */
void c2_blit_mask(c2_painter_t *p, int dst_x, int dst_y, int w, int h, const uint8_t *mask, int mask_stride, uint32_t color);
void c2_blit_rgba(c2_painter_t *p, int dst_x, int dst_y, int w, int h, const uint32_t *src, int src_stride);

/* Dégradés linéaires et curseurs */
void c2_fill_linear_gradient(c2_painter_t *p, int x, int y, int w, int h, uint32_t color0, uint32_t color1, int vertical);
void c2_draw_cursor(c2_painter_t *p, int x, int y, int w, int h, int style, uint32_t color);

/* Rendu en lot d'un snapshot de scène linéaire contiguë */
void c2_scene_render(c2_painter_t *p, const c2_node_t *nodes, int count);

#endif /* C2_PAINTER_H */

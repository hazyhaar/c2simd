#include "c2_painter.h"

int c2_isqrt(int val) {
    if (val <= 0) {
        return 0;
    }
    int res = 0;
    int bit = 1 << 30;
    while (bit > val) {
        bit = bit >> 2;
    }
    while (bit != 0) {
        if (val >= res + bit) {
            val = val - (res + bit);
            res = (res >> 1) + bit;
        } else {
            res = res >> 1;
        }
        bit = bit >> 2;
    }
    return res;
}

int64_t c2_isqrt64(int64_t val) {
    if (val <= 0) {
        return 0;
    }
    int64_t res = 0;
    int64_t bit = (int64_t)1 << 62;
    while (bit > val) {
        bit = bit >> 2;
    }
    while (bit != 0) {
        if (val >= res + bit) {
            val = val - (res + bit);
            res = (res >> 1) + bit;
        } else {
            res = res >> 1;
        }
        bit = bit >> 2;
    }
    return res;
}

uint32_t c2_rgba(uint8_t r, uint8_t g, uint8_t b, uint8_t a) {
    return ((uint32_t)r) | (((uint32_t)g) << 8) | (((uint32_t)b) << 16) | (((uint32_t)a) << 24);
}

uint32_t c2_rgba_premul(uint8_t r, uint8_t g, uint8_t b, uint8_t a) {
    uint32_t pr = ((uint32_t)r * (uint32_t)a + 127) / 255;
    uint32_t pg = ((uint32_t)g * (uint32_t)a + 127) / 255;
    uint32_t pb = ((uint32_t)b * (uint32_t)a + 127) / 255;
    return pr | (pg << 8) | (pb << 16) | (((uint32_t)a) << 24);
}

uint32_t c2_blend_pixel(uint32_t dst, uint32_t src) {
    uint32_t sa = (src >> 24) & 0xFF;
    if (sa == 255) {
        return src;
    }
    if (sa == 0) {
        return dst;
    }
    uint32_t sr = src & 0xFF;
    uint32_t sg = (src >> 8) & 0xFF;
    uint32_t sb = (src >> 16) & 0xFF;

    uint32_t dr = dst & 0xFF;
    uint32_t dg = (dst >> 8) & 0xFF;
    uint32_t db = (dst >> 16) & 0xFF;
    uint32_t da = (dst >> 24) & 0xFF;

    uint32_t inv_a = 255 - sa;
    uint32_t out_r = sr + (dr * inv_a + 127) / 255;
    uint32_t out_g = sg + (dg * inv_a + 127) / 255;
    uint32_t out_b = sb + (db * inv_a + 127) / 255;
    uint32_t out_a = sa + (da * inv_a + 127) / 255;

    if (out_r > 255) out_r = 255;
    if (out_g > 255) out_g = 255;
    if (out_b > 255) out_b = 255;
    if (out_a > 255) out_a = 255;

    return out_r | (out_g << 8) | (out_b << 16) | (out_a << 24);
}

uint32_t c2_blend_pixel_cov(uint32_t dst, uint32_t src, uint32_t cov) {
    if (cov == 0) {
        return dst;
    }
    if (cov >= 255) {
        return c2_blend_pixel(dst, src);
    }
    uint32_t sr = ((src & 0xFF) * cov + 127) / 255;
    uint32_t sg = (((src >> 8) & 0xFF) * cov + 127) / 255;
    uint32_t sb = (((src >> 16) & 0xFF) * cov + 127) / 255;
    uint32_t sa = (((src >> 24) & 0xFF) * cov + 127) / 255;
    uint32_t scaled_src = sr | (sg << 8) | (sb << 16) | (sa << 24);
    return c2_blend_pixel(dst, scaled_src);
}

void c2_painter_init(c2_painter_t *p, uint32_t *pixels, int width, int height, int stride) {
    p->pixels = pixels;
    p->width = width;
    p->height = height;
    p->stride = stride;
    p->clip.x = 0;
    p->clip.y = 0;
    p->clip.w = width;
    p->clip.h = height;
}

void c2_painter_set_clip(c2_painter_t *p, int x, int y, int w, int h) {
    if (x < 0) {
        w = w + x;
        x = 0;
    }
    if (y < 0) {
        h = h + y;
        y = 0;
    }
    if (x + w > p->width) {
        w = p->width - x;
    }
    if (y + h > p->height) {
        h = p->height - y;
    }
    if (w < 0) {
        w = 0;
    }
    if (h < 0) {
        h = 0;
    }
    p->clip.x = x;
    p->clip.y = y;
    p->clip.w = w;
    p->clip.h = h;
}

void c2_painter_reset_clip(c2_painter_t *p) {
    p->clip.x = 0;
    p->clip.y = 0;
    p->clip.w = p->width;
    p->clip.h = p->height;
}

void c2_clear(c2_painter_t *p, uint32_t color) {
    if (p->width <= 0 || p->height <= 0) {
        return;
    }
    if (p->stride == p->width) {
        int total = p->width * p->height;
        int i;
        for (i = 0; i < total; i++) {
            p->pixels[i] = color;
        }
        return;
    }
    int y;
    int x;
    for (y = 0; y < p->height; y++) {
        int row = y * p->stride;
        for (x = 0; x < p->width; x++) {
            p->pixels[row + x] = color;
        }
    }
}

void c2_fill_rect(c2_painter_t *p, int x, int y, int w, int h, uint32_t color) {
    if (w <= 0 || h <= 0) {
        return;
    }
    int x0 = x;
    int y0 = y;
    int x1 = x + w;
    int y1 = y + h;

    int cx0 = p->clip.x;
    int cy0 = p->clip.y;
    int cx1 = p->clip.x + p->clip.w;
    int cy1 = p->clip.y + p->clip.h;

    if (x0 < cx0) x0 = cx0;
    if (y0 < cy0) y0 = cy0;
    if (x1 > cx1) x1 = cx1;
    if (y1 > cy1) y1 = cy1;

    if (x0 >= x1 || y0 >= y1) {
        return;
    }

    uint32_t sa = (color >> 24) & 0xFF;
    int cy;
    int cx;
    if (sa == 255) {
        for (cy = y0; cy < y1; cy++) {
            int row = cy * p->stride;
            for (cx = x0; cx < x1; cx++) {
                p->pixels[row + cx] = color;
            }
        }
    } else if (sa > 0) {
        for (cy = y0; cy < y1; cy++) {
            int row = cy * p->stride;
            for (cx = x0; cx < x1; cx++) {
                p->pixels[row + cx] = c2_blend_pixel(p->pixels[row + cx], color);
            }
        }
    }
}

void c2_stroke_rect(c2_painter_t *p, int x, int y, int w, int h, int stroke_width, uint32_t color) {
    if (stroke_width <= 0 || w <= 0 || h <= 0) {
        return;
    }
    if (stroke_width * 2 >= w || stroke_width * 2 >= h) {
        c2_fill_rect(p, x, y, w, h, color);
        return;
    }
    /* Top */
    c2_fill_rect(p, x, y, w, stroke_width, color);
    /* Bottom */
    c2_fill_rect(p, x, y + h - stroke_width, w, stroke_width, color);
    /* Left */
    c2_fill_rect(p, x, y + stroke_width, stroke_width, h - 2 * stroke_width, color);
    /* Right */
    c2_fill_rect(p, x + w - stroke_width, y + stroke_width, stroke_width, h - 2 * stroke_width, color);
}

void c2_fill_rounded_rect(c2_painter_t *p, int x, int y, int w, int h, int radius, uint32_t color) {
    if (w <= 0 || h <= 0) {
        return;
    }
    if (radius <= 0) {
        c2_fill_rect(p, x, y, w, h, color);
        return;
    }
    if (radius * 2 > w) {
        radius = w / 2;
    }
    if (radius * 2 > h) {
        radius = h / 2;
    }

    /* Center cross */
    c2_fill_rect(p, x + radius, y, w - 2 * radius, h, color);
    c2_fill_rect(p, x, y + radius, radius, h - 2 * radius, color);
    c2_fill_rect(p, x + w - radius, y + radius, radius, h - 2 * radius, color);

    /* 4 antialiased corners */
    int r = radius;
    int r_fp = r << 8;
    int cx0 = p->clip.x;
    int cy0 = p->clip.y;
    int cx1 = p->clip.x + p->clip.w;
    int cy1 = p->clip.y + p->clip.h;

    int corner_centers_x[4];
    int corner_centers_y[4];
    int corner_min_x[4];
    int corner_min_y[4];
    int corner_max_x[4];
    int corner_max_y[4];

    /* Top-left */
    corner_centers_x[0] = x + r;
    corner_centers_y[0] = y + r;
    corner_min_x[0] = x;
    corner_min_y[0] = y;
    corner_max_x[0] = x + r;
    corner_max_y[0] = y + r;

    /* Top-right */
    corner_centers_x[1] = x + w - r;
    corner_centers_y[1] = y + r;
    corner_min_x[1] = x + w - r;
    corner_min_y[1] = y;
    corner_max_x[1] = x + w;
    corner_max_y[1] = y + r;

    /* Bottom-left */
    corner_centers_x[2] = x + r;
    corner_centers_y[2] = y + h - r;
    corner_min_x[2] = x;
    corner_min_y[2] = y + h - r;
    corner_max_x[2] = x + r;
    corner_max_y[2] = y + h;

    /* Bottom-right */
    corner_centers_x[3] = x + w - r;
    corner_centers_y[3] = y + h - r;
    corner_min_x[3] = x + w - r;
    corner_min_y[3] = y + h - r;
    corner_max_x[3] = x + w;
    corner_max_y[3] = y + h;

    int c;
    for (c = 0; c < 4; c++) {
        int ccx = corner_centers_x[c];
        int ccy = corner_centers_y[c];
        int min_x = corner_min_x[c];
        int min_y = corner_min_y[c];
        int max_x = corner_max_x[c];
        int max_y = corner_max_y[c];

        if (min_x < cx0) min_x = cx0;
        if (min_y < cy0) min_y = cy0;
        if (max_x > cx1) max_x = cx1;
        if (max_y > cy1) max_y = cy1;

        int64_t inner_r = (int64_t)r - 1;
        int64_t inner_sq = (inner_r > 0) ? (inner_r * inner_r) : 0;
        int64_t outer_sq = (int64_t)(r + 2) * (r + 2);

        int py;
        int px;
        for (py = min_y; py < max_y; py++) {
            int row = py * p->stride;
            int64_t dy = (py < ccy) ? (ccy - py) : (py - ccy + 1);
            int64_t dy2 = dy * dy;
            for (px = min_x; px < max_x; px++) {
                int64_t dx = (px < ccx) ? (ccx - px) : (px - ccx + 1);
                int64_t dist_sq = dx * dx + dy2;
                if (dist_sq <= inner_sq) {
                    p->pixels[row + px] = c2_blend_pixel(p->pixels[row + px], color);
                } else if (dist_sq < outer_sq) {
                    int dist_fp = (int)c2_isqrt64(dist_sq << 16);
                    int diff = r_fp + 128 - dist_fp;
                    if (diff >= 255) {
                        p->pixels[row + px] = c2_blend_pixel(p->pixels[row + px], color);
                    } else if (diff > 0) {
                        p->pixels[row + px] = c2_blend_pixel_cov(p->pixels[row + px], color, (uint32_t)diff);
                    }
                }
            }
        }
    }
}

void c2_stroke_rounded_rect(c2_painter_t *p, int x, int y, int w, int h, int radius, int stroke_width, uint32_t color) {
    if (stroke_width <= 0 || w <= 0 || h <= 0) {
        return;
    }
    if (stroke_width * 2 >= w || stroke_width * 2 >= h) {
        c2_fill_rounded_rect(p, x, y, w, h, radius, color);
        return;
    }
    if (radius <= 0) {
        c2_stroke_rect(p, x, y, w, h, stroke_width, color);
        return;
    }
    if (radius * 2 > w) {
        radius = w / 2;
    }
    if (radius * 2 > h) {
        radius = h / 2;
    }

    /* 4 straight borders */
    c2_fill_rect(p, x + radius, y, w - 2 * radius, stroke_width, color);
    c2_fill_rect(p, x + radius, y + h - stroke_width, w - 2 * radius, stroke_width, color);
    c2_fill_rect(p, x, y + radius, stroke_width, h - 2 * radius, color);
    c2_fill_rect(p, x + w - stroke_width, y + radius, stroke_width, h - 2 * radius, color);

    /* 4 corner arcs */
    int r_out = radius;
    int r_in = radius - stroke_width;
    if (r_in < 0) {
        r_in = 0;
    }
    int r_out_fp = r_out << 8;
    int r_in_fp = r_in << 8;

    int cx0 = p->clip.x;
    int cy0 = p->clip.y;
    int cx1 = p->clip.x + p->clip.w;
    int cy1 = p->clip.y + p->clip.h;

    int corner_centers_x[4];
    int corner_centers_y[4];
    int corner_min_x[4];
    int corner_min_y[4];
    int corner_max_x[4];
    int corner_max_y[4];

    /* Top-left */
    corner_centers_x[0] = x + radius;
    corner_centers_y[0] = y + radius;
    corner_min_x[0] = x;
    corner_min_y[0] = y;
    corner_max_x[0] = x + radius;
    corner_max_y[0] = y + radius;

    /* Top-right */
    corner_centers_x[1] = x + w - radius;
    corner_centers_y[1] = y + radius;
    corner_min_x[1] = x + w - radius;
    corner_min_y[1] = y;
    corner_max_x[1] = x + w;
    corner_max_y[1] = y + radius;

    /* Bottom-left */
    corner_centers_x[2] = x + radius;
    corner_centers_y[2] = y + h - radius;
    corner_min_x[2] = x;
    corner_min_y[2] = y + h - radius;
    corner_max_x[2] = x + radius;
    corner_max_y[2] = y + h;

    /* Bottom-right */
    corner_centers_x[3] = x + w - radius;
    corner_centers_y[3] = y + h - radius;
    corner_min_x[3] = x + w - radius;
    corner_min_y[3] = y + h - radius;
    corner_max_x[3] = x + w;
    corner_max_y[3] = y + h;

    int c;
    for (c = 0; c < 4; c++) {
        int ccx = corner_centers_x[c];
        int ccy = corner_centers_y[c];
        int min_x = corner_min_x[c];
        int min_y = corner_min_y[c];
        int max_x = corner_max_x[c];
        int max_y = corner_max_y[c];

        if (min_x < cx0) min_x = cx0;
        if (min_y < cy0) min_y = cy0;
        if (max_x > cx1) max_x = cx1;
        if (max_y > cy1) max_y = cy1;

        int py;
        int px;
        for (py = min_y; py < max_y; py++) {
            int row = py * p->stride;
            int64_t dy = (py < ccy) ? (ccy - py) : (py - ccy + 1);
            int64_t dy2 = dy * dy;
            for (px = min_x; px < max_x; px++) {
                int64_t dx = (px < ccx) ? (ccx - px) : (px - ccx + 1);
                int64_t dist_sq = dx * dx + dy2;
                int dist_fp = (int)c2_isqrt64(dist_sq << 16);
                int diff_out = r_out_fp + 128 - dist_fp;
                int diff_in = dist_fp + 128 - r_in_fp;
                int cov = diff_out;
                if (diff_in < cov) {
                    cov = diff_in;
                }
                if (cov >= 255) {
                    p->pixels[row + px] = c2_blend_pixel(p->pixels[row + px], color);
                } else if (cov > 0) {
                    p->pixels[row + px] = c2_blend_pixel_cov(p->pixels[row + px], color, (uint32_t)cov);
                }
            }
        }
    }
}

void c2_fill_circle(c2_painter_t *p, int cx, int cy, int radius, uint32_t color) {
    if (radius <= 0) {
        return;
    }
    int r_fp = radius << 8;
    int min_x = cx - radius - 1;
    int max_x = cx + radius + 1;
    int min_y = cy - radius - 1;
    int max_y = cy + radius + 1;

    int cx0 = p->clip.x;
    int cy0 = p->clip.y;
    int cx1 = p->clip.x + p->clip.w;
    int cy1 = p->clip.y + p->clip.h;

    if (min_x < cx0) min_x = cx0;
    if (min_y < cy0) min_y = cy0;
    if (max_x > cx1) max_x = cx1;
    if (max_y > cy1) max_y = cy1;

    if (min_x >= max_x || min_y >= max_y) {
        return;
    }

    int64_t inner_r = (int64_t)radius - 1;
    int64_t inner_sq_fp = (inner_r > 0) ? (inner_r * 256 * inner_r * 256) : 0;
    int64_t outer_r = (int64_t)radius + 2;
    int64_t outer_sq_fp = outer_r * 256 * outer_r * 256;

    int py;
    int px;
    for (py = min_y; py < max_y; py++) {
        int row = py * p->stride;
        int64_t dy_fp = (int64_t)(py - cy) * 256;
        int64_t dy2 = dy_fp * dy_fp;
        for (px = min_x; px < max_x; px++) {
            int64_t dx_fp = (int64_t)(px - cx) * 256;
            int64_t dist_sq = dx_fp * dx_fp + dy2;
            if (dist_sq <= inner_sq_fp) {
                p->pixels[row + px] = c2_blend_pixel(p->pixels[row + px], color);
            } else if (dist_sq < outer_sq_fp) {
                int dist_fp = (int)c2_isqrt64(dist_sq);
                int diff = r_fp + 128 - dist_fp;
                if (diff >= 255) {
                    p->pixels[row + px] = c2_blend_pixel(p->pixels[row + px], color);
                } else if (diff > 0) {
                    p->pixels[row + px] = c2_blend_pixel_cov(p->pixels[row + px], color, (uint32_t)diff);
                }
            }
        }
    }
}

void c2_stroke_circle(c2_painter_t *p, int cx, int cy, int radius, int stroke_width, uint32_t color) {
    if (radius <= 0 || stroke_width <= 0) {
        return;
    }
    int hw = (stroke_width + 1) / 2;
    int min_x = cx - radius - hw - 1;
    int max_x = cx + radius + hw + 1;
    int min_y = cy - radius - hw - 1;
    int max_y = cy + radius + hw + 1;

    int cx0 = p->clip.x;
    int cy0 = p->clip.y;
    int cx1 = p->clip.x + p->clip.w;
    int cy1 = p->clip.y + p->clip.h;

    if (min_x < cx0) min_x = cx0;
    if (min_y < cy0) min_y = cy0;
    if (max_x > cx1) max_x = cx1;
    if (max_y > cy1) max_y = cy1;

    if (min_x >= max_x || min_y >= max_y) {
        return;
    }

    int r_out_fp = (radius + hw) << 8;
    int r_in_val = radius - hw;
    if (r_in_val < 0) {
        r_in_val = 0;
    }
    int r_in_fp = r_in_val << 8;

    int py;
    int px;
    for (py = min_y; py < max_y; py++) {
        int row = py * p->stride;
        int64_t dy_fp = (int64_t)(py - cy) * 256;
        int64_t dy2 = dy_fp * dy_fp;
        for (px = min_x; px < max_x; px++) {
            int64_t dx_fp = (int64_t)(px - cx) * 256;
            int64_t dist_sq = dx_fp * dx_fp + dy2;
            int dist_fp = (int)c2_isqrt64(dist_sq);
            int diff_out = r_out_fp + 128 - dist_fp;
            int diff_in = dist_fp + 128 - r_in_fp;
            int cov = diff_out;
            if (diff_in < cov) {
                cov = diff_in;
            }
            if (cov >= 255) {
                p->pixels[row + px] = c2_blend_pixel(p->pixels[row + px], color);
            } else if (cov > 0) {
                p->pixels[row + px] = c2_blend_pixel_cov(p->pixels[row + px], color, (uint32_t)cov);
            }
        }
    }
}

void c2_fill_ellipse(c2_painter_t *p, int cx, int cy, int rx, int ry, uint32_t color) {
    if (rx <= 0 || ry <= 0) {
        return;
    }
    int min_x = cx - rx - 1;
    int max_x = cx + rx + 1;
    int min_y = cy - ry - 1;
    int max_y = cy + ry + 1;

    int cx0 = p->clip.x;
    int cy0 = p->clip.y;
    int cx1 = p->clip.x + p->clip.w;
    int cy1 = p->clip.y + p->clip.h;

    if (min_x < cx0) min_x = cx0;
    if (min_y < cy0) min_y = cy0;
    if (max_x > cx1) max_x = cx1;
    if (max_y > cy1) max_y = cy1;

    if (min_x >= max_x || min_y >= max_y) {
        return;
    }

    int64_t limit = (int64_t)rx * ry;
    int64_t rx2 = (int64_t)rx * rx;
    int64_t ry2 = (int64_t)ry * ry;
    int64_t norm = (rx > ry) ? rx : ry;

    int py;
    int px;
    for (py = min_y; py < max_y; py++) {
        int row = py * p->stride;
        int64_t dy = py - cy;
        int64_t dy_term = dy * dy * rx2;
        for (px = min_x; px < max_x; px++) {
            int64_t dx = px - cx;
            int64_t dx_term = dx * dx * ry2;
            int64_t dist = c2_isqrt64(dx_term + dy_term);
            int64_t diff = limit - dist;
            int cov = 0;
            if (norm > 0) {
                cov = (int)((diff * 255) / norm + 128);
            }
            if (cov >= 255) {
                p->pixels[row + px] = c2_blend_pixel(p->pixels[row + px], color);
            } else if (cov > 0) {
                p->pixels[row + px] = c2_blend_pixel_cov(p->pixels[row + px], color, (uint32_t)cov);
            }
        }
    }
}

void c2_stroke_ellipse(c2_painter_t *p, int cx, int cy, int rx, int ry, int stroke_width, uint32_t color) {
    if (rx <= 0 || ry <= 0 || stroke_width <= 0) {
        return;
    }
    int hw = (stroke_width + 1) / 2;
    int min_x = cx - rx - hw - 1;
    int max_x = cx + rx + hw + 1;
    int min_y = cy - ry - hw - 1;
    int max_y = cy + ry + hw + 1;

    int cx0 = p->clip.x;
    int cy0 = p->clip.y;
    int cx1 = p->clip.x + p->clip.w;
    int cy1 = p->clip.y + p->clip.h;

    if (min_x < cx0) min_x = cx0;
    if (min_y < cy0) min_y = cy0;
    if (max_x > cx1) max_x = cx1;
    if (max_y > cy1) max_y = cy1;

    if (min_x >= max_x || min_y >= max_y) {
        return;
    }

    int64_t rx_out = rx + hw;
    int64_t ry_out = ry + hw;
    int64_t rx_in = rx - hw;
    if (rx_in < 0) rx_in = 0;
    int64_t ry_in = ry - hw;
    if (ry_in < 0) ry_in = 0;

    int64_t limit_out = rx_out * ry_out;
    int64_t limit_in = rx_in * ry_in;
    int64_t rx2_out = rx_out * rx_out;
    int64_t ry2_out = ry_out * ry_out;
    int64_t rx2_in = rx_in * rx_in;
    int64_t ry2_in = ry_in * ry_in;

    int64_t norm = (rx_out > ry_out) ? rx_out : ry_out;

    int py;
    int px;
    for (py = min_y; py < max_y; py++) {
        int row = py * p->stride;
        int64_t dy = py - cy;
        int64_t dy2 = dy * dy;
        int64_t dy_term_out = dy2 * rx2_out;
        int64_t dy_term_in = dy2 * rx2_in;

        for (px = min_x; px < max_x; px++) {
            int64_t dx = px - cx;
            int64_t dx2 = dx * dx;
            int64_t dx_term_out = dx2 * ry2_out;
            int64_t dx_term_in = dx2 * ry2_in;

            int64_t dist_out = c2_isqrt64(dx_term_out + dy_term_out);
            int64_t dist_in = c2_isqrt64(dx_term_in + dy_term_in);

            int64_t diff_out = limit_out - dist_out;
            int64_t diff_in = dist_in - limit_in;

            int64_t diff = diff_out;
            if (diff_in < diff) {
                diff = diff_in;
            }

            int cov = 0;
            if (norm > 0) {
                cov = (int)((diff * 255) / norm + 128);
            }

            if (cov >= 255) {
                p->pixels[row + px] = c2_blend_pixel(p->pixels[row + px], color);
            } else if (cov > 0) {
                p->pixels[row + px] = c2_blend_pixel_cov(p->pixels[row + px], color, (uint32_t)cov);
            }
        }
    }
}

void c2_draw_line(c2_painter_t *p, int x0, int y0, int x1, int y1, int stroke_width, uint32_t color) {
    if (stroke_width <= 0) {
        return;
    }
    int64_t dx = x1 - x0;
    int64_t dy = y1 - y0;
    int64_t len_sq = dx * dx + dy * dy;

    if (len_sq == 0) {
        c2_fill_circle(p, x0, y0, (stroke_width + 1) / 2, color);
        return;
    }

    int margin = (stroke_width + 3) / 2 + 1;
    int min_x = (x0 < x1) ? x0 : x1;
    int max_x = (x0 > x1) ? x0 : x1;
    int min_y = (y0 < y1) ? y0 : y1;
    int max_y = (y0 > y1) ? y0 : y1;

    min_x = min_x - margin;
    max_x = max_x + margin;
    min_y = min_y - margin;
    max_y = max_y + margin;

    int cx0 = p->clip.x;
    int cy0 = p->clip.y;
    int cx1 = p->clip.x + p->clip.w;
    int cy1 = p->clip.y + p->clip.h;

    if (min_x < cx0) min_x = cx0;
    if (min_y < cy0) min_y = cy0;
    if (max_x > cx1) max_x = cx1;
    if (max_y > cy1) max_y = cy1;

    if (min_x >= max_x || min_y >= max_y) {
        return;
    }

    int64_t len = c2_isqrt64(len_sq);
    int64_t bound = (int64_t)margin * (len + 1);
    int64_t hw_fp = (int64_t)stroke_width * 128;

    int py;
    int px;
    for (py = min_y; py < max_y; py++) {
        int span_min_x = min_x;
        int span_max_x = max_x;

        if (dy != 0) {
            int64_t center_term = dx * (int64_t)(py - y0);
            int64_t x_inf;
            int64_t x_sup;
            if (dy > 0) {
                x_inf = (int64_t)x0 + (center_term - bound) / dy - 1;
                x_sup = (int64_t)x0 + (center_term + bound) / dy + 2;
            } else {
                x_inf = (int64_t)x0 + (center_term + bound) / dy - 1;
                x_sup = (int64_t)x0 + (center_term - bound) / dy + 2;
            }
            if (x_inf > span_min_x) span_min_x = (int)x_inf;
            if (x_sup < span_max_x) span_max_x = (int)x_sup;
        }

        if (span_min_x < min_x) span_min_x = min_x;
        if (span_max_x > max_x) span_max_x = max_x;
        if (span_min_x >= span_max_x) continue;

        int row = py * p->stride;
        int64_t vy = py - y0;
        for (px = span_min_x; px < span_max_x; px++) {
            int64_t vx = px - x0;
            int64_t dot = vx * dx + vy * dy;
            int64_t t_fp = 0;
            if (dot >= len_sq) {
                t_fp = 256;
            } else if (dot > 0) {
                t_fp = (dot * 256) / len_sq;
            }
            int64_t qx_fp = ((int64_t)x0 * 256) + (dx * t_fp);
            int64_t qy_fp = ((int64_t)y0 * 256) + (dy * t_fp);
            int64_t ex_fp = ((int64_t)px * 256 + 128) - qx_fp;
            int64_t ey_fp = ((int64_t)py * 256 + 128) - qy_fp;
            int64_t dist_sq = ex_fp * ex_fp + ey_fp * ey_fp;
            int dist_fp = (int)c2_isqrt64(dist_sq);
            int diff = (int)(hw_fp + 128 - dist_fp);
            if (diff >= 255) {
                p->pixels[row + px] = c2_blend_pixel(p->pixels[row + px], color);
            } else if (diff > 0) {
                p->pixels[row + px] = c2_blend_pixel_cov(p->pixels[row + px], color, (uint32_t)diff);
            }
        }
    }
}

void c2_blit_mask(c2_painter_t *p, int dst_x, int dst_y, int w, int h, const uint8_t *mask, int mask_stride, uint32_t color) {
    if (w <= 0 || h <= 0) {
        return;
    }
    int x0 = dst_x;
    int y0 = dst_y;
    int x1 = dst_x + w;
    int y1 = dst_y + h;

    int cx0 = p->clip.x;
    int cy0 = p->clip.y;
    int cx1 = p->clip.x + p->clip.w;
    int cy1 = p->clip.y + p->clip.h;

    if (x0 < cx0) x0 = cx0;
    if (y0 < cy0) y0 = cy0;
    if (x1 > cx1) x1 = cx1;
    if (y1 > cy1) y1 = cy1;

    if (x0 >= x1 || y0 >= y1) {
        return;
    }

    int cy;
    int cx;
    for (cy = y0; cy < y1; cy++) {
        int dst_row = cy * p->stride;
        int mask_row = (cy - dst_y) * mask_stride;
        for (cx = x0; cx < x1; cx++) {
            uint8_t cov = mask[mask_row + (cx - dst_x)];
            if (cov > 0) {
                p->pixels[dst_row + cx] = c2_blend_pixel_cov(p->pixels[dst_row + cx], color, (uint32_t)cov);
            }
        }
    }
}

void c2_blit_rgba(c2_painter_t *p, int dst_x, int dst_y, int w, int h, const uint32_t *src, int src_stride) {
    if (w <= 0 || h <= 0) {
        return;
    }
    int x0 = dst_x;
    int y0 = dst_y;
    int x1 = dst_x + w;
    int y1 = dst_y + h;

    int cx0 = p->clip.x;
    int cy0 = p->clip.y;
    int cx1 = p->clip.x + p->clip.w;
    int cy1 = p->clip.y + p->clip.h;

    if (x0 < cx0) x0 = cx0;
    if (y0 < cy0) y0 = cy0;
    if (x1 > cx1) x1 = cx1;
    if (y1 > cy1) y1 = cy1;

    if (x0 >= x1 || y0 >= y1) {
        return;
    }

    int cy;
    int cx;
    for (cy = y0; cy < y1; cy++) {
        int dst_row = cy * p->stride;
        int src_row = (cy - dst_y) * src_stride;
        for (cx = x0; cx < x1; cx++) {
            uint32_t pixel = src[src_row + (cx - dst_x)];
            p->pixels[dst_row + cx] = c2_blend_pixel(p->pixels[dst_row + cx], pixel);
        }
    }
}

void c2_fill_linear_gradient(c2_painter_t *p, int x, int y, int w, int h, uint32_t color0, uint32_t color1, int vertical) {
    if (w <= 0 || h <= 0) {
        return;
    }
    int x0 = x;
    int y0 = y;
    int x1 = x + w;
    int y1 = y + h;

    int cx0 = p->clip.x;
    int cy0 = p->clip.y;
    int cx1 = p->clip.x + p->clip.w;
    int cy1 = p->clip.y + p->clip.h;

    if (x0 < cx0) x0 = cx0;
    if (y0 < cy0) y0 = cy0;
    if (x1 > cx1) x1 = cx1;
    if (y1 > cy1) y1 = cy1;

    if (x0 >= x1 || y0 >= y1) {
        return;
    }

    uint32_t r0 = color0 & 0xFF;
    uint32_t g0 = (color0 >> 8) & 0xFF;
    uint32_t b0 = (color0 >> 16) & 0xFF;
    uint32_t a0 = (color0 >> 24) & 0xFF;

    uint32_t r1 = color1 & 0xFF;
    uint32_t g1 = (color1 >> 8) & 0xFF;
    uint32_t b1 = (color1 >> 16) & 0xFF;
    uint32_t a1 = (color1 >> 24) & 0xFF;

    int span = vertical ? h : w;
    if (span <= 1) {
        c2_fill_rect(p, x, y, w, h, color0);
        return;
    }

    int cy;
    int cx;
    if (vertical) {
        for (cy = y0; cy < y1; cy++) {
            int t = ((cy - y) * 255) / (span - 1);
            int inv_t = 255 - t;
            uint32_t cr = (r0 * inv_t + r1 * t + 127) / 255;
            uint32_t cg = (g0 * inv_t + g1 * t + 127) / 255;
            uint32_t cb = (b0 * inv_t + b1 * t + 127) / 255;
            uint32_t ca = (a0 * inv_t + a1 * t + 127) / 255;
            uint32_t col = cr | (cg << 8) | (cb << 16) | (ca << 24);

            int row = cy * p->stride;
            for (cx = x0; cx < x1; cx++) {
                p->pixels[row + cx] = c2_blend_pixel(p->pixels[row + cx], col);
            }
        }
    } else {
        for (cx = x0; cx < x1; cx++) {
            int t = ((cx - x) * 255) / (span - 1);
            int inv_t = 255 - t;
            uint32_t cr = (r0 * inv_t + r1 * t + 127) / 255;
            uint32_t cg = (g0 * inv_t + g1 * t + 127) / 255;
            uint32_t cb = (b0 * inv_t + b1 * t + 127) / 255;
            uint32_t ca = (a0 * inv_t + a1 * t + 127) / 255;
            uint32_t col = cr | (cg << 8) | (cb << 16) | (ca << 24);

            for (cy = y0; cy < y1; cy++) {
                int row = cy * p->stride;
                p->pixels[row + cx] = c2_blend_pixel(p->pixels[row + cx], col);
            }
        }
    }
}

void c2_draw_cursor(c2_painter_t *p, int x, int y, int w, int h, int style, uint32_t color) {
    if (w <= 0 || h <= 0) {
        return;
    }
    switch (style) {
    case 0: /* Block plein */
        c2_fill_rect(p, x, y, w, h, color);
        break;
    case 1: /* Bar verticale */
        c2_fill_rect(p, x, y, (2 > w) ? w : 2, h, color);
        break;
    case 2: /* Underline */
        c2_fill_rect(p, x, y + h - ((2 > h) ? h : 2), w, (2 > h) ? h : 2, color);
        break;
    case 3: /* Hollow box */
        c2_stroke_rect(p, x, y, w, h, 1, color);
        break;
    default:
        c2_fill_rect(p, x, y, w, h, color);
        break;
    }
}

void c2_scene_render(c2_painter_t *p, const c2_node_t *nodes, int count) {
    if (p == 0 || nodes == 0 || count <= 0) {
        return;
    }
    int i;
    for (i = 0; i < count; i++) {
        const c2_node_t *n = &nodes[i];
        switch (n->type) {
        case C2_NODE_RECT:
            if (n->flags == 1) {
                c2_stroke_rect(p, n->x, n->y, n->w, n->h, n->param0, n->color0);
            } else {
                c2_fill_rect(p, n->x, n->y, n->w, n->h, n->color0);
            }
            break;
        case C2_NODE_ROUNDED_RECT:
            if (n->flags == 1) {
                c2_stroke_rounded_rect(p, n->x, n->y, n->w, n->h, n->param0, n->param1, n->color0);
            } else {
                c2_fill_rounded_rect(p, n->x, n->y, n->w, n->h, n->param0, n->color0);
            }
            break;
        case C2_NODE_CIRCLE:
            if (n->flags == 1) {
                c2_stroke_circle(p, n->x, n->y, n->param0, n->param1, n->color0);
            } else {
                c2_fill_circle(p, n->x, n->y, n->param0, n->color0);
            }
            break;
        case C2_NODE_ELLIPSE:
            c2_fill_ellipse(p, n->x, n->y, n->param0, n->param1, n->color0);
            break;
        case C2_NODE_LINE:
            c2_draw_line(p, n->x, n->y, n->w, n->h, n->param0, n->color0);
            break;
        default:
            break;
        }
    }
}

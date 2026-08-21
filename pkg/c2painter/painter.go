package c2painter

// Surface représente un tampon mémoire 2D RGBA 32-bit pour le rendu graphique.
type Surface struct {
	Pixels []uint32
	Width  int
	Height int
	Stride int
}

// NewSurface alloue une nouvelle surface de dimensions spécifiées.
func NewSurface(width, height int) *Surface {
	if width <= 0 || height <= 0 {
		return &Surface{
			Pixels: nil,
			Width:  0,
			Height: 0,
			Stride: 0,
		}
	}
	stride := width
	pixels := make([]uint32, width*height)
	return &Surface{
		Pixels: pixels,
		Width:  width,
		Height: height,
		Stride: stride,
	}
}

// RGBA construit une couleur 32-bit RGBA (R: octet 0, G: octet 1, B: octet 2, A: octet 3).
func RGBA(r, g, b, a uint8) uint32 {
	return C2_rgba(r, g, b, a)
}

// PackRGBA construit une couleur 32-bit RGBA (R: octet 0, G: octet 1, B: octet 2, A: octet 3).
func PackRGBA(r, g, b, a uint8) uint32 {
	return C2_rgba(r, g, b, a)
}

// UnpackRGBA extrait les 4 composantes R, G, B, A d'une couleur uint32.
func UnpackRGBA(c uint32) (r, g, b, a uint8) {
	r = uint8(c & 0xFF)
	g = uint8((c >> 8) & 0xFF)
	b = uint8((c >> 16) & 0xFF)
	a = uint8((c >> 24) & 0xFF)
	return
}

// RGBAPremul construit une couleur 32-bit RGBA prémultipliée.
func RGBAPremul(r, g, b, a uint8) uint32 {
	return C2_rgba_premul(r, g, b, a)
}

// Painter est l'enveloppe idiomatique Go du contexte C2_painter_t transpilé.
type Painter struct {
	ctx C2_painter_t
}

// NewPainter crée un peintre opérant sur la surface fournie.
func NewPainter(surface *Surface) *Painter {
	p := &Painter{}
	if surface != nil {
		C2_painter_init(&p.ctx, surface.Pixels, surface.Width, surface.Height, surface.Stride)
	}
	return p
}

// NewPainterFromBuffer initialise un peintre directement depuis un slice de pixels RGBA.
func NewPainterFromBuffer(pixels []uint32, width, height, stride int) *Painter {
	p := &Painter{}
	C2_painter_init(&p.ctx, pixels, width, height, stride)
	return p
}

// Pixels retourne le slice de pixels sous-jacent.
func (p *Painter) Pixels() []uint32 {
	return p.ctx.Pixels
}

// Width retourne la largeur du tampon.
func (p *Painter) Width() int {
	return p.ctx.Width
}

// Height retourne la hauteur du tampon.
func (p *Painter) Height() int {
	return p.ctx.Height
}

// Stride retourne le stride du tampon.
func (p *Painter) Stride() int {
	return p.ctx.Stride
}

// SetClip définit le rectangle de découpage actif.
func (p *Painter) SetClip(x, y, w, h int) {
	C2_painter_set_clip(&p.ctx, x, y, w, h)
}

// ResetClip réinitialise le rectangle de découpage aux limites complètes de la surface.
func (p *Painter) ResetClip() {
	C2_painter_reset_clip(&p.ctx)
}

// Clear remplit l'intégralité du tampon avec la couleur indiquée en exploitant le doublement exponentiel memmove.
func (p *Painter) Clear(color uint32) {
	pixels := p.ctx.Pixels
	if len(pixels) == 0 || p.ctx.Width <= 0 || p.ctx.Height <= 0 {
		return
	}
	if p.ctx.Stride == p.ctx.Width {
		total := p.ctx.Width * p.ctx.Height
		if total > len(pixels) {
			total = len(pixels)
		}
		pixels[0] = color
		for n := 1; n < total; n *= 2 {
			copy(pixels[n:total], pixels[:n])
		}
		return
	}
	C2_clear(&p.ctx, color)
}

// DrawRect remplit un rectangle avec couleur RGBA prémultipliée.
func (p *Painter) DrawRect(x, y, w, h int, color uint32) {
	C2_fill_rect(&p.ctx, x, y, w, h, color)
}

// StrokeRect trace le contour d'un rectangle.
func (p *Painter) StrokeRect(x, y, w, h, strokeWidth int, color uint32) {
	C2_stroke_rect(&p.ctx, x, y, w, h, strokeWidth, color)
}

// DrawRoundedRect remplit un rectangle à coins arrondis avec antialiasing.
func (p *Painter) DrawRoundedRect(x, y, w, h, radius int, color uint32) {
	C2_fill_rounded_rect(&p.ctx, x, y, w, h, radius, color)
}

// StrokeRoundedRect trace le contour d'un rectangle à coins arrondis avec antialiasing.
func (p *Painter) StrokeRoundedRect(x, y, w, h, radius, strokeWidth int, color uint32) {
	C2_stroke_rounded_rect(&p.ctx, x, y, w, h, radius, strokeWidth, color)
}

// DrawCircle remplit un cercle avec antialiasing.
func (p *Painter) DrawCircle(cx, cy, radius int, color uint32) {
	C2_fill_circle(&p.ctx, cx, cy, radius, color)
}

// StrokeCircle trace le contour d'un cercle avec antialiasing.
func (p *Painter) StrokeCircle(cx, cy, radius, strokeWidth int, color uint32) {
	C2_stroke_circle(&p.ctx, cx, cy, radius, strokeWidth, color)
}

// DrawEllipse remplit une ellipse avec antialiasing.
func (p *Painter) DrawEllipse(cx, cy, rx, ry int, color uint32) {
	C2_fill_ellipse(&p.ctx, cx, cy, rx, ry, color)
}

// StrokeEllipse trace le contour d'une ellipse avec antialiasing.
func (p *Painter) StrokeEllipse(cx, cy, rx, ry, strokeWidth int, color uint32) {
	C2_stroke_ellipse(&p.ctx, cx, cy, rx, ry, strokeWidth, color)
}

// DrawLine trace un segment de droite antialiasé d'épaisseur strokeWidth.
func (p *Painter) DrawLine(x0, y0, x1, y1, strokeWidth int, color uint32) {
	C2_draw_line(&p.ctx, x0, y0, x1, y1, strokeWidth, color)
}

// DrawTextGlyph blitte un masque alpha 8-bit (glyphe textuel) avec la couleur spécifiée.
func (p *Painter) DrawTextGlyph(dstX, dstY, w, h int, mask []byte, maskStride int, color uint32) {
	C2_blit_mask(&p.ctx, dstX, dstY, w, h, mask, maskStride, color)
}

// Blit copie une image RGBA avec composition alpha Porter-Duff Source-Over.
func (p *Painter) Blit(dstX, dstY, w, h int, src []uint32, srcStride int) {
	C2_blit_rgba(&p.ctx, dstX, dstY, w, h, src, srcStride)
}

// FillLinearGradient remplit un rectangle avec un dégradé linéaire bicolore.
func (p *Painter) FillLinearGradient(x, y, w, h int, color0, color1 uint32, vertical bool) {
	v := 0
	if vertical {
		v = 1
	}
	C2_fill_linear_gradient(&p.ctx, x, y, w, h, color0, color1, v)
}

// DrawLinearGradient est un alias de FillLinearGradient pour la compatibilité d'API.
func (p *Painter) DrawLinearGradient(x, y, w, h int, color0, color1 uint32, vertical bool) {
	p.FillLinearGradient(x, y, w, h, color0, color1, vertical)
}

// DrawCursor trace un curseur contrasté idempotent selon son style (0: Block, 1: Bar, 2: Underline, 3: Hollow).
func (p *Painter) DrawCursor(x, y, w, h, style int, color uint32) {
	C2_draw_cursor(&p.ctx, x, y, w, h, style, color)
}

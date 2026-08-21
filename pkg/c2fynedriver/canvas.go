package c2fynedriver

import (
	"fmt"
	"sync"

	c2display "code.hazyhaar.fr/devhoros/pkg/c2display"
	c2fyneterm "code.hazyhaar.fr/devhoros/pkg/c2fyneterm"
	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
)

// Position définit des coordonnées 2D entières relatives ou absolues.
type Position struct {
	X int
	Y int
}

// Add additionne deux positions.
func (p Position) Add(o Position) Position {
	return Position{X: p.X + o.X, Y: p.Y + o.Y}
}

// Subtract soustrait deux positions.
func (p Position) Subtract(o Position) Position {
	return Position{X: p.X - o.X, Y: p.Y - o.Y}
}

// Size définit les dimensions 2D entières d'un objet graphique.
type Size struct {
	Width  int
	Height int
}

// CanvasObject est l'interface commune de tous les éléments graphiques de l'arbre visuel.
type CanvasObject interface {
	Position() Position
	Move(Position)
	Size() Size
	Resize(Size)
	MinSize() Size
	Visible() bool
	Show()
	Hide()
	Refresh()
}

// InteractiveObject est implémenté par les widgets réagissant aux événements pointeur.
type InteractiveObject interface {
	CanvasObject
	OnMousePress(pos Position, btn c2display.MouseButton) bool
	OnMouseRelease(pos Position, btn c2display.MouseButton) bool
	OnMouseEnter()
	OnMouseLeave()
}

// KeyReceiver est implémenté par les widgets recevant les entrées clavier.
type KeyReceiver interface {
	CanvasObject
	OnKeyEvent(ev c2display.KeyEvent) bool
}

// BaseObject fournit l'implémentation standard des propriétés d'un CanvasObject.
type BaseObject struct {
	mu      sync.RWMutex
	pos     Position
	size    Size
	visible bool
	dirty   bool
}

func (b *BaseObject) Position() Position {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.pos
}

func (b *BaseObject) Move(p Position) {
	b.mu.Lock()
	b.pos = p
	b.dirty = true
	b.mu.Unlock()
}

func (b *BaseObject) Size() Size {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.size
}

func (b *BaseObject) Resize(s Size) {
	b.mu.Lock()
	b.size = s
	b.dirty = true
	b.mu.Unlock()
}

func (b *BaseObject) MinSize() Size {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.size
}

func (b *BaseObject) Visible() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.visible
}

func (b *BaseObject) Show() {
	b.mu.Lock()
	b.visible = true
	b.dirty = true
	b.mu.Unlock()
}

func (b *BaseObject) Hide() {
	b.mu.Lock()
	b.visible = false
	b.dirty = true
	b.mu.Unlock()
}

func (b *BaseObject) Refresh() {
	b.mu.Lock()
	b.dirty = true
	b.mu.Unlock()
}

// NewBaseObject initialise un objet visible par défaut.
func NewBaseObject() BaseObject {
	return BaseObject{
		visible: true,
		dirty:   true,
	}
}

// ==========================================
// Primitives Graphiques Fyne
// ==========================================

// Rectangle représente un polygone rectangulaire plein ou avec contour.
type Rectangle struct {
	BaseObject
	FillColor   uint32
	StrokeColor uint32
	StrokeWidth int
}

// NewRectangle instancie un rectangle plein.
func NewRectangle(fillColor uint32) *Rectangle {
	return &Rectangle{
		BaseObject: NewBaseObject(),
		FillColor:  fillColor,
	}
}

// RoundedRectangle / Card représente un rectangle à coins arrondis et ombre portée optionnelle.
type RoundedRectangle struct {
	BaseObject
	FillColor    uint32
	StrokeColor  uint32
	StrokeWidth  int
	CornerRadius int
	Elevation    int
	ShadowColor  uint32
}

// NewRoundedRectangle instancie un rectangle arrondi.
func NewRoundedRectangle(fillColor uint32, radius int) *RoundedRectangle {
	return &RoundedRectangle{
		BaseObject:   NewBaseObject(),
		FillColor:    fillColor,
		CornerRadius: radius,
	}
}

// Card représente un conteneur carte graphique stylisé avec en-tête, ombre portée et contenu.
type Card struct {
	BaseObject
	Title       string
	Subtitle    string
	HeaderBadge string
	BadgeBg     uint32
	BadgeFg     uint32
	FillColor   uint32
	BorderColor uint32
	Radius      int
	Elevation   int
	Content     CanvasObject
}

// NewCard instancie une carte UI stylisée.
func NewCard(title, subtitle string, content CanvasObject) *Card {
	return &Card{
		BaseObject:  NewBaseObject(),
		Title:       title,
		Subtitle:    subtitle,
		FillColor:   c2painter.PackRGBA(30, 41, 59, 255), // Slate 800
		BorderColor: c2painter.PackRGBA(51, 65, 85, 255), // Slate 700
		Radius:      12,
		Elevation:   4,
		Content:     content,
	}
}

// Circle représente un cercle graphique plein ou cerclé.
type Circle struct {
	BaseObject
	FillColor   uint32
	StrokeColor uint32
	StrokeWidth int
}

// NewCircle instancie un cercle.
func NewCircle(fillColor uint32) *Circle {
	return &Circle{
		BaseObject: NewBaseObject(),
		FillColor:  fillColor,
	}
}

// Line représente un segment de droite entre deux points.
type Line struct {
	BaseObject
	Position1   Position
	Position2   Position
	StrokeColor uint32
	StrokeWidth int
}

// NewLine instancie un segment de ligne.
func NewLine(p1, p2 Position, color uint32, width int) *Line {
	if width <= 0 {
		width = 1
	}
	return &Line{
		BaseObject:  NewBaseObject(),
		Position1:   p1,
		Position2:   p2,
		StrokeColor: color,
		StrokeWidth: width,
	}
}

// TextAlign spécifie l'alignement d'un texte.
type TextAlign uint8

const (
	TextAlignLeading TextAlign = iota
	TextAlignCenter
	TextAlignTrailing
)

// Label représente un texte typographique mono ou multi-lignes.
type Label struct {
	BaseObject
	Text      string
	TextSize  int
	Color     uint32
	Alignment TextAlign
	Bold      bool
}

// NewLabel instancie un texte simple.
func NewLabel(text string) *Label {
	return &Label{
		BaseObject: NewBaseObject(),
		Text:       text,
		TextSize:   14,
		Color:      c2painter.PackRGBA(241, 245, 249, 255), // Slate 100
		Alignment:  TextAlignLeading,
	}
}

// MinSize calcule les dimensions minimales du texte.
func (l *Label) MinSize() Size {
	w, h := MeasureString(l.Text, l.TextSize)
	return Size{Width: w, Height: h}
}

// Badge représente une étiquette de statut compacte arrondie.
type Badge struct {
	BaseObject
	Text         string
	TextSize     int
	BgColor      uint32
	FgColor      uint32
	CornerRadius int
	PaddingH     int
	PaddingV     int
}

// NewBadge instancie un badge de statut.
func NewBadge(text string, bg, fg uint32) *Badge {
	return &Badge{
		BaseObject:   NewBaseObject(),
		Text:         text,
		TextSize:     11,
		BgColor:      bg,
		FgColor:      fg,
		CornerRadius: 6,
		PaddingH:     8,
		PaddingV:     4,
	}
}

func (b *Badge) MinSize() Size {
	w, h := MeasureString(b.Text, b.TextSize)
	return Size{
		Width:  w + b.PaddingH*2,
		Height: h + b.PaddingV*2,
	}
}

// ButtonImportance détermine la hiérarchie visuelle d'un bouton.
type ButtonImportance uint8

const (
	ButtonMedium ButtonImportance = iota
	ButtonHigh
	ButtonLow
	ButtonDanger
)

// Button représente un bouton interactif réagissant au survol et au clic.
type Button struct {
	BaseObject
	Text         string
	Icon         string
	Importance   ButtonImportance
	BgColor      uint32
	HoverColor   uint32
	PressedColor uint32
	TextColor    uint32
	CornerRadius int
	OnTapped     func()
	isHovered    bool
	isPressed    bool
}

// NewButton instancie un bouton standard.
func NewButton(text string, onTapped func()) *Button {
	return &Button{
		BaseObject:   NewBaseObject(),
		Text:         text,
		Importance:   ButtonMedium,
		BgColor:      c2painter.PackRGBA(37, 99, 235, 255),  // Blue 600
		HoverColor:   c2painter.PackRGBA(29, 78, 216, 255),  // Blue 700
		PressedColor: c2painter.PackRGBA(30, 58, 138, 255),  // Blue 900
		TextColor:    c2painter.PackRGBA(255, 255, 255, 255),
		CornerRadius: 8,
		OnTapped:     onTapped,
	}
}

func (b *Button) MinSize() Size {
	tw, th := MeasureString(b.Text, 14)
	return Size{
		Width:  tw + 32,
		Height: th + 16,
	}
}

func (b *Button) OnMousePress(pos Position, btn c2display.MouseButton) bool {
	if btn == c2display.ButtonLeft {
		b.isPressed = true
		b.Refresh()
		return true
	}
	return false
}

// SetFillColor configure la couleur de fond du rectangle sous verrouillage.
func (r *Rectangle) SetFillColor(col uint32) {
	r.mu.Lock()
	r.FillColor = col
	r.dirty = true
	r.mu.Unlock()
}

// SetFillColor configure la couleur de fond du rectangle arrondi sous verrouillage.
func (rr *RoundedRectangle) SetFillColor(col uint32) {
	rr.mu.Lock()
	rr.FillColor = col
	rr.dirty = true
	rr.mu.Unlock()
}

// SetFillColor configure la couleur de remplissage du cercle sous verrouillage.
func (c *Circle) SetFillColor(col uint32) {
	c.mu.Lock()
	c.FillColor = col
	c.dirty = true
	c.mu.Unlock()
}

// SetStrokeColor configure la couleur du trait sous verrouillage.
func (l *Line) SetStrokeColor(col uint32) {
	l.mu.Lock()
	l.StrokeColor = col
	l.dirty = true
	l.mu.Unlock()
}

// SetText met à jour le texte du label sous verrouillage.
func (l *Label) SetText(text string) {
	l.mu.Lock()
	l.Text = text
	l.dirty = true
	l.mu.Unlock()
}

// SetBgColor met à jour la couleur d'arrière-plan du badge sous verrouillage.
func (b *Badge) SetBgColor(col uint32) {
	b.mu.Lock()
	b.BgColor = col
	b.dirty = true
	b.mu.Unlock()
}

// SetBgColor met à jour la couleur d'arrière-plan du bouton sous verrouillage.
func (b *Button) SetBgColor(col uint32) {
	b.mu.Lock()
	b.BgColor = col
	b.dirty = true
	b.mu.Unlock()
}

func (b *Button) OnMouseRelease(pos Position, btn c2display.MouseButton) bool {
	if btn == c2display.ButtonLeft && b.isPressed {
		b.isPressed = false
		b.Refresh()
		if b.OnTapped != nil {
			b.OnTapped()
		}
		return true
	}
	b.isPressed = false
	b.Refresh()
	return false
}

func (b *Button) OnMouseEnter() {
	b.isHovered = true
	b.Refresh()
}

func (b *Button) OnMouseLeave() {
	b.isHovered = false
	b.isPressed = false
	b.Refresh()
}

// TerminalComponent intègre le TerminalWidget haute performance de c2fyneterm dans l'arbre graphique Fyne.
type TerminalComponent struct {
	BaseObject
	Term      *c2fyneterm.TerminalWidget
	Surface   *c2painter.Surface
	Focused   bool
	OnChanged func()
}

// NewTerminalComponent instancie le composant terminal Fyne avec dimensions de grille spécifiées.
func NewTerminalComponent(cols, rows int) *TerminalComponent {
	term := c2fyneterm.NewTerminalWidget(cols, rows)
	cellW, cellH := term.CellSize()
	width := cols * cellW
	height := rows * cellH

	surf := c2painter.NewSurface(width, height)

	tc := &TerminalComponent{
		BaseObject: NewBaseObject(),
		Term:       term,
		Surface:    surf,
		Focused:    true,
	}
	tc.Resize(Size{Width: width, Height: height})
	return tc
}

func (tc *TerminalComponent) MinSize() Size {
	cellW, cellH := tc.Term.CellSize()
	return Size{
		Width:  tc.Term.Cols() * cellW,
		Height: tc.Term.Rows() * cellH,
	}
}

func (tc *TerminalComponent) OnKeyEvent(ev c2display.KeyEvent) bool {
	if tc.Term == nil || ev.Action != c2display.EventKeyPress {
		return false
	}

	if ev.Rune != 0 {
		_, _ = tc.Term.Write([]byte(string(ev.Rune)))
		tc.Refresh()
		return true
	}

	switch ev.Key {
	case c2display.KeyEnter:
		_, _ = tc.Term.Write([]byte("\r\n"))
	case c2display.KeyBackspace:
		_, _ = tc.Term.Write([]byte("\b \b"))
	case c2display.KeyTab:
		_, _ = tc.Term.Write([]byte("\t"))
	case c2display.KeyUp:
		_, _ = tc.Term.Write([]byte("\x1b[A"))
	case c2display.KeyDown:
		_, _ = tc.Term.Write([]byte("\x1b[B"))
	case c2display.KeyRight:
		_, _ = tc.Term.Write([]byte("\x1b[C"))
	case c2display.KeyLeft:
		_, _ = tc.Term.Write([]byte("\x1b[D"))
	}
	tc.Refresh()
	return true
}

func (tc *TerminalComponent) OnMousePress(pos Position, btn c2display.MouseButton) bool {
	tc.Focused = true
	cellW, cellH := tc.Term.CellSize()
	if cellW > 0 && cellH > 0 {
		col := pos.X / cellW
		row := pos.Y / cellH
		tc.Term.StartSelection(col, row)
		tc.Refresh()
		return true
	}
	return false
}

func (tc *TerminalComponent) OnMouseRelease(pos Position, btn c2display.MouseButton) bool {
	tc.Term.EndSelection()
	tc.Refresh()
	return false
}

func (tc *TerminalComponent) OnMouseEnter() {}
func (tc *TerminalComponent) OnMouseLeave() {}

// FPSCounter est un widget compact affichant le taux de rafraîchissement d'affichage en temps réel.
type FPSCounter struct {
	BaseObject
	driver *Driver
}

// NewFPSCounter instancie un compteur de FPS connecté au Driver actif.
func NewFPSCounter(d *Driver) *FPSCounter {
	fc := &FPSCounter{
		BaseObject: NewBaseObject(),
		driver:     d,
	}
	fc.Resize(Size{Width: 130, Height: 28})
	return fc
}

func (fc *FPSCounter) MinSize() Size {
	return Size{Width: 130, Height: 28}
}

// ==========================================
// Conteneurs et Disposition (Layout)
// ==========================================

// Container regroupe plusieurs CanvasObjects avec positionnement hiérarchique.
type Container struct {
	BaseObject
	Objects []CanvasObject
	Layout  func(c *Container)
}

// NewContainer instancie un conteneur simple.
func NewContainer(objects ...CanvasObject) *Container {
	c := &Container{
		BaseObject: NewBaseObject(),
		Objects:    objects,
	}
	return c
}

// NewHBox instancie un conteneur horizontal avec espacement automatique.
func NewHBox(spacing int, objects ...CanvasObject) *Container {
	c := &Container{
		BaseObject: NewBaseObject(),
		Objects:    objects,
	}
	c.Layout = func(cont *Container) {
		curX := 0
		maxH := 0
		for _, obj := range cont.Objects {
			if !obj.Visible() {
				continue
			}
			ms := obj.MinSize()
			obj.Move(Position{X: curX, Y: 0})
			obj.Resize(ms)
			curX += ms.Width + spacing
			if ms.Height > maxH {
				maxH = ms.Height
			}
		}
		cont.Resize(Size{Width: curX, Height: maxH})
	}
	c.LayoutChildren()
	return c
}

// NewVBox instancie un conteneur vertical avec espacement automatique.
func NewVBox(spacing int, objects ...CanvasObject) *Container {
	c := &Container{
		BaseObject: NewBaseObject(),
		Objects:    objects,
	}
	c.Layout = func(cont *Container) {
		curY := 0
		maxW := 0
		for _, obj := range cont.Objects {
			if !obj.Visible() {
				continue
			}
			ms := obj.MinSize()
			obj.Move(Position{X: 0, Y: curY})
			obj.Resize(ms)
			curY += ms.Height + spacing
			if ms.Width > maxW {
				maxW = ms.Width
			}
		}
		cont.Resize(Size{Width: maxW, Height: curY})
	}
	c.LayoutChildren()
	return c
}

// Add insère un ou plusieurs objets dans le conteneur.
func (c *Container) Add(objects ...CanvasObject) {
	c.Objects = append(c.Objects, objects...)
	c.LayoutChildren()
	c.Refresh()
}

// LayoutChildren recalcule les positions et tailles des enfants.
func (c *Container) LayoutChildren() {
	if c.Layout != nil {
		c.Layout(c)
	}
}

// ==========================================
// Moteur de Canevas (Canvas)
// ==========================================

// Canvas encapsule la surface de dessin d'une fenêtre et gère le parcours d'arbre et le rendu.
type Canvas struct {
	mu          sync.RWMutex
	size        Size
	content     CanvasObject
	focused     KeyReceiver
	hovered     InteractiveObject
	driver      *Driver
	window      *Window
	clearColor  uint32
	isDirty     bool
	damageTrack bool
}

// NewCanvas instancie un canevas rattaché à une fenêtre et des dimensions spécifiées.
func NewCanvas(win *Window, width, height int) *Canvas {
	return &Canvas{
		size:        Size{Width: width, Height: height},
		window:      win,
		clearColor:  c2painter.PackRGBA(15, 23, 42, 255), // Slate 900
		isDirty:     true,
		damageTrack: true,
	}
}

// SetContent définit l'élément racine de l'arbre graphique du canevas.
func (c *Canvas) SetContent(content CanvasObject) {
	c.mu.Lock()
	c.content = content
	c.isDirty = true
	c.mu.Unlock()
}

// Content retourne l'élément racine du canevas.
func (c *Canvas) Content() CanvasObject {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.content
}

// Size retourne les dimensions actuelles du canevas.
func (c *Canvas) Size() Size {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}

// Resize met à jour les dimensions du canevas.
func (c *Canvas) Resize(s Size) {
	c.mu.Lock()
	c.size = s
	c.isDirty = true
	c.mu.Unlock()
}

// SetClearColor configure la couleur de fond du canevas.
func (c *Canvas) SetClearColor(col uint32) {
	c.mu.Lock()
	c.clearColor = col
	c.isDirty = true
	c.mu.Unlock()
}

// Walk parcourt récursivement l'arbre d'objets graphiques du canevas.
func (c *Canvas) Walk(visitor func(obj CanvasObject, absPos Position, size Size) bool) {
	c.mu.RLock()
	root := c.content
	c.mu.RUnlock()

	if root == nil || !root.Visible() {
		return
	}

	walkObject(root, Position{X: 0, Y: 0}, visitor)
}

func walkObject(obj CanvasObject, parentPos Position, visitor func(CanvasObject, Position, Size) bool) {
	if obj == nil || !obj.Visible() {
		return
	}

	absPos := parentPos.Add(obj.Position())
	size := obj.Size()

	cont := visitor(obj, absPos, size)
	if !cont {
		return
	}

	switch container := obj.(type) {
	case *Container:
		for _, child := range container.Objects {
			walkObject(child, absPos, visitor)
		}
	case *Card:
		if container.Content != nil {
			contentPos := absPos.Add(Position{X: 16, Y: 56})
			walkObject(container.Content, contentPos, visitor)
		}
	}
}

// Paint traduit les objets graphiques de l'arbre Fyne vers les appels du peintre vectoriel c2painter.
func (c *Canvas) Paint(p *c2painter.Painter) {
	if p == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Nettoyage de la surface
	p.Clear(c.clearColor)

	if c.content == nil || !c.content.Visible() {
		return
	}

	// 2. Rendu récursif ordonné (Painter's Algorithm)
	walkObject(c.content, Position{X: 0, Y: 0}, func(obj CanvasObject, absPos Position, size Size) bool {
		renderObject(p, obj, absPos, size, c)
		return true
	})

	c.isDirty = false
}

// IsDirty retourne vrai si le canevas nécessite un rafraîchissement d'affichage.
func (c *Canvas) IsDirty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isDirty
}

// SetDirty marque explicitement le canevas comme devant être redessiné.
func (c *Canvas) SetDirty(dirty bool) {
	c.mu.Lock()
	c.isDirty = dirty
	c.mu.Unlock()
}

// renderObject dispatche chaque CanvasObject vers la primitive correspondante de c2painter.
func renderObject(p *c2painter.Painter, obj CanvasObject, absPos Position, size Size, canvas *Canvas) {
	x := absPos.X
	y := absPos.Y
	w := size.Width
	h := size.Height

	if w <= 0 || h <= 0 {
		return
	}

	switch elem := obj.(type) {
	case *Rectangle:
		elem.mu.RLock()
		fill := elem.FillColor
		strokeW := elem.StrokeWidth
		strokeCol := elem.StrokeColor
		elem.mu.RUnlock()
		if fill != 0 {
			p.DrawRect(x, y, w, h, fill)
		}
		if strokeW > 0 && strokeCol != 0 {
			p.StrokeRect(x, y, w, h, strokeW, strokeCol)
		}

	case *RoundedRectangle:
		elem.mu.RLock()
		elev := elem.Elevation
		shadowCol := elem.ShadowColor
		radius := elem.CornerRadius
		fill := elem.FillColor
		strokeW := elem.StrokeWidth
		strokeCol := elem.StrokeColor
		elem.mu.RUnlock()
		if elev > 0 && shadowCol != 0 {
			drawShadow(p, x, y, w, h, radius, elev, shadowCol)
		}
		if fill != 0 {
			p.DrawRoundedRect(x, y, w, h, radius, fill)
		}
		if strokeW > 0 && strokeCol != 0 {
			p.StrokeRoundedRect(x, y, w, h, radius, strokeW, strokeCol)
		}

	case *Card:
		elem.mu.RLock()
		elev := elem.Elevation
		radius := elem.Radius
		fill := elem.FillColor
		borderCol := elem.BorderColor
		title := elem.Title
		sub := elem.Subtitle
		badge := elem.HeaderBadge
		badgeBg := elem.BadgeBg
		badgeFg := elem.BadgeFg
		elem.mu.RUnlock()
		if elev > 0 {
			shadowCol := c2painter.PackRGBA(0, 0, 0, 90)
			drawShadow(p, x, y, w, h, radius, elev, shadowCol)
		}
		p.DrawRoundedRect(x, y, w, h, radius, fill)
		if borderCol != 0 {
			p.StrokeRoundedRect(x, y, w, h, radius, 1, borderCol)
		}
		if title != "" {
			DrawString(p, title, x+16, y+16, 16, c2painter.PackRGBA(248, 250, 252, 255))
		}
		if sub != "" {
			DrawString(p, sub, x+16, y+36, 12, c2painter.PackRGBA(148, 163, 184, 255))
		}
		if badge != "" {
			bw, bh := MeasureString(badge, 11)
			bx := x + w - bw - 24
			by := y + 14
			p.DrawRoundedRect(bx-6, by-2, bw+12, bh+4, 4, badgeBg)
			DrawString(p, badge, bx, by, 11, badgeFg)
		}

	case *Circle:
		cx := x + w/2
		cy := y + h/2
		r := w / 2
		if h/2 < r {
			r = h / 2
		}
		elem.mu.RLock()
		fill := elem.FillColor
		strokeW := elem.StrokeWidth
		strokeCol := elem.StrokeColor
		elem.mu.RUnlock()
		if fill != 0 {
			p.DrawCircle(cx, cy, r, fill)
		}
		if strokeW > 0 && strokeCol != 0 {
			p.StrokeCircle(cx, cy, r, strokeW, strokeCol)
		}

	case *Line:
		elem.mu.RLock()
		p1 := elem.Position1
		p2 := elem.Position2
		strokeW := elem.StrokeWidth
		strokeCol := elem.StrokeColor
		elem.mu.RUnlock()
		p.DrawLine(x+p1.X, y+p1.Y, x+p2.X, y+p2.Y, strokeW, strokeCol)

	case *Label:
		elem.mu.RLock()
		txt := elem.Text
		sz := elem.TextSize
		col := elem.Color
		elem.mu.RUnlock()
		DrawString(p, txt, x, y, sz, col)

	case *Badge:
		elem.mu.RLock()
		radius := elem.CornerRadius
		bg := elem.BgColor
		txt := elem.Text
		sz := elem.TextSize
		fg := elem.FgColor
		elem.mu.RUnlock()
		p.DrawRoundedRect(x, y, w, h, radius, bg)
		tw, th := MeasureString(txt, sz)
		tx := x + (w-tw)/2
		ty := y + (h-th)/2
		DrawString(p, txt, tx, ty, sz, fg)

	case *Button:
		elem.mu.RLock()
		col := elem.BgColor
		if elem.isPressed {
			col = elem.PressedColor
		} else if elem.isHovered {
			col = elem.HoverColor
		}
		radius := elem.CornerRadius
		txt := elem.Text
		txtCol := elem.TextColor
		elem.mu.RUnlock()
		drawShadow(p, x, y, w, h, radius, 2, c2painter.PackRGBA(0, 0, 0, 60))
		p.DrawRoundedRect(x, y, w, h, radius, col)
		p.StrokeRoundedRect(x, y, w, h, radius, 1, c2painter.PackRGBA(255, 255, 255, 30))

		tw, _ := MeasureString(txt, 14)
		tx := x + (w-tw)/2
		ty := y + (h-14)/2
		DrawString(p, txt, tx, ty, 14, txtCol)

	case *TerminalComponent:
		if elem.Term != nil {
			elem.mu.Lock()
			if elem.Surface == nil || elem.Surface.Width != w || elem.Surface.Height != h {
				elem.Surface = c2painter.NewSurface(w, h)
			}
			surf := elem.Surface
			focused := elem.Focused
			elem.mu.Unlock()
			elem.Term.RenderToSurface(surf)
			p.Blit(x, y, w, h, surf.Pixels, surf.Stride)

			// Bordure d'accentuation si focus
			if focused {
				p.StrokeRect(x-1, y-1, w+2, h+2, 1, c2painter.PackRGBA(59, 130, 246, 200))
			}
		}

	case *FPSCounter:
		// Rendu Compteur de FPS dynamique
		fps := 60.0
		if elem.driver != nil {
			fps = elem.driver.CurrentFPS()
		}

		// Fond translucide sombre avec bordure fine
		p.DrawRoundedRect(x, y, w, h, 6, c2painter.PackRGBA(15, 23, 42, 220))
		p.StrokeRoundedRect(x, y, w, h, 6, 1, c2painter.PackRGBA(71, 85, 105, 180))

		// Pastille de statut couleur (vert si > 50 FPS, orange si > 30, rouge sinon)
		dotCol := c2painter.PackRGBA(34, 197, 94, 255) // Vert 500
		if fps < 30.0 {
			dotCol = c2painter.PackRGBA(239, 68, 68, 255) // Rouge 500
		} else if fps < 50.0 {
			dotCol = c2painter.PackRGBA(245, 158, 11, 255) // Orange 500
		}
		p.DrawCircle(x+10, y+h/2, 4, dotCol)

		// Texte formaté FPS sans allocation
		fpsText := getCachedFPS(fps)
		DrawString(p, fpsText, x+20, y+7, 11, c2painter.PackRGBA(226, 232, 240, 255))
	}
}

var fpsCacheTable [1000]string

func init() {
	for i := 0; i < 1000; i++ {
		fpsCacheTable[i] = fmt.Sprintf("%d FPS", i)
	}
}

func getCachedFPS(fps float64) string {
	intFps := int(fps + 0.5)
	if intFps >= 0 && intFps < 1000 {
		return fpsCacheTable[intFps]
	}
	return "FPS"
}

// drawShadow trace une ombre portée douce sous un rectangle arrondi.
func drawShadow(p *c2painter.Painter, x, y, w, h, radius, elevation int, color uint32) {
	if elevation <= 0 || color == 0 {
		return
	}
	alphaBase := uint8((color >> 24) & 0xff)
	col1 := (color & 0x00FFFFFF) | (uint32(alphaBase/3) << 24)
	p.DrawRoundedRect(x-elevation, y+elevation, w+elevation*2, h+elevation, radius+elevation, col1)

	col2 := (color & 0x00FFFFFF) | (uint32((alphaBase*2)/3) << 24)
	p.DrawRoundedRect(x-elevation/2, y+elevation/2, w+elevation, h+elevation/2, radius+elevation/2, col2)
}

// HandleMouseEvent route les interactions de la souris vers les objets graphiques interactifs.
func (c *Canvas) HandleMouseEvent(ev c2display.MouseEvent) {
	c.mu.Lock()

	pt := Position{X: ev.X, Y: ev.Y}

	var hitObj InteractiveObject
	var hitPos Position

	if c.content != nil && c.content.Visible() {
		walkObject(c.content, Position{X: 0, Y: 0}, func(obj CanvasObject, absPos Position, size Size) bool {
			if !obj.Visible() {
				return false
			}
			// Vérification d'inclusion du point
			if pt.X >= absPos.X && pt.X < absPos.X+size.Width &&
				pt.Y >= absPos.Y && pt.Y < absPos.Y+size.Height {
				if inter, ok := obj.(InteractiveObject); ok {
					hitObj = inter
					hitPos = pt.Subtract(absPos)
				}
			}
			return true
		})
	}

	prevHovered := c.hovered
	var leaveObj, enterObj InteractiveObject
	if hitObj != prevHovered {
		leaveObj = prevHovered
		enterObj = hitObj
		c.hovered = hitObj
	}

	if hitObj != nil && ev.Action == c2display.EventMousePress {
		if keyRec, ok := hitObj.(KeyReceiver); ok {
			c.focused = keyRec
		}
	}

	// F3: Libération de c.mu AVANT d'exécuter les rappels utilisateurs pour interdire tout interblocage
	c.mu.Unlock()

	if leaveObj != nil {
		leaveObj.OnMouseLeave()
	}
	if enterObj != nil {
		enterObj.OnMouseEnter()
	}

	if hitObj != nil {
		switch ev.Action {
		case c2display.EventMousePress:
			hitObj.OnMousePress(hitPos, ev.Button)
		case c2display.EventMouseRelease:
			hitObj.OnMouseRelease(hitPos, ev.Button)
		}
	}
}

// HandleKeyEvent route les événements clavier vers l'élément ayant le focus.
func (c *Canvas) HandleKeyEvent(ev c2display.KeyEvent) {
	c.mu.RLock()
	f := c.focused
	c.mu.RUnlock()

	if f != nil {
		f.OnKeyEvent(ev)
	}
}

// SetFocused définit explicitement le récepteur clavier avec focus.
func (c *Canvas) SetFocused(k KeyReceiver) {
	c.mu.Lock()
	c.focused = k
	c.mu.Unlock()
}

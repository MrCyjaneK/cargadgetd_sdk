// Package fynekit hosts a Fyne software canvas for C ABI plugins.
package fynekit

import (
	"image"
	"image/draw"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// Kit is a windowless Fyne canvas: Capture for draw, tap/drag for touch.
type Kit struct {
	mu sync.Mutex

	canvas test.WindowlessCanvas
	last   *image.RGBA
	paused bool
	dirty  bool

	dragOn      bool
	dragObj     fyne.Draggable
	dragAbs     fyne.Position
	dragRel     fyne.Position
	dragFrom    fyne.Position
	transparent bool
}

func New() *Kit { return &Kit{} }

// NewTransparent is a kit whose capture keeps empty pixels clear, so a
// caller can paint a frame underneath and composite the widgets over it.
func NewTransparent() *Kit {
	k := New()
	k.transparent = true
	return k
}

// Ensure builds the widget tree once. build runs under the kit lock.
func (k *Kit) Ensure(build func() fyne.CanvasObject) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.canvas != nil {
		return
	}
	test.NewApp()
	fyne.CurrentApp().Settings().SetTheme(theme.DarkTheme())
	var c test.WindowlessCanvas
	if k.transparent {
		c = software.NewTransparentCanvas()
	} else {
		c = software.NewCanvas()
	}
	c.SetScale(1)
	c.SetPadded(true)
	c.SetContent(build())
	k.canvas = c
	k.dirty = true
}

func (k *Kit) Canvas() test.WindowlessCanvas {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.canvas
}

// MarkDirty forces the next Draw to re-Capture. Software Capture is expensive
// (one core at ~30fps); skip it when the widget tree has not changed.
func (k *Kit) MarkDirty() {
	k.mu.Lock()
	k.dirty = true
	k.mu.Unlock()
}

func (k *Kit) Draw(dst *image.RGBA) {
	b := dst.Bounds()
	if k.prepare(b.Dx(), b.Dy()) == nil {
		return
	}
	k.mu.Lock()
	last := k.last
	k.mu.Unlock()
	if last != nil {
		draw.Draw(dst, b, last, last.Bounds().Min, draw.Src)
	}
}

// Overlay composites the canvas over dst. Pixels the widgets leave clear
// stay as they were, so a map frame can sit underneath.
func (k *Kit) Overlay(dst *image.RGBA) {
	b := dst.Bounds()
	if k.prepare(b.Dx(), b.Dy()) == nil {
		return
	}
	k.mu.Lock()
	last := k.last
	k.mu.Unlock()
	if last != nil {
		draw.Draw(dst, b, last, last.Bounds().Min, draw.Over)
	}
}

// Hits reports a visible leaf widget under the pixel. Containers and empty
// gaps do not count, so a drag on the map still reaches the caller.
func (k *Kit) Hits(x, y int) bool {
	k.mu.Lock()
	c := k.canvas
	k.mu.Unlock()
	if c == nil || c.Content() == nil {
		return false
	}
	return hitLeaf(c.Content(), fyne.NewPos(float32(x), float32(y)), fyne.NewPos(0, 0))
}

func (k *Kit) prepare(w, h int) *image.RGBA {
	if w < 1 || h < 1 {
		return nil
	}
	k.mu.Lock()
	paused := k.paused
	dirty := k.dirty
	last := k.last
	c := k.canvas
	k.mu.Unlock()
	if c == nil {
		return nil
	}
	sizeOK := last != nil && last.Bounds().Dx() == w && last.Bounds().Dy() == h
	if sizeOK && (paused || !dirty) {
		return last
	}
	c.Resize(fyne.NewSize(float32(w), float32(h)))
	// Fyne drops unused widget renderers after 1 minute. Software Capture
	// then CreateRenderer without Layout, and Resize is a no-op at the same
	// size, so icons vanish and labels snap left while hit boxes stay.
	relayout(c.Content())
	img := c.Capture()
	cp := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(cp, cp.Bounds(), img, image.Point{}, draw.Src)
	k.mu.Lock()
	k.last = cp
	k.dirty = false
	k.mu.Unlock()
	return cp
}

func hitLeaf(o fyne.CanvasObject, abs, acc fyne.Position) bool {
	if o == nil || !o.Visible() {
		return false
	}
	pos := acc.Add(o.Position())
	sz := o.Size()
	if abs.X < pos.X || abs.Y < pos.Y || abs.X >= pos.X+sz.Width || abs.Y >= pos.Y+sz.Height {
		return false
	}
	if c, ok := o.(*fyne.Container); ok {
		for i := len(c.Objects) - 1; i >= 0; i-- {
			if hitLeaf(c.Objects[i], abs, pos) {
				return true
			}
		}
		return false
	}
	return true
}

// Touch maps plugin ABI (0 up, 1 down, 2 move) onto Fyne tap and drag.
func (k *Kit) Touch(x, y, action int) {
	pos := fyne.NewPos(float32(x), float32(y))
	k.mu.Lock()
	c := k.canvas
	k.mu.Unlock()
	if c == nil {
		return
	}
	switch action {
	case 1:
		obj, rel := findDraggable(c, pos)
		k.mu.Lock()
		k.dragOn = true
		k.dragObj = obj
		k.dragAbs = pos
		k.dragRel = rel
		k.dragFrom = pos
		k.mu.Unlock()
		k.MarkDirty()
	case 2:
		k.mu.Lock()
		if !k.dragOn || k.dragObj == nil {
			k.mu.Unlock()
			return
		}
		dx, dy := pos.X-k.dragAbs.X, pos.Y-k.dragAbs.Y
		k.dragAbs = pos
		k.dragRel = k.dragRel.Add(fyne.NewPos(dx, dy))
		obj, rel := k.dragObj, k.dragRel
		k.mu.Unlock()
		// Fyne moves a scroll on drag only when the device reports mobile.
		// This canvas does not, so a drag on a scroll moves the offset here.
		if sc, ok := obj.(*container.Scroll); ok {
			sc.ScrollToOffset(fyne.NewPos(sc.Offset.X-dx, sc.Offset.Y-dy))
		} else {
			obj.Dragged(&fyne.DragEvent{
				PointEvent: fyne.PointEvent{Position: rel, AbsolutePosition: pos},
				Dragged:    fyne.Delta{DX: dx, DY: dy},
			})
		}
		k.MarkDirty()
	case 0:
		k.mu.Lock()
		if !k.dragOn {
			k.mu.Unlock()
			return
		}
		k.dragOn = false
		obj := k.dragObj
		from := k.dragFrom
		k.dragObj = nil
		k.mu.Unlock()
		dx, dy := pos.X-from.X, pos.Y-from.Y
		if obj != nil && dx*dx+dy*dy >= 64 {
			obj.DragEnd()
			k.MarkDirty()
			return
		}
		test.TapCanvas(c, pos)
		k.MarkDirty()
	}
}

func relayout(o fyne.CanvasObject) {
	if o == nil {
		return
	}
	sz := o.Size()
	if sz.Width < 1 || sz.Height < 1 {
		return
	}
	o.Resize(fyne.NewSize(0, 0))
	o.Resize(sz)
}

func (k *Kit) Pause() {
	k.mu.Lock()
	k.paused = true
	k.mu.Unlock()
}

func (k *Kit) Resume() {
	k.mu.Lock()
	k.paused = false
	k.dirty = true
	k.mu.Unlock()
}

func findDraggable(c fyne.Canvas, pos fyne.Position) (fyne.Draggable, fyne.Position) {
	if c == nil {
		return nil, fyne.Position{}
	}
	if ov := c.Overlays(); ov != nil {
		if top := ov.Top(); top != nil {
			if d, rel, ok := walkDraggable(top, pos, fyne.NewPos(0, 0)); ok {
				return d, rel
			}
		}
	}
	if content := c.Content(); content != nil {
		if d, rel, ok := walkDraggable(content, pos, fyne.NewPos(0, 0)); ok {
			return d, rel
		}
	}
	return nil, fyne.Position{}
}

func walkDraggable(o fyne.CanvasObject, abs, acc fyne.Position) (fyne.Draggable, fyne.Position, bool) {
	if o == nil || !o.Visible() {
		return nil, fyne.Position{}, false
	}
	pos := acc.Add(o.Position())
	sz := o.Size()
	if abs.X < pos.X || abs.Y < pos.Y || abs.X >= pos.X+sz.Width || abs.Y >= pos.Y+sz.Height {
		return nil, fyne.Position{}, false
	}
	if c, ok := o.(*fyne.Container); ok {
		for i := len(c.Objects) - 1; i >= 0; i-- {
			if d, rel, found := walkDraggable(c.Objects[i], abs, pos); found {
				return d, rel, true
			}
		}
	}
	if d, ok := o.(fyne.Draggable); ok {
		return d, abs.Subtract(pos), true
	}
	return nil, fyne.Position{}, false
}

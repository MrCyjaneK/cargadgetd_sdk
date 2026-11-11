package fynekit

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func TestKitDrawAndTap(t *testing.T) {
	k := New()
	var n int
	var btn *widget.Button
	k.Ensure(func() fyne.CanvasObject {
		btn = widget.NewButton("Tap", func() { n++ })
		return container.NewCenter(btn)
	})
	dst := image.NewRGBA(image.Rect(0, 0, 800, 480))
	k.Draw(dst)
	if k.Canvas() == nil {
		t.Fatal("no canvas")
	}
	pos, ok := absPos(k.Canvas().Content(), btn, fyne.NewPos(0, 0))
	if !ok {
		t.Fatal("button not in tree")
	}
	sz := btn.Size()
	x, y := int(pos.X+sz.Width/2), int(pos.Y+sz.Height/2)
	k.Touch(x, y, 1)
	k.Touch(x, y, 0)
	if n != 1 {
		t.Fatalf("taps=%d", n)
	}
	var nonempty int
	for i := 0; i < len(dst.Pix); i += 4 {
		if dst.Pix[i] != 0 || dst.Pix[i+1] != 0 || dst.Pix[i+2] != 0 {
			nonempty++
		}
	}
	if nonempty < 1000 {
		t.Fatalf("blank capture nonempty=%d", nonempty)
	}
}

func absPos(o fyne.CanvasObject, target fyne.CanvasObject, acc fyne.Position) (fyne.Position, bool) {
	if o == nil {
		return fyne.Position{}, false
	}
	p := acc.Add(o.Position())
	if o == target {
		return p, true
	}
	c, ok := o.(*fyne.Container)
	if !ok {
		return fyne.Position{}, false
	}
	for _, ch := range c.Objects {
		if pos, found := absPos(ch, target, p); found {
			return pos, true
		}
	}
	return fyne.Position{}, false
}

func TestOverlayLeavesUnderlyingPixels(t *testing.T) {
	k := NewTransparent()
	var btn *widget.Button
	k.Ensure(func() fyne.CanvasObject {
		btn = widget.NewButton("Go", nil)
		return container.NewCenter(btn)
	})
	dst := image.NewRGBA(image.Rect(0, 0, 200, 80))
	for i := 0; i < len(dst.Pix); i += 4 {
		dst.Pix[i] = 9
		dst.Pix[i+3] = 255
	}
	k.Overlay(dst)
	kept := 0
	for y := 0; y < 8; y++ {
		p := dst.RGBAAt(2, y)
		if p.R == 9 && p.A == 255 {
			kept++
		}
	}
	if kept < 4 {
		t.Fatalf("corner overwritten, kept %d", kept)
	}
	if !k.Hits(100, 40) {
		t.Fatal("button missed")
	}
	if k.Hits(2, 2) {
		t.Fatal("empty corner hit")
	}
}

func TestKitDragScrolls(t *testing.T) {
	k := New()
	var sc *container.Scroll
	k.Ensure(func() fyne.CanvasObject {
		box := container.NewVBox()
		for i := 0; i < 40; i++ {
			box.Add(widget.NewLabel("row"))
		}
		sc = container.NewVScroll(box)
		return sc
	})
	dst := image.NewRGBA(image.Rect(0, 0, 400, 200))
	k.Draw(dst)
	if sc.Content.MinSize().Height <= sc.Size().Height {
		t.Fatalf("content %v view %v", sc.Content.MinSize(), sc.Size())
	}
	k.Touch(200, 160, 1)
	k.Touch(200, 40, 2)
	k.Touch(200, 40, 0)
	if sc.Offset.Y < 60 {
		t.Fatalf("offset %v", sc.Offset.Y)
	}
}

func TestSeekBarMinSize(t *testing.T) {
	s := NewSeekBar(0, 100)
	if s.MinSize().Height < seekMinH {
		t.Fatalf("height %v", s.MinSize())
	}
	if s.MinSize().Width < seekMinW {
		t.Fatalf("width %v", s.MinSize())
	}
}

func TestKitPauseSkipsCapture(t *testing.T) {
	k := New()
	lab := widget.NewLabel("hello")
	k.Ensure(func() fyne.CanvasObject { return lab })
	dst := image.NewRGBA(image.Rect(0, 0, 400, 240))
	k.Draw(dst)
	k.Pause()
	before := k.last
	lab.SetText("changed")
	k.Draw(dst)
	if k.last != before {
		t.Fatal("paused draw recaptured")
	}
	k.Resume()
	k.Draw(dst)
	if k.last == before {
		t.Fatal("resume did not recapture")
	}
	if !strings.Contains(lab.Text, "changed") {
		t.Fatalf("%q", lab.Text)
	}
}

// layoutProbe keeps a pointer to its inner object so a test can zero it the
// way Fyne's renderer cache drop does (CreateRenderer, no Layout).
type layoutProbe struct {
	widget.BaseWidget
	inner *canvas.Rectangle
}

func newLayoutProbe() *layoutProbe {
	p := &layoutProbe{inner: canvas.NewRectangle(color.NRGBA{R: 0xff, A: 0xff})}
	p.ExtendBaseWidget(p)
	return p
}

func (p *layoutProbe) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.inner)
}

func TestKitRelayoutRestoresRendererObjects(t *testing.T) {
	k := New()
	p := newLayoutProbe()
	k.Ensure(func() fyne.CanvasObject {
		return container.NewMax(p)
	})
	dst := image.NewRGBA(image.Rect(0, 0, 400, 240))
	k.Draw(dst)
	if p.inner.Size().Width < 100 {
		t.Fatalf("setup size %v", p.inner.Size())
	}
	p.inner.Resize(fyne.NewSize(0, 0))
	k.MarkDirty()
	k.Draw(dst)
	if p.inner.Size().Width < 100 {
		t.Fatalf("after recapture size %v (renderer not laid out)", p.inner.Size())
	}
}

func TestKitIdleSkipsCapture(t *testing.T) {
	k := New()
	lab := widget.NewLabel("hello")
	k.Ensure(func() fyne.CanvasObject { return lab })
	dst := image.NewRGBA(image.Rect(0, 0, 400, 240))
	k.Draw(dst)
	before := k.last
	k.Draw(dst)
	if k.last != before {
		t.Fatal("idle draw recaptured")
	}
	lab.SetText("changed")
	k.MarkDirty()
	k.Draw(dst)
	if k.last == before {
		t.Fatal("MarkDirty did not recapture")
	}
}

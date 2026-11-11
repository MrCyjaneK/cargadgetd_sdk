package fynekit

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	seekTrackH float32 = 28
	seekThumb  float32 = 40
	seekMinH   float32 = 56
	seekMinW   float32 = 240
)

// SeekBar is a thick horizontal seek control for HU touch (Fyne's Slider
// track is only ~4px). Implements Tappable and Draggable.
type SeekBar struct {
	widget.BaseWidget
	Min, Max, Value float64
	OnChanged       func(float64)
	OnChangeEnded   func(float64)
	disabled        bool
}

func NewSeekBar(min, max float64) *SeekBar {
	s := &SeekBar{Min: min, Max: max}
	s.ExtendBaseWidget(s)
	return s
}

func (s *SeekBar) CreateRenderer() fyne.WidgetRenderer {
	s.ExtendBaseWidget(s)
	track := canvas.NewRectangle(color.NRGBA{R: 0x2a, G: 0x33, B: 0x40, A: 0xff})
	fill := canvas.NewRectangle(color.NRGBA{R: 0xc0, G: 0x6a, B: 0xf7, A: 0xff})
	thumb := canvas.NewCircle(color.NRGBA{R: 0xe8, G: 0xee, B: 0xf4, A: 0xff})
	track.CornerRadius = 14
	fill.CornerRadius = 14
	r := &seekBarRenderer{bar: s, track: track, fill: fill, thumb: thumb}
	r.Refresh()
	return r
}

func (s *SeekBar) MinSize() fyne.Size {
	s.ExtendBaseWidget(s)
	m := s.BaseWidget.MinSize()
	if m.Height < seekMinH {
		m.Height = seekMinH
	}
	if m.Width < seekMinW {
		m.Width = seekMinW
	}
	return m
}

func (s *SeekBar) SetValue(v float64) {
	if v < s.Min {
		v = s.Min
	}
	if s.Max > s.Min && v > s.Max {
		v = s.Max
	}
	if s.Value == v {
		return
	}
	s.Value = v
	s.Refresh()
}

func (s *SeekBar) Enable() {
	if !s.disabled {
		return
	}
	s.disabled = false
	s.Refresh()
}

func (s *SeekBar) Disable() {
	if s.disabled {
		return
	}
	s.disabled = true
	s.Refresh()
}

func (s *SeekBar) Disabled() bool { return s.disabled }

func (s *SeekBar) Tapped(e *fyne.PointEvent) {
	if s.disabled {
		return
	}
	s.setFromX(e.Position.X)
	if s.OnChangeEnded != nil {
		s.OnChangeEnded(s.Value)
	}
}

func (s *SeekBar) Dragged(e *fyne.DragEvent) {
	if s.disabled {
		return
	}
	s.setFromX(e.Position.X)
	if s.OnChanged != nil {
		s.OnChanged(s.Value)
	}
}

func (s *SeekBar) DragEnd() {
	if s.disabled {
		return
	}
	if s.OnChangeEnded != nil {
		s.OnChangeEnded(s.Value)
	}
}

func (s *SeekBar) setFromX(x float32) {
	w := s.Size().Width
	if w < 1 {
		return
	}
	ratio := float64(x / w)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	s.SetValue(s.Min + ratio*(s.Max-s.Min))
}

type seekBarRenderer struct {
	bar   *SeekBar
	track *canvas.Rectangle
	fill  *canvas.Rectangle
	thumb *canvas.Circle
}

func (r *seekBarRenderer) Destroy() {}

func (r *seekBarRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.track, r.fill, r.thumb}
}

func (r *seekBarRenderer) MinSize() fyne.Size {
	return fyne.NewSize(seekMinW, seekMinH)
}

func (r *seekBarRenderer) Layout(size fyne.Size) {
	trackY := (size.Height - seekTrackH) / 2
	r.track.Move(fyne.NewPos(0, trackY))
	r.track.Resize(fyne.NewSize(size.Width, seekTrackH))

	span := r.bar.Max - r.bar.Min
	ratio := 0.0
	if span > 0 {
		ratio = (r.bar.Value - r.bar.Min) / span
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	fw := float32(ratio) * size.Width
	r.fill.Move(fyne.NewPos(0, trackY))
	r.fill.Resize(fyne.NewSize(fw, seekTrackH))

	tx := fw - seekThumb/2
	if tx < 0 {
		tx = 0
	}
	if tx > size.Width-seekThumb {
		tx = size.Width - seekThumb
	}
	ty := (size.Height - seekThumb) / 2
	r.thumb.Move(fyne.NewPos(tx, ty))
	r.thumb.Resize(fyne.NewSize(seekThumb, seekThumb))
}

func (r *seekBarRenderer) Refresh() {
	th := theme.Current()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	r.track.FillColor = th.Color(theme.ColorNameInputBackground, v)
	if r.bar.disabled {
		r.fill.FillColor = th.Color(theme.ColorNameDisabled, v)
		r.thumb.FillColor = th.Color(theme.ColorNameDisabled, v)
	} else {
		r.fill.FillColor = th.Color(theme.ColorNamePrimary, v)
		r.thumb.FillColor = th.Color(theme.ColorNameForeground, v)
	}
	r.Layout(r.bar.Size())
	canvas.Refresh(r.bar)
}

// BigButton is a high-importance button with a larger min size for HU taps.
type BigButton struct {
	widget.Button
}

func NewBigButton(label string, icon fyne.Resource, tapped func()) *BigButton {
	b := &BigButton{}
	b.Text = label
	b.Icon = icon
	b.OnTapped = tapped
	b.Importance = widget.HighImportance
	b.ExtendBaseWidget(b)
	return b
}

func (b *BigButton) MinSize() fyne.Size {
	b.ExtendBaseWidget(b)
	m := b.BaseWidget.MinSize()
	if m.Height < 72 {
		m.Height = 72
	}
	if m.Width < 96 {
		m.Width = 96
	}
	return m
}

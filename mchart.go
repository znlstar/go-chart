package chart

import (
	"errors"
	"io"
)

const (
	DefaultLayoutWidth  = 600
	DefaultLayoutHeight = 400
)

type Layout struct {
	rows        int
	columns     int
	totalWidth  int
	totalHeight int
	Boxes       []Box
	Width       int
	Height      int
}

func (this Layout) GetTotalWidth() int {
	return this.totalWidth
}

func (this Layout) GetTotalHeight() int {
	return this.totalHeight
}

func (this Layout) GetColumns() int {
	return this.columns
}

func NewLayout(r, col, width, height int) (layout Layout, err error) {
	if r <= 0 {
		err = errors.New("layout row at least 1")
		return
	}
	if col <= 0 {
		err = errors.New("layout height at least 1")
		return
	}
	if width <= 0 {
		width = DefaultLayoutWidth
	}
	if height <= 0 {
		height = DefaultLayoutHeight
	}

	//calc total width
	totalwidth := col * width
	//calc total height
	totalheight := r * height

	//init box
	layout = Layout{
		rows:        r,
		columns:     col,
		totalWidth:  totalwidth,
		totalHeight: totalheight,
		Width:       width,
		Height:      height,
	}
	for i := 0; i < r; i++ {
		for j := 0; j < col; j++ {
			top := i * height
			left := j * width
			b := Box{
				Top:  top,
				Left: left,
			}
			layout.Boxes = append(layout.Boxes, b)
		}
	}

	return
}

type MChart struct {
	Charts       []Chart
	CanvasLayout Layout
	ColorPalette ColorPalette
	Background   Style
}

func (this *MChart) GetTotalWidth() int {
	return this.CanvasLayout.GetTotalWidth()
}
func (this *MChart) GetTotalHeight() int {
	return this.CanvasLayout.GetTotalHeight()
}

func (this *MChart) GetWidth() int {
	return this.CanvasLayout.Width
}

func (this *MChart) GetHeight() int {
	return this.CanvasLayout.Height
}

func (this *MChart) GetBoxes(i int) Box {
	return this.CanvasLayout.Boxes[i]
}

func (this *MChart) Render(rp RendererProvider, w io.Writer) (err error) {
	if len(this.Charts) == 0 {
		err = errors.New("please provide at least on chart")
		return
	}
	for _, c := range this.Charts {
		if visibleSeriesErr := c.checkHasVisibleSeries(); visibleSeriesErr != nil {
			return visibleSeriesErr
		}
		c.YAxisSecondary.AxisType = YAxisSecondary
	}

	r, err := rp(this.GetTotalWidth(), this.GetTotalHeight())
	if err != nil {
		return
	}
	//draw all canvas
	this.drawBackground(r)

	//loop every chart
	for i, c := range this.Charts {
		//mesure size
		if c.GetWidth() > this.GetWidth() {
			c.Width = this.GetWidth()
		}
		if c.GetHeight() > this.GetHeight() {
			c.Height = this.GetHeight()
		}

		if c.Font == nil {
			defaultFont, err := GetDefaultFont()
			if err != nil {
				return err
			}
			c.defaultFont = defaultFont
		}
		//r.SetDPI(c.GetDPI(DefaultDPI))

		var xt, yt, yta []Tick
		xr, yr, yra := c.getRanges()
		canvasBox := c.getDefaultCanvasBox()
		if true {
			//need add base size
			basetop := this.GetBoxes(i).Top
			baseleft := this.GetBoxes(i).Left
			//fix size
			canvasBox.Top = canvasBox.Top + basetop
			canvasBox.Left = canvasBox.Left + baseleft
			canvasBox.Right = canvasBox.Right + baseleft
			canvasBox.Bottom = canvasBox.Bottom + basetop
		}

		xf, yf, yfa := c.getValueFormatters()

		xr, yr, yra = c.setRangeDomains(canvasBox, xr, yr, yra)

		err = c.checkRanges(xr, yr, yra)
		if err != nil {
			//r.Save(w)
			return err
		}

		if c.hasAxes() {
			xt, yt, yta = c.getAxesTicks(r, xr, yr, yra, xf, yf, yfa)
			canvasBox = c.getMAxesAdjustedCanvasBox(r, canvasBox, xr, yr, yra, xt, yt, yta, this.GetBoxes(i))
			xr, yr, yra = c.setRangeDomains(canvasBox, xr, yr, yra)

			// do a second pass in case things haven't settled yet.
			xt, yt, yta = c.getAxesTicks(r, xr, yr, yra, xf, yf, yfa)
			canvasBox = c.getMAxesAdjustedCanvasBox(r, canvasBox, xr, yr, yra, xt, yt, yta, this.GetBoxes(i))
			xr, yr, yra = c.setRangeDomains(canvasBox, xr, yr, yra)
		}

		if c.hasAnnotationSeries() {
			canvasBox = c.getMAnnotationAdjustedCanvasBox(r, canvasBox, xr, yr, yra, xf, yf, yfa, this.GetBoxes(i))
			xr, yr, yra = c.setRangeDomains(canvasBox, xr, yr, yra)
			xt, yt, yta = c.getAxesTicks(r, xr, yr, yra, xf, yf, yfa)
		}

		c.drawCanvas(r, canvasBox)
		c.drawAxes(r, canvasBox, xr, yr, yra, xt, yt, yta)
		for index, series := range c.Series {
			c.drawSeries(r, canvasBox, xr, yr, yra, series, index)
		}

		c.drawMTitle(r, this.GetBoxes(i))
		for _, a := range c.Elements {
			canvasBox = this.GetBoxes(i)
			a(r, canvasBox, c.styleDefaultsElements())
		}
	}

	return r.Save(w)
}

func (this *MChart) drawBackground(r Renderer) {
	Draw.Box(r, Box{
		Right:  this.GetTotalWidth(),
		Bottom: this.GetTotalHeight(),
	}, this.getBackgroundStyle())
}

func (this *MChart) getBackgroundStyle() Style {
	return this.Background.InheritFrom(this.styleDefaultsBackground())
}

func (this *MChart) styleDefaultsBackground() Style {
	return Style{
		FillColor:   this.GetColorPalette().BackgroundColor(),
		StrokeColor: this.GetColorPalette().BackgroundStrokeColor(),
		StrokeWidth: DefaultBackgroundStrokeWidth,
	}
}

func (this *MChart) GetColorPalette() ColorPalette {
	if this.ColorPalette != nil {
		return this.ColorPalette
	}
	return DefaultColorPalette
}

func (c Chart) getMAxesAdjustedCanvasBox(r Renderer, canvasBox Box, xr, yr, yra Range, xticks, yticks, yticksAlt []Tick, baseBox Box) Box {
	axesOuterBox := canvasBox.Clone()
	if c.XAxis.Style.Show {
		axesBounds := c.XAxis.Measure(r, canvasBox, xr, c.styleDefaultsAxes(), xticks)
		axesOuterBox = axesOuterBox.Grow(axesBounds)
	}
	if c.YAxis.Style.Show {
		axesBounds := c.YAxis.Measure(r, canvasBox, yr, c.styleDefaultsAxes(), yticks)
		axesOuterBox = axesOuterBox.Grow(axesBounds)
	}
	if c.YAxisSecondary.Style.Show {
		axesBounds := c.YAxisSecondary.Measure(r, canvasBox, yra, c.styleDefaultsAxes(), yticksAlt)
		axesOuterBox = axesOuterBox.Grow(axesBounds)
	}
	b := c.Box()
	if true {
		b.Top += baseBox.Top
		b.Left += baseBox.Left
		b.Right += baseBox.Left
		b.Bottom += baseBox.Top
	}
	return canvasBox.OuterConstrain(b, axesOuterBox)
}

func (c Chart) getMAnnotationAdjustedCanvasBox(r Renderer, canvasBox Box, xr, yr, yra Range, xf, yf, yfa ValueFormatter, baseBox Box) Box {
	annotationSeriesBox := canvasBox.Clone()
	for seriesIndex, s := range c.Series {
		if as, isAnnotationSeries := s.(AnnotationSeries); isAnnotationSeries {
			if as.Style.IsZero() || as.Style.Show {
				style := c.styleDefaultsSeries(seriesIndex)
				var annotationBounds Box
				if as.YAxis == YAxisPrimary {
					annotationBounds = as.Measure(r, canvasBox, xr, yr, style)
				} else if as.YAxis == YAxisSecondary {
					annotationBounds = as.Measure(r, canvasBox, xr, yra, style)
				}

				annotationSeriesBox = annotationSeriesBox.Grow(annotationBounds)
			}
		}
	}
	b := c.Box()
	b.Top += baseBox.Top
	b.Left += baseBox.Left
	b.Right += baseBox.Right
	b.Bottom += baseBox.Bottom
	return canvasBox.OuterConstrain(b, annotationSeriesBox)
}

func (c Chart) drawMTitle(r Renderer, canvasBox Box) {
	if len(c.Title) > 0 && c.TitleStyle.Show {
		r.SetFont(c.TitleStyle.GetFont(c.GetFont()))
		r.SetFontColor(c.TitleStyle.GetFontColor(c.GetColorPalette().TextColor()))
		titleFontSize := c.TitleStyle.GetFontSize(DefaultTitleFontSize)
		r.SetFontSize(titleFontSize)

		textBox := r.MeasureText(c.Title)

		textWidth := textBox.Width()
		textHeight := textBox.Height()

		titleX := canvasBox.Left + (c.GetWidth() >> 1) - (textWidth >> 1)
		titleY := canvasBox.Top + c.TitleStyle.Padding.GetTop(DefaultTitleTop) + textHeight

		r.Text(c.Title, titleX, titleY)
	}
}

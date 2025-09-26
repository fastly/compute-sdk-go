package imageopto

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Region string

const (
	RegionUsWest    Region = "us_west"
	RegionUsCentral Region = "us_central"
	RegionUsEast    Region = "us_east"
	RegionEuWest    Region = "eu_west"
	RegionEuCentral Region = "eu_central"
	RegionAsia      Region = "asia"
	RegionAustralia Region = "australia"
)

type Format string

const (
	FormatAuto   Format = "auto"
	FormatAVIF   Format = "avif"
	FormatGIF    Format = "gif"
	FormatJPEG   Format = "jpeg"
	FormatJPEGXL Format = "jpegxl"
	FormatMP4    Format = "mp4"
	FormatPNG    Format = "png"
	FormatWebP   Format = "webp"
)

func (f Format) IsSet() bool    { return f != "" }
func (f Format) String() string { return string(f) }

type Auto string

const (
	AutoAVIF Auto = "avif"
	AutoWebP Auto = "webp"
)

func (a Auto) IsSet() bool    { return a != "" }
func (a Auto) String() string { return string(a) }

func fmtFloat(f float64) string {
	nearest3 := math.Round(1000.0*f) / 1000.0
	if _, fract := math.Modf(nearest3); fract > 0.0 {
		return fmt.Sprintf("%v", nearest3)
	}
	return fmt.Sprintf("%v", int32(f))
}

type PixelsOrPercentageState uint8

const (
	PixelsOrPercentageStateNone PixelsOrPercentageState = iota
	PixelsOrPercentageStatePixels
	PixelsOrPercentageStatePercentage
)

type PixelsOrPercentage struct {
	percent float64
	pixels  uint32
	state   PixelsOrPercentageState
}

func (p *PixelsOrPercentage) IsSet() bool { return p.state != PixelsOrPercentageStateNone }

func (p *PixelsOrPercentage) String() string {
	switch p.state {
	case PixelsOrPercentageStateNone:
		return ""
	case PixelsOrPercentageStatePercentage:
		return fmtFloat(p.percent) + "p"
	case PixelsOrPercentageStatePixels:
		return strconv.Itoa(int(p.pixels))
	}

	return "error"
}

func NewPixelsOrPercentagePixels(p uint32) PixelsOrPercentage {
	return PixelsOrPercentage{
		pixels: p,
		state:  PixelsOrPercentageStatePixels,
	}
}

func NewPixelsOrPercentagePercent(p float64) PixelsOrPercentage {
	return PixelsOrPercentage{
		percent: p,
		state:   PixelsOrPercentageStatePercentage,
	}
}

type CropMode string

const (
	CropModeSafe  = "safe"
	CropModeSmart = "smart"
)

func (c CropMode) IsSet() bool    { return c != "" }
func (c CropMode) String() string { return string(c) }

type AreaState uint8

const (
	AreaStateNone AreaState = iota
	AreaStateAspectRatio
	AreaStateWidthHeight
)

type areaAspectRatio struct{ w, h uint32 }
type areaWidthHeight struct{ w, h PixelsOrPercentage }

type Area struct {
	aspect      areaAspectRatio
	widthHeight areaWidthHeight
	state       AreaState
}

func (a *Area) IsSet() bool { return a.state != AreaStateNone }

func NewAreaAspectRatio(w, h uint32) Area {
	return Area{
		aspect: areaAspectRatio{w, h},
		state:  AreaStateAspectRatio,
	}
}

func NewAreaWidthHeight(w, h PixelsOrPercentage) Area {
	return Area{
		widthHeight: areaWidthHeight{w, h},
		state:       AreaStateWidthHeight,
	}
}

func (area *Area) String() string {
	switch area.state {
	case AreaStateNone:
		return ""
	case AreaStateAspectRatio:
		return fmt.Sprintf("%v:%v", area.aspect.w, area.aspect.h)
	case AreaStateWidthHeight:
		return fmt.Sprintf("%v,%v", area.widthHeight.w.String(), area.widthHeight.h.String())
	}
	return "error"
}

type PointOrOffsetState uint8

const (
	PointOrOffsetStateNone PointOrOffsetState = iota
	PointOrOffsetStatePoint
	PointOrOffsetStateOffset
)

type PointOrOffset struct {
	point  PixelsOrPercentage
	offset uint32 // percentage
	state  PointOrOffsetState
}

func (p *PointOrOffset) IsSet() bool { return p.state != PointOrOffsetStateNone }

func (p *PointOrOffset) ToString(xy string) string {
	switch p.state {
	case PointOrOffsetStateNone:
		return ""
	case PointOrOffsetStatePoint:
		return xy + p.point.String()
	case PointOrOffsetStateOffset:
		return "offset-" + xy + strconv.Itoa(int(p.offset))
	}

	return "error"
}

func NewPointOrOffsetPoint(p PixelsOrPercentage) PointOrOffset {
	return PointOrOffset{
		point: p,
		state: PointOrOffsetStatePoint,
	}
}

func NewPointOrOffsetOffset(p uint32) PointOrOffset {
	return PointOrOffset{
		offset: p,
		state:  PointOrOffsetStateOffset,
	}
}

type Position struct {
	X PointOrOffset
	Y PointOrOffset
}

func (p *Position) String() string {
	if p.X.IsSet() && p.Y.IsSet() {
		return p.X.ToString("x") + "," + p.Y.ToString("y")
	}

	if p.X.IsSet() {
		return p.X.ToString("x")
	}

	return p.Y.ToString("y")
}

type Crop struct {
	Size     Area
	Position *Position
	Mode     CropMode
}

func (c *Crop) String() string {
	s := c.Size.String()

	if c.Position != nil {
		s += "," + c.Position.String()
	}

	if c.Mode.IsSet() {
		s += "," + c.Mode.String()
	}

	return s
}

type Sides struct {
	Top    PixelsOrPercentage
	Right  PixelsOrPercentage
	Bottom PixelsOrPercentage
	Left   PixelsOrPercentage
}

func (s *Sides) String() string {
	return fmt.Sprintf("%v,%v,%v,%v", s.Top.String(), s.Right.String(), s.Bottom.String(), s.Left.String())
}

type OptimizeLevel string

const (
	OptimizeLevelLow    OptimizeLevel = "low"
	OptimizeLevelMedium OptimizeLevel = "medium"
	OptimizeLevelHigh   OptimizeLevel = "high"
)

func (o OptimizeLevel) IsSet() bool    { return o != "" }
func (o OptimizeLevel) String() string { return string(o) }

type Orientation int

const (
	OrientationDefault                   Orientation = 1
	OrientationFlipHorizontal            Orientation = 2
	OrientationFlipHorizontalAndVertical Orientation = 3
	OrientationFlipVertical              Orientation = 4
	OrientationFlipHorizontalOrientLeft  Orientation = 5
	OrientationOrientRight               Orientation = 6
	OrientationFlipHorizontalOrientRight Orientation = 7
	OrientationOrientLeft                Orientation = 8
)

func (o Orientation) IsSet() bool    { return o != 0 }
func (o Orientation) String() string { return strconv.Itoa(int(o)) }

type HexColor struct {
	R, G, B uint8
	A       float32
}

func (h *HexColor) String() string {
	return fmt.Sprintf("%v,%v,%v,%v", h.R, h.G, h.B, h.A)

}

type TrimColor struct {
	Color     HexColor
	Threshold float32
}

func (t *TrimColor) String() string {
	if t.Threshold != 0 {
		return t.Color.String() + ",t" + fmtFloat(float64(t.Threshold))
	}

	return t.Color.String()
}

type BWModeState int

const (
	BWModeStateNone BWModeState = iota
	BWModeStateDefaultThreshold
	BWModeStateAtkinson
	BWModeStateThreshold
)

type BWMode struct {
	state     BWModeState
	luminance uint32
}

func (bw *BWMode) IsSet() bool { return bw.state != BWModeStateNone }

func NewBWModeDefaultThreshold() BWMode {
	return BWMode{state: BWModeStateDefaultThreshold}
}

func NewBWModeAtkinson() BWMode {
	return BWMode{state: BWModeStateAtkinson}
}

func NewBWModeThreshold(luminance uint32) BWMode {
	return BWMode{state: BWModeStateThreshold, luminance: luminance}
}

func (bw *BWMode) String() string {
	switch bw.state {
	case BWModeStateNone:
		return ""
	case BWModeStateAtkinson:
		return "atkinson"
	case BWModeStateDefaultThreshold:
		return "threshold"
	case BWModeStateThreshold:
		return "threshold," + strconv.Itoa(int(bw.luminance))
	}

	return "error"
}

type BlurModeState int

const (
	BlurModeStateNone       BlurModeState = 0
	BlurModeStatePixels     BlurModeState = 1
	BlurModeStatePercentage BlurModeState = 2
)

type BlurMode struct {
	state      BlurModeState
	pixels     float64
	percentage float64
}

func (b *BlurMode) IsSet() bool { return b.state != BlurModeStateNone }

func (b *BlurMode) String() string {
	switch b.state {
	case BlurModeStateNone:
		return ""
	case BlurModeStatePercentage:
		return fmt.Sprintf("%vp", b.percentage)
	case BlurModeStatePixels:
		return fmt.Sprintf("%v", b.pixels)
	}

	return "error"
}

func NewBlurModePixels(p float64) BlurMode {
	return BlurMode{
		pixels: p,
		state:  BlurModeStatePixels,
	}
}

func NewBlurModePercentage(p float64) BlurMode {
	return BlurMode{
		percentage: p,
		state:      BlurModeStatePercentage,
	}
}

type Canvas struct {
	Size     Area
	Position *Position
}

func (c *Canvas) String() string {
	if c.Position != nil {
		return c.Size.String() + "," + c.Position.String()
	}

	return c.Size.String()
}

type Fit string

const (
	FitBounds Fit = "bounds"
	FitCover  Fit = "cover"
	FitCrop   Fit = "crop"
)

func (f Fit) IsSet() bool    { return f != "" }
func (f Fit) String() string { return string(f) }

type Level string

const (
	Level1_0 Level = "1.0"
	Level1_1 Level = "1.1"
	Level1_2 Level = "1.2"
	Level1_3 Level = "1.3"
	Level2_0 Level = "2.0"
	Level2_1 Level = "2.1"
	Level2_2 Level = "2.2"
	Level3_0 Level = "3.0"
	Level3_1 Level = "3.1"
	Level3_2 Level = "3.2"
	Level4_0 Level = "4.0"
	Level4_1 Level = "4.1"
	Level4_2 Level = "4.2"
	Level5_0 Level = "5.0"
	Level5_1 Level = "5.1"
	Level5_2 Level = "5.2"
	Level6_0 Level = "6.0"
	Level6_1 Level = "6.1"
	Level6_2 Level = "6.2"
)

func (l Level) IsSet() bool    { return l != "" }
func (l Level) String() string { return string(l) }

type Metadata string

const (
	MetadataCopyright Metadata = "copyright"
)

func (m Metadata) IsSet() bool    { return m != "" }
func (m Metadata) String() string { return string(m) }

type Profile string

const (
	ProfileBaseline Profile = "baseline"
	ProfileMain     Profile = "main"
	ProfileHigh     Profile = "high"
)

func (p Profile) IsSet() bool    { return p != "" }
func (p Profile) String() string { return string(p) }

type ResizeAlgorithm string

const (
	ResizeAlgorithmNearest  ResizeAlgorithm = "nearest"
	ResizeAlgorithmBilinear ResizeAlgorithm = "bilinear"
	ResizeAlgorithmBicubic  ResizeAlgorithm = "bicubic"
	ResizeAlgorithmLanczos2 ResizeAlgorithm = "lanczos2"
	ResizeAlgorithmLanczos3 ResizeAlgorithm = "lanczos3"
)

func (r ResizeAlgorithm) IsSet() bool    { return r != "" }
func (r ResizeAlgorithm) String() string { return string(r) }

type Sharpen struct {
	Amount    uint8
	Radius    float32
	Threshold uint8
}

func (s *Sharpen) String() string {
	return fmt.Sprintf("a%v,r%v,t%v", s.Amount, fmtFloat(float64(s.Radius)), s.Threshold)
}

type EnableOpt string

const (
	EnableOptUpscale EnableOpt = "upscale"
)

func (e EnableOpt) IsSet() bool    { return e != "" }
func (e EnableOpt) String() string { return string(e) }

type Opts struct {
	Region                             Region
	PreserveQueryStringOnOriginRequest bool
	Auto                               Auto
	BgColor                            *HexColor
	Blur                               BlurMode
	Brightness                         int
	Bw                                 BWMode
	Canvas                             *Canvas
	Contrast                           int
	Crop                               *Crop
	Dpr                                float32
	Enable                             EnableOpt
	Fit                                Fit
	Format                             Format
	Frame                              uint32
	Height                             PixelsOrPercentage
	Level                              Level
	Metadata                           Metadata
	Optimize                           OptimizeLevel
	Orient                             Orientation
	Pad                                *Sides
	Precrop                            *Crop
	Profile                            Profile
	Quality                            uint32
	ResizeFilter                       ResizeAlgorithm
	Saturation                         int
	Sharpen                            *Sharpen
	Trim                               *Sides
	TrimColor                          *TrimColor
	Width                              PixelsOrPercentage
}

func encodeCommas(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, ",", "%2C"), ":", "%3A")
}

func (o *Opts) QueryString() string {
	var args []string

	if o.Region != "" {
		args = append(args, "region="+string(o.Region))
	}

	if o.Auto.IsSet() {
		args = append(args, "auto="+o.Auto.String())
	}

	if o.BgColor != nil {
		args = append(args, "bg-color="+encodeCommas(o.BgColor.String()))
	}

	if o.Blur.IsSet() {
		args = append(args, "blur="+o.Blur.String())
	}

	if o.Brightness != 0 {
		args = append(args, "brightness="+strconv.Itoa(o.Brightness))
	}

	if o.Bw.IsSet() {
		args = append(args, "bw="+o.Bw.String())
	}

	if o.Canvas != nil {
		args = append(args, "canvas="+encodeCommas(o.Canvas.String()))
	}

	if o.Contrast != 0 {
		args = append(args, "constrast="+strconv.Itoa(o.Contrast))
	}

	if o.Crop != nil {
		args = append(args, "crop="+encodeCommas(o.Crop.String()))
	}

	if o.Dpr != 0 {
		args = append(args, "dpr="+fmt.Sprintf("%v", o.Dpr))
	}

	if o.Enable.IsSet() {
		args = append(args, "enable="+o.Enable.String())
	}

	if o.Fit.IsSet() {
		args = append(args, "fit="+o.Fit.String())
	}

	if o.Format.IsSet() {
		args = append(args, "format="+o.Format.String())
	}

	if o.Frame != 0 {
		args = append(args, "frame="+strconv.Itoa(int(o.Frame)))
	}

	if o.Height.IsSet() {
		args = append(args, "height="+o.Height.String())
	}

	if o.Level.IsSet() {
		args = append(args, "level="+o.Level.String())
	}

	if o.Profile.IsSet() {
		args = append(args, "profile="+o.Profile.String())
	}

	if o.Metadata.IsSet() {
		args = append(args, "metadata="+o.Metadata.String())
	}

	if o.Optimize.IsSet() {
		args = append(args, "optimize="+o.Optimize.String())
	}

	if o.Orient.IsSet() {
		args = append(args, "orient="+o.Orient.String())
	}

	if o.Pad != nil {
		args = append(args, "pad="+encodeCommas(o.Pad.String()))
	}

	if o.Precrop != nil {
		args = append(args, "precrop="+encodeCommas(o.Precrop.String()))
	}

	if o.ResizeFilter.IsSet() {
		args = append(args, "resize-filter="+o.ResizeFilter.String())
	}

	if o.Sharpen != nil {
		args = append(args, "sharpen="+encodeCommas(o.Sharpen.String()))
	}

	if o.Trim != nil {
		args = append(args, "trim="+encodeCommas(o.Trim.String()))
	}

	if o.TrimColor != nil {
		args = append(args, "trim-color="+encodeCommas(o.TrimColor.String()))
	}

	if o.Width.IsSet() {
		args = append(args, "width="+o.Width.String())
	}

	return strings.Join(args, "&")
}

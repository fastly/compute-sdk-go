package imageopto

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Region sets the region for image transformation.
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

func (r Region) isSet() bool { return r != "" }

// Format is the desired output encoding for the image.
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

func (f Format) isSet() bool    { return f != "" }
func (f Format) String() string { return string(f) }

// Auto enables optimizations based on content negotiation.
//
// This functionality is also possible by setting Format to FormatAuto,
// which will additionally support JPEGXL output.
type Auto string

const (
	AutoAVIF Auto = "avif"
	AutoWebP Auto = "webp"
)

func (a Auto) isSet() bool    { return a != "" }
func (a Auto) String() string { return string(a) }

func fmtFloat(f float64) string {
	nearest3 := math.Round(1000.0*f) / 1000.0
	if _, fract := math.Modf(nearest3); fract > 0.0 {
		return fmt.Sprintf("%v", nearest3)
	}
	return fmt.Sprintf("%v", int32(f))
}

// PixelsOrPercentage specifies a position along an exist.
//
// Valid values are either an integer number of pixels, or a percentage followed by `p`.
type PixelsOrPercentage string

func (p PixelsOrPercentage) isSet() bool    { return p != "" }
func (p PixelsOrPercentage) String() string { return string(p) }

func (p PixelsOrPercentage) validate() error {
	if !rePixelsOrPercentage.MatchString(string(p)) {
		return fmt.Errorf("imageopto: pixels or percentage: invalid: %q", string(p))
	}

	return nil
}

var rePixelsOrPercentage = regexp.MustCompile("^[0-9]+(.[0-9]+)?p?$")

// CropMode determines cropping behavior.
type CropMode string

const (
	// CropModeSafe avoids invalid parameter errors and return an image.
	//
	// By default, if the specified cropped region is outside the bounds of the image, then the
	// transformation will fail with the error "Invalid transformation for requested image:
	// Invalid crop, region out of bounds". This option will instead deliver the image as an
	// intersection of the origin image and the specified cropped region. This avoids the error,
	// but the resulting image may not be of the specified dimensions.
	CropModeSafe = "safe"

	// CropModeSmart enables content-aware algorithms to attempt to crop the image to the desired aspect ratio
	// while intelligently focusing on the most important visual content, including the detection
	// of faces.
	CropModeSmart = "smart"
)

func (c CropMode) isSet() bool    { return c != "" }
func (c CropMode) String() string { return string(c) }

type areaState uint8

const (
	areaStateNone areaState = iota
	areaStateAspectRatio
	areaStateWidthHeight
)

type areaAspectRatio struct{ w, h uint32 }
type areaWidthHeight struct{ w, h PixelsOrPercentage }

// Area is an image area.
type Area struct {
	aspect      areaAspectRatio
	widthHeight areaWidthHeight
	state       areaState
}

// NewAreaAspectRatio specifies an area with the given width/height aspect ratio.
func NewAreaAspectRatio(w, h uint32) Area {
	return Area{
		aspect: areaAspectRatio{w, h},
		state:  areaStateAspectRatio,
	}
}

// NewAreaWidthHeight specifies an area with precise number of pixels.
func NewAreaWidthHeight(w, h PixelsOrPercentage) Area {
	return Area{
		widthHeight: areaWidthHeight{w, h},
		state:       areaStateWidthHeight,
	}
}

func (a *Area) validate() error {
	if a.state == areaStateWidthHeight {
		if a.widthHeight.w.isSet() {
			if err := a.widthHeight.w.validate(); err != nil {
				return err
			}
		}
		if a.widthHeight.h.isSet() {
			if err := a.widthHeight.h.validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (area *Area) String() string {
	switch area.state {
	case areaStateNone:
		return ""
	case areaStateAspectRatio:
		return fmt.Sprintf("%v:%v", area.aspect.w, area.aspect.h)
	case areaStateWidthHeight:
		return fmt.Sprintf("%v,%v", area.widthHeight.w.String(), area.widthHeight.h.String())
	}
	return "error"
}

// PointOrOffset specifies a position along an axis
type PointOrOffset string

func (p PointOrOffset) isSet() bool    { return p != "" }
func (p PointOrOffset) String() string { return string(p) }

// Position is a position within an image.
//
// Valid values:
//
//	x{x},y{y}
//	offset-x{offset-x},offset-y{offset-y}
//	x{x},offset-y{offset-y}
//	offset-x{offset-x},y{y}
type Position struct {
	X PointOrOffset
	Y PointOrOffset
}

func (p *Position) String() string {
	if p.X.isSet() && p.Y.isSet() {
		return p.X.String() + "," + p.Y.String()
	}

	if p.X.isSet() {
		return p.X.String()
	}

	return p.Y.String()
}

var reXPosition = regexp.MustCompile("^(x[0-9]+p?)|(offset-x[0-9]+)$")
var reYPosition = regexp.MustCompile("^(y[0-9]+p?)|(offset-y[0-9]+)$")

func (p *Position) validate() error {
	if p.X != "" && !reXPosition.MatchString(string(p.X)) {
		return fmt.Errorf("imageopto: position.x : invalid %q", p.X)
	}

	if p.Y != "" && !reYPosition.MatchString(string(p.Y)) {
		return fmt.Errorf("imageopto: position.y: invalid %q", p.Y)
	}

	return nil
}

// Crop removes pixels from an image.
type Crop struct {
	// Size is the desired width and height.
	Size Area
	// Position is the offset for determining the starting position of the cropped region.
	Position *Position
	// Mode is the crop mode to use.
	Mode CropMode
}

func (c *Crop) validate() error {
	if err := c.Size.validate(); err != nil {
		return err
	}
	if c.Position != nil {
		if err := c.Position.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Crop) String() string {
	s := c.Size.String()

	if c.Position != nil {
		s += "," + c.Position.String()
	}

	if c.Mode.isSet() {
		s += "," + c.Mode.String()
	}

	return s
}

// Sides specifies a border around an image for adding or removing pixels.
type Sides struct {
	Top    PixelsOrPercentage
	Right  PixelsOrPercentage
	Bottom PixelsOrPercentage
	Left   PixelsOrPercentage
}

func (s *Sides) String() string {
	return fmt.Sprintf("%v,%v,%v,%v", s.Top.String(), s.Right.String(), s.Bottom.String(), s.Left.String())
}

func (s *Sides) validate() error {
	if s.Top.isSet() {
		if err := s.Top.validate(); err != nil {
			return err
		}
	}

	if s.Right.isSet() {
		if err := s.Right.validate(); err != nil {
			return err
		}
	}
	if s.Bottom.isSet() {
		if err := s.Bottom.validate(); err != nil {
			return err
		}
	}
	if s.Left.isSet() {
		if err := s.Left.validate(); err != nil {
			return err
		}
	}

	return nil
}

// OptimizeLevel specifies the desired level of image compression.
type OptimizeLevel string

const (
	// OptimizeLevelLow means the output mage quality will be similar to the input image quality.
	OptimizeLevelLow OptimizeLevel = "low"

	// OptimizeLevelMedium means more optimization is allowed, while attempting to preserve the visual
	// quality of the input image.
	OptimizeLevelMedium OptimizeLevel = "medium"

	// OptimizeLevelHigh means minor visual artifacts may be visible. This produces the smallest file.
	OptimizeLevelHigh OptimizeLevel = "high"
)

func (o OptimizeLevel) isSet() bool    { return o != "" }
func (o OptimizeLevel) String() string { return string(o) }

// / Orientation specifies the cardinal orientation of the image.
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

func (o Orientation) isSet() bool    { return o != 0 }
func (o Orientation) String() string { return strconv.Itoa(int(o)) }

// HexColor is a hex color
type HexColor struct {
	// R, G, B are the amount of Red, Green, and Blue.
	R, G, B uint8
	// A is the alpha and should range from 0 (transparent) to 1 (opaque).
	A float32
}

func (h *HexColor) String() string {
	return fmt.Sprintf("%v,%v,%v,%v", h.R, h.G, h.B, h.A)

}

func (h *HexColor) validate() error {
	if h.A < 0.0 || h.A > 1.0 {
		return errors.New("imageopto: hexcolor alpha out of range")
	}

	return nil
}

// TrimColor identifies a rectangular border based on specified color and removes this
// / border from the edges of an image.
type TrimColor struct {
	// Color is the color to trim
	Color HexColor
	// Threshold specifies how different a color can be from the given trim color and still be trimmed.
	//
	// Valid values are 0 (exact color only) to 1 (trim everything).
	Threshold float32
}

func (t *TrimColor) String() string {
	if t.Threshold != 0 {
		return t.Color.String() + ",t" + fmtFloat(float64(t.Threshold))
	}

	return t.Color.String()
}

func (t *TrimColor) validate() error {
	if err := t.Color.validate(); err != nil {
		return err
	}

	if t.Threshold < 0.0 || t.Threshold > 1.0 {
		return errors.New("imageopto: trimcolor threshold out of range 0.0 .. 1.0")
	}

	return nil
}

type bwModeState int

const (
	bwModeStateNone bwModeState = iota
	bwModeStateDefaultThreshold
	bwModeStateAtkinson
	bwModeStateThreshold
)

// BWMode specifies how the image should be converted to black and white duotone.
type BWMode struct {
	state     bwModeState
	luminance uint32
}

func (bw *BWMode) isSet() bool { return bw.state != bwModeStateNone }

// NewBWModeDefaultThreshold returns a BWMode for default threshold dithering.
func NewBWModeDefaultThreshold() BWMode {
	return BWMode{state: bwModeStateDefaultThreshold}
}

// NewBWModeAtkinston returns a BWMode for Atkinson dithering.
func NewBWModeAtkinson() BWMode {
	return BWMode{state: bwModeStateAtkinson}
}

// NewBWModeThreshold returns a BWMode for default threshold dithering with a specified luminance.
func NewBWModeThreshold(luminance uint32) BWMode {
	return BWMode{state: bwModeStateThreshold, luminance: luminance}
}

func (bw *BWMode) String() string {
	switch bw.state {
	case bwModeStateNone:
		return ""
	case bwModeStateAtkinson:
		return "atkinson"
	case bwModeStateDefaultThreshold:
		return "threshold"
	case bwModeStateThreshold:
		return "threshold," + strconv.Itoa(int(bw.luminance))
	}

	return "error"
}

type blurModeState int

const (
	blurModeStateNone       blurModeState = 0
	blurModeStatePixels     blurModeState = 1
	blurModeStatePercentage blurModeState = 2
)

// BlurMode specifies the Gaussian Blur to apply to the image.
type BlurMode struct {
	state      blurModeState
	pixels     float64
	percentage float64
}

func (b *BlurMode) isSet() bool { return b.state != blurModeStateNone }

func (b *BlurMode) String() string {
	switch b.state {
	case blurModeStateNone:
		return ""
	case blurModeStatePercentage:
		return fmt.Sprintf("%vp", b.percentage)
	case blurModeStatePixels:
		return fmt.Sprintf("%v", b.pixels)
	}

	return "error"
}

// NewBlurModePixels returns a new BlurMode with a blur radius in pixels.
func NewBlurModePixels(p float64) BlurMode {
	return BlurMode{
		pixels: p,
		state:  blurModeStatePixels,
	}
}

// NewBlurModePercentage returns a new BlurMode with a blur radius as a percentage of the image size.
func NewBlurModePercentage(p float64) BlurMode {
	return BlurMode{
		percentage: p,
		state:      blurModeStatePercentage,
	}
}

// Canvas is the size of an image canvas.
type Canvas struct {
	// Size is the desired width and height.
	Size Area

	// Position is how to distribute the remaining space around the image.
	Position *Position
}

func (c *Canvas) validate() error {
	if err := c.Size.validate(); err != nil {
		return nil
	}
	if c.Position != nil {
		if err := c.Position.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Canvas) String() string {
	if c.Position != nil {
		return c.Size.String() + "," + c.Position.String()
	}
	return c.Size.String()
}

// Fit controls how the image will be constrained within
// the provided size (width and height) values, in order to maintain
// the correct proportions.
type Fit string

const (
	// FitBounds resizes the image to fit entirely within the specified region, making
	// one dimension smaller if needed.
	FitBounds Fit = "bounds"
	// FitCover resizes the image to entirely cover the specified region, making one
	// dimension larger if needed.
	FitCover Fit = "cover"
	// FitCrop resizes and crops the image centrally to exactly fit the specified region.
	FitCrop Fit = "crop"
)

func (f Fit) isSet() bool    { return f != "" }
func (f Fit) String() string { return string(f) }

// Level specifies a set of constraints indicating a degree of required decoder performance] for a profile.
//
// This option is only used when converting animated GIFs to the MP4 format and when used in
// conjunction with the profile parameter,
//
// See https://en.wikipedia.org/wiki/Advanced_Video_Coding#Levels
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

func (l Level) isSet() bool    { return l != "" }
func (l Level) String() string { return string(l) }

// Metadata controls what metadata to preserve in the image.
type Metadata string

const (
	/// MetadataCopyright preserves copyright notice, creator, credit line, licensor, and web statement of rights
	/// fields.
	MetadataCopyright Metadata = "copyright"
)

func (m Metadata) isSet() bool    { return m != "" }
func (m Metadata) String() string { return string(m) }

// Profile specifies features the video encoder can use based on a target class
// of application for decoding the specific video bitstream.
//
// This option is only used when converting animated GIFs to the MP4 format and when used in
// conjunction with the level parameter,
type Profile string

const (
	/// ProfileBaseline is the profile recommended for video conferencing and mobile applications.
	ProfileBaseline Profile = "baseline"
	/// ProfileMain is the profile recommended for standard-definition broadcasts.
	ProfileMain Profile = "main"
	/// ProfileHigh is the profile recommended for high-definition broadcasts.
	ProfileHigh Profile = "high"
)

func (p Profile) isSet() bool    { return p != "" }
func (p Profile) String() string { return string(p) }

// / ResizeAlgorithm specifies the resizing filter used to generate a new image with a higher or lower
// / number of pixels.
type ResizeAlgorithm string

const (
	// ResizeAlgorithmNearest uses the value of nearby translated pixel values.
	ResizeAlgorithmNearest ResizeAlgorithm = "nearest"
	// ResizeAlgorithmBilear uses an average of a 2x2 environment of pixels.
	ResizeAlgorithmBilinear ResizeAlgorithm = "bilinear"
	// ResizeAlgorithmBicubic uses an average of a 4x4 environment of pixels, weighing the innermost pixels higher.
	ResizeAlgorithmBicubic ResizeAlgorithm = "bicubic"
	// ResizeAlgorithmLanczos2 uses the Lanczos filter to increase the ability to detect edges and linear features within
	// an image and uses sinc resampling to provide the best possible reconstruction.
	ResizeAlgorithmLanczos2 ResizeAlgorithm = "lanczos2"
	// ResizeAlgorithmLanczos3 uses a better approximation of the sinc resampling function.
	ResizeAlgorithmLanczos3 ResizeAlgorithm = "lanczos3"
)

func (r ResizeAlgorithm) isSet() bool    { return r != "" }
func (r ResizeAlgorithm) String() string { return string(r) }

// Sharpen specifies options for increasing the definition of the edges of objects in an image.
type Sharpen struct {
	// Amount parameter for the unsharp mask.
	Amount uint8
	// Radius parameter for the unsharp mask.
	Radius float32
	// Threshold parameter for the unsharp mask.
	Threshold uint8
}

func (s *Sharpen) String() string {
	return fmt.Sprintf("a%v,r%v,t%v", s.Amount, fmtFloat(float64(s.Radius)), s.Threshold)
}

func (s *Sharpen) validate() error {
	if s.Amount > 10 {
		return errors.New("imageopto: sharpen amount out of range 0 .. 10")
	}
	if s.Radius < 0.5 || s.Radius > 1000 {
		return errors.New("imageopto: sharpen radius out of range 0.5 .. 1000")
	}

	return nil
}

// EnableOpt specifies features that are disabled by default, but can be requested to be enabled.
type EnableOpt string

const (
	/// EnableOptUpsace allow images to be resized such that the output image's dimensions are
	/// larger than the source image.
	EnableOptUpscale EnableOpt = "upscale"
)

func (e EnableOpt) isSet() bool    { return e != "" }
func (e EnableOpt) String() string { return string(e) }

// Opts contains options that correspond to the public ImageOptimzation API.
//
// For more details, see https://www.fastly.com/documentation/reference/io/
type Opts struct {
	// Region indicates where image transformations will occur. Must be set.
	//
	// The chosen region should be close to your origin.
	Region Region

	// Auto requests an output format based on the `Accept` header.
	Auto Auto

	// BgColor Sets a background color when replacing transparent pixels
	// or with `pad` or `canvas`.
	BgColor *HexColor

	// Blur blurs the image.
	Blur BlurMode

	// Brightness adjusts image brightness.
	Brightness int

	// Bw converts image to black and white duotone.
	Bw BWMode

	// Canvas adds a canvas surrounding the image.
	Canvas *Canvas

	// Contrast adjusts image contrast.
	Contrast int

	/// Crop crops the image.
	Crop *Crop

	// Dps is the ratio between physical and logical pixels, a float between 0-10.
	//
	// Adjusts any resize requests according to this ratio.
	Dpr float32

	// Enable Allows for various image transformations to occur, particularly upscaling images.
	Enable EnableOpt

	// Fit describes how the image should fit within the requested dimensions.
	Fit Fit

	// Format re-encodes the image to a given output format.
	Format Format

	// Frame requests a single frame for an animated image.
	//
	// Currently only supported for animated GIFs.
	Frame uint32

	// Height adjusts image height.
	Height PixelsOrPercentage

	// Level configures GIF to MP4 conversions.
	//
	// See <https://www.fastly.com/documentation/reference/io/level/> for detailed information.
	Level Level

	// Metadata preserves metadata on the input image.
	Metadata Metadata

	// Optimize attempts to select an output quality to optimize the image.
	Optimize OptimizeLevel

	// Orient controls how the image will be oriented.
	Orient Orientation

	// Pad adds pixels surrounding the image.
	Pad *Sides

	// Precrop applies the crop instruction before all other parameters.
	Precrop *Crop

	// Profile is For use with GIF to MP4 conversions.
	//
	// See <https://www.fastly.com/documentation/reference/io/level/> for detailed information.
	Profile Profile

	// Quality requests an output quality.
	Quality uint32

	// ResizeFilter specifies which algorithm to use when resizing an image.

	ResizeFilter ResizeAlgorithm

	// Saturation adjusts image saturation.
	Saturation int

	// Sharpen applies an unsharp mask to the image.
	Sharpen *Sharpen

	// Trim removes pixels from the image on all sides.
	Trim *Sides

	// TrimColor trims the image on all sides based on a given color.
	TrimColor *TrimColor

	// Width adjusts image width.
	Width PixelsOrPercentage

	/// If true, preserves query parameters not belonging to the
	/// Image Optimizer API when requesting the origin image.
	PreserveQueryStringOnOriginRequest bool
}

func (o *Opts) validateParams() error {

	if !o.Region.isSet() {
		return errors.New("imageopto: region is not set")
	}

	if b := o.BgColor; b != nil {
		if b.A < 0.0 || b.A > 1.0 {
			return errors.New("imageopto: alpha out of range 0..1")
		}
	}

	if b := o.Brightness; b != 0 {
		if b < -100 || b > 100 {
			return errors.New("imageopto: brightness out of range -100 .. 100")
		}
	}

	if c := o.Contrast; c != 0 {
		if c < -100 || c > 100 {
			return errors.New("imageopto: contrast out of range -100 .. 100")
		}
	}

	if c := o.Canvas; c != nil {
		if err := c.validate(); err != nil {
			return err
		}
	}

	if c := o.Crop; c != nil {
		if err := c.validate(); err != nil {
			return err
		}
	}

	if d := o.Dpr; d != 0 {
		if d < 0 || d > 10 {
			return errors.New("imageopto: dpr out of range 0 .. 10")
		}
	}

	if h := o.Height; h.isSet() {
		if err := h.validate(); err != nil {
			return err

		}
	}

	if p := o.Pad; p != nil {
		if err := p.validate(); err != nil {
			return err

		}
	}

	if c := o.Precrop; c != nil {
		if err := c.validate(); err != nil {
			return err
		}
	}

	if q := o.Quality; q != 0 {
		if q > 100 {
			return errors.New("imageopto: quality out of range 0 .. 10")
		}
	}

	if s := o.Saturation; s != 0 {
		if s < -100 || s > 100 {
			return errors.New("imageopto: saturation out of range -100 .. 100")
		}
	}

	if s := o.Sharpen; s != nil {
		if err := s.validate(); err != nil {
			return err
		}
	}

	if t := o.Trim; t != nil {
		if err := t.validate(); err != nil {
			return err

		}
	}

	if t := o.TrimColor; t != nil {
		if err := t.validate(); err != nil {
			return err
		}
	}

	if w := o.Width; w.isSet() {
		if err := w.validate(); err != nil {
			return err

		}
	}

	return nil
}

func (o *Opts) QueryString() (string, error) {
	if err := o.validateParams(); err != nil {
		return "", err
	}

	var args []string

	if o.Region != "" {
		args = append(args, "region="+string(o.Region))
	}

	if o.Auto.isSet() {
		args = append(args, "auto="+o.Auto.String())
	}

	if o.BgColor != nil {
		args = append(args, "bg-color="+o.BgColor.String())
	}

	if o.Blur.isSet() {
		args = append(args, "blur="+o.Blur.String())
	}

	if o.Brightness != 0 {
		args = append(args, "brightness="+strconv.Itoa(o.Brightness))
	}

	if o.Bw.isSet() {
		args = append(args, "bw="+o.Bw.String())
	}

	if o.Canvas != nil {
		args = append(args, "canvas="+o.Canvas.String())
	}

	if o.Contrast != 0 {
		args = append(args, "constrast="+strconv.Itoa(o.Contrast))
	}

	if o.Crop != nil {
		args = append(args, "crop="+o.Crop.String())
	}

	if o.Dpr != 0 {
		args = append(args, "dpr="+fmt.Sprintf("%v", o.Dpr))
	}

	if o.Enable.isSet() {
		args = append(args, "enable="+o.Enable.String())
	}

	if o.Fit.isSet() {
		args = append(args, "fit="+o.Fit.String())
	}

	if o.Format.isSet() {
		args = append(args, "format="+o.Format.String())
	}

	if o.Frame != 0 {
		args = append(args, "frame="+strconv.Itoa(int(o.Frame)))
	}

	if o.Height.isSet() {
		args = append(args, "height="+o.Height.String())
	}

	if o.Level.isSet() {
		args = append(args, "level="+o.Level.String())
	}

	if o.Profile.isSet() {
		args = append(args, "profile="+o.Profile.String())
	}

	if o.Metadata.isSet() {
		args = append(args, "metadata="+o.Metadata.String())
	}

	if o.Optimize.isSet() {
		args = append(args, "optimize="+o.Optimize.String())
	}

	if o.Orient.isSet() {
		args = append(args, "orient="+o.Orient.String())
	}

	if o.Pad != nil {
		args = append(args, "pad="+o.Pad.String())
	}

	if o.Precrop != nil {
		args = append(args, "precrop="+o.Precrop.String())
	}

	if o.ResizeFilter.isSet() {
		args = append(args, "resize-filter="+o.ResizeFilter.String())
	}

	if o.Sharpen != nil {
		args = append(args, "sharpen="+o.Sharpen.String())
	}

	if o.Trim != nil {
		args = append(args, "trim="+o.Trim.String())
	}

	if o.TrimColor != nil {
		args = append(args, "trim-color="+o.TrimColor.String())
	}

	if o.Width.isSet() {
		args = append(args, "width="+o.Width.String())
	}

	return strings.Join(args, "&"), nil
}

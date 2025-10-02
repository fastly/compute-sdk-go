package imageopto

import "testing"

func TestOpts(t *testing.T) {

	tests := []struct {
		opts  *Opts
		query string
	}{
		{&Opts{Region: RegionUsEast}, "region=us_east"},
		{&Opts{Region: RegionUsEast, Width: "50p"}, "region=us_east&width=50p"},
		{&Opts{Region: RegionUsEast, Auto: AutoWebP}, "region=us_east&auto=webp"},
		{&Opts{Region: RegionUsEast, BgColor: &HexColor{R: 0, G: 255, B: 0, A: 0.3}}, "region=us_east&bg-color=0,255,0,0.3"},
		{&Opts{Region: RegionUsEast, Blur: NewBlurModePixels(50)}, "region=us_east&blur=50"}, // Rust: region=us_east&blur=50.0
		{&Opts{Region: RegionUsEast, Blur: NewBlurModePercentage(0.8)}, "region=us_east&blur=0.8p"},
		{&Opts{Region: RegionUsEast, Brightness: -50}, "region=us_east&brightness=-50"},
		{&Opts{Region: RegionUsEast, Bw: NewBWModeThreshold(10)}, "region=us_east&bw=threshold,10"},
		{&Opts{Region: RegionUsEast, Contrast: -5}, "region=us_east&constrast=-5"},
		{&Opts{Region: RegionUsEast, Dpr: 3.2}, "region=us_east&dpr=3.2"},
		{&Opts{Region: RegionUsEast, Enable: EnableOptUpscale}, "region=us_east&enable=upscale"},
		{&Opts{Region: RegionUsEast, Format: FormatJPEGXL}, "region=us_east&format=jpegxl"},
		{&Opts{Region: RegionUsEast, Frame: 1}, "region=us_east&frame=1"},
		{&Opts{Region: RegionUsEast, Height: "80p"}, "region=us_east&height=80p"},
		{&Opts{Region: RegionUsEast, Level: Level2_0, Format: FormatMP4, Profile: ProfileHigh}, "region=us_east&format=mp4&level=2.0&profile=high"},
		{&Opts{Region: RegionUsEast, Metadata: MetadataCopyright}, "region=us_east&metadata=copyright"},
		{&Opts{Region: RegionUsEast, Optimize: OptimizeLevelHigh}, "region=us_east&optimize=high"},
		{&Opts{Region: RegionUsEast, Orient: OrientationFlipVertical}, "region=us_east&orient=4"},

		{
			&Opts{
				Region: RegionUsEast,
				Pad: &Sides{
					Top:    "10p",
					Right:  "10p",
					Bottom: "10p",
					Left:   "10p",
				}},
			"region=us_east&pad=10p,10p,10p,10p",
		},

		{&Opts{Region: RegionUsEast, ResizeFilter: ResizeAlgorithmLanczos3}, "region=us_east&resize-filter=lanczos3"},
		{&Opts{Region: RegionUsEast, Sharpen: &Sharpen{Amount: 5, Radius: 2.0, Threshold: 1}}, "region=us_east&sharpen=a5,r2,t1"},

		{
			&Opts{
				Region: RegionUsEast,
				Trim: &Sides{
					Top:    "20.556p",
					Right:  "33.333p",
					Bottom: "20.555p",
					Left:   "33.333p",
				}},
			"region=us_east&trim=20.556p,33.333p,20.555p,33.333p",
		},

		{
			&Opts{Region: RegionUsEast, TrimColor: &TrimColor{
				Color:     HexColor{R: 255, G: 0, B: 0, A: 1.0},
				Threshold: 0.5},
			},
			"region=us_east&trim-color=255,0,0,1,t0.5", // Rust: "region=us_east&trim-color=255,0,0,1.0,t0.5",
		},

		// canvas
		{
			&Opts{
				Region: RegionUsEast,
				Canvas: &Canvas{Size: NewAreaWidthHeight("200", "200")},
			},
			"region=us_east&canvas=200,200",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Canvas: &Canvas{
					Size:     NewAreaWidthHeight("200", "200"),
					Position: &Position{X: "x10"},
				},
			},
			"region=us_east&canvas=200,200,x10",
		},
		{
			&Opts{
				Region: RegionUsEast,
				Canvas: &Canvas{
					Size:     NewAreaWidthHeight("200", "200"),
					Position: &Position{X: "x50p", Y: "y50p"},
				},
			},
			"region=us_east&canvas=200,200,x50p,y50p",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Canvas: &Canvas{
					Size:     NewAreaWidthHeight("200", "200"),
					Position: &Position{Y: "offset-y20"},
				},
			},

			"region=us_east&canvas=200,200,offset-y20",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Canvas: &Canvas{
					Size:     NewAreaWidthHeight("200", "200"),
					Position: &Position{X: "offset-x30", Y: "offset-y20"},
				},
			},

			"region=us_east&canvas=200,200,offset-x30,offset-y20",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Canvas: &Canvas{
					Size:     NewAreaAspectRatio(16, 9),
					Position: &Position{X: "offset-x30", Y: "offset-y20"},
				},
			},

			"region=us_east&canvas=16:9,offset-x30,offset-y20",
		},

		// crop
		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1)},
			},
			"region=us_east&crop=1:1",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe},
			},
			"region=us_east&crop=1:1,safe",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: "x30"}},
			},
			"region=us_east&crop=1:1,x30,safe",
		},
		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: "x30p", Y: "y20p"}},
			},
			"region=us_east&crop=1:1,x30p,y20p,safe",
		},
		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: "x30", Y: "y20p"}},
			},
			"region=us_east&crop=1:1,x30,y20p,safe",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: "offset-x30"}},
			},
			"region=us_east&crop=1:1,offset-x30,safe",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: "offset-x30", Y: "offset-y15"}},
			},
			"region=us_east&crop=1:1,offset-x30,offset-y15,safe",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: "offset-x30", Y: "offset-y15"}},
				Fit:    FitBounds,
			},
			"region=us_east&crop=1:1,offset-x30,offset-y15,safe&fit=bounds",
		},
		{
			&Opts{
				Region:  RegionUsEast,
				Precrop: &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: "offset-x30", Y: "offset-y15"}},
			},

			"region=us_east&precrop=1:1,offset-x30,offset-y15,safe",
		},
	}

	for i, tt := range tests {
		q, err := tt.opts.QueryString()
		if err != nil {
			t.Errorf("%v: error validating %+v: %v", i, tt.opts, err)
			continue
		}
		if q != tt.query {
			t.Errorf("%v: %+v .QueryString()=%q, want %q", i, tt.opts, q, tt.query)
		}
	}
}

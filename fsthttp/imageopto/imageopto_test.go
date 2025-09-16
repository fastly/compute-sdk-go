package imageopto

import "testing"

func TestOpts(t *testing.T) {

	tests := []struct {
		opts  *Opts
		query string
	}{
		{&Opts{Region: RegionUsEast}, "region=us_east"},
		{&Opts{Region: RegionUsEast, Width: NewPixelsOrPercentagePercent(50)}, "region=us_east&width=50p"},
		{&Opts{Region: RegionUsEast, Auto: AutoWebP}, "region=us_east&auto=webp"},
		{&Opts{Region: RegionUsEast, BgColor: &HexColor{R: 0, G: 255, B: 0, A: 0.3}}, "region=us_east&bg-color=0%2C255%2C0%2C0.3"},
		{&Opts{Region: RegionUsEast, Blur: NewBlurModePixels(50)}, "region=us_east&blur=50"}, // Rust: region=us_east&blur=50.0
		{&Opts{Region: RegionUsEast, Blur: NewBlurModePercentage(0.8)}, "region=us_east&blur=0.8p"},
		{&Opts{Region: RegionUsEast, Brightness: -50}, "region=us_east&brightness=-50"},
		{&Opts{Region: RegionUsEast, Bw: NewBWModeThreshold(10)}, "region=us_east&bw=threshold,10"},
		{&Opts{Region: RegionUsEast, Contrast: -5}, "region=us_east&constrast=-5"},
		{&Opts{Region: RegionUsEast, Dpr: 3.2}, "region=us_east&dpr=3.2"},
		{&Opts{Region: RegionUsEast, Enable: EnableOptUpscale}, "region=us_east&enable=upscale"},
		{&Opts{Region: RegionUsEast, Format: FormatJPEGXL}, "region=us_east&format=jpegxl"},
		{&Opts{Region: RegionUsEast, Frame: 1}, "region=us_east&frame=1"},
		{&Opts{Region: RegionUsEast, Height: NewPixelsOrPercentagePercent(80.0)}, "region=us_east&height=80p"},
		{&Opts{Region: RegionUsEast, Level: Level2_0, Format: FormatMP4, Profile: ProfileHigh}, "region=us_east&format=mp4&level=2.0&profile=high"},
		{&Opts{Region: RegionUsEast, Metadata: MetadataCopyright}, "region=us_east&metadata=copyright"},
		{&Opts{Region: RegionUsEast, Optimize: OptimizeLevelHigh}, "region=us_east&optimize=high"},
		{&Opts{Region: RegionUsEast, Orient: OrientationFlipVertical}, "region=us_east&orient=4"},

		{
			&Opts{
				Region: RegionUsEast,
				Pad: &Sides{
					Top:    NewPixelsOrPercentagePercent(10.0),
					Right:  NewPixelsOrPercentagePercent(10.0),
					Bottom: NewPixelsOrPercentagePercent(10.0),
					Left:   NewPixelsOrPercentagePercent(10.0),
				}},
			"region=us_east&pad=10p%2C10p%2C10p%2C10p",
		},

		{&Opts{Region: RegionUsEast, ResizeFilter: ResizeAlgorithmLanczos3}, "region=us_east&resize-filter=lanczos3"},
		{&Opts{Region: RegionUsEast, Sharpen: &Sharpen{Amount: 5, Radius: 2.0, Threshold: 1}}, "region=us_east&sharpen=a5%2Cr2%2Ct1"},

		{
			&Opts{
				Region: RegionUsEast,
				Trim: &Sides{
					Top:    NewPixelsOrPercentagePercent(20.5555),
					Right:  NewPixelsOrPercentagePercent(33.3333),
					Bottom: NewPixelsOrPercentagePercent(20.555),
					Left:   NewPixelsOrPercentagePercent(33.3333),
				}},
			"region=us_east&trim=20.556p%2C33.333p%2C20.555p%2C33.333p",
		},

		{
			&Opts{Region: RegionUsEast, TrimColor: &TrimColor{
				Color:     HexColor{R: 255, G: 0, B: 0, A: 1.0},
				Threshold: 0.5},
			},
			"region=us_east&trim-color=255%2C0%2C0%2C1%2Ct0.5", // Rust: "region=us_east&trim-color=255%2C0%2C0%2C1.0%2Ct0.5",
		},

		// canvas
		{
			&Opts{
				Region: RegionUsEast,
				Canvas: &Canvas{Size: NewAreaWidthHeight(NewPixelsOrPercentagePixels(200), NewPixelsOrPercentagePixels(200))},
			},
			"region=us_east&canvas=200%2C200",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Canvas: &Canvas{
					Size:     NewAreaWidthHeight(NewPixelsOrPercentagePixels(200), NewPixelsOrPercentagePixels(200)),
					Position: &Position{X: NewPointOrOffsetPoint(NewPixelsOrPercentagePixels(10))},
				},
			},
			"region=us_east&canvas=200%2C200%2Cx10",
		},
		{
			&Opts{
				Region: RegionUsEast,
				Canvas: &Canvas{
					Size:     NewAreaWidthHeight(NewPixelsOrPercentagePixels(200), NewPixelsOrPercentagePixels(200)),
					Position: &Position{X: NewPointOrOffsetPoint(NewPixelsOrPercentagePercent(50)), Y: NewPointOrOffsetPoint(NewPixelsOrPercentagePercent(50))},
				},
			},
			"region=us_east&canvas=200%2C200%2Cx50p%2Cy50p",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Canvas: &Canvas{
					Size:     NewAreaWidthHeight(NewPixelsOrPercentagePixels(200), NewPixelsOrPercentagePixels(200)),
					Position: &Position{Y: NewPointOrOffsetOffset(20)},
				},
			},

			"region=us_east&canvas=200%2C200%2Coffset-y20",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Canvas: &Canvas{
					Size:     NewAreaWidthHeight(NewPixelsOrPercentagePixels(200), NewPixelsOrPercentagePixels(200)),
					Position: &Position{X: NewPointOrOffsetOffset(30), Y: NewPointOrOffsetOffset(20)},
				},
			},

			"region=us_east&canvas=200%2C200%2Coffset-x30%2Coffset-y20",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Canvas: &Canvas{
					Size:     NewAreaAspectRatio(16, 9),
					Position: &Position{X: NewPointOrOffsetOffset(30), Y: NewPointOrOffsetOffset(20)},
				},
			},

			"region=us_east&canvas=16%3A9%2Coffset-x30%2Coffset-y20",
		},

		// crop
		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1)},
			},
			"region=us_east&crop=1%3A1",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe},
			},
			"region=us_east&crop=1%3A1%2Csafe",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: NewPointOrOffsetPoint(NewPixelsOrPercentagePixels(30))}},
			},
			"region=us_east&crop=1%3A1%2Cx30%2Csafe",
		},
		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: NewPointOrOffsetPoint(NewPixelsOrPercentagePercent(30)), Y: NewPointOrOffsetPoint(NewPixelsOrPercentagePercent(20))}},
			},
			"region=us_east&crop=1%3A1%2Cx30p%2Cy20p%2Csafe",
		},
		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: NewPointOrOffsetPoint(NewPixelsOrPercentagePixels(30)), Y: NewPointOrOffsetPoint(NewPixelsOrPercentagePercent(20))}},
			},
			"region=us_east&crop=1%3A1%2Cx30%2Cy20p%2Csafe",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: NewPointOrOffsetOffset(30)}},
			},
			"region=us_east&crop=1%3A1%2Coffset-x30%2Csafe",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: NewPointOrOffsetOffset(30), Y: NewPointOrOffsetOffset(15)}},
			},
			"region=us_east&crop=1%3A1%2Coffset-x30%2Coffset-y15%2Csafe",
		},

		{
			&Opts{
				Region: RegionUsEast,
				Crop:   &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: NewPointOrOffsetOffset(30), Y: NewPointOrOffsetOffset(15)}},
				Fit:    FitBounds,
			},
			"region=us_east&crop=1%3A1%2Coffset-x30%2Coffset-y15%2Csafe&fit=bounds",
		},
		{
			&Opts{
				Region:  RegionUsEast,
				Precrop: &Crop{Size: NewAreaAspectRatio(1, 1), Mode: CropModeSafe, Position: &Position{X: NewPointOrOffsetOffset(30), Y: NewPointOrOffsetOffset(15)}},
			},

			"region=us_east&precrop=1%3A1%2Coffset-x30%2Coffset-y15%2Csafe",
		},
	}

	for i, tt := range tests {
		q := tt.opts.QueryString()
		if q != tt.query {
			t.Errorf("%v: %+v .QueryString()=%q, want %q", i, tt.opts, q, tt.query)
		}
	}
}

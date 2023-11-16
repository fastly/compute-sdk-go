// Package device provides device dection based on the User-Agent
// header.
package device

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var (
	// ErrDeviceNotFound is returned when the device is not found.
	ErrDeviceNotFound = errors.New("device not found")

	// ErrUnexpected indicates that an unexpected error occurred.
	ErrUnexpected = errors.New("unexpected error")
)

type Device struct {
	info deviceInfo
}

type deviceInfo struct {
	Device struct {
		Name          string `json:"name"`
		Brand         string `json:"brand"`
		Model         string `json:"model"`
		HWType        string `json:"hwtype"`
		IsEReader     bool   `json:"is_ereader"`
		IsGameConsole bool   `json:"is_gameconsole"`
		IsMediaPlayer bool   `json:"is_mediaplayer"`
		IsMobile      bool   `json:"is_mobile"`
		IsSmartTV     bool   `json:"is_smarttv"`
		IsTablet      bool   `json:"is_tablet"`
		IsTVPlayer    bool   `json:"is_tvplayer"`
		IsDesktop     bool   `json:"is_desktop"`
		IsTouchscreen bool   `json:"is_touchscreen"`
	} `json:"device"`
}

func Lookup(userAgent string) (Device, error) {
	var d Device

	raw, err := fastly.DeviceLookup(userAgent)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		switch {
		case ok && status == fastly.FastlyStatusNone:
			return d, ErrDeviceNotFound
		case ok:
			return d, fmt.Errorf("%w (%s)", ErrUnexpected, status)
		default:
			return d, err
		}
	}

	if err := json.Unmarshal(raw, &d.info); err != nil {
		return d, err
	}

	return d, nil
}

// Name returns the name of the client device.
func (d *Device) Name() string {
	return d.info.Device.Name
}

// Brand returns the brand of the client device, possibly different from
// the manufacturer of that device.
func (d *Device) Brand() string {
	return d.info.Device.Brand
}

// Model returns the model of the client device.
func (d *Device) Model() string {
	return d.info.Device.Model
}

// HWType returns a string representation of the primary client platform
// hardware.  The most commonly used device types are also identified
// via boolean variables.  Because a device may have multiple device
// types and this variable only has the primary type, we recommend using
// the boolean variables for logic and using this string representation
// for logging.
func (d *Device) HWType() string {
	return d.info.Device.HWType
}

// IsEReader returns true if the client device is a reading device (like
// a Kindle).
func (d *Device) IsEReader() bool {
	return d.info.Device.IsEReader
}

// IsGameConsole returns true if the client device is a video game
// console (like a PlayStation or Xbox).
func (d *Device) IsGameConsole() bool {
	return d.info.Device.IsGameConsole
}

// IsMediaPlayer returns true if the client device is a media player
// (like Blu-ray players, iPod devices, and smart speakers such as
// Amazon Echo).
func (d *Device) IsMediaPlayer() bool {
	return d.info.Device.IsMediaPlayer
}

// IsMobile returns true if the client device is a mobile phone.
func (d *Device) IsMobile() bool {
	return d.info.Device.IsMobile
}

// IsSmartTV returns true if the client device is a smart TV.
func (d *Device) IsSmartTV() bool {
	return d.info.Device.IsSmartTV
}

// IsTablet returns true if the client device is a tablet (like an
// iPad).
func (d *Device) IsTablet() bool {
	return d.info.Device.IsTablet
}

// IsTVPlayer returns true if the client device is a set-top box or
// other TV player (like a Roku or Apple TV).
func (d *Device) IsTVPlayer() bool {
	return d.info.Device.IsTVPlayer
}

// IsDesktop returns true if the client device is a desktop web browser.
func (d *Device) IsDesktop() bool {
	return d.info.Device.IsDesktop
}

// IsTouchscreen returns true if the client device's screen is touch
// sensitive.
func (d *Device) IsTouchscreen() bool {
	return d.info.Device.IsTouchscreen
}

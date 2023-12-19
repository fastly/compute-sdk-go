// Copyright 2022 Fastly, Inc.

package geo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var (
	// ErrInvalidIP indicates the input IP was invalid.
	ErrInvalidIP = errors.New("geo: invalid IP")

	// ErrUnexpected indicates an unexpected error occurred.
	ErrUnexpected = errors.New("geo: unexpected error")
)

// Geo represents the geographic data for an IP address.
type Geo struct {
	AsName           string  `json:"as_name"`           // The name of the organization associated with AsNumber
	AsNumber         int     `json:"as_number"`         // Autonomous system (AS) number
	AreaCode         int     `json:"area_code"`         // The telephone area code associated with an IP address
	City             string  `json:"city"`              // City or town name
	ConnSpeed        string  `json:"conn_speed"`        // Connection speed
	ConnType         string  `json:"conn_type"`         // Connection type
	ContinentCode    string  `json:"continent"`         // A two-character UN M.49 continent code
	CountryCode      string  `json:"country_code"`      // A two-character ISO 3166-1 country code for the country associated with an IP address
	CountryCode3     string  `json:"country_code3"`     // A three-character ISO 3166-1 alpha-3 country code for the country associated with the IP address
	CountryName      string  `json:"country_name"`      // Country name
	Latitude         float64 `json:"latitude"`          // Latitude, in units of degrees from the equator
	Longitude        float64 `json:"longitude"`         // Longitude, in units of degrees from the IERS Reference Meridian
	MetroCode        int     `json:"metro_code"`        // Metro code, representing designated market areas (DMAs) in the United States
	PostalCode       string  `json:"postal_code"`       // The postal code associated with the IP address
	ProxyDescription string  `json:"proxy_description"` // Client proxy description
	ProxyType        string  `json:"proxy_type"`        // Client proxy type
	Region           string  `json:"region"`            // ISO 3166-2 country subdivision code
	UTCOffset        int     `json:"utc_offset"`        // Time zone offset from coordinated universal time (UTC) for city
}

// Lookup returns the geographic data associated with a particular IP address.
func Lookup(ip net.IP) (*Geo, error) {
	buf, err := fastly.GeoLookup(ip)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		switch {
		case ok && status == fastly.FastlyStatusNone:
			// Viceroy <= 0.9.3 returns fastly.FastlyStatusNone when no geolocation
			// data is available. The Compute production environment instead returns
			// empty data, which is handled by falling through to code below this switch.

			// TODO: potential breaking change if bumping major version
			// return nil, ErrNotFound
		case ok && status == fastly.FastlyStatusInval:
			return nil, ErrInvalidIP
		case ok:
			return nil, fmt.Errorf("%w (%s)", ErrUnexpected, status)
		default:
			return nil, err
		}
	}

	var g Geo

	// Check if there is geographic data for this IP address.
	if len(buf) == 0 {
		return &g, nil
	}

	if err := json.Unmarshal(buf, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

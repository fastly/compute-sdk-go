// Copyright 2022 Fastly, Inc.

package geo

import (
	"encoding/json"
	"net"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
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
		return nil, err
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

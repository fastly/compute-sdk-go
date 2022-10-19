// Copyright 2022 Fastly, Inc.

package geo

import (
	"fmt"
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

	return parseGeoJSON(buf)
}

func parseGeoJSON(buf []byte) (*Geo, error) {
	// Check if there is geographic data for this IP address.
	if len(buf) == 0 {
		return &Geo{}, nil
	}

	s := newScanner(buf)
	if tok := s.scan(); tok != tokenObjectStart {
		return nil, fmt.Errorf("unexpected JSON type at start of buffer %s", tok)
	}

	var g Geo
	for tok := s.scan(); tok != tokenObjectEnd; {
		if tok != tokenString {
			return nil, fmt.Errorf("expected quoted object key, got %s", tok)
		}

		key, err := s.decodeString()
		if err != nil {
			return nil, err
		}

		if tok = s.scan(); tok != tokenColon {
			return nil, fmt.Errorf("expected colon after hash key, got %s", tok)
		}

		// advance the scanner; the various decode functions will verify it's the correct type
		s.scan()

		switch key {
		case "as_name":
			g.AsName, err = s.decodeString()
		case "as_number":
			g.AsNumber, err = s.decodeInt()
		case "area_code":
			g.AreaCode, err = s.decodeInt()
		case "city":
			g.City, err = s.decodeString()
		case "conn_speed":
			g.ConnSpeed, err = s.decodeString()
		case "conn_type":
			g.ConnType, err = s.decodeString()
		case "continent":
			g.ContinentCode, err = s.decodeString()
		case "country_code":
			g.CountryCode, err = s.decodeString()
		case "country_code3":
			g.CountryCode3, err = s.decodeString()
		case "country_name":
			g.CountryName, err = s.decodeString()
		case "latitude":
			g.Latitude, err = s.decodeFloat()
		case "longitude":
			g.Longitude, err = s.decodeFloat()
		case "metro_code":
			g.MetroCode, err = s.decodeInt()
		case "postal_code":
			g.PostalCode, err = s.decodeString()
		case "proxy_description":
			g.ProxyDescription, err = s.decodeString()
		case "proxy_type":
			g.ProxyType, err = s.decodeString()
		case "region":
			g.Region, err = s.decodeString()
		case "utc_offset":
			g.UTCOffset, err = s.decodeInt()
		default:
			s.skipValue()
		}
		if err != nil {
			return nil, err
		}

		tok = s.scan()
		if tok != tokenComma && tok != tokenObjectEnd {
			return nil, fmt.Errorf("unexpected JSON token after value, got %v", tok)
		}
		if tok == tokenComma {
			tok = s.scan()
			if tok != tokenString {
				return nil, fmt.Errorf("unexpected JSON token after comma, got %v", tok)
			}
		}
	}

	if tok := s.scan(); tok != tokenEOF {
		return nil, fmt.Errorf("unexpected JSON type at end of object %s", tok)
	}

	return &g, nil
}

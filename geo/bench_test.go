package geo

import (
	"os"
	"testing"
)

var sink int

func BenchmarkParseGeoJSON(b *testing.B) {

	const filename = "testdata/geo_1.json"

	buf, err := os.ReadFile(filename)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g, _ := parseGeoJSON(buf)
		sink += g.UTCOffset
	}
}

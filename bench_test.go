package dateparse

import (
	"fmt"
	"testing"
	"time"
)

/*

go test -bench Parse

BenchmarkShotgunParse			50000	     37588 ns/op	   13258 B/op	     167 allocs/op
BenchmarkDateparseParseAny		500000	      5752 ns/op	       0 B/op	       0 allocs/op

*/
func BenchmarkShotgunParse(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, dateStr := range testDates {
			// This is the non dateparse traditional approach
			parseShotgunStyle(dateStr)
		}
	}
}

func BenchmarkParseAny(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, dateStr := range testDates {
			ParseAny(dateStr)
		}
	}
}

var (
	testDates = []string{
		"2012/03/19 10:11:59",
		"2012/03/19 10:11:59.3186369",
		"2009-08-12T22:15:09-07:00",
		"2014-04-26 17:24:37.3186369",
		"2012-08-03 18:31:59.257000000",
		"2013-04-01 22:43:22",
		"2014-04-26 17:24:37.123",
		"2014-12-16 06:20:00 UTC",
		"1384216367189",
		"1332151919",
		"2014-05-11 08:20:13,787",
		"2014-04-26 05:24:37 PM",
		"2014-04-26",
	}

	ErrDateFormat = fmt.Errorf("Invalid Date Format")

	timeFormats = []string{
		// ISO 8601ish formats
		time.RFC3339Nano,
		time.RFC3339,

		// Unusual formats, prefer formats with timezones
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.UnixDate,
		time.RubyDate,
		time.ANSIC,

		// Hilariously, Go doesn't have a const for it's own time layout.
		// See: https://code.google.com/p/go/issues/detail?id=6587
		"2006-01-02 15:04:05.999999999 -0700 MST",

		// No timezone information
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
)

func parseShotgunStyle(raw string) (time.Time, error) {

	for _, format := range timeFormats {
		t, err := time.Parse(format, raw)
		if err == nil {
			// Parsed successfully
			return t, nil
		}
	}
	return time.Time{}, ErrDateFormat
}

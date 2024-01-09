package dateparse

import (
	"fmt"
	"testing"
	"time"
)

/*
go test -bench Parse

// Aarons Laptop Lenovo 900 Feb 2018
BenchmarkShotgunParse-4   	   50000	     30045 ns/op	   13136 B/op	     169 allocs/op
BenchmarkParseAny-4       	  200000	      8627 ns/op	     144 B/op	       3 allocs/op

// ifreddyrondon Laptop MacBook Pro (Retina, Mid 2012) March 2018
BenchmarkShotgunParse-8   	   50000	     33940 ns/op	   13136 B/op	     169 allocs/op
BenchmarkParseAny-8   	  		200000	     10146 ns/op	     912 B/op	      29 allocs/op
BenchmarkParseDateString-8   	10000	    123077 ns/op	     208 B/op	      13 allocs/op

// Klondike Dragon Dec 2023
cpu: 12th Gen Intel(R) Core(TM) i7-1255U
BenchmarkShotgunParse-12                           62788             18113 ns/op           19448 B/op        474 allocs/op
BenchmarkParseAny-12                              347020              3455 ns/op              48 B/op          2 allocs/op
BenchmarkBigShotgunParse-12                         1226            951271 ns/op         1214937 B/op      27245 allocs/op
BenchmarkBigParseAny-12                             4234            267893 ns/op           27492 B/op        961 allocs/op
BenchmarkBigParseIn-12                              4032            280900 ns/op           30422 B/op       1033 allocs/op
BenchmarkBigParseRetryAmbiguous-12                  4453            282475 ns/op           29558 B/op       1030 allocs/op
BenchmarkShotgunParseErrors-12                     19240             62641 ns/op           67080 B/op       1679 allocs/op
BenchmarkParseAnyErrors-12                        185677              6179 ns/op             752 B/op         23 allocs/op
BenchmarkBigParseAnyErrors-12                      26688             44885 ns/op             480 B/op         94 allocs/op
BenchmarkParseAmbiguous-12                       1590302               752.9 ns/op           296 B/op          7 allocs/op
BenchmarkParseWeekdayAndFullMonth-12             2141109               555.0 ns/op            16 B/op          2 allocs/op
*/
func BenchmarkShotgunParse(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, dateStr := range testDates {
			// This is the non dateparse traditional approach
			_, _ = parseShotgunStyle(dateStr)
		}
	}
}

func BenchmarkParseAny(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, dateStr := range testDates {
			_, _ = ParseAny(dateStr)
		}
	}
}

func BenchmarkBigShotgunParse(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, t := range testInputs {
			// This is the non dateparse traditional approach
			_, _ = parseShotgunStyle(t.in)
		}
	}
}

func BenchmarkBigParseAny(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, t := range testInputs {
			_, _ = ParseAny(t.in)
		}
	}
}

func BenchmarkBigParseIn(b *testing.B) {
	b.ReportAllocs()
	loc, _ := time.LoadLocation("America/New_York")
	for i := 0; i < b.N; i++ {
		for _, t := range testInputs {
			_, _ = ParseIn(t.in, loc)
		}
	}
}

func BenchmarkBigParseRetryAmbiguous(b *testing.B) {
	b.ReportAllocs()
	opts := []ParserOption{RetryAmbiguousDateWithSwap(true)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range testInputs {
			_, _ = ParseAny(t.in, opts...)
		}
	}
}

func BenchmarkShotgunParseErrors(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, t := range testParseErrors {
			// This is the non dateparse traditional approach
			_, _ = parseShotgunStyle(t.in)
		}
	}
}

func BenchmarkParseAnyErrors(b *testing.B) {
	b.ReportAllocs()
	opts := []ParserOption{SimpleErrorMessages(true)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range testParseErrors {
			_, _ = ParseAny(t.in, opts...)
		}
	}
}

func BenchmarkBigParseAnyErrors(b *testing.B) {
	b.ReportAllocs()

	opts := []ParserOption{SimpleErrorMessages(true)}
	// manufacture a bunch of different tests with random errors put in them
	var testBigErrorInputs []string
	for index, t := range testInputs {
		b := []byte(t.in)
		spread := 4 + (index % 4)
		startingIndex := spread % len(b)
		for i := startingIndex; i < len(b); i += spread {
			b[i] = '?'
		}
		testBigErrorInputs = append(testBigErrorInputs, string(b))
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, in := range testBigErrorInputs {
			_, err := ParseAny(in, opts...)
			if err == nil {
				panic(fmt.Sprintf("expected parsing to fail: %s", in))
			}
		}
	}
}

func BenchmarkParseAmbiguous(b *testing.B) {
	b.ReportAllocs()
	opts := []ParserOption{RetryAmbiguousDateWithSwap(true)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MustParse("13/02/2014 04:08:09 +0000 UTC", opts...)
	}
}

func BenchmarkParseWeekdayAndFullMonth(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		MustParse("Monday 02 December 2006 03:04:05 PM UTC")
	}
}

/*
func BenchmarkParseDateString(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, dateStr := range testDates {
			timeutils.ParseDateString(dateStr)
		}
	}
}
*/

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

	ErrDateFormat = fmt.Errorf("invalid date format")

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

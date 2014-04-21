package dateparse

import (
	u "github.com/araddon/gou"
	"github.com/bmizerany/assert"
	"testing"
	"time"
)

var _ = time.April

/*
	ANSIC       = "Mon Jan _2 15:04:05 2006"
	UnixDate    = "Mon Jan _2 15:04:05 MST 2006"
	RubyDate    = "Mon Jan 02 15:04:05 -0700 2006"
	RFC822      = "02 Jan 06 15:04 MST"
	RFC822Z     = "02 Jan 06 15:04 -0700" // RFC822 with numeric zone
	RFC850      = "Monday, 02-Jan-06 15:04:05 MST"
	RFC1123     = "Mon, 02 Jan 2006 15:04:05 MST"
	RFC1123Z    = "Mon, 02 Jan 2006 15:04:05 -0700" // RFC1123 with numeric zone
	RFC3339     = "2006-01-02T15:04:05Z07:00"
	RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
	Kitchen     = "3:04PM"
	// Handy time stamps.
	Stamp      = "Jan _2 15:04:05"
	StampMilli = "Jan _2 15:04:05.000"
	StampMicro = "Jan _2 15:04:05.000000"
	StampNano  = "Jan _2 15:04:05.000000000"

	// unix etc
	1398045032   time.Now().Unix()
	1398045078199135196   time.Now().UnixNano()

	// Others
	"May 8, 2009 5:57:51 PM"


	Apr 7, 2014 4:58:55 PM
	03/19/2012 10:11:59
	3/1/2014
	10/13/2014
	01/02/2006

	2016-03-14 00:00:00.000
	2006-01-02

*/

func init() {
	u.SetupLogging("debug")
}

func TestParse(t *testing.T) {
	//f := parseFeatures("May 8, 2009 5:57:51 PM")
	ts, _ := ParseAny("May 8, 2009 5:57:51 PM")
	//u.Debug(" ", ts.Unix())
	//u.Debugf("%v", f)
	//u.Debugf("%v", ts)
	assert.T(t, ts.Unix() == 1241805471)
	ts, _ = ParseAny("03/19/2012 10:11:59")
	u.Debug(ts.Unix(), ts)
	assert.T(t, ts.Unix() == 1332151919)

}

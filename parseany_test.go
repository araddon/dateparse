package dateparse

import (
	"fmt"
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

	ts, err := ParseAny("May 8, 2009 5:57:51 PM")
	assert.T(t, err == nil)
	assert.T(t, ts.In(time.UTC).Unix() == 1241805471)

	ts, err = ParseAny("03/19/2012 10:11:59")
	assert.T(t, err == nil)
	//u.Debug(ts.Unix(), ts)
	assert.T(t, ts.Unix() == 1332151919)

	ts, err = ParseAny("3/31/2014")
	assert.T(t, err == nil)
	//u.Debug(ts.Unix(), ts)
	assert.T(t, ts.Unix() == 1396224000)

	ts, err = ParseAny("03/31/2014")
	assert.T(t, err == nil)
	//u.Debug(ts.Unix(), ts)
	assert.T(t, ts.Unix() == 1396224000)

	ts, err = ParseAny("4/8/2014 22:05")
	assert.T(t, err == nil)
	//u.Debug(ts.Unix(), ts)
	assert.T(t, "2014-04-08 22:05:00 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("04/08/2014 22:05")
	assert.T(t, err == nil)
	//u.Debug(ts.Unix(), ts)
	assert.T(t, ts.Unix() == 1396994700)

	// Unix Time Stamp
	ts, err = ParseAny("1332151919")
	assert.T(t, err == nil)
	//u.Debug(ts.Unix(), ts)
	assert.T(t, ts.Unix() == 1332151919)

	ts2, err := ParseAny("2009-08-12T22:15:09-07:00")
	assert.T(t, err == nil)
	//u.Debug(ts2.In(time.UTC), " ", ts2.Unix())
	assert.T(t, "2009-08-13 05:15:09 +0000 UTC" == fmt.Sprintf("%v", ts2.In(time.UTC)))

	//2014-04-26 05:24:37.3186369
	ts, err = ParseAny("2014-04-26 17:24:37.3186369")
	assert.T(t, err == nil)
	//u.Debug(ts.Unix(), ts)
	assert.T(t, "2014-04-26 17:24:37.3186369 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	//2014-04-26 17:24:37.123
	ts, err = ParseAny("2014-04-26 17:24:37.123")
	assert.T(t, err == nil)
	//u.Debugf("unix=%v   ts='%v'", ts.Unix(), ts)
	assert.T(t, "2014-04-26 17:24:37.123 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("2014-04-26 05:24:37 PM")
	assert.T(t, err == nil)
	//u.Debug(ts.Unix(), ts)
	assert.T(t, "2014-04-26 17:24:37 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("2014-04-26")
	assert.T(t, err == nil)
	//u.Debug(ts.Unix(), ts)
	assert.T(t, "2014-04-26 00:00:00 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))
}

// func TestParseAMPM(t *testing.T) {
// 	//2014-04-26 05:24:37 PM
// 	ts, err := ParseAny("2014-04-26 05:24:37 PM")
// 	assert.T(t, err == nil)
// 	u.Debug(ts.Unix(), ts)
// 	assert.T(t, "2014-04-26 17:24:37 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))
// }

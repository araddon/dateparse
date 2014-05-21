package dateparse

import (
	"fmt"
	u "github.com/araddon/gou"
	"github.com/bmizerany/assert"
	"testing"
	"time"
)

/*
	ANSIC       = "Mon Jan _2 15:04:05 2006"         x
	UnixDate    = "Mon Jan _2 15:04:05 MST 2006"     x
	RubyDate    = "Mon Jan 02 15:04:05 -0700 2006"    x
	RFC822      = "02 Jan 06 15:04 MST"               x
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
	04/2/2014 03:00:37
	3/1/2014
	10/13/2014
	01/02/2006

	2016-03-14 00:00:00.000
	2006-01-02
	2014-05-11 08:20:13,787   // i couldn't find parser for this in go?

*/

func init() {
	u.SetupLogging("debug")
}

func TestParse(t *testing.T) {

	ts, err := ParseAny("May 8, 2009 5:57:51 PM")
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2009-05-08 17:57:51 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	//   ANSIC       = "Mon Jan _2 15:04:05 2006"
	ts, err = ParseAny("Mon Jan  2 15:04:05 2006")
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2006-01-02 15:04:05 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	//   UnixDate    = "Mon Jan _2 15:04:05 MST 2006"
	ts, err = ParseAny("Mon Jan  2 15:04:05 MST 2006")
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2006-01-02 15:04:05 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	// RubyDate    = "Mon Jan 02 15:04:05 -0700 2006"
	ts, err = ParseAny("Mon Jan 02 15:04:05 -0700 2006")
	//u.Debug(fmt.Sprintf("%v", ts.In(time.UTC)), "  ---- ", ts)
	// Are we SURE this is right time?
	assert.T(t, "2006-01-02 15:04:05 -0700 -0700" == fmt.Sprintf("%v", ts))

	// Wat?  Go can't parse a date that it supplies a format for?
	// ts, err = ParseAny("Mon, 02 Jan 2006 15:04:05 -0700")
	// //u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	// assert.T(t, "2006-01-02 15:04:05 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("03/19/2012 10:11:59")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2012-03-19 10:11:59 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("3/31/2014")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2014-03-31 00:00:00 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("03/31/2014")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2014-03-31 00:00:00 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("4/8/2014 22:05")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2014-04-08 22:05:00 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("04/08/2014 22:05")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2014-04-08 22:05:00 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("04/2/2014 03:00:51")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2014-04-02 03:00:51 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("4/02/2014 03:00:51")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2014-04-02 03:00:51 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("1332151919")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2012-03-19 10:11:59 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("2009-08-12T22:15:09-07:00")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2009-08-13 05:15:09 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("2014-04-26 17:24:37.3186369")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2014-04-26 17:24:37.3186369 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("2012-08-03 18:31:59.257000000")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2012-08-03 18:31:59.257 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("2013-04-01 22:43:22")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2013-04-01 22:43:22 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("2014-04-26 17:24:37.123")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2014-04-26 17:24:37.123 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("2014-04-26 05:24:37 PM")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2014-04-26 17:24:37 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("2014-04-26")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2014-04-26 00:00:00 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseAny("2014-05-11 08:20:13,787")
	assert.Tf(t, err == nil, "%v", err)
	//u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
	assert.T(t, "2014-05-11 08:20:13.787 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))
}

// func TestWIP(t *testing.T) {
// 	ts, err := ParseAny("2013-04-01 22:43:22")
// 	assert.Tf(t, err == nil, "%v", err)
// 	u.Debug(ts.In(time.UTC).Unix(), ts.In(time.UTC))
// 	assert.T(t, "2013-04-01 22:43:22 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))
// }

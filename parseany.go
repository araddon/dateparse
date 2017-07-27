package dateparse

import (
	"fmt"
	"strconv"
	"time"
	"unicode"
)

type dateState int

const (
	st_START dateState = iota
	st_DIGIT
	st_DIGITDASH
	st_DIGITDASHALPHA
	st_DIGITDASHWS
	st_DIGITDASHWSWS
	st_DIGITDASHWSWSAMPMMAYBE
	st_DIGITDASHWSWSOFFSET
	st_DIGITDASHWSWSOFFSETALPHA
	st_DIGITDASHWSWSOFFSETCOLONALPHA
	st_DIGITDASHWSWSOFFSETCOLON
	st_DIGITDASHWSOFFSET
	st_DIGITDASHWSWSALPHA
	st_DIGITDASHWSDOT
	st_DIGITDASHWSDOTALPHA
	st_DIGITDASHWSDOTOFFSET
	st_DIGITDASHWSDOTOFFSETALPHA
	st_DIGITDASHT
	st_DIGITDASHTZ
	st_DIGITDASHTZDIGIT
	st_DIGITDASHTDELTA
	st_DIGITDASHTDELTACOLON
	st_DIGITSLASH
	st_DIGITSLASHWS
	st_DIGITSLASHWSCOLON
	st_DIGITSLASHWSCOLONAMPM
	st_DIGITSLASHWSCOLONCOLON
	st_DIGITSLASHWSCOLONCOLONAMPM
	st_DIGITALPHA
	st_ALPHA
	st_ALPHAWS
	st_ALPHAWSDIGITCOMMA
	st_ALPHAWSALPHA
	st_ALPHAWSALPHACOLON
	st_ALPHAWSALPHACOLONOFFSET
	st_ALPHAWSALPHACOLONALPHA
	st_ALPHAWSALPHACOLONALPHAOFFSET
	st_ALPHAWSALPHACOLONALPHAOFFSETALPHA
	st_WEEKDAYCOMMA
	st_WEEKDAYCOMMADELTA
	st_WEEKDAYABBREVCOMMA
	st_WEEKDAYABBREVCOMMADELTA
	st_WEEKDAYABBREVCOMMADELTAZONE
)

var (
	shortDates = []string{"01/02/2006", "1/2/2006", "06/01/02", "01/02/06", "1/2/06"}
)

// MustParse Parse a date, and panic if it can't be parsed
func MustParse(datestr string) time.Time {
	t, err := parseTime(datestr, nil)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// ParseAny Given an unknown date format, detect the layout, parse.
func ParseAny(datestr string) (time.Time, error) {
	return parseTime(datestr, nil)
}

// ParseIn Given an unknown date format, detect the layout,
// using given location, parse.
//
// If no recognized Timezone/Offset info exists in the datestring, it uses
// given location. IF there IS timezone/offset info it uses the given location
// info for any zone interpretation.  That is, MST means one thing when using
// America/Denver and something else in other locations.
func ParseIn(datestr string, loc *time.Location) (time.Time, error) {
	return parseTime(datestr, loc)
}

// ParseLocal Given an unknown date format, detect the layout,
// using time.Local, parse.
//
// If no recognized Timezone/Offset info exists in the datestring, it uses
// given location. IF there IS timezone/offset info it uses the given location
// info for any zone interpretation.  That is, MST means one thing when using
// America/Denver and something else in other locations.
func ParseLocal(datestr string) (time.Time, error) {
	return parseTime(datestr, time.Local)
}

func parse(layout, datestr string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		return time.Parse(layout, datestr)
	}
	return time.ParseInLocation(layout, datestr, loc)
}

func parseTime(datestr string, loc *time.Location) (time.Time, error) {
	state := st_START

	firstSlash := 0

	// General strategy is to read rune by rune through the date looking for
	// certain hints of what type of date we are dealing with.
	// Hopefully we only need to read about 5 or 6 bytes before
	// we figure it out and then attempt a parse
iterRunes:
	for i := 0; i < len(datestr); i++ {
		r := rune(datestr[i])
		// r, bytesConsumed := utf8.DecodeRuneInString(datestr[ri:])
		// if bytesConsumed > 1 {
		// 	ri += (bytesConsumed - 1)
		// }

		switch state {
		case st_START:
			if unicode.IsDigit(r) {
				state = st_DIGIT
			} else if unicode.IsLetter(r) {
				state = st_ALPHA
			}
		case st_DIGIT: // starts digits
			if unicode.IsDigit(r) {
				continue
			} else if unicode.IsLetter(r) {
				state = st_DIGITALPHA
				continue
			}
			switch r {
			case '-', '\u2212':
				state = st_DIGITDASH
			case '/':
				state = st_DIGITSLASH
				firstSlash = i
			}
		case st_DIGITDASH: // starts digit then dash 02-
			// 2006-01-02T15:04:05Z07:00
			// 2017-06-25T17:46:57.45706582-07:00
			// 2006-01-02T15:04:05.999999999Z07:00
			// 2006-01-02T15:04:05+0000
			// 2012-08-03 18:31:59.257000000
			// 2014-04-26 17:24:37.3186369
			// 2017-01-27 00:07:31.945167
			// 2016-03-14 00:00:00.000
			// 2014-05-11 08:20:13,787
			// 2017-07-19 03:21:51+00:00
			// 2006-01-02
			// 2013-04-01 22:43:22
			// 2014-04-26 05:24:37 PM
			// 2013-Feb-03
			switch {
			case r == ' ':
				state = st_DIGITDASHWS
			case r == 'T':
				state = st_DIGITDASHT
			default:
				if unicode.IsLetter(r) {
					state = st_DIGITDASHALPHA
					break iterRunes
				}
			}
		case st_DIGITDASHWS:
			// 2013-04-01 22:43:22
			// 2014-05-11 08:20:13,787
			// st_DIGITDASHWSWS
			//   2014-04-26 05:24:37 PM
			//   2014-12-16 06:20:00 UTC
			//   2015-02-18 00:12:00 +0000 UTC
			//   2006-01-02 15:04:05 -0700
			//   2006-01-02 15:04:05 -07:00
			// st_DIGITDASHWSOFFSET
			//   2017-07-19 03:21:51+00:00
			// st_DIGITDASHWSDOT
			//   2014-04-26 17:24:37.3186369
			//   2017-01-27 00:07:31.945167
			//   2012-08-03 18:31:59.257000000
			//   2016-03-14 00:00:00.000
			//   st_DIGITDASHWSDOTOFFSET
			//     2017-01-27 00:07:31.945167 +0000
			//     2016-03-14 00:00:00.000 +0000
			//     st_DIGITDASHWSDOTOFFSETALPHA
			//       2017-01-27 00:07:31.945167 +0000 UTC
			//       2016-03-14 00:00:00.000 +0000 UTC
			//   st_DIGITDASHWSDOTALPHA
			//     2014-12-16 06:20:00.000 UTC
			switch r {
			case ',':
				if len(datestr) == len("2014-05-11 08:20:13,787") {
					// go doesn't seem to parse this one natively?   or did i miss it?
					t, err := parse("2006-01-02 03:04:05", datestr[:i], loc)
					if err == nil {
						ms, err := strconv.Atoi(datestr[i+1:])
						if err == nil {
							return time.Unix(0, t.UnixNano()+int64(ms)*1e6), nil
						}
					}
					return t, err
				}
			case '-', '+':
				state = st_DIGITDASHWSOFFSET
			case '.':
				state = st_DIGITDASHWSDOT
			case ' ':
				state = st_DIGITDASHWSWS
			}

		case st_DIGITDASHWSWS:
			// st_DIGITDASHWSWSALPHA
			//   2014-12-16 06:20:00 UTC
			//   st_DIGITDASHWSWSAMPMMAYBE
			//     2014-04-26 05:24:37 PM
			// st_DIGITDASHWSWSOFFSET
			//   2006-01-02 15:04:05 -0700
			//   st_DIGITDASHWSWSOFFSETCOLON
			//     2006-01-02 15:04:05 -07:00
			//     st_DIGITDASHWSWSOFFSETCOLONALPHA
			//       2015-02-18 00:12:00 +00:00 UTC
			//   st_DIGITDASHWSWSOFFSETALPHA
			//     2015-02-18 00:12:00 +0000 UTC
			switch r {
			case 'A', 'P':
				state = st_DIGITDASHWSWSAMPMMAYBE
			case '+', '-':
				state = st_DIGITDASHWSWSOFFSET
			default:
				if unicode.IsLetter(r) {
					// 2014-12-16 06:20:00 UTC
					state = st_DIGITDASHWSWSALPHA
					break iterRunes
				}
			}

		case st_DIGITDASHWSWSAMPMMAYBE:
			if r == 'M' {
				return parse("2006-01-02 03:04:05 PM", datestr, loc)
			}
			state = st_DIGITDASHWSWSALPHA

		case st_DIGITDASHWSWSOFFSET:
			// st_DIGITDASHWSWSOFFSET
			//   2006-01-02 15:04:05 -0700
			//   st_DIGITDASHWSWSOFFSETCOLON
			//     2006-01-02 15:04:05 -07:00
			//     st_DIGITDASHWSWSOFFSETCOLONALPHA
			//       2015-02-18 00:12:00 +00:00 UTC
			//   st_DIGITDASHWSWSOFFSETALPHA
			//     2015-02-18 00:12:00 +0000 UTC
			if r == ':' {
				state = st_DIGITDASHWSWSOFFSETCOLON
			} else if unicode.IsLetter(r) {
				// 2015-02-18 00:12:00 +0000 UTC
				state = st_DIGITDASHWSWSOFFSETALPHA
				break iterRunes
			}

		case st_DIGITDASHWSWSOFFSETCOLON:
			// st_DIGITDASHWSWSOFFSETCOLON
			//   2006-01-02 15:04:05 -07:00
			//   st_DIGITDASHWSWSOFFSETCOLONALPHA
			//     2015-02-18 00:12:00 +00:00 UTC
			if unicode.IsLetter(r) {
				// 2015-02-18 00:12:00 +00:00 UTC
				state = st_DIGITDASHWSWSOFFSETCOLONALPHA
				break iterRunes
			}

		case st_DIGITDASHWSDOT:
			// 2014-04-26 17:24:37.3186369
			// 2017-01-27 00:07:31.945167
			// 2012-08-03 18:31:59.257000000
			// 2016-03-14 00:00:00.000
			// st_DIGITDASHWSDOTOFFSET
			//   2017-01-27 00:07:31.945167 +0000
			//   2016-03-14 00:00:00.000 +0000
			//   st_DIGITDASHWSDOTOFFSETALPHA
			//     2017-01-27 00:07:31.945167 +0000 UTC
			//     2016-03-14 00:00:00.000 +0000 UTC
			// st_DIGITDASHWSDOTALPHA
			//   2014-12-16 06:20:00.000 UTC
			if unicode.IsLetter(r) {
				// 2014-12-16 06:20:00.000 UTC
				state = st_DIGITDASHWSDOTALPHA
				break iterRunes
			} else if r == '+' || r == '-' {
				state = st_DIGITDASHWSDOTOFFSET
			}
		case st_DIGITDASHWSDOTOFFSET:
			// 2017-01-27 00:07:31.945167 +0000
			// 2016-03-14 00:00:00.000 +0000
			// st_DIGITDASHWSDOTOFFSETALPHA
			//   2017-01-27 00:07:31.945167 +0000 UTC
			//   2016-03-14 00:00:00.000 +0000 UTC
			if unicode.IsLetter(r) {
				// 2014-12-16 06:20:00.000 UTC
				// 2017-01-27 00:07:31.945167 +0000 UTC
				// 2016-03-14 00:00:00.000 +0000 UTC
				state = st_DIGITDASHWSDOTOFFSETALPHA
				break iterRunes
			}
		case st_DIGITDASHT: // starts digit then dash 02-  then T
			// st_DIGITDASHT
			// 2006-01-02T15:04:05
			// st_DIGITDASHTZ
			// 2006-01-02T15:04:05.999999999Z
			// 2006-01-02T15:04:05.99999999Z
			// 2006-01-02T15:04:05.9999999Z
			// 2006-01-02T15:04:05.999999Z
			// 2006-01-02T15:04:05.99999Z
			// 2006-01-02T15:04:05.9999Z
			// 2006-01-02T15:04:05.999Z
			// 2006-01-02T15:04:05.99Z
			// 2009-08-12T22:15Z
			// st_DIGITDASHTZDIGIT
			// 2006-01-02T15:04:05.999999999Z07:00
			// 2006-01-02T15:04:05Z07:00
			// With another dash aka time-zone at end
			// st_DIGITDASHTDELTA
			//   st_DIGITDASHTDELTACOLON
			//     2017-06-25T17:46:57.45706582-07:00
			//     2017-06-25T17:46:57+04:00
			// 2006-01-02T15:04:05+0000
			switch r {
			case '-', '+':
				state = st_DIGITDASHTDELTA
			case 'Z':
				state = st_DIGITDASHTZ
			}
		case st_DIGITDASHTZ:
			if unicode.IsDigit(r) {
				state = st_DIGITDASHTZDIGIT
			}
		case st_DIGITDASHTDELTA:
			if r == ':' {
				state = st_DIGITDASHTDELTACOLON
			}
		case st_DIGITSLASH: // starts digit then slash 02/
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 3/1/2014
			// 10/13/2014
			// 01/02/2006
			// 1/2/06
			if unicode.IsDigit(r) || r == '/' {
				continue
			}
			switch r {
			case ' ':
				state = st_DIGITSLASHWS
			}
		case st_DIGITSLASHWS: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			switch r {
			case ':':
				state = st_DIGITSLASHWSCOLON
			}
		case st_DIGITSLASHWSCOLON: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 3/1/2012 10:11:59 AM
			switch r {
			case ':':
				state = st_DIGITSLASHWSCOLONCOLON
			case 'A', 'P':
				state = st_DIGITSLASHWSCOLONAMPM
			}
		case st_DIGITSLASHWSCOLONCOLON: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 3/1/2012 10:11:59 AM
			switch r {
			case 'A', 'P':
				state = st_DIGITSLASHWSCOLONCOLONAMPM
			}
		case st_DIGITALPHA:
			// 12 Feb 2006, 19:17
			// 12 Feb 2006, 19:17:22
			switch {
			case len(datestr) == len("02 Jan 2006, 15:04"):
				return parse("02 Jan 2006, 15:04", datestr, loc)
			case len(datestr) == len("02 Jan 2006, 15:04:05"):
				return parse("02 Jan 2006, 15:04:05", datestr, loc)
			}
		case st_ALPHA: // starts alpha
			// st_ALPHAWS
			//  Mon Jan _2 15:04:05 2006
			//  Mon Jan _2 15:04:05 MST 2006
			//  Mon Jan 02 15:04:05 -0700 2006
			//  Mon Aug 10 15:44:11 UTC+0100 2015
			//  Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			//  st_ALPHAWSDIGITCOMMA
			//    May 8, 2009 5:57:51 PM
			//
			// st_WEEKDAYCOMMA
			//   Monday, 02-Jan-06 15:04:05 MST
			//   st_WEEKDAYCOMMADELTA
			//     Monday, 02 Jan 2006 15:04:05 -0700
			//     Monday, 02 Jan 2006 15:04:05 +0100
			// st_WEEKDAYABBREVCOMMA
			//   Mon, 02-Jan-06 15:04:05 MST
			//   Mon, 02 Jan 2006 15:04:05 MST
			//   st_WEEKDAYABBREVCOMMADELTA
			//     Mon, 02 Jan 2006 15:04:05 -0700
			//     Thu, 13 Jul 2017 08:58:40 +0100
			//     st_WEEKDAYABBREVCOMMADELTAZONE
			//       Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			switch {
			case unicode.IsLetter(r):
				continue
			case r == ' ':
				state = st_ALPHAWS
			case r == ',':
				if i == 3 {
					state = st_WEEKDAYABBREVCOMMA
				} else {
					state = st_WEEKDAYCOMMA
				}
			}
		case st_WEEKDAYCOMMA: // Starts alpha then comma
			// Mon, 02-Jan-06 15:04:05 MST
			// Mon, 02 Jan 2006 15:04:05 MST
			// st_WEEKDAYCOMMADELTA
			//   Monday, 02 Jan 2006 15:04:05 -0700
			//   Monday, 02 Jan 2006 15:04:05 +0100
			switch {
			case r == '-':
				if i < 15 {
					return parse("Monday, 02-Jan-06 15:04:05 MST", datestr, loc)
				} else {
					state = st_WEEKDAYCOMMADELTA
				}
			case r == '+':
				state = st_WEEKDAYCOMMADELTA
			}
		case st_WEEKDAYABBREVCOMMA: // Starts alpha then comma
			// Mon, 02-Jan-06 15:04:05 MST
			// Mon, 02 Jan 2006 15:04:05 MST
			// st_WEEKDAYABBREVCOMMADELTA
			//   Mon, 02 Jan 2006 15:04:05 -0700
			//   Thu, 13 Jul 2017 08:58:40 +0100
			//   st_WEEKDAYABBREVCOMMADELTAZONE
			//     Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			switch {
			case r == '-':
				if i < 15 {
					return parse("Mon, 02-Jan-06 15:04:05 MST", datestr, loc)
				} else {
					state = st_WEEKDAYABBREVCOMMADELTA
				}
			case r == '+':
				state = st_WEEKDAYABBREVCOMMADELTA
			}

		case st_WEEKDAYABBREVCOMMADELTA:
			// st_WEEKDAYABBREVCOMMADELTA
			//   Mon, 02 Jan 2006 15:04:05 -0700
			//   Thu, 13 Jul 2017 08:58:40 +0100
			//   st_WEEKDAYABBREVCOMMADELTAZONE
			//     Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			if r == '(' {
				state = st_WEEKDAYABBREVCOMMADELTAZONE
			}

		case st_ALPHAWS: // Starts alpha then whitespace
			// Mon Jan _2 15:04:05 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			// Mon Aug 10 15:44:11 UTC+0100 2015
			switch {
			case unicode.IsLetter(r):
				state = st_ALPHAWSALPHA
			case unicode.IsDigit(r):
				state = st_ALPHAWSDIGITCOMMA
			}

		case st_ALPHAWSDIGITCOMMA: // Starts Alpha, whitespace, digit, comma
			// May 8, 2009 5:57:51 PM
			return parse("Jan 2, 2006 3:04:05 PM", datestr, loc)

		case st_ALPHAWSALPHA: // Alpha, whitespace, alpha
			// Mon Jan _2 15:04:05 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == ':' {
				state = st_ALPHAWSALPHACOLON
			}
		case st_ALPHAWSALPHACOLON: // Alpha, whitespace, alpha, :
			// Mon Jan _2 15:04:05 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if unicode.IsLetter(r) {
				state = st_ALPHAWSALPHACOLONALPHA
			} else if r == '-' || r == '+' {
				state = st_ALPHAWSALPHACOLONOFFSET
			}
		case st_ALPHAWSALPHACOLONALPHA: // Alpha, whitespace, alpha, :, alpha
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == '+' {
				state = st_ALPHAWSALPHACOLONALPHAOFFSET
			}
		case st_ALPHAWSALPHACOLONALPHAOFFSET: // Alpha, whitespace, alpha, : , alpha, offset, ?
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if unicode.IsLetter(r) {
				state = st_ALPHAWSALPHACOLONALPHAOFFSETALPHA
			}
		default:
			break iterRunes
		}
	}

	switch state {
	case st_DIGIT:
		// unixy timestamps ish
		//  1499979655583057426  nanoseconds
		//  1499979795437000     micro-seconds
		//  1499979795437        milliseconds
		//  1384216367189
		//  1332151919           seconds
		//  20140601             yyyymmdd
		//  2014                 yyyy
		if len(datestr) > len("1499979795437000") {
			if nanoSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				return time.Unix(0, nanoSecs), nil
			}
		} else if len(datestr) > len("1499979795437") {
			if microSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				return time.Unix(0, microSecs*1000), nil
			}
		} else if len(datestr) > len("1332151919") {
			if miliSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				return time.Unix(0, miliSecs*1000*1000), nil
			}
		} else if len(datestr) == len("20140601") {
			return parse("20060102", datestr, loc)
		} else if len(datestr) == len("2014") {
			return parse("2006", datestr, loc)
		} else {
			if secs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				return time.Unix(secs, 0), nil
			}
		}
	case st_DIGITDASH: // starts digit then dash 02-
		// 2006-01-02
		// 2006-01
		if len(datestr) == len("2014-04-26") {
			return parse("2006-01-02", datestr, loc)
		} else if len(datestr) == len("2014-04") {
			return parse("2006-01", datestr, loc)
		}
	case st_DIGITDASHALPHA:
		// 2013-Feb-03
		return parse("2006-Jan-02", datestr, loc)

	case st_DIGITDASHTDELTA:
		// 2006-01-02T15:04:05+0000
		return parse("2006-01-02T15:04:05-0700", datestr, loc)

	case st_DIGITDASHTDELTACOLON:
		// With another +/- time-zone at end
		// 2006-01-02T15:04:05.999999999+07:00
		// 2006-01-02T15:04:05.999999999-07:00
		// 2006-01-02T15:04:05.999999+07:00
		// 2006-01-02T15:04:05.999999-07:00
		// 2006-01-02T15:04:05.999+07:00
		// 2006-01-02T15:04:05.999-07:00
		// 2006-01-02T15:04:05+07:00
		// 2006-01-02T15:04:05-07:00
		return parse("2006-01-02T15:04:05-07:00", datestr, loc)

	case st_DIGITDASHT: // starts digit then dash 02-  then T
		// 2006-01-02T15:04:05.999999
		// 2006-01-02T15:04:05.999999
		return parse("2006-01-02T15:04:05", datestr, loc)

	case st_DIGITDASHTZDIGIT:
		// With a time-zone at end after Z
		// 2006-01-02T15:04:05.999999999Z07:00
		// 2006-01-02T15:04:05Z07:00
		// RFC3339     = "2006-01-02T15:04:05Z07:00"
		// RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
		return time.Time{}, fmt.Errorf("RFC339 Dates may not contain both Z & Offset for %q see https://github.com/golang/go/issues/5294", datestr)

	case st_DIGITDASHTZ: // starts digit then dash 02-  then T Then Z
		// 2006-01-02T15:04:05.999999999Z
		// 2006-01-02T15:04:05.99999999Z
		// 2006-01-02T15:04:05.9999999Z
		// 2006-01-02T15:04:05.999999Z
		// 2006-01-02T15:04:05.99999Z
		// 2006-01-02T15:04:05.9999Z
		// 2006-01-02T15:04:05.999Z
		// 2006-01-02T15:04:05.99Z
		// 2009-08-12T22:15Z  -- No seconds/milliseconds
		switch len(datestr) {
		case len("2009-08-12T22:15Z"):
			return parse("2006-01-02T15:04Z", datestr, loc)
		default:
			return parse("2006-01-02T15:04:05Z", datestr, loc)
		}
	case st_DIGITDASHWS: // starts digit then dash 02-  then whitespace   1 << 2  << 5 + 3
		// 2013-04-01 22:43:22
		return parse("2006-01-02 15:04:05", datestr, loc)

	case st_DIGITDASHWSWSOFFSET:
		// 2006-01-02 15:04:05 -0700
		return parse("2006-01-02 15:04:05 -0700", datestr, loc)

	case st_DIGITDASHWSWSOFFSETCOLON:
		// 2006-01-02 15:04:05 -07:00
		return parse("2006-01-02 15:04:05 -07:00", datestr, loc)

	case st_DIGITDASHWSWSOFFSETALPHA:
		// 2015-02-18 00:12:00 +0000 UTC
		t, err := parse("2006-01-02 15:04:05 -0700 UTC", datestr, loc)
		if err == nil {
			return t, nil
		}
		return parse("2006-01-02 15:04:05 +0000 GMT", datestr, loc)

	case st_DIGITDASHWSWSOFFSETCOLONALPHA:
		// 2015-02-18 00:12:00 +00:00 UTC
		return parse("2006-01-02 15:04:05 -07:00 UTC", datestr, loc)

	case st_DIGITDASHWSOFFSET:
		// 2017-07-19 03:21:51+00:00
		return parse("2006-01-02 15:04:05-07:00", datestr, loc)

	case st_DIGITDASHWSWSALPHA:
		// 2014-12-16 06:20:00 UTC
		t, err := parse("2006-01-02 15:04:05 UTC", datestr, loc)
		if err == nil {
			return t, nil
		}
		t, err = parse("2006-01-02 15:04:05 GMT", datestr, loc)
		if err == nil {
			return t, nil
		}
		if len(datestr) > len("2006-01-02 03:04:05") {
			t, err = parse("2006-01-02 03:04:05", datestr[:len("2006-01-02 03:04:05")], loc)
			if err == nil {
				return t, nil
			}
		}

	case st_DIGITDASHWSDOT:
		// 2012-08-03 18:31:59.257000000
		// 2014-04-26 17:24:37.3186369
		// 2017-01-27 00:07:31.945167
		// 2016-03-14 00:00:00.000
		return parse("2006-01-02 15:04:05", datestr, loc)

	case st_DIGITDASHWSDOTALPHA:
		// 2012-08-03 18:31:59.257000000 UTC
		// 2014-04-26 17:24:37.3186369 UTC
		// 2017-01-27 00:07:31.945167 UTC
		// 2016-03-14 00:00:00.000 UTC
		return parse("2006-01-02 15:04:05 UTC", datestr, loc)

	case st_DIGITDASHWSDOTOFFSET:
		// 2012-08-03 18:31:59.257000000 +0000
		// 2014-04-26 17:24:37.3186369 +0000
		// 2017-01-27 00:07:31.945167 +0000
		// 2016-03-14 00:00:00.000 +0000
		return parse("2006-01-02 15:04:05 -0700", datestr, loc)

	case st_DIGITDASHWSDOTOFFSETALPHA:
		// 2012-08-03 18:31:59.257000000 +0000 UTC
		// 2014-04-26 17:24:37.3186369 +0000 UTC
		// 2017-01-27 00:07:31.945167 +0000 UTC
		// 2016-03-14 00:00:00.000 +0000 UTC
		return parse("2006-01-02 15:04:05 -0700 UTC", datestr, loc)

	case st_ALPHAWSALPHACOLON:
		// Mon Jan _2 15:04:05 2006
		return parse(time.ANSIC, datestr, loc)

	case st_ALPHAWSALPHACOLONOFFSET:
		// Mon Jan 02 15:04:05 -0700 2006
		return parse(time.RubyDate, datestr, loc)

	case st_ALPHAWSALPHACOLONALPHA:
		// Mon Jan _2 15:04:05 MST 2006
		return parse(time.UnixDate, datestr, loc)

	case st_ALPHAWSALPHACOLONALPHAOFFSET:
		// Mon Aug 10 15:44:11 UTC+0100 2015
		return parse("Mon Jan 02 15:04:05 MST-0700 2006", datestr, loc)

	case st_ALPHAWSALPHACOLONALPHAOFFSETALPHA:
		// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
		if len(datestr) > len("Mon Jan 02 2006 15:04:05 MST-0700") {
			// What effing time stamp is this?
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			dateTmp := datestr[:33]
			return parse("Mon Jan 02 2006 15:04:05 MST-0700", dateTmp, loc)
		}
	case st_DIGITSLASH: // starts digit then slash 02/ (but nothing else)
		// 3/1/2014
		// 10/13/2014
		// 01/02/2006
		// 2014/10/13
		if firstSlash == 4 {
			if len(datestr) == len("2006/01/02") {
				return parse("2006/01/02", datestr, loc)
			} else {
				return parse("2006/1/2", datestr, loc)
			}
		} else {
			for _, parseFormat := range shortDates {
				if t, err := parse(parseFormat, datestr, loc); err == nil {
					return t, nil
				}
			}
		}

	case st_DIGITSLASHWSCOLON: // starts digit then slash 02/ more digits/slashes then whitespace
		// 4/8/2014 22:05
		// 04/08/2014 22:05
		// 2014/4/8 22:05
		// 2014/04/08 22:05

		if firstSlash == 4 {
			for _, layout := range []string{"2006/01/02 15:04", "2006/1/2 15:04", "2006/01/2 15:04", "2006/1/02 15:04"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 15:04", "01/2/2006 15:04", "1/02/2006 15:04", "1/2/2006 15:04"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		}

	case st_DIGITSLASHWSCOLONAMPM: // starts digit then slash 02/ more digits/slashes then whitespace
		// 4/8/2014 22:05 PM
		// 04/08/2014 22:05 PM
		// 04/08/2014 1:05 PM
		// 2014/4/8 22:05 PM
		// 2014/04/08 22:05 PM

		if firstSlash == 4 {
			for _, layout := range []string{"2006/01/02 03:04 PM", "2006/01/2 03:04 PM", "2006/1/02 03:04 PM", "2006/1/2 03:04 PM",
				"2006/01/02 3:04 PM", "2006/01/2 3:04 PM", "2006/1/02 3:04 PM", "2006/1/2 3:04 PM"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 03:04 PM", "01/2/2006 03:04 PM", "1/02/2006 03:04 PM", "1/2/2006 03:04 PM",
				"01/02/2006 3:04 PM", "01/2/2006 3:04 PM", "1/02/2006 3:04 PM", "1/2/2006 3:04 PM"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}

			}
		}

	case st_DIGITSLASHWSCOLONCOLON: // starts digit then slash 02/ more digits/slashes then whitespace double colons
		// 2014/07/10 06:55:38.156283
		// 03/19/2012 10:11:59
		// 3/1/2012 10:11:59
		// 03/1/2012 10:11:59
		// 3/01/2012 10:11:59
		if firstSlash == 4 {
			for _, layout := range []string{"2006/01/02 15:04:05", "2006/1/02 15:04:05", "2006/01/2 15:04:05", "2006/1/2 15:04:05"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 15:04:05", "1/02/2006 15:04:05", "01/2/2006 15:04:05", "1/2/2006 15:04:05"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		}

	case st_DIGITSLASHWSCOLONCOLONAMPM: // starts digit then slash 02/ more digits/slashes then whitespace double colons
		// 2014/07/10 06:55:38.156283 PM
		// 03/19/2012 10:11:59 PM
		// 3/1/2012 10:11:59 PM
		// 03/1/2012 10:11:59 PM
		// 3/01/2012 10:11:59 PM

		if firstSlash == 4 {
			for _, layout := range []string{"2006/01/02 03:04:05 PM", "2006/1/02 03:04:05 PM", "2006/01/2 03:04:05 PM", "2006/1/2 03:04:05 PM",
				"2006/01/02 3:04:05 PM", "2006/1/02 3:04:05 PM", "2006/01/2 3:04:05 PM", "2006/1/2 3:04:05 PM"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 03:04:05 PM", "1/02/2006 03:04:05 PM", "01/2/2006 03:04:05 PM", "1/2/2006 03:04:05 PM"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		}

	case st_WEEKDAYCOMMADELTA:
		// Monday, 02 Jan 2006 15:04:05 -0700
		// Monday, 02 Jan 2006 15:04:05 +0100
		return parse("Monday, 02 Jan 2006 15:04:05 -0700", datestr, loc)
	case st_WEEKDAYABBREVCOMMA: // Starts alpha then comma
		// Mon, 02-Jan-06 15:04:05 MST
		// Mon, 02 Jan 2006 15:04:05 MST
		return parse("Mon, 02 Jan 2006 15:04:05 MST", datestr, loc)
	case st_WEEKDAYABBREVCOMMADELTA:
		// Mon, 02 Jan 2006 15:04:05 -0700
		// Thu, 13 Jul 2017 08:58:40 +0100
		// RFC1123Z    = "Mon, 02 Jan 2006 15:04:05 -0700" // RFC1123 with numeric zone
		return parse("Mon, 02 Jan 2006 15:04:05 -0700", datestr, loc)
	case st_WEEKDAYABBREVCOMMADELTAZONE:
		// Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
		return parse("Mon, 02 Jan 2006 15:04:05 -0700 (CEST)", datestr, loc)
	}

	return time.Time{}, fmt.Errorf("Could not find date format for %s", datestr)
}

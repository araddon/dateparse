package dateparse

import (
	"fmt"
	"strconv"
	"time"
	"unicode"
)

type DateState int

const (
	// C = Comma
	// O = Colon
	// E = Period/Dot
	// A = Alpha
	// N = Digits/Numeric
	// S = Slash  /
	// P = Plus +
	// m = Minus, Dash, -
	// T = T
	// Z = Z
	// M = AM/PM
	// W = Whitespace
	ST_START DateState = iota
	ST_DIGIT
	ST_DIGITDASH
	ST_DIGITDASHWS
	ST_DIGITDASHWSALPHA
	ST_DIGITDASHWSDOT
	ST_DIGITDASHWSDOTALPHA
	ST_DIGITDASHWSDOTPLUS
	ST_DIGITDASHWSDOTPLUSALPHA
	ST_DIGITDASHT
	ST_DIGITDASHTZ
	ST_DIGITDASHTZDIGIT
	ST_DIGITDASHTDELTA
	ST_DIGITDASHTDELTACOLON
	ST_DIGITSLASH
	ST_DIGITSLASHWS
	ST_DIGITSLASHWSCOLON
	ST_DIGITSLASHWSCOLONAMPM
	ST_DIGITSLASHWSCOLONCOLON
	ST_DIGITSLASHWSCOLONCOLONAMPM
	ST_DIGITALPHA
	ST_ALPHA
	ST_ALPHAWS
	ST_ALPHAWSDIGITCOMMA
	ST_ALPHAWSALPHA
	ST_ALPHAWSALPHACOLON
	ST_ALPHAWSALPHACOLONOFFSET
	ST_ALPHAWSALPHACOLONALPHA
	ST_ALPHAWSALPHACOLONALPHAOFFSET
	ST_ALPHAWSALPHACOLONALPHAOFFSETALPHA
	ST_ALPHACOMMA
	ST_WEEKDAYCOMMA
	ST_WEEKDAYCOMMADELTA
	ST_WEEKDAYABBREVCOMMA
	ST_WEEKDAYABBREVCOMMADELTA
	ST_WEEKDAYABBREVCOMMADELTAZONE
)

var (
	shortDates = []string{"01/02/2006", "1/2/2006", "06/01/02", "01/02/06", "1/2/06"}
)

// Parse a date, and panic if it can't be parsed
func MustParse(datestr string) time.Time {
	t, err := ParseAny(datestr)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// Given an unknown date format, detect the type, parse, return time
func ParseAny(datestr string) (time.Time, error) {

	state := ST_START

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
		case ST_START:
			if unicode.IsDigit(r) {
				state = ST_DIGIT
			} else if unicode.IsLetter(r) {
				state = ST_ALPHA
			}
		case ST_DIGIT: // starts digits
			if unicode.IsDigit(r) {
				continue
			} else if unicode.IsLetter(r) {
				state = ST_DIGITALPHA
				continue
			}
			switch r {
			case '-', '\u2212':
				state = ST_DIGITDASH
			case '/':
				state = ST_DIGITSLASH
				firstSlash = i
			}
		case ST_DIGITDASH: // starts digit then dash 02-
			// 2006-01-02T15:04:05Z07:00
			// 2017-06-25T17:46:57.45706582-07:00
			// 2006-01-02T15:04:05.999999999Z07:00
			// 2006-01-02T15:04:05+0000
			// 2012-08-03 18:31:59.257000000
			// 2014-04-26 17:24:37.3186369
			// 2017-01-27 00:07:31.945167
			// 2016-03-14 00:00:00.000
			// 2014-05-11 08:20:13,787
			// 2006-01-02
			// 2013-04-01 22:43:22
			// 2014-04-26 05:24:37 PM
			switch {
			case r == ' ':
				state = ST_DIGITDASHWS
			case r == 'T':
				state = ST_DIGITDASHT
			}
		case ST_DIGITDASHWS: // starts digit then dash 02- then whitespace
			// 2013-04-01 22:43:22
			// 2014-05-11 08:20:13,787
			// 2014-04-26 05:24:37 PM
			// 2014-12-16 06:20:00 UTC
			// 2015-02-18 00:12:00 +0000 UTC
			// 2015-06-25 01:25:37.115208593 +0000 UTC
			switch r {
			case 'A', 'P':
				if len(datestr) == len("2014-04-26 03:24:37 PM") {
					return time.Parse("2006-01-02 03:04:05 PM", datestr)
				}
			case ',':
				if len(datestr) == len("2014-05-11 08:20:13,787") {
					// go doesn't seem to parse this one natively?   or did i miss it?
					t, err := time.Parse("2006-01-02 03:04:05", datestr[:i])
					if err == nil {
						ms, err := strconv.Atoi(datestr[i+1:])
						if err == nil {
							return time.Unix(0, t.UnixNano()+int64(ms)*1e6), nil
						}
					}
					return t, err
				}
			case '.':
				state = ST_DIGITDASHWSDOT
			default:
				if unicode.IsLetter(r) {
					// 2014-12-16 06:20:00 UTC
					state = ST_DIGITDASHWSALPHA
					break iterRunes
				}
			}
		case ST_DIGITDASHWSDOT:
			// 2014-04-26 17:24:37.3186369
			// 2017-01-27 00:07:31.945167
			// 2012-08-03 18:31:59.257000000
			// 2016-03-14 00:00:00.000
			if unicode.IsLetter(r) {
				// 2014-12-16 06:20:00.000 UTC
				state = ST_DIGITDASHWSDOTALPHA
				break iterRunes
			} else if r == '+' || r == '-' {
				state = ST_DIGITDASHWSDOTPLUS
			}
		case ST_DIGITDASHWSDOTPLUS:
			// 2014-04-26 17:24:37.3186369
			// 2017-01-27 00:07:31.945167
			// 2012-08-03 18:31:59.257000000
			// 2016-03-14 00:00:00.000
			if unicode.IsLetter(r) {
				// 2014-12-16 06:20:00.000 UTC
				state = ST_DIGITDASHWSDOTPLUSALPHA
				break iterRunes
			}

		case ST_DIGITDASHT: // starts digit then dash 02-  then T
			// ST_DIGITDASHT
			// 2006-01-02T15:04:05
			// ST_DIGITDASHTZ
			// 2006-01-02T15:04:05.999999999Z
			// 2006-01-02T15:04:05.99999999Z
			// 2006-01-02T15:04:05.9999999Z
			// 2006-01-02T15:04:05.999999Z
			// 2006-01-02T15:04:05.99999Z
			// 2006-01-02T15:04:05.9999Z
			// 2006-01-02T15:04:05.999Z
			// 2006-01-02T15:04:05.99Z
			// 2009-08-12T22:15Z
			// ST_DIGITDASHTZDIGIT
			// 2006-01-02T15:04:05.999999999Z07:00
			// 2006-01-02T15:04:05Z07:00
			// With another dash aka time-zone at end
			// ST_DIGITDASHTDELTA
			//   ST_DIGITDASHTDELTACOLON
			//     2017-06-25T17:46:57.45706582-07:00
			//     2017-06-25T17:46:57+04:00
			// 2006-01-02T15:04:05+0000
			switch r {
			case '-', '+':
				state = ST_DIGITDASHTDELTA
			case 'Z':
				state = ST_DIGITDASHTZ
			}
		case ST_DIGITDASHTZ:
			if unicode.IsDigit(r) {
				state = ST_DIGITDASHTZDIGIT
			}
		case ST_DIGITDASHTDELTA:
			if r == ':' {
				state = ST_DIGITDASHTDELTACOLON
			}
		case ST_DIGITSLASH: // starts digit then slash 02/
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
				state = ST_DIGITSLASHWS
			}
		case ST_DIGITSLASHWS: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			switch r {
			case ':':
				state = ST_DIGITSLASHWSCOLON
			}
		case ST_DIGITSLASHWSCOLON: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 3/1/2012 10:11:59 AM
			switch r {
			case ':':
				state = ST_DIGITSLASHWSCOLONCOLON
			case 'A', 'P':
				state = ST_DIGITSLASHWSCOLONAMPM
			}
		case ST_DIGITSLASHWSCOLONCOLON: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 3/1/2012 10:11:59 AM
			switch r {
			case 'A', 'P':
				state = ST_DIGITSLASHWSCOLONCOLONAMPM
			}
		case ST_DIGITALPHA:
			// 12 Feb 2006, 19:17
			// 12 Feb 2006, 19:17:22
			switch {
			case len(datestr) == len("02 Jan 2006, 15:04"):
				return time.Parse("02 Jan 2006, 15:04", datestr)
			case len(datestr) == len("02 Jan 2006, 15:04:05"):
				return time.Parse("02 Jan 2006, 15:04:05", datestr)
			}
		case ST_ALPHA: // starts alpha
			// ST_ALPHAWS
			//  Mon Jan _2 15:04:05 2006
			//  Mon Jan _2 15:04:05 MST 2006
			//  Mon Jan 02 15:04:05 -0700 2006
			//  Mon Aug 10 15:44:11 UTC+0100 2015
			//  Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			//  ST_ALPHAWSDIGITCOMMA
			//    May 8, 2009 5:57:51 PM
			//
			// ST_WEEKDAYCOMMA
			//   Monday, 02-Jan-06 15:04:05 MST
			//   ST_WEEKDAYCOMMADELTA
			//     Monday, 02 Jan 2006 15:04:05 -0700
			//     Monday, 02 Jan 2006 15:04:05 +0100
			// ST_WEEKDAYABBREVCOMMA
			//   Mon, 02-Jan-06 15:04:05 MST
			//   Mon, 02 Jan 2006 15:04:05 MST
			//   ST_WEEKDAYABBREVCOMMADELTA
			//     Mon, 02 Jan 2006 15:04:05 -0700
			//     Thu, 13 Jul 2017 08:58:40 +0100
			//     ST_WEEKDAYABBREVCOMMADELTAZONE
			//       Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			switch {
			case unicode.IsLetter(r):
				continue
			case r == ' ':
				state = ST_ALPHAWS
			case r == ',':
				if i == 3 {
					state = ST_WEEKDAYABBREVCOMMA
				} else {
					state = ST_WEEKDAYCOMMA
				}
			}
		case ST_WEEKDAYCOMMA: // Starts alpha then comma
			// Mon, 02-Jan-06 15:04:05 MST
			// Mon, 02 Jan 2006 15:04:05 MST
			// ST_WEEKDAYCOMMADELTA
			//   Monday, 02 Jan 2006 15:04:05 -0700
			//   Monday, 02 Jan 2006 15:04:05 +0100
			switch {
			case r == '-':
				if i < 15 {
					return time.Parse("Monday, 02-Jan-06 15:04:05 MST", datestr)
				} else {
					state = ST_WEEKDAYCOMMADELTA
				}
			case r == '+':
				state = ST_WEEKDAYCOMMADELTA
			}
		case ST_WEEKDAYABBREVCOMMA: // Starts alpha then comma
			// Mon, 02-Jan-06 15:04:05 MST
			// Mon, 02 Jan 2006 15:04:05 MST
			// ST_WEEKDAYABBREVCOMMADELTA
			//   Mon, 02 Jan 2006 15:04:05 -0700
			//   Thu, 13 Jul 2017 08:58:40 +0100
			//   ST_WEEKDAYABBREVCOMMADELTAZONE
			//     Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			switch {
			case r == '-':
				if i < 15 {
					return time.Parse("Mon, 02-Jan-06 15:04:05 MST", datestr)
				} else {
					state = ST_WEEKDAYABBREVCOMMADELTA
				}
			case r == '+':
				state = ST_WEEKDAYABBREVCOMMADELTA
			}

		case ST_WEEKDAYABBREVCOMMADELTA:
			// ST_WEEKDAYABBREVCOMMADELTA
			//   Mon, 02 Jan 2006 15:04:05 -0700
			//   Thu, 13 Jul 2017 08:58:40 +0100
			//   ST_WEEKDAYABBREVCOMMADELTAZONE
			//     Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			if r == '(' {
				state = ST_WEEKDAYABBREVCOMMADELTAZONE
			}

		case ST_ALPHAWS: // Starts alpha then whitespace
			// Mon Jan _2 15:04:05 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			// Mon Aug 10 15:44:11 UTC+0100 2015
			switch {
			case unicode.IsLetter(r):
				state = ST_ALPHAWSALPHA
			case unicode.IsDigit(r):
				state = ST_ALPHAWSDIGITCOMMA
			}

		case ST_ALPHAWSDIGITCOMMA: // Starts Alpha, whitespace, digit, comma
			// May 8, 2009 5:57:51 PM
			return time.Parse("Jan 2, 2006 3:04:05 PM", datestr)

		case ST_ALPHAWSALPHA: // Alpha, whitespace, alpha
			// Mon Jan _2 15:04:05 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == ':' {
				state = ST_ALPHAWSALPHACOLON
			}
		case ST_ALPHAWSALPHACOLON: // Alpha, whitespace, alpha, :
			// Mon Jan _2 15:04:05 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if unicode.IsLetter(r) {
				state = ST_ALPHAWSALPHACOLONALPHA
			} else if r == '-' || r == '+' {
				state = ST_ALPHAWSALPHACOLONOFFSET
			}
		case ST_ALPHAWSALPHACOLONALPHA: // Alpha, whitespace, alpha, :, alpha
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == '+' {
				state = ST_ALPHAWSALPHACOLONALPHAOFFSET
			}
		case ST_ALPHAWSALPHACOLONALPHAOFFSET: // Alpha, whitespace, alpha, : , alpha, offset, ?
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if unicode.IsLetter(r) {
				state = ST_ALPHAWSALPHACOLONALPHAOFFSETALPHA
			}
		default:
			break iterRunes
		}
	}

	switch state {
	case ST_DIGIT:
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
			return time.Parse("20060102", datestr)
		} else if len(datestr) == len("2014") {
			return time.Parse("2006", datestr)
		} else {
			if secs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				return time.Unix(secs, 0), nil
			}
		}
	case ST_DIGITDASH: // starts digit then dash 02-
		// 2006-01-02
		// 2006-01
		if len(datestr) == len("2014-04-26") {
			return time.Parse("2006-01-02", datestr)
		} else if len(datestr) == len("2014-04") {
			return time.Parse("2006-01", datestr)
		}
	case ST_DIGITDASHTDELTA:
		// 2006-01-02T15:04:05+0000
		return time.Parse("2006-01-02T15:04:05-0700", datestr)

	case ST_DIGITDASHTDELTACOLON:
		// With another +/- time-zone at end
		// 2006-01-02T15:04:05.999999999+07:00
		// 2006-01-02T15:04:05.999999999-07:00
		// 2006-01-02T15:04:05.999999+07:00
		// 2006-01-02T15:04:05.999999-07:00
		// 2006-01-02T15:04:05.999+07:00
		// 2006-01-02T15:04:05.999-07:00
		// 2006-01-02T15:04:05+07:00
		// 2006-01-02T15:04:05-07:00
		return time.Parse("2006-01-02T15:04:05-07:00", datestr)

	case ST_DIGITDASHT: // starts digit then dash 02-  then T
		// 2006-01-02T15:04:05.999999
		// 2006-01-02T15:04:05.999999
		return time.Parse("2006-01-02T15:04:05", datestr)

	case ST_DIGITDASHTZDIGIT:
		// With a time-zone at end after Z
		// 2006-01-02T15:04:05.999999999Z07:00
		// 2006-01-02T15:04:05Z07:00
		// RFC3339     = "2006-01-02T15:04:05Z07:00"
		// RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
		return time.Time{}, fmt.Errorf("RFC339 Dates may not contain both Z & Offset for %q see https://github.com/golang/go/issues/5294", datestr)

	case ST_DIGITDASHTZ: // starts digit then dash 02-  then T Then Z
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
			return time.Parse("2006-01-02T15:04Z", datestr)
		default:
			return time.Parse("2006-01-02T15:04:05Z", datestr)
		}
	case ST_DIGITDASHWS: // starts digit then dash 02-  then whitespace   1 << 2  << 5 + 3
		// 2013-04-01 22:43:22
		// 2006-01-02 15:04:05 -0700
		// 2006-01-02 15:04:05 -07:00
		switch len(datestr) {
		case len("2006-01-02 15:04:05"):
			return time.Parse("2006-01-02 15:04:05", datestr)
		case len("2006-01-02 15:04:05 -0700"):
			return time.Parse("2006-01-02 15:04:05 -0700", datestr)
		case len("2006-01-02 15:04:05 -07:00"):
			return time.Parse("2006-01-02 15:04:05 -07:00", datestr)
		}
	case ST_DIGITDASHWSALPHA: // starts digit then dash 02-  then whitespace   1 << 2  << 5 + 3
		// 2014-12-16 06:20:00 UTC
		// 2015-02-18 00:12:00 +0000 UTC
		// 2015-06-25 01:25:37.115208593 +0000 UTC
		switch len(datestr) {
		case len("2006-01-02 15:04:05 UTC"):
			t, err := time.Parse("2006-01-02 15:04:05 UTC", datestr)
			if err == nil {
				return t, nil
			}
			return time.Parse("2006-01-02 15:04:05 GMT", datestr)
		case len("2015-02-18 00:12:00 +0000 UTC"):
			t, err := time.Parse("2006-01-02 15:04:05 -0700 UTC", datestr)
			if err == nil {
				return t, nil
			}
			return time.Parse("2006-01-02 15:04:05 -0700 GMT", datestr)
		}
	case ST_DIGITDASHWSDOT:
		// 2012-08-03 18:31:59.257000000
		// 2014-04-26 17:24:37.3186369
		// 2017-01-27 00:07:31.945167
		// 2016-03-14 00:00:00.000
		return time.Parse("2006-01-02 15:04:05", datestr)

	case ST_DIGITDASHWSDOTALPHA:
		// 2012-08-03 18:31:59.257000000 UTC
		// 2014-04-26 17:24:37.3186369 UTC
		// 2017-01-27 00:07:31.945167 UTC
		// 2016-03-14 00:00:00.000 UTC
		return time.Parse("2006-01-02 15:04:05 UTC", datestr)

	case ST_DIGITDASHWSDOTPLUS:
		// 2012-08-03 18:31:59.257000000 +0000
		// 2014-04-26 17:24:37.3186369 +0000
		// 2017-01-27 00:07:31.945167 +0000
		// 2016-03-14 00:00:00.000 +0000
		return time.Parse("2006-01-02 15:04:05 -0700", datestr)

	case ST_DIGITDASHWSDOTPLUSALPHA:
		// 2012-08-03 18:31:59.257000000 +0000 UTC
		// 2014-04-26 17:24:37.3186369 +0000 UTC
		// 2017-01-27 00:07:31.945167 +0000 UTC
		// 2016-03-14 00:00:00.000 +0000 UTC
		return time.Parse("2006-01-02 15:04:05 -0700 UTC", datestr)
		// if err == nil {
		// 	return t, nil
		// }
		// return time.Parse("2006-01-02 15:04:05 -0700 GMT", datestr)

	case ST_ALPHAWSALPHACOLON:
		// Mon Jan _2 15:04:05 2006
		return time.Parse(time.ANSIC, datestr)

	case ST_ALPHAWSALPHACOLONOFFSET:
		// Mon Jan 02 15:04:05 -0700 2006
		return time.Parse(time.RubyDate, datestr)

	case ST_ALPHAWSALPHACOLONALPHA:
		// Mon Jan _2 15:04:05 MST 2006
		return time.Parse(time.UnixDate, datestr)

	case ST_ALPHAWSALPHACOLONALPHAOFFSET:
		// Mon Aug 10 15:44:11 UTC+0100 2015
		return time.Parse("Mon Jan 02 15:04:05 MST-0700 2006", datestr)

	case ST_ALPHAWSALPHACOLONALPHAOFFSETALPHA:
		// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
		if len(datestr) > len("Mon Jan 02 2006 15:04:05 MST-0700") {
			// What effing time stamp is this?
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			dateTmp := datestr[:33]
			return time.Parse("Mon Jan 02 2006 15:04:05 MST-0700", dateTmp)
		}
	case ST_DIGITSLASH: // starts digit then slash 02/ (but nothing else)
		// 3/1/2014
		// 10/13/2014
		// 01/02/2006
		// 2014/10/13
		if firstSlash == 4 {
			if len(datestr) == len("2006/01/02") {
				return time.Parse("2006/01/02", datestr)
			} else {
				return time.Parse("2006/1/2", datestr)
			}
		} else {
			for _, parseFormat := range shortDates {
				if t, err := time.Parse(parseFormat, datestr); err == nil {
					return t, nil
				}
			}
		}

	case ST_DIGITSLASHWSCOLON: // starts digit then slash 02/ more digits/slashes then whitespace
		// 4/8/2014 22:05
		// 04/08/2014 22:05
		// 2014/4/8 22:05
		// 2014/04/08 22:05

		if firstSlash == 4 {
			for _, layout := range []string{"2006/01/02 15:04", "2006/1/2 15:04", "2006/01/2 15:04", "2006/1/02 15:04"} {
				if t, err := time.Parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 15:04", "01/2/2006 15:04", "1/02/2006 15:04", "1/2/2006 15:04"} {
				if t, err := time.Parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		}

	case ST_DIGITSLASHWSCOLONAMPM: // starts digit then slash 02/ more digits/slashes then whitespace
		// 4/8/2014 22:05 PM
		// 04/08/2014 22:05 PM
		// 04/08/2014 1:05 PM
		// 2014/4/8 22:05 PM
		// 2014/04/08 22:05 PM

		if firstSlash == 4 {
			for _, layout := range []string{"2006/01/02 03:04 PM", "2006/01/2 03:04 PM", "2006/1/02 03:04 PM", "2006/1/2 03:04 PM",
				"2006/01/02 3:04 PM", "2006/01/2 3:04 PM", "2006/1/02 3:04 PM", "2006/1/2 3:04 PM"} {
				if t, err := time.Parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 03:04 PM", "01/2/2006 03:04 PM", "1/02/2006 03:04 PM", "1/2/2006 03:04 PM",
				"01/02/2006 3:04 PM", "01/2/2006 3:04 PM", "1/02/2006 3:04 PM", "1/2/2006 3:04 PM"} {
				if t, err := time.Parse(layout, datestr); err == nil {
					return t, nil
				}

			}
		}

	case ST_DIGITSLASHWSCOLONCOLON: // starts digit then slash 02/ more digits/slashes then whitespace double colons
		// 2014/07/10 06:55:38.156283
		// 03/19/2012 10:11:59
		// 3/1/2012 10:11:59
		// 03/1/2012 10:11:59
		// 3/01/2012 10:11:59
		if firstSlash == 4 {
			for _, layout := range []string{"2006/01/02 15:04:05", "2006/1/02 15:04:05", "2006/01/2 15:04:05", "2006/1/2 15:04:05"} {
				if t, err := time.Parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 15:04:05", "1/02/2006 15:04:05", "01/2/2006 15:04:05", "1/2/2006 15:04:05"} {
				if t, err := time.Parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		}

	case ST_DIGITSLASHWSCOLONCOLONAMPM: // starts digit then slash 02/ more digits/slashes then whitespace double colons
		// 2014/07/10 06:55:38.156283 PM
		// 03/19/2012 10:11:59 PM
		// 3/1/2012 10:11:59 PM
		// 03/1/2012 10:11:59 PM
		// 3/01/2012 10:11:59 PM

		if firstSlash == 4 {
			for _, layout := range []string{"2006/01/02 03:04:05 PM", "2006/1/02 03:04:05 PM", "2006/01/2 03:04:05 PM", "2006/1/2 03:04:05 PM",
				"2006/01/02 3:04:05 PM", "2006/1/02 3:04:05 PM", "2006/01/2 3:04:05 PM", "2006/1/2 3:04:05 PM"} {
				if t, err := time.Parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 03:04:05 PM", "1/02/2006 03:04:05 PM", "01/2/2006 03:04:05 PM", "1/2/2006 03:04:05 PM"} {
				if t, err := time.Parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		}

	case ST_WEEKDAYCOMMADELTA:
		// Monday, 02 Jan 2006 15:04:05 -0700
		// Monday, 02 Jan 2006 15:04:05 +0100
		return time.Parse("Monday, 02 Jan 2006 15:04:05 -0700", datestr)
	case ST_WEEKDAYABBREVCOMMA: // Starts alpha then comma
		// Mon, 02-Jan-06 15:04:05 MST
		// Mon, 02 Jan 2006 15:04:05 MST
		return time.Parse("Mon, 02 Jan 2006 15:04:05 MST", datestr)
	case ST_WEEKDAYABBREVCOMMADELTA:
		// Mon, 02 Jan 2006 15:04:05 -0700
		// Thu, 13 Jul 2017 08:58:40 +0100
		// RFC1123Z    = "Mon, 02 Jan 2006 15:04:05 -0700" // RFC1123 with numeric zone
		return time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", datestr)
	case ST_WEEKDAYABBREVCOMMADELTAZONE:
		// Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
		return time.Parse("Mon, 02 Jan 2006 15:04:05 -0700 (CEST)", datestr)
	}

	return time.Time{}, fmt.Errorf("Could not find date format for %s", datestr)
}

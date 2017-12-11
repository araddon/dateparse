// Package dateparse parses date-strings without knowing the format
// in advance, using a fast lex based approach to eliminate shotgun
// attempts.  It leans towards US style dates when there is a conflict.
package dateparse

import (
	"fmt"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"
)

type dateState int

const (
	stateStart dateState = iota
	stateDigit
	stateDigitDash
	stateDigitDashAlpha
	stateDigitDashWs
	stateDigitDashWsWs
	stateDigitDashWsWsAMPMMaybe
	stateDigitDashWsWsOffset
	stateDigitDashWsWsOffsetAlpha
	stateDigitDashWsWsOffsetColonAlpha
	stateDigitDashWsWsOffsetColon
	stateDigitDashWsOffset
	stateDigitDashWsWsAlpha
	stateDigitDashWsPeriod
	stateDigitDashWsPeriodAlpha
	stateDigitDashWsPeriodOffset
	stateDigitDashWsPeriodOffsetAlpha
	stateDigitDashT
	stateDigitDashTZ
	stateDigitDashTZDigit
	stateDigitDashTOffset
	stateDigitDashTOffsetColon
	stateDigitSlash
	stateDigitSlashWS
	stateDigitSlashWSColon
	stateDigitSlashWSColonAMPM
	stateDigitSlashWSColonColon
	stateDigitSlashWSColonColonAMPM
	stateDigitChineseYear
	stateDigitChineseYearWs
	stateDigitAlpha
	stateAlpha
	stateAlphaWS
	stateAlphaWSDigitComma
	stateAlphaWSAlpha
	stateAlphaWSAlphaColon
	stateAlphaWSAlphaColonOffset
	stateAlphaWSAlphaColonAlpha
	stateAlphaWSAlphaColonAlphaOffset
	stateAlphaWSAlphaColonAlphaOffsetAlpha
	stateWeekdayComma
	stateWeekdayCommaOffset
	stateWeekdayAbbrevComma
	stateWeekdayAbbrevCommaOffset
	stateWeekdayAbbrevCommaOffsetZone
)

var (
	shortDates = []string{"01/02/2006", "1/2/2006", "06/01/02", "01/02/06", "1/2/06"}
)

// ParseAny parse an unknown date format, detect the layout, parse.
// Normal parse.  Equivalent Timezone rules as time.Parse()
func ParseAny(datestr string) (time.Time, error) {
	return parseTime(datestr, nil)
}

// ParseIn with Location, equivalent to time.ParseInLocation() timezone/offset
// rules.  Using location arg, if timezone/offset info exists in the
// datestring, it uses the given location rules for any zone interpretation.
// That is, MST means one thing when using America/Denver and something else
// in other locations.
func ParseIn(datestr string, loc *time.Location) (time.Time, error) {
	return parseTime(datestr, loc)
}

// ParseLocal Given an unknown date format, detect the layout,
// using time.Local, parse.
//
// Set Location to time.Local.  Same as ParseIn Location but lazily uses
// the global time.Local variable for Location argument.
//
//     denverLoc, _ := time.LoadLocation("America/Denver")
//     time.Local = denverLoc
//
//     t, err := dateparse.ParseLocal("3/1/2014")
//
// Equivalent to:
//
//     t, err := dateparse.ParseIn("3/1/2014", denverLoc)
//
func ParseLocal(datestr string) (time.Time, error) {
	return parseTime(datestr, time.Local)
}

// MustParse  parse a date, and panic if it can't be parsed.  Used for testing.
// Not recommended for most use-cases.
func MustParse(datestr string) time.Time {
	t, err := parseTime(datestr, nil)
	if err != nil {
		panic(err.Error())
	}
	return t
}

func parse(layout, datestr string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		return time.Parse(layout, datestr)
	}
	return time.ParseInLocation(layout, datestr, loc)
}

func parseTime(datestr string, loc *time.Location) (time.Time, error) {
	state := stateStart

	firstSlash := 0

	// General strategy is to read rune by rune through the date looking for
	// certain hints of what type of date we are dealing with.
	// Hopefully we only need to read about 5 or 6 bytes before
	// we figure it out and then attempt a parse
iterRunes:
	for i := 0; i < len(datestr); i++ {
		//r := rune(datestr[i])
		r, bytesConsumed := utf8.DecodeRuneInString(datestr[i:])
		if bytesConsumed > 1 {
			i += (bytesConsumed - 1)
		}

		switch state {
		case stateStart:
			if unicode.IsDigit(r) {
				state = stateDigit
			} else if unicode.IsLetter(r) {
				state = stateAlpha
			}
		case stateDigit: // starts digits
			if unicode.IsDigit(r) {
				continue
			} else if unicode.IsLetter(r) {
				if r == '年' {
					// Chinese Year
					state = stateDigitChineseYear
					continue
				}
				state = stateDigitAlpha
				continue
			}
			switch r {
			case '-', '\u2212':
				state = stateDigitDash
			case '/':
				state = stateDigitSlash
				firstSlash = i
			}
		case stateDigitDash: // starts digit then dash 02-
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
				state = stateDigitDashWs
			case r == 'T':
				state = stateDigitDashT
			default:
				if unicode.IsLetter(r) {
					state = stateDigitDashAlpha
					break iterRunes
				}
			}
		case stateDigitDashWs:
			// 2013-04-01 22:43:22
			// 2014-05-11 08:20:13,787
			// stateDigitDashWsWs
			//   2014-04-26 05:24:37 PM
			//   2014-12-16 06:20:00 UTC
			//   2015-02-18 00:12:00 +0000 UTC
			//   2006-01-02 15:04:05 -0700
			//   2006-01-02 15:04:05 -07:00
			// stateDigitDashWsOffset
			//   2017-07-19 03:21:51+00:00
			// stateDigitDashWsPeriod
			//   2014-04-26 17:24:37.3186369
			//   2017-01-27 00:07:31.945167
			//   2012-08-03 18:31:59.257000000
			//   2016-03-14 00:00:00.000
			//   stateDigitDashWsPeriodOffset
			//     2017-01-27 00:07:31.945167 +0000
			//     2016-03-14 00:00:00.000 +0000
			//     stateDigitDashWsPeriodOffsetAlpha
			//       2017-01-27 00:07:31.945167 +0000 UTC
			//       2016-03-14 00:00:00.000 +0000 UTC
			//   stateDigitDashWsPeriodAlpha
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
				state = stateDigitDashWsOffset
			case '.':
				state = stateDigitDashWsPeriod
			case ' ':
				state = stateDigitDashWsWs
			}

		case stateDigitDashWsWs:
			// stateDigitDashWsWsAlpha
			//   2014-12-16 06:20:00 UTC
			//   stateDigitDashWsWsAMPMMaybe
			//     2014-04-26 05:24:37 PM
			// stateDigitDashWsWsOffset
			//   2006-01-02 15:04:05 -0700
			//   stateDigitDashWsWsOffsetColon
			//     2006-01-02 15:04:05 -07:00
			//     stateDigitDashWsWsOffsetColonAlpha
			//       2015-02-18 00:12:00 +00:00 UTC
			//   stateDigitDashWsWsOffsetAlpha
			//     2015-02-18 00:12:00 +0000 UTC
			switch r {
			case 'A', 'P':
				state = stateDigitDashWsWsAMPMMaybe
			case '+', '-':
				state = stateDigitDashWsWsOffset
			default:
				if unicode.IsLetter(r) {
					// 2014-12-16 06:20:00 UTC
					state = stateDigitDashWsWsAlpha
					break iterRunes
				}
			}

		case stateDigitDashWsWsAMPMMaybe:
			if r == 'M' {
				return parse("2006-01-02 03:04:05 PM", datestr, loc)
			}
			state = stateDigitDashWsWsAlpha

		case stateDigitDashWsWsOffset:
			// stateDigitDashWsWsOffset
			//   2006-01-02 15:04:05 -0700
			//   stateDigitDashWsWsOffsetColon
			//     2006-01-02 15:04:05 -07:00
			//     stateDigitDashWsWsOffsetColonAlpha
			//       2015-02-18 00:12:00 +00:00 UTC
			//   stateDigitDashWsWsOffsetAlpha
			//     2015-02-18 00:12:00 +0000 UTC
			if r == ':' {
				state = stateDigitDashWsWsOffsetColon
			} else if unicode.IsLetter(r) {
				// 2015-02-18 00:12:00 +0000 UTC
				state = stateDigitDashWsWsOffsetAlpha
				break iterRunes
			}

		case stateDigitDashWsWsOffsetColon:
			// stateDigitDashWsWsOffsetColon
			//   2006-01-02 15:04:05 -07:00
			//   stateDigitDashWsWsOffsetColonAlpha
			//     2015-02-18 00:12:00 +00:00 UTC
			if unicode.IsLetter(r) {
				// 2015-02-18 00:12:00 +00:00 UTC
				state = stateDigitDashWsWsOffsetColonAlpha
				break iterRunes
			}

		case stateDigitDashWsPeriod:
			// 2014-04-26 17:24:37.3186369
			// 2017-01-27 00:07:31.945167
			// 2012-08-03 18:31:59.257000000
			// 2016-03-14 00:00:00.000
			// stateDigitDashWsPeriodOffset
			//   2017-01-27 00:07:31.945167 +0000
			//   2016-03-14 00:00:00.000 +0000
			//   stateDigitDashWsPeriodOffsetAlpha
			//     2017-01-27 00:07:31.945167 +0000 UTC
			//     2016-03-14 00:00:00.000 +0000 UTC
			// stateDigitDashWsPeriodAlpha
			//   2014-12-16 06:20:00.000 UTC
			if unicode.IsLetter(r) {
				// 2014-12-16 06:20:00.000 UTC
				state = stateDigitDashWsPeriodAlpha
				break iterRunes
			} else if r == '+' || r == '-' {
				state = stateDigitDashWsPeriodOffset
			}
		case stateDigitDashWsPeriodOffset:
			// 2017-01-27 00:07:31.945167 +0000
			// 2016-03-14 00:00:00.000 +0000
			// stateDigitDashWsPeriodOffsetAlpha
			//   2017-01-27 00:07:31.945167 +0000 UTC
			//   2016-03-14 00:00:00.000 +0000 UTC
			if unicode.IsLetter(r) {
				// 2014-12-16 06:20:00.000 UTC
				// 2017-01-27 00:07:31.945167 +0000 UTC
				// 2016-03-14 00:00:00.000 +0000 UTC
				state = stateDigitDashWsPeriodOffsetAlpha
				break iterRunes
			}
		case stateDigitDashT: // starts digit then dash 02-  then T
			// stateDigitDashT
			// 2006-01-02T15:04:05
			// stateDigitDashTZ
			// 2006-01-02T15:04:05.999999999Z
			// 2006-01-02T15:04:05.99999999Z
			// 2006-01-02T15:04:05.9999999Z
			// 2006-01-02T15:04:05.999999Z
			// 2006-01-02T15:04:05.99999Z
			// 2006-01-02T15:04:05.9999Z
			// 2006-01-02T15:04:05.999Z
			// 2006-01-02T15:04:05.99Z
			// 2009-08-12T22:15Z
			// stateDigitDashTZDigit
			// 2006-01-02T15:04:05.999999999Z07:00
			// 2006-01-02T15:04:05Z07:00
			// With another dash aka time-zone at end
			// stateDigitDashTOffset
			//   stateDigitDashTOffsetColon
			//     2017-06-25T17:46:57.45706582-07:00
			//     2017-06-25T17:46:57+04:00
			// 2006-01-02T15:04:05+0000
			switch r {
			case '-', '+':
				state = stateDigitDashTOffset
			case 'Z':
				state = stateDigitDashTZ
			}
		case stateDigitDashTZ:
			if unicode.IsDigit(r) {
				state = stateDigitDashTZDigit
			}
		case stateDigitDashTOffset:
			if r == ':' {
				state = stateDigitDashTOffsetColon
			}
		case stateDigitSlash: // starts digit then slash 02/
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
				state = stateDigitSlashWS
			}
		case stateDigitSlashWS: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 4/8/14 22:05
			switch r {
			case ':':
				state = stateDigitSlashWSColon
			}
		case stateDigitSlashWSColon: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 4/8/14 22:05
			// 3/1/2012 10:11:59 AM
			switch r {
			case ':':
				state = stateDigitSlashWSColonColon
			case 'A', 'P':
				state = stateDigitSlashWSColonAMPM
			}
		case stateDigitSlashWSColonColon: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 4/8/14 22:05
			// 3/1/2012 10:11:59 AM
			switch r {
			case 'A', 'P':
				state = stateDigitSlashWSColonColonAMPM
			}
		case stateDigitChineseYear:
			// stateDigitChineseYear
			//   2014年04月08日
			//               weekday  %Y年%m月%e日 %A %I:%M %p
			// 2013年07月18日 星期四 10:27 上午
			if r == ' ' {
				state = stateDigitChineseYearWs
				break
			}
		case stateDigitAlpha:
			// 12 Feb 2006, 19:17
			// 12 Feb 2006, 19:17:22
			switch {
			case len(datestr) == len("02 Jan 2006, 15:04"):
				return parse("02 Jan 2006, 15:04", datestr, loc)
			case len(datestr) == len("02 Jan 2006, 15:04:05"):
				return parse("02 Jan 2006, 15:04:05", datestr, loc)
			}
		case stateAlpha: // starts alpha
			// stateAlphaWS
			//  Mon Jan _2 15:04:05 2006
			//  Mon Jan _2 15:04:05 MST 2006
			//  Mon Jan 02 15:04:05 -0700 2006
			//  Mon Aug 10 15:44:11 UTC+0100 2015
			//  Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			//  stateAlphaWSDigitComma
			//    May 8, 2009 5:57:51 PM
			//
			// stateWeekdayComma
			//   Monday, 02-Jan-06 15:04:05 MST
			//   stateWeekdayCommaOffset
			//     Monday, 02 Jan 2006 15:04:05 -0700
			//     Monday, 02 Jan 2006 15:04:05 +0100
			// stateWeekdayAbbrevComma
			//   Mon, 02-Jan-06 15:04:05 MST
			//   Mon, 02 Jan 2006 15:04:05 MST
			//   stateWeekdayAbbrevCommaOffset
			//     Mon, 02 Jan 2006 15:04:05 -0700
			//     Thu, 13 Jul 2017 08:58:40 +0100
			//     stateWeekdayAbbrevCommaOffsetZone
			//       Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			switch {
			case unicode.IsLetter(r):
				continue
			case r == ' ':
				state = stateAlphaWS
			case r == ',':
				if i == 3 {
					state = stateWeekdayAbbrevComma
				} else {
					state = stateWeekdayComma
				}
			}
		case stateWeekdayComma: // Starts alpha then comma
			// Mon, 02-Jan-06 15:04:05 MST
			// Mon, 02 Jan 2006 15:04:05 MST
			// stateWeekdayCommaOffset
			//   Monday, 02 Jan 2006 15:04:05 -0700
			//   Monday, 02 Jan 2006 15:04:05 +0100
			switch {
			case r == '-':
				if i < 15 {
					return parse("Monday, 02-Jan-06 15:04:05 MST", datestr, loc)
				}
				state = stateWeekdayCommaOffset
			case r == '+':
				state = stateWeekdayCommaOffset
			}
		case stateWeekdayAbbrevComma: // Starts alpha then comma
			// Mon, 02-Jan-06 15:04:05 MST
			// Mon, 02 Jan 2006 15:04:05 MST
			// stateWeekdayAbbrevCommaOffset
			//   Mon, 02 Jan 2006 15:04:05 -0700
			//   Thu, 13 Jul 2017 08:58:40 +0100
			//   stateWeekdayAbbrevCommaOffsetZone
			//     Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			switch {
			case r == '-':
				if i < 15 {
					return parse("Mon, 02-Jan-06 15:04:05 MST", datestr, loc)
				}
				state = stateWeekdayAbbrevCommaOffset
			case r == '+':
				state = stateWeekdayAbbrevCommaOffset
			}

		case stateWeekdayAbbrevCommaOffset:
			// stateWeekdayAbbrevCommaOffset
			//   Mon, 02 Jan 2006 15:04:05 -0700
			//   Thu, 13 Jul 2017 08:58:40 +0100
			//   stateWeekdayAbbrevCommaOffsetZone
			//     Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			if r == '(' {
				state = stateWeekdayAbbrevCommaOffsetZone
			}

		case stateAlphaWS: // Starts alpha then whitespace
			// Mon Jan _2 15:04:05 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			// Mon Aug 10 15:44:11 UTC+0100 2015
			switch {
			case unicode.IsLetter(r):
				state = stateAlphaWSAlpha
			case unicode.IsDigit(r):
				state = stateAlphaWSDigitComma
			}

		case stateAlphaWSDigitComma: // Starts Alpha, whitespace, digit, comma
			// May 8, 2009 5:57:51 PM
			return parse("Jan 2, 2006 3:04:05 PM", datestr, loc)

		case stateAlphaWSAlpha: // Alpha, whitespace, alpha
			// Mon Jan _2 15:04:05 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == ':' {
				state = stateAlphaWSAlphaColon
			}
		case stateAlphaWSAlphaColon: // Alpha, whitespace, alpha, :
			// Mon Jan _2 15:04:05 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if unicode.IsLetter(r) {
				state = stateAlphaWSAlphaColonAlpha
			} else if r == '-' || r == '+' {
				state = stateAlphaWSAlphaColonOffset
			}
		case stateAlphaWSAlphaColonAlpha: // Alpha, whitespace, alpha, :, alpha
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == '+' {
				state = stateAlphaWSAlphaColonAlphaOffset
			}
		case stateAlphaWSAlphaColonAlphaOffset: // Alpha, whitespace, alpha, : , alpha, offset, ?
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if unicode.IsLetter(r) {
				state = stateAlphaWSAlphaColonAlphaOffsetAlpha
			}
		default:
			break iterRunes
		}
	}

	switch state {
	case stateDigit:
		// unixy timestamps ish
		//  1499979655583057426  nanoseconds
		//  1499979795437000     micro-seconds
		//  1499979795437        milliseconds
		//  1384216367189
		//  1332151919           seconds
		//  20140601             yyyymmdd
		//  2014                 yyyy
		t := time.Time{}
		if len(datestr) > len("1499979795437000") {
			if nanoSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				t = time.Unix(0, nanoSecs)
			}
		} else if len(datestr) > len("1499979795437") {
			if microSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				t = time.Unix(0, microSecs*1000)
			}
		} else if len(datestr) > len("1332151919") {
			if miliSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				t = time.Unix(0, miliSecs*1000*1000)
			}
		} else if len(datestr) == len("20140601") {
			return parse("20060102", datestr, loc)
		} else if len(datestr) == len("2014") {
			return parse("2006", datestr, loc)
		}
		if t.IsZero() {
			if secs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				if secs < 0 {
					// Now, for unix-seconds we aren't going to guess a lot
					// nothing before unix-epoch
				} else {
					t = time.Unix(secs, 0)
				}
			}
		}
		if !t.IsZero() {
			if loc == nil {
				return t, nil
			}
			return t.In(loc), nil
		}

	case stateDigitDash: // starts digit then dash 02-
		// 2006-01-02
		// 2006-01
		if len(datestr) == len("2014-04-26") {
			return parse("2006-01-02", datestr, loc)
		} else if len(datestr) == len("2014-04") {
			return parse("2006-01", datestr, loc)
		}
	case stateDigitDashAlpha:
		// 2013-Feb-03
		return parse("2006-Jan-02", datestr, loc)

	case stateDigitDashTOffset:
		// 2006-01-02T15:04:05+0000
		return parse("2006-01-02T15:04:05-0700", datestr, loc)

	case stateDigitDashTOffsetColon:
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

	case stateDigitDashT: // starts digit then dash 02-  then T
		// 2006-01-02T15:04:05.999999
		// 2006-01-02T15:04:05.999999
		return parse("2006-01-02T15:04:05", datestr, loc)

	case stateDigitDashTZDigit:
		// With a time-zone at end after Z
		// 2006-01-02T15:04:05.999999999Z07:00
		// 2006-01-02T15:04:05Z07:00
		// RFC3339     = "2006-01-02T15:04:05Z07:00"
		// RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
		return time.Time{}, fmt.Errorf("RFC339 Dates may not contain both Z & Offset for %q see https://github.com/golang/go/issues/5294", datestr)

	case stateDigitDashTZ: // starts digit then dash 02-  then T Then Z
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
	case stateDigitDashWs: // starts digit then dash 02-  then whitespace   1 << 2  << 5 + 3
		// 2013-04-01 22:43:22
		// 2013-04-01 22:43
		switch len(datestr) {
		case len("2013-04-01 22:43"):
			return parse("2006-01-02 15:04", datestr, loc)
		default:
			return parse("2006-01-02 15:04:05", datestr, loc)
		}

	case stateDigitDashWsWsOffset:
		// 2006-01-02 15:04:05 -0700
		return parse("2006-01-02 15:04:05 -0700", datestr, loc)

	case stateDigitDashWsWsOffsetColon:
		// 2006-01-02 15:04:05 -07:00
		return parse("2006-01-02 15:04:05 -07:00", datestr, loc)

	case stateDigitDashWsWsOffsetAlpha:
		// 2015-02-18 00:12:00 +0000 UTC
		t, err := parse("2006-01-02 15:04:05 -0700 UTC", datestr, loc)
		if err == nil {
			return t, nil
		}
		return parse("2006-01-02 15:04:05 +0000 GMT", datestr, loc)

	case stateDigitDashWsWsOffsetColonAlpha:
		// 2015-02-18 00:12:00 +00:00 UTC
		return parse("2006-01-02 15:04:05 -07:00 UTC", datestr, loc)

	case stateDigitDashWsOffset:
		// 2017-07-19 03:21:51+00:00
		return parse("2006-01-02 15:04:05-07:00", datestr, loc)

	case stateDigitDashWsWsAlpha:
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

	case stateDigitDashWsPeriod:
		// 2012-08-03 18:31:59.257000000
		// 2014-04-26 17:24:37.3186369
		// 2017-01-27 00:07:31.945167
		// 2016-03-14 00:00:00.000
		return parse("2006-01-02 15:04:05", datestr, loc)

	case stateDigitDashWsPeriodAlpha:
		// 2012-08-03 18:31:59.257000000 UTC
		// 2014-04-26 17:24:37.3186369 UTC
		// 2017-01-27 00:07:31.945167 UTC
		// 2016-03-14 00:00:00.000 UTC
		return parse("2006-01-02 15:04:05 UTC", datestr, loc)

	case stateDigitDashWsPeriodOffset:
		// 2012-08-03 18:31:59.257000000 +0000
		// 2014-04-26 17:24:37.3186369 +0000
		// 2017-01-27 00:07:31.945167 +0000
		// 2016-03-14 00:00:00.000 +0000
		return parse("2006-01-02 15:04:05 -0700", datestr, loc)

	case stateDigitDashWsPeriodOffsetAlpha:
		// 2012-08-03 18:31:59.257000000 +0000 UTC
		// 2014-04-26 17:24:37.3186369 +0000 UTC
		// 2017-01-27 00:07:31.945167 +0000 UTC
		// 2016-03-14 00:00:00.000 +0000 UTC
		return parse("2006-01-02 15:04:05 -0700 UTC", datestr, loc)

	case stateAlphaWSAlphaColon:
		// Mon Jan _2 15:04:05 2006
		return parse(time.ANSIC, datestr, loc)

	case stateAlphaWSAlphaColonOffset:
		// Mon Jan 02 15:04:05 -0700 2006
		return parse(time.RubyDate, datestr, loc)

	case stateAlphaWSAlphaColonAlpha:
		// Mon Jan _2 15:04:05 MST 2006
		return parse(time.UnixDate, datestr, loc)

	case stateAlphaWSAlphaColonAlphaOffset:
		// Mon Aug 10 15:44:11 UTC+0100 2015
		return parse("Mon Jan 02 15:04:05 MST-0700 2006", datestr, loc)

	case stateAlphaWSAlphaColonAlphaOffsetAlpha:
		// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
		if len(datestr) > len("Mon Jan 02 2006 15:04:05 MST-0700") {
			// What effing time stamp is this?
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			dateTmp := datestr[:33]
			return parse("Mon Jan 02 2006 15:04:05 MST-0700", dateTmp, loc)
		}
	case stateDigitSlash: // starts digit then slash 02/ (but nothing else)
		// 3/1/2014
		// 10/13/2014
		// 01/02/2006
		// 2014/10/13
		if firstSlash == 4 {
			if len(datestr) == len("2006/01/02") {
				return parse("2006/01/02", datestr, loc)
			}
			return parse("2006/1/2", datestr, loc)
		}
		for _, parseFormat := range shortDates {
			if t, err := parse(parseFormat, datestr, loc); err == nil {
				return t, nil
			}
		}

	case stateDigitSlashWSColon: // starts digit then slash 02/ more digits/slashes then whitespace
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
			for _, layout := range []string{"01/02/2006 15:04", "01/2/2006 15:04", "1/02/2006 15:04", "1/2/2006 15:04", "1/2/06 15:04", "01/02/06 15:04"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		}

	case stateDigitSlashWSColonAMPM: // starts digit then slash 02/ more digits/slashes then whitespace
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

	case stateDigitSlashWSColonColon: // starts digit then slash 02/ more digits/slashes then whitespace double colons
		// 2014/07/10 06:55:38.156283
		// 03/19/2012 10:11:59
		// 3/1/2012 10:11:59
		// 03/1/2012 10:11:59
		// 3/01/2012 10:11:59
		// 4/8/14 22:05
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

	case stateDigitSlashWSColonColonAMPM: // starts digit then slash 02/ more digits/slashes then whitespace double colons
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
	case stateDigitChineseYear:
		// stateDigitChineseYear
		//   2014年04月08日
		return parse("2006年01月02日", datestr, loc)
	case stateDigitChineseYearWs:
		return parse("2006年01月02日 15:04:05", datestr, loc)
	case stateWeekdayCommaOffset:
		// Monday, 02 Jan 2006 15:04:05 -0700
		// Monday, 02 Jan 2006 15:04:05 +0100
		return parse("Monday, 02 Jan 2006 15:04:05 -0700", datestr, loc)
	case stateWeekdayAbbrevComma: // Starts alpha then comma
		// Mon, 02-Jan-06 15:04:05 MST
		// Mon, 02 Jan 2006 15:04:05 MST
		return parse("Mon, 02 Jan 2006 15:04:05 MST", datestr, loc)
	case stateWeekdayAbbrevCommaOffset:
		// Mon, 02 Jan 2006 15:04:05 -0700
		// Thu, 13 Jul 2017 08:58:40 +0100
		// RFC1123Z    = "Mon, 02 Jan 2006 15:04:05 -0700" // RFC1123 with numeric zone
		return parse("Mon, 02 Jan 2006 15:04:05 -0700", datestr, loc)
	case stateWeekdayAbbrevCommaOffsetZone:
		// Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
		return parse("Mon, 02 Jan 2006 15:04:05 -0700 (CEST)", datestr, loc)
	}

	return time.Time{}, fmt.Errorf("Could not find date format for %s", datestr)
}

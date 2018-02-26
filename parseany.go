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

	u "github.com/araddon/gou"
)

func init() {
	u.SetupLogging("debug")
	u.SetColorOutput()
}

type dateState uint8
type timeState uint8

const (
	dateStart dateState = iota
	dateDigit
	dateDigitDash
	dateDigitDashWs
	dateDigitDashT
	dateDigitDashTZ
	dateDigitDashAlpha
	dateDigitDot
	dateDigitDotDot
	dateDigitSlash
	dateDigitSlashWS
	dateDigitSlashWSColon
	dateDigitSlashWSColonAMPM
	dateDigitSlashWSColonColon
	dateDigitSlashWSColonColonAMPM
	dateDigitChineseYear
	dateDigitChineseYearWs
	dateDigitWs
	dateDigitWsMoShort
	dateDigitWsMoShortColon
	dateDigitWsMoShortColonColon
	dateDigitWsMoShortComma
	dateAlpha
	dateAlphaWS
	dateAlphaWSDigit
	dateAlphaWSDigitComma
	dateAlphaWSDigitCommaWs
	dateAlphaWSDigitCommaWsYear
	dateAlphaWSAlpha
	dateAlphaWSAlphaColon
	dateAlphaWSAlphaColonOffset
	dateAlphaWSAlphaColonAlpha
	dateAlphaWSAlphaColonAlphaOffset
	dateAlphaWSAlphaColonAlphaOffsetAlpha
	dateWeekdayComma
	dateWeekdayCommaDash
	dateWeekdayCommaOffset
	dateWeekdayAbbrevComma
	dateWeekdayAbbrevCommaDash
	dateWeekdayAbbrevCommaOffset
	dateWeekdayAbbrevCommaOffsetZone

	// Now time ones
	timeIgnore timeState = iota
	timeStart
	timeWs
	timeWsAMPMMaybe
	timeWsAMPM
	timeWsOffset
	timeWsAlpha
	timeWsOffsetAlpha
	timeWsOffsetColonAlpha
	timeWsOffsetColon
	timeOffset
	timeOffsetColon
	timeAlpha
	timePeriod
	timePeriodAlpha
	timePeriodOffset
	timePeriodOffsetAlpha
	timeZ
	timeZDigit
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

	stateDate := dateStart
	stateTime := timeIgnore
	dateFormat := []byte(datestr)
	i := 0

	part1Len := 0
	part2Len := 0
	part3Len := 0

	// General strategy is to read rune by rune through the date looking for
	// certain hints of what type of date we are dealing with.
	// Hopefully we only need to read about 5 or 6 bytes before
	// we figure it out and then attempt a parse
iterRunes:
	for ; i < len(datestr); i++ {
		//r := rune(datestr[i])
		r, bytesConsumed := utf8.DecodeRuneInString(datestr[i:])
		if bytesConsumed > 1 {
			i += (bytesConsumed - 1)
		}

		switch stateDate {
		case dateStart:
			if unicode.IsDigit(r) {
				stateDate = dateDigit
			} else if unicode.IsLetter(r) {
				stateDate = dateAlpha
			}
		case dateDigit: // starts digits
			if unicode.IsDigit(r) {
				continue
			} else if unicode.IsLetter(r) {
				if r == '年' {
					// Chinese Year
					stateDate = dateDigitChineseYear
				}
			}
			switch r {
			case '-', '\u2212':
				stateDate = dateDigitDash
				part1Len = i
			case '/':
				stateDate = dateDigitSlash
				part1Len = i
			case '.':
				stateDate = dateDigitDot
				part1Len = i
			case ' ':
				stateDate = dateDigitWs
				part1Len = i
			}

		case dateDigitDash: // starts digit then dash 02-
			// 2006-01-02
			// dateDigitDashT
			//  2006-01-02T15:04:05Z07:00
			//  2017-06-25T17:46:57.45706582-07:00
			//  2006-01-02T15:04:05.999999999Z07:00
			//  2006-01-02T15:04:05+0000
			// dateDigitDashWs
			//  2012-08-03 18:31:59.257000000
			//  2014-04-26 17:24:37.3186369
			//  2017-01-27 00:07:31.945167
			//  2016-03-14 00:00:00.000
			//  2014-05-11 08:20:13,787
			//  2017-07-19 03:21:51+00:00
			//  2013-04-01 22:43:22
			//  2014-04-26 05:24:37 PM
			// dateDigitDashAlpha
			//  2013-Feb-03
			switch {
			case r == '-':
				part2Len = i - part1Len - 1
			case r == ' ':
				part3Len = i - part1Len - part2Len - 1 - 1
				stateDate = dateDigitDashWs
				stateTime = timeStart
				break iterRunes
			case r == 'T':
				stateDate = dateDigitDashT
				stateTime = timeStart
				break iterRunes
			default:
				if unicode.IsLetter(r) {
					stateDate = dateDigitDashAlpha
					break iterRunes
				}
			}
		case dateDigitSlash: // starts digit then slash 02/
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
				stateDate = dateDigitSlashWS
			}
		case dateDigitSlashWS: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 4/8/14 22:05
			switch r {
			case ':':
				stateDate = dateDigitSlashWSColon
			}
		case dateDigitSlashWSColon: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 4/8/14 22:05
			// 3/1/2012 10:11:59 AM
			switch r {
			case ':':
				stateDate = dateDigitSlashWSColonColon
			case 'A', 'P':
				stateDate = dateDigitSlashWSColonAMPM
			}
		case dateDigitSlashWSColonColon: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 4/8/14 22:05
			// 3/1/2012 10:11:59 AM
			switch r {
			case 'A', 'P':
				stateDate = dateDigitSlashWSColonColonAMPM
			}

		case dateDigitWs:
			// 18 January 2018
			// 8 January 2018
			//dateDigitWsMoShort
			//   02 Jan 2018 23:59
			//   02 Jan 2018 23:59:34
			// dateDigitWsMoShortComma
			//   12 Feb 2006, 19:17
			//   12 Feb 2006, 19:17:22
			if r == ' ' {
				if i <= part1Len+len(" Feb") {
					stateDate = dateDigitWsMoShort
				} else {
					break iterRunes
				}
			}
		case dateDigitWsMoShort:
			// 18 January 2018
			// 8 January 2018
			// dateDigitWsMoShort
			//  dateDigitWsMoShortColon
			//    02 Jan 2018 23:59
			//   dateDigitWsMoShortComma
			//    12 Feb 2006, 19:17
			//    12 Feb 2006, 19:17:22
			switch r {
			case ':':
				stateDate = dateDigitWsMoShortColon
			case ',':
				stateDate = dateDigitWsMoShortComma
			}
		case dateDigitWsMoShortColon:
			//  02 Jan 2018 23:59
			//  dateDigitWsMoShortColonColon
			//    02 Jan 2018 23:59:45

			switch r {
			case ':':
				stateDate = dateDigitWsMoShortColonColon
				break iterRunes
			}

		case dateDigitChineseYear:
			// dateDigitChineseYear
			//   2014年04月08日
			//               weekday  %Y年%m月%e日 %A %I:%M %p
			// 2013年07月18日 星期四 10:27 上午
			if r == ' ' {
				stateDate = dateDigitChineseYearWs
				break
			}
		case dateDigitDot:
			// 3.31.2014
			if r == '.' {
				stateDate = dateDigitDotDot
				part2Len = i
			}
		case dateDigitDotDot:
			// iterate all the way through
		case dateAlpha:
			// dateAlphaWS
			//  Mon Jan _2 15:04:05 2006
			//  Mon Jan _2 15:04:05 MST 2006
			//  Mon Jan 02 15:04:05 -0700 2006
			//  Mon Aug 10 15:44:11 UTC+0100 2015
			//  Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			//  dateAlphaWSDigit
			//    May 8, 2009 5:57:51 PM
			//
			// dateWeekdayComma
			//   Monday, 02 Jan 2006 15:04:05 MST
			//   dateWeekdayCommaDash
			//     Monday, 02-Jan-06 15:04:05 MST
			//   dateWeekdayCommaOffset
			//     Monday, 02 Jan 2006 15:04:05 -0700
			//     Monday, 02 Jan 2006 15:04:05 +0100
			// dateWeekdayAbbrevComma
			//   Mon, 02 Jan 2006 15:04:05 MST
			//   dateWeekdayAbbrevCommaDash
			//     Mon, 02-Jan-06 15:04:05 MST
			//   dateWeekdayAbbrevCommaOffset
			//     Mon, 02 Jan 2006 15:04:05 -0700
			//     Thu, 13 Jul 2017 08:58:40 +0100
			//     dateWeekdayAbbrevCommaOffsetZone
			//       Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			switch {
			case r == ' ':
				stateDate = dateAlphaWS
				dateFormat[0], dateFormat[1], dateFormat[2] = 'M', 'o', 'n'
				part1Len = i
			case r == ',':
				part1Len = i
				if i == 3 {
					stateDate = dateWeekdayAbbrevComma
					dateFormat[0], dateFormat[1], dateFormat[2] = 'M', 'o', 'n'
				} else {
					stateDate = dateWeekdayComma
					dateFormat[0], dateFormat[1], dateFormat[2], dateFormat[3], dateFormat[4], dateFormat[5] = 'M', 'o', 'n', 'd', 'a', 'y'
				}
				i++
			}
		case dateWeekdayComma: // Starts alpha then comma
			// Monday, 02 Jan 2006 15:04:05 MST
			// dateWeekdayCommaDash
			//   Monday, 02-Jan-06 15:04:05 MST
			// dateWeekdayCommaOffset
			//   Monday, 02 Jan 2006 15:04:05 -0700
			//   Monday, 02 Jan 2006 15:04:05 +0100
			switch {
			case r == '-':
				if i < 15 {
					stateDate = dateWeekdayCommaDash
					break iterRunes
				}
				stateDate = dateWeekdayCommaOffset
			case r == '+':
				stateDate = dateWeekdayCommaOffset
			}
		case dateWeekdayAbbrevComma: // Starts alpha then comma
			// Mon, 02 Jan 2006 15:04:05 MST
			// dateWeekdayAbbrevCommaDash
			//   Mon, 02-Jan-06 15:04:05 MST
			// dateWeekdayAbbrevCommaOffset
			//   Mon, 02 Jan 2006 15:04:05 -0700
			//   Thu, 13 Jul 2017 08:58:40 +0100
			//   Thu, 4 Jan 2018 17:53:36 +0000
			//   dateWeekdayAbbrevCommaOffsetZone
			//     Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			switch {
			case r == ' ' && part3Len == 0:
				part3Len = i - part1Len - 2
			case r == '-':
				if i < 15 {
					stateDate = dateWeekdayAbbrevCommaDash
					break iterRunes
				}
				stateDate = dateWeekdayAbbrevCommaOffset
			case r == '+':
				stateDate = dateWeekdayAbbrevCommaOffset
			}

		case dateWeekdayAbbrevCommaOffset:
			// dateWeekdayAbbrevCommaOffset
			//   Mon, 02 Jan 2006 15:04:05 -0700
			//   Thu, 13 Jul 2017 08:58:40 +0100
			//   Thu, 4 Jan 2018 17:53:36 +0000
			//   dateWeekdayAbbrevCommaOffsetZone
			//     Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			if r == '(' {
				stateDate = dateWeekdayAbbrevCommaOffsetZone
			}

		case dateAlphaWS: // Starts alpha then whitespace
			// Mon Jan _2 15:04:05 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			// Mon Aug 10 15:44:11 UTC+0100 2015
			switch {
			case r == ' ':
				part2Len = i - part1Len
			case unicode.IsLetter(r):
				stateDate = dateAlphaWSAlpha
			case unicode.IsDigit(r):
				stateDate = dateAlphaWSDigit
			default:
				u.Warnf("can we drop isLetter? case r=%s", string(r))
			}

		case dateAlphaWSDigit: // Starts Alpha, whitespace, digit, comma
			//  dateAlphaWSDigit
			//    May 8, 2009 5:57:51 PM
			switch {
			case r == ',':
				stateDate = dateAlphaWSDigitComma
			case unicode.IsDigit(r):
				stateDate = dateAlphaWSDigit
			default:
				u.Warnf("hm, can we drop a case here? %v", string(r))
			}
		case dateAlphaWSDigitComma:
			//          x
			//    May 8, 2009 5:57:51 PM
			switch {
			case r == ' ':
				stateDate = dateAlphaWSDigitCommaWs
			default:
				u.Warnf("hm, can we drop a case here? %v", string(r))
				return time.Time{}, fmt.Errorf("could not find format for %v expected white-space after comma", datestr)
			}
		case dateAlphaWSDigitCommaWs:
			//               x
			//    May 8, 2009 5:57:51 PM
			if !unicode.IsDigit(r) {
				stateDate = dateAlphaWSDigitCommaWsYear
				break iterRunes
			}

		case dateAlphaWSAlpha: // Alpha, whitespace, alpha
			// Mon Jan _2 15:04:05 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == ':' {
				stateDate = dateAlphaWSAlphaColon
			}
		case dateAlphaWSAlphaColon: // Alpha, whitespace, alpha, :
			// Mon Jan _2 15:04:05 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if unicode.IsLetter(r) {
				stateDate = dateAlphaWSAlphaColonAlpha
			} else if r == '-' || r == '+' {
				stateDate = dateAlphaWSAlphaColonOffset
			}
		case dateAlphaWSAlphaColonAlpha: // Alpha, whitespace, alpha, :, alpha
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == '+' {
				stateDate = dateAlphaWSAlphaColonAlphaOffset
			}
		case dateAlphaWSAlphaColonAlphaOffset: // Alpha, whitespace, alpha, : , alpha, offset, ?
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if unicode.IsLetter(r) {
				stateDate = dateAlphaWSAlphaColonAlphaOffsetAlpha
			}
		default:
			break iterRunes
		}
	}

	timeMarker := i
	time1Len := 0
	time2Len := 0
	time3Len := 0

iterTimeRunes:
	for ; i < len(datestr); i++ {
		r, bytesConsumed := utf8.DecodeRuneInString(datestr[i:])
		if bytesConsumed > 1 {
			i += (bytesConsumed - 1)
		}
		switch stateTime {
		case timeIgnore:
			// not used
		case timeStart:
			// 22:43:22
			// timeComma
			//   08:20:13,787
			// timeWs
			//   05:24:37 PM
			//   06:20:00 UTC
			//   00:12:00 +0000 UTC
			//   15:04:05 -0700
			//   15:04:05 -07:00
			// timeOffset
			//   03:21:51+00:00
			// timePeriod
			//   17:24:37.3186369
			//   00:07:31.945167
			//   18:31:59.257000000
			//   00:00:00.000
			//   timePeriodOffset
			//     00:07:31.945167 +0000
			//     00:00:00.000 +0000
			//     timePeriodOffsetAlpha
			//       00:07:31.945167 +0000 UTC
			//       00:00:00.000 +0000 UTC
			//   timePeriodAlpha
			//     06:20:00.000 UTC
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
				stateTime = timeOffset
			case '.':
				stateTime = timePeriod
			case ' ':
				stateTime = timeWs
			case ':':
				if time2Len > 0 {
					time3Len = i - timeMarker
				} else if time1Len > 0 {
					time2Len = i - timeMarker
				} else if time1Len == 0 {
					time1Len = i - timeMarker
				}
				timeMarker = i
				u.Infof("time len: i=%d r=%s  marker:%d 1:%d 2:%d 3:%d", i, string(r), timeMarker, time1Len, time2Len, time3Len)
			}
		case timeOffset:
			// With another +/- time-zone at end
			// 15:04:05.999999999+07:00
			// 15:04:05.999999999-07:00
			// 15:04:05.999999+07:00
			// 15:04:05.999999-07:00
			// 15:04:05.999+07:00
			// 15:04:05.999-07:00
			// 15:04:05+07:00
			// 15:04:05-07:00
			if r == ':' {
				stateTime = timeOffsetColon
			}
		/*
			case dateDigitDashT: // starts digit then dash 02-  then T
				// dateDigitDashT
				// 2006-01-02T15:04:05
				// dateDigitDashTZ
				// 2006-01-02T15:04:05.999999999Z
				// 2006-01-02T15:04:05.99999999Z
				// 2006-01-02T15:04:05.9999999Z
				// 2006-01-02T15:04:05.999999Z
				// 2006-01-02T15:04:05.99999Z
				// 2006-01-02T15:04:05.9999Z
				// 2006-01-02T15:04:05.999Z
				// 2006-01-02T15:04:05.99Z
				// 2009-08-12T22:15Z
				// dateDigitDashTZDigit
				// 2006-01-02T15:04:05.999999999Z07:00
				// 2006-01-02T15:04:05Z07:00
				// With another dash aka time-zone at end
				// dateDigitDashTOffset
				//   dateDigitDashTOffsetColon
				//     2017-06-25T17:46:57.45706582-07:00
				//     2017-06-25T17:46:57+04:00
				// 2006-01-02T15:04:05+0000
				switch r {
				case '-', '+':
					stateTime = dateDigitDashTOffset
				case 'Z':
					stateTime = dateDigitDashTZ
				}
		*/
		case timeWs:
			// timeAlpha
			//   06:20:00 UTC
			//   timeWsAMPMMaybe
			//     05:24:37 PM
			// timeWsOffset
			//   15:04:05 -0700
			//   timeWsOffsetColon
			//     15:04:05 -07:00
			//     timeWsOffsetColonAlpha
			//       00:12:00 +00:00 UTC
			//   timeWsOffsetAlpha
			//     00:12:00 +0000 UTC
			// timeZ
			//   15:04:05.99Z
			switch r {
			case 'A', 'P':
				// Could be AM/PM or could be PST or similar
				stateTime = timeWsAMPMMaybe
			case '+', '-':
				stateTime = timeWsOffset
			default:
				if unicode.IsLetter(r) {
					// 06:20:00 UTC
					stateTime = timeWsAlpha
					break iterTimeRunes
				}
			}

		case timeWsAMPMMaybe:
			// timeWsAMPMMaybe
			//   timeWsAMPM
			//     05:24:37 PM
			//   timeWsAlpha
			//     00:12:00 PST
			if r == 'M' {
				//return parse("2006-01-02 03:04:05 PM", datestr, loc)
				stateTime = timeWsAMPM
			} else {
				stateTime = timeWsAlpha
			}

		case timeWsOffset:
			// timeWsOffset
			//   15:04:05 -0700
			//   timeWsOffsetColon
			//     15:04:05 -07:00
			//     timeWsOffsetColonAlpha
			//       00:12:00 +00:00 UTC
			//   timeWsOffsetAlpha
			//     00:12:00 +0000 UTC
			if r == ':' {
				stateTime = timeWsOffsetColon
			} else if unicode.IsLetter(r) {
				// 2015-02-18 00:12:00 +0000 UTC
				stateTime = timeWsOffsetAlpha
				break iterTimeRunes
			}

		case timeWsOffsetColon:
			// timeWsOffsetColon
			//   15:04:05 -07:00
			//   timeWsOffsetColonAlpha
			//     2015-02-18 00:12:00 +00:00 UTC
			if unicode.IsLetter(r) {
				// 2015-02-18 00:12:00 +00:00 UTC
				stateTime = timeWsOffsetColonAlpha
				break iterTimeRunes
			}

		case timePeriod:
			// 2014-04-26 17:24:37.3186369
			// 2017-01-27 00:07:31.945167
			// 2012-08-03 18:31:59.257000000
			// 2016-03-14 00:00:00.000
			// timePeriodOffset
			//   2017-01-27 00:07:31.945167 +0000
			//   2016-03-14 00:00:00.000 +0000
			//   timePeriodOffsetAlpha
			//     2017-01-27 00:07:31.945167 +0000 UTC
			//     2016-03-14 00:00:00.000 +0000 UTC
			// timePeriodAlpha
			//   2014-12-16 06:20:00.000 UTC
			if unicode.IsLetter(r) {
				// 2014-12-16 06:20:00.000 UTC
				stateTime = timePeriodAlpha
				break iterTimeRunes
			} else if r == '+' || r == '-' {
				stateTime = timePeriodOffset
			}
		case timePeriodOffset:
			// 00:07:31.945167 +0000
			// 00:00:00.000 +0000
			// timePeriodOffsetAlpha
			//   00:07:31.945167 +0000 UTC
			//   00:00:00.000 +0000 UTC
			if unicode.IsLetter(r) {
				// 06:20:00.000 UTC
				// 00:07:31.945167 +0000 UTC
				// 00:00:00.000 +0000 UTC
				stateTime = timePeriodOffsetAlpha
				break iterTimeRunes
			}
		case timeZ:
			if unicode.IsDigit(r) {
				stateTime = timeZDigit
			}

		}
	}

	u.Infof("time %60s   marker:%2d 1:%d 2:%d 3:%d  timeFormat=%q", datestr, timeMarker, time1Len, time2Len, time3Len, string(dateFormat))

	switch stateDate {
	case dateDigit:
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

	case dateDigitDash: // starts digit then dash 02-
		// 2006-01-02
		// 2006-1-02
		// 2006-1-2
		// 2006-01-2
		// 2006-01
		for _, layout := range []string{
			"2006-01-02",
			"2006-01",
			"2006-1-2",
			"2006-01-2",
			"2006-1-02",
		} {
			if t, err := parse(layout, datestr, loc); err == nil {
				return t, nil
			}
		}

	case dateDigitDashAlpha:
		// 2013-Feb-03
		// 2013-Feb-3
		for _, layout := range []string{
			"2006-Jan-02",
			"2006-Jan-2",
		} {
			if t, err := parse(layout, datestr, loc); err == nil {
				return t, nil
			}
		}

		/*
							case dateDigitDashTOffset:
								// 2006-01-02T15:04:05+0000
								for _, layout := range []string{
									"2006-01-02T15:04:05-0700",
									"2006-01-02T15:04:5-0700",
									"2006-01-02T15:4:05-0700",
									"2006-01-02T15:4:5-0700",
									"2006-1-02T15:04:05-0700",
									"2006-1-02T15:4:05-0700",
									"2006-1-02T15:04:5-0700",
									"2006-1-02T15:4:5-0700",
									"2006-01-2T15:04:05-0700",
									"2006-01-2T15:04:5-0700",
									"2006-01-2T15:4:05-0700",
									"2006-01-2T15:4:5-0700",
									"2006-1-2T15:04:05-0700",
									"2006-1-2T15:04:5-0700",
									"2006-1-2T15:4:05-0700",
									"2006-1-2T15:4:5-0700",
								} {
									if t, err := parse(layout, datestr, loc); err == nil {
										return t, nil
									}
								}

							case dateDigitDashTOffsetColon:
								// With another +/- time-zone at end
								// 2006-01-02T15:04:05.999999999+07:00
								// 2006-01-02T15:04:05.999999999-07:00
								// 2006-01-02T15:04:05.999999+07:00
								// 2006-01-02T15:04:05.999999-07:00
								// 2006-01-02T15:04:05.999+07:00
								// 2006-01-02T15:04:05.999-07:00
								// 2006-01-02T15:04:05+07:00
								// 2006-01-02T15:04:05-07:00
								for _, layout := range []string{
									"2006-01-02T15:04:05-07:00",
									"2006-01-02T15:04:5-07:00",
									"2006-01-02T15:4:05-07:00",
									"2006-01-02T15:4:5-07:00",
									"2006-1-02T15:04:05-07:00",
									"2006-1-02T15:4:05-07:00",
									"2006-1-02T15:04:5-07:00",
									"2006-1-02T15:4:5-07:00",
									"2006-01-2T15:04:05-07:00",
									"2006-01-2T15:04:5-07:00",
									"2006-01-2T15:4:05-07:00",
									"2006-01-2T15:4:5-07:00",
									"2006-1-2T15:04:05-07:00",
									"2006-1-2T15:04:5-07:00",
									"2006-1-2T15:4:05-07:00",
									"2006-1-2T15:4:5-07:00",
								} {
									if t, err := parse(layout, datestr, loc); err == nil {
										return t, nil
									}
								}

					case dateDigitDashT: // starts digit then dash 02-  then T
						// 2006-01-02T15:04:05.999999
						// 2006-01-02T15:04:05.999999
						for _, layout := range []string{
							"2006-01-02T15:04:05",
							"2006-01-02T15:04:5",
							"2006-01-02T15:4:05",
							"2006-01-02T15:4:5",
							"2006-1-02T15:04:05",
							"2006-1-02T15:4:05",
							"2006-1-02T15:04:5",
							"2006-1-02T15:4:5",
							"2006-01-2T15:04:05",
							"2006-01-2T15:04:5",
							"2006-01-2T15:4:05",
							"2006-01-2T15:4:5",
							"2006-1-2T15:04:05",
							"2006-1-2T15:04:5",
							"2006-1-2T15:4:05",
							"2006-1-2T15:4:5",
						} {
							if t, err := parse(layout, datestr, loc); err == nil {
								return t, nil
							}
						}

					case dateDigitDashTZDigit:
						// With a time-zone at end after Z
						// 2006-01-02T15:04:05.999999999Z07:00
						// 2006-01-02T15:04:05Z07:00
						// RFC3339     = "2006-01-02T15:04:05Z07:00"
						// RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
						return time.Time{}, fmt.Errorf("RFC339 Dates may not contain both Z & Offset for %q see https://github.com/golang/go/issues/5294", datestr)

					case dateDigitDashTZ: // starts digit then dash 02-  then T Then Z
						// 2006-01-02T15:04:05.999999999Z
						// 2006-01-02T15:04:05.99999999Z
						// 2006-01-02T15:04:05.9999999Z
						// 2006-01-02T15:04:05.999999Z
						// 2006-01-02T15:04:05.99999Z
						// 2006-01-02T15:04:05.9999Z
						// 2006-01-02T15:04:05.999Z
						// 2006-01-02T15:04:05.99Z
						// 2009-08-12T22:15Z  -- No seconds/milliseconds
						for _, layout := range []string{
							"2006-01-02T15:04:05Z",
							"2006-01-02T15:04:5Z",
							"2006-01-02T15:4:05Z",
							"2006-01-02T15:4:5Z",
							"2006-01-02T15:4Z",
							"2006-01-02T15:04Z",
							"2006-1-02T15:04:05Z",
							"2006-1-02T15:4:05Z",
							"2006-1-02T15:04:5Z",
							"2006-1-02T15:4:5Z",
							"2006-1-02T15:04Z",
							"2006-1-02T15:4Z",
							"2006-01-2T15:04:05Z",
							"2006-01-2T15:04:5Z",
							"2006-01-2T15:4:05Z",
							"2006-01-2T15:4:5Z",
							"2006-01-2T15:4Z",
							"2006-01-2T15:04Z",
							"2006-1-2T15:04:05Z",
							"2006-1-2T15:04:5Z",
							"2006-1-2T15:4:05Z",
							"2006-1-2T15:4:5Z",
							"2006-1-2T15:04Z",
							"2006-1-2T15:4Z",
						} {
							if t, err := parse(layout, datestr, loc); err == nil {
								return t, nil
							}
						}

					case dateDigitDashWs: // starts digit then dash 02-  then whitespace   1 << 2  << 5 + 3
						// 2013-04-01 22:43:22
						// 2013-04-01 22:43
						for _, layout := range []string{
							"2006-01-02 15:04:05",
							"2006-01-02 15:04:5",
							"2006-01-02 15:4:05",
							"2006-01-02 15:4:5",
							"2006-01-02 15:4",
							"2006-01-02 15:04",
							"2006-1-02 15:04:05",
							"2006-1-02 15:4:05",
							"2006-1-02 15:04:5",
							"2006-1-02 15:4:5",
							"2006-1-02 15:04",
							"2006-1-02 15:4",
							"2006-01-2 15:04:05",
							"2006-01-2 15:04:5",
							"2006-01-2 15:4:05",
							"2006-01-2 15:4:5",
							"2006-01-2 15:4",
							"2006-01-2 15:04",
							"2006-1-2 15:04:05",
							"2006-1-2 15:04:5",
							"2006-1-2 15:4:05",
							"2006-1-2 15:4:5",
							"2006-1-2 15:04",
							"2006-1-2 15:4",
						} {
							if t, err := parse(layout, datestr, loc); err == nil {
								return t, nil
							}
						}

					case dateDigitDashWsWsOffset:
						// 2006-01-02 15:04:05 -0700
						for _, layout := range []string{
							"2006-01-02 15:04:05 -0700",
							"2006-01-02 15:04:5 -0700",
							"2006-01-02 15:4:05 -0700",
							"2006-01-02 15:4:5 -0700",
							"2006-1-02 15:04:05 -0700",
							"2006-1-02 15:4:05 -0700",
							"2006-1-02 15:04:5 -0700",
							"2006-1-02 15:4:5 -0700",
							"2006-01-2 15:04:05 -0700",
							"2006-01-2 15:04:5 -0700",
							"2006-01-2 15:4:05 -0700",
							"2006-01-2 15:4:5 -0700",
							"2006-1-2 15:04:05 -0700",
							"2006-1-2 15:04:5 -0700",
							"2006-1-2 15:4:05 -0700",
							"2006-1-2 15:4:5 -0700",
						} {
							if t, err := parse(layout, datestr, loc); err == nil {
								return t, nil
							}
						}

					case dateDigitDashWsWsOffsetColon:
						// 2006-01-02 15:04:05 -07:00
						switch {
						case part2Len == 2 && part3Len == 2:
							for _, layout := range []string{
								"2006-01-02 15:04:05 -07:00",
								"2006-01-02 15:04:5 -07:00",
								"2006-01-02 15:4:05 -07:00",
								"2006-01-02 15:4:5 -07:00",
							} {
								if t, err := parse(layout, datestr, loc); err == nil {
									return t, nil
								}
							}
						case part2Len == 2 && part3Len == 1:
							for _, layout := range []string{
								"2006-01-2 15:04:05 -07:00",
								"2006-01-2 15:04:5 -07:00",
								"2006-01-2 15:4:05 -07:00",
								"2006-01-2 15:4:5 -07:00",
							} {
								if t, err := parse(layout, datestr, loc); err == nil {
									return t, nil
								}
							}
						case part2Len == 1 && part3Len == 2:
							for _, layout := range []string{
								"2006-1-02 15:04:05 -07:00",
								"2006-1-02 15:4:05 -07:00",
								"2006-1-02 15:04:5 -07:00",
								"2006-1-02 15:4:5 -07:00",
							} {
								if t, err := parse(layout, datestr, loc); err == nil {
									return t, nil
								}
							}
						case part2Len == 1 && part3Len == 1:
							for _, layout := range []string{
								"2006-1-2 15:04:05 -07:00",
								"2006-1-2 15:04:5 -07:00",
								"2006-1-2 15:4:05 -07:00",
								"2006-1-2 15:4:5 -07:00",
							} {
								if t, err := parse(layout, datestr, loc); err == nil {
									return t, nil
								}
							}
						}

					case dateDigitDashWsWsOffsetAlpha:
						// 2015-02-18 00:12:00 +0000 UTC

						switch {
						case part2Len == 2 && part3Len == 2:
							for _, layout := range []string{
								"2006-01-02 15:04:05 -0700 MST",
								"2006-01-02 15:04:5 -0700 MST",
								"2006-01-02 15:4:05 -0700 MST",
								"2006-01-02 15:4:5 -0700 MST",
								"2006-01-02 15:04:05 +0000 GMT",
								"2006-01-02 15:04:5 +0000 GMT",
								"2006-01-02 15:4:05 +0000 GMT",
								"2006-01-02 15:4:5 +0000 GMT",
							} {
								if t, err := parse(layout, datestr, loc); err == nil {
									return t, nil
								}
							}
						case part2Len == 2 && part3Len == 1:
							for _, layout := range []string{
								"2006-01-2 15:04:05 -0700 MST",
								"2006-01-2 15:04:5 -0700 MST",
								"2006-01-2 15:4:05 -0700 MST",
								"2006-01-2 15:4:5 -0700 MST",
								"2006-01-2 15:04:05 +0000 GMT",
								"2006-01-2 15:04:5 +0000 GMT",
								"2006-01-2 15:4:05 +0000 GMT",
								"2006-01-2 15:4:5 +0000 GMT",
							} {
								if t, err := parse(layout, datestr, loc); err == nil {
									return t, nil
								}
							}
						case part2Len == 1 && part3Len == 2:
							for _, layout := range []string{
								"2006-1-02 15:04:05 -0700 MST",
								"2006-1-02 15:4:05 -0700 MST",
								"2006-1-02 15:04:5 -0700 MST",
								"2006-1-02 15:4:5 -0700 MST",
								"2006-1-02 15:04:05 +0000 GMT",
								"2006-1-02 15:4:05 +0000 GMT",
								"2006-1-02 15:04:5 +0000 GMT",
								"2006-1-02 15:4:5 +0000 GMT",
							} {
								if t, err := parse(layout, datestr, loc); err == nil {
									return t, nil
								}
							}
						case part2Len == 1 && part3Len == 1:
							for _, layout := range []string{
								"2006-1-2 15:04:05 -0700 MST",
								"2006-1-2 15:04:5 -0700 MST",
								"2006-1-2 15:4:05 -0700 MST",
								"2006-1-2 15:4:5 -0700 MST",
								"2006-1-2 15:04:05 +0000 GMT",
								"2006-1-2 15:04:5 +0000 GMT",
								"2006-1-2 15:4:05 +0000 GMT",
								"2006-1-2 15:4:5 +0000 GMT",
							} {
								if t, err := parse(layout, datestr, loc); err == nil {
									return t, nil
								}
							}
						}

					case dateDigitDashWsWsOffsetColonAlpha:
						// 2015-02-18 00:12:00 +00:00 UTC
						for _, layout := range []string{
							"2006-01-02 15:04:05 -07:00 UTC",
							"2006-01-02 15:04:5 -07:00 UTC",
							"2006-01-02 15:4:05 -07:00 UTC",
							"2006-01-02 15:4:5 -07:00 UTC",
							"2006-1-02 15:04:05 -07:00 UTC",
							"2006-1-02 15:4:05 -07:00 UTC",
							"2006-1-02 15:04:5 -07:00 UTC",
							"2006-1-02 15:4:5 -07:00 UTC",
							"2006-01-2 15:04:05 -07:00 UTC",
							"2006-01-2 15:04:5 -07:00 UTC",
							"2006-01-2 15:4:05 -07:00 UTC",
							"2006-01-2 15:4:5 -07:00 UTC",
							"2006-1-2 15:04:05 -07:00 UTC",
							"2006-1-2 15:04:5 -07:00 UTC",
							"2006-1-2 15:4:05 -07:00 UTC",
							"2006-1-2 15:4:5 -07:00 UTC",
						} {
							if t, err := parse(layout, datestr, loc); err == nil {
								return t, nil
							}
						}

					case dateDigitDashWsOffset:
						// 2017-07-19 03:21:51+00:00
						for _, layout := range []string{
							"2006-01-02 15:04:05-07:00",
							"2006-01-02 15:04:5-07:00",
							"2006-01-02 15:4:05-07:00",
							"2006-01-02 15:4:5-07:00",
							"2006-1-02 15:04:05-07:00",
							"2006-1-02 15:4:05-07:00",
							"2006-1-02 15:04:5-07:00",
							"2006-1-02 15:4:5-07:00",
							"2006-01-2 15:04:05-07:00",
							"2006-01-2 15:04:5-07:00",
							"2006-01-2 15:4:05-07:00",
							"2006-01-2 15:4:5-07:00",
							"2006-1-2 15:04:05-07:00",
							"2006-1-2 15:04:5-07:00",
							"2006-1-2 15:4:05-07:00",
							"2006-1-2 15:4:5-07:00",
						} {
							if t, err := parse(layout, datestr, loc); err == nil {
								return t, nil
							}
						}

					case dateDigitDashWsWsAlpha:
						// 2014-12-16 06:20:00 UTC
						for _, layout := range []string{
							"2006-01-02 15:04:05 UTC",
							"2006-01-02 15:04:5 UTC",
							"2006-01-02 15:4:05 UTC",
							"2006-01-02 15:4:5 UTC",
							"2006-1-02 15:04:05 UTC",
							"2006-1-02 15:4:05 UTC",
							"2006-1-02 15:04:5 UTC",
							"2006-1-02 15:4:5 UTC",
							"2006-01-2 15:04:05 UTC",
							"2006-01-2 15:04:5 UTC",
							"2006-01-2 15:4:05 UTC",
							"2006-01-2 15:4:5 UTC",
							"2006-1-2 15:04:05 UTC",
							"2006-1-2 15:04:5 UTC",
							"2006-1-2 15:4:05 UTC",
							"2006-1-2 15:4:5 UTC",
							"2006-01-02 15:04:05 GMT",
							"2006-01-02 15:04:5 GMT",
							"2006-01-02 15:4:05 GMT",
							"2006-01-02 15:4:5 GMT",
							"2006-1-02 15:04:05 GMT",
							"2006-1-02 15:4:05 GMT",
							"2006-1-02 15:04:5 GMT",
							"2006-1-02 15:4:5 GMT",
							"2006-01-2 15:04:05 GMT",
							"2006-01-2 15:04:5 GMT",
							"2006-01-2 15:4:05 GMT",
							"2006-01-2 15:4:5 GMT",
							"2006-1-2 15:04:05 GMT",
							"2006-1-2 15:04:5 GMT",
							"2006-1-2 15:4:05 GMT",
							"2006-1-2 15:4:5 GMT",
						} {
							if t, err := parse(layout, datestr, loc); err == nil {
								return t, nil
							}
						}

						if len(datestr) > len("2006-01-02 03:04:05") {
							t, err := parse("2006-01-02 03:04:05", datestr[:len("2006-01-02 03:04:05")], loc)
							if err == nil {
								return t, nil
							}
						}

					case dateDigitDashWsPeriod:
						// 2012-08-03 18:31:59.257000000
						// 2014-04-26 17:24:37.3186369
						// 2017-01-27 00:07:31.945167
						// 2016-03-14 00:00:00.000
						for _, layout := range []string{
							"2006-01-02 15:04:05",
							"2006-01-02 15:04:5",
							"2006-01-02 15:4:05",
							"2006-01-02 15:4:5",
							"2006-1-02 15:04:05",
							"2006-1-02 15:4:05",
							"2006-1-02 15:04:5",
							"2006-1-02 15:4:5",
							"2006-01-2 15:04:05",
							"2006-01-2 15:04:5",
							"2006-01-2 15:4:05",
							"2006-01-2 15:4:5",
							"2006-1-2 15:04:05",
							"2006-1-2 15:04:5",
							"2006-1-2 15:4:05",
							"2006-1-2 15:4:5",
						} {
							if t, err := parse(layout, datestr, loc); err == nil {
								return t, nil
							}
						}

					case dateDigitDashWsPeriodAlpha:
						// 2012-08-03 18:31:59.257000000 UTC
						// 2014-04-26 17:24:37.3186369 UTC
						// 2017-01-27 00:07:31.945167 UTC
						// 2016-03-14 00:00:00.000 UTC
						for _, layout := range []string{
							"2006-01-02 15:04:05 UTC",
							"2006-01-02 15:04:5 UTC",
							"2006-01-02 15:4:05 UTC",
							"2006-01-02 15:4:5 UTC",
							"2006-1-02 15:04:05 UTC",
							"2006-1-02 15:4:05 UTC",
							"2006-1-02 15:04:5 UTC",
							"2006-1-02 15:4:5 UTC",
							"2006-01-2 15:04:05 UTC",
							"2006-01-2 15:04:5 UTC",
							"2006-01-2 15:4:05 UTC",
							"2006-01-2 15:4:5 UTC",
							"2006-1-2 15:04:05 UTC",
							"2006-1-2 15:04:5 UTC",
							"2006-1-2 15:4:05 UTC",
							"2006-1-2 15:4:5 UTC",
						} {
							if t, err := parse(layout, datestr, loc); err == nil {
								return t, nil
							}
						}

					case dateDigitDashWsPeriodOffset:
						// 2012-08-03 18:31:59.257000000 +0000
						// 2014-04-26 17:24:37.3186369 +0000
						// 2017-01-27 00:07:31.945167 +0000
						// 2016-03-14 00:00:00.000 +0000
						for _, layout := range []string{
							"2006-01-02 15:04:05 -0700",
							"2006-01-02 15:04:5 -0700",
							"2006-01-02 15:4:05 -0700",
							"2006-01-02 15:4:5 -0700",
							"2006-1-02 15:04:05 -0700",
							"2006-1-02 15:4:05 -0700",
							"2006-1-02 15:04:5 -0700",
							"2006-1-02 15:4:5 -0700",
							"2006-01-2 15:04:05 -0700",
							"2006-01-2 15:04:5 -0700",
							"2006-01-2 15:4:05 -0700",
							"2006-01-2 15:4:5 -0700",
							"2006-1-2 15:04:05 -0700",
							"2006-1-2 15:04:5 -0700",
							"2006-1-2 15:4:05 -0700",
							"2006-1-2 15:4:5 -0700",
						} {
							if t, err := parse(layout, datestr, loc); err == nil {
								return t, nil
							}
						}
			case dateDigitDashWsPeriodOffsetAlpha:
				// 2012-08-03 18:31:59.257000000 +0000 UTC
				// 2014-04-26 17:24:37.3186369 +0000 UTC
				// 2017-01-27 00:07:31.945167 +0000 UTC
				// 2016-03-14 00:00:00.000 +0000 UTC
				for _, layout := range []string{
					"2006-01-02 15:04:05 -0700 UTC",
					"2006-01-02 15:04:5 -0700 UTC",
					"2006-01-02 15:4:05 -0700 UTC",
					"2006-01-02 15:4:5 -0700 UTC",
					"2006-1-02 15:04:05 -0700 UTC",
					"2006-1-02 15:4:05 -0700 UTC",
					"2006-1-02 15:04:5 -0700 UTC",
					"2006-1-02 15:4:5 -0700 UTC",
					"2006-01-2 15:04:05 -0700 UTC",
					"2006-01-2 15:04:5 -0700 UTC",
					"2006-01-2 15:4:05 -0700 UTC",
					"2006-01-2 15:4:5 -0700 UTC",
					"2006-1-2 15:04:05 -0700 UTC",
					"2006-1-2 15:04:5 -0700 UTC",
					"2006-1-2 15:4:05 -0700 UTC",
					"2006-1-2 15:4:5 -0700 UTC",
				} {
					if t, err := parse(layout, datestr, loc); err == nil {
						return t, nil
					}
				}
		*/
	case dateDigitDotDot:
		switch {
		case len(datestr) == len("01.02.2006"):
			return parse("01.02.2006", datestr, loc)
		case len(datestr)-part2Len == 3:
			for _, layout := range []string{"01.02.06", "1.02.06", "01.2.06", "1.2.06"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		default:
			for _, layout := range []string{"1.02.2006", "01.2.2006", "1.2.2006"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		}

	case dateDigitWs:
		// 18 January 2018
		// 8 January 2018
		if part1Len == 1 {
			return parse("2 January 2006", datestr, loc)
		}
		return parse("02 January 2006", datestr, loc)
		// 02 Jan 2018 23:59
	case dateDigitWsMoShortColon:
		// 2 Jan 2018 23:59
		// 02 Jan 2018 23:59
		if part1Len == 1 {
			for _, layout := range []string{
				"2 Jan 2006 15:04",
				"2 Jan 2006 15:4",
			} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		}

		for _, layout := range []string{
			"02 Jan 2006 15:04",
			"02 Jan 2006 15:4",
		} {
			if t, err := parse(layout, datestr, loc); err == nil {
				return t, nil
			}
		}
	case dateDigitWsMoShortColonColon:
		// 02 Jan 2018 23:59:45
		if part1Len == 1 {
			for _, layout := range []string{
				"2 Jan 2006 15:04:05",
				"2 Jan 2006 15:04:5",
				"2 Jan 2006 15:4:5",
				"2 Jan 2006 15:4:05",
			} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		}
		for _, layout := range []string{
			"2 Jan 2006 15:04:05",
			"2 Jan 2006 15:04:5",
			"2 Jan 2006 15:4:5",
			"2 Jan 2006 15:4:05",
		} {
			if t, err := parse(layout, datestr, loc); err == nil {
				return t, nil
			}
		}

	case dateDigitWsMoShortComma:
		// 12 Feb 2006, 19:17
		// 12 Feb 2006, 19:17:22
		for _, layout := range []string{
			"02 Jan 2006, 15:04",
			"02 Jan 2006, 15:4",
			"2 Jan 2006, 15:04",
			"2 Jan 2006, 15:4",
			"02 Jan 2006, 15:04:05",
			"02 Jan 2006, 15:4:05",
			"02 Jan 2006, 15:4:5",
			"02 Jan 2006, 15:04:5",
			"2 Jan 2006, 15:04:05",
			"2 Jan 2006, 15:04:5",
			"2 Jan 2006, 15:4:5",
			"2 Jan 2006, 15:4:05",
		} {
			if t, err := parse(layout, datestr, loc); err == nil {
				return t, nil
			}
		}

	case dateAlphaWSDigitCommaWsYear:
		// May 8, 2009 5:57:51 PM
		for _, layout := range []string{
			"Jan 2, 2006 3:04:05 PM",
			"Jan 2, 2006 3:4:05 PM",
			"Jan 2, 2006 3:4:5 PM",
			"Jan 2, 2006 3:04:5 PM",
			"Jan 02, 2006 3:04:05 PM",
			"Jan 02, 2006 3:4:05 PM",
			"Jan 02, 2006 3:4:5 PM",
			"Jan 02, 2006 3:04:5 PM",
		} {
			if t, err := parse(layout, datestr, loc); err == nil {
				return t, nil
			}
		}
	case dateAlphaWSAlphaColon:
		// Mon Jan _2 15:04:05 2006
		return parse(time.ANSIC, datestr, loc)

	case dateAlphaWSAlphaColonOffset:
		// Mon Jan 02 15:04:05 -0700 2006
		return parse(time.RubyDate, datestr, loc)

	case dateAlphaWSAlphaColonAlpha:
		// Mon Jan _2 15:04:05 MST 2006
		return parse(time.UnixDate, datestr, loc)

	case dateAlphaWSAlphaColonAlphaOffset:
		// Mon Aug 10 15:44:11 UTC+0100 2015
		return parse("Mon Jan 02 15:04:05 MST-0700 2006", datestr, loc)

	case dateAlphaWSAlphaColonAlphaOffsetAlpha:
		// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
		if len(datestr) > len("Mon Jan 02 2006 15:04:05 MST-0700") {
			// What effing time stamp is this?
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			dateTmp := datestr[:33]
			return parse("Mon Jan 02 2006 15:04:05 MST-0700", dateTmp, loc)
		}
	case dateDigitSlash: // starts digit then slash 02/ (but nothing else)
		// 3/1/2014
		// 10/13/2014
		// 01/02/2006
		// 2014/10/13
		if part1Len == 4 {
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

	case dateDigitSlashWSColon: // starts digit then slash 02/ more digits/slashes then whitespace
		// 4/8/2014 22:05
		// 04/08/2014 22:05
		// 2014/4/8 22:05
		// 2014/04/08 22:05

		if part1Len == 4 {
			for _, layout := range []string{"2006/01/02 15:04", "2006/1/2 15:04", "2006/01/2 15:04", "2006/1/02 15:04", "2006/01/02 15:4", "2006/1/2 15:4", "2006/01/2 15:4", "2006/1/02 15:4"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 15:4", "01/2/2006 15:4", "1/02/2006 15:4", "1/2/2006 15:4", "1/2/06 15:4", "01/02/06 15:4", "01/02/2006 15:04", "01/2/2006 15:04", "1/02/2006 15:04", "1/2/2006 15:04", "1/2/06 15:04", "01/02/06 15:04"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		}

	case dateDigitSlashWSColonAMPM: // starts digit then slash 02/ more digits/slashes then whitespace
		// 4/8/2014 22:05 PM
		// 04/08/2014 22:05 PM
		// 04/08/2014 1:05 PM
		// 2014/4/8 22:05 PM
		// 2014/04/08 22:05 PM

		if part1Len == 4 {
			for _, layout := range []string{"2006/01/02 03:04 PM", "2006/01/2 03:04 PM", "2006/1/02 03:04 PM", "2006/1/2 03:04 PM",
				"2006/01/02 3:04 PM", "2006/01/2 3:04 PM", "2006/1/02 3:04 PM", "2006/1/2 3:04 PM", "2006/01/02 3:4 PM", "2006/01/2 3:4 PM", "2006/1/02 3:4 PM", "2006/1/2 3:4 PM",
				"2006/01/02 3:4 PM", "2006/01/2 3:4 PM", "2006/1/02 3:4 PM", "2006/1/2 3:4 PM"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 03:04 PM", "01/2/2006 03:04 PM", "1/02/2006 03:04 PM", "1/2/2006 03:04 PM",
				"01/02/2006 03:4 PM", "01/2/2006 03:4 PM", "1/02/2006 03:4 PM", "1/2/2006 03:4 PM",
				"01/02/2006 3:04 PM", "01/2/2006 3:04 PM", "1/02/2006 3:04 PM", "1/2/2006 3:04 PM",
				"01/02/2006 3:04 PM", "01/2/2006 3:04 PM", "1/02/2006 3:04 PM", "1/2/2006 3:04 PM",
				"01/02/2006 3:4 PM", "01/2/2006 3:4 PM", "1/02/2006 3:4 PM", "1/2/2006 3:4 PM",
				"01/02/2006 3:4 PM", "01/2/2006 3:4 PM", "1/02/2006 3:4 PM", "1/2/2006 3:4 PM"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}

			}
		}

	case dateDigitSlashWSColonColon: // starts digit then slash 02/ more digits/slashes then whitespace double colons
		// 2014/07/10 06:55:38.156283
		// 03/19/2012 10:11:59
		// 3/1/2012 10:11:59
		// 03/1/2012 10:11:59
		// 3/01/2012 10:11:59
		// 4/8/14 22:05
		if part1Len == 4 {
			for _, layout := range []string{"2006/01/02 15:04:05", "2006/1/02 15:04:05", "2006/01/2 15:04:05", "2006/1/2 15:04:05",
				"2006/01/02 15:04:5", "2006/1/02 15:04:5", "2006/01/2 15:04:5", "2006/1/2 15:04:5",
				"2006/01/02 15:4:05", "2006/1/02 15:4:05", "2006/01/2 15:4:05", "2006/1/2 15:4:05",
				"2006/01/02 15:4:5", "2006/1/02 15:4:5", "2006/01/2 15:4:5", "2006/1/2 15:4:5"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 15:04:05", "1/02/2006 15:04:05", "01/2/2006 15:04:05", "1/2/2006 15:04:05",
				"01/02/2006 15:4:5", "1/02/2006 15:4:5", "01/2/2006 15:4:5", "1/2/2006 15:4:5",
				"01/02/2006 15:4:05", "1/02/2006 15:4:05", "01/2/2006 15:4:05", "1/2/2006 15:4:05",
				"01/02/2006 15:04:5", "1/02/2006 15:04:5", "01/2/2006 15:04:5", "1/2/2006 15:04:5"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		}

	case dateDigitSlashWSColonColonAMPM: // starts digit then slash 02/ more digits/slashes then whitespace double colons
		// 2014/07/10 06:55:38.156283 PM
		// 03/19/2012 10:11:59 PM
		// 3/1/2012 10:11:59 PM
		// 03/1/2012 10:11:59 PM
		// 3/01/2012 10:11:59 PM

		if part1Len == 4 {
			for _, layout := range []string{"2006/01/02 03:04:05 PM", "2006/1/02 03:04:05 PM", "2006/01/2 03:04:05 PM", "2006/1/2 03:04:05 PM",
				"2006/01/02 03:4:5 PM", "2006/1/02 03:4:5 PM", "2006/01/2 03:4:5 PM", "2006/1/2 03:4:5 PM",
				"2006/01/02 03:4:05 PM", "2006/1/02 03:4:05 PM", "2006/01/2 03:4:05 PM", "2006/1/2 03:4:05 PM",
				"2006/01/02 03:04:5 PM", "2006/1/02 03:04:5 PM", "2006/01/2 03:04:5 PM", "2006/1/2 03:04:5 PM",

				"2006/01/02 3:4:5 PM", "2006/1/02 3:4:5 PM", "2006/01/2 3:4:5 PM", "2006/1/2 3:4:5 PM",
				"2006/01/02 3:4:05 PM", "2006/1/02 3:4:05 PM", "2006/01/2 3:4:05 PM", "2006/1/2 3:4:05 PM",
				"2006/01/02 3:04:5 PM", "2006/1/02 3:04:5 PM", "2006/01/2 3:04:5 PM", "2006/1/2 3:04:5 PM"} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 03:04:05 PM", "1/02/2006 03:04:05 PM", "01/2/2006 03:04:05 PM", "1/2/2006 03:04:05 PM",
				"01/02/2006 03:4:05 PM", "1/02/2006 03:4:05 PM", "01/2/2006 03:4:05 PM", "1/2/2006 03:4:05 PM",
				"01/02/2006 03:04:5 PM", "1/02/2006 03:04:5 PM", "01/2/2006 03:04:5 PM", "1/2/2006 03:04:5 PM",
				"01/02/2006 03:4:5 PM", "1/02/2006 03:4:5 PM", "01/2/2006 03:4:5 PM", "1/2/2006 03:4:5 PM",
			} {
				if t, err := parse(layout, datestr, loc); err == nil {
					return t, nil
				}
			}
		}
	case dateDigitChineseYear:
		// dateDigitChineseYear
		//   2014年04月08日
		return parse("2006年01月02日", datestr, loc)
	case dateDigitChineseYearWs:
		return parse("2006年01月02日 15:04:05", datestr, loc)
	case dateWeekdayCommaOffset:
		// Monday, 02 Jan 2006 15:04:05 -0700
		// Monday, 02 Jan 2006 15:04:05 +0100
		for _, layout := range []string{
			"Monday, _2 Jan 2006 15:04:05 -0700",
			"Monday, _2 Jan 2006 15:04:5 -0700",
			"Monday, _2 Jan 2006 15:4:05 -0700",
			"Monday, _2 Jan 2006 15:4:5 -0700",
		} {
			if t, err := parse(layout, datestr, loc); err == nil {
				return t, nil
			}
		}

	case dateWeekdayCommaDash:
		// Monday, 02-Jan-06 15:04:05 MST
		for _, layout := range []string{
			"Monday, 02-Jan-06 15:04:05 MST",
			"Monday, 02-Jan-06 15:4:05 MST",
			"Monday, 02-Jan-06 15:04:5 MST",
			"Monday, 02-Jan-06 15:4:5 MST",
			"Monday, 2-Jan-06 15:04:05 MST",
			"Monday, 2-Jan-06 15:4:05 MST",
			"Monday, 2-Jan-06 15:4:5 MST",
			"Monday, 2-Jan-06 15:04:5 MST",
			"Monday, 2-Jan-6 15:04:05 MST",
			"Monday, 2-Jan-6 15:4:05 MST",
			"Monday, 2-Jan-6 15:4:5 MST",
			"Monday, 2-Jan-6 15:04:5 MST",
		} {
			if t, err := parse(layout, datestr, loc); err == nil {
				return t, nil
			}
		}

	case dateWeekdayAbbrevComma: // Starts alpha then comma
		// Mon, 02-Jan-06 15:04:05 MST
		// Mon, 02 Jan 2006 15:04:05 MST
		for _, layout := range []string{
			"Mon, _2 Jan 2006 15:04:05 MST",
			"Mon, _2 Jan 2006 15:04:5 MST",
			"Mon, _2 Jan 2006 15:4:5 MST",
			"Mon, _2 Jan 2006 15:4:05 MST",
		} {
			if t, err := parse(layout, datestr, loc); err == nil {
				return t, nil
			}
		}

	case dateWeekdayAbbrevCommaDash:
		// Mon, 02-Jan-06 15:04:05 MST
		// Mon, 2-Jan-06 15:04:05 MST
		for _, layout := range []string{
			"Mon, 02-Jan-06 15:04:05 MST",
			"Mon, 02-Jan-06 15:4:05 MST",
			"Mon, 02-Jan-06 15:04:5 MST",
			"Mon, 02-Jan-06 15:4:5 MST",
			"Mon, 2-Jan-06 15:04:05 MST",
			"Mon, 2-Jan-06 15:4:05 MST",
			"Mon, 2-Jan-06 15:4:5 MST",
			"Mon, 2-Jan-06 15:04:5 MST",
			"Mon, 2-Jan-6 15:04:05 MST",
			"Mon, 2-Jan-6 15:4:05 MST",
			"Mon, 2-Jan-6 15:4:5 MST",
			"Mon, 2-Jan-6 15:04:5 MST",
		} {
			if t, err := parse(layout, datestr, loc); err == nil {
				return t, nil
			}
		}

	case dateWeekdayAbbrevCommaOffset:
		// Mon, 02 Jan 2006 15:04:05 -0700
		// Thu, 13 Jul 2017 08:58:40 +0100
		// RFC1123Z    = "Mon, 02 Jan 2006 15:04:05 -0700" // RFC1123 with numeric zone
		//
		// Thu, 4 Jan 2018 17:53:36 +0000
		for _, layout := range []string{
			"Mon, _2 Jan 2006 15:04:05 -0700",
			"Mon, _2 Jan 2006 15:4:05 -0700",
			"Mon, _2 Jan 2006 15:4:5 -0700",
			"Mon, _2 Jan 2006 15:04:5 -0700",
		} {
			if t, err := parse(layout, datestr, loc); err == nil {
				return t, nil
			}
		}

	case dateWeekdayAbbrevCommaOffsetZone:
		// Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
		if part3Len == 1 {
			return parse("Mon, 2 Jan 2006 15:04:05 -0700 (MST)", datestr, loc)
		}
		return parse("Mon, _2 Jan 2006 15:04:05 -0700 (MST)", datestr, loc)
	}

	return time.Time{}, fmt.Errorf("Could not find date format for %s", datestr)
}

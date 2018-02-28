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
	dateDigitDashDash
	dateDigitDashDashWs
	dateDigitDashDashT
	dateDigitDashDashAlpha
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
	dateDigitWsMoYear
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
)
const (
	// Time state
	timeIgnore timeState = iota
	timeStart
	timeWs
	timeWsAMPMMaybe
	timeWsAMPM
	timeWsOffset // 5
	timeWsAlpha
	timeWsOffsetAlpha
	timeWsOffsetColonAlpha
	timeWsOffsetColon
	timeOffset // 10
	timeOffsetColon
	timeAlpha
	timePeriod
	timePeriodOffset
	timePeriodOffsetColon // 15
	timePeriodWs
	timePeriodWsAlpha
	timePeriodWsOffset
	timePeriodWsOffsetWs
	timePeriodWsOffsetWsAlpha // 20
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

type parser struct {
	loc              *time.Location
	preferMonthFirst bool
	stateDate        dateState
	stateTime        timeState
	format           []byte
	datestr          string
	part1Len         int
	part2Len         int
	part3Len         int
	yeari            int
	yearlen          int
	moi              int
	molen            int
	dayi             int
	daylen           int
	houri            int
	hourlen          int
	mini             int
	minlen           int
	seci             int
	seclen           int
	msi              int
	mslen            int
	offseti          int
}

func newParser(dateStr string, loc *time.Location) parser {
	p := parser{
		stateDate: dateStart,
		stateTime: timeIgnore,
		datestr:   dateStr,
		loc:       loc,
	}
	p.format = []byte(dateStr)
	return p
}
func (p parser) set(start int, val string) {
	if start < 0 {
		return
	}
	if len(p.format) < start+len(val) {
		return
	}
	for i, r := range val {
		p.format[start+i] = byte(r)
	}
}
func (p parser) setMonth() {
	if p.moi <= 0 {
		return
	}
	if p.molen == 2 {
		p.set(p.moi, "01")
	} else {
		p.set(p.moi, "1")
	}
}
func (p parser) monthConvert(start, end int, mo string) {
	if len(p.format) <= end {
		return
	}
	for i := start; i < end; i++ {
		p.format[i] = ' '
	}
	p.set(start, mo)
}

func (p parser) setDay() {
	if p.dayi < 0 {
		return
	}
	if p.daylen == 2 {
		p.set(p.dayi, "02")
	} else {
		p.set(p.dayi, "2")
	}
}

func (p parser) coalesceDate(end int) {
	if p.yeari > 0 {
		if p.yearlen == 0 {
			p.yearlen = end - p.yeari
		}
		if p.yearlen == 2 {
			p.set(p.yeari, "06")
		} else if p.yearlen == 4 {
			p.set(p.yeari, "2006")
		}
	}
	if p.moi > 0 && p.molen == 0 {
		p.molen = end - p.moi
		p.setMonth()
	}
	if p.dayi > 0 && p.daylen == 0 {
		p.daylen = end - p.dayi
		p.setDay()
	}
}

func (p parser) coalesceTime(end int) {
	// 03:04:05
	// 15:04:05
	// 3:04:05
	// 3:4:5
	// 15:04:05.00
	if p.houri > 0 {
		if p.hourlen == 2 {
			p.set(p.houri, "15")
		} else {
			p.set(p.houri, "3")
		}
	}
	if p.mini > 0 {
		if p.minlen == 0 {
			p.minlen = end - p.mini
		}
		if p.minlen == 2 {
			p.set(p.mini, "04")
		} else {
			p.set(p.mini, "4")
		}
	}
	if p.seci > 0 {
		if p.seclen == 0 {
			p.seclen = end - p.seci
			//u.Infof("fixing seconds  p.seci=%d seclen=%d  end=%d", p.seci, p.seclen, end)
		}
		if p.seclen == 2 {
			p.set(p.seci, "05")
		} else {
			p.set(p.seci, "5")
		}
	}

	if p.msi > 0 {
		if p.mslen == 0 {
			p.mslen = end - p.msi
			//u.Warnf("set mslen??? %v", p.datestr)
		}
		for i := 0; i < p.mslen; i++ {
			p.format[p.msi+i] = '0'
		}
	}
	//u.Debugf("coalesce %+v", p)
}

func (p parser) parse() (time.Time, error) {
	u.Debugf("parse() loc=%v %50s AS %50s", p.loc.String(), p.datestr, p.format)
	if p.loc == nil {
		return time.Parse(string(p.format), p.datestr)
	}
	return time.ParseInLocation(string(p.format), p.datestr, p.loc)
}
func parseTime(datestr string, loc *time.Location) (time.Time, error) {

	p := newParser(datestr, loc)
	i := 0

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

		switch p.stateDate {
		case dateStart:
			if unicode.IsDigit(r) {
				p.stateDate = dateDigit
			} else if unicode.IsLetter(r) {
				p.stateDate = dateAlpha
			}
		case dateDigit:

			switch r {
			case '-', '\u2212':
				// 2006-01-02
				// 2006-01-02T15:04:05Z07:00
				// 13-Feb-03
				// 2013-Feb-03
				p.stateDate = dateDigitDash
				p.yeari = 0
				p.yearlen = i
				p.moi = i + 1
				if i == 4 {
					p.set(0, "2006")
				}
			case '/':
				p.stateDate = dateDigitSlash
			case '.':
				// 3.31.2014
				p.moi = 0
				p.molen = i
				p.dayi = i + 1
				p.stateDate = dateDigitDot
			case ' ':
				// 18 January 2018
				// 8 January 2018
				// 02 Jan 2018 23:59
				// 02 Jan 2018 23:59:34
				// 12 Feb 2006, 19:17
				// 12 Feb 2006, 19:17:22
				p.stateDate = dateDigitWs
				p.dayi = 0
				p.daylen = i
			case '年':
				// Chinese Year
				p.stateDate = dateDigitChineseYear
			default:
				//if unicode.IsDigit(r) {
				continue
			}
			p.part1Len = i

		case dateDigitDash:
			// 2006-01
			// 2006-01-02
			// dateDigitDashDashT
			//  2006-01-02T15:04:05Z07:00
			//  2017-06-25T17:46:57.45706582-07:00
			//  2006-01-02T15:04:05.999999999Z07:00
			//  2006-01-02T15:04:05+0000
			// dateDigitDashDashWs
			//  2012-08-03 18:31:59.257000000
			//  2014-04-26 17:24:37.3186369
			//  2017-01-27 00:07:31.945167
			//  2016-03-14 00:00:00.000
			//  2014-05-11 08:20:13,787
			//  2017-07-19 03:21:51+00:00
			//  2013-04-01 22:43:22
			//  2014-04-26 05:24:37 PM
			// dateDigitDashDashAlpha
			//  2013-Feb-03
			//  13-Feb-03
			switch r {
			case '-':
				p.molen = i - p.moi
				p.dayi = i + 1
				p.stateDate = dateDigitDashDash
				p.setMonth()
			default:
				if unicode.IsDigit(r) {
					//continue
				} else if unicode.IsLetter(r) {
					p.stateDate = dateDigitDashDashAlpha
				}
			}
		case dateDigitDashDash:
			// 2006-01-02
			// dateDigitDashDashT
			//  2006-01-02T15:04:05Z07:00
			//  2017-06-25T17:46:57.45706582-07:00
			//  2006-01-02T15:04:05.999999999Z07:00
			//  2006-01-02T15:04:05+0000
			// dateDigitDashDashWs
			//  2012-08-03 18:31:59.257000000
			//  2014-04-26 17:24:37.3186369
			//  2017-01-27 00:07:31.945167
			//  2016-03-14 00:00:00.000
			//  2014-05-11 08:20:13,787
			//  2017-07-19 03:21:51+00:00
			//  2013-04-01 22:43:22
			//  2014-04-26 05:24:37 PM

			switch r {
			case ' ':
				p.daylen = i - p.dayi
				p.stateDate = dateDigitDashDashWs
				p.stateTime = timeStart
				p.setDay()
				break iterRunes
			case 'T':
				p.daylen = i - p.dayi
				p.stateDate = dateDigitDashDashT
				p.stateTime = timeStart
				p.setDay()
				break iterRunes
			}
		case dateDigitDashDashAlpha:
			// 2013-Feb-03
			// 13-Feb-03
			switch r {
			case '-':
				p.molen = i - p.moi
				p.set(p.moi, "Jan")
				p.dayi = i + 1
			}
		case dateDigitSlash:
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 3/1/2014
			// 10/13/2014
			// 01/02/2006
			// 1/2/06

			switch r {
			case ' ':
				p.stateDate = dateDigitSlashWS
			case '/':
				continue
			default:
				// if unicode.IsDigit(r) || r == '/' {
				// 	continue
				// }
			}
		case dateDigitSlashWS:
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 4/8/14 22:05
			switch r {
			case ':':
				p.stateDate = dateDigitSlashWSColon
			}
		case dateDigitSlashWSColon:
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 4/8/14 22:05
			// 3/1/2012 10:11:59 AM
			switch r {
			case ':':
				p.stateDate = dateDigitSlashWSColonColon
			case 'A', 'P':
				p.stateDate = dateDigitSlashWSColonAMPM
			}
		case dateDigitSlashWSColonColon:
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 4/8/14 22:05
			// 3/1/2012 10:11:59 AM
			switch r {
			case 'A', 'P':
				p.stateDate = dateDigitSlashWSColonColonAMPM
			}

		case dateDigitWs:
			// 18 January 2018
			// 8 January 2018
			// 02 Jan 2018 23:59
			// 02 Jan 2018 23:59:34
			// 12 Feb 2006, 19:17
			// 12 Feb 2006, 19:17:22
			switch r {
			case ' ':
				u.Infof("part1=%d  i=%d", p.part1Len, i)
				p.yeari = i + 1
				p.yearlen = 4
				p.dayi = 0
				p.daylen = p.part1Len
				p.setDay()
				p.stateTime = timeStart
				if i <= len("12 Feb") {

					p.moi = p.daylen + 1
					p.molen = 3
					p.set(p.moi, "Jan")
					u.Infof("set day dayi=%d len=%d", p.dayi, p.daylen)
				} else {
					u.Warnf("unhandled long month")
					p.monthConvert(p.daylen+1, i, "Jan")
				}
				p.stateDate = dateDigitWsMoYear
			}

		case dateDigitWsMoYear:
			u.Debugf("dateDigitWsMoYear ")
			// 02 Jan 2018 23:59
			// 02 Jan 2018 23:59:34
			// 12 Feb 2006, 19:17
			// 12 Feb 2006, 19:17:22
			switch r {
			case ',':
				i++
				break iterRunes
			case ' ':
				break iterRunes
			}

		case dateDigitChineseYear:
			// dateDigitChineseYear
			//   2014年04月08日
			//               weekday  %Y年%m月%e日 %A %I:%M %p
			// 2013年07月18日 星期四 10:27 上午
			if r == ' ' {
				p.stateDate = dateDigitChineseYearWs
				break
			}
		case dateDigitDot:
			// 3.31.2014
			if r == '.' {
				p.daylen = i - p.dayi
				p.yeari = i + 1
				p.stateDate = dateDigitDotDot
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
				p.stateDate = dateAlphaWS
				p.set(0, "Mon")
				p.part1Len = i
			case r == ',':
				p.part1Len = i
				if i == 3 {
					p.stateDate = dateWeekdayAbbrevComma
					p.set(0, "Mon")
				} else {
					p.stateDate = dateWeekdayComma
					p.set(0, "Monday")
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
					p.stateDate = dateWeekdayCommaDash
					break iterRunes
				}
				p.stateDate = dateWeekdayCommaOffset
			case r == '+':
				p.stateDate = dateWeekdayCommaOffset
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
			case r == ' ' && p.part3Len == 0:
				p.part3Len = i - p.part1Len - 2
			case r == '-':
				if i < 15 {
					p.stateDate = dateWeekdayAbbrevCommaDash
					break iterRunes
				}
				p.stateDate = dateWeekdayAbbrevCommaOffset
			case r == '+':
				p.stateDate = dateWeekdayAbbrevCommaOffset
			}

		case dateWeekdayAbbrevCommaOffset:
			// dateWeekdayAbbrevCommaOffset
			//   Mon, 02 Jan 2006 15:04:05 -0700
			//   Thu, 13 Jul 2017 08:58:40 +0100
			//   Thu, 4 Jan 2018 17:53:36 +0000
			//   dateWeekdayAbbrevCommaOffsetZone
			//     Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			if r == '(' {
				p.stateDate = dateWeekdayAbbrevCommaOffsetZone
			}

		case dateAlphaWS: // Starts alpha then whitespace
			// Mon Jan _2 15:04:05 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			// Mon Aug 10 15:44:11 UTC+0100 2015
			switch {
			case r == ' ':
				p.part2Len = i - p.part1Len
			case unicode.IsLetter(r):
				p.stateDate = dateAlphaWSAlpha
			case unicode.IsDigit(r):
				p.stateDate = dateAlphaWSDigit
			default:
				u.Warnf("can we drop isLetter? case r=%s", string(r))
			}

		case dateAlphaWSDigit: // Starts Alpha, whitespace, digit, comma
			//  dateAlphaWSDigit
			//    May 8, 2009 5:57:51 PM
			switch {
			case r == ',':
				p.stateDate = dateAlphaWSDigitComma
			case unicode.IsDigit(r):
				p.stateDate = dateAlphaWSDigit
			default:
				u.Warnf("hm, can we drop a case here? %v", string(r))
			}
		case dateAlphaWSDigitComma:
			//          x
			//    May 8, 2009 5:57:51 PM
			switch {
			case r == ' ':
				p.stateDate = dateAlphaWSDigitCommaWs
			default:
				u.Warnf("hm, can we drop a case here? %v", string(r))
				return time.Time{}, fmt.Errorf("could not find format for %v expected white-space after comma", datestr)
			}
		case dateAlphaWSDigitCommaWs:
			//               x
			//    May 8, 2009 5:57:51 PM
			if !unicode.IsDigit(r) {
				p.stateDate = dateAlphaWSDigitCommaWsYear
				break iterRunes
			}

		case dateAlphaWSAlpha: // Alpha, whitespace, alpha
			// Mon Jan _2 15:04:05 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == ':' {
				p.stateDate = dateAlphaWSAlphaColon
			}
		case dateAlphaWSAlphaColon:
			// Mon Jan _2 15:04:05 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if unicode.IsLetter(r) {
				p.stateDate = dateAlphaWSAlphaColonAlpha
			} else if r == '-' || r == '+' {
				p.stateDate = dateAlphaWSAlphaColonOffset
			}
		case dateAlphaWSAlphaColonAlpha:
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == '+' {
				p.stateDate = dateAlphaWSAlphaColonAlphaOffset
			}
		case dateAlphaWSAlphaColonAlphaOffset:
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if unicode.IsLetter(r) {
				p.stateDate = dateAlphaWSAlphaColonAlphaOffsetAlpha
			}
		default:
			break iterRunes
		}
	}

	p.coalesceDate(i)

	if p.stateTime == timeStart {
		// increment first one, since the i++ occurs at end of loop
		i++

	iterTimeRunes:
		for ; i < len(datestr); i++ {
			r := rune(datestr[i])

			u.Debugf("i=%d   r=%s timeState=%d", i, string(r), p.stateTime)
			switch p.stateTime {
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
				//   19:55:00+0100
				// timePeriod
				//   17:24:37.3186369
				//   00:07:31.945167
				//   18:31:59.257000000
				//   00:00:00.000
				//   timePeriodOffset
				//     19:55:00.799+0100
				//     timePeriodOffsetColon
				//       15:04:05.999-07:00
				//   timePeriodWs
				//     timePeriodWsOffset
				//       00:07:31.945167 +0000
				//       00:00:00.000 +0000
				//     timePeriodWsOffsetAlpha
				//       00:07:31.945167 +0000 UTC
				//       00:00:00.000 +0000 UTC
				//     timePeriodWsAlpha
				//       06:20:00.000 UTC
				if p.houri == 0 {
					p.houri = i
				}
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
					//   03:21:51+00:00
					p.stateTime = timeOffset
					p.seclen = i - p.seci
					p.offseti = i
				case '.':
					p.stateTime = timePeriod
					p.seclen = i - p.seci
					p.msi = i + 1
				case 'Z':
					p.stateTime = timeZ
					if p.seci == 0 {
						p.minlen = i - p.mini
					} else {
						p.seclen = i - p.seci
					}
				case ' ':
					p.stateTime = timeWs
					p.seclen = i - p.seci
				case ':':
					if p.mini == 0 {
						p.mini = i + 1
						p.hourlen = i - p.houri
					} else if p.seci == 0 {
						p.seci = i + 1
						p.minlen = i - p.mini
					}

				}
			case timeOffset:
				// 19:55:00+0100
				// timeOffsetColon
				//   15:04:05+07:00
				//   15:04:05-07:00
				if r == ':' {
					p.stateTime = timeOffsetColon
				}
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
					p.stateTime = timeWsAMPMMaybe
				case '+', '-':
					p.offseti = i
					p.stateTime = timeWsOffset
				default:
					if unicode.IsLetter(r) {
						// 06:20:00 UTC
						p.stateTime = timeWsAlpha
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
					p.stateTime = timeWsAMPM
					p.set(i-1, "PM")
				} else {
					p.stateTime = timeWsAlpha
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
				switch r {
				case ':':
					p.stateTime = timeWsOffsetColon
				case ' ':
					p.set(p.offseti, "-0700")
					p.stateTime = timeWsOffset
				default:
					if unicode.IsLetter(r) {
						// 00:12:00 +0000 UTC
						p.stateTime = timeWsOffsetAlpha
						break iterTimeRunes
					}
				}

			case timeWsOffsetColon:
				// timeWsOffsetColon
				//   15:04:05 -07:00
				//   timeWsOffsetColonAlpha
				//     2015-02-18 00:12:00 +00:00 UTC
				if unicode.IsLetter(r) {
					// 2015-02-18 00:12:00 +00:00 UTC
					p.stateTime = timeWsOffsetColonAlpha
					break iterTimeRunes
				}

			case timePeriod:
				// 15:04:05.999999999+07:00
				// 15:04:05.999999999-07:00
				// 15:04:05.999999+07:00
				// 15:04:05.999999-07:00
				// 15:04:05.999+07:00
				// 15:04:05.999-07:00
				// timePeriod
				//   17:24:37.3186369
				//   00:07:31.945167
				//   18:31:59.257000000
				//   00:00:00.000
				//   timePeriodOffset
				//     19:55:00.799+0100
				//     timePeriodOffsetColon
				//       15:04:05.999-07:00
				//   timePeriodWs
				//     timePeriodWsOffset
				//       00:07:31.945167 +0000
				//       00:00:00.000 +0000
				//     timePeriodWsOffsetAlpha
				//       00:07:31.945167 +0000 UTC
				//       00:00:00.000 +0000 UTC
				//     timePeriodWsAlpha
				//       06:20:00.000 UTC
				switch r {
				case ' ':
					p.mslen = i - p.msi
					p.stateTime = timePeriodWs
				case '+', '-':
					// This really shouldn't happen
					p.mslen = i - p.msi
					p.offseti = i
					p.stateTime = timePeriodOffset
				default:
					if unicode.IsLetter(r) {
						// 06:20:00.000 UTC
						p.mslen = i - p.msi
						p.stateTime = timePeriodWsAlpha
					}
				}
			case timePeriodOffset:
				// timePeriodOffset
				//   19:55:00.799+0100
				//   timePeriodOffsetColon
				//     15:04:05.999-07:00
				switch r {
				case ':':
					p.stateTime = timePeriodOffsetColon
				default:
					if unicode.IsLetter(r) {
						//     00:07:31.945167 +0000 UTC
						//     00:00:00.000 +0000 UTC
						p.stateTime = timePeriodWsOffsetWsAlpha
						break iterTimeRunes
					}
				}
			case timePeriodOffsetColon:
				// timePeriodOffset
				//   timePeriodOffsetColon
				//     15:04:05.999-07:00

			case timePeriodWs:
				// timePeriodWs
				//   timePeriodWsOffset
				//     00:07:31.945167 +0000
				//     00:00:00.000 +0000
				//   timePeriodWsOffsetAlpha
				//     00:07:31.945167 +0000 UTC
				//     00:00:00.000 +0000 UTC
				//   timePeriodWsAlpha
				//     06:20:00.000 UTC
				if p.offseti == 0 {
					p.offseti = i
				}
				switch r {
				case '+', '-':
					p.mslen = i - p.msi - 1
					p.stateTime = timePeriodWsOffset
				default:
					if unicode.IsLetter(r) {
						//     00:07:31.945167 +0000 UTC
						//     00:00:00.000 +0000 UTC
						p.stateTime = timePeriodWsOffsetWsAlpha
						break iterTimeRunes
					}
				}

			case timePeriodWsOffset:

				// timePeriodWs
				//   timePeriodWsOffset
				//     00:07:31.945167 +0000
				//     00:00:00.000 +0000
				//   timePeriodWsOffsetAlpha
				//     00:07:31.945167 +0000 UTC
				//     00:00:00.000 +0000 UTC
				//   timePeriodWsAlpha
				//     06:20:00.000 UTC
				switch r {
				case ' ':
					p.set(p.offseti, "-0700")
				case ':':
					u.Errorf("timePeriodWsOffset UNHANDLED COLON")
				default:
					if unicode.IsLetter(r) {
						// 00:07:31.945167 +0000 UTC
						// 00:00:00.000 +0000 UTC
						p.stateTime = timePeriodWsOffsetWsAlpha
						break iterTimeRunes
					}
				}

			case timeZ:
				// timeZ
				//   15:04:05.99Z
				// With a time-zone at end after Z
				// 2006-01-02T15:04:05.999999999Z07:00
				// 2006-01-02T15:04:05Z07:00
				// RFC3339     = "2006-01-02T15:04:05Z07:00"
				// RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
				if unicode.IsDigit(r) {
					p.stateTime = timeZDigit
				}

			}
		}

		switch p.stateTime {
		case timePeriod:
			p.mslen = i - p.msi
		case timeOffset:
			// 19:55:00+0100
			p.set(p.offseti, "-0700")
		case timeOffsetColon:
			// 15:04:05+07:00
			p.set(p.offseti, "-07:00")
		// case timeZ:
		// 	u.Warnf("wtf? timeZ")
		// case timeZDigit:
		// 	u.Warnf("got timeZDigit Z00:00")
		case timePeriodOffset:
			// 19:55:00.799+0100
			p.set(p.offseti, "-0700")
		case timePeriodOffsetColon:
			p.set(p.offseti, "-07:00")
		case timePeriodWsOffset:
			p.set(p.offseti, "-0700")
		// case timePeriodWsOffsetWsAlpha:
		// 	u.Warnf("timePeriodWsOffsetAlpha")
		// case timeWsOffsetAlpha:
		// 	u.Warnf("timeWsOffsetAlpha   offseti=%d", p.offseti)
		default:
			//u.Warnf("un-handled statetime: %d for %v", p.stateTime, p.datestr)
		}

		p.coalesceTime(i)
	}

	//u.Infof("%60s %q\n\t%+v", datestr, string(p.format), p)

	switch p.stateDate {
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

	case dateDigitDash:
		// 2006-01
		return p.parse()

	case dateDigitDashDash:
		// 2006-01-02
		// 2006-1-02
		// 2006-1-2
		// 2006-01-2
		return p.parse()

	case dateDigitDashDashAlpha:
		// 2013-Feb-03
		// 2013-Feb-3
		p.daylen = i - p.dayi
		p.setDay()
		return p.parse()

	case dateDigitDashDashWs: // starts digit then dash 02-  then whitespace   1 << 2  << 5 + 3
		// 2013-04-01 22:43:22
		// 2013-04-01 22:43
		return p.parse()

	case dateDigitDashDashT:
		return p.parse()

	case dateDigitDotDot:
		// 03.31.1981
		// 3.2.1981
		// 3.2.81
		p.yearlen = i - p.yeari
		return p.parse()

	case dateDigitWs:
		// 18 January 2018
		// 8 January 2018
		return p.parse()

	case dateDigitWsMoYear:
		// 2 Jan 2018 23:59
		// 02 Jan 2018 23:59
		// 02 Jan 2018 23:59:45
		// 12 Feb 2006, 19:17
		// 12 Feb 2006, 19:17:22
		return p.parse()

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
		if p.part1Len == 4 {
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

		if p.part1Len == 4 {
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

		if p.part1Len == 4 {
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
		if p.part1Len == 4 {
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

		if p.part1Len == 4 {
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
		if p.part3Len == 1 {
			return parse("Mon, 2 Jan 2006 15:04:05 -0700 (MST)", datestr, loc)
		}
		return parse("Mon, _2 Jan 2006 15:04:05 -0700 (MST)", datestr, loc)
	}

	//u.Warnf("no format for %d  %d   %s", p.stateDate, p.stateTime, p.datestr)

	return time.Time{}, fmt.Errorf("Could not find date format for %s", datestr)
}

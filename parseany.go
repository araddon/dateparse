// Package dateparse parses date-strings without knowing the format
// in advance, using a fast lex based approach to eliminate shotgun
// attempts.  Validates comprehensively to avoid false positives.
//
// By default it leans towards US style dates when there is a
// conflict.  This can be adjusted using the `PreferMonthFirst`
// parser option. Some ambiguous formats can fail (e.g., trying to
// parse `31/03/2023“ as the default month-first format
// `MM/DD/YYYY`), but can be automatically retried with
// `RetryAmbiguousDateWithSwap`.
//
// Consider turning on the the `SimpleErrorMessages` option if you
// will be attempting to parse many strings that do not match any
// known format and you need to maximize performance.
//
// See README.md for key points on how timezone/location parsing
// works in go, as this can be counterintuitive initially.
package dateparse

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

// func init() {
// 	gou.SetupLogging("debug")
// 	gou.SetColorOutput()
// }

var knownDays = map[string]struct{}{
	"mon":       {},
	"tue":       {},
	"wed":       {},
	"thu":       {},
	"fri":       {},
	"sat":       {},
	"sun":       {},
	"monday":    {},
	"tuesday":   {},
	"wednesday": {},
	"thursday":  {},
	"friday":    {},
	"saturday":  {},
	"sunday":    {},
}

var knownMonths = map[string]struct{}{
	"january":   {},
	"february":  {},
	"march":     {},
	"april":     {},
	"may":       {},
	"june":      {},
	"july":      {},
	"august":    {},
	"september": {},
	"october":   {},
	"november":  {},
	"december":  {},
}

type dateState uint8
type timeState uint8

const (
	dateStart dateState = iota // 0
	dateDigit
	dateDigitSt
	dateYearDash
	dateYearDashAlpha
	dateYearDashDash
	dateYearDashDashWs // 6
	dateYearDashDashT
	dateYearDashDashOffset
	dateDigitDash
	dateDigitDashAlpha
	dateDigitDashAlphaDash // 11
	dateDigitDashDigit
	dateDigitDashDigitDash
	dateDigitDot
	dateDigitDotDot
	dateDigitDotDotWs
	dateDigitDotDotT
	dateDigitDotDotOffset
	dateDigitSlash
	dateDigitYearSlash
	dateDigitSlashAlpha // 21
	dateDigitSlashAlphaSlash
	dateDigitColon
	dateDigitChineseYear
	dateDigitChineseYearWs
	dateDigitWs
	dateDigitWsMoYear // 27
	dateAlpha
	dateAlphaWs
	dateAlphaWsDigit
	dateAlphaWsDigitMore // 31
	dateAlphaWsDigitMoreWs
	dateAlphaWsDigitMoreWsYear
	dateAlphaWsDigitYearMaybe
	dateVariousDaySuffix
	dateAlphaFullMonthWs
	dateAlphaFullMonthWsDayWs
	dateAlphaWsAlpha
	dateAlphaPeriodWsDigit
	dateAlphaSlash
	dateAlphaSlashDigit
	dateAlphaSlashDigitSlash
	dateYearWs
	dateYearWsMonthWs
)
const (
	// Time state
	timeIgnore timeState = iota // 0
	timeStart
	timeWs
	timeWsAlpha
	timeWsAlphaRParen
	timeWsAlphaWs
	timeWsAlphaWsYear
	timeWsAlphaZoneOffset // 7
	timeWsAlphaZoneOffsetWs
	timeWsAlphaZoneOffsetWsYear
	timeWsOffsetWsTZDescInParen // overloaded, can come from timeWsAlphaWs, timeWsAlphaZoneOffsetWs, timeWsOffsetWs, timeWsOffsetWsAlphaZoneWs
	timeWsAMPMMaybe
	timeWsAMPM         // 12
	timeWsOffset       // overloaded, can come from timeWs or timeWsYear
	timeWsOffsetWs     // 14
	timeWsOffsetWsYear // overloaded, can come from timeWsOffsetWs or timeWsOffsetWsAlphaZoneWs (ensures year is only set once)
	timeWsOffsetWsAlphaZone
	timeWsOffsetWsAlphaZoneWs
	timeWsYear
	timePeriod
	timePeriodAMPM
	timeZ
)

var (
	// ErrAmbiguousMMDD for date formats such as 04/02/2014 the mm/dd vs dd/mm are
	// ambiguous, so it is an error for strict parse rules.
	ErrAmbiguousMMDD     = fmt.Errorf("this date has ambiguous mm/dd vs dd/mm type format")
	ErrCouldntFindFormat = fmt.Errorf("could not find format for")
	ErrUnexpectedTail    = fmt.Errorf("unexpected content after date/time: ")
	ErrUnknownTZOffset   = fmt.Errorf("TZ offset not recognized")
	ErrUnknownTimeZone   = fmt.Errorf("timezone not recognized")
	ErrFracSecTooLong    = fmt.Errorf("fractional seconds too long")
)

func (p *parser) unknownErr(datestr string) error {
	if p == nil || !p.simpleErrorMessages {
		return fmt.Errorf("%w %q", ErrCouldntFindFormat, datestr)
	} else {
		return ErrCouldntFindFormat
	}
}

func (p *parser) unexpectedTail(tailStart int) error {
	if p != nil && !p.simpleErrorMessages {
		return fmt.Errorf("%w %q", ErrUnexpectedTail, p.datestr[tailStart:])
	} else {
		return ErrUnexpectedTail
	}
}

// go 1.20 allows us to convert a byte slice to a string without a memory allocation.
// See https://github.com/golang/go/issues/53003#issuecomment-1140276077.
func bytesToString(b []byte) string {
	if len(b) <= 0 {
		return ""
	} else {
		return unsafe.String(&b[0], len(b))
	}
}

// ParseAny parse an unknown date format, detect the layout.
// Normal parse.  Equivalent Timezone rules as time.Parse().
// NOTE:  please see readme on mmdd vs ddmm ambiguous dates.
func ParseAny(datestr string, opts ...ParserOption) (time.Time, error) {
	p, err := parseTime(datestr, nil, opts...)
	defer putBackParser(p)
	if err != nil {
		return time.Time{}, err
	}
	return p.parse(nil, opts...)
}

// ParseIn with Location, equivalent to time.ParseInLocation() timezone/offset
// rules.  Using location arg, if timezone/offset info exists in the
// datestring, it uses the given location rules for any zone interpretation.
// That is, MST means one thing when using America/Denver and something else
// in other locations. See README for a more detailed explanation.
func ParseIn(datestr string, loc *time.Location, opts ...ParserOption) (time.Time, error) {
	p, err := parseTime(datestr, loc, opts...)
	defer putBackParser(p)
	if err != nil {
		return time.Time{}, err
	}
	return p.parse(loc, opts...)
}

// ParseLocal Given an unknown date format, detect the layout,
// using time.Local, parse.
//
// Set Location to time.Local.  Same as ParseIn Location but lazily uses
// the global time.Local variable for Location argument.
//
//	denverLoc, _ := time.LoadLocation("America/Denver")
//	time.Local = denverLoc
//
//	t, err := dateparse.ParseLocal("3/1/2014")
//
// Equivalent to:
//
//	t, err := dateparse.ParseIn("3/1/2014", denverLoc)
func ParseLocal(datestr string, opts ...ParserOption) (time.Time, error) {
	p, err := parseTime(datestr, time.Local, opts...)
	defer putBackParser(p)
	if err != nil {
		return time.Time{}, err
	}
	return p.parse(time.Local, opts...)
}

// MustParse  parse a date, and panic if it can't be parsed.  Used for testing.
// Not recommended for most use-cases.
func MustParse(datestr string, opts ...ParserOption) time.Time {
	p, err := parseTime(datestr, nil, opts...)
	defer putBackParser(p)
	if err != nil {
		panic(err.Error())
	}
	t, err := p.parse(nil, opts...)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// ParseFormat parses an unknown date-time string and returns a layout
// string that can parse this (and exact same format) other date-time strings.
//
// In certain edge cases, this may produce a format string of a different
// length than the input string. If this happens, it's an edge case that
// requires individually parsing each time.
//
//	layout, err := dateparse.ParseFormat("2013-02-01 00:00:00")
//	// layout = "2006-01-02 15:04:05"
func ParseFormat(datestr string, opts ...ParserOption) (string, error) {
	p, err := parseTime(datestr, nil, opts...)
	defer putBackParser(p)
	if err != nil {
		return "", err
	}
	_, err = p.parse(nil, opts...)
	if err != nil {
		return "", err
	}
	return string(p.format), nil
}

// ParseStrict parse an unknown date format.  IF the date is ambigous
// mm/dd vs dd/mm then return an error. These return errors:   3.3.2014 , 8/8/71 etc
func ParseStrict(datestr string, opts ...ParserOption) (time.Time, error) {
	p, err := parseTime(datestr, nil, opts...)
	defer putBackParser(p)
	if err != nil {
		return time.Time{}, err
	}
	if p.ambiguousMD {
		return time.Time{}, ErrAmbiguousMMDD
	}
	return p.parse(nil, opts...)
}

// Creates a new parser and parses the given datestr in the given loc with the given options.
// The caller must call putBackParser on the returned parser when done with it.
func parseTime(datestr string, loc *time.Location, opts ...ParserOption) (p *parser, err error) {

	p, err = newParser(datestr, loc, opts...)
	if err != nil {
		return
	}

	// IMPORTANT: we may need to modify the datestr while we are parsing (e.g., to
	// remove pieces of the string that should be ignored during golang parsing).
	// We will iterate over the modified datestr, and whenever we update datestr,
	// we need to make sure that i is adjusted accordingly to resume parsing in
	// the correct place. In error messages though we'll use the original datestr.
	i := 0

	// General strategy is to read rune by rune through the date looking for
	// certain hints of what type of date we are dealing with.
	// Hopefully we only need to read about 5 or 6 bytes before
	// we figure it out and then attempt a parse
iterRunes:
	for ; i < len(p.datestr); i++ {
		r, bytesConsumed := utf8.DecodeRuneInString(p.datestr[i:])
		if bytesConsumed > 1 {
			i += bytesConsumed - 1
		}
		adjustedI := i - p.skip

		// gou.Debugf("i=%d r=%s state=%d   %s", i, string(r), p.stateDate, p.datestr)
		switch p.stateDate {
		case dateStart:
			// Note that we can reach this state either at the very start of the string,
			// or after skipping something (like a weekday, etc).
			// NOTE: don't use unicode.IsDigit and unicode.IsLetter here because
			// we don't expect non-ANSI chars to start a valid date/time format.
			// This will let us quickly reject strings that begin with any non-ANSI char.
			if '0' <= r && r <= '9' {
				p.stateDate = dateDigit
			} else if ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') {
				p.stateDate = dateAlpha
			} else if r == ' ' {
				// we can safely ignore whitespace at the start of strings (helps with
				// situations where we skipped a weekday and came back to this state)
				p.skip = i + 1
			} else {
				return p, p.unknownErr(datestr)
			}
		case dateDigit:

			switch r {
			case '-', '\u2212':
				// 2006-01-02
				// 2013-Feb-03
				// 13-Feb-03
				// 29-Jun-2016
				if adjustedI == 4 {
					p.stateDate = dateYearDash
					p.yeari = p.skip
					p.yearlen = i - p.skip
					p.moi = i + 1
					p.set(p.skip, "2006")
				} else {
					p.stateDate = dateDigitDash
				}
			case '/':
				// 08/May/2005
				// 03/31/2005
				// 2014/02/24
				p.stateDate = dateDigitSlash
				if adjustedI == 4 {
					// 2014/02/24  -  Year first /
					p.yeari = p.skip
					p.yearlen = i - p.skip
					p.moi = i + 1
					if !p.setYear() {
						return p, p.unknownErr(datestr)
					}
					p.stateDate = dateDigitYearSlash
				} else {
					// Either Ambiguous dd/mm vs mm/dd  OR dd/month/yy
					// 08/May/2005
					// 03/31/2005
					// 31/03/2005
					if i+2 < len(p.datestr) && unicode.IsLetter(rune(p.datestr[i+1])) {
						// 08/May/2005
						p.stateDate = dateDigitSlashAlpha
						p.moi = i + 1
						p.daylen = 2
						p.dayi = p.skip
						if !p.setDay() {
							return p, p.unknownErr(datestr)
						}
						continue
					}
					// Ambiguous dd/mm vs mm/dd the bane of date-parsing
					// 03/31/2005
					// 31/03/2005
					p.ambiguousMD = true
					p.ambiguousRetryable = true
					if p.preferMonthFirst {
						if p.molen == 0 {
							// 03/31/2005
							p.moi = p.skip
							p.molen = i - p.skip
							if !p.setMonth() {
								return p, p.unknownErr(datestr)
							}
							p.dayi = i + 1
						} else {
							return p, p.unknownErr(datestr)
						}
					} else {
						if p.daylen == 0 {
							p.dayi = p.skip
							p.daylen = i - p.skip
							if !p.setDay() {
								return p, p.unknownErr(datestr)
							}
							p.moi = i + 1
						} else {
							return p, p.unknownErr(datestr)
						}
					}
				}

			case ':':
				// 03:31:2005
				// 2014:02:24
				p.stateDate = dateDigitColon
				if adjustedI == 4 {
					p.yeari = p.skip
					p.yearlen = i - p.skip
					p.moi = i + 1
					if !p.setYear() {
						return p, p.unknownErr(datestr)
					}
				} else {
					p.ambiguousMD = true
					p.ambiguousRetryable = true
					if p.preferMonthFirst {
						if p.molen == 0 {
							p.moi = p.skip
							p.molen = i - p.skip
							if !p.setMonth() {
								return p, p.unknownErr(datestr)
							}
							p.dayi = i + 1
						} else {
							return p, p.unknownErr(datestr)
						}
					} else {
						if p.daylen == 0 {
							p.dayi = p.skip
							p.daylen = i - p.skip
							if !p.setDay() {
								return p, p.unknownErr(datestr)
							}
							p.moi = i + 1
						} else {
							return p, p.unknownErr(datestr)
						}
					}
				}

			case '.':
				// 3.31.2014
				// 08.21.71
				// 2014.05
				p.stateDate = dateDigitDot
				if adjustedI == 4 {
					p.yeari = p.skip
					p.yearlen = i - p.skip
					p.moi = i + 1
					if !p.setYear() {
						return p, p.unknownErr(datestr)
					}
				} else if adjustedI <= 2 {
					p.ambiguousMD = true
					p.ambiguousRetryable = true
					if p.preferMonthFirst {
						if p.molen == 0 {
							// 03.31.2005
							p.moi = p.skip
							p.molen = i - p.skip
							if !p.setMonth() {
								return p, p.unknownErr(datestr)
							}
							p.dayi = i + 1
						} else {
							return p, p.unknownErr(datestr)
						}
					} else {
						if p.daylen == 0 {
							p.dayi = p.skip
							p.daylen = i - p.skip
							if !p.setDay() {
								return p, p.unknownErr(datestr)
							}
							p.moi = i + 1
						} else {
							return p, p.unknownErr(datestr)
						}
					}
				}
				// else this might be a unixy combined datetime of the form:
				// yyyyMMddhhmmss.SSS

			case ' ':
				// 18 January 2018
				// 8 January 2018
				// 8 jan 2018
				// 02 Jan 2018 23:59
				// 02 Jan 2018 23:59:34
				// 12 Feb 2006, 19:17
				// 12 Feb 2006, 19:17:22
				// 2013 Jan 06 15:04:05
				switch adjustedI {
				case 4:
					p.yeari = p.skip
					p.yearlen = i - p.skip
					p.moi = i + 1
					if !p.setYear() {
						return p, p.unknownErr(datestr)
					}
					p.stateDate = dateYearWs
				case 6:
					p.stateDate = dateDigitSt
				default:
					p.stateDate = dateDigitWs
					p.dayi = p.skip
					p.daylen = i - p.skip
				}
			case '年':
				// Chinese Year
				p.stateDate = dateDigitChineseYear
				p.yeari = p.skip
				p.yearlen = i - 2 - p.skip
				p.moi = i + 1
				if !p.setYear() {
					return p, p.unknownErr(datestr)
				}
			case ',':
				return p, p.unknownErr(datestr)
			case 's', 'S', 'r', 'R', 't', 'T', 'n', 'N':
				// 1st January 2018
				// 2nd Jan 2018 23:59
				// st, rd, nd, th
				p.stateDate = dateVariousDaySuffix
				i--
			default:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
				continue
			}
			p.part1Len = i - p.skip

		case dateDigitSt:
			p.set(p.skip, "060102")
			i = i - 1
			p.stateTime = timeStart
			break iterRunes
		case dateYearDash:
			// dateYearDashDashT
			//  2006-01-02T15:04:05Z07:00
			//  2020-08-17T17:00:00:000+0100
			// dateYearDashDashWs
			//  2013-04-01 22:43:22
			// dateYearDashAlpha
			//     2013-Feb-03
			//     2013-February-03
			switch r {
			case '-', '\u2212':
				p.molen = i - p.moi
				p.dayi = i + 1
				p.stateDate = dateYearDashDash
				if !p.setMonth() {
					return p, p.unknownErr(datestr)
				}
			default:
				if unicode.IsLetter(r) {
					p.stateDate = dateYearDashAlpha
				} else if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateYearDashDash:
			// dateYearDashDashT
			//  2006-01-02T15:04:05Z07:00
			// dateYearDashDashWs
			//  2013-04-01 22:43:22
			// dateYearDashDashOffset
			//  2020-07-20+00:00
			// (these states are also reused after dateYearDashAlpha, like 2020-July-20...)
			switch r {
			case '+', '-':
				p.offseti = i
				p.daylen = i - p.dayi
				p.stateDate = dateYearDashDashOffset
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
			case ' ':
				p.daylen = i - p.dayi
				p.stateDate = dateYearDashDashWs
				p.stateTime = timeStart
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
				break iterRunes
			case 'T':
				p.daylen = i - p.dayi
				p.stateDate = dateYearDashDashT
				p.stateTime = timeStart
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
				break iterRunes
			default:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateYearDashDashT:
			// dateYearDashDashT
			//  2006-01-02T15:04:05Z07:00
			//  2020-08-17T17:00:00:000+0100
			// (this state should never be reached, we break out when in this state)
			return p, p.unknownErr(datestr)

		case dateYearDashDashOffset:
			//  2020-07-20+00:00
			if r != ':' && !unicode.IsDigit(r) {
				return p, p.unknownErr(datestr)
			}

		case dateYearDashAlpha:
			// dateYearDashAlpha
			//   2013-Feb-03
			//   2013-February-03
			switch r {
			case '-', '\u2212':
				p.molen = i - p.moi
				// Must be a valid short or long month
				if p.molen == 3 {
					p.set(p.moi, "Jan")
					p.dayi = i + 1
					p.stateDate = dateYearDashDash
				} else {
					possibleFullMonth := strings.ToLower(p.datestr[p.moi:(p.moi + p.molen)])
					if i > 3 && isMonthFull(possibleFullMonth) {
						p.fullMonth = possibleFullMonth
						p.dayi = i + 1
						p.stateDate = dateYearDashDash
					} else {
						return p, p.unknownErr(datestr)
					}
				}
			default:
				if !unicode.IsLetter(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateDigitDash:
			// 13-Feb-03
			// 29-Jun-2016
			if unicode.IsLetter(r) {
				p.stateDate = dateDigitDashAlpha
				p.moi = i
			} else if unicode.IsDigit(r) {
				p.stateDate = dateDigitDashDigit
				p.moi = i
			} else {
				return p, p.unknownErr(datestr)
			}
		case dateDigitDashAlpha:
			// 13-Feb-03
			// 28-Feb-03
			// 29-Jun-2016
			switch r {
			case '-', '\u2212':
				p.molen = i - p.moi

				// Must be a valid short or long month
				if p.molen == 3 {
					p.set(p.moi, "Jan")
					p.yeari = i + 1
					p.stateDate = dateDigitDashAlphaDash
				} else {
					possibleFullMonth := strings.ToLower(p.datestr[p.moi:(p.moi + p.molen)])
					if i > 3 && isMonthFull(possibleFullMonth) {
						p.fullMonth = possibleFullMonth
						p.yeari = i + 1
						p.stateDate = dateDigitDashAlphaDash
					} else {
						return p, p.unknownErr(datestr)
					}
				}
			default:
				if !unicode.IsLetter(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateDigitDashDigit:
			// 29-06-2026
			switch r {
			case '-', '\u2212':
				//      X
				// 29-06-2026
				p.molen = i - p.moi
				if p.molen == 2 {
					p.set(p.moi, "01")
					p.yeari = i + 1
					p.stateDate = dateDigitDashDigitDash
				} else {
					return p, p.unknownErr(datestr)
				}
			default:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateDigitDashAlphaDash, dateDigitDashDigitDash:
			// dateDigitDashAlphaDash:
			//   13-Feb-03   ambiguous
			//   28-Feb-03   ambiguous
			//   29-Jun-2016  dd-month(alpha)-yyyy
			//   8-Mar-2018::
			// dateDigitDashDigitDash:
			//   29-06-2026
			//   08-03-18:: ambiguous (dd-mm-yy or yy-mm-dd)
			switch r {
			case ' ', ':':
				doubleColonTimeConnector := false
				if r == ':' {
					p.link++
					if p.link == 2 {
						if i+1 < len(p.datestr) {
							// only legitimate content to follow "::" is the start of the time
							nextChar, _ := utf8.DecodeRuneInString(p.datestr[i+1:])
							if unicode.IsDigit(nextChar) {
								doubleColonTimeConnector = true
							}
						}
						if !doubleColonTimeConnector {
							return p, p.unknownErr(datestr)
						}
					}
				} else if p.link > 0 {
					return p, p.unknownErr(datestr)
				}
				if r == ' ' || doubleColonTimeConnector {
					// we need to find if this was 4 digits, aka year
					// or 2 digits which makes it ambiguous year/day
					var sepLen int
					if doubleColonTimeConnector {
						sepLen = 2
					} else {
						sepLen = 1
					}
					length := i - (p.moi + p.molen + sepLen)
					switch length {
					case 4:
						p.yearlen = 4
						p.set(p.yeari, "2006")
						// We now also know that part1 was the day
						p.dayi = p.skip
						p.daylen = p.part1Len
						if !p.setDay() {
							return p, p.unknownErr(datestr)
						}
					case 2:
						// We have no idea if this is
						// yy-mon-dd   OR  dd-mon-yy
						// (or for dateDigitDashDigitDash, yy-mm-dd  OR  dd-mm-yy)
						//
						// We are going to ASSUME (bad, bad) that it is dd-mon-yy (dd-mm-yy),
						// which is a horrible assumption, but seems to be the convention for
						// dates that are formatted in this way.
						p.ambiguousMD = true // not retryable
						p.yearlen = 2
						p.set(p.yeari, "06")
						// We now also know that part1 was the day
						p.dayi = p.skip
						p.daylen = p.part1Len
						if !p.setDay() {
							return p, p.unknownErr(datestr)
						}
					default:
						return p, p.unknownErr(datestr)
					}
					p.stateTime = timeStart
					break iterRunes
				}
			default:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateDigitYearSlash:
			// 2014/07/10 06:55:38.156283
			// I honestly don't know if this format ever shows up as yyyy/

			switch r {
			case ' ':
				fallthrough
			case ':':
				p.stateTime = timeStart
				if p.daylen == 0 {
					p.daylen = i - p.dayi
					if !p.setDay() {
						return p, p.unknownErr(datestr)
					}
				}
				break iterRunes
			case '/':
				if p.molen == 0 {
					p.molen = i - p.moi
					if !p.setMonth() {
						return p, p.unknownErr(datestr)
					}
					p.dayi = i + 1
				}
			default:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateDigitSlashAlpha:
			// 06/May/2008
			// 06/September/2008

			switch r {
			case '/':
				//       |
				// 06/May/2008
				if p.molen == 0 {
					p.molen = i - p.moi
					if p.molen == 3 {
						p.set(p.moi, "Jan")
						p.yeari = i + 1
						p.stateDate = dateDigitSlashAlphaSlash
					} else {
						possibleFullMonth := strings.ToLower(p.datestr[p.moi:(p.moi + p.molen)])
						if i > 3 && isMonthFull(possibleFullMonth) {
							p.fullMonth = possibleFullMonth
							p.yeari = i + 1
							p.stateDate = dateDigitSlashAlphaSlash
						} else {
							return p, p.unknownErr(datestr)
						}
					}
				} else {
					return p, p.unknownErr(datestr)
				}
			default:
				if !unicode.IsLetter(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateDigitSlashAlphaSlash:
			switch r {
			case ' ':
				fallthrough
			case ':':
				p.stateTime = timeStart
				if p.yearlen == 0 {
					p.yearlen = i - p.yeari
					if !p.setYear() {
						return p, p.unknownErr(datestr)
					}
				}
				break iterRunes
			default:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateDigitSlash:
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 04/2/2014, 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 3/1/2014
			// 10/13/2014
			// 01/02/2006
			// 1/2/06

			switch r {
			case '/':
				// This is the 2nd / so now we should know start pts of all of the dd, mm, yy
				if p.preferMonthFirst {
					if p.daylen == 0 {
						p.daylen = i - p.dayi
						if !p.setDay() {
							return p, p.unknownErr(datestr)
						}
						p.yeari = i + 1
					}
				} else {
					if p.molen == 0 {
						p.molen = i - p.moi
						if !p.setMonth() {
							return p, p.unknownErr(datestr)
						}
						p.yeari = i + 1
					}
				}
				// Note no break, we are going to pass by and re-enter this dateDigitSlash
				// and look for ending (space) or not (just date)
			case ' ', ',':
				p.stateTime = timeStart
				if p.yearlen == 0 {
					p.yearlen = i - p.yeari
					if r == ',' {
						// skip the comma
						i++
					}
					if !p.setYear() {
						return p, p.unknownErr(datestr)
					}
				}
				break iterRunes
			default:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateDigitColon:
			// 2014:07:10 06:55:38.156283
			// 03:19:2012 10:11:59
			// 04:2:2014 03:00:37
			// 3:1:2012 10:11:59
			// 4:8:2014 22:05
			// 3:1:2014
			// 10:13:2014
			// 01:02:2006
			// 1:2:06

			switch r {
			case ' ':
				p.stateTime = timeStart
				if p.yearlen == 0 {
					p.yearlen = i - p.yeari
					if !p.setYear() {
						return p, p.unknownErr(datestr)
					}
				} else if p.daylen == 0 {
					p.daylen = i - p.dayi
					if !p.setDay() {
						return p, p.unknownErr(datestr)
					}
				} else if p.molen == 0 {
					p.molen = i - p.moi
					if !p.setMonth() {
						return p, p.unknownErr(datestr)
					}
				}
				break iterRunes
			case ':':
				if p.yearlen > 0 {
					// 2014:07:10 06:55:38.156283
					if p.molen == 0 {
						p.molen = i - p.moi
						if !p.setMonth() {
							return p, p.unknownErr(datestr)
						}
						p.dayi = i + 1
					}
				} else if p.preferMonthFirst {
					if p.daylen == 0 {
						p.daylen = i - p.dayi
						if !p.setDay() {
							return p, p.unknownErr(datestr)
						}
						p.yeari = i + 1
					}
				} else {
					if p.molen == 0 {
						p.molen = i - p.moi
						if !p.setMonth() {
							return p, p.unknownErr(datestr)
						}
						p.yeari = i + 1
					}
				}
			default:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateDigitWs:
			// 18 January 2018
			// 8 January 2018
			// 8 jan 2018
			// 1 jan 18
			// 02 Jan 2018 23:59
			// 02 Jan 2018 23:59:34
			// 12 Feb 2006, 19:17
			// 12 Feb 2006, 19:17:22
			switch r {
			case ' ':
				p.yeari = i + 1
				//p.yearlen = 4
				p.dayi = p.skip
				p.daylen = p.part1Len
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
				p.stateTime = timeStart
				if adjustedI > p.daylen+len(" Sep") { //  November etc
					// If this is a legit full month, then change the string we're parsing
					// to compensate for the longest month, and do the same with the format string. We
					// must maintain a corresponding length/content and this is the easiest
					// way to do this.
					possibleFullMonth := strings.ToLower(p.datestr[(p.dayi + (p.daylen + 1)):i])
					if isMonthFull(possibleFullMonth) {
						p.moi = p.dayi + p.daylen + 1
						p.molen = i - p.moi
						p.fullMonth = possibleFullMonth
						p.stateDate = dateDigitWsMoYear
					} else {
						return p, p.unknownErr(datestr)
					}
				} else {
					// If len=3, the might be Feb or May?  Ie ambigous abbreviated but
					// we can parse may with either.  BUT, that means the
					// format may not be correct?
					// mo := strings.ToLower(p.datestr[p.daylen+1 : i])
					p.moi = p.dayi + p.daylen + 1
					p.molen = i - p.moi
					p.set(p.moi, "Jan")
					p.stateDate = dateDigitWsMoYear
				}
			default:
				if !unicode.IsLetter(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateDigitWsMoYear:
			// 8 jan 2018
			// 02 Jan 2018 23:59
			// 02 Jan 2018 23:59:34
			// 12 Feb 2006, 19:17
			// 12 Feb 2006, 19:17:22
			switch r {
			case ',':
				p.yearlen = i - p.yeari
				if !p.setYear() {
					return p, p.unknownErr(datestr)
				}
				i++
				break iterRunes
			case ' ':
				p.yearlen = i - p.yeari
				if !p.setYear() {
					return p, p.unknownErr(datestr)
				}
				break iterRunes
			default:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateYearWs:
			// 2013 Jan 06 15:04:05
			// 2013 January 06 15:04:05
			if r == ' ' {
				p.molen = i - p.moi
				// Must be a valid short or long month
				if p.molen == 3 {
					p.set(p.moi, "Jan")
					p.dayi = i + 1
					p.stateDate = dateYearWsMonthWs
				} else {
					possibleFullMonth := strings.ToLower(p.datestr[p.moi:(p.moi + p.molen)])
					if i > 3 && isMonthFull(possibleFullMonth) {
						p.fullMonth = possibleFullMonth
						p.dayi = i + 1
						p.stateDate = dateYearWsMonthWs
					} else {
						return p, p.unknownErr(datestr)
					}
				}
			} else if !unicode.IsLetter(r) {
				return p, p.unknownErr(datestr)
			}
		case dateYearWsMonthWs:
			// 2013 Jan 06 15:04:05
			// 2013 January 06 15:04:05
			switch r {
			case ',':
				p.daylen = i - p.dayi
				p.setDay()
				i++
				p.stateTime = timeStart
				break iterRunes
			case ' ':
				p.daylen = i - p.dayi
				p.setDay()
				p.stateTime = timeStart
				break iterRunes
			default:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateDigitChineseYear:
			// dateDigitChineseYear
			//   2014年04月08日
			//               weekday  %Y年%m月%e日 %A %I:%M %p
			// 2013年07月18日 星期四 10:27 上午
			switch r {
			case '月':
				// month
				p.molen = i - p.moi - 2
				p.dayi = i + 1
				if !p.setMonth() {
					return p, p.unknownErr(datestr)
				}
			case '日':
				// day
				p.daylen = i - p.dayi - 2
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
			case ' ':
				if p.daylen <= 0 {
					return p, p.unknownErr(datestr)
				}
				p.stateDate = dateDigitChineseYearWs
				p.stateTime = timeStart
				break iterRunes
			default:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}
		case dateDigitDot:
			// 3.31.2014
			// 08.21.71
			// 2014.05
			// 2018.09.30

			// This is the 2nd period
			if r == '.' {
				if p.moi == 0 {
					// 3.31.2014
					p.daylen = i - p.dayi
					p.yeari = i + 1
					if !p.setDay() {
						return p, p.unknownErr(datestr)
					}
					p.stateDate = dateDigitDotDot
				} else if p.dayi == 0 && p.yearlen == 0 {
					// 23.07.2002
					p.molen = i - p.moi
					p.yeari = i + 1
					if !p.setMonth() {
						return p, p.unknownErr(datestr)
					}
					p.stateDate = dateDigitDotDot
				} else {
					// 2018.09.30
					// p.molen = 2
					p.molen = i - p.moi
					p.dayi = i + 1
					if !p.setMonth() {
						return p, p.unknownErr(datestr)
					}
					p.stateDate = dateDigitDotDot
				}
			} else if !unicode.IsDigit(r) {
				return p, p.unknownErr(datestr)
			}

		case dateDigitDotDot:
			// dateDigitDotDotT
			//  2006.01.02T15:04:05Z07:00
			// dateDigitDotDotWs
			//  2013.04.01 22:43:22
			// dateDigitDotDotOffset
			//  2020.07.20+00:00
			switch r {
			case '+', '-':
				p.offseti = i
				p.daylen = i - p.dayi
				p.stateDate = dateDigitDotDotOffset
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
			case ' ':
				if p.daylen == 0 && p.molen > 0 && p.yearlen > 0 {
					p.daylen = i - p.dayi
					if !p.setDay() {
						return p, p.unknownErr(datestr)
					}
				} else if p.molen == 0 && p.daylen > 0 && p.yearlen > 0 {
					p.molen = i - p.moi
					if !p.setMonth() {
						return p, p.unknownErr(datestr)
					}
				} else if p.yearlen == 0 && p.daylen > 0 && p.molen > 0 {
					p.yearlen = i - p.yeari
					if !p.setYear() {
						return p, p.unknownErr(datestr)
					}
				} else {
					return p, p.unknownErr(datestr)
				}
				p.stateDate = dateDigitDotDotWs
				p.stateTime = timeStart
				break iterRunes
			case 'T':
				p.daylen = i - p.dayi
				p.stateDate = dateDigitDotDotT
				p.stateTime = timeStart
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
				break iterRunes
			default:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateDigitDotDotT:
			// dateDigitDotDotT
			//  2006-01-02T15:04:05Z07:00
			//  2020-08-17T17:00:00:000+0100
			// (should be unreachable, we break in this state)
			return p, p.unknownErr(datestr)

		case dateDigitDotDotOffset:
			//  2020-07-20+00:00
			if r != ':' && !unicode.IsDigit(r) {
				return p, p.unknownErr(datestr)
			}

		case dateAlpha:
			// dateAlphaWs
			//  Mon Jan _2 15:04:05 2006
			//  Mon Jan _2 15:04:05 MST 2006
			//  Mon Jan 02 15:04:05 -0700 2006
			//  Mon Jan 02 15:04:05 2006 -0700
			//  Mon Aug 10 15:44:11 UTC+0100 2015
			//  Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			//  Fri Jul 03 2015 18:04:07 GMT+01:00 (GMT Daylight Time)
			//  dateAlphaWsDigit
			//    May 8, 2009 5:57:51 PM
			//    oct 1, 1970
			//  dateAlphaFullMonthWs
			//      January 02, 2006 3:04pm
			//      January 02, 2006 3:04pm MST-07
			//      January 02, 2006 at 3:04pm MST-07
			//
			//  dateAlphaPeriodWsDigit
			//    oct. 1, 1970
			//  dateAlphaSlash
			//    dateAlphaSlashDigit
			//      dateAlphaSlashDigitSlash
			//        Oct/ 7/1970
			//        Oct/07/1970
			//        February/ 7/1970
			//        February/07/1970
			switch r {
			case ' ':
				// This could be a weekday or a month, detect and parse both cases.
				// skip & return to dateStart
				//   Tue 05 May 2020, 05:05:05
				//   Tuesday 05 May 2020, 05:05:05
				//   Mon Jan  2 15:04:05 2006
				//   Monday Jan  2 15:04:05 2006
				maybeDayOrMonth := strings.ToLower(p.datestr[p.skip:i])
				if isDay(maybeDayOrMonth) {
					p.skip = i + 1
					p.stateDate = dateStart
				} else if adjustedI > 3 {
					//      X
					// April 8, 2009
					// Expecting a full month name at this point
					if isMonthFull(maybeDayOrMonth) {
						p.moi = p.skip
						p.molen = i - p.skip
						p.fullMonth = maybeDayOrMonth
						p.stateDate = dateAlphaFullMonthWs
						p.dayi = i + 1
						break
					} else {
						return p, p.unknownErr(datestr)
					}

				} else if adjustedI == 3 {
					// dateAlphaWs
					//   May 05, 2005, 05:05:05
					//   May 05 2005, 05:05:05
					//   Jul 05, 2005, 05:05:05
					//   May 8 17:57:51 2009
					//   May  8 17:57:51 2009
					p.stateDate = dateAlphaWs
				} else {
					return p, p.unknownErr(datestr)
				}

			case ',':
				// Mon, 02 Jan 2006
				// Monday, 02 Jan 2006
				if adjustedI >= 3 && p.nextIs(i, ' ') {
					maybeDay := strings.ToLower(p.datestr[p.skip:i])
					if isDay(maybeDay) {
						p.stateDate = dateStart
						// Just skip past the weekday, it contains no valuable info
						p.skip = i + 2
						i++
					} else {
						return p, p.unknownErr(datestr)
					}
				}
			case '.':
				// sept. 28, 2017
				// jan. 28, 2017
				p.stateDate = dateAlphaPeriodWsDigit
				switch adjustedI {
				case 3:
					p.moi = p.skip
					p.molen = i - p.skip
					p.set(p.skip, "Jan")
				case 4:
					// gross
					newDateStr := p.datestr[p.skip:i-1] + p.datestr[i:]
					putBackParser(p)
					return parseTime(newDateStr, loc, opts...)
				default:
					return p, p.unknownErr(datestr)
				}
			case '/':
				//    X
				// Oct/ 7/1970
				// Oct/07/1970
				//         X
				// February/ 7/1970
				// February/07/1970
				// Must be a valid short or long month
				if adjustedI == 3 {
					p.moi = p.skip
					p.molen = i - p.moi
					p.set(p.moi, "Jan")
					p.stateDate = dateAlphaSlash
				} else {
					possibleFullMonth := strings.ToLower(p.datestr[p.skip:i])
					if adjustedI > 3 && isMonthFull(possibleFullMonth) {
						p.moi = p.skip
						p.molen = i - p.moi
						p.fullMonth = possibleFullMonth
						p.stateDate = dateAlphaSlash
					} else {
						return p, p.unknownErr(datestr)
					}
				}
			default:
				if !unicode.IsLetter(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateAlphaWs:
			// dateAlphaWsAlpha
			//   Mon Jan _2 15:04:05 2006
			//   Mon Jan _2 15:04:05 MST 2006
			//   Mon Jan 02 15:04:05 -0700 2006
			//   Mon Jan 02 15:04:05 2006 -0700
			//   Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			//   Fri Jul 03 2015 18:04:07 GMT+01:00 (GMT Daylight Time)
			//   Mon Aug 10 15:44:11 UTC+0100 2015
			// dateAlphaWsDigit
			//   May 8, 2009 5:57:51 PM
			//   May 8 2009 5:57:51 PM
			//   May 8 17:57:51 2009
			//   May  8 17:57:51 2009
			//   May 08 17:57:51 2009
			//   oct 1, 1970
			//   oct 7, '70
			// (this state is only entered if the skip-adjusted length is 3)
			switch {
			case unicode.IsLetter(r):
				// have to have a day of week and then at least a 3 digit month to follow
				if adjustedI >= 3 && (i+3) < len(p.datestr) {
					maybeDay := strings.ToLower(p.datestr[p.skip:i])
					if isDay(maybeDay) {
						p.skip = i
						p.stateDate = dateAlphaWsAlpha
						p.set(i, "Jan")
					} else {
						return p, p.unknownErr(datestr)
					}
				} else {
					return p, p.unknownErr(datestr)
				}
			case unicode.IsDigit(r):
				p.set(p.skip, "Jan")
				p.stateDate = dateAlphaWsDigit
				p.dayi = i
			case r == ' ':
				// continue
			default:
				return p, p.unknownErr(datestr)
			}

		case dateAlphaWsDigit:
			// May 8, 2009 5:57:51 PM
			// May 8 2009 5:57:51 PM
			// oct 1, 1970
			// oct 7, '70
			// oct. 7, 1970
			// May 8 17:57:51 2009
			// May  8 17:57:51 2009
			// May 08 17:57:51 2009
			if r == ',' {
				p.daylen = i - p.dayi
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
				p.stateDate = dateAlphaWsDigitMore
			} else if r == ' ' {
				p.daylen = i - p.dayi
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
				p.yeari = i + 1
				p.stateDate = dateAlphaWsDigitYearMaybe
				p.stateTime = timeStart
			} else if unicode.IsLetter(r) {
				p.stateDate = dateVariousDaySuffix
				i--
			} else if !unicode.IsDigit(r) {
				return p, p.unknownErr(datestr)
			}
		case dateAlphaWsDigitYearMaybe:
			//       x
			// May 8 2009 5:57:51 PM
			// May 8 17:57:51 2009
			// May  8 17:57:51 2009
			// May 08 17:57:51 2009
			// Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			// Jul 03 2015 18:04:07 GMT+01:00 (GMT Daylight Time)
			if r == ':' {
				// Guessed wrong; was not a year
				p.yeari = 0
				i = i - 3
				p.stateDate = dateAlphaWsDigit
				break iterRunes
			} else if r == ' ' {
				// must be year format, not 15:04
				p.yearlen = i - p.yeari
				if !p.setYear() {
					return p, p.unknownErr(datestr)
				}
				break iterRunes
			} else if !unicode.IsDigit(r) {
				return p, p.unknownErr(datestr)
			}
		case dateAlphaWsDigitMore:
			//       x
			// May 8, 2009 5:57:51 PM
			// May 05, 2005, 05:05:05
			// May 05 2005, 05:05:05
			// oct 1, 1970
			// oct 7, '70
			if r == ' ' {
				p.yeari = i + 1
				p.stateDate = dateAlphaWsDigitMoreWs
			} else {
				return p, p.unknownErr(datestr)
			}
		case dateAlphaWsDigitMoreWs:
			//            x
			// May 8, 2009 5:57:51 PM
			// May 05, 2005, 05:05:05
			// oct 1, 1970
			// oct 7, '70
			switch r {
			case '\'':
				p.yeari = i + 1
			case ' ':
				fallthrough
			case ',':
				//            x
				// May 8, 2009 5:57:51 PM
				//            x
				// May 8, 2009, 5:57:51 PM
				p.stateDate = dateAlphaWsDigitMoreWsYear
				p.yearlen = i - p.yeari
				if !p.setYear() {
					return p, p.unknownErr(datestr)
				}
				p.stateTime = timeStart
				break iterRunes
			default:
				if r != '\'' && !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateVariousDaySuffix:
			//        x
			// April 8th, 2009
			// April 8th 2009
			switch r {
			case 't', 'T':
				if p.nextIs(i, 'h') || p.nextIs(i, 'H') {
					if len(p.datestr) > i+2 {
						newDateStr := p.datestr[p.skip:i] + p.datestr[i+2:]
						putBackParser(p)
						return parseTime(newDateStr, loc, opts...)
					}
				}
				return p, p.unknownErr(datestr)
			case 'n', 'N':
				if p.nextIs(i, 'd') || p.nextIs(i, 'D') {
					if len(p.datestr) > i+2 {
						newDateStr := p.datestr[p.skip:i] + p.datestr[i+2:]
						putBackParser(p)
						return parseTime(newDateStr, loc, opts...)
					}
				}
				return p, p.unknownErr(datestr)
			case 's', 'S':
				if p.nextIs(i, 't') || p.nextIs(i, 'T') {
					if len(p.datestr) > i+2 {
						newDateStr := p.datestr[p.skip:i] + p.datestr[i+2:]
						putBackParser(p)
						return parseTime(newDateStr, loc, opts...)
					}
				}
				return p, p.unknownErr(datestr)
			case 'r', 'R':
				if p.nextIs(i, 'd') || p.nextIs(i, 'D') {
					if len(p.datestr) > i+2 {
						newDateStr := p.datestr[p.skip:i] + p.datestr[i+2:]
						putBackParser(p)
						return parseTime(newDateStr, loc, opts...)
					}
				}
				return p, p.unknownErr(datestr)
			default:
				return p, p.unknownErr(datestr)
			}

		case dateAlphaFullMonthWs:
			// January 02, 2006, 15:04:05
			// January 02 2006, 15:04:05
			// January 2nd, 2006, 15:04:05
			// January 2nd 2006, 15:04:05
			// September 17, 2012 at 5:00pm UTC-05
			switch {
			case r == ',':
				//           x
				// January 02, 2006, 15:04:05
				if p.nextIs(i, ' ') {
					p.daylen = i - p.dayi
					if !p.setDay() {
						return p, p.unknownErr(datestr)
					}
					p.yeari = i + 2
					p.stateDate = dateAlphaFullMonthWsDayWs
					i++
				} else {
					return p, p.unknownErr(datestr)
				}

			case r == ' ':
				//           x
				// January 02 2006, 15:04:05
				p.daylen = i - p.dayi
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
				p.yeari = i + 1
				p.stateDate = dateAlphaFullMonthWsDayWs
			case unicode.IsDigit(r):
				//         XX
				// January 02, 2006, 15:04:05
				continue
			case unicode.IsLetter(r):
				//          X
				// January 2nd, 2006, 15:04:05
				p.daylen = i - p.dayi
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
				p.stateDate = dateVariousDaySuffix
				i--
			default:
				return p, p.unknownErr(datestr)
			}
		case dateAlphaFullMonthWsDayWs:
			//                  X
			// January 02, 2006, 15:04:05
			// January 02 2006, 15:04:05
			// January 02, 2006 15:04:05
			// January 02 2006 15:04:05
			switch r {
			case ',':
				p.yearlen = i - p.yeari
				if !p.setYear() {
					return p, p.unknownErr(datestr)
				}
				p.stateTime = timeStart
				i++
				break iterRunes
			case ' ':
				p.yearlen = i - p.yeari
				if !p.setYear() {
					return p, p.unknownErr(datestr)
				}
				p.stateTime = timeStart
				break iterRunes
			default:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}
			}

		case dateAlphaPeriodWsDigit:
			//    oct. 7, '70
			switch {
			case r == ' ':
				// continue
			case unicode.IsDigit(r):
				p.stateDate = dateAlphaWsDigit
				p.dayi = i
			default:
				return p, p.unknownErr(datestr)
			}

		case dateAlphaSlash:
			//       Oct/ 7/1970
			//       February/07/1970
			switch {
			case r == ' ':
				// continue
			case unicode.IsDigit(r):
				p.stateDate = dateAlphaSlashDigit
				p.dayi = i
			default:
				return p, p.unknownErr(datestr)
			}

		case dateAlphaSlashDigit:
			// dateAlphaSlash:
			//   dateAlphaSlashDigit:
			//     dateAlphaSlashDigitSlash:
			//       Oct/ 7/1970
			//       Oct/07/1970
			//       February/ 7/1970
			//       February/07/1970
			switch {
			case r == '/':
				p.yeari = i + 1
				p.daylen = i - p.dayi
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
				p.stateDate = dateAlphaSlashDigitSlash
			case unicode.IsDigit(r):
				// continue
			default:
				return p, p.unknownErr(datestr)
			}

		case dateAlphaSlashDigitSlash:
			switch {
			case unicode.IsDigit(r):
				// continue
			case r == ' ':
				p.stateTime = timeStart
				break iterRunes
			default:
				return p, p.unknownErr(datestr)
			}

		default:
			// Reaching an unhandled state unexpectedly should always fail parsing
			return p, p.unknownErr(datestr)
		}
	}
	if !p.coalesceDate(i) {
		return p, p.unknownErr(datestr)
	}
	if p.stateTime == timeStart {
		// increment first one, since the i++ occurs at end of loop
		if i < len(p.datestr) {
			i++
		}
		// ensure we skip any whitespace prefix
		for ; i < len(p.datestr); i++ {
			r := rune(p.datestr[i])
			if r != ' ' {
				break
			}
		}

		for ; i < len(p.datestr); i++ {
			r := rune(p.datestr[i])

			// gou.Debugf("i=%d r=%s state=%d iterTimeRunes  %s %s", i, string(r), p.stateTime, p.ds(), p.ts())

			switch p.stateTime {
			case timeStart:
				// 22:43:22
				// 22:43
				// timeComma
				//   08:20:13,787
				// timeWs
				//   05:24:37 PM
				//   06:20:00 UTC
				//   06:20:00 UTC-05
				//   timeWsYear
				//     15:04:05 2008
				//   timeWsOffset
				//     00:12:00 +0000 UTC
				//     22:18:00 +0000 UTC m=+0.000000001
				//     03:21:51+00:00
				//     19:55:00+0100
				//     15:04:05 -0700
				//     15:04:05 -07:00
				// timePeriod
				//   17:24:37.3186369
				//   00:07:31.945167
				//   18:31:59.257000000
				//   00:00:00.000
				//   (and all variants that can follow the seconds portion of a time format, same as above)
				if p.houri == 0 {
					p.houri = i
				}
				switch r {
				case '-', '+':
					//   03:21:51+00:00
					p.offseti = i
					p.stateTime = timeWsOffset
					if p.seci == 0 {
						// 22:18+0530
						p.minlen = i - p.mini
					} else if p.seclen == 0 {
						p.seclen = i - p.seci
					} else if p.msi > 0 && p.mslen == 0 {
						p.mslen = i - p.msi
					} else if p.parsedAMPM {
						// time fully parsed, plus AM/PM indicator, this is OK
					} else {
						return p, p.unknownErr(datestr)
					}
				case '.', ',':
					// NOTE: go 1.20 can now parse a string that has a comma delimiter properly
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
					// (Z)ulu time
					p.loc = time.UTC
					endPos := i + 1
					if endPos > p.formatSetLen {
						p.formatSetLen = endPos
					}
				case 'a', 'A', 'p', 'P':
					if (r == 'a' || r == 'A') && (p.nextIs(i, 't') || p.nextIs(i, 'T')) {
						//                    x
						// September 17, 2012 at 5:00pm UTC-05
						i++ // skip 't'
						if p.nextIs(i, ' ') {
							//                      x
							// September 17, 2012 at 5:00pm UTC-05
							i++         // skip ' '
							p.houri = 0 // reset hour
						} else {
							return p, p.unknownErr(datestr)
						}
					} else {
						// Could be AM/PM (followed by whitespace or an offset)
						isLower := r == 'a' || r == 'p'
						isTwoLetterWord := ((i+2) == len(p.datestr) || (len(p.datestr) > i+2 && (p.datestr[i+2] == ' ' || p.datestr[i+2] == '+' || p.datestr[i+2] == '-')))
						switch {
						case isLower && p.nextIs(i, 'm') && isTwoLetterWord && !p.parsedAMPM:
							if !p.coalesceTime(i) {
								return p, p.unknownErr(datestr)
							}
							p.set(i, "pm")
							p.parsedAMPM = true
							// skip 'm'
							i++
						case !isLower && p.nextIs(i, 'M') && isTwoLetterWord && !p.parsedAMPM:
							if !p.coalesceTime(i) {
								return p, p.unknownErr(datestr)
							}
							p.set(i, "PM")
							p.parsedAMPM = true
							// skip 'M'
							i++
						default:
							return p, p.unexpectedTail(i)
						}
					}
				case ' ':
					if !p.coalesceTime(i) {
						return p, p.unknownErr(datestr)
					}
					p.stateTime = timeWs
				case ':':
					if p.mini == 0 {
						p.mini = i + 1
						p.hourlen = i - p.houri
					} else if p.seci == 0 {
						p.seci = i + 1
						p.minlen = i - p.mini
					} else if p.seci > 0 {
						// 18:31:59:257    ms uses colon, wtf
						p.seclen = i - p.seci
						if p.seclen == 2 {
							p.set(p.seci, "05")
						} else {
							return p, p.unknownErr(datestr)
						}
						p.msi = i + 1

						// gross, gross, gross.   manipulating the datestr is horrible.
						// https://github.com/araddon/dateparse/issues/117
						// Could not get the parsing to work using golang time.Parse() without
						// replacing that colon with period.
						p.set(i, ".")
						newDatestr := p.datestr[0:i] + "." + p.datestr[i+1:]
						p.datestr = newDatestr
						p.stateTime = timePeriod
					}
				}
			case timeWs:
				// timeWsAlpha
				//   06:20:00 UTC
				//   06:20:00 UTC-05
				//   15:44:11 UTC+0100 2015
				//   18:04:07 GMT+0100 (GMT Daylight Time)
				//   18:04:07 GMT+01:00 (GMT Daylight Time)
				//   17:57:51 MST 2009
				//   timeWsAMPMMaybe
				//     05:24:37 PM
				// timeWsOffset
				//   15:04:05 -0700
				//   00:12:00 +0000 UTC
				//   15:04:05 -07:00
				//   17:57:51 -0700 2009
				//   00:12:00 +00:00 UTC
				// timeWsYear
				//   00:12:00 2008
				//   merge to state timeWsOffset
				//     00:12:00 2008 -0700
				//     00:12:00 2008 -07:00
				// timeZ
				//   15:04:05.99Z
				switch r {
				case 'a', 'p', 'A', 'P':
					// Could be AM/PM or could be PST or similar
					p.tzi = i
					p.stateTime = timeWsAMPMMaybe
				case '+', '-':
					p.offseti = i
					p.stateTime = timeWsOffset
				default:
					if unicode.IsLetter(r) {
						// 06:20:00 UTC
						// 06:20:00 UTC-05
						// 15:44:11 UTC+0100 2015
						// 17:57:51 MST 2009
						p.tzi = i
						p.stateTime = timeWsAlpha
					} else if unicode.IsDigit(r) {
						// 00:12:00 2008
						if p.yeari == 0 {
							p.stateTime = timeWsYear
							p.yeari = i
						} else {
							return p, p.unknownErr(datestr)
						}
					} else if r == '(' {
						// (start of time zone description, ignore)
					} else {
						return p, p.unknownErr(datestr)
					}
				}
			case timeWsYear:
				// merge to state timeWsOffset
				//   00:12:00 2008 -0700
				//   00:12:00 2008 -07:00
				switch r {
				case ' ':
					if p.yearlen == 0 {
						p.yearlen = i - p.yeari
						if !p.setYear() {
							return p, p.unknownErr(datestr)
						}
					}
					// else allow multiple trailing whitespace
				case '+', '-':
					// The year must be followed by a space before an offset!
					if p.yearlen > 0 {
						p.offseti = i
						p.stateTime = timeWsOffset
					} else {
						return p, p.unknownErr(datestr)
					}
				default:
					if unicode.IsDigit(r) {
						if p.yearlen > 0 {
							return p, p.unknownErr(datestr)
						}
					} else {
						return p, p.unknownErr(datestr)
					}
				}
			case timeWsAlpha:
				// 06:20:00 UTC
				// 06:20:00 UTC-05
				// 06:20:00 (EST)
				// timeWsAlphaWs
				//   timeWsAlphaWsYear
				//     17:57:51 MST 2009
				// timeWsAlphaZoneOffset
				// timeWsAlphaZoneOffsetWs
				//   timeWsAlphaZoneOffsetWsExtra
				//     18:04:07 GMT+0100 (GMT Daylight Time)
				//     18:04:07 GMT+01:00 (GMT Daylight Time)
				//   timeWsAlphaZoneOffsetWsYear
				//     15:44:11 UTC+0100 2015
				switch r {
				case '+', '-':
					tzNameLower := strings.ToLower(p.datestr[p.tzi:i])
					if tzNameLower == "gmt" || tzNameLower == "utc" || tzNameLower == "tz" {
						// This is a special form where the actual timezone isn't UTC, but is rather
						// specifying that the correct offset is a specified numeric offset from UTC:
						// 06:20:00 UTC-05
						// 06:20:00 GMT+02
						p.tzi = 0
						p.tzlen = 0
					} else {
						p.tzlen = i - p.tzi
					}
					if p.tzlen > 0 {
						if err := p.setTZName(datestr); err != nil {
							return p, err
						}
					}
					p.stateTime = timeWsAlphaZoneOffset
					p.offseti = i
				case ' ', ')':
					// 17:57:51 MST 2009
					// 17:57:51 MST
					// 06:20:00 (EST)
					p.tzlen = i - p.tzi
					if err := p.setTZName(datestr); err != nil {
						return p, err
					}
					if r == ' ' {
						p.stateTime = timeWsAlphaWs
					} else {
						// 06:20:00 (EST)
						// This must be the end of the datetime or the format is unknown
						if i+1 == len(p.datestr) {
							p.stateTime = timeWsAlphaRParen
						} else {
							return p, p.unknownErr(datestr)
						}
					}
				}
			case timeWsAlphaWs:
				// timeWsAlphaWsYear
				//   17:57:51 MST 2009
				if unicode.IsDigit(r) {
					if p.yeari == 0 {
						p.yeari = i
					} else {
						return p, p.unknownErr(datestr)
					}
					p.stateTime = timeWsAlphaWsYear
				} else if r == '(' {
					p.extra = i - 1
					p.stateTime = timeWsOffsetWsTZDescInParen
				}
			case timeWsAlphaWsYear:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}

			case timeWsAlphaZoneOffset:
				// 06:20:00 UTC-05
				// timeWsAlphaZoneOffset
				// timeWsAlphaZoneOffsetWs
				//   timeWsAlphaZoneOffsetWsExtra
				//     18:04:07 GMT+0100 (GMT Daylight Time)
				//     18:04:07 GMT+01:00 (GMT Daylight Time)
				//   timeWsAlphaZoneOffsetWsYear
				//     15:44:11 UTC+0100 2015
				switch r {
				case ' ':
					if err := p.setTZOffset(i, datestr); err != nil {
						return p, err
					}
					p.stateTime = timeWsAlphaZoneOffsetWs
				default:
					if r != ':' && !unicode.IsDigit(r) {
						return p, p.unknownErr(datestr)
					}
				}
			case timeWsAlphaZoneOffsetWs:
				// timeWsAlphaZoneOffsetWs
				//   timeWsAlphaZoneOffsetWsExtra
				//     18:04:07 GMT+0100 (GMT Daylight Time)
				//     18:04:07 GMT+01:00 (GMT Daylight Time)
				//   timeWsAlphaZoneOffsetWsYear
				//     15:44:11 UTC+0100 2015
				if unicode.IsDigit(r) {
					if p.yeari == 0 {
						p.yeari = i
						p.stateTime = timeWsAlphaZoneOffsetWsYear
					} else {
						return p, p.unknownErr(datestr)
					}
				} else if r == '(' {
					p.extra = i - 1
					p.stateTime = timeWsOffsetWsTZDescInParen
				} else {
					return p, p.unknownErr(datestr)
				}
			case timeWsOffsetWsTZDescInParen:
				// timeWsAlphaZoneOffsetWs
				//   timeWsAlphaZoneOffsetWsExtra
				//     18:04:07 GMT+0100 (GMT Daylight Time)
				//     18:04:07 GMT+01:00 (GMT Daylight Time)
				switch r {
				case '(':
					return p, p.unknownErr(datestr)
				case ')':
					// must be the end
					if i != len(p.datestr)-1 {
						return p, p.unknownErr(datestr)
					}
				}
				// any other char is OK
			case timeWsAlphaZoneOffsetWsYear:
				// 15:44:11 UTC+0100 2015
				if unicode.IsDigit(r) {
					p.yearlen = i - p.yeari + 1
					if p.yearlen == 4 {
						if !p.setYear() {
							return p, p.unknownErr(datestr)
						}
					} else if p.yearlen > 4 {
						return p, p.unknownErr(datestr)
					}
				} else {
					return p, p.unknownErr(datestr)
				}
			case timeWsAMPMMaybe:
				// timeWsAMPMMaybe
				//   timeWsAMPM
				//     05:24:37 PM
				//   timeWsAlpha
				//     00:12:00 PST
				//     15:44:11 UTC+0100 2015
				isTwoLetterWord := ((i+1) == len(p.datestr) || p.nextIs(i, ' '))
				if (r == 'm' || r == 'M') && isTwoLetterWord {
					if p.parsedAMPM {
						return p, p.unexpectedTail(i)
					}
					// This isn't a time zone after all...
					p.tzi = 0
					p.stateTime = timeWsAMPM
					if r == 'm' {
						p.set(i-1, "pm")
					} else {
						p.set(i-1, "PM")
					}
					p.parsedAMPM = true
					switch p.hourlen {
					case 2:
						p.set(p.houri, "03")
					case 1:
						p.set(p.houri, "3")
					default:
						return p, p.unknownErr(datestr)
					}
				} else {
					p.stateTime = timeWsAlpha
				}

			case timeWsAMPM:
				// If we have a continuation after AM/PM indicator, reset parse state back to ws
				if r == ' ' {
					p.stateTime = timeWs
				} else {
					// unexpected garbage after AM/PM indicator, fail
					return p, p.unexpectedTail(i)
				}

			case timeWsOffset:
				// timeWsOffset
				//   15:04:05 -0700
				//   timeWsOffsetWsOffset
				//     17:57:51 -0700 -07
				//   timeWsOffsetWs
				//     15:04:05 -07:00
				//     timeWsOffsetWsYear
				//       17:57:51 -0700 2009
				//     timeWsOffsetWsAlphaZone
				//       00:12:00 +0000 UTC
				//       00:12:00 +00:00 UTC
				//       timeWsOffsetWsAlphaZoneWs --> timeWsOffsetWsYear (overloaded)
				//         00:12:00 +00:00 UTC 2009
				//       timeWsOffsetWsTZDescInParen
				//         00:12:00 +00:00 UTC (Universal Coordinated Time)
				switch r {
				case ':':
					// Parse the case where an offset has a colon the same as timeWsOffset!
					// continue
				case ' ':
					if err := p.setTZOffset(i, datestr); err != nil {
						return p, err
					}
					p.stateTime = timeWsOffsetWs
				default:
					if !unicode.IsDigit(r) {
						return p, p.unknownErr(datestr)
					}
				}
			case timeWsOffsetWs:
				// timeWsOffsetWs
				//   timeWsOffsetWsYear
				//     17:57:51 -0700 2009
				//     17:57:51 -07:00 2009
				//   timeWsOffsetWsAlphaZone
				//     00:12:00 +0000 UTC
				//     00:12:00 +00:00 UTC
				//     22:18:00.001 +0000 UTC m=+0.000000001
				//     22:18:00.001 +00:00 UTC m=+0.000000001
				// w Extra
				//   17:57:51 -0700 -07
				//   17:57:51 -07:00 -07
				//   22:18:00.001 +0000 m=+0.000000001
				//   00:00:00 +0300 (European Daylight Time)
				//   00:00:00 +03:00 (European Daylight Time)
				switch r {
				case '+', '-':
					// This really doesn't seem valid, but for some reason when round-tripping a go date
					// their is an extra +03 printed out.  seems like go bug to me, but, parsing anyway.
					// 00:00:00 +0300 +03
					// 00:00:00 +0300 +0300
					// 00:00:00 +03:00 +03
					// 00:00:00 +03:00 +0300
					p.extra = i - 1
					p.trimExtra(false)
					p.stateTime = timeWsOffset
				case '(':
					// 00:00:00 +0300 (European Daylight Time)
					// 00:00:00 +03:00 (European Daylight Time)
					p.extra = i - 1
					p.stateTime = timeWsOffsetWsTZDescInParen
				case ' ':
					// continue
				default:
					switch {
					case unicode.IsDigit(r):
						if p.yeari == 0 {
							p.yeari = i
						} else {
							return p, p.unknownErr(datestr)
						}
						p.stateTime = timeWsOffsetWsYear
					case unicode.IsLetter(r):
						if r == 'm' && p.nextIs(i, '=') {
							// 22:18:00.001 +0000 UTC m=+0.000000001
							// 22:18:00.001 +00:00 UTC m=+0.000000001
							// very strange syntax!
							p.extra = i - 1
							p.trimExtra(false)
						} else {
							// 15:04:05 -0700 MST
							// 15:04:05 -07:00 MST
							// 15:04:05 -07:00 MST (Mountain Standard Time)
							// 15:04:05 -07:00 MST 2006
							if p.tzi == 0 {
								p.tzi = i
							} else {
								return p, p.unknownErr(datestr)
							}
							p.stateTime = timeWsOffsetWsAlphaZone
						}
					default:
						return p, p.unknownErr(datestr)
					}
				}

			case timeWsOffsetWsAlphaZone:
				switch {
				case r == ' ':
					if p.tzi > 0 {
						p.tzlen = i - p.tzi
						if err := p.setTZName(datestr); err != nil {
							return p, err
						}
					} else {
						return p, p.unknownErr(datestr)
					}
					p.stateTime = timeWsOffsetWsAlphaZoneWs
				case unicode.IsLetter(r):
					// continue
				}

			case timeWsOffsetWsAlphaZoneWs:
				switch r {
				case '=':
					// 22:18:00.001 +0000 UTC m=+0.000000001
					// very strange syntax!
					if p.datestr[i-1] == 'm' {
						p.extra = i - 2
						p.trimExtra(false)
					} else {
						return p, p.unknownErr(datestr)
					}
				case '(':
					// 00:00:00 -0600 MDT (Mountain Daylight Time)
					// 00:00:00 -06:00 MDT (Mountain Daylight Time)
					p.extra = i - 1
					p.stateTime = timeWsOffsetWsTZDescInParen
				case ' ':
					// continue (extra whitespace)
				case 'm':
					if !p.nextIs(i, '=') {
						return p, p.unknownErr(datestr)
					}
				default:
					if unicode.IsDigit(r) {
						if p.yeari == 0 {
							p.yeari = i
						} else {
							return p, p.unknownErr(datestr)
						}
						p.stateTime = timeWsOffsetWsYear
					} else {
						return p, p.unknownErr(datestr)
					}
				}

			case timeWsOffsetWsYear:
				if !unicode.IsDigit(r) {
					return p, p.unknownErr(datestr)
				}

			case timePeriod:
				// 15:04:05.999999999
				// 15:04:05.999999999
				// 15:04:05.999999
				// 15:04:05.999999
				// 15:04:05.999
				// 15:04:05.999
				// timePeriod
				//   17:24:37.3186369
				//   00:07:31.945167
				//   18:31:59.257000000
				//   00:00:00.000
				//   (note: if we have an offset (+/-) or whitespace (Ws) after this state, re-enter the timeWs or timeWsOffset
				//    state above so that we do not have to duplicate all of the logic again for this parsing just because we
				//    have parsed a fractional second...)
				switch r {
				case ' ':
					p.mslen = i - p.msi
					if !p.coalesceTime(i) {
						return p, p.unknownErr(datestr)
					}
					p.stateTime = timeWs
				case '+', '-':
					p.mslen = i - p.msi
					p.offseti = i
					p.stateTime = timeWsOffset
				case 'Z':
					p.stateTime = timeZ
					p.mslen = i - p.msi
					// (Z)ulu time
					p.loc = time.UTC
					endPos := i + 1
					if endPos > p.formatSetLen {
						p.formatSetLen = endPos
					}
				case 'a', 'A', 'p', 'P':
					// Could be AM/PM
					isLower := r == 'a' || r == 'p'
					isTwoLetterWord := ((i+2) == len(p.datestr) || p.nextIs(i+1, ' '))
					switch {
					case isLower && p.nextIs(i, 'm') && isTwoLetterWord && !p.parsedAMPM:
						p.mslen = i - p.msi
						if !p.coalesceTime(i) {
							return p, p.unknownErr(datestr)
						}
						p.set(i, "pm")
						p.parsedAMPM = true
						// skip 'm'
						i++
						p.stateTime = timePeriodAMPM
					case !isLower && p.nextIs(i, 'M') && isTwoLetterWord && !p.parsedAMPM:
						p.mslen = i - p.msi
						if !p.coalesceTime(i) {
							return p, p.unknownErr(datestr)
						}
						p.set(i, "PM")
						p.parsedAMPM = true
						// skip 'M'
						i++
						p.stateTime = timePeriodAMPM
					default:
						return p, p.unexpectedTail(i)
					}
				default:
					if !unicode.IsDigit(r) {
						return p, p.unexpectedTail(i)
					}
				}
			case timePeriodAMPM:
				switch r {
				case ' ':
					p.stateTime = timeWs
				case '+', '-':
					p.offseti = i
					p.stateTime = timeWsOffset
				default:
					return p, p.unexpectedTail(i)
				}
			case timeZ:
				// nothing expected can come after Z
				return p, p.unexpectedTail(i)
			}
		}

		switch p.stateTime {
		case timeWsAlpha:
			p.tzlen = i - p.tzi
			if err := p.setTZName(datestr); err != nil {
				return p, err
			}

		case timeWsAlphaRParen:
			// nothing extra to do

		case timeWsYear, timeWsAlphaWsYear:
			p.yearlen = i - p.yeari
			if !p.setYear() {
				return p, p.unknownErr(datestr)
			}
		case timeWsOffsetWsTZDescInParen:
			// The last character must be a closing ')'
			if i <= 0 || p.datestr[i-1] != ')' {
				return p, p.unknownErr(datestr)
			}
			// As a special case, if we don't yet have a timezone name,
			// and the content in the paren is 3-4 characters, then treat
			// this as a time zone name instead
			if len(p.datestr) >= p.extra+1+3+1 {
				parenContentsLen := (i - 1) - (p.extra + 2)
				if p.tzi == 0 && (parenContentsLen >= 3 && parenContentsLen <= 4) {
					p.tzi = p.extra + 2
					p.tzlen = parenContentsLen
					if err := p.setTZName(datestr); err != nil {
						return p, err
					}
					p.extra = 0
				}
			}
			if p.extra > 0 {
				p.trimExtra(false)
			}
		case timeWsAlphaZoneOffset:
			// 06:20:00 UTC-05
			if err := p.setTZOffset(i, datestr); err != nil {
				return p, err
			}

		case timePeriod:
			p.mslen = i - p.msi
			if p.mslen >= 10 {
				if p.simpleErrorMessages {
					return p, ErrFracSecTooLong
				} else {
					return p, fmt.Errorf("%w in %q near %q", ErrFracSecTooLong, datestr, p.datestr[p.msi:p.mslen])
				}
			}
		case timeWsOffset:
			// 17:57:51 -07:00 (or 19:55:00.799 +01:00)
			// 15:04:05+07:00 (or 19:55:00.799+01:00)
			// 17:57:51 2006 -07:00 (or 19:55:00.799 +01:00)
			if err := p.setTZOffset(len(p.datestr), datestr); err != nil {
				return p, err
			}

		case timeWsOffsetWsYear:
			// 17:57:51 -0700 2009
			p.yearlen = len(p.datestr) - p.yeari
			if p.yearlen == 4 {
				if !p.setYear() {
					return p, p.unknownErr(datestr)
				}
			} else if p.yearlen > 4 {
				return p, p.unknownErr(datestr)
			}

		case timeWsOffsetWsAlphaZone:
			// 00:12:00 +0000 UTC
			if p.tzi > 0 {
				p.tzlen = i - p.tzi
				if err := p.setTZName(datestr); err != nil {
					return p, err
				}
			} else {
				return p, p.unknownErr(datestr)
			}
		}
		if !p.coalesceTime(i) {
			return p, p.unknownErr(datestr)
		}
	}

	switch p.stateDate {
	case dateDigit:
		// unixy timestamps ish
		//  example              ct type
		//  1499979655583057426  19 nanoseconds
		//  1499979795437000     16 micro-seconds
		//  20180722105203       14 yyyyMMddhhmmss
		//  1499979795437        13 milliseconds
		//  1332151919           10 seconds
		//  20140601             8  yyyymmdd
		//  2014                 4  yyyy
		t := time.Time{}
		if len(p.datestr) == len("1499979655583057426") { // 19
			// nano-seconds
			if nanoSecs, err := strconv.ParseInt(p.datestr, 10, 64); err == nil {
				t = time.Unix(0, nanoSecs)
			}
		} else if len(p.datestr) == len("1499979795437000") { // 16
			// micro-seconds
			if microSecs, err := strconv.ParseInt(p.datestr, 10, 64); err == nil {
				t = time.Unix(0, microSecs*1000)
			}
		} else if len(p.datestr) == len("yyyyMMddhhmmss") { // 14
			// yyyyMMddhhmmss
			p.setEntireFormat([]byte("20060102150405"))
			return p, nil
		} else if len(p.datestr) == len("1332151919000") { // 13
			if miliSecs, err := strconv.ParseInt(p.datestr, 10, 64); err == nil {
				t = time.Unix(0, miliSecs*1000*1000)
			}
		} else if len(p.datestr) == len("1332151919") { //10
			if secs, err := strconv.ParseInt(p.datestr, 10, 64); err == nil {
				t = time.Unix(secs, 0)
			}
		} else if len(p.datestr) == len("20140601") {
			p.setEntireFormat([]byte("20060102"))
			return p, nil
		} else if len(p.datestr) == len("2014") {
			p.setEntireFormat([]byte("2006"))
			return p, nil
		} else {
			return p, p.unknownErr(datestr)
		}
		if !t.IsZero() {
			if loc == nil {
				p.t = &t
				return p, nil
			}
			t = t.In(loc)
			p.t = &t
			return p, nil
		} else {
			return p, p.unknownErr(datestr)
		}
	case dateDigitSt:
		// 171113 14:14:20
		return p, nil

	case dateYearDash:
		// 2006-01
		return p, nil

	case dateYearDashDash:
		// 2006-01-02
		// 2006-1-02
		// 2006-1-2
		// 2006-01-2
		return p, nil

	case dateYearDashDashOffset:
		///  2020-07-20+00:00
		if err := p.setTZOffset(len(p.datestr), datestr); err != nil {
			return p, err
		}
		return p, nil

	case dateYearDashAlpha:
		// 2013-Feb-03
		// 2013-Feb-3
		// 2013-February-3
		return p, nil

	case dateYearDashDashWs:
		// 2013-04-01
		return p, nil

	case dateYearDashDashT:
		return p, nil

	case dateDigitDashAlphaDash, dateDigitDashDigitDash:
		// This has already been done if we parsed the time already
		if p.stateTime == timeIgnore {
			// dateDigitDashAlphaDash:
			//   13-Feb-03   ambiguous
			//   28-Feb-03   ambiguous
			//   29-Jun-2016
			// dateDigitDashDigitDash:
			//   29-06-2026
			length := len(p.datestr) - (p.moi + p.molen + 1)
			switch length {
			case 4:
				p.yearlen = 4
				p.set(p.yeari, "2006")
				// We now also know that part1 was the day
				p.dayi = p.skip
				p.daylen = p.part1Len
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
			case 2:
				// We have no idea if this is
				// yy-mon-dd   OR  dd-mon-yy
				// (or for dateDigitDashDigitDash, yy-mm-dd  OR  dd-mm-yy)
				//
				// We are going to ASSUME (bad, bad) that it is dd-mon-yy (dd-mm-yy),
				// which is a horrible assumption, but seems to be the convention for
				// dates that are formatted in this way.
				p.ambiguousMD = true // not retryable
				p.yearlen = 2
				p.set(p.yeari, "06")
				// We now also know that part1 was the day
				p.dayi = p.skip
				p.daylen = p.part1Len
				if !p.setDay() {
					return p, p.unknownErr(datestr)
				}
			default:
				return p, p.unknownErr(datestr)
			}
		}

		return p, nil

	case dateDigitDot:
		if len(datestr) == len("yyyyMMddhhmmss.SSS") { // 18
			p.setEntireFormat([]byte("20060102150405.000"))
			return p, nil
		} else {
			// 2014.05
			p.molen = i - p.moi
			if !p.setMonth() {
				return p, p.unknownErr(datestr)
			}
			return p, nil
		}

	case dateDigitDotDot:
		// 03.31.1981
		// 3.31.2014
		// 3.2.1981
		// 3.2.81
		// 08.21.71
		// 2018.09.30
		return p, nil

	case dateDigitDotDotWs:
		// 2013.04.01
		return p, nil

	case dateDigitDotDotT:
		return p, nil

	case dateDigitDotDotOffset:
		//  2020.07.20+00:00
		if err := p.setTZOffset(len(p.datestr), datestr); err != nil {
			return p, err
		}
		return p, nil

	case dateDigitWsMoYear:
		// 2 Jan 2018
		// 2 Jan 18
		// 2 Jan 2018 23:59
		// 02 Jan 2018 23:59
		// 12 Feb 2006, 19:17
		return p, nil

	case dateAlphaFullMonthWs:
		if p.stateTime == timeIgnore && p.yearlen == 0 {
			p.yearlen = i - p.yeari
			if !p.setYear() {
				return p, p.unknownErr(datestr)
			}
		}
		return p, nil

	case dateAlphaFullMonthWsDayWs:
		return p, nil

	case dateAlphaWsDigitMoreWs:
		// oct 1, 1970
		p.yearlen = i - p.yeari
		if !p.setYear() {
			return p, p.unknownErr(datestr)
		}
		return p, nil

	case dateAlphaWsDigitMoreWsYear:
		// May 8, 2009 5:57:51 PM
		// Jun 7, 2005, 05:57:51
		return p, nil

	case dateAlphaWsAlpha:
		return p, nil

	case dateAlphaWsDigit:
		return p, nil

	case dateAlphaWsDigitYearMaybe:
		return p, nil

	case dateDigitSlash:
		// 3/1/2014
		// 10/13/2014
		// 01/02/2006
		return p, nil

	case dateDigitSlashAlphaSlash:
		// 03/Jun/2014
		return p, nil

	case dateDigitYearSlash:
		// 2014/10/13
		return p, nil

	case dateDigitColon:
		// 3:1:2014
		// 10:13:2014
		// 01:02:2006
		// 2014:10:13
		return p, nil

	case dateDigitChineseYear:
		// dateDigitChineseYear
		//   2014年04月08日
		//   2014年4月12日
		return p, nil

	case dateDigitChineseYearWs:
		//   2014年04月08日 00:00:00 ...
		return p, nil

	case dateAlphaSlashDigitSlash:
		// Oct/ 7/1970
		// February/07/1970
		return p, nil

	case dateYearWsMonthWs:
		// 2013 May 02 11:37:55
		// 2013 December 02 11:37:55
		return p, nil

	}

	return p, p.unknownErr(datestr)
}

type parser struct {
	loc                        *time.Location
	preferMonthFirst           bool
	retryAmbiguousDateWithSwap bool
	ambiguousMD                bool
	ambiguousRetryable         bool
	allowPartialStringMatch    bool
	simpleErrorMessages        bool
	stateDate                  dateState
	stateTime                  timeState
	format                     []byte
	formatSetLen               int
	datestr                    string
	fullMonth                  string
	parsedAMPM                 bool
	skip                       int
	link                       int
	extra                      int
	part1Len                   int
	yeari                      int
	yearlen                    int
	moi                        int
	molen                      int
	dayi                       int
	daylen                     int
	houri                      int
	hourlen                    int
	mini                       int
	minlen                     int
	seci                       int
	seclen                     int
	msi                        int
	mslen                      int
	offseti                    int
	tzi                        int
	tzlen                      int
	t                          *time.Time
}

// something like: "Wednesday,  8 February 2023 19:00:46.999999999 +11:00 (AEDT) m=+0.000000001"
const longestPossibleDateStr = 78

// the format byte slice is always a little larger, in case we need to expand it to contain a full month
const formatExtraBufferBytes = 16
const formatBufferCapacity = longestPossibleDateStr + formatExtraBufferBytes

var parserPool = sync.Pool{
	New: func() interface{} {
		// allocate a max-sized fixed-capacity format byte slice
		// that will be re-used with this parser struct
		return &parser{
			format: make([]byte, 0, formatBufferCapacity),
		}
	},
}

var emptyString = ""

// Use to put a parser back into the pool in the right way
func putBackParser(p *parser) {
	if p == nil {
		return
	}
	// we'll be reusing the backing memory for the format byte slice, put it back
	// to maximum capacity
	if cap(p.format) == formatBufferCapacity {
		p.format = p.format[:formatBufferCapacity]
	} else {
		// the parsing improperly process replaced this, get back a new one with the right cap
		p.format = make([]byte, 0, formatBufferCapacity)
	}
	// clear out pointers so we don't leak memory we don't need any longer
	p.loc = nil
	p.datestr = emptyString
	p.fullMonth = emptyString
	p.t = nil
	parserPool.Put(p)
}

// ParserOption defines a function signature implemented by options
// Options defined like this accept the parser and operate on the data within
type ParserOption func(*parser) error

// PreferMonthFirst is an option that allows preferMonthFirst to be changed from its default
func PreferMonthFirst(preferMonthFirst bool) ParserOption {
	return func(p *parser) error {
		p.preferMonthFirst = preferMonthFirst
		return nil
	}
}

// RetryAmbiguousDateWithSwap is an option that allows retryAmbiguousDateWithSwap to be changed from its default
func RetryAmbiguousDateWithSwap(retryAmbiguousDateWithSwap bool) ParserOption {
	return func(p *parser) error {
		p.retryAmbiguousDateWithSwap = retryAmbiguousDateWithSwap
		return nil
	}
}

// AllowPartialStringMatch is an option that allows allowPartialStringMatch to be changed from its default.
// If true, then strings can be attempted to be parsed / matched even if the end of the string might contain
// more than a date/time. This defaults to false.
func AllowPartialStringMatch(allowPartialStringMatch bool) ParserOption {
	return func(p *parser) error {
		p.allowPartialStringMatch = allowPartialStringMatch
		return nil
	}
}

// SimpleErrorMessages is an option that will cause returned error messages to contain less detail,
// but it will avoid allocating any memory for the custom error message. If you expect to attempt
// to parse a lot of text that is not valid, this could help reduce GC pressure.
func SimpleErrorMessages(simpleErrorMessages bool) ParserOption {
	return func(p *parser) error {
		p.simpleErrorMessages = simpleErrorMessages
		return nil
	}
}

// Creates a new parser. The caller must call putBackParser on the returned parser when done with it.
func newParser(dateStr string, loc *time.Location, opts ...ParserOption) (*parser, error) {
	dateStrLen := len(dateStr)
	if dateStrLen > longestPossibleDateStr {
		var nilParser *parser
		return nil, nilParser.unknownErr(dateStr)
	}

	// Make sure to re-use the format byte slice from the pooled parser struct
	p := parserPool.Get().(*parser)
	// This re-slicing is guaranteed to work because of the length check above
	startingFormat := p.format[:dateStrLen]
	copy(startingFormat, dateStr)
	*p = parser{
		stateDate:                  dateStart,
		stateTime:                  timeIgnore,
		datestr:                    dateStr,
		loc:                        loc,
		preferMonthFirst:           true,
		retryAmbiguousDateWithSwap: false,
		format:                     startingFormat,
		// this tracks how much of the format string has been set, to make sure all of it is set
		formatSetLen: 0,
	}

	// allow the options to mutate the parser fields from their defaults
	for _, option := range opts {
		if err := option(p); err != nil {
			return nil, fmt.Errorf("option error: %w", err)
		}
	}
	return p, nil
}

func (p *parser) nextIs(i int, b byte) bool {
	if len(p.datestr) > i+1 && p.datestr[i+1] == b {
		return true
	}
	return false
}

func (p *parser) setEntireFormat(format []byte) {
	// Copy so that we don't lose this pooled format byte slice
	oldLen := len(p.format)
	newLen := len(format)
	if oldLen != newLen {
		// guaranteed to work because of the allocated capacity for format buffers
		p.format = p.format[:newLen]
	}
	copy(p.format, format)
	p.formatSetLen = len(format)
}

func (p *parser) set(start int, val string) {
	if start < 0 {
		return
	}
	if len(p.format) < start+len(val) {
		return
	}
	for i, r := range val {
		p.format[start+i] = byte(r)
	}
	endingPos := start + len(val)
	if endingPos > p.formatSetLen {
		p.formatSetLen = endingPos
	}
}

func (p *parser) setMonth() bool {
	switch p.molen {
	case 2:
		p.set(p.moi, "01")
		return true
	case 1:
		p.set(p.moi, "1")
		return true
	default:
		return false
	}
}

func (p *parser) setDay() bool {
	switch p.daylen {
	case 2:
		p.set(p.dayi, "02")
		return true
	case 1:
		p.set(p.dayi, "2")
		return true
	default:
		return false
	}
}

func (p *parser) setYear() bool {
	switch p.yearlen {
	case 2:
		p.set(p.yeari, "06")
		return true
	case 4:
		p.set(p.yeari, "2006")
		return true
	default:
		return false
	}
}

func (p *parser) setTZOffset(i int, datestr string) error {
	offsetlen := i - p.offseti
	switch offsetlen {
	case 3:
		p.set(p.offseti, "-07")
	case 5:
		p.set(p.offseti, "-0700")
	case 6:
		p.set(p.offseti, "-07:00")
	default:
		if p.simpleErrorMessages {
			return ErrUnknownTZOffset
		} else {
			return fmt.Errorf("%w %q near %q (must be 2 or 4 digits optional colon)", ErrUnknownTZOffset, datestr, p.datestr[p.offseti:i])
		}
	}
	return nil
}

func (p *parser) setTZName(datestr string) error {
	switch p.tzlen {
	case 3:
		p.set(p.tzi, "MST")
	case 4:
		p.set(p.tzi, "MST ")
	default:
		if p.simpleErrorMessages {
			return ErrUnknownTimeZone
		} else {
			return fmt.Errorf("%w %q near %q (must be 3 or 4 characters)", ErrUnknownTimeZone, datestr, p.datestr[p.tzi:p.tzi+p.tzlen])
		}
	}
	return nil
}

// Removes the characters at the given range from the format string.
// Fills the end of the format string with spaces rather than shortening it.
func (p *parser) removeRangeFromFormat(i, numBytes int) {
	if i < 0 || i >= len(p.format) {
		return
	}
	var startErase int
	afterRemovedRange := i + numBytes
	bytesToCopy := len(p.format) - afterRemovedRange
	if bytesToCopy <= 0 {
		// nothing to copy, erase everything from the removal point
		startErase = i
	} else {
		copy(p.format[i:], p.format[afterRemovedRange:])
		startErase = i + bytesToCopy
	}
	// fill in spaces to erase the moved content in its old location
	for index := startErase; index < len(p.format); index++ {
		p.format[index] = ' '
	}
}

// Find the proper end of the current component (scanning chars starting from start and going
// up until the end, and either returning at end or returning the first character that is
// not allowed, as determined by allowNumeric, allowAlpha, and allowOther)
func findProperEnd(s string, start, end int, allowNumeric bool, allowAlpha bool, allowOther bool) int {
	for i := start; i < end; i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			if !allowNumeric {
				return i
			}
		} else if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			if !allowAlpha {
				return i
			}
		} else {
			if !allowOther {
				return i
			}
		}
	}
	return end
}

func (p *parser) coalesceDate(end int) bool {
	if p.yeari > 0 {
		if p.yearlen == 0 {
			p.yearlen = findProperEnd(p.datestr, p.yeari, end, true, false, false) - p.yeari
		}
		if !p.setYear() {
			return false
		}
	}
	if p.moi > 0 && p.molen == 0 {
		p.molen = findProperEnd(p.datestr, p.moi, end, true, true, false) - p.moi
		// The month may be the name of the month, so don't treat as invalid in this case.
		// We can ignore the return value here.
		p.setMonth()
	}
	if p.dayi > 0 && p.daylen == 0 {
		p.daylen = findProperEnd(p.datestr, p.dayi, end, true, false, false) - p.dayi
		if !p.setDay() {
			return false
		}
	}
	return true
}
func (p *parser) ts() string {
	return fmt.Sprintf("h:(%d:%d) m:(%d:%d) s:(%d:%d)", p.houri, p.hourlen, p.mini, p.minlen, p.seci, p.seclen)
}
func (p *parser) ds() string {
	return fmt.Sprintf("%s d:(%d:%d) m:(%d:%d) y:(%d:%d)", p.datestr, p.dayi, p.daylen, p.moi, p.molen, p.yeari, p.yearlen)
}
func (p *parser) coalesceTime(end int) bool {
	// 03:04:05
	// 15:04:05
	// 3:04:05
	// 3:4:5
	// 15:04:05.00
	if p.houri > 0 {
		switch p.hourlen {
		case 2:
			p.set(p.houri, "15")
		case 1:
			p.set(p.houri, "3")
		default:
			return false
		}
	}
	if p.mini > 0 {
		if p.minlen == 0 {
			p.minlen = end - p.mini
		}
		switch p.minlen {
		case 2:
			p.set(p.mini, "04")
		case 1:
			p.set(p.mini, "4")
		default:
			return false
		}
	}
	if p.seci > 0 {
		if p.seclen == 0 {
			p.seclen = end - p.seci
		}
		switch p.seclen {
		case 2:
			p.set(p.seci, "05")
		case 1:
			p.set(p.seci, "5")
		default:
			return false
		}
	}

	if p.msi > 0 {
		for i := 0; i < p.mslen; i++ {
			p.format[p.msi+i] = '0'
		}
		endPos := p.msi + p.mslen
		if endPos > p.formatSetLen {
			p.formatSetLen = endPos
		}
	}
	return true
}
func (p *parser) setFullMonth(month string) {
	oldLen := len(p.format)
	const fullMonth = "January"
	// Do an overlapping copy so we don't lose the pooled format buffer
	part1Len := p.moi
	part3 := p.format[p.moi+len(month):]
	newLen := part1Len + len(fullMonth) + len(part3)
	if newLen > oldLen {
		// We can re-slice this, because the capacity is guaranteed to be a little longer than any possible datestr
		p.format = p.format[:newLen]
	}
	// first part will not change, we need to shift the third part
	copy(p.format[part1Len+len(fullMonth):], part3)
	copy(p.format[part1Len:], fullMonth)
	// shorten the format slice now if needed
	if newLen < oldLen {
		p.format = p.format[:newLen]
	}

	if newLen > oldLen && p.formatSetLen >= p.moi {
		p.formatSetLen += newLen - oldLen
	} else if newLen < oldLen && p.formatSetLen >= p.moi {
		p.formatSetLen -= oldLen - newLen
	}

	if p.formatSetLen > len(p.format) {
		p.formatSetLen = len(p.format)
	} else if p.formatSetLen < len(fullMonth) {
		p.formatSetLen = len(fullMonth)
	} else if p.formatSetLen < 0 {
		p.formatSetLen = 0
	}
}

func (p *parser) trimExtra(onlyTrimFormat bool) {
	if p.extra > 0 && len(p.format) > p.extra {
		p.format = p.format[0:p.extra]
		if p.formatSetLen > len(p.format) {
			p.formatSetLen = len(p.format)
		}
		if !onlyTrimFormat {
			p.datestr = p.datestr[0:p.extra]
		}
	}
}

func (p *parser) parse(originalLoc *time.Location, originalOpts ...ParserOption) (t time.Time, err error) {
	if p == nil {
		var nilParser *parser
		return time.Time{}, nilParser.unknownErr("")
	}
	if p.t != nil {
		return *p.t, nil
	}

	// Make sure that the entire string matched to a known format that was detected
	if !p.allowPartialStringMatch && p.formatSetLen < len(p.format) {
		// We can always ignore punctuation at the end of a date/time, but do not allow
		// any numbers or letters in the format string.
		validFormatTo := findProperEnd(bytesToString(p.format), p.formatSetLen, len(p.format), false, false, true)
		if validFormatTo < len(p.format) {
			return time.Time{}, p.unexpectedTail(p.formatSetLen)
		}
	}

	// Special case where the TZ name is 4 characters long and followed by punctuation, will cause parsing problems
	// with the format 'MST ' (will expect a whitespace that isn't there after 4 char timezone). Most robust
	// solution is to remove the extra whitespace. Even though it will cause offsets after this point to not match
	// between the datestr and format string, it's not an issue at this point.
	if p.tzlen == 4 && p.tzi+4 < len(p.format) && p.format[p.tzi+3] == ' ' && p.format[p.tzi+4] != ' ' {
		p.removeRangeFromFormat(p.tzi+3, 1)
	}

	// If we have a full month name, update the format string to use it (can change length of format string)
	if len(p.fullMonth) > 0 {
		p.setFullMonth(p.fullMonth)
	}

	if p.retryAmbiguousDateWithSwap && p.ambiguousMD && p.ambiguousRetryable {
		// month out of range signifies that a day/month swap is the correct solution to an ambiguous date
		// this is because it means that a day is being interpreted as a month and overflowing the valid value for that
		// by retrying in this case, we can fix a common situation with no assumptions
		defer func() {
			// if actual time parsing errors out with the following error, swap before we
			// get out of this function to reduce scope it needs to be applied on
			if err != nil && strings.Contains(err.Error(), "month out of range") {
				// simple optimized case where mm and dd can be swapped directly
				if p.molen == 2 && p.daylen == 2 && len(p.fullMonth) <= 0 && (p.tzi == 0 || (p.moi < p.tzi && p.dayi < p.tzi)) {
					// skipped bytes have already been removed, so compensate for that
					moi := p.moi - p.skip
					p.moi = p.dayi - p.skip
					p.dayi = moi
					if !p.setDay() || !p.setMonth() {
						err = p.unknownErr(p.datestr)
					} else {
						if p.loc == nil {
							t, err = time.Parse(bytesToString(p.format), p.datestr)
						} else {
							t, err = time.ParseInLocation(bytesToString(p.format), p.datestr, p.loc)
						}
					}
				} else {
					// create the option to reverse the preference
					preferMonthFirst := PreferMonthFirst(!p.preferMonthFirst)
					// turn off the retry to avoid endless recursion
					retryAmbiguousDateWithSwap := RetryAmbiguousDateWithSwap(false)
					modifiedOpts := append(originalOpts, preferMonthFirst, retryAmbiguousDateWithSwap)
					var newParser *parser
					newParser, err = parseTime(p.datestr, originalLoc, modifiedOpts...)
					defer putBackParser(newParser)
					if err == nil {
						t, err = newParser.parse(originalLoc, modifiedOpts...)
						// The caller might use the format and datestr, so copy that back to the original parser
						p.setEntireFormat(newParser.format)
						p.datestr = newParser.datestr
					}
				}
			}
		}()
	}

	if p.skip > len(p.format) {
		p.skip = len(p.format)
	}
	if p.skip > 0 {
		// copy and then re-slice to shorten to avoid losing the header of the pooled format string
		copy(p.format, p.format[p.skip:])
		p.format = p.format[:len(p.format)-p.skip]
		p.formatSetLen -= p.skip
		if p.formatSetLen < 0 {
			p.formatSetLen = 0
		}
		p.datestr = p.datestr[p.skip:]
	}

	if p.loc == nil {
		// gou.Debugf("parse layout=%q input=%q   \ntx, err := time.Parse(%q, %q)", string(p.format), p.datestr, string(p.format), p.datestr)
		return time.Parse(bytesToString(p.format), p.datestr)
	} else {
		//gou.Debugf("parse layout=%q input=%q   \ntx, err := time.ParseInLocation(%q, %q, %v)", string(p.format), p.datestr, string(p.format), p.datestr, p.loc)
		return time.ParseInLocation(bytesToString(p.format), p.datestr, p.loc)
	}
}
func isDay(alpha string) bool {
	_, ok := knownDays[alpha]
	return ok
}
func isMonthFull(alpha string) bool {
	_, ok := knownMonths[alpha]
	return ok
}

package dateparse

import (
	"fmt"
	u "github.com/araddon/gou"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	ST_START uint64 = 1
)

var _ = u.EMPTY

// Given an unknown date format, detect the type, parse, return time
func ParseAny(datestr string) (time.Time, error) {

	var state uint64 = 1

iterRunes:
	for i := 0; i < len(datestr); i++ {
		r, _ := utf8.DecodeRuneInString(datestr[i:])

		//u.Infof("r=%s st=%d ", string(r), state)
		switch state {
		case ST_START:
			if unicode.IsDigit(r) {
				state = state << 1 // 2
			} else if unicode.IsLetter(r) {
				state = state << 2 // 4
			}
		case 2: // starts digit  1 << 2
			if unicode.IsDigit(r) {
				continue
			}
			switch r {
			case ' ':
				state = state << 3
			case ',':
				state = state << 4
			case '-':
				state = state << 5
			case ':':
				state = state << 6
			case '/':
				state = state << 7
			}
		case 64: // starts digit then dash 02-   1 << 2  << 5
			// 2006-01-02T15:04:05Z07:00
			// 2006-01-02T15:04:05.999999999Z07:00
			// 2014-04-26 17:24:37.3186369
			// 2016-03-14 00:00:00.000
			// 2014-05-11 08:20:13,787
			// 2006-01-02
			// 2014-04-26 05:24:37 PM
			switch {
			case r == ' ':
				state = state + 1
			case r == 'T':
				state = state + 3
			}
		case 65: // starts digit then dash 02- then whitespace   1 << 2  << 5 + 1
			// 2014-04-26 17:24:37.3186369
			// 2016-03-14 00:00:00.000
			// 2014-05-11 08:20:13,787
			// 2014-04-26 05:24:37 PM
			switch r {
			case 'A', 'P':
				if len(datestr) == len("2014-04-26 03:24:37 PM") {
					if t, err := time.Parse("2006-01-02 03:04:05 PM", datestr); err == nil {
						return t, nil
					} else {
						u.Error(err)
					}
				}
			case ',':
				if len(datestr) == len("2014-05-11 08:20:13,787") {
					if t, err := time.Parse("2006-01-02 03:04:05,999", datestr); err == nil {
						return t, nil
					} else {
						u.Error(err)
					}
				}
			}
		case 67: // starts digit then dash 02-  then T   1 << 2  << 5 + 3
			// 2006-01-02T15:04:05Z07:00
			// 2006-01-02T15:04:05.999999999Z07:00
			if len(datestr) == len("2006-01-02T15:04:05Z07:00") {
				if t, err := time.Parse("2006-01-02T15:04:05Z07:00", datestr); err == nil {
					return t, nil
				} else {
					u.Error(err)
				}
			} else if len(datestr) == len("2006-01-02T15:04:05.999999999Z07:00") {
				if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", datestr); err == nil {
					return t, nil
				} else {
					u.Error(err)
				}
			}
		case 256: // starts digit then slash 02/
			// 03/19/2012 10:11:59
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 3/1/2014
			// 10/13/2014
			// 01/02/2006
			if unicode.IsDigit(r) || r == '/' {
				continue
			}
			switch r {
			case ' ':
				state = state << 3
			}
		case 2048: // starts digit then slash 02/ more digits/slashes then whitespace
			// 03/19/2012 10:11:59
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			switch r {
			case ':':
				state = state << 8
			}
		case 524288: // starts digit then slash 02/ more digits/slashes then whitespace
			// 03/19/2012 10:11:59
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			switch r {
			case ':':
				state = state << 9
			}
		case 4: // starts alpha   1 << 2
			if unicode.IsLetter(r) {
				continue
			}
			switch {
			case r == ' ':
				state = state << 3
			case r == ',':
				state = state << 4
			case unicode.IsDigit(r):
				state = state << 5
			}
		case 32: // Starts alpha then whitespace   1 << 2  << 3
			switch {
			case r == ' ':
				state = state << 6
			case r == ',':
				state = state << 7
			case unicode.IsDigit(r):
				state = state << 8
			case unicode.IsLetter(r):
				state = state << 9
			}
		case 8192: // Starts Alpha then whitespace then digit  1 << 2  << 8
			// May 8, 2009 5:57:51 PM
			if t, err := time.Parse("Jan 2, 2006 3:04:05 PM", datestr); err == nil {
				return t, nil
			} else {
				u.Error(err)
			}
		default:
			//u.Infof("no case for: %d", state)
			break iterRunes
		}
	}

	switch state {
	case 2:
		// unixy timestamps ish
		if len(datestr) >= len("13980450781991351") {
			if nanoSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				return time.Unix(0, nanoSecs), nil
			} else {
				u.Error(err)
			}
		} else if len(datestr) >= len("13980450781991") {
			if microSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				return time.Unix(0, microSecs*1000), nil
			} else {
				u.Error(err)
			}
		} else {
			if secs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				return time.Unix(secs, 0), nil
			} else {
				u.Error(err)
			}
		}
	case 64: // starts digit then dash 02-    1 << 2  << 5
		// 2006-01-02
		if len(datestr) == len("2014-04-26") {
			if t, err := time.Parse("2006-01-02", datestr); err == nil {
				return t, nil
			} else {
				u.Error(err)
			}
		}
	case 65: // starts digit then dash 02-  then whitespace   1 << 2  << 5 + 3
		// 2014-04-26 17:24:37.3186369
		// 2016-03-14 00:00:00.000
		if len(datestr) == len("2014-04-26 05:24:37.3186369") {
			if t, err := time.Parse("2006-01-02 15:04:05.0000000", datestr); err == nil {
				return t, nil
			} else {
				u.Error(err)
			}
		} else if len(datestr) == len("2014-04-26 05:24:37.000") {
			if t, err := time.Parse("2006-01-02 15:04:05.000", datestr); err == nil {
				return t, nil
			} else {
				u.Error(err)
			}
		}
	case 256: // starts digit then slash 02/
		// 3/1/2014
		// 10/13/2014
		// 01/02/2006

		if len(datestr) == len("01/02/2006") {
			if t, err := time.Parse("01/02/2006", datestr); err == nil {
				return t, nil
			} else {
				u.Error(err)
			}
		} else {
			if t, err := time.Parse("1/2/2006", datestr); err == nil {
				return t, nil
			} else {
				u.Error(err)
			}
		}

	case 524288: // starts digit then slash 02/ more digits/slashes then whitespace
		// 4/8/2014 22:05
		if len(datestr) == len("01/02/2006 15:04") {
			if t, err := time.Parse("01/02/2006 15:04", datestr); err == nil {
				return t, nil
			} else {
				u.Error(err)
			}
		} else {
			if t, err := time.Parse("1/2/2006 15:04", datestr); err == nil {
				return t, nil
			} else {
				u.Error(err)
			}
		}
	case 268435456: // starts digit then slash 02/ more digits/slashes then whitespace double colons
		// 03/19/2012 10:11:59
		// 3/1/2012 10:11:59

		if len(datestr) == len("01/02/2006 15:04:05") {
			if t, err := time.Parse("01/02/2006 15:04:05", datestr); err == nil {
				return t, nil
			} else {
				u.Error(err)
			}
		} else {
			if t, err := time.Parse("1/2/2006 15:04:05", datestr); err == nil {
				return t, nil
			} else {
				u.Error(err)
			}
		}
	default:
		u.Infof("no case for: %d", state)
	}

	return time.Now(), fmt.Errorf("Could not find date format for %s", datestr)
}

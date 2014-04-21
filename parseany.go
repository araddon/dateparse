package dateparse

import (
	"fmt"
	u "github.com/araddon/gou"
	"time"
	"unicode"
	//"unicode/utf8"
)

var _ = u.EMPTY

const (
	S_START      = 0
	S_NUMERIC    = 1
	S_COMMA      = 2
	S_SEMICOLON  = 3
	S_COLON      = 4
	S_AMPM       = 5 // AM or PM after numeric
	S_ALPHA      = 6
	S_DASH       = 7 // -
	s_WHITESPACE = 8
)

/*
	features = []uint8{

	}


*/
type Features []TimeFeature

func (f Features) Has(ask TimeFeature) bool {
	return f[ask] == ask
}

func (f Features) ParseString() string {
	return ""
}

type TimeFeature uint8

const (
	ALL_NUMERIC    TimeFeature = 0
	STARTS_ALPHA   TimeFeature = 1
	STARTS_NUMERIC TimeFeature = 2
	HAS_NUMERIC    TimeFeature = 3
	HAS_ALPHA      TimeFeature = 4
	HAS_WHITESPACE TimeFeature = 5
	HAS_COMMA      TimeFeature = 6
	HAS_SLASH      TimeFeature = 7
	HAS_DASH       TimeFeature = 8
	HAS_Z          TimeFeature = 9
	HAS_T          TimeFeature = 10
	HAS_COLON      TimeFeature = 11
	HAS_AMPM       TimeFeature = 12
)

func parseFeatures(datestr string) Features {
	features := make(Features, 20)
	// totalLen := len(datestr)
	// switch {
	// case totalLen < 4:
	// 	u.Debug("len < 4")
	// }

	// if unicode.IsLetter(datestr[0]) {
	// 	features = append(FEAUTURES, STARTS_ALPHA)
	// }

	var lexeme string
	state := S_START
	//prevState := S_START
	next := int32(datestr[0])
	//prev := int32(datestr[0])
	for i := 0; i < len(datestr); i++ {
		char := int32(datestr[i])
		if i+1 < len(datestr) {
			next = int32(datestr[i+1])
			//u.Debugf("set next: %s", string(next))
		}
		switch char {
		case ' ', '\n', '\t':
			features[HAS_WHITESPACE] = HAS_WHITESPACE
		case ',':
			features[HAS_COMMA] = HAS_COMMA
		case '-':
			features[HAS_DASH] = HAS_DASH
		case ':':
			features[HAS_COLON] = HAS_COLON
		case '/':
			features[HAS_SLASH] = HAS_SLASH
		case 'Z':
			features[HAS_Z] = HAS_Z
		case 'T':
			features[HAS_T] = HAS_T
		case 'A', 'P':
			if next == 'M' {
				u.Info("Found feature AMPM")
				features[HAS_AMPM] = HAS_AMPM
			}
		}
		switch state {
		case S_START:
			lexeme = lexeme + string(char)
			if unicode.IsLetter(char) {
				state = S_ALPHA
				features[STARTS_ALPHA] = STARTS_ALPHA
				features[HAS_ALPHA] = HAS_ALPHA
			} else if unicode.IsNumber(char) {
				state = S_NUMERIC
				features[STARTS_NUMERIC] = STARTS_NUMERIC
				features[HAS_NUMERIC] = HAS_NUMERIC
			} else if char == ' ' {
				//u.Info("is whitespace")
				state = s_WHITESPACE
			} else {
				u.Error("unrecognized input? ", char, " ", string(char))
			}
		case S_ALPHA:
			if unicode.IsLetter(char) {
				features[STARTS_ALPHA] = STARTS_ALPHA
				features[HAS_ALPHA] = HAS_ALPHA
			} else if unicode.IsNumber(char) {
				state = S_NUMERIC
				features[STARTS_NUMERIC] = STARTS_NUMERIC
				features[HAS_NUMERIC] = HAS_NUMERIC
			} else if char == ' ' {
				//u.Info("is whitespace")
				state = s_WHITESPACE
			} else {
				u.Error("unrecognized input? ", char, " ", string(char))
			}
			lexeme = lexeme + string(char)
		case S_NUMERIC:
			if unicode.IsLetter(char) {
				features[HAS_ALPHA] = HAS_ALPHA
			} else if unicode.IsNumber(char) {
				features[HAS_NUMERIC] = HAS_NUMERIC
			} else if char == ' ' {
				u.Info("is whitespace")
				state = s_WHITESPACE
			} else {
				u.Error("unrecognized input? ", char, " ", string(char))
			}
			lexeme = lexeme + string(char)
		}
		//prev = char
	}

	return features
}

// Given an unknown date format, detect the type, parse, return time
func ParseAny(datestr string) (time.Time, error) {
	f := parseFeatures(datestr)
	switch {
	case f.Has(HAS_DASH) && !f.Has(HAS_SLASH):
		switch {
		case f.Has(HAS_WHITESPACE) && f.Has(HAS_COLON):
			//2006-01-02 15:04:05.000
		case f.Has(HAS_WHITESPACE) && f.Has(HAS_COLON):
			//2006-01-02
			//2006-01-02
		}
	case f.Has(HAS_SLASH):
		switch {
		case f.Has(HAS_WHITESPACE) && f.Has(HAS_COLON):
			// 03/03/2012 10:11:59
			// 3/1/2012 10:11:59
			u.Debugf("trying format:  3/1/2012 10:11:59 ")
			//May 8, 2009 5:57:51 PM      2006-01-02 15:04:05.000
			if t, err := time.Parse("01/02/2006 15:04:05", datestr); err == nil {
				return t, nil
			} else {
				if t, err := time.Parse("1/2/2006 15:04:05", datestr); err == nil {
					return t, nil
				} else {
					u.Error(err)
				}
			}
		case !f.Has(HAS_SLASH):
			// 3/1/2014
			// 10/13/2014
			// 01/02/2006
		}
	case f.Has(HAS_ALPHA) && f.Has(HAS_COMMA):
		switch {
		case f.Has(HAS_AMPM):
			u.Debugf("trying format:  2006-01-02 15:04:05 PM ")
			//May 8, 2009 5:57:51 PM      2006-01-02 15:04:05.000
			if t, err := time.Parse("Jan 2, 2006 3:04:05 PM", datestr); err == nil {
				return t, nil
			} else {
				u.Error(err)
			}
		}
	default:
		u.Errorf("Could not find format: %s", datestr)
	}
	return time.Now(), fmt.Errorf("Not found: %s", datestr)
}

// Given an unknown date format, detect the type and return
// the
func FindParseFormat(datestr string) (string, error) {

	return "", fmt.Errorf("Not recognized: %s", datestr)
}

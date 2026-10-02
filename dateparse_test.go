package dateparse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseStrictNamedMonth(t *testing.T) {
	// Issue #152: Parsing failed for 31-Oct-22 with ambiguous mm/dd error
	parsed, err := ParseStrict("31-Oct-22")
	assert.NoError(t, err)
	expected := time.Date(2022, time.October, 31, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, parsed.In(time.UTC))

	parsed, err = ParseStrict("1-Oct-22")
	assert.NoError(t, err)
	expected = time.Date(2022, time.October, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, parsed.In(time.UTC))

	parsed, err = ParseStrict("31-Oct-22 15:04:05")
	assert.NoError(t, err)
	expected = time.Date(2022, time.October, 31, 15, 4, 5, 0, time.UTC)
	assert.Equal(t, expected, parsed.In(time.UTC))
}

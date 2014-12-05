Go Date Parser 
---------------------------

Parse Any date format without knowing format in advance.  Uses
a Scan/Lex based approach to minimize shotgun based parse attempts.
See bench_test.go for performance comparison.




```go

func TestParse(t *testing.T) {

	ts, _ := ParseAny("May 8, 2009 5:57:51 PM")
	assert.T(t, "2009-05-08 17:57:51 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, _ = ParseAny("03/19/2012 10:11:59")
	assert.T(t, "2012-03-19 10:11:59 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, _ = ParseAny("3/31/2014")
	assert.T(t, "2014-03-31 00:00:00 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, _ = ParseAny("03/31/2014")
	assert.T(t, "2014-03-31 00:00:00 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, _ = ParseAny("4/8/2014 22:05")
	assert.T(t, "2014-04-08 22:05:00 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, _ = ParseAny("04/08/2014 22:05")
	assert.T(t, "2014-04-08 22:05:00 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, _ = ParseAny("1332151919")
	assert.T(t, "2012-03-19 10:11:59 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	assert.T(t, "2009-08-13 05:15:09 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, _ = ParseAny("2014-04-26 17:24:37.3186369")
	assert.T(t, "2014-04-26 17:24:37.3186369 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, _ = ParseAny("2014-04-26 17:24:37.123")
	assert.T(t, "2014-04-26 17:24:37.123 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, _ = ParseAny("2014-04-26 05:24:37 PM")
	assert.T(t, "2014-04-26 17:24:37 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, _ = ParseAny("2014-04-26")
	assert.T(t, "2014-04-26 00:00:00 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, _ = ParseAny("2014-05-11 08:20:13,787")
	assert.T(t, "2014-05-11 08:20:13.787 +0000 UTC" == fmt.Sprintf("%v", ts.In(time.UTC)))
}
```
Go Date Parser 
---------------------------

Parse many date strings without knowing format in advance. Validates comprehensively to avoid false positives.  Uses a scanner to read bytes with a state machine to find format.  Much faster than shotgun based parse methods.  See [bench_test.go](https://github.com/araddon/dateparse/blob/master/bench_test.go) for performance comparison. See the critical note below about timezones.


[![Code Coverage](https://codecov.io/gh/araddon/dateparse/branch/master/graph/badge.svg)](https://codecov.io/gh/araddon/dateparse)
[![GoDoc](https://godoc.org/github.com/araddon/dateparse?status.svg)](http://godoc.org/github.com/araddon/dateparse)
[![Build Status](https://travis-ci.org/araddon/dateparse.svg?branch=master)](https://travis-ci.org/araddon/dateparse)
[![Go ReportCard](https://goreportcard.com/badge/araddon/dateparse)](https://goreportcard.com/report/araddon/dateparse)

**MM/DD/YYYY VS DD/MM/YYYY** Right now this uses mm/dd/yyyy WHEN ambiguous if this is not desired behavior, use `ParseStrict` which will fail on ambiguous date strings. This can be adjusted using the `PreferMonthFirst` parser option. Some ambiguous formats can fail (e.g., trying to parse 31/03/2023 as the default month-first format `MM/DD/YYYY`), but can be automatically retried with `RetryAmbiguousDateWithSwap`.

```go

// Normal parse.  Equivalent Timezone rules as time.Parse()
t, err := dateparse.ParseAny("3/1/2014")

// Parse Strict, error on ambigous mm/dd vs dd/mm dates
t, err := dateparse.ParseStrict("3/1/2014")
> returns error 

// Return a string that represents the layout to parse the given date-time.
// For certain highly complex date formats, ParseFormat may not be accurate,
// even if ParseAny is able to correctly parse it (e.g., anything that starts
// with a weekday).
layout, err := dateparse.ParseFormat("May 8, 2009 5:57:51 PM")
> "Jan 2, 2006 3:04:05 PM"

```

Performance Considerations
----------------------------------

Internally a memory pool is used to minimize allocation overhead. If you could
be frequently parsing text that does not match any format, consider turning on
the the `SimpleErrorMessages` option. This will make error messages have no
contextual details, but will reduce allocation overhead 13x and will be 4x
faster (most of the time is spent in generating a complex error message if the
option is off (default)).

Timezone Considerations
----------------------------------

**Timezones** The location your server is configured affects the results! See example or https://play.golang.org/p/IDHRalIyXh and last paragraph here https://golang.org/pkg/time/#Parse.

Important points to understand:
* If you are parsing a date string that does *not* reference a timezone, if you use `Parse` it will assume UTC, or for `ParseIn` it will use the specified location.
* If you are parsing a date string that *does* reference a timezone and *does* specify an explicit offset (e.g., `2012-08-03 13:31:59 -0600 MST`), then it will return a time object with a location that represents a fixed timezone that has the given offset and name (it will not validate that the timezone abbreviation specified in the date string is a potential valid match for the given offset).
  * This can lead to some potentially unexpected results, for example consider the date string `2012-08-03 18:31:59.000+00:00 PST` -- this string has an explicit offset of `+00:00` (UTC), and so the returned time will have a location with a zero offset (18:31:59.000 UTC) even though the name of the fixed time zone associated with the returned time is `PST`. Essentially, it will always prioritize an explicit offset as accurate over an explicit 
* If you are parsing a date string that *does* reference a timezone but *without* an explicit offset (e.g., `2012-08-03 14:32:59 MST`), then it will only recognize and map the timezone name and add an offset if you are using `ParseIn` and specify a location that knows about the given time zone abbreviation (e.g., in this example, you would need to pass the `America/Denver` location and it will recognize the `MST` and `MDT` time zone names)
  * If a time zone abbreviation is recognized based on the passed location, then it will use the appropriate offset, and make any appropriate adjustment for daylight saving time (e.g., in the above example, the parsed time would actually contain a zone name of `MDT` because the date is within the range when daylight savings time is active).
  * If a time zone abbreviation is *not* recognized for the passed location, then it will create a fake time zone with a *zero* offset but with the specified name. This requires further processing if you are trying to actually get the correct absolute time in the UTC time zone.
  * If you receive a parsed time that has a zero offset but a non-UTC timezone name, then you should use a method to map the (sometimes ambiguous) timezone name (e.g., `"EEG"`) into a location name (e.g., `"Africa/Cairo"` or `"Europe/Bucharest"`), and then reconstruct a new time object with the same date/time/nanosecond but with the properly mapped location. (Do not use the `time.In` method to convert it to the new location, as this will treat the original time as if it was in UTC with a zero offset -- you need to reconstruct the time as if it was constructed with the proper location in the first place.)

cli tool for testing dateformats
----------------------------------

[Date Parse CLI](https://github.com/araddon/dateparse/blob/master/dateparse)


Extended example
-------------------

https://github.com/araddon/dateparse/blob/master/example/main.go

```go
package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/araddon/dateparse"
	"github.com/scylladb/termtables"
)

var examples = []string{
	// mon day year (time)
	"May 8, 2009 5:57:51 PM",
	"oct 7, 1970",
	"oct 7, '70",
	"oct. 7, 1970",
	"oct. 7, 70",
	"October 7, 1970",
	"October 7th, 1970",
	"Sept. 7, 1970 11:15:26pm",
	"Sep 7 2009 11:15:26.123 PM PST",
	"September 3rd, 2009 11:15:26.123456789pm",
	"September 17 2012 10:09am",
	"September 17, 2012, 10:10:09",
	"Sep 17, 2012 at 10:02am (EST)",
	// (PST-08 will have an offset of -0800, and a zone name of "PST")
	"September 17, 2012 at 10:09am PST-08",
	// (UTC-0700 has the same offset as -0700, and the returned zone name will be empty)
	"September 17 2012 5:00pm UTC-0700",
	"September 17 2012 5:00pm GMT-0700",
	// (weekday) day mon year (time)
	"7 oct 70",
	"7 Oct 1970",
	"7 September 1970 23:15",
	"7 September 1970 11:15:26pm",
	"03 February 2013",
	"12 Feb 2006, 19:17",
	"12 Feb 2006 19:17",
	"14 May 2019 19:11:40.164",
	"4th Sep 2012",
	"1st February 2018 13:58:24",
	"Mon, 02 Jan 2006 15:04:05 MST", // RFC1123
	"Mon, 02 Jan 2006 15:04:05 -0700",
	"Tue, 11 Jul 2017 16:28:13 +0200 (CEST)",
	"Mon 30 Sep 2018 09:09:09 PM UTC",
	"Sun, 07 Jun 2020 00:00:00 +0100",
	"Wed,  8 Feb 2023 19:00:46 +1100 (AEDT)",
	// ANSIC and UnixDate - weekday month day time year
	"Mon Jan  2 15:04:05 2006",
	"Mon Jan  2 15:04:05 MST 2006",
	"Monday Jan 02 15:04:05 -0700 2006",
	"Mon Jan 2 15:04:05.103786 2006",
	// RubyDate - weekday month day time offset year
	"Mon Jan 02 15:04:05 -0700 2006",
	// ANSIC_GLIBC - weekday day month year time
	"Mon 02 Jan 2006 03:04:05 PM UTC",
	"Monday 02 Jan 2006 03:04:05 PM MST",
	// weekday month day time timezone-offset year
	"Mon Aug 10 15:44:11 UTC+0000 2015",
	// git log default date format
	"Thu Apr 7 15:13:13 2005 -0700",
	// variants of git log default date format
	"Thu Apr 7 15:13:13 2005 -07:00",
	"Thu Apr 7 15:13:13 2005 -07:00 PST",
	"Thu Apr 7 15:13:13 2005 -07:00 PST (Pacific Standard Time)",
	"Thu Apr 7 15:13:13 -0700 2005",
	"Thu Apr 7 15:13:13 -07:00 2005",
	"Thu Apr 7 15:13:13 -0700 PST 2005",
	"Thu Apr 7 15:13:13 -07:00 PST 2005",
	"Thu Apr 7 15:13:13 PST 2005",
	// Variants of the above with a (full time zone description)
	"Fri Jul 3 2015 06:04:07 PST-0700 (Pacific Daylight Time)",
	"Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)",
	"Sun, 3 Jan 2021 00:12:23 +0800 (GMT+08:00)",
	// year month day
	"2013 May 2",
	"2013 May 02 11:37:55",
	// dd/Mon/year  alpha Months
	"06/Jan/2008 15:04:05 -0700",
	"06/January/2008 15:04:05 -0700",
	"06/Jan/2008:15:04:05 -0700", // ngnix-log
	"06/January/2008:08:11:17 -0700",
	// mm/dd/year (see also PreferMonthFirst and RetryAmbiguousDateWithSwap options)
	"3/31/2014",
	"03/31/2014",
	"08/21/71",
	"8/1/71",
	"4/8/2014 22:05",
	"04/08/2014 22:05",
	"04/08/2014, 22:05",
	"4/8/14 22:05",
	"04/2/2014 03:00:51",
	"8/8/1965 1:00 PM",
	"8/8/1965 01:00 PM",
	"8/8/1965 12:00 AM",
	"8/8/1965 12:00:00AM",
	"8/8/1965 01:00:01 PM",
	"8/8/1965 01:00:01PM -0700",
	"8/8/1965 13:00:01 -0700 PST",
	"8/8/1965 01:00:01 PM -0700 PST",
	"8/8/1965 01:00:01 PM -07:00 PST (Pacific Standard Time)",
	"4/02/2014 03:00:51",
	"03/19/2012 10:11:59",
	"03/19/2012 10:11:59.3186369",
	// mon/dd/year
	"Oct/ 7/1970",
	"Oct/03/1970 22:33:44",
	"February/03/1970 11:33:44.555 PM PST",
	// yyyy/mm/dd
	"2014/3/31",
	"2014/03/31",
	"2014/4/8 22:05",
	"2014/04/08 22:05",
	"2014/04/2 03:00:51",
	"2014/4/02 03:00:51",
	"2012/03/19 10:11:59",
	"2012/03/19 10:11:59.3186369",
	// weekday, day-mon-yy time
	"Fri, 03-Jul-15 08:08:08 CEST",
	"Monday, 02-Jan-06 15:04:05 MST", // RFC850
	"Monday, 02 Jan 2006 15:04:05 -0600",
	"02-Jan-06 15:04:05 MST",
	// RFC3339 - yyyy-mm-ddThh
	"2006-01-02T15:04:05+0000",
	"2009-08-12T22:15:09-07:00",
	"2009-08-12T22:15:09",
	"2009-08-12T22:15:09.988",
	"2009-08-12T22:15:09Z",
	"2009-08-12T22:15:09.52Z",
	"2017-07-19T03:21:51:897+0100",
	"2019-05-29T08:41-04", // no seconds, 2 digit TZ offset
	// yyyy-mm-dd hh:mm:ss
	"2014-04-26 17:24:37.3186369",
	"2012-08-03 18:31:59.257000000",
	"2014-04-26 17:24:37.123",
	"2014-04-01 12:01am",
	"2014-04-01 12:01:59.765 AM",
	"2014-04-01 12:01:59,765",
	"2014-04-01 22:43",
	"2014-04-01 22:43:22",
	"2014-12-16 06:20:00 UTC",
	"2014-12-16 06:20:00 GMT",
	"2014-04-26 05:24:37 PM",
	"2014-04-26 13:13:43 +0800",
	"2014-04-26 13:13:43 +0800 +08",
	"2014-04-26 13:13:44 +09:00",
	"2012-08-03 18:31:59.257000000 +0000 UTC",
	"2015-09-30 18:48:56.35272715 +0000 UTC",
	"2015-02-18 00:12:00 +0000 GMT", // golang native format
	"2015-02-18 00:12:00 +0000 UTC",
	"2015-02-08 03:02:00 +0300 MSK m=+0.000000001",
	"2015-02-08 03:02:00.001 +0300 MSK m=+0.000000001",
	"2017-07-19 03:21:51+00:00",
	"2017-04-03 22:32:14.322 CET",
	"2017-04-03 22:32:14,322 CET",
	"2017-04-03 22:32:14:322 CET",
	"2018-09-30 08:09:13.123PM PMDT", // PMDT time zone
	"2018-09-30 08:09:13.123 am AMT", // AMT time zone
	"2014-04-26",
	"2014-04",
	"2014",
	// yyyy-mm-dd(offset)
	"2020-07-20+08:00",
	"2020-07-20+0800",
	// year-mon-dd
	"2013-Feb-03",
	"2013-February-03 09:07:08.123",
	// dd-mon-year
	"03-Feb-13",
	"03-Feb-2013",
	"07-Feb-2004 09:07:07 +0200",
	"07-February-2004 09:07:07 +0200",
	// dd-mm-year (this format (common in Europe) always puts the day first, regardless of PreferMonthFirst)
	"28-02-02",
	"28-02-02 15:16:17",
	"28-02-2002",
	"28-02-2002 15:16:17",
	// mm.dd.yy (see also PreferMonthFirst and RetryAmbiguousDateWithSwap options)
	"3.31.2014",
	"03.31.14",
	"03.31.2014",
	"03.31.2014 10:11:59 MST",
	"03.31.2014 10:11:59.3186369Z",
	// year.mm.dd
	"2014.03",
	"2014.03.30",
	"2014.03.30 08:33pm",
	"2014.03.30T08:33:44.555 PM -0700 MST",
	"2014.03.30-0600",
	// yyyy:mm:dd
	"2014:3:31",
	"2014:03:31",
	"2014:4:8 22:05",
	"2014:04:08 22:05",
	"2014:04:2 03:00:51",
	"2014:4:02 03:00:51",
	"2012:03:19 10:11:59",
	"2012:03:19 10:11:59.3186369",
	// mm:dd:yyyy (see also PreferMonthFirst and RetryAmbiguousDateWithSwap options)
	"08:03:2012",
	"08:04:2012 18:31:59+00:00",
	// yyyymmdd and similar
	"20140601",
	"20140722105203",
	"20140722105203.364",
	// Chinese
	"2014年4月25日",
	"2014年04月08日",
	"2014年04月08日 19:17:22 -0700",
	// RabbitMQ log format
	"8-Mar-2018::14:09:27",
	"08-03-2018::02:09:29 PM",
	// yymmdd hh:mm:yy mysql log
	// 080313 05:21:55 mysqld started
	"171113 14:14:20",
	"190910 11:51:49",
	// unix seconds, ms, micro, nano
	"1332151919",
	"1384216367189",
	"1384216367111222",
	"1384216367111222333",
}

var (
	timezone = ""
)

func main() {
	flag.StringVar(&timezone, "timezone", "UTC", "Timezone aka `America/Los_Angeles` formatted time-zone")
	flag.Parse()

	if timezone != "" {
		// NOTE:  This is very, very important to understand
		// time-parsing in go
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			panic(err.Error())
		}
		time.Local = loc
	}

	table := termtables.CreateTable()

	table.AddHeaders("Input", "Parsed, and Output as %v")
	for _, dateExample := range examples {
		t, err := dateparse.ParseLocal(dateExample)
		if err != nil {
			panic(err.Error())
		}
		table.AddRow(dateExample, fmt.Sprintf("%v", t))
	}
	fmt.Println(table.Render())
}

/*
+------------------------------------------------------------+-----------------------------------------+
| Input                                                      | Parsed, and Output as %v                |
+------------------------------------------------------------+-----------------------------------------+
| May 8, 2009 5:57:51 PM                                     | 2009-05-08 17:57:51 +0000 UTC           |
| oct 7, 1970                                                | 1970-10-07 00:00:00 +0000 UTC           |
| oct 7, '70                                                 | 1970-10-07 00:00:00 +0000 UTC           |
| oct. 7, 1970                                               | 1970-10-07 00:00:00 +0000 UTC           |
| oct. 7, 70                                                 | 1970-10-07 00:00:00 +0000 UTC           |
| October 7, 1970                                            | 1970-10-07 00:00:00 +0000 UTC           |
| October 7th, 1970                                          | 1970-10-07 00:00:00 +0000 UTC           |
| Sept. 7, 1970 11:15:26pm                                   | 1970-09-07 23:15:26 +0000 UTC           |
| Sep 7 2009 11:15:26.123 PM PST                             | 2009-09-07 23:15:26.123 +0000 PST       |
| September 3rd, 2009 11:15:26.123456789pm                   | 2009-09-03 23:15:26.123456789 +0000 UTC |
| September 17 2012 10:09am                                  | 2012-09-17 10:09:00 +0000 UTC           |
| September 17, 2012, 10:10:09                               | 2012-09-17 10:10:09 +0000 UTC           |
| Sep 17, 2012 at 10:02am (EST)                              | 2012-09-17 10:02:00 +0000 EST           |
| September 17, 2012 at 10:09am PST-08                       | 2012-09-17 10:09:00 -0800 PST           |
| September 17 2012 5:00pm UTC-0700                          | 2012-09-17 17:00:00 -0700 -0700         |
| September 17 2012 5:00pm GMT-0700                          | 2012-09-17 17:00:00 -0700 -0700         |
| 7 oct 70                                                   | 1970-10-07 00:00:00 +0000 UTC           |
| 7 Oct 1970                                                 | 1970-10-07 00:00:00 +0000 UTC           |
| 7 September 1970 23:15                                     | 1970-09-07 23:15:00 +0000 UTC           |
| 7 September 1970 11:15:26pm                                | 1970-09-07 23:15:26 +0000 UTC           |
| 03 February 2013                                           | 2013-02-03 00:00:00 +0000 UTC           |
| 12 Feb 2006, 19:17                                         | 2006-02-12 19:17:00 +0000 UTC           |
| 12 Feb 2006 19:17                                          | 2006-02-12 19:17:00 +0000 UTC           |
| 14 May 2019 19:11:40.164                                   | 2019-05-14 19:11:40.164 +0000 UTC       |
| 4th Sep 2012                                               | 2012-09-04 00:00:00 +0000 UTC           |
| 1st February 2018 13:58:24                                 | 2018-02-01 13:58:24 +0000 UTC           |
| Mon, 02 Jan 2006 15:04:05 MST                              | 2006-01-02 15:04:05 +0000 MST           |
| Mon, 02 Jan 2006 15:04:05 -0700                            | 2006-01-02 15:04:05 -0700 -0700         |
| Tue, 11 Jul 2017 16:28:13 +0200 (CEST)                     | 2017-07-11 16:28:13 +0200 +0200         |
| Mon 30 Sep 2018 09:09:09 PM UTC                            | 2018-09-30 21:09:09 +0000 UTC           |
| Sun, 07 Jun 2020 00:00:00 +0100                            | 2020-06-07 00:00:00 +0100 +0100         |
| Wed,  8 Feb 2023 19:00:46 +1100 (AEDT)                     | 2023-02-08 19:00:46 +1100 +1100         |
| Mon Jan  2 15:04:05 2006                                   | 2006-01-02 15:04:05 +0000 UTC           |
| Mon Jan  2 15:04:05 MST 2006                               | 2006-01-02 15:04:05 +0000 MST           |
| Monday Jan 02 15:04:05 -0700 2006                          | 2006-01-02 15:04:05 -0700 -0700         |
| Mon Jan 2 15:04:05.103786 2006                             | 2006-01-02 15:04:05.103786 +0000 UTC    |
| Mon Jan 02 15:04:05 -0700 2006                             | 2006-01-02 15:04:05 -0700 -0700         |
| Mon 02 Jan 2006 03:04:05 PM UTC                            | 2006-01-02 15:04:05 +0000 UTC           |
| Monday 02 Jan 2006 03:04:05 PM MST                         | 2006-01-02 15:04:05 +0000 MST           |
| Mon Aug 10 15:44:11 UTC+0000 2015                          | 2015-08-10 15:44:11 +0000 UTC           |
| Thu Apr 7 15:13:13 2005 -0700                              | 2005-04-07 15:13:13 -0700 -0700         |
| Thu Apr 7 15:13:13 2005 -07:00                             | 2005-04-07 15:13:13 -0700 -0700         |
| Thu Apr 7 15:13:13 2005 -07:00 PST                         | 2005-04-07 15:13:13 -0700 PST           |
| Thu Apr 7 15:13:13 2005 -07:00 PST (Pacific Standard Time) | 2005-04-07 15:13:13 -0700 PST           |
| Thu Apr 7 15:13:13 -0700 2005                              | 2005-04-07 15:13:13 -0700 -0700         |
| Thu Apr 7 15:13:13 -07:00 2005                             | 2005-04-07 15:13:13 -0700 -0700         |
| Thu Apr 7 15:13:13 -0700 PST 2005                          | 2005-04-07 15:13:13 -0700 PST           |
| Thu Apr 7 15:13:13 -07:00 PST 2005                         | 2005-04-07 15:13:13 -0700 PST           |
| Thu Apr 7 15:13:13 PST 2005                                | 2005-04-07 15:13:13 +0000 PST           |
| Fri Jul 3 2015 06:04:07 PST-0700 (Pacific Daylight Time)   | 2015-07-03 06:04:07 -0700 PST           |
| Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)      | 2015-07-03 18:04:07 +0100 +0100         |
| Sun, 3 Jan 2021 00:12:23 +0800 (GMT+08:00)                 | 2021-01-03 00:12:23 +0800 +0800         |
| 2013 May 2                                                 | 2013-05-02 00:00:00 +0000 UTC           |
| 2013 May 02 11:37:55                                       | 2013-05-02 11:37:55 +0000 UTC           |
| 06/Jan/2008 15:04:05 -0700                                 | 2008-01-06 15:04:05 -0700 -0700         |
| 06/January/2008 15:04:05 -0700                             | 2008-01-06 15:04:05 -0700 -0700         |
| 06/Jan/2008:15:04:05 -0700                                 | 2008-01-06 15:04:05 -0700 -0700         |
| 06/January/2008:08:11:17 -0700                             | 2008-01-06 08:11:17 -0700 -0700         |
| 3/31/2014                                                  | 2014-03-31 00:00:00 +0000 UTC           |
| 03/31/2014                                                 | 2014-03-31 00:00:00 +0000 UTC           |
| 08/21/71                                                   | 1971-08-21 00:00:00 +0000 UTC           |
| 8/1/71                                                     | 1971-08-01 00:00:00 +0000 UTC           |
| 4/8/2014 22:05                                             | 2014-04-08 22:05:00 +0000 UTC           |
| 04/08/2014 22:05                                           | 2014-04-08 22:05:00 +0000 UTC           |
| 04/08/2014, 22:05                                          | 2014-04-08 22:05:00 +0000 UTC           |
| 4/8/14 22:05                                               | 2014-04-08 22:05:00 +0000 UTC           |
| 04/2/2014 03:00:51                                         | 2014-04-02 03:00:51 +0000 UTC           |
| 8/8/1965 1:00 PM                                           | 1965-08-08 13:00:00 +0000 UTC           |
| 8/8/1965 01:00 PM                                          | 1965-08-08 13:00:00 +0000 UTC           |
| 8/8/1965 12:00 AM                                          | 1965-08-08 00:00:00 +0000 UTC           |
| 8/8/1965 12:00:00AM                                        | 1965-08-08 00:00:00 +0000 UTC           |
| 8/8/1965 01:00:01 PM                                       | 1965-08-08 13:00:01 +0000 UTC           |
| 8/8/1965 01:00:01PM -0700                                  | 1965-08-08 13:00:01 -0700 -0700         |
| 8/8/1965 13:00:01 -0700 PST                                | 1965-08-08 13:00:01 -0700 PST           |
| 8/8/1965 01:00:01 PM -0700 PST                             | 1965-08-08 13:00:01 -0700 PST           |
| 8/8/1965 01:00:01 PM -07:00 PST (Pacific Standard Time)    | 1965-08-08 13:00:01 -0700 PST           |
| 4/02/2014 03:00:51                                         | 2014-04-02 03:00:51 +0000 UTC           |
| 03/19/2012 10:11:59                                        | 2012-03-19 10:11:59 +0000 UTC           |
| 03/19/2012 10:11:59.3186369                                | 2012-03-19 10:11:59.3186369 +0000 UTC   |
| Oct/ 7/1970                                                | 1970-10-07 00:00:00 +0000 UTC           |
| Oct/03/1970 22:33:44                                       | 1970-10-03 22:33:44 +0000 UTC           |
| February/03/1970 11:33:44.555 PM PST                       | 1970-02-03 23:33:44.555 +0000 PST       |
| 2014/3/31                                                  | 2014-03-31 00:00:00 +0000 UTC           |
| 2014/03/31                                                 | 2014-03-31 00:00:00 +0000 UTC           |
| 2014/4/8 22:05                                             | 2014-04-08 22:05:00 +0000 UTC           |
| 2014/04/08 22:05                                           | 2014-04-08 22:05:00 +0000 UTC           |
| 2014/04/2 03:00:51                                         | 2014-04-02 03:00:51 +0000 UTC           |
| 2014/4/02 03:00:51                                         | 2014-04-02 03:00:51 +0000 UTC           |
| 2012/03/19 10:11:59                                        | 2012-03-19 10:11:59 +0000 UTC           |
| 2012/03/19 10:11:59.3186369                                | 2012-03-19 10:11:59.3186369 +0000 UTC   |
| Fri, 03-Jul-15 08:08:08 CEST                               | 2015-07-03 08:08:08 +0000 CEST          |
| Monday, 02-Jan-06 15:04:05 MST                             | 2006-01-02 15:04:05 +0000 MST           |
| Monday, 02 Jan 2006 15:04:05 -0600                         | 2006-01-02 15:04:05 -0600 -0600         |
| 02-Jan-06 15:04:05 MST                                     | 2006-01-02 15:04:05 +0000 MST           |
| 2006-01-02T15:04:05+0000                                   | 2006-01-02 15:04:05 +0000 UTC           |
| 2009-08-12T22:15:09-07:00                                  | 2009-08-12 22:15:09 -0700 -0700         |
| 2009-08-12T22:15:09                                        | 2009-08-12 22:15:09 +0000 UTC           |
| 2009-08-12T22:15:09.988                                    | 2009-08-12 22:15:09.988 +0000 UTC       |
| 2009-08-12T22:15:09Z                                       | 2009-08-12 22:15:09 +0000 UTC           |
| 2009-08-12T22:15:09.52Z                                    | 2009-08-12 22:15:09.52 +0000 UTC        |
| 2017-07-19T03:21:51:897+0100                               | 2017-07-19 03:21:51.897 +0100 +0100     |
| 2019-05-29T08:41-04                                        | 2019-05-29 08:41:00 -0400 -0400         |
| 2014-04-26 17:24:37.3186369                                | 2014-04-26 17:24:37.3186369 +0000 UTC   |
| 2012-08-03 18:31:59.257000000                              | 2012-08-03 18:31:59.257 +0000 UTC       |
| 2014-04-26 17:24:37.123                                    | 2014-04-26 17:24:37.123 +0000 UTC       |
| 2014-04-01 12:01am                                         | 2014-04-01 00:01:00 +0000 UTC           |
| 2014-04-01 12:01:59.765 AM                                 | 2014-04-01 00:01:59.765 +0000 UTC       |
| 2014-04-01 12:01:59,765                                    | 2014-04-01 12:01:59.765 +0000 UTC       |
| 2014-04-01 22:43                                           | 2014-04-01 22:43:00 +0000 UTC           |
| 2014-04-01 22:43:22                                        | 2014-04-01 22:43:22 +0000 UTC           |
| 2014-12-16 06:20:00 UTC                                    | 2014-12-16 06:20:00 +0000 UTC           |
| 2014-12-16 06:20:00 GMT                                    | 2014-12-16 06:20:00 +0000 GMT           |
| 2014-04-26 05:24:37 PM                                     | 2014-04-26 17:24:37 +0000 UTC           |
| 2014-04-26 13:13:43 +0800                                  | 2014-04-26 13:13:43 +0800 +0800         |
| 2014-04-26 13:13:43 +0800 +08                              | 2014-04-26 13:13:43 +0800 +0800         |
| 2014-04-26 13:13:44 +09:00                                 | 2014-04-26 13:13:44 +0900 +0900         |
| 2012-08-03 18:31:59.257000000 +0000 UTC                    | 2012-08-03 18:31:59.257 +0000 UTC       |
| 2015-09-30 18:48:56.35272715 +0000 UTC                     | 2015-09-30 18:48:56.35272715 +0000 UTC  |
| 2015-02-18 00:12:00 +0000 GMT                              | 2015-02-18 00:12:00 +0000 GMT           |
| 2015-02-18 00:12:00 +0000 UTC                              | 2015-02-18 00:12:00 +0000 UTC           |
| 2015-02-08 03:02:00 +0300 MSK m=+0.000000001               | 2015-02-08 03:02:00 +0300 MSK           |
| 2015-02-08 03:02:00.001 +0300 MSK m=+0.000000001           | 2015-02-08 03:02:00.001 +0300 MSK       |
| 2017-07-19 03:21:51+00:00                                  | 2017-07-19 03:21:51 +0000 UTC           |
| 2017-04-03 22:32:14.322 CET                                | 2017-04-03 22:32:14.322 +0000 CET       |
| 2017-04-03 22:32:14,322 CET                                | 2017-04-03 22:32:14.322 +0000 CET       |
| 2017-04-03 22:32:14:322 CET                                | 2017-04-03 22:32:14.322 +0000 CET       |
| 2018-09-30 08:09:13.123PM PMDT                             | 2018-09-30 20:09:13.123 +0000 PMDT      |
| 2018-09-30 08:09:13.123 am AMT                             | 2018-09-30 08:09:13.123 +0000 AMT       |
| 2014-04-26                                                 | 2014-04-26 00:00:00 +0000 UTC           |
| 2014-04                                                    | 2014-04-01 00:00:00 +0000 UTC           |
| 2014                                                       | 2014-01-01 00:00:00 +0000 UTC           |
| 2020-07-20+08:00                                           | 2020-07-20 00:00:00 +0800 +0800         |
| 2020-07-20+0800                                            | 2020-07-20 00:00:00 +0800 +0800         |
| 2013-Feb-03                                                | 2013-02-03 00:00:00 +0000 UTC           |
| 2013-February-03 09:07:08.123                              | 2013-02-03 09:07:08.123 +0000 UTC       |
| 03-Feb-13                                                  | 2013-02-03 00:00:00 +0000 UTC           |
| 03-Feb-2013                                                | 2013-02-03 00:00:00 +0000 UTC           |
| 07-Feb-2004 09:07:07 +0200                                 | 2004-02-07 09:07:07 +0200 +0200         |
| 07-February-2004 09:07:07 +0200                            | 2004-02-07 09:07:07 +0200 +0200         |
| 28-02-02                                                   | 2002-02-28 00:00:00 +0000 UTC           |
| 28-02-02 15:16:17                                          | 2002-02-28 15:16:17 +0000 UTC           |
| 28-02-2002                                                 | 2002-02-28 00:00:00 +0000 UTC           |
| 28-02-2002 15:16:17                                        | 2002-02-28 15:16:17 +0000 UTC           |
| 3.31.2014                                                  | 2014-03-31 00:00:00 +0000 UTC           |
| 03.31.14                                                   | 2014-03-31 00:00:00 +0000 UTC           |
| 03.31.2014                                                 | 2014-03-31 00:00:00 +0000 UTC           |
| 03.31.2014 10:11:59 MST                                    | 2014-03-31 10:11:59 +0000 MST           |
| 03.31.2014 10:11:59.3186369Z                               | 2014-03-31 10:11:59.3186369 +0000 UTC   |
| 2014.03                                                    | 2014-03-01 00:00:00 +0000 UTC           |
| 2014.03.30                                                 | 2014-03-30 00:00:00 +0000 UTC           |
| 2014.03.30 08:33pm                                         | 2014-03-30 20:33:00 +0000 UTC           |
| 2014.03.30T08:33:44.555 PM -0700 MST                       | 2014-03-30 20:33:44.555 -0700 MST       |
| 2014.03.30-0600                                            | 2014-03-30 00:00:00 -0600 -0600         |
| 2014:3:31                                                  | 2014-03-31 00:00:00 +0000 UTC           |
| 2014:03:31                                                 | 2014-03-31 00:00:00 +0000 UTC           |
| 2014:4:8 22:05                                             | 2014-04-08 22:05:00 +0000 UTC           |
| 2014:04:08 22:05                                           | 2014-04-08 22:05:00 +0000 UTC           |
| 2014:04:2 03:00:51                                         | 2014-04-02 03:00:51 +0000 UTC           |
| 2014:4:02 03:00:51                                         | 2014-04-02 03:00:51 +0000 UTC           |
| 2012:03:19 10:11:59                                        | 2012-03-19 10:11:59 +0000 UTC           |
| 2012:03:19 10:11:59.3186369                                | 2012-03-19 10:11:59.3186369 +0000 UTC   |
| 08:03:2012                                                 | 2012-08-03 00:00:00 +0000 UTC           |
| 08:04:2012 18:31:59+00:00                                  | 2012-08-04 18:31:59 +0000 UTC           |
| 20140601                                                   | 2014-06-01 00:00:00 +0000 UTC           |
| 20140722105203                                             | 2014-07-22 10:52:03 +0000 UTC           |
| 20140722105203.364                                         | 2014-07-22 10:52:03.364 +0000 UTC       |
| 2014年4月25日                                              | 2014-04-25 00:00:00 +0000 UTC           |
| 2014年04月08日                                             | 2014-04-08 00:00:00 +0000 UTC           |
| 2014年04月08日 19:17:22 -0700                              | 2014-04-08 19:17:22 -0700 -0700         |
| 8-Mar-2018::14:09:27                                       | 2018-03-08 14:09:27 +0000 UTC           |
| 08-03-2018::02:09:29 PM                                    | 2018-03-08 14:09:29 +0000 UTC           |
| 171113 14:14:20                                            | 2017-11-13 14:14:20 +0000 UTC           |
| 190910 11:51:49                                            | 2019-09-10 11:51:49 +0000 UTC           |
| 1332151919                                                 | 2012-03-19 10:11:59 +0000 UTC           |
| 1384216367189                                              | 2013-11-12 00:32:47.189 +0000 UTC       |
| 1384216367111222                                           | 2013-11-12 00:32:47.111222 +0000 UTC    |
| 1384216367111222333                                        | 2013-11-12 00:32:47.111222333 +0000 UTC |
+------------------------------------------------------------+-----------------------------------------+
*/

```

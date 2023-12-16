Go Date Parser 
---------------------------

Parse many date strings without knowing format in advance.  Uses a scanner to read bytes and use a state machine to find format.  Much faster than shotgun based parse methods.  See [bench_test.go](https://github.com/araddon/dateparse/blob/master/bench_test.go) for performance comparison.


[![Code Coverage](https://codecov.io/gh/araddon/dateparse/branch/master/graph/badge.svg)](https://codecov.io/gh/araddon/dateparse)
[![GoDoc](https://godoc.org/github.com/araddon/dateparse?status.svg)](http://godoc.org/github.com/araddon/dateparse)
[![Build Status](https://travis-ci.org/araddon/dateparse.svg?branch=master)](https://travis-ci.org/araddon/dateparse)
[![Go ReportCard](https://goreportcard.com/badge/araddon/dateparse)](https://goreportcard.com/report/araddon/dateparse)

**MM/DD/YYYY VS DD/MM/YYYY** Right now this uses mm/dd/yyyy WHEN ambiguous if this is not desired behavior, use `ParseStrict` which will fail on ambiguous date strings. This can be adjusted using the `PreferMonthFirst` parser option.

```go

// Normal parse.  Equivalent Timezone rules as time.Parse()
t, err := dateparse.ParseAny("3/1/2014")

// Parse Strict, error on ambigous mm/dd vs dd/mm dates
t, err := dateparse.ParseStrict("3/1/2014")
> returns error 

// Return a string that represents the layout to parse the given date-time.
layout, err := dateparse.ParseFormat("May 8, 2009 5:57:51 PM")
> "Jan 2, 2006 3:04:05 PM"

```

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

	"github.com/scylladb/termtables"
	"github.com/araddon/dateparse"
)

var examples = []string{
	"May 8, 2009 5:57:51 PM",
	"oct 7, 1970",
	"oct 7, '70",
	"oct. 7, 1970",
	"oct. 7, 70",
	"Mon Jan  2 15:04:05 2006",
	"Mon Jan  2 15:04:05 MST 2006",
	"Mon Jan 02 15:04:05 -0700 2006",
	"Monday, 02-Jan-06 15:04:05 MST",
	"Mon, 02 Jan 2006 15:04:05 MST",
	"Tue, 11 Jul 2017 16:28:13 +0200 (CEST)",
	"Mon, 02 Jan 2006 15:04:05 -0700",
	"Mon 30 Sep 2018 09:09:09 PM UTC",
	"Mon Aug 10 15:44:11 UTC+0100 2015",
	"Thu, 4 Jan 2018 17:53:36 +0000",
	"Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)",
	"Sun, 3 Jan 2021 00:12:23 +0800 (GMT+08:00)",
	"September 17, 2012 10:09am",
	"September 17, 2012 at 10:09am PST-08",
	"September 17, 2012, 10:10:09",
	"October 7, 1970",
	"October 7th, 1970",
	"12 Feb 2006, 19:17",
	"12 Feb 2006 19:17",
	"14 May 2019 19:11:40.164",
	"7 oct 70",
	"7 oct 1970",
	"03 February 2013",
	"1 July 2013",
	"2013-Feb-03",
	// dd/Mon/yyy  alpha Months
	"06/Jan/2008:15:04:05 -0700",
	"06/Jan/2008 15:04:05 -0700",
	//   mm/dd/yy
	"3/31/2014",
	"03/31/2014",
	"08/21/71",
	"8/1/71",
	"4/8/2014 22:05",
	"04/08/2014 22:05",
	"4/8/14 22:05",
	"04/2/2014 03:00:51",
	"8/8/1965 12:00:00 AM",
	"8/8/1965 01:00:01 PM",
	"8/8/1965 01:00 PM",
	"8/8/1965 1:00 PM",
	"8/8/1965 12:00 AM",
	"4/02/2014 03:00:51",
	"03/19/2012 10:11:59",
	"03/19/2012 10:11:59.3186369",
	// yyyy/mm/dd
	"2014/3/31",
	"2014/03/31",
	"2014/4/8 22:05",
	"2014/04/08 22:05",
	"2014/04/2 03:00:51",
	"2014/4/02 03:00:51",
	"2012/03/19 10:11:59",
	"2012/03/19 10:11:59.3186369",
	// yyyy:mm:dd
	"2014:3:31",
	"2014:03:31",
	"2014:4:8 22:05",
	"2014:04:08 22:05",
	"2014:04:2 03:00:51",
	"2014:4:02 03:00:51",
	"2012:03:19 10:11:59",
	"2012:03:19 10:11:59.3186369",
	// Chinese
	"2014年04月08日",
	//   yyyy-mm-ddThh
	"2006-01-02T15:04:05+0000",
	"2009-08-12T22:15:09-07:00",
	"2009-08-12T22:15:09",
	"2009-08-12T22:15:09.988",
	"2009-08-12T22:15:09Z",
	"2017-07-19T03:21:51:897+0100",
	"2019-05-29T08:41-04", // no seconds, 2 digit TZ offset
	//   yyyy-mm-dd hh:mm:ss
	"2014-04-26 17:24:37.3186369",
	"2012-08-03 18:31:59.257000000",
	"2014-04-26 17:24:37.123",
	"2013-04-01 22:43",
	"2013-04-01 22:43:22",
	"2014-12-16 06:20:00 UTC",
	"2014-12-16 06:20:00 GMT",
	"2014-04-26 05:24:37 PM",
	"2014-04-26 13:13:43 +0800",
	"2014-04-26 13:13:43 +0800 +08",
	"2014-04-26 13:13:44 +09:00",
	"2012-08-03 18:31:59.257000000 +0000 UTC",
	"2015-09-30 18:48:56.35272715 +0000 UTC",
	"2015-02-18 00:12:00 +0000 GMT",
	"2015-02-18 00:12:00 +0000 UTC",
	"2015-02-08 03:02:00 +0300 MSK m=+0.000000001",
	"2015-02-08 03:02:00.001 +0300 MSK m=+0.000000001",
	"2017-07-19 03:21:51+00:00",
	"2014-04-26",
	"2014-04",
	"2014",
	"2014-05-11 08:20:13,787",
	// yyyy-mm-dd-07:00
	"2020-07-20+08:00",
	// mm.dd.yy
	"3.31.2014",
	"03.31.2014",
	"08.21.71",
	"2014.03",
	"2014.03.30",
	//  yyyymmdd and similar
	"20140601",
	"20140722105203",
	// yymmdd hh:mm:yy  mysql log
	// 080313 05:21:55 mysqld started
	"171113 14:14:20",
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
+-------------------------------------------------------+-----------------------------------------+
| Input                                                 | Parsed, and Output as %v                |
+-------------------------------------------------------+-----------------------------------------+
| May 8, 2009 5:57:51 PM                                | 2009-05-08 17:57:51 +0000 UTC           |
| oct 7, 1970                                           | 1970-10-07 00:00:00 +0000 UTC           |
| oct 7, '70                                            | 1970-10-07 00:00:00 +0000 UTC           |
| oct. 7, 1970                                          | 1970-10-07 00:00:00 +0000 UTC           |
| oct. 7, 70                                            | 1970-10-07 00:00:00 +0000 UTC           |
| Mon Jan  2 15:04:05 2006                              | 2006-01-02 15:04:05 +0000 UTC           |
| Mon Jan  2 15:04:05 MST 2006                          | 2006-01-02 15:04:05 +0000 MST           |
| Mon Jan 02 15:04:05 -0700 2006                        | 2006-01-02 15:04:05 -0700 -0700         |
| Monday, 02-Jan-06 15:04:05 MST                        | 2006-01-02 15:04:05 +0000 MST           |
| Mon, 02 Jan 2006 15:04:05 MST                         | 2006-01-02 15:04:05 +0000 MST           |
| Tue, 11 Jul 2017 16:28:13 +0200 (CEST)                | 2017-07-11 16:28:13 +0200 +0200         |
| Mon, 02 Jan 2006 15:04:05 -0700                       | 2006-01-02 15:04:05 -0700 -0700         |
| Mon 30 Sep 2018 09:09:09 PM UTC                       | 2018-09-30 21:09:09 +0000 UTC           |
| Mon Aug 10 15:44:11 UTC+0100 2015                     | 2015-08-10 15:44:11 +0000 UTC           |
| Thu, 4 Jan 2018 17:53:36 +0000                        | 2018-01-04 17:53:36 +0000 UTC           |
| Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time) | 2015-07-03 18:04:07 +0100 GMT           |
| Sun, 3 Jan 2021 00:12:23 +0800 (GMT+08:00)            | 2021-01-03 00:12:23 +0800 +0800         |
| September 17, 2012 10:09am                            | 2012-09-17 10:09:00 +0000 UTC           |
| September 17, 2012 at 10:09am PST-08                  | 2012-09-17 10:09:00 -0800 PST           |
| September 17, 2012, 10:10:09                          | 2012-09-17 10:10:09 +0000 UTC           |
| October 7, 1970                                       | 1970-10-07 00:00:00 +0000 UTC           |
| October 7th, 1970                                     | 1970-10-07 00:00:00 +0000 UTC           |
| 12 Feb 2006, 19:17                                    | 2006-02-12 19:17:00 +0000 UTC           |
| 12 Feb 2006 19:17                                     | 2006-02-12 19:17:00 +0000 UTC           |
| 14 May 2019 19:11:40.164                              | 2019-05-14 19:11:40.164 +0000 UTC       |
| 7 oct 70                                              | 1970-10-07 00:00:00 +0000 UTC           |
| 7 oct 1970                                            | 1970-10-07 00:00:00 +0000 UTC           |
| 03 February 2013                                      | 2013-02-03 00:00:00 +0000 UTC           |
| 1 July 2013                                           | 2013-07-01 00:00:00 +0000 UTC           |
| 2013-Feb-03                                           | 2013-02-03 00:00:00 +0000 UTC           |
| 06/Jan/2008:15:04:05 -0700                            | 2008-01-06 15:04:05 -0700 -0700         |
| 06/Jan/2008 15:04:05 -0700                            | 2008-01-06 15:04:05 -0700 -0700         |
| 3/31/2014                                             | 2014-03-31 00:00:00 +0000 UTC           |
| 03/31/2014                                            | 2014-03-31 00:00:00 +0000 UTC           |
| 08/21/71                                              | 1971-08-21 00:00:00 +0000 UTC           |
| 8/1/71                                                | 1971-08-01 00:00:00 +0000 UTC           |
| 4/8/2014 22:05                                        | 2014-04-08 22:05:00 +0000 UTC           |
| 04/08/2014 22:05                                      | 2014-04-08 22:05:00 +0000 UTC           |
| 4/8/14 22:05                                          | 2014-04-08 22:05:00 +0000 UTC           |
| 04/2/2014 03:00:51                                    | 2014-04-02 03:00:51 +0000 UTC           |
| 8/8/1965 12:00:00 AM                                  | 1965-08-08 00:00:00 +0000 UTC           |
| 8/8/1965 01:00:01 PM                                  | 1965-08-08 13:00:01 +0000 UTC           |
| 8/8/1965 01:00 PM                                     | 1965-08-08 13:00:00 +0000 UTC           |
| 8/8/1965 1:00 PM                                      | 1965-08-08 13:00:00 +0000 UTC           |
| 8/8/1965 12:00 AM                                     | 1965-08-08 00:00:00 +0000 UTC           |
| 4/02/2014 03:00:51                                    | 2014-04-02 03:00:51 +0000 UTC           |
| 03/19/2012 10:11:59                                   | 2012-03-19 10:11:59 +0000 UTC           |
| 03/19/2012 10:11:59.3186369                           | 2012-03-19 10:11:59.3186369 +0000 UTC   |
| 2014/3/31                                             | 2014-03-31 00:00:00 +0000 UTC           |
| 2014/03/31                                            | 2014-03-31 00:00:00 +0000 UTC           |
| 2014/4/8 22:05                                        | 2014-04-08 22:05:00 +0000 UTC           |
| 2014/04/08 22:05                                      | 2014-04-08 22:05:00 +0000 UTC           |
| 2014/04/2 03:00:51                                    | 2014-04-02 03:00:51 +0000 UTC           |
| 2014/4/02 03:00:51                                    | 2014-04-02 03:00:51 +0000 UTC           |
| 2012/03/19 10:11:59                                   | 2012-03-19 10:11:59 +0000 UTC           |
| 2012/03/19 10:11:59.3186369                           | 2012-03-19 10:11:59.3186369 +0000 UTC   |
| 2014:3:31                                             | 2014-03-31 00:00:00 +0000 UTC           |
| 2014:03:31                                            | 2014-03-31 00:00:00 +0000 UTC           |
| 2014:4:8 22:05                                        | 2014-04-08 22:05:00 +0000 UTC           |
| 2014:04:08 22:05                                      | 2014-04-08 22:05:00 +0000 UTC           |
| 2014:04:2 03:00:51                                    | 2014-04-02 03:00:51 +0000 UTC           |
| 2014:4:02 03:00:51                                    | 2014-04-02 03:00:51 +0000 UTC           |
| 2012:03:19 10:11:59                                   | 2012-03-19 10:11:59 +0000 UTC           |
| 2012:03:19 10:11:59.3186369                           | 2012-03-19 10:11:59.3186369 +0000 UTC   |
| 2014年04月08日                                         | 2014-04-08 00:00:00 +0000 UTC           |
| 2006-01-02T15:04:05+0000                              | 2006-01-02 15:04:05 +0000 UTC           |
| 2009-08-12T22:15:09-07:00                             | 2009-08-12 22:15:09 -0700 -0700         |
| 2009-08-12T22:15:09                                   | 2009-08-12 22:15:09 +0000 UTC           |
| 2009-08-12T22:15:09.988                               | 2009-08-12 22:15:09.988 +0000 UTC       |
| 2009-08-12T22:15:09Z                                  | 2009-08-12 22:15:09 +0000 UTC           |
| 2017-07-19T03:21:51:897+0100                          | 2017-07-19 03:21:51.897 +0100 +0100     |
| 2019-05-29T08:41-04                                   | 2019-05-29 08:41:00 -0400 -0400         |
| 2014-04-26 17:24:37.3186369                           | 2014-04-26 17:24:37.3186369 +0000 UTC   |
| 2012-08-03 18:31:59.257000000                         | 2012-08-03 18:31:59.257 +0000 UTC       |
| 2014-04-26 17:24:37.123                               | 2014-04-26 17:24:37.123 +0000 UTC       |
| 2013-04-01 22:43                                      | 2013-04-01 22:43:00 +0000 UTC           |
| 2013-04-01 22:43:22                                   | 2013-04-01 22:43:22 +0000 UTC           |
| 2014-12-16 06:20:00 UTC                               | 2014-12-16 06:20:00 +0000 UTC           |
| 2014-12-16 06:20:00 GMT                               | 2014-12-16 06:20:00 +0000 UTC           |
| 2014-04-26 05:24:37 PM                                | 2014-04-26 17:24:37 +0000 UTC           |
| 2014-04-26 13:13:43 +0800                             | 2014-04-26 13:13:43 +0800 +0800         |
| 2014-04-26 13:13:43 +0800 +08                         | 2014-04-26 13:13:43 +0800 +0800         |
| 2014-04-26 13:13:44 +09:00                            | 2014-04-26 13:13:44 +0900 +0900         |
| 2012-08-03 18:31:59.257000000 +0000 UTC               | 2012-08-03 18:31:59.257 +0000 UTC       |
| 2015-09-30 18:48:56.35272715 +0000 UTC                | 2015-09-30 18:48:56.35272715 +0000 UTC  |
| 2015-02-18 00:12:00 +0000 GMT                         | 2015-02-18 00:12:00 +0000 UTC           |
| 2015-02-18 00:12:00 +0000 UTC                         | 2015-02-18 00:12:00 +0000 UTC           |
| 2015-02-08 03:02:00 +0300 MSK m=+0.000000001          | 2015-02-08 03:02:00 +0300 +0300         |
| 2015-02-08 03:02:00.001 +0300 MSK m=+0.000000001      | 2015-02-08 03:02:00.001 +0300 +0300     |
| 2017-07-19 03:21:51+00:00                             | 2017-07-19 03:21:51 +0000 UTC           |
| 2014-04-26                                            | 2014-04-26 00:00:00 +0000 UTC           |
| 2014-04                                               | 2014-04-01 00:00:00 +0000 UTC           |
| 2014                                                  | 2014-01-01 00:00:00 +0000 UTC           |
| 2014-05-11 08:20:13,787                               | 2014-05-11 08:20:13.787 +0000 UTC       |
| 2020-07-20+08:00                                      | 2020-07-20 00:00:00 +0800 +0800         |
| 3.31.2014                                             | 2014-03-31 00:00:00 +0000 UTC           |
| 03.31.2014                                            | 2014-03-31 00:00:00 +0000 UTC           |
| 08.21.71                                              | 1971-08-21 00:00:00 +0000 UTC           |
| 2014.03                                               | 2014-03-01 00:00:00 +0000 UTC           |
| 2014.03.30                                            | 2014-03-30 00:00:00 +0000 UTC           |
| 20140601                                              | 2014-06-01 00:00:00 +0000 UTC           |
| 20140722105203                                        | 2014-07-22 10:52:03 +0000 UTC           |
| 171113 14:14:20                                       | 2017-11-13 14:14:20 +0000 UTC           |
| 1332151919                                            | 2012-03-19 10:11:59 +0000 UTC           |
| 1384216367189                                         | 2013-11-12 00:32:47.189 +0000 UTC       |
| 1384216367111222                                      | 2013-11-12 00:32:47.111222 +0000 UTC    |
| 1384216367111222333                                   | 2013-11-12 00:32:47.111222333 +0000 UTC |
+-------------------------------------------------------+-----------------------------------------+
*/

```

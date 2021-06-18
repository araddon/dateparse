package dateparse

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOne(t *testing.T) {
	time.Local = time.UTC
	var ts time.Time
	ts = MustParse("2020-07-20+08:00")
	assert.Equal(t, "2020-07-19 16:00:00 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))
}

type dateTest struct {
	in, out, loc string
	err          bool
}

var testInputs = []dateTest{
	{in: "oct 7, 1970", out: "1970-10-07 00:00:00 +0000 UTC"},
	{in: "oct 7, '70", out: "1970-10-07 00:00:00 +0000 UTC"},
	{in: "Oct 7, '70", out: "1970-10-07 00:00:00 +0000 UTC"},
	{in: "Oct. 7, '70", out: "1970-10-07 00:00:00 +0000 UTC"},
	{in: "oct. 7, '70", out: "1970-10-07 00:00:00 +0000 UTC"},
	{in: "oct. 7, 1970", out: "1970-10-07 00:00:00 +0000 UTC"},
	{in: "Sept. 7, '70", out: "1970-09-07 00:00:00 +0000 UTC"},
	{in: "sept. 7, 1970", out: "1970-09-07 00:00:00 +0000 UTC"},
	{in: "Feb 8, 2009 5:57:51 AM", out: "2009-02-08 05:57:51 +0000 UTC"},
	{in: "May 8, 2009 5:57:51 PM", out: "2009-05-08 17:57:51 +0000 UTC"},
	{in: "May 8, 2009 5:57:1 PM", out: "2009-05-08 17:57:01 +0000 UTC"},
	{in: "May 8, 2009 5:7:51 PM", out: "2009-05-08 17:07:51 +0000 UTC"},
	{in: "May 8, 2009, 5:7:51 PM", out: "2009-05-08 17:07:51 +0000 UTC"},
	{in: "7 oct 70", out: "1970-10-07 00:00:00 +0000 UTC"},
	{in: "7 oct 1970", out: "1970-10-07 00:00:00 +0000 UTC"},
	{in: "7 May 1970", out: "1970-05-07 00:00:00 +0000 UTC"},
	{in: "7 Sep 1970", out: "1970-09-07 00:00:00 +0000 UTC"},
	{in: "7 June 1970", out: "1970-06-07 00:00:00 +0000 UTC"},
	{in: "7 September 1970", out: "1970-09-07 00:00:00 +0000 UTC"},
	//   ANSIC       = "Mon Jan _2 15:04:05 2006"
	{in: "Mon Jan  2 15:04:05 2006", out: "2006-01-02 15:04:05 +0000 UTC"},
	{in: "Thu May 8 17:57:51 2009", out: "2009-05-08 17:57:51 +0000 UTC"},
	{in: "Thu May  8 17:57:51 2009", out: "2009-05-08 17:57:51 +0000 UTC"},
	//   ANSIC_GLIBC = "Mon 02 Jan 2006 03:04:05 PM UTC"
	{in: "Mon 02 Jan 2006 03:04:05 PM UTC", out: "2006-01-02 15:04:05 +0000 UTC"},
	{in: "Mon 30 Sep 2018 09:09:09 PM UTC", out: "2018-09-30 21:09:09 +0000 UTC"},
	// RubyDate    = "Mon Jan 02 15:04:05 -0700 2006"
	{in: "Mon Jan 02 15:04:05 -0700 2006", out: "2006-01-02 22:04:05 +0000 UTC"},
	{in: "Thu May 08 11:57:51 -0700 2009", out: "2009-05-08 18:57:51 +0000 UTC"},
	//   UnixDate    = "Mon Jan _2 15:04:05 MST 2006"
	{in: "Mon Jan  2 15:04:05 MST 2006", out: "2006-01-02 15:04:05 +0000 UTC"},
	{in: "Thu May  8 17:57:51 MST 2009", out: "2009-05-08 17:57:51 +0000 UTC"},
	{in: "Thu May  8 17:57:51 PST 2009", out: "2009-05-08 17:57:51 +0000 UTC"},
	{in: "Thu May 08 17:57:51 PST 2009", out: "2009-05-08 17:57:51 +0000 UTC"},
	{in: "Thu May 08 17:57:51 CEST 2009", out: "2009-05-08 17:57:51 +0000 UTC"},
	{in: "Thu May 08 05:05:07 PST 2009", out: "2009-05-08 05:05:07 +0000 UTC"},
	{in: "Thu May 08 5:5:7 PST 2009", out: "2009-05-08 05:05:07 +0000 UTC"},
	// Day Month dd time
	{in: "Mon Aug 10 15:44:11 UTC+0000 2015", out: "2015-08-10 15:44:11 +0000 UTC"},
	{in: "Mon Aug 10 15:44:11 PST-0700 2015", out: "2015-08-10 22:44:11 +0000 UTC"},
	{in: "Mon Aug 10 15:44:11 CEST+0200 2015", out: "2015-08-10 13:44:11 +0000 UTC"},
	{in: "Mon Aug 1 15:44:11 CEST+0200 2015", out: "2015-08-01 13:44:11 +0000 UTC"},
	{in: "Mon Aug 1 5:44:11 CEST+0200 2015", out: "2015-08-01 03:44:11 +0000 UTC"},
	// ??
	{in: "Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)", out: "2015-07-03 17:04:07 +0000 UTC"},
	{in: "Fri Jul 3 2015 06:04:07 GMT+0100 (GMT Daylight Time)", out: "2015-07-03 05:04:07 +0000 UTC"},
	{in: "Fri Jul 3 2015 06:04:07 PST-0700 (Pacific Daylight Time)", out: "2015-07-03 13:04:07 +0000 UTC"},
	// Month dd, yyyy at time
	{in: "September 17, 2012 at 5:00pm UTC-05", out: "2012-09-17 17:00:00 +0000 UTC"},
	{in: "September 17, 2012 at 10:09am PST-08", out: "2012-09-17 18:09:00 +0000 UTC"},
	{in: "September 17, 2012, 10:10:09", out: "2012-09-17 10:10:09 +0000 UTC"},
	{in: "May 17, 2012 at 10:09am PST-08", out: "2012-05-17 18:09:00 +0000 UTC"},
	{in: "May 17, 2012 AT 10:09am PST-08", out: "2012-05-17 18:09:00 +0000 UTC"},
	// Month dd, yyyy time
	{in: "September 17, 2012 5:00pm UTC-05", out: "2012-09-17 17:00:00 +0000 UTC"},
	{in: "September 17, 2012 10:09am PST-08", out: "2012-09-17 18:09:00 +0000 UTC"},
	{in: "September 17, 2012 09:01:00", out: "2012-09-17 09:01:00 +0000 UTC"},
	// Month dd yyyy time
	{in: "September 17 2012 5:00pm UTC-05", out: "2012-09-17 17:00:00 +0000 UTC"},
	{in: "September 17 2012 5:00pm UTC-0500", out: "2012-09-17 17:00:00 +0000 UTC"},
	{in: "September 17 2012 10:09am PST-08", out: "2012-09-17 18:09:00 +0000 UTC"},
	{in: "September 17 2012 5:00PM UTC-05", out: "2012-09-17 17:00:00 +0000 UTC"},
	{in: "September 17 2012 10:09AM PST-08", out: "2012-09-17 18:09:00 +0000 UTC"},
	{in: "September 17 2012 09:01:00", out: "2012-09-17 09:01:00 +0000 UTC"},
	{in: "May 17, 2012 10:10:09", out: "2012-05-17 10:10:09 +0000 UTC"},
	// Month dd, yyyy
	{in: "September 17, 2012", out: "2012-09-17 00:00:00 +0000 UTC"},
	{in: "May 7, 2012", out: "2012-05-07 00:00:00 +0000 UTC"},
	{in: "June 7, 2012", out: "2012-06-07 00:00:00 +0000 UTC"},
	{in: "June 7 2012", out: "2012-06-07 00:00:00 +0000 UTC"},
	// Month dd[th,nd,st,rd] yyyy
	{in: "September 17th, 2012", out: "2012-09-17 00:00:00 +0000 UTC"},
	{in: "September 17th 2012", out: "2012-09-17 00:00:00 +0000 UTC"},
	{in: "September 7th, 2012", out: "2012-09-07 00:00:00 +0000 UTC"},
	{in: "September 7th 2012", out: "2012-09-07 00:00:00 +0000 UTC"},
	{in: "September 7tH 2012", out: "2012-09-07 00:00:00 +0000 UTC"},
	{in: "May 1st 2012", out: "2012-05-01 00:00:00 +0000 UTC"},
	{in: "May 1st, 2012", out: "2012-05-01 00:00:00 +0000 UTC"},
	{in: "May 21st 2012", out: "2012-05-21 00:00:00 +0000 UTC"},
	{in: "May 21st, 2012", out: "2012-05-21 00:00:00 +0000 UTC"},
	{in: "May 23rd 2012", out: "2012-05-23 00:00:00 +0000 UTC"},
	{in: "May 23rd, 2012", out: "2012-05-23 00:00:00 +0000 UTC"},
	{in: "June 2nd, 2012", out: "2012-06-02 00:00:00 +0000 UTC"},
	{in: "June 2nd 2012", out: "2012-06-02 00:00:00 +0000 UTC"},
	{in: "June 22nd, 2012", out: "2012-06-22 00:00:00 +0000 UTC"},
	{in: "June 22nd 2012", out: "2012-06-22 00:00:00 +0000 UTC"},
	// RFC1123     = "Mon, 02 Jan 2006 15:04:05 MST"
	{in: "Fri, 03 Jul 2015 08:08:08 MST", out: "2015-07-03 08:08:08 +0000 UTC"},
	//{in: "Fri, 03 Jul 2015 08:08:08 CET", out: "2015-07-03 08:08:08 +0000 UTC"},
	{in: "Fri, 03 Jul 2015 08:08:08 PST", out: "2015-07-03 16:08:08 +0000 UTC", loc: "America/Los_Angeles"},
	{in: "Fri, 03 Jul 2015 08:08:08 PST", out: "2015-07-03 08:08:08 +0000 UTC"},
	{in: "Fri, 3 Jul 2015 08:08:08 MST", out: "2015-07-03 08:08:08 +0000 UTC"},
	{in: "Fri, 03 Jul 2015 8:08:08 MST", out: "2015-07-03 08:08:08 +0000 UTC"},
	{in: "Fri, 03 Jul 2015 8:8:8 MST", out: "2015-07-03 08:08:08 +0000 UTC"},
	// ?
	{in: "Thu, 03 Jul 2017 08:08:04 +0100", out: "2017-07-03 07:08:04 +0000 UTC"},
	{in: "Thu, 03 Jul 2017 08:08:04 -0100", out: "2017-07-03 09:08:04 +0000 UTC"},
	{in: "Thu, 3 Jul 2017 08:08:04 +0100", out: "2017-07-03 07:08:04 +0000 UTC"},
	{in: "Thu, 03 Jul 2017 8:08:04 +0100", out: "2017-07-03 07:08:04 +0000 UTC"},
	{in: "Thu, 03 Jul 2017 8:8:4 +0100", out: "2017-07-03 07:08:04 +0000 UTC"},
	//
	{in: "Tue, 11 Jul 2017 04:08:03 +0200 (CEST)", out: "2017-07-11 02:08:03 +0000 UTC"},
	{in: "Tue, 5 Jul 2017 04:08:03 -0700 (CEST)", out: "2017-07-05 11:08:03 +0000 UTC"},
	{in: "Tue, 11 Jul 2017 04:08:03 +0200 (CEST)", out: "2017-07-11 02:08:03 +0000 UTC", loc: "Europe/Berlin"},
	// day, dd-Mon-yy hh:mm:zz TZ
	{in: "Fri, 03-Jul-15 08:08:08 MST", out: "2015-07-03 08:08:08 +0000 UTC"},
	{in: "Fri, 03-Jul-15 08:08:08 PST", out: "2015-07-03 16:08:08 +0000 UTC", loc: "America/Los_Angeles"},
	{in: "Fri, 03-Jul 2015 08:08:08 PST", out: "2015-07-03 08:08:08 +0000 UTC"},
	{in: "Fri, 3-Jul-15 08:08:08 MST", out: "2015-07-03 08:08:08 +0000 UTC"},
	{in: "Fri, 03-Jul-15 8:08:08 MST", out: "2015-07-03 08:08:08 +0000 UTC"},
	{in: "Fri, 03-Jul-15 8:8:8 MST", out: "2015-07-03 08:08:08 +0000 UTC"},
	// day, dd-Mon-yy hh:mm:zz TZ (text) https://github.com/araddon/dateparse/issues/116
	{in: "Sun, 3 Jan 2021 00:12:23 +0800 (GMT+08:00)", out: "2021-01-02 16:12:23 +0000 UTC"},
	// RFC850    = "Monday, 02-Jan-06 15:04:05 MST"
	{in: "Wednesday, 07-May-09 08:00:43 MST", out: "2009-05-07 08:00:43 +0000 UTC"},
	{in: "Wednesday, 28-Feb-18 09:01:00 MST", out: "2018-02-28 09:01:00 +0000 UTC"},
	{in: "Wednesday, 28-Feb-18 09:01:00 MST", out: "2018-02-28 16:01:00 +0000 UTC", loc: "America/Denver"},
	// with offset then with variations on non-zero filled stuff
	{in: "Monday, 02 Jan 2006 15:04:05 +0100", out: "2006-01-02 14:04:05 +0000 UTC"},
	{in: "Wednesday, 28 Feb 2018 09:01:00 -0300", out: "2018-02-28 12:01:00 +0000 UTC"},
	{in: "Wednesday, 2 Feb 2018 09:01:00 -0300", out: "2018-02-02 12:01:00 +0000 UTC"},
	{in: "Wednesday, 2 Feb 2018 9:01:00 -0300", out: "2018-02-02 12:01:00 +0000 UTC"},
	{in: "Wednesday, 2 Feb 2018 09:1:00 -0300", out: "2018-02-02 12:01:00 +0000 UTC"},
	//  dd mon yyyy  12 Feb 2006, 19:17:08
	{in: "07 Feb 2004, 09:07", out: "2004-02-07 09:07:00 +0000 UTC"},
	{in: "07 Feb 2004, 09:07:07", out: "2004-02-07 09:07:07 +0000 UTC"},
	{in: "7 Feb 2004, 09:07:07", out: "2004-02-07 09:07:07 +0000 UTC"},
	{in: "07 Feb 2004, 9:7:7", out: "2004-02-07 09:07:07 +0000 UTC"},
	// dd Mon yyyy hh:mm:ss
	{in: "07 Feb 2004 09:07:08", out: "2004-02-07 09:07:08 +0000 UTC"},
	{in: "07 Feb 2004 09:07", out: "2004-02-07 09:07:00 +0000 UTC"},
	{in: "7 Feb 2004 9:7:8", out: "2004-02-07 09:07:08 +0000 UTC"},
	{in: "07 Feb 2004 09:07:08.123", out: "2004-02-07 09:07:08.123 +0000 UTC"},
	//  dd-mon-yyyy  12 Feb 2006, 19:17:08 GMT
	{in: "07 Feb 2004, 09:07:07 GMT", out: "2004-02-07 09:07:07 +0000 UTC"},
	//  dd-mon-yyyy  12 Feb 2006, 19:17:08 +0100
	{in: "07 Feb 2004, 09:07:07 +0100", out: "2004-02-07 08:07:07 +0000 UTC"},
	//  dd-mon-yyyy   12-Feb-2006 19:17:08
	{in: "07-Feb-2004 09:07:07 +0100", out: "2004-02-07 08:07:07 +0000 UTC"},
	//  dd-mon-yy   12-Feb-2006 19:17:08
	{in: "07-Feb-04 09:07:07 +0100", out: "2004-02-07 08:07:07 +0000 UTC"},
	// yyyy-mon-dd    2013-Feb-03
	{in: "2013-Feb-03", out: "2013-02-03 00:00:00 +0000 UTC"},
	// 03 February 2013
	{in: "03 February 2013", out: "2013-02-03 00:00:00 +0000 UTC"},
	{in: "3 February 2013", out: "2013-02-03 00:00:00 +0000 UTC"},
	// Chinese 2014年04月18日
	{in: "2014年04月08日", out: "2014-04-08 00:00:00 +0000 UTC"},
	{in: "2014年04月08日 19:17:22", out: "2014-04-08 19:17:22 +0000 UTC"},
	//  mm/dd/yyyy
	{in: "03/31/2014", out: "2014-03-31 00:00:00 +0000 UTC"},
	{in: "3/31/2014", out: "2014-03-31 00:00:00 +0000 UTC"},
	{in: "3/5/2014", out: "2014-03-05 00:00:00 +0000 UTC"},
	//  mm/dd/yy
	{in: "08/08/71", out: "1971-08-08 00:00:00 +0000 UTC"},
	{in: "8/8/71", out: "1971-08-08 00:00:00 +0000 UTC"},
	//  mm/dd/yy hh:mm:ss
	{in: "04/02/2014 04:08:09", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "4/2/2014 04:08:09", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "04/02/2014 4:08:09", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "04/02/2014 4:8:9", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "04/02/2014 04:08", out: "2014-04-02 04:08:00 +0000 UTC"},
	{in: "04/02/2014 4:8", out: "2014-04-02 04:08:00 +0000 UTC"},
	{in: "04/02/2014 04:08:09.123", out: "2014-04-02 04:08:09.123 +0000 UTC"},
	{in: "04/02/2014 04:08:09.12312", out: "2014-04-02 04:08:09.12312 +0000 UTC"},
	{in: "04/02/2014 04:08:09.123123", out: "2014-04-02 04:08:09.123123 +0000 UTC"},
	//  mm:dd:yy hh:mm:ss
	{in: "04:02:2014 04:08:09", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "4:2:2014 04:08:09", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "04:02:2014 4:08:09", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "04:02:2014 4:8:9", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "04:02:2014 04:08", out: "2014-04-02 04:08:00 +0000 UTC"},
	{in: "04:02:2014 4:8", out: "2014-04-02 04:08:00 +0000 UTC"},
	{in: "04:02:2014 04:08:09.123", out: "2014-04-02 04:08:09.123 +0000 UTC"},
	{in: "04:02:2014 04:08:09.12312", out: "2014-04-02 04:08:09.12312 +0000 UTC"},
	{in: "04:02:2014 04:08:09.123123", out: "2014-04-02 04:08:09.123123 +0000 UTC"},
	//  mm/dd/yy hh:mm:ss AM
	{in: "04/02/2014 04:08:09 AM", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "04/02/2014 04:08:09 PM", out: "2014-04-02 16:08:09 +0000 UTC"},
	{in: "04/02/2014 04:08 AM", out: "2014-04-02 04:08:00 +0000 UTC"},
	{in: "04/02/2014 04:08 PM", out: "2014-04-02 16:08:00 +0000 UTC"},
	{in: "04/02/2014 4:8 AM", out: "2014-04-02 04:08:00 +0000 UTC"},
	{in: "04/02/2014 4:8 PM", out: "2014-04-02 16:08:00 +0000 UTC"},
	{in: "04/02/2014 04:08:09.123 AM", out: "2014-04-02 04:08:09.123 +0000 UTC"},
	{in: "04/02/2014 04:08:09.123 PM", out: "2014-04-02 16:08:09.123 +0000 UTC"},
	//   yyyy/mm/dd
	{in: "2014/04/02", out: "2014-04-02 00:00:00 +0000 UTC"},
	{in: "2014/03/31", out: "2014-03-31 00:00:00 +0000 UTC"},
	{in: "2014/4/2", out: "2014-04-02 00:00:00 +0000 UTC"},
	//   yyyy/mm/dd hh:mm:ss AM
	{in: "2014/04/02 04:08", out: "2014-04-02 04:08:00 +0000 UTC"},
	{in: "2014/03/31 04:08", out: "2014-03-31 04:08:00 +0000 UTC"},
	{in: "2014/4/2 04:08", out: "2014-04-02 04:08:00 +0000 UTC"},
	{in: "2014/04/02 4:8", out: "2014-04-02 04:08:00 +0000 UTC"},
	{in: "2014/04/02 04:08:09", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "2014/03/31 04:08:09", out: "2014-03-31 04:08:09 +0000 UTC"},
	{in: "2014/4/2 04:08:09", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "2014/04/02 04:08:09.123", out: "2014-04-02 04:08:09.123 +0000 UTC"},
	{in: "2014/04/02 04:08:09.123123", out: "2014-04-02 04:08:09.123123 +0000 UTC"},
	{in: "2014/04/02 04:08:09 AM", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "2014/03/31 04:08:09 AM", out: "2014-03-31 04:08:09 +0000 UTC"},
	{in: "2014/4/2 04:08:09 AM", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "2014/04/02 04:08:09.123 AM", out: "2014-04-02 04:08:09.123 +0000 UTC"},
	{in: "2014/04/02 04:08:09.123 PM", out: "2014-04-02 16:08:09.123 +0000 UTC"},
	// dd/mon/yyyy:hh:mm:ss tz  nginx-log?    https://github.com/araddon/dateparse/issues/118
	// 112.195.209.90 - - [20/Feb/2018:12:12:14 +0800] "GET / HTTP/1.1" 200 190 "-" "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.132 Mobile Safari/537.36" "-"
	{in: "06/May/2008:08:11:17 -0700", out: "2008-05-06 15:11:17 +0000 UTC"},
	{in: "30/May/2008:08:11:17 -0700", out: "2008-05-30 15:11:17 +0000 UTC"},
	// dd/mon/yyy hh:mm:ss tz
	{in: "06/May/2008:08:11:17 -0700", out: "2008-05-06 15:11:17 +0000 UTC"},
	{in: "30/May/2008:08:11:17 -0700", out: "2008-05-30 15:11:17 +0000 UTC"},
	//   yyyy-mm-dd
	{in: "2014-04-02", out: "2014-04-02 00:00:00 +0000 UTC"},
	{in: "2014-03-31", out: "2014-03-31 00:00:00 +0000 UTC"},
	{in: "2014-4-2", out: "2014-04-02 00:00:00 +0000 UTC"},
	//   yyyy-mm-dd-07:00
	{in: "2020-07-20+08:00", out: "2020-07-19 16:00:00 +0000 UTC"},
	{in: "2020-07-20+0800", out: "2020-07-19 16:00:00 +0000 UTC"},
	//   dd-mmm-yy
	{in: "28-Feb-02", out: "2002-02-28 00:00:00 +0000 UTC"},
	{in: "15-Jan-18", out: "2018-01-15 00:00:00 +0000 UTC"},
	{in: "15-Jan-2017", out: "2017-01-15 00:00:00 +0000 UTC"},
	// yyyy-mm
	{in: "2014-04", out: "2014-04-01 00:00:00 +0000 UTC"},
	//   yyyy-mm-dd hh:mm:ss AM
	{in: "2014-04-02 04:08", out: "2014-04-02 04:08:00 +0000 UTC"},
	{in: "2014-03-31 04:08", out: "2014-03-31 04:08:00 +0000 UTC"},
	{in: "2014-4-2 04:08", out: "2014-04-02 04:08:00 +0000 UTC"},
	{in: "2014-04-02 4:8", out: "2014-04-02 04:08:00 +0000 UTC"},
	{in: "2014-04-02 04:08:09", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "2014-03-31 04:08:09", out: "2014-03-31 04:08:09 +0000 UTC"},
	{in: "2014-4-2 04:08:09", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "2014-04-02 04:08:09.123", out: "2014-04-02 04:08:09.123 +0000 UTC"},
	{in: "2014-04-02 04:08:09.123123", out: "2014-04-02 04:08:09.123123 +0000 UTC"},
	{in: "2014-04-02 04:08:09.12312312", out: "2014-04-02 04:08:09.12312312 +0000 UTC"},
	{in: "2014-04-02 04:08:09 AM", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "2014-03-31 04:08:09 AM", out: "2014-03-31 04:08:09 +0000 UTC"},
	{in: "2014-04-26 05:24:37 PM", out: "2014-04-26 17:24:37 +0000 UTC"},
	{in: "2014-4-2 04:08:09 AM", out: "2014-04-02 04:08:09 +0000 UTC"},
	{in: "2014-04-02 04:08:09.123 AM", out: "2014-04-02 04:08:09.123 +0000 UTC"},
	{in: "2014-04-02 04:08:09.123 PM", out: "2014-04-02 16:08:09.123 +0000 UTC"},
	//   yyyy-mm-dd hh:mm:ss,000
	{in: "2014-05-11 08:20:13,787", out: "2014-05-11 08:20:13.787 +0000 UTC"},
	//   yyyy-mm-dd hh:mm:ss +0000
	{in: "2012-08-03 18:31:59 +0000", out: "2012-08-03 18:31:59 +0000 UTC"},
	{in: "2012-08-03 13:31:59 -0600", out: "2012-08-03 19:31:59 +0000 UTC"},
	{in: "2012-08-03 18:31:59.257000000 +0000", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2012-08-03 8:1:59.257000000 +0000", out: "2012-08-03 08:01:59.257 +0000 UTC"},
	{in: "2012-8-03 18:31:59.257000000 +0000", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2012-8-3 18:31:59.257000000 +0000", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2014-04-26 17:24:37.123456 +0000", out: "2014-04-26 17:24:37.123456 +0000 UTC"},
	{in: "2014-04-26 17:24:37.12 +0000", out: "2014-04-26 17:24:37.12 +0000 UTC"},
	{in: "2014-04-26 17:24:37.1 +0000", out: "2014-04-26 17:24:37.1 +0000 UTC"},
	{in: "2014-05-11 08:20:13 +0000", out: "2014-05-11 08:20:13 +0000 UTC"},
	{in: "2014-05-11 08:20:13 +0530", out: "2014-05-11 02:50:13 +0000 UTC"},
	//   yyyy-mm-dd hh:mm:ss +0300 +03  ?? issue author said this is from golang?
	{in: "2018-06-29 19:09:57.77297118 +0300 +03", out: "2018-06-29 16:09:57.77297118 +0000 UTC"},
	{in: "2018-06-29 19:09:57.77297118 +0300 +0300", out: "2018-06-29 16:09:57.77297118 +0000 UTC"},
	{in: "2018-06-29 19:09:57 +0300 +03", out: "2018-06-29 16:09:57 +0000 UTC"},
	{in: "2018-06-29 19:09:57 +0300 +0300", out: "2018-06-29 16:09:57 +0000 UTC"},

	// 13:31:51.999 -07:00 MST
	//   yyyy-mm-dd hh:mm:ss +00:00
	{in: "2012-08-03 18:31:59 +00:00", out: "2012-08-03 18:31:59 +0000 UTC"},
	{in: "2014-05-01 08:02:13 +00:00", out: "2014-05-01 08:02:13 +0000 UTC"},
	{in: "2014-5-01 08:02:13 +00:00", out: "2014-05-01 08:02:13 +0000 UTC"},
	{in: "2014-05-1 08:02:13 +00:00", out: "2014-05-01 08:02:13 +0000 UTC"},
	{in: "2012-08-03 13:31:59 -06:00", out: "2012-08-03 19:31:59 +0000 UTC"},
	{in: "2012-08-03 18:31:59.257000000 +00:00", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2012-08-03 8:1:59.257000000 +00:00", out: "2012-08-03 08:01:59.257 +0000 UTC"},
	{in: "2012-8-03 18:31:59.257000000 +00:00", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2012-8-3 18:31:59.257000000 +00:00", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2014-04-26 17:24:37.123456 +00:00", out: "2014-04-26 17:24:37.123456 +0000 UTC"},
	{in: "2014-04-26 17:24:37.12 +00:00", out: "2014-04-26 17:24:37.12 +0000 UTC"},
	{in: "2014-04-26 17:24:37.1 +00:00", out: "2014-04-26 17:24:37.1 +0000 UTC"},
	//   yyyy-mm-dd hh:mm:ss +0000 TZ
	// Golang Native Format
	{in: "2012-08-03 18:31:59 +0000 UTC", out: "2012-08-03 18:31:59 +0000 UTC"},
	{in: "2012-08-03 13:31:59 -0600 MST", out: "2012-08-03 19:31:59 +0000 UTC", loc: "America/Denver"},
	{in: "2015-02-18 00:12:00 +0000 UTC", out: "2015-02-18 00:12:00 +0000 UTC"},
	{in: "2015-02-18 00:12:00 +0000 GMT", out: "2015-02-18 00:12:00 +0000 UTC"},
	{in: "2015-02-08 03:02:00 +0200 CEST", out: "2015-02-08 01:02:00 +0000 UTC", loc: "Europe/Berlin"},
	{in: "2015-02-08 03:02:00 +0300 MSK", out: "2015-02-08 00:02:00 +0000 UTC"},
	{in: "2015-2-08 03:02:00 +0300 MSK", out: "2015-02-08 00:02:00 +0000 UTC"},
	{in: "2015-02-8 03:02:00 +0300 MSK", out: "2015-02-08 00:02:00 +0000 UTC"},
	{in: "2015-2-8 03:02:00 +0300 MSK", out: "2015-02-08 00:02:00 +0000 UTC"},
	{in: "2012-08-03 18:31:59.257000000 +0000 UTC", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2012-08-03 8:1:59.257000000 +0000 UTC", out: "2012-08-03 08:01:59.257 +0000 UTC"},
	{in: "2012-8-03 18:31:59.257000000 +0000 UTC", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2012-8-3 18:31:59.257000000 +0000 UTC", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2014-04-26 17:24:37.123456 +0000 UTC", out: "2014-04-26 17:24:37.123456 +0000 UTC"},
	{in: "2014-04-26 17:24:37.12 +0000 UTC", out: "2014-04-26 17:24:37.12 +0000 UTC"},
	{in: "2014-04-26 17:24:37.1 +0000 UTC", out: "2014-04-26 17:24:37.1 +0000 UTC"},
	{in: "2015-02-08 03:02:00 +0200 CEST m=+0.000000001", out: "2015-02-08 01:02:00 +0000 UTC", loc: "Europe/Berlin"},
	{in: "2015-02-08 03:02:00 +0300 MSK m=+0.000000001", out: "2015-02-08 00:02:00 +0000 UTC"},
	{in: "2015-02-08 03:02:00.001 +0300 MSK m=+0.000000001", out: "2015-02-08 00:02:00.001 +0000 UTC"},
	//   yyyy-mm-dd hh:mm:ss TZ
	{in: "2012-08-03 18:31:59 UTC", out: "2012-08-03 18:31:59 +0000 UTC"},
	{in: "2014-12-16 06:20:00 GMT", out: "2014-12-16 06:20:00 +0000 UTC"},
	{in: "2012-08-03 13:31:59 MST", out: "2012-08-03 20:31:59 +0000 UTC", loc: "America/Denver"},
	{in: "2012-08-03 18:31:59.257000000 UTC", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2012-08-03 8:1:59.257000000 UTC", out: "2012-08-03 08:01:59.257 +0000 UTC"},
	{in: "2012-8-03 18:31:59.257000000 UTC", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2012-8-3 18:31:59.257000000 UTC", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2014-04-26 17:24:37.123456 UTC", out: "2014-04-26 17:24:37.123456 +0000 UTC"},
	{in: "2014-04-26 17:24:37.12 UTC", out: "2014-04-26 17:24:37.12 +0000 UTC"},
	{in: "2014-04-26 17:24:37.1 UTC", out: "2014-04-26 17:24:37.1 +0000 UTC"},
	// This one is pretty special, it is TIMEZONE based but starts with P to emulate collions with PM
	{in: "2014-04-26 05:24:37 PST", out: "2014-04-26 05:24:37 +0000 UTC"},
	{in: "2014-04-26 05:24:37 PST", out: "2014-04-26 13:24:37 +0000 UTC", loc: "America/Los_Angeles"},
	//   yyyy-mm-dd hh:mm:ss+00:00
	{in: "2012-08-03 18:31:59+00:00", out: "2012-08-03 18:31:59 +0000 UTC"},
	{in: "2017-07-19 03:21:51+00:00", out: "2017-07-19 03:21:51 +0000 UTC"},
	//   yyyy:mm:dd hh:mm:ss+00:00
	{in: "2012:08:03 18:31:59+00:00", out: "2012-08-03 18:31:59 +0000 UTC"},
	//   dd:mm:yyyy hh:mm:ss+00:00
	{in: "08:03:2012 18:31:59+00:00", out: "2012-08-03 18:31:59 +0000 UTC"},
	//   yyyy-mm-dd hh:mm:ss.000+00:00 PST
	{in: "2012-08-03 18:31:59.000+00:00 PST", out: "2012-08-03 18:31:59 +0000 UTC", loc: "America/Los_Angeles"},
	//   yyyy-mm-dd hh:mm:ss +00:00 TZ
	{in: "2012-08-03 18:31:59 +00:00 UTC", out: "2012-08-03 18:31:59 +0000 UTC"},
	{in: "2012-08-03 13:31:51 -07:00 MST", out: "2012-08-03 20:31:51 +0000 UTC", loc: "America/Denver"},
	{in: "2012-08-03 18:31:59.257000000 +00:00 UTC", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2012-08-03 13:31:51.123 -08:00 PST", out: "2012-08-03 21:31:51.123 +0000 UTC", loc: "America/Los_Angeles"},
	{in: "2012-08-03 13:31:51.123 +02:00 CEST", out: "2012-08-03 11:31:51.123 +0000 UTC", loc: "Europe/Berlin"},
	{in: "2012-08-03 8:1:59.257000000 +00:00 UTC", out: "2012-08-03 08:01:59.257 +0000 UTC"},
	{in: "2012-8-03 18:31:59.257000000 +00:00 UTC", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2012-8-3 18:31:59.257000000 +00:00 UTC", out: "2012-08-03 18:31:59.257 +0000 UTC"},
	{in: "2014-04-26 17:24:37.123456 +00:00 UTC", out: "2014-04-26 17:24:37.123456 +0000 UTC"},
	{in: "2014-04-26 17:24:37.12 +00:00 UTC", out: "2014-04-26 17:24:37.12 +0000 UTC"},
	{in: "2014-04-26 17:24:37.1 +00:00 UTC", out: "2014-04-26 17:24:37.1 +0000 UTC"},
	//   yyyy-mm-ddThh:mm:ss
	{in: "2009-08-12T22:15:09", out: "2009-08-12 22:15:09 +0000 UTC"},
	{in: "2009-08-08T02:08:08", out: "2009-08-08 02:08:08 +0000 UTC"},
	{in: "2009-08-08T2:8:8", out: "2009-08-08 02:08:08 +0000 UTC"},
	{in: "2009-08-12T22:15:09.123", out: "2009-08-12 22:15:09.123 +0000 UTC"},
	{in: "2009-08-12T22:15:09.123456", out: "2009-08-12 22:15:09.123456 +0000 UTC"},
	{in: "2009-08-12T22:15:09.12", out: "2009-08-12 22:15:09.12 +0000 UTC"},
	{in: "2009-08-12T22:15:09.1", out: "2009-08-12 22:15:09.1 +0000 UTC"},
	{in: "2014-04-26 17:24:37.3186369", out: "2014-04-26 17:24:37.3186369 +0000 UTC"},
	//   yyyy-mm-ddThh:mm:ss-07:00
	{in: "2009-08-12T22:15:09-07:00", out: "2009-08-13 05:15:09 +0000 UTC"},
	{in: "2009-08-12T22:15:09-03:00", out: "2009-08-13 01:15:09 +0000 UTC"},
	{in: "2009-08-12T22:15:9-07:00", out: "2009-08-13 05:15:09 +0000 UTC"},
	{in: "2009-08-12T22:15:09.123-07:00", out: "2009-08-13 05:15:09.123 +0000 UTC"},
	{in: "2016-06-21T19:55:00+01:00", out: "2016-06-21 18:55:00 +0000 UTC"},
	{in: "2016-06-21T19:55:00.799+01:00", out: "2016-06-21 18:55:00.799 +0000 UTC"},
	//   yyyy-mm-ddThh:mm:ss-07   TZ truncated to 2 digits instead of 4
	{in: "2019-05-29T08:41-04", out: "2019-05-29 12:41:00 +0000 UTC"},
	//   yyyy-mm-ddThh:mm:ss-0700
	{in: "2009-08-12T22:15:09-0700", out: "2009-08-13 05:15:09 +0000 UTC"},
	{in: "2009-08-12T22:15:09-0300", out: "2009-08-13 01:15:09 +0000 UTC"},
	{in: "2009-08-12T22:15:9-0700", out: "2009-08-13 05:15:09 +0000 UTC"},
	{in: "2009-08-12T22:15:09.123-0700", out: "2009-08-13 05:15:09.123 +0000 UTC"},
	{in: "2016-06-21T19:55:00+0100", out: "2016-06-21 18:55:00 +0000 UTC"},
	{in: "2016-06-21T19:55:00.799+0100", out: "2016-06-21 18:55:00.799 +0000 UTC"},
	{in: "2016-06-21T19:55:00+0100", out: "2016-06-21 18:55:00 +0000 UTC"},
	{in: "2016-06-21T19:55:00-0700", out: "2016-06-22 02:55:00 +0000 UTC"},
	{in: "2016-06-21T19:55:00.799+0100", out: "2016-06-21 18:55:00.799 +0000 UTC"},
	{in: "2016-06-21T19:55+0100", out: "2016-06-21 18:55:00 +0000 UTC"},
	{in: "2016-06-21T19:55+0130", out: "2016-06-21 18:25:00 +0000 UTC"},
	//   yyyy-mm-ddThh:mm:ss:000+0000    - weird format with additional colon in front of milliseconds
	{in: "2012-08-17T18:31:59:257+0100", out: "2012-08-17 17:31:59.257 +0000 UTC"}, // https://github.com/araddon/dateparse/issues/117

	//   yyyy-mm-ddThh:mm:ssZ
	{in: "2009-08-12T22:15Z", out: "2009-08-12 22:15:00 +0000 UTC"},
	{in: "2009-08-12T22:15:09Z", out: "2009-08-12 22:15:09 +0000 UTC"},
	{in: "2009-08-12T22:15:09.99Z", out: "2009-08-12 22:15:09.99 +0000 UTC"},
	{in: "2009-08-12T22:15:09.9999Z", out: "2009-08-12 22:15:09.9999 +0000 UTC"},
	{in: "2009-08-12T22:15:09.99999999Z", out: "2009-08-12 22:15:09.99999999 +0000 UTC"},
	{in: "2009-08-12T22:15:9.99999999Z", out: "2009-08-12 22:15:09.99999999 +0000 UTC"},
	// yyyy.mm
	{in: "2014.05", out: "2014-05-01 00:00:00 +0000 UTC"},
	{in: "2018.09.30", out: "2018-09-30 00:00:00 +0000 UTC"},

	//   mm.dd.yyyy
	{in: "3.31.2014", out: "2014-03-31 00:00:00 +0000 UTC"},
	{in: "3.3.2014", out: "2014-03-03 00:00:00 +0000 UTC"},
	{in: "03.31.2014", out: "2014-03-31 00:00:00 +0000 UTC"},
	//   mm.dd.yy
	{in: "08.21.71", out: "1971-08-21 00:00:00 +0000 UTC"},
	//  yyyymmdd and similar
	{in: "2014", out: "2014-01-01 00:00:00 +0000 UTC"},
	{in: "20140601", out: "2014-06-01 00:00:00 +0000 UTC"},
	{in: "20140722105203", out: "2014-07-22 10:52:03 +0000 UTC"},
	// yymmdd hh:mm:yy  mysql log  https://github.com/araddon/dateparse/issues/119
	// 080313 05:21:55 mysqld started
	// 080313 5:21:55 InnoDB: Started; log sequence number 0 43655
	{in: "171113 14:14:20", out: "2017-11-13 14:14:20 +0000 UTC"},

	// all digits:  unix secs, ms etc
	{in: "1332151919", out: "2012-03-19 10:11:59 +0000 UTC"},
	{in: "1332151919", out: "2012-03-19 10:11:59 +0000 UTC", loc: "America/Denver"},
	{in: "1384216367111", out: "2013-11-12 00:32:47.111 +0000 UTC"},
	{in: "1384216367111222", out: "2013-11-12 00:32:47.111222 +0000 UTC"},
	{in: "1384216367111222333", out: "2013-11-12 00:32:47.111222333 +0000 UTC"},

	// dd[th,nd,st,rd] Month yyyy
	{in: "1st September 2012", out: "2012-09-01 00:00:00 +0000 UTC"},
	{in: "2nd September 2012", out: "2012-09-02 00:00:00 +0000 UTC"},
	{in: "3rd September 2012", out: "2012-09-03 00:00:00 +0000 UTC"},
	{in: "4th September 2012", out: "2012-09-04 00:00:00 +0000 UTC"},
	{in: "2nd January 2018", out: "2018-01-02 00:00:00 +0000 UTC"},
	{in: "3nd Feb 2018 13:58:24", out: "2018-02-03 13:58:24 +0000 UTC"},
}

func TestParse(t *testing.T) {

	// Lets ensure we are operating on UTC
	time.Local = time.UTC

	zeroTime := time.Time{}.Unix()
	ts, err := ParseAny("INVALID")
	assert.Equal(t, zeroTime, ts.Unix())
	assert.NotEqual(t, nil, err)

	assert.Equal(t, true, testDidPanic("NOT GONNA HAPPEN"))
	// https://github.com/golang/go/issues/5294
	_, err = ParseAny(time.RFC3339)
	assert.NotEqual(t, nil, err)

	for _, th := range testInputs {
		if len(th.loc) > 0 {
			loc, err := time.LoadLocation(th.loc)
			if err != nil {
				t.Fatalf("Expected to load location %q but got %v", th.loc, err)
			}
			ts, err = ParseIn(th.in, loc)
			if err != nil {
				t.Fatalf("expected to parse %q but got %v", th.in, err)
			}
			got := fmt.Sprintf("%v", ts.In(time.UTC))
			assert.Equal(t, th.out, got, "Expected %q but got %q from %q", th.out, got, th.in)
			if th.out != got {
				panic("whoops")
			}
		} else {
			ts = MustParse(th.in)
			got := fmt.Sprintf("%v", ts.In(time.UTC))
			assert.Equal(t, th.out, got, "Expected %q but got %q from %q", th.out, got, th.in)
			if th.out != got {
				panic("whoops")
			}
		}
	}

	// some errors

	assert.Equal(t, true, testDidPanic(`{"ts":"now"}`))

	_, err = ParseAny("138421636711122233311111") // too many digits
	assert.NotEqual(t, nil, err)

	_, err = ParseAny("-1314")
	assert.NotEqual(t, nil, err)

	_, err = ParseAny("2014-13-13 08:20:13,787") // month 13 doesn't exist so error
	assert.NotEqual(t, nil, err)
}

func testDidPanic(datestr string) (paniced bool) {
	defer func() {
		if r := recover(); r != nil {
			paniced = true
		}
	}()
	MustParse(datestr)
	return false
}

func TestPStruct(t *testing.T) {

	denverLoc, err := time.LoadLocation("America/Denver")
	assert.Equal(t, nil, err)

	p := newParser("08.21.71", denverLoc)

	p.setMonth()
	assert.Equal(t, 0, p.moi)
	p.setDay()
	assert.Equal(t, 0, p.dayi)
	p.set(-1, "not")
	p.set(15, "not")
	assert.Equal(t, "08.21.71", p.datestr)
	assert.Equal(t, "08.21.71", string(p.format))
	assert.True(t, len(p.ds()) > 0)
	assert.True(t, len(p.ts()) > 0)
}

var testParseErrors = []dateTest{
	{in: "3", err: true},
	{in: `{"hello"}`, err: true},
	{in: "2009-15-12T22:15Z", err: true},
	{in: "5,000-9,999", err: true},
	{in: "xyzq-baad"},
	{in: "oct.-7-1970", err: true},
	{in: "septe. 7, 1970", err: true},
	{in: "SeptemberRR 7th, 1970", err: true},
	{in: "29-06-2016", err: true},
	// this is just testing the empty space up front
	{in: " 2018-01-02 17:08:09 -07:00", err: true},
}

func TestParseErrors(t *testing.T) {
	for _, th := range testParseErrors {
		v, err := ParseAny(th.in)
		assert.NotEqual(t, nil, err, "%v for %v", v, th.in)

		v, err = ParseAny(th.in, RetryAmbiguousDateWithSwap(true))
		assert.NotEqual(t, nil, err, "%v for %v", v, th.in)
	}
}

func TestParseLayout(t *testing.T) {

	time.Local = time.UTC
	// These tests are verifying that the layout returned by ParseFormat
	// are correct.   Above tests correct parsing, this tests correct
	// re-usable formatting string
	var testParseFormat = []dateTest{
		// errors
		{in: "3", err: true},
		{in: `{"hello"}`, err: true},
		{in: "2009-15-12T22:15Z", err: true},
		{in: "5,000-9,999", err: true},
		// This 3 digit TZ offset (should be 2 or 4?  is 3 a thing?)
		{in: "2019-05-29T08:41-047", err: true},
		//
		{in: "06/May/2008 15:04:05 -0700", out: "02/Jan/2006 15:04:05 -0700"},
		{in: "06/May/2008:15:04:05 -0700", out: "02/Jan/2006:15:04:05 -0700"},
		{in: "14 May 2019 19:11:40.164", out: "02 Jan 2006 15:04:05.000"},
		{in: "171113 14:14:20", out: "060102 15:04:05"},

		{in: "oct 7, 1970", out: "Jan 2, 2006"},
		{in: "sept. 7, 1970", out: "Jan. 2, 2006"},
		{in: "May 05, 2015, 05:05:07", out: "Jan 02, 2006, 15:04:05"},
		// 03 February 2013
		{in: "03 February 2013", out: "02 January 2006"},
		// 13:31:51.999 -07:00 MST
		//   yyyy-mm-dd hh:mm:ss +00:00
		{in: "2012-08-03 18:31:59 +00:00", out: "2006-01-02 15:04:05 -07:00"},
		//   yyyy-mm-dd hh:mm:ss +0000 TZ
		// Golang Native Format = "2006-01-02 15:04:05.999999999 -0700 MST"
		{in: "2012-08-03 18:31:59 +0000 UTC", out: "2006-01-02 15:04:05 -0700 MST"},
		//   yyyy-mm-dd hh:mm:ss TZ
		{in: "2012-08-03 18:31:59 UTC", out: "2006-01-02 15:04:05 MST"},
		{in: "2012-08-03 18:31:59 CEST", out: "2006-01-02 15:04:05 MST"},
		//   yyyy-mm-ddThh:mm:ss-07:00
		{in: "2009-08-12T22:15:09-07:00", out: "2006-01-02T15:04:05-07:00"},
		//   yyyy-mm-ddThh:mm:ss-0700
		{in: "2009-08-12T22:15:09-0700", out: "2006-01-02T15:04:05-0700"},
		//   yyyy-mm-ddThh:mm:ssZ
		{in: "2009-08-12T22:15Z", out: "2006-01-02T15:04Z"},
	}

	for _, th := range testParseFormat {
		l, err := ParseFormat(th.in)
		if th.err {
			assert.NotEqual(t, nil, err)
		} else {
			assert.Equal(t, nil, err)
			assert.Equal(t, th.out, l, "for in=%v", th.in)
		}
	}
}

var testParseStrict = []dateTest{
	//   dd-mon-yy  13-Feb-03
	{in: "03-03-14"},
	//   mm.dd.yyyy
	{in: "3.3.2014"},
	//   mm.dd.yy
	{in: "08.09.71"},
	//  mm/dd/yyyy
	{in: "3/5/2014"},
	//  mm/dd/yy
	{in: "08/08/71"},
	{in: "8/8/71"},
	//  mm/dd/yy hh:mm:ss
	{in: "04/02/2014 04:08:09"},
	{in: "4/2/2014 04:08:09"},
}

func TestParseStrict(t *testing.T) {

	for _, th := range testParseStrict {
		_, err := ParseStrict(th.in)
		assert.NotEqual(t, nil, err)
	}

	_, err := ParseStrict(`{"hello"}`)
	assert.NotEqual(t, nil, err)

	_, err = ParseStrict("2009-08-12T22:15Z")
	assert.Equal(t, nil, err)
}

// Lets test to see how this performs using different Timezones/Locations
// Also of note, try changing your server/machine timezones and repeat
//
// !!!!! The time-zone of local machine effects the results!
// https://play.golang.org/p/IDHRalIyXh
// https://github.com/golang/go/issues/18012
func TestInLocation(t *testing.T) {

	denverLoc, err := time.LoadLocation("America/Denver")
	assert.Equal(t, nil, err)

	// Start out with time.UTC
	time.Local = time.UTC

	// Just normal parse to test out zone/offset
	ts := MustParse("2013-02-01 00:00:00")
	zone, offset := ts.Zone()
	assert.Equal(t, 0, offset, "Should have found offset = 0 %v", offset)
	assert.Equal(t, "UTC", zone, "Should have found zone = UTC %v", zone)
	assert.Equal(t, "2013-02-01 00:00:00 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))

	// Now lets set to denver (MST/MDT) and re-parse the same time string
	// and since no timezone info in string, we expect same result
	time.Local = denverLoc
	ts = MustParse("2013-02-01 00:00:00")
	zone, offset = ts.Zone()
	assert.Equal(t, 0, offset, "Should have found offset = 0 %v", offset)
	assert.Equal(t, "UTC", zone, "Should have found zone = UTC %v", zone)
	assert.Equal(t, "2013-02-01 00:00:00 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))

	ts = MustParse("Tue, 5 Jul 2017 16:28:13 -0700 (MST)")
	assert.Equal(t, "2017-07-05 23:28:13 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))

	// Now we are going to use ParseIn() and see that it gives different answer
	// with different zone, offset
	time.Local = nil
	ts, err = ParseIn("2013-02-01 00:00:00", denverLoc)
	assert.Equal(t, nil, err)
	zone, offset = ts.Zone()
	assert.Equal(t, -25200, offset, "Should have found offset = -25200 %v  %v", offset, denverLoc)
	assert.Equal(t, "MST", zone, "Should have found zone = MST %v", zone)
	assert.Equal(t, "2013-02-01 07:00:00 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))

	ts, err = ParseIn("18 January 2018", denverLoc)
	assert.Equal(t, nil, err)
	zone, offset = ts.Zone()
	assert.Equal(t, -25200, offset, "Should have found offset = 0 %v", offset)
	assert.Equal(t, "MST", zone, "Should have found zone = UTC %v", zone)
	assert.Equal(t, "2018-01-18 07:00:00 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))

	// Now we are going to use ParseLocal() and see that it gives same
	// answer as ParseIn when we have time.Local set to a location
	time.Local = denverLoc
	ts, err = ParseLocal("2013-02-01 00:00:00")
	assert.Equal(t, nil, err)
	zone, offset = ts.Zone()
	assert.Equal(t, -25200, offset, "Should have found offset = -25200 %v  %v", offset, denverLoc)
	assert.Equal(t, "MST", zone, "Should have found zone = MST %v", zone)
	assert.Equal(t, "2013-02-01 07:00:00 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))

	// Lets advance past daylight savings time start
	// use parseIn and see offset/zone has changed to Daylight Savings Equivalents
	ts, err = ParseIn("2013-04-01 00:00:00", denverLoc)
	assert.Equal(t, nil, err)
	zone, offset = ts.Zone()
	assert.Equal(t, -21600, offset, "Should have found offset = -21600 %v  %v", offset, denverLoc)
	assert.Equal(t, "MDT", zone, "Should have found zone = MDT %v", zone)
	assert.Equal(t, "2013-04-01 06:00:00 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))

	// reset to UTC
	time.Local = time.UTC

	//   UnixDate    = "Mon Jan _2 15:04:05 MST 2006"
	ts = MustParse("Mon Jan  2 15:04:05 MST 2006")

	_, offset = ts.Zone()
	assert.Equal(t, 0, offset, "Should have found offset = 0 %v", offset)
	assert.Equal(t, "2006-01-02 15:04:05 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))

	// Now lets set to denver(mst/mdt)
	time.Local = denverLoc
	ts = MustParse("Mon Jan  2 15:04:05 MST 2006")

	// this time is different from one above parsed with time.Local set to UTC
	_, offset = ts.Zone()
	assert.Equal(t, -25200, offset, "Should have found offset = -25200 %v", offset)
	assert.Equal(t, "2006-01-02 22:04:05 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))

	// Now Reset To UTC
	time.Local = time.UTC

	// RFC850    = "Monday, 02-Jan-06 15:04:05 MST"
	ts = MustParse("Monday, 02-Jan-06 15:04:05 MST")
	_, offset = ts.Zone()
	assert.Equal(t, 0, offset, "Should have found offset = 0 %v", offset)
	assert.Equal(t, "2006-01-02 15:04:05 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))

	// Now lets set to denver
	time.Local = denverLoc
	ts = MustParse("Monday, 02-Jan-06 15:04:05 MST")
	_, offset = ts.Zone()
	assert.NotEqual(t, 0, offset, "Should have found offset %v", offset)
	assert.Equal(t, "2006-01-02 22:04:05 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))

	// Now some errors
	zeroTime := time.Time{}.Unix()
	ts, err = ParseIn("INVALID", denverLoc)
	assert.Equal(t, zeroTime, ts.Unix())
	assert.NotEqual(t, nil, err)

	ts, err = ParseLocal("INVALID")
	assert.Equal(t, zeroTime, ts.Unix())
	assert.NotEqual(t, nil, err)
}

func TestPreferMonthFirst(t *testing.T) {
	// default case is true
	ts, err := ParseAny("04/02/2014 04:08:09 +0000 UTC")
	assert.Equal(t, nil, err)
	assert.Equal(t, "2014-04-02 04:08:09 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))

	preferMonthFirstTrue := PreferMonthFirst(true)
	ts, err = ParseAny("04/02/2014 04:08:09 +0000 UTC", preferMonthFirstTrue)
	assert.Equal(t, nil, err)
	assert.Equal(t, "2014-04-02 04:08:09 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))

	// allows the day to be preferred before the month, when completely ambiguous
	preferMonthFirstFalse := PreferMonthFirst(false)
	ts, err = ParseAny("04/02/2014 04:08:09 +0000 UTC", preferMonthFirstFalse)
	assert.Equal(t, nil, err)
	assert.Equal(t, "2014-02-04 04:08:09 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))
}

func TestRetryAmbiguousDateWithSwap(t *testing.T) {
	// default is false
	_, err := ParseAny("13/02/2014 04:08:09 +0000 UTC")
	assert.NotEqual(t, nil, err)

	// will fail error if the month preference cannot work due to the value being larger than 12
	retryAmbiguousDateWithSwapFalse := RetryAmbiguousDateWithSwap(false)
	_, err = ParseAny("13/02/2014 04:08:09 +0000 UTC", retryAmbiguousDateWithSwapFalse)
	assert.NotEqual(t, nil, err)

	// will retry with the other month preference if this error is detected
	retryAmbiguousDateWithSwapTrue := RetryAmbiguousDateWithSwap(true)
	ts, err := ParseAny("13/02/2014 04:08:09 +0000 UTC", retryAmbiguousDateWithSwapTrue)
	assert.Equal(t, nil, err)
	assert.Equal(t, "2014-02-13 04:08:09 +0000 UTC", fmt.Sprintf("%v", ts.In(time.UTC)))
}

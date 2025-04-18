DateParse CLI
----------------------

Simple CLI to test out dateparse.


```sh

# Since this date string has no timezone/offset it's more affected by
# which method you use to parse

$ dateparse --timezone="America/Denver" "2017-07-19 03:21:00"

Your Current time.Local zone is PDT

Layout String: dateparse.ParseFormat() => 2006-01-02 15:04:05

Your Using time.Local set to location=America/Denver MDT 

+-------------+---------------------------+-------------------------------+-------------------------------------+
| method      | Zone Source               | Parsed                        | Parsed: t.In(time.UTC)              |
+-------------+---------------------------+-------------------------------+-------------------------------------+
| ParseAny    | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC day=3 |
| ParseAny    | time.Local = timezone arg | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC day=3 |
| ParseAny    | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC day=3 |
| ParseIn     | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseIn     | time.Local = timezone arg | 2017-07-19 03:21:00 -0600 MDT | 2017-07-19 09:21:00 +0000 UTC       |
| ParseIn     | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseLocal  | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseLocal  | time.Local = timezone arg | 2017-07-19 03:21:00 -0600 MDT | 2017-07-19 09:21:00 +0000 UTC       |
| ParseLocal  | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseStrict | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseStrict | time.Local = timezone arg | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseStrict | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
+-------------+---------------------------+-------------------------------+-------------------------------------+


#  Note on this one that the outputed zone is always UTC/0 offset as opposed to above

$ dateparse --timezone="America/Denver" "2017-07-19 03:21:51+00:00"

Your Current time.Local zone is PDT

Layout String: dateparse.ParseFormat() => 2006-01-02 15:04:05-07:00

Your Using time.Local set to location=America/Denver MDT 

+-------------+---------------------------+---------------------------------+-------------------------------------+
| method      | Zone Source               | Parsed                          | Parsed: t.In(time.UTC)              |
+-------------+---------------------------+---------------------------------+-------------------------------------+
| ParseAny    | time.Local = nil          | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC day=3 |
| ParseAny    | time.Local = timezone arg | 2017-07-19 03:21:51 +0000 +0000 | 2017-07-19 03:21:51 +0000 UTC day=3 |
| ParseAny    | time.Local = time.UTC     | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC day=3 |
| ParseIn     | time.Local = nil          | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC       |
| ParseIn     | time.Local = timezone arg | 2017-07-19 03:21:51 +0000 +0000 | 2017-07-19 03:21:51 +0000 UTC       |
| ParseIn     | time.Local = time.UTC     | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC       |
| ParseLocal  | time.Local = nil          | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC       |
| ParseLocal  | time.Local = timezone arg | 2017-07-19 03:21:51 +0000 +0000 | 2017-07-19 03:21:51 +0000 UTC       |
| ParseLocal  | time.Local = time.UTC     | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC       |
| ParseStrict | time.Local = nil          | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC       |
| ParseStrict | time.Local = timezone arg | 2017-07-19 03:21:51 +0000 +0000 | 2017-07-19 03:21:51 +0000 UTC       |
| ParseStrict | time.Local = time.UTC     | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC       |
+-------------+---------------------------+---------------------------------+-------------------------------------+


$ dateparse --timezone="America/Denver" "Monday, 19-Jul-17 03:21:00 MDT"

Your Current time.Local zone is PDT

Layout String: dateparse.ParseFormat() => 02-Jan-06 15:04:05 MDT

Your Using time.Local set to location=America/Denver MDT 

+-------------+---------------------------+-------------------------------+-------------------------------------+
| method      | Zone Source               | Parsed                        | Parsed: t.In(time.UTC)              |
+-------------+---------------------------+-------------------------------+-------------------------------------+
| ParseStrict | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseStrict | time.Local = timezone arg | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseStrict | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseAny    | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC day=3 |
| ParseAny    | time.Local = timezone arg | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC day=3 |
| ParseAny    | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC day=3 |
| ParseIn     | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseIn     | time.Local = timezone arg | 2017-07-19 03:21:00 -0600 MDT | 2017-07-19 09:21:00 +0000 UTC       |
| ParseIn     | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseLocal  | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseLocal  | time.Local = timezone arg | 2017-07-19 03:21:00 -0600 MDT | 2017-07-19 09:21:00 +0000 UTC       |
| ParseLocal  | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
+-------------+---------------------------+-------------------------------+-------------------------------------+



# pass in a wrong timezone "MST" (should be MDT)
$ dateparse --timezone="America/Denver" "Monday, 19-Jul-17 03:21:00 MST"

Your Current time.Local zone is PDT

Layout String: dateparse.ParseFormat() => 02-Jan-06 15:04:05 MST

Your Using time.Local set to location=America/Denver MDT 

+-------------+---------------------------+-------------------------------+-------------------------------------+
| method      | Zone Source               | Parsed                        | Parsed: t.In(time.UTC)              |
+-------------+---------------------------+-------------------------------+-------------------------------------+
| ParseAny    | time.Local = nil          | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC day=3 |
| ParseAny    | time.Local = timezone arg | 2017-07-19 04:21:00 -0600 MDT | 2017-07-19 10:21:00 +0000 UTC day=3 |
| ParseAny    | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC day=3 |
| ParseIn     | time.Local = nil          | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC       |
| ParseIn     | time.Local = timezone arg | 2017-07-19 04:21:00 -0600 MDT | 2017-07-19 10:21:00 +0000 UTC       |
| ParseIn     | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC       |
| ParseLocal  | time.Local = nil          | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC       |
| ParseLocal  | time.Local = timezone arg | 2017-07-19 04:21:00 -0600 MDT | 2017-07-19 10:21:00 +0000 UTC       |
| ParseLocal  | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC       |
| ParseStrict | time.Local = nil          | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC       |
| ParseStrict | time.Local = timezone arg | 2017-07-19 04:21:00 -0600 MDT | 2017-07-19 10:21:00 +0000 UTC       |
| ParseStrict | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC       |
+-------------+---------------------------+-------------------------------+-------------------------------------+



# note, we are using America/New_York which doesn't recognize MDT so essentially ignores it
$ dateparse --timezone="America/New_York" "Monday, 19-Jul-17 03:21:00 MDT"

Your Current time.Local zone is PDT

Layout String: dateparse.ParseFormat() => 02-Jan-06 15:04:05 MDT

Your Using time.Local set to location=America/New_York EDT 

+-------------+---------------------------+-------------------------------+-------------------------------------+
| method      | Zone Source               | Parsed                        | Parsed: t.In(time.UTC)              |
+-------------+---------------------------+-------------------------------+-------------------------------------+
| ParseAny    | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC day=3 |
| ParseAny    | time.Local = timezone arg | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC day=3 |
| ParseAny    | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC day=3 |
| ParseIn     | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseIn     | time.Local = timezone arg | 2017-07-19 03:21:00 -0400 EDT | 2017-07-19 07:21:00 +0000 UTC       |
| ParseIn     | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseLocal  | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseLocal  | time.Local = timezone arg | 2017-07-19 03:21:00 -0400 EDT | 2017-07-19 07:21:00 +0000 UTC       |
| ParseLocal  | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseStrict | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseStrict | time.Local = timezone arg | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
| ParseStrict | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC       |
+-------------+---------------------------+-------------------------------+-------------------------------------+

$ dateparse --timezone="America/New_York" "03/03/2017"

Your Current time.Local zone is PDT

Layout String: dateparse.ParseFormat() => 01/02/2006

Your Using time.Local set to location=America/New_York EDT 

+-------------+---------------------------+----------------------------------------------------+----------------------------------------------------+
| method      | Zone Source               | Parsed                                             | Parsed: t.In(time.UTC)                             |
+-------------+---------------------------+----------------------------------------------------+----------------------------------------------------+
| ParseIn     | time.Local = nil          | 2017-03-03 00:00:00 +0000 UTC                      | 2017-03-03 00:00:00 +0000 UTC                      |
| ParseIn     | time.Local = timezone arg | 2017-03-03 00:00:00 -0500 EST                      | 2017-03-03 05:00:00 +0000 UTC                      |
| ParseIn     | time.Local = time.UTC     | 2017-03-03 00:00:00 +0000 UTC                      | 2017-03-03 00:00:00 +0000 UTC                      |
| ParseLocal  | time.Local = nil          | 2017-03-03 00:00:00 +0000 UTC                      | 2017-03-03 00:00:00 +0000 UTC                      |
| ParseLocal  | time.Local = timezone arg | 2017-03-03 00:00:00 -0500 EST                      | 2017-03-03 05:00:00 +0000 UTC                      |
| ParseLocal  | time.Local = time.UTC     | 2017-03-03 00:00:00 +0000 UTC                      | 2017-03-03 00:00:00 +0000 UTC                      |
| ParseStrict | time.Local = nil          | This date has ambiguous mm/dd vs dd/mm type format | This date has ambiguous mm/dd vs dd/mm type format |
| ParseStrict | time.Local = timezone arg | This date has ambiguous mm/dd vs dd/mm type format | This date has ambiguous mm/dd vs dd/mm type format |
| ParseStrict | time.Local = time.UTC     | This date has ambiguous mm/dd vs dd/mm type format | This date has ambiguous mm/dd vs dd/mm type format |
| ParseAny    | time.Local = nil          | 2017-03-03 00:00:00 +0000 UTC                      | 2017-03-03 00:00:00 +0000 UTC day=5                |
| ParseAny    | time.Local = timezone arg | 2017-03-03 00:00:00 +0000 UTC                      | 2017-03-03 00:00:00 +0000 UTC day=5                |
| ParseAny    | time.Local = time.UTC     | 2017-03-03 00:00:00 +0000 UTC                      | 2017-03-03 00:00:00 +0000 UTC day=5                |
+-------------+---------------------------+----------------------------------------------------+----------------------------------------------------+

# Automatically retry date formats that are ambiguous mm/dd vs dd/mm
$ ./dateparse --retry-ambiguous "28.09.2024"

Your Current time.Local zone is MDT

Layout String: dateparse.ParseFormat() => 02.01.2006

+-------------+-----------------------+----------------------------------------------------+----------------------------------------------------+
| method      | Zone Source           | Parsed                                             | Parsed: t.In(time.UTC)                             |
+-------------+-----------------------+----------------------------------------------------+----------------------------------------------------+
| ParseIn     | time.Local = nil      | 2024-09-28 00:00:00 +0000 UTC                      | 2024-09-28 00:00:00 +0000 UTC                      |
| ParseIn     | time.Local = time.UTC | 2024-09-28 00:00:00 +0000 UTC                      | 2024-09-28 00:00:00 +0000 UTC                      |
| ParseLocal  | time.Local = nil      | 2024-09-28 00:00:00 +0000 UTC                      | 2024-09-28 00:00:00 +0000 UTC                      |
| ParseLocal  | time.Local = time.UTC | 2024-09-28 00:00:00 +0000 UTC                      | 2024-09-28 00:00:00 +0000 UTC                      |
| ParseStrict | time.Local = nil      | this date has ambiguous mm/dd vs dd/mm type format | this date has ambiguous mm/dd vs dd/mm type format |
| ParseStrict | time.Local = time.UTC | this date has ambiguous mm/dd vs dd/mm type format | this date has ambiguous mm/dd vs dd/mm type format |
| ParseAny    | time.Local = nil      | 2024-09-28 00:00:00 +0000 UTC                      | 2024-09-28 00:00:00 +0000 UTC day=6                |
| ParseAny    | time.Local = time.UTC | 2024-09-28 00:00:00 +0000 UTC                      | 2024-09-28 00:00:00 +0000 UTC day=6                |
+-------------+-----------------------+----------------------------------------------------+----------------------------------------------------+

# Force dates to be interpreted as day-first instead of month-first
$ ./dateparse --prefer-day-first "28.09.2024"

Your Current time.Local zone is MDT

Layout String: dateparse.ParseFormat() => 02.01.2006

+-------------+-----------------------+----------------------------------------------------+----------------------------------------------------+
| method      | Zone Source           | Parsed                                             | Parsed: t.In(time.UTC)                             |
+-------------+-----------------------+----------------------------------------------------+----------------------------------------------------+
| ParseAny    | time.Local = nil      | 2024-09-28 00:00:00 +0000 UTC                      | 2024-09-28 00:00:00 +0000 UTC day=6                |
| ParseAny    | time.Local = time.UTC | 2024-09-28 00:00:00 +0000 UTC                      | 2024-09-28 00:00:00 +0000 UTC day=6                |
| ParseIn     | time.Local = nil      | 2024-09-28 00:00:00 +0000 UTC                      | 2024-09-28 00:00:00 +0000 UTC                      |
| ParseIn     | time.Local = time.UTC | 2024-09-28 00:00:00 +0000 UTC                      | 2024-09-28 00:00:00 +0000 UTC                      |
| ParseLocal  | time.Local = nil      | 2024-09-28 00:00:00 +0000 UTC                      | 2024-09-28 00:00:00 +0000 UTC                      |
| ParseLocal  | time.Local = time.UTC | 2024-09-28 00:00:00 +0000 UTC                      | 2024-09-28 00:00:00 +0000 UTC                      |
| ParseStrict | time.Local = nil      | this date has ambiguous mm/dd vs dd/mm type format | this date has ambiguous mm/dd vs dd/mm type format |
| ParseStrict | time.Local = time.UTC | this date has ambiguous mm/dd vs dd/mm type format | this date has ambiguous mm/dd vs dd/mm type format |
+-------------+-----------------------+----------------------------------------------------+----------------------------------------------------+

```
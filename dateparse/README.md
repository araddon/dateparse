DateParse CLI
----------------------

Simple CLI to test out dateparse.


```sh

# Since this date string has no timezone/offset so is more effected by
# which method you use to parse

$ dateparse --timezone="America/Denver" "2017-07-19 03:21:00"

Your Current time.Local zone is PDT

+------------+---------------------------+-------------------------------+-------------------------------+
| method     | Zone Source               | Parsed                        | Parsed: t.In(time.UTC)        |
+------------+---------------------------+-------------------------------+-------------------------------+
| ParseAny   | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC |
| ParseAny   | time.Local = timezone arg | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC |
| ParseAny   | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC |
| ParseIn    | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC |
| ParseIn    | time.Local = timezone arg | 2017-07-19 03:21:00 -0600 MDT | 2017-07-19 09:21:00 +0000 UTC |
| ParseIn    | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC |
| ParseLocal | time.Local = nil          | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC |
| ParseLocal | time.Local = timezone arg | 2017-07-19 03:21:00 -0600 MDT | 2017-07-19 09:21:00 +0000 UTC |
| ParseLocal | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 UTC | 2017-07-19 03:21:00 +0000 UTC |
+------------+---------------------------+-------------------------------+-------------------------------+

#  Note on this one that the outputed zone is always UTC/0 offset as opposed to above

$ dateparse --timezone="America/Denver" "2017-07-19 03:21:51+00:00"

Your Current time.Local zone is PDT

+------------+---------------------------+---------------------------------+-------------------------------+
| method     | Zone Source               | Parsed                          | Parsed: t.In(time.UTC)        |
+------------+---------------------------+---------------------------------+-------------------------------+
| ParseAny   | time.Local = nil          | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC |
| ParseAny   | time.Local = timezone arg | 2017-07-19 03:21:51 +0000 +0000 | 2017-07-19 03:21:51 +0000 UTC |
| ParseAny   | time.Local = time.UTC     | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC |
| ParseIn    | time.Local = nil          | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC |
| ParseIn    | time.Local = timezone arg | 2017-07-19 03:21:51 +0000 +0000 | 2017-07-19 03:21:51 +0000 UTC |
| ParseIn    | time.Local = time.UTC     | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC |
| ParseLocal | time.Local = nil          | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC |
| ParseLocal | time.Local = timezone arg | 2017-07-19 03:21:51 +0000 +0000 | 2017-07-19 03:21:51 +0000 UTC |
| ParseLocal | time.Local = time.UTC     | 2017-07-19 03:21:51 +0000 UTC   | 2017-07-19 03:21:51 +0000 UTC |
+------------+---------------------------+---------------------------------+-------------------------------+


$ dateparse --timezone="America/Denver" "Monday, 19-Jul-17 03:21:00 MDT"

Your Current time.Local zone is PDT

+------------+---------------------------+-------------------------------+-------------------------------+
| method     | Zone Source               | Parsed                        | Parsed: t.In(time.UTC)        |
+------------+---------------------------+-------------------------------+-------------------------------+
| ParseAny   | time.Local = nil          | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
| ParseAny   | time.Local = timezone arg | 2017-07-19 03:21:00 -0600 MDT | 2017-07-19 09:21:00 +0000 UTC |
| ParseAny   | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
| ParseIn    | time.Local = nil          | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
| ParseIn    | time.Local = timezone arg | 2017-07-19 03:21:00 -0600 MDT | 2017-07-19 09:21:00 +0000 UTC |
| ParseIn    | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
| ParseLocal | time.Local = nil          | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
| ParseLocal | time.Local = timezone arg | 2017-07-19 03:21:00 -0600 MDT | 2017-07-19 09:21:00 +0000 UTC |
| ParseLocal | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
+------------+---------------------------+-------------------------------+-------------------------------+


# pass in a wrong timezone "MST" (should be MDT)
$ dateparse --timezone="America/Denver" "Monday, 19-Jul-17 03:21:00 MST"

Your Current time.Local zone is PDT

+------------+---------------------------+-------------------------------+-------------------------------+
| method     | Zone Source               | Parsed                        | Parsed: t.In(time.UTC)        |
+------------+---------------------------+-------------------------------+-------------------------------+
| ParseAny   | time.Local = nil          | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC |
| ParseAny   | time.Local = timezone arg | 2017-07-19 04:21:00 -0600 MDT | 2017-07-19 10:21:00 +0000 UTC |
| ParseAny   | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC |
| ParseIn    | time.Local = nil          | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC |
| ParseIn    | time.Local = timezone arg | 2017-07-19 04:21:00 -0600 MDT | 2017-07-19 10:21:00 +0000 UTC |
| ParseIn    | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC |
| ParseLocal | time.Local = nil          | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC |
| ParseLocal | time.Local = timezone arg | 2017-07-19 04:21:00 -0600 MDT | 2017-07-19 10:21:00 +0000 UTC |
| ParseLocal | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 MST | 2017-07-19 03:21:00 +0000 UTC |
+------------+---------------------------+-------------------------------+-------------------------------+


# note, we are using America/New_York which doesn't recognize MDT so essentially ignores it
$ dateparse --timezone="America/New_York" "Monday, 19-Jul-17 03:21:00 MDT"

Your Current time.Local zone is PDT

+------------+---------------------------+-------------------------------+-------------------------------+
| method     | Zone Source               | Parsed                        | Parsed: t.In(time.UTC)        |
+------------+---------------------------+-------------------------------+-------------------------------+
| ParseAny   | time.Local = nil          | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
| ParseAny   | time.Local = timezone arg | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
| ParseAny   | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
| ParseIn    | time.Local = nil          | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
| ParseIn    | time.Local = timezone arg | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
| ParseIn    | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
| ParseLocal | time.Local = nil          | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
| ParseLocal | time.Local = timezone arg | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
| ParseLocal | time.Local = time.UTC     | 2017-07-19 03:21:00 +0000 MDT | 2017-07-19 03:21:00 +0000 UTC |
+------------+---------------------------+-------------------------------+-------------------------------+


```
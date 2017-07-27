DateParse CLI
----------------------

Simple CLI to test out dateparse.


```sh

# Since this date string has no timezone/offset so is more effected by
# which method you use to parse

$ dateparse --timezone="America/Denver" "2017-07-19 03:21:00"
+------------+---------------------+---------------+----------+-------------------------------+
| method     | Input               | Zone Source   | Timezone | Parsed, and Output as %v      |
+------------+---------------------+---------------+----------+-------------------------------+
| ParseAny   | 2017-07-19 03:21:00 | Local Default | PDT      | 2017-07-19 03:21:00 +0000 UTC |
| ParseAny   | 2017-07-19 03:21:00 | timezone arg  | PDT      | 2017-07-19 03:21:00 +0000 UTC |
| ParseAny   | 2017-07-19 03:21:00 | UTC           | UTC      | 2017-07-19 03:21:00 +0000 UTC |
| ParseIn    | 2017-07-19 03:21:00 | Local Default | PDT      | 2017-07-19 03:21:00 +0000 UTC |
| ParseIn    | 2017-07-19 03:21:00 | timezone arg  | PDT      | 2017-07-19 03:21:00 -0600 MDT |
| ParseIn    | 2017-07-19 03:21:00 | UTC           | UTC      | 2017-07-19 03:21:00 +0000 UTC |
| ParseLocal | 2017-07-19 03:21:00 | Local Default | PDT      | 2017-07-19 03:21:00 +0000 UTC |
| ParseLocal | 2017-07-19 03:21:00 | timezone arg  | PDT      | 2017-07-19 03:21:00 -0600 MDT |
| ParseLocal | 2017-07-19 03:21:00 | UTC           | UTC      | 2017-07-19 03:21:00 +0000 UTC |
+------------+---------------------+---------------+----------+-------------------------------+

#  Note on this one that the outputed zone is always UTC/0 offset as opposed to above

$ dateparse --timezone="America/Denver" "2017-07-19 03:21:51+00:00"
+------------+---------------------------+---------------+----------+---------------------------------+
| method     | Input                     | Zone Source   | Timezone | Parsed, and Output as %v        |
+------------+---------------------------+---------------+----------+---------------------------------+
| ParseAny   | 2017-07-19 03:21:51+00:00 | Local Default | PDT      | 2017-07-19 03:21:51 +0000 UTC   |
| ParseAny   | 2017-07-19 03:21:51+00:00 | timezone arg  | PDT      | 2017-07-19 03:21:51 +0000 UTC   |
| ParseAny   | 2017-07-19 03:21:51+00:00 | UTC           | UTC      | 2017-07-19 03:21:51 +0000 UTC   |
| ParseIn    | 2017-07-19 03:21:51+00:00 | Local Default | PDT      | 2017-07-19 03:21:51 +0000 UTC   |
| ParseIn    | 2017-07-19 03:21:51+00:00 | timezone arg  | PDT      | 2017-07-19 03:21:51 +0000 +0000 |
| ParseIn    | 2017-07-19 03:21:51+00:00 | UTC           | UTC      | 2017-07-19 03:21:51 +0000 UTC   |
| ParseLocal | 2017-07-19 03:21:51+00:00 | Local Default | PDT      | 2017-07-19 03:21:51 +0000 UTC   |
| ParseLocal | 2017-07-19 03:21:51+00:00 | timezone arg  | PDT      | 2017-07-19 03:21:51 +0000 +0000 |
| ParseLocal | 2017-07-19 03:21:51+00:00 | UTC           | UTC      | 2017-07-19 03:21:51 +0000 UTC   |
+------------+---------------------------+---------------+----------+---------------------------------+


```
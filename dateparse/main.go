package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/scylladb/termtables"
	"github.com/araddon/dateparse"
)

var (
	timezone = ""
	datestr  = ""
)

func main() {
	flag.StringVar(&timezone, "timezone", "", "Timezone aka `America/Los_Angeles` formatted time-zone")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Println(`Must pass a time, and optional location:

		./dateparse "2009-08-12T22:15:09.99Z"

		./dateparse --timezone="America/Denver" "2017-07-19 03:21:51+00:00"
		`)
		return
	}

	datestr = flag.Args()[0]

	layout, err := dateparse.ParseFormat(datestr, true)
	if err != nil {
		fatal(err)
	}

	zonename, _ := time.Now().In(time.Local).Zone()
	fmt.Printf("\nYour Current time.Local zone is %v\n", zonename)
	fmt.Printf("\nLayout String: dateparse.ParseFormat() => %v\n", layout)
	var loc *time.Location
	if timezone != "" {
		// NOTE:  This is very, very important to understand
		// time-parsing in go
		l, err := time.LoadLocation(timezone)
		if err != nil {
			fatal(err)
		}
		loc = l
		zonename, _ := time.Now().In(l).Zone()
		fmt.Printf("\nYour Using time.Local set to location=%s %v \n", timezone, zonename)
	}
	fmt.Printf("\n")

	table := termtables.CreateTable()

	table.AddHeaders("method", "Zone Source", "Parsed", "Parsed: t.In(time.UTC)")

	parsers := map[string]parser{
		"ParseAny":    parseAny,
		"ParseIn":     parseIn,
		"ParseLocal":  parseLocal,
		"ParseStrict": parseStrict,
	}

	for name, parser := range parsers {
		time.Local = nil
		table.AddRow(name, "time.Local = nil", parser(datestr, nil, false), parser(datestr, nil, true))
		if timezone != "" {
			time.Local = loc
			table.AddRow(name, "time.Local = timezone arg", parser(datestr, loc, false), parser(datestr, loc, true))
		}
		time.Local = time.UTC
		table.AddRow(name, "time.Local = time.UTC", parser(datestr, time.UTC, false), parser(datestr, time.UTC, true))
	}

	fmt.Println(table.Render())
}

type parser func(datestr string, loc *time.Location, utc bool) string

func parseLocal(datestr string, loc *time.Location, utc bool) string {
	time.Local = loc
	t, err := dateparse.ParseLocal(datestr, true)
	if err != nil {
		return err.Error()
	}
	if utc {
		return t.In(time.UTC).String()
	}
	return t.String()
}

func parseIn(datestr string, loc *time.Location, utc bool) string {
	t, err := dateparse.ParseIn(datestr, loc, true)
	if err != nil {
		return err.Error()
	}
	if utc {
		return t.In(time.UTC).String()
	}
	return t.String()
}

func parseAny(datestr string, loc *time.Location, utc bool) string {
	t, err := dateparse.ParseAny(datestr, true)
	if err != nil {
		return err.Error()
	}
	if utc {
		return fmt.Sprintf("%s day=%d", t.In(time.UTC), t.In(time.UTC).Weekday())
	}
	return t.String()
}

func parseStrict(datestr string, loc *time.Location, utc bool) string {
	t, err := dateparse.ParseStrict(datestr, true)
	if err != nil {
		return err.Error()
	}
	if utc {
		return t.In(time.UTC).String()
	}
	return t.String()
}

func fatal(err error) {
	fmt.Printf("fatal: %s\n", err)
	os.Exit(1)
}

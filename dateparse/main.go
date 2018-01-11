package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/apcera/termtables"
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

	var loc *time.Location
	if timezone != "" {
		// NOTE:  This is very, very important to understand
		// time-parsing in go
		l, err := time.LoadLocation(timezone)
		if err != nil {
			panic(err.Error())
		}
		loc = l
	}

	zonename, _ := time.Now().In(time.Local).Zone()
	fmt.Printf("\nYour Current time.Local zone is %v\n\n", zonename)

	table := termtables.CreateTable()

	table.AddHeaders("method", "Zone Source", "Parsed", "Parsed: t.In(time.UTC)")

	parsers := map[string]parser{
		"ParseAny":   parseAny,
		"ParseIn":    parseIn,
		"ParseLocal": parseLocal,
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

func stuff() (string, string) {
	return "more", "stuff"
}

type parser func(datestr string, loc *time.Location, utc bool) string

func parseLocal(datestr string, loc *time.Location, utc bool) string {
	time.Local = loc
	t, err := dateparse.ParseLocal(datestr)
	if err != nil {
		return err.Error()
	}
	if utc {
		return t.In(time.UTC).String()
	}
	return t.String()
}

func parseIn(datestr string, loc *time.Location, utc bool) string {
	t, err := dateparse.ParseIn(datestr, loc)
	if err != nil {
		return err.Error()
	}
	if utc {
		return t.In(time.UTC).String()
	}
	return t.String()
}

func parseAny(datestr string, loc *time.Location, utc bool) string {
	t, err := dateparse.ParseAny(datestr)
	if err != nil {
		return err.Error()
	}
	if utc {
		return t.In(time.UTC).String()
	}
	return t.String()
}

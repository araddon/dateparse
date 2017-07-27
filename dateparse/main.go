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

	table := termtables.CreateTable()

	table.AddHeaders("method", "Input", "Zone Source", "Timezone", "Parsed, and Output as %v")

	parsers := map[string]parser{
		"ParseAny":   parseAny,
		"ParseIn":    parseIn,
		"ParseLocal": parseLocal,
	}

	zonename, _ := time.Now().In(time.Local).Zone()
	for name, parser := range parsers {
		time.Local = nil
		table.AddRow(name, datestr, "Local Default", zonename, parser(datestr, nil))
		if timezone != "" {
			table.AddRow(name, datestr, "timezone arg", zonename, parser(datestr, loc))
		}
		time.Local = time.UTC
		table.AddRow(name, datestr, "UTC", "UTC", parser(datestr, time.UTC))
	}

	fmt.Println(table.Render())
}

type parser func(datestr string, loc *time.Location) string

func parseLocal(datestr string, loc *time.Location) string {
	time.Local = loc
	t, _ := dateparse.ParseLocal(datestr)
	return t.String()
}

func parseIn(datestr string, loc *time.Location) string {
	t, _ := dateparse.ParseIn(datestr, loc)
	return t.String()
}

func parseAny(datestr string, loc *time.Location) string {
	t, _ := dateparse.ParseAny(datestr)
	return t.String()
}

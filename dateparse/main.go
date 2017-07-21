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
	flag.StringVar(&timezone, "timezone", "UTC", "Timezone aka `America/Los_Angeles` formatted time-zone")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Println(`Must pass   ./dateparse "2009-08-12T22:15:09.99Z"`)
		return
	}
	datestr = flag.Args()[0]

	table := termtables.CreateTable()

	table.AddHeaders("Input", "Timezone", "Parsed, and Output as %v")

	zonename, _ := time.Now().In(time.Local).Zone()

	table.AddRow(datestr, fmt.Sprintf("%v", zonename), fmt.Sprintf("%v", dateparse.MustParse(datestr)))

	if timezone != "" {
		// NOTE:  This is very, very important to understand
		// time-parsing in go
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			panic(err.Error())
		}
		time.Local = loc
		table.AddRow(datestr, fmt.Sprintf("%v", timezone), fmt.Sprintf("%v", dateparse.MustParse(datestr)))
	}

	time.Local = time.UTC
	table.AddRow(datestr, "UTC", fmt.Sprintf("%v", dateparse.MustParse(datestr)))

	fmt.Println(table.Render())
}

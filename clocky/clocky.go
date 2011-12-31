package clocky

import (
	"fmt"
	"log"
	"http"
	"template"
	"time"
)

const LAT = 37.79
const LNG = -122.42

var tmpl = template.Must(template.New("page").Parse(PAGE))

// Pacify converts utc to US Pacific time (2007 rules).  We have to do
// this by hand because Go r60 doesn't have any real time zone
// support.  Things are better in Go 1.
func Pacify(utc *time.Time) *time.Time {
	// Find the second Sunday in March and the first Sunday in
	// November.  The second Sunday in March is the first Sunday
	// that is or follows March 8.
	mar8, _ := time.Parse("2006-01-02 15", fmt.Sprintf("%d-03-08 10", utc.Year))
	dstStart := mar8.Seconds() + int64((7-weekday(mar8))%7*86400)
	nov1, _ := time.Parse("2006-01-02 15", fmt.Sprintf("%d-11-01 09", utc.Year))
	dstEnd := nov1.Seconds() + int64((7-weekday(nov1))%7*86400)

	offset, zone := -8*3600, "PST"
	if utc.Seconds() >= dstStart && utc.Seconds() < dstEnd {
		offset, zone = -7*3600, "PDT"
	}
	local := time.SecondsToUTC(utc.Seconds() + int64(offset))
	local.ZoneOffset = offset
	local.Zone = zone
	log.Println(local)
	return local
}

var sakamotoTable = []int{0, 0, 3, 2, 5, 0, 3, 5, 1, 4, 6, 2, 4}

// weekday calculates the day of a week using Sakamoto's method.
func weekday(t *time.Time) int {
	y := int(t.Year) // This algorithm won't work for years >= 2**31 anyway.
	if t.Month < 3 {
		y--
	}
	return (y + y/4 - y/100 + y/400 + sakamotoTable[t.Month] + t.Day) % 7
}

func handler(w http.ResponseWriter, r *http.Request) {
	// TODO: Refresh minutely; use JS to fake things.
	w.Header().Set("Refresh", "2")

	now := Pacify(time.UTC())
	d := map[string]map[string]string{
		"Time": map[string]string{
			"Big":     now.Format("3:04"),
			"Small":   now.Format(":05&thinsp;pm"),
			"Date":    now.Format("Sunday, January 2"),
			"Sunrise": "7:24&thinsp;am",
			"Sunset":  "4:52&thinsp;pm",
		},
		"Weather": map[string]string{
			"Temp":     "12°",
			"Forecast": DUMMY_FORECAST,
		},
	}
	tmpl.Execute(w, d)
}

func init() {
	http.HandleFunc("/", handler)
}

const PAGE = `<!DOCTYPE html>
<head>
    <title>Clocky</title>
    <style>
        body { font-size: 32px; margin: 0; }
        div { margin: 4px; }
        .header { font-weight: bold; }
        .smaller { font-size: 61%; }
        .larger { font-size: 164%; }
        .box { position: absolute; }
        .bus { margin: 8px 0 8px 0; }
        .route { font-weight: bold; font-size: 24px; }
        .arrivals { font-size: 20px; }
    </style>
</head>

{{with .Time}}
<div class=box style="width: 350px; height: 130px; top: 18px; left:24px; text-align: center; background-color: #eee">
    <div class=header><span class=larger>{{.Big}}</span>{{.Small}}</div>
    <div class=smaller>{{.Date}}</div>
    <div class=smaller>sunrise {{.Sunrise}}; sunset {{.Sunset}}</div>
</div>
{{end}}

{{with .Weather}}
<div class=box style="width: 400px; top: 175px; left: 24px">
    <div class=header><span class=larger>{{.Temp}}</span> calm, 96%</div>
    <div class=smaller style="text-align: left">{{.Forecast}}</div>
</div>
{{end}}

<div class=box style="width: 300px; left: 475px; top: 10px">
    <div class=bus>
        <div class=route>47 outbound</div>
        <div class=arrivals>11, 30, 50, 68 minutes</div>
    </div>
    <div class=bus>
        <div class=route>49 outbound</div>
        <div class=arrivals>½, 19, 39, 59 minutes</div>
    </div>
    <div class=bus>
        <div class=route>10, 12 outbound</div>
        <div class=arrivals>18 minutes</div>
    </div>
    <div class=bus>
        <div class=route>27 outbound</div>
        <div class=arrivals>Probably never</div>
    </div>
    <div class=bus>
        <div class=route>1 inbound</div>
        <div class=arrivals>6½, 31, 51, 69 minutes</div>
    </div>
    <div class=bus>
        <div class=route>1 outbound</div>
        <div class=arrivals>Now, 41, 59, 79 minutes</div>
    </div>
</div>
`

// km/h, am, pm after number: convert no space or ASCII space to &thinsp;
// line-ending number: change ASCII space to &nbsp;
const DUMMY_FORECAST = `
<div><span class=header>Tonight:</span> Patchy fog after
10&thinsp;pm. Otherwise, mostly cloudy, with a low
around&nbsp;9. Northwest wind around 10&thinsp;km/h becoming
calm.</div>

<div style="margin-top: 8px"><span class=header>Saturday:</span>
Patchy fog before 10&thinsp;am. Otherwise, mostly sunny, with a high
near&nbsp;16. North northeast wind between 10 and 13&thinsp;km/h
becoming calm.</div>`

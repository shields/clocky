package clocky

import (
	"fmt"
	"io"
	"template"
	"time"

	"appengine"

	"solar"
)

const TimeFormat = "3:04&thinsp;pm"

// Weekday calculates the day of a week using Sakamoto's method.
func Weekday(t *time.Time) int {
	data := []int{0, 0, 3, 2, 5, 0, 3, 5, 1, 4, 6, 2, 4}
	y := int(t.Year) // This algorithm won't work for years >= 2**31 anyway.
	if t.Month < 3 {
		y--
	}
	return (y + y/4 - y/100 + y/400 + data[t.Month] + t.Day) % 7
}

// Pacify converts utc to US Pacific time (2007 rules).  We have to do
// this by hand because Go r60 doesn't have any real time zone
// support.  Things are better in Go 1.
func Pacify(utc *time.Time) *time.Time {
	// Find the second Sunday in March and the first Sunday in
	// November.  The second Sunday in March is the first Sunday
	// that is or follows March 8.
	mar8, _ := time.Parse("2006-01-02 15", fmt.Sprintf("%d-03-08 10", utc.Year))
	dstStart := mar8.Seconds() + int64((7-Weekday(mar8))%7*86400)
	nov1, _ := time.Parse("2006-01-02 15", fmt.Sprintf("%d-11-01 09", utc.Year))
	dstEnd := nov1.Seconds() + int64((7-Weekday(nov1))%7*86400)

	offset, zone := -8*3600, "PST"
	if utc.Seconds() >= dstStart && utc.Seconds() < dstEnd {
		offset, zone = -7*3600, "PDT"
	}
	local := time.SecondsToUTC(utc.Seconds() + int64(offset))
	local.ZoneOffset = offset
	local.Zone = zone
	return local
}

func Time(w io.Writer, c appengine.Context) {
	now := Pacify(time.UTC())
	sunrise := Pacify(solar.Rise(now, Lat, Lng))
	sunset := Pacify(solar.Set(now, Lat, Lng))
	sun1 := "sunrise " + sunrise.Format(TimeFormat)
	sun2 := "sunset " + sunset.Format(TimeFormat)
	if sunrise.Seconds() > sunset.Seconds() {
		sun1, sun2 = sun2, sun1
	}
	timeTmpl.Execute(w, map[string]string{
		"Big":   now.Format("3:04"),
		"Small": now.Format(":05&thinsp;pm"),
		"Date":  now.Format("Monday, January 2"),
		"Sun1":  sun1,
		"Sun2":  sun2,
	})
}

var timeTmpl = template.Must(template.New("time").Parse(`
<div class=box style="width: 350px; height: 128px; top: 24px; left:28px; text-align: center; background-color: #eee">
    <div class=header><span class=larger>{{.Big}}</span>{{.Small}}</div>
    <div class=smaller>{{.Date}}</div>
    <div class=smaller>{{.Sun1}}, {{.Sun2}}</div>
</div>
`))

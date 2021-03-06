// Copyright 2011-2012 Michael Shields
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     http://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clocky

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"text/template" // TODO: Switch to Go 1's html/template.

	"appengine"
	"appengine/memcache"
)

// WindChill returns the Celsius wind chill (2001 North American
// formula) for a given air temperature in degrees Celsius and a wind
// speed in km/h.
func WindChill(temp, speed float64) *float64 {
	if temp > 10 || speed <= 4.8 {
		return nil
	}
	chill := 13.12 + 0.6215*temp - 11.37*math.Pow(speed, 0.16) + 0.3965*temp*math.Pow(speed, 0.16)
	return &chill
}

func Cardinal(degrees int) string {
	switch {
	case degrees < 0:
		break
	case degrees < 23:
		return "N"
	case degrees < 68:
		return "NW"
	case degrees < 113:
		return "W"
	case degrees < 158:
		return "SW"
	case degrees < 203:
		return "S"
	case degrees < 248:
		return "SE"
	case degrees < 293:
		return "E"
	case degrees < 338:
		return "NE"
	case degrees < 360:
		return "N"
	}
	return ""
}

func Conditions(w io.Writer, c appengine.Context) {
	item, err := memcache.Get(c, "conditions")
	if err != nil {
		c.Errorf("%s", err)
		return
	}

	var dir string
	var speed, temp, chill *float64

	for _, line := range strings.Split(string(item.Value), "\n") {
		if len(line) != 116 || line[0] == '#' {
			continue
		}
		// FTPC1 is a C-MAN automated buoy near Crissy Field.
		if line[:5] != "FTPC1" {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimSpace(line[40:43])); err != nil || n < 0 || n > 359 {
			c.Errorf("weather: bad wind direction in %q", line)
		} else {
			dir = Cardinal(n)
		}
		if n, err := strconv.ParseFloat(strings.TrimSpace(line[44:49]), 64); err != nil {
			c.Errorf("weather: bad wind speed in %q", line)
		} else {
			n *= 3.6 // m/s to km/h
			speed = &n
		}
		if n, err := strconv.ParseFloat(strings.TrimSpace(line[87:92]), 64); err != nil {
			c.Errorf("weather: bad temp in %q", line)
		} else {
			temp = &n
		}
		break
	}
	if temp != nil && speed != nil {
		chill = WindChill(*temp, *speed)
	}

	io.WriteString(w, `<div class=header>`)
	if temp != nil {
		// Don't round this, since we are using the value
		// directly from the data, not a converted value like
		// wind speed or a derived value like wind chill.
		fmt.Fprintf(w, `<span class=larger>%.1f°</span> `, *temp)
	}
	switch {
	case speed == nil:
		// Output nothing.
	case chill != nil && *chill < *temp-1:
		fmt.Fprintf(w, `wind chill %.1f°`, *chill+0.05)
	case *speed > 1:
		fmt.Fprintf(w,
			" %s wind <span style=\"white-space: nowrap\">%d&thinsp;km/\u2060h</span>",
			dir, int(*speed+0.5))
	default:
		io.WriteString(w, `wind calm`)
	}
	io.WriteString(w, `</div>`)
}

var (
	nbspRegexp   = regexp.MustCompile(` [0-9]+\.`)
	thinspRegexp = regexp.MustCompile(`[0-9] (am|pm|km/h)`)
)

func Forecast(w io.Writer, c appengine.Context) {
	item, err := memcache.Get(c, "forecast")
	if err != nil {
		c.Errorf("%s", err)
		return
	}

	data := struct {
		Data []struct {
			Type       string `xml:"type,attr"`
			TimeLayout []struct {
				LayoutKey      string `xml:"layout-key"`
				StartValidTime []struct {
					PeriodName string `xml:"period-name,attr"`
				} `xml:"start-valid-time"`
			} `xml:"time-layout"`
			Parameters struct {
				WordedForecast struct {
					TimeLayout string   `xml:"time-layout,attr"`
					Text       []string `xml:"text"`
				} `xml:"wordedForecast"`
			} `xml:"parameters"`
		} `xml:"data"`
	}{}
	p := xml.NewDecoder(strings.NewReader(string(item.Value)))
	// NWS serves XML in ISO-8859-1 for no reason; the data is really ASCII.
	p.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}
	if err = p.DecodeElement(&data, nil); err != nil {
		c.Errorf("%s", err)
		return
	}

	io.WriteString(w, `<div class=smaller style="text-align: left">`)
	for _, d := range data.Data {
		if d.Type != "forecast" {
			continue
		}
		var periods []string
		for _, tl := range d.TimeLayout {
			if tl.LayoutKey != d.Parameters.WordedForecast.TimeLayout {
				continue
			}
			for _, svt := range tl.StartValidTime {
				pn := svt.PeriodName
				pn = strings.Replace(pn, " Morning", " morning", -1)
				pn = strings.Replace(pn, " Afternoon", " afternoon", -1)
				pn = strings.Replace(pn, " Night", " night", -1)
				periods = append(periods, pn)
			}
		}
		texts := d.Parameters.WordedForecast.Text
		if len(texts) != len(periods) {
			c.Errorf("weather: len(texts) = %d, len(periods) = %d",
				len(texts), len(periods))
			continue
		}
		if len(texts) > 4 {
			texts = texts[:4]
		}
		for i, text := range texts {
			io.WriteString(w, `<div style="margin-bottom: 8px"><span class=header>`)
			template.HTMLEscape(w, []byte(periods[i]))
			io.WriteString(w, `:</span> `)

			text = strings.Replace(text, "km/h", "km/\u2060h", -1)
			spaceSubs := make(map[int]string)
			matches := nbspRegexp.FindAllStringIndex(text, -1)
			if len(matches) > 0 {
				for i := 0; i < len(matches[0]); i += 2 {
					spaceSubs[matches[0][i]] = "&nbsp;"
				}
			}
			matches = thinspRegexp.FindAllStringIndex(text, -1)
			if len(matches) > 0 {
				for i := 0; i < len(matches[0]); i += 2 {
					spaceSubs[matches[0][i]+1] = `<span style="white-space: nowrap">&thinsp;</span>`
				}
			}
			for i, ch := range text {
				sub, ok := spaceSubs[i]
				if ok {
					io.WriteString(w, sub)
				} else {
					io.WriteString(w, string(ch))
				}
			}

			io.WriteString(w, `</div>`)
		}
	}
	io.WriteString(w, `</div>`)
}

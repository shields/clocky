package clocky

import (
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"template"
	"xml"

	"appengine"
	"appengine/memcache"
)

// WindChill returns the Celsius wind chill (2001 North American
// formula) for a given air temperature in degrees Celsius and a wind
// speed in m/s.
func WindChill(temp, wind float64) float64 {
	if wind < 4.0/3 {
		return temp
	}
	return 13.12 + 0.6215*temp - 13.96*math.Pow(wind, 0.16) + 0.4867*temp*math.Pow(wind, 0.16)
}

func Conditions(w io.Writer, c appengine.Context) {
	item, err := memcache.Get(c, "conditions")
	if err != nil {
		c.Errorf("%s", err)
		return
	}

	var temp, wind float64
	for _, line := range strings.Split(string(item.Value), "\n") {
		if len(line) != 116 || line[0] == '#' {
			continue
		}
		// FTPC1 is a C-MAN automated buoy near Crissy Field.
		if line[:5] != "FTPC1" {
			continue
		}
		wind, err = strconv.Atof64(strings.TrimSpace(line[44:49]))
		if err != nil {
			c.Errorf("weather: bad wind speed in %q", line)
			return
		}
		temp, err = strconv.Atof64(strings.TrimSpace(line[87:92]))
		if err != nil {
			c.Errorf("weather: bad temp in %q", line)
			return
		}
		break
	}
	chill := WindChill(temp, wind)

	io.WriteString(w, `<div class=header>`)
	fmt.Fprintf(w, `<span class=larger>%.1f°</span> `, temp)
	switch {
	case chill < temp-1:
		fmt.Fprintf(w, `wind chill %.1f°`, chill)
	case wind*3.6 > 1:
		fmt.Fprintf(w, "wind %d&thinsp;km/h", int(wind*3.6))
	default:
		io.WriteString(w, "wind calm")
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
			Type       string `xml:"attr"`
			TimeLayout []struct {
				LayoutKey      string `xml:"layout-key"`
				StartValidTime []struct {
					PeriodName string `xml:"attr"`
				}
			}
			Parameters struct {
				WordedForecast struct {
					TimeLayout string   `xml:"attr"`
					Text       []string `xml:"name>text"`
				}
			}
		}
	}{}
	p := xml.NewParser(strings.NewReader(string(item.Value)))
	// NWS serves XML in ISO-8859-1 for no reason; the data is really ASCII.
	p.CharsetReader = func(charset string, input io.Reader) (io.Reader, os.Error) {
		return input, nil
	}
	if err = p.Unmarshal(&data, nil); err != nil {
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

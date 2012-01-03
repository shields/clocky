package clocky

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"template"
	"xml"

	"appengine"
	"appengine/memcache"
)

func newParser(b []byte) (p *xml.Parser) {
	// NWS serves XML in ISO-8859-1 for no reason; the data is really ASCII.
	p = xml.NewParser(strings.NewReader(string(b)))
	p.CharsetReader = func(charset string, input io.Reader) (io.Reader, os.Error) {
		return input, nil
	}
	return p
}

func Conditions(w io.Writer, c appengine.Context) {
	item, err := memcache.Get(c, "conditions")
	if err != nil {
		c.Errorf("%q", err)
		return
	}

	data := struct {
		Temp_c      string
		WindChill_c string
		Wind_Mph    string
	}{}
	p := newParser(item.Value)
	if err = p.Unmarshal(&data, nil); err != nil {
		c.Errorf("%q", err)
		return
	}

	io.WriteString(w, `<div class=header>`)
	if len(data.Temp_c) != 0 {
		io.WriteString(w, `<span class=larger>`)
		template.HTMLEscape(w, []byte(data.Temp_c))
		io.WriteString(w, `°</span> `)
	}
	if len(data.WindChill_c) != 0 {
		io.WriteString(w, `wind chill `)
		template.HTMLEscape(w, []byte(data.WindChill_c))
		io.WriteString(w, `°`)
	} else if data.Wind_Mph != "" {
		mph, err := strconv.Atof64(data.Wind_Mph)
		if err == nil {
			if mph == 0 {
				io.WriteString(w, "wind calm")
			} else {
				fmt.Fprintf(w, "wind %d&thinsp;km/h", int(mph*1.609344))
			}
		}
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
		c.Errorf("%q", err)
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
	p := newParser(item.Value)
	if err = p.Unmarshal(&data, nil); err != nil {
		c.Errorf("%q", err)
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
		if len(texts) > 3 {
			texts = texts[:3]
		}
		for i, text := range texts {
			io.WriteString(w, `<div style="margin-bottom: 8px"><span class=header>`)
			template.HTMLEscape(w, []byte(periods[i]))
			io.WriteString(w, `:</span> `)

			spaceSubs := make(map[int]string)
			matches := nbspRegexp.FindAllStringIndex(text, -1)
			for i := 0; i < len(matches[0]); i += 2 {
				spaceSubs[matches[0][i]] = "&nbsp;"
			}
			matches = thinspRegexp.FindAllStringIndex(text, -1)
			for i := 0; i < len(matches[0]); i += 2 {
				spaceSubs[matches[0][i]+1] = `<span style="white-space: nowrap">&thinsp;</span>`
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

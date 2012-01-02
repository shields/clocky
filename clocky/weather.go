package clocky

import (
	"fmt"
	"io"
	"os"
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

func Conditions(c appengine.Context) map[string]string {
	item, err := memcache.Get(c, "conditions")
	if err != nil {
		c.Errorf("%q", err)
		return nil
	}

	data := struct {
		Temp_c      string
		WindChill_c string
		Wind_Mph    string
	}{}
	p := newParser(item.Value)
	if err = p.Unmarshal(&data, nil); err != nil {
		c.Errorf("%q", err)
		return nil
	}

	result := make(map[string]string)
	if data.Temp_c != "" {
		result["Temp"] = template.HTMLEscapeString(data.Temp_c) + "°"
	}
	if data.WindChill_c != "" {
		result["WindChill"] = "wind chill " + template.HTMLEscapeString(data.WindChill_c) + "°"
	}
	if data.Wind_Mph != "" {
		mph, err := strconv.Atof64(data.Wind_Mph)
		if err == nil {
			result["Wind"] = fmt.Sprintf("wind %d&thinsp;km/h", int(mph*1.609344))
		}
	}
	return result
}

func Forecast(c appengine.Context) map[string]string {
	item, err := memcache.Get(c, "forecast")
	if err != nil {
		c.Errorf("%q", err)
		return nil
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
		return nil
	}

	forecast := ""
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
				periods = append(periods, svt.PeriodName)
			}
		}
		texts := d.Parameters.WordedForecast.Text
		if len(texts) != len(periods) {
			c.Errorf("weather: len(texts) = %d, len(periods) = %d",
				len(texts), len(periods))
			continue
		}
		if len(texts) > 2 {
			texts = texts[:2]
		}
		for i, text := range texts {
			forecast += fmt.Sprintf(
				`<div style="margin-bottom: 8px"><span class=header>%s:</span> %s</div>`,
				template.HTMLEscapeString(periods[i]),
				text)
		}
	}

	return map[string]string{"Forecast": forecast}
}

// km/h, am, pm after number: convert no space or ASCII space to &thinsp;
// line-ending number: change ASCII space to &nbsp;
const dummyForecast = `
<div><span class=header>Tonight:</span> Patchy fog after
10&thinsp;pm. Otherwise, mostly cloudy, with a low
around&nbsp;9. Northwest wind around 10&thinsp;km/h becoming
calm.</div>

<div style="margin-top: 8px"><span class=header>Saturday:</span>
Patchy fog before 10&thinsp;am. Otherwise, mostly sunny, with a high
near&nbsp;16. North northeast wind between 10 and 13&thinsp;km/h
becoming calm.</div>`

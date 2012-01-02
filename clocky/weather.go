package clocky

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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
	cond := struct {
		Temp_c      string
		WindChill_c string
		Wind_Mph    string
	}{}

	p := newParser(item.Value)
	if err = p.Unmarshal(&cond, nil); err != nil {
		c.Errorf("%q", err)
		return nil
	}

	result := make(map[string]string)
	if cond.Temp_c != "" {
		result["Temp"] = cond.Temp_c + "°"
	}
	if cond.WindChill_c != "" {
		result["WindChill"] = "wind chill " + cond.WindChill_c + "°"
	}
	if cond.Wind_Mph != "" {
		mph, err := strconv.Atof64(cond.Wind_Mph)
		if err == nil {
			result["Wind"] = fmt.Sprintf("wind %d&thinsp;km/h", int(mph*1.609344))
		}
	}
	return result
}

func Forecast(c appengine.Context) map[string]string {
	return map[string]string{"Forecast": dummyForecast}
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

package clocky

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"template"
	"time"
	"xml"

	"appengine"
	"appengine/memcache"
)

var nextBusTmpl = template.Must(template.New("nextbus").Parse(nextBusDiv))

type prediction struct {
	EpochTime   int64 `xml:"attr"`
	IsDeparture bool  `xml:"attr"`
}

func (p prediction) ToString() string {
	if p.IsDeparture {
		return Pacify(time.SecondsToUTC(p.EpochTime)).Format(TimeFormat)
	}
	s := p.EpochTime - time.Seconds()
	if s < 60 {
		return "now"
	}
	result := strconv.Itoa64(s / 60)
	if s < 600 && s%60 >= 30 {
		result += "½"
	}
	return result
}

func NextBus(w io.Writer, c appengine.Context) {
	item, err := memcache.Get(c, "nextbus")
	if err != nil {
		c.Errorf("%q", err)
		return
	}

	data := struct {
		Predictions []struct {
			RouteTag  string `xml:"attr"`
			Direction []struct {
				Title      string `xml:"attr"`
				Prediction []prediction
			}
			Message []struct {
				Text string `xml:"attr"`
			}
		}
	}{}
	if err := xml.Unmarshal(strings.NewReader(string(item.Value)), &data); err != nil {
		c.Errorf("%q", err)
		return
	}

	result := `<div class=box style="width: 300px; top: 66px; left: 475px">`
	for _, p := range data.Predictions {
		for _, d := range p.Direction {
			result += fmt.Sprintf("<div class=bus><div class=route>%s</div><div class=arrivals>",
				template.HTMLEscapeString(p.RouteTag))
			if len(d.Prediction) == 0 {
				result += "Probably never"
			}

		}
	}

	//return map[string]string{"NextBus": fmt.Sprintf("<div class=box style='width: 300px; top: 66px; left: 475px; font-size: 12px'>%s</div>", data)}
	io.WriteString(w, nextBusDiv)
}

const nextBusDiv = `
<div class=box style="width: 300px; top: 16px; left: 475px">
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
        <div class=arrivals>now, 41, 59, 79 minutes</div>
    </div>
</div>
`

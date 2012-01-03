package clocky

import (
	"io"
	"strconv"
	"strings"
	"template"
	"time"
	"xml"

	"appengine"
	"appengine/memcache"
)

type prediction struct {
	EpochTime   int64 `xml:"attr"`
	IsDeparture bool  `xml:"attr"`
}

func (p prediction) ToString() string {
	s := p.EpochTime/1000 - time.Seconds()
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

	for _, p := range data.Predictions {
		for _, d := range p.Direction {
			io.WriteString(w, `<div class=bus><div class=route>`)
			template.HTMLEscape(w, []byte(p.RouteTag))
			io.WriteString(w, ` <span class=smaller>`)
			template.HTMLEscape(w, []byte(d.Title))
			io.WriteString(w, `</span></div><div class=arrivals>`)
			for i, pp := range d.Prediction {
				if pp.IsDeparture && i == 0 {
					io.WriteString(w, "departs ")
				}
				s := pp.ToString()
				io.WriteString(w, s)
				if i == len(d.Prediction)-1 {
					if s == "1" {
						io.WriteString(w, " minute")
					} else {
						io.WriteString(w, " minutes")
					}
				} else {
					io.WriteString(w, ", ")
				}
			}
			io.WriteString(w, `</div>`)
		}
	}
}

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
	"strings"
	"text/template" // TODO: Switch to Go 1's html/template.
	"time"

	"appengine"
	"appengine/memcache"
)

type prediction struct {
	Millis    int64 `xml:"epochTime,attr"`
	Departure bool  `xml:"isDeparture,attr"`
}

func (p prediction) String() string {
	d := time.Unix(p.Millis/1000, p.Millis%1000).Sub(time.Now())
	if d < 60*time.Second {
		return "now"
	}
	result := fmt.Sprintf("%.0f", d.Minutes())
	if d < 600*time.Second && int(d.Seconds())%60 >= 30 {
		result += "½"
	}
	return result
}

var BoringMuniMessages = map[string]bool{
	"On board, watch\nyour valuables.":                                        true,
	"PROOF OF PAYMENT\nis required when\non a Muni vehicle\nor in a station.": true,
	"sfmta.com or 3 1 1\nfor Muni info":                                       true,
}

func NextBus(w io.Writer, c appengine.Context) {
	item, err := memcache.Get(c, "nextbus")
	if err != nil {
		c.Errorf("%s", err)
		return
	}

	data := struct {
		Predictions []struct {
			RouteTag  string `xml:"routeTag,attr"`
			Direction []struct {
				Title      string       `xml:"title,attr"`
				Prediction []prediction `xml:"prediction"`
			} `xml:"direction"`
			Message []struct {
				Text string `xml:"text,attr"`
			} `xml:"message"`
		} `xml:"predictions"`
	}{}
	if err := xml.Unmarshal(item.Value, &data); err != nil {
		c.Errorf("%s", err)
		return
	}

	// Messages are given per route, but they seem to be used for
	// systemwide messages.  Annoyingly, not every route gets the
	// message, so we can't infer that a message is about a
	// systemwide event.
	seen := make(map[string]bool)
	for _, p := range data.Predictions {
		for _, m := range p.Message {
			if BoringMuniMessages[m.Text] || seen[m.Text] {
				continue
			}
			seen[m.Text] = true
			io.WriteString(w, `<div class=munimessage>`)
			template.HTMLEscape(w, []byte(m.Text))
			io.WriteString(w, `</div>`)
		}
	}

	for _, p := range data.Predictions {
		for _, d := range p.Direction {
			io.WriteString(w, `<div class=bus><div class=route>`)
			template.HTMLEscape(w, []byte(p.RouteTag))
			io.WriteString(w, ` <span class=smaller>`)
			title := d.Title
			title = strings.Replace(title, "Inbound", "inbound", -1)
			title = strings.Replace(title, "Outbound", "outbound", -1)
			title = strings.Replace(title, "Downtown", "downtown", -1)
			title = strings.Replace(title, " District", "", -1)
			title = strings.Replace(title, " Disrict", "", -1)
			template.HTMLEscape(w, []byte(title))
			io.WriteString(w, `</span></div><div>`)
			for i, pp := range d.Prediction {
				if pp.Departure && i == 0 {
					io.WriteString(w, "departs ")
				}
				s := pp.String()
				io.WriteString(w, s)
				if i == len(d.Prediction)-1 {
					if s == "1" {
						io.WriteString(w, ` minute`)
					} else {
						io.WriteString(w, ` minutes`)
					}
				} else {
					io.WriteString(w, `, `)
				}
			}
			io.WriteString(w, `</div>`)
		}
	}
}

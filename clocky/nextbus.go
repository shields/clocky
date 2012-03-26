// Copyright 2011 Michael Shields
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
				if pp.IsDeparture && i == 0 {
					io.WriteString(w, "departs ")
				}
				s := pp.ToString()
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

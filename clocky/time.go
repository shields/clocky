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
	"html/template"
	"io"
	"time"

	"appengine"

	"solar"
)

const TimeFormat = "3:04\u2009pm"

const Zone = "America/Los_Angeles"

func Time(w io.Writer, c appengine.Context) {
	location, _ := time.LoadLocation(Zone)
	now := time.Now().In(location)
	sunrise := solar.Rise(now, Lat, Lng).In(location)
	sunset := solar.Set(now, Lat, Lng).In(location)
	sun1 := "sunrise " + sunrise.Format(TimeFormat)
	sun2 := "sunset " + sunset.Format(TimeFormat)
	if sunrise.Sub(sunset) > 0 {
		sun1, sun2 = sun2, sun1
	}
	timeTmpl.Execute(w, map[string]string{
		"Big":   now.Format("3:04"),
		"Small": now.Format(":05\u2009pm"),
		"Date":  now.Format("Monday, January 2"),
		"Sun1":  sun1,
		"Sun2":  sun2,
	})
}

var timeTmpl = template.Must(template.New("time").Parse(`
 <div class=header><span class=larger>{{.Big}}</span>{{.Small}}</div>
 <div class=smaller>{{.Date}}</div>
 <div class=smaller>{{.Sun1}}, {{.Sun2}}</div>
`))

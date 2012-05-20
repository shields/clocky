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
	"io"
	"net/http"

	"appengine"
)

const Lat, Lng = 37.79, -122.42

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	c := appengine.NewContext(r)
	ch := make(chan error)
	go func() { ch <- freshenAll(c) }()

	// TODO: Refresh less often; use JS to tick clock.
	// TODO: Have browser refresh; safer since error pages will get retried.
	w.Header().Set("Refresh", "2")

	io.WriteString(w, header)

	io.WriteString(w, `<div class=box style="width: 350px; height: 128px; top: 24px; left:28px; text-align: center; background-color: #eee">`)
	Time(w, c)
	io.WriteString(w, `</div>`)

	io.WriteString(w, `<div class=box style="width: 400px; top: 170px; left: 24px">`)
	Conditions(w, c)
	Forecast(w, c)
	io.WriteString(w, `</div>`)

	io.WriteString(w, `<div class=box style="width: 320px; top: 16px; left: 460px; font-size: 20px">`)
	NextBus(w, c)
	io.WriteString(w, `</div>`)

	if err := <-ch; err != nil {
		c.Errorf("%s", err)
	}
}

func init() {
	http.HandleFunc("/", handler)
}

const header = `<!DOCTYPE html>
<head>
    <title>Clocky</title>
    <style>
        body { font-size: 32px; font-family: sans-serif; margin: 0; }
        div { margin: 4px; }
        .header { font-weight: bold; }
        .smaller { font-size: 61%; }
        .larger { font-size: 164%; }
        .box { position: absolute; }
        .bus { margin: 8px 0 8px 0; }
        .route { font-size: 24px; font-weight: bold; }
        .munimessage { font-style: italic; }
    </style>
</head>
`

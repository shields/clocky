package clocky

import (
	"http"
	"template"

	"appengine"
)

const Lat, Lng = 37.79, -122.42

var tmpl = template.Must(template.New("page").Parse(page))

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/_ah/warmup" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Data probably won't be fresh by the time we need it, but
	// instead of blocking, it's better to serve what we have and
	// warm up for the next refresh.
	c := appengine.NewContext(r)
	for _, s := range Sources {
		go func(s *Source) { s.Freshen(c) }(s)
	}

	// TODO: Refresh less often; use JS to tick clock.
	// TODO: Have browser refresh; safer since error pages will get retried.
	w.Header().Set("Refresh", "2")

	d := map[string]map[string]string{
		"Time":       Time(c),
		"Conditions": Conditions(c),
		"Forecast":   map[string]string{"Forecast": dummyForecast},
	}
	tmpl.Execute(w, d)
}

func init() {
	http.HandleFunc("/", handler)
}

const page = `<!DOCTYPE html>
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
        .route { font-weight: bold; font-size: 24px; }
        .arrivals { font-size: 20px; }
    </style>
</head>

{{/* Everything temporarily down 50px to avoid blinking status bar in Boat. */}}
{{with .Time}}
<div class=box style="width: 350px; height: 128px; top: 74px; left:28px; text-align: center; background-color: #eee">
    <div class=header><span class=larger>{{.Big}}</span>{{.Small}}</div>
    <div class=smaller>{{.Date}}</div>
    <div class=smaller>{{.Sun1}}, {{.Sun2}}</div>
</div>
{{end}}

<div class=box style="width: 400px; top: 220px; left: 24px">
    {{with .Conditions}}
    <div class=header><span class=larger>{{.Temp}}</span>
        {{if .WindChill}}{{.WindChill}}{{else}}{{.Wind}}{{end}}
    </div>
    {{end}}
    <div class=smaller style="text-align: left">{{.Forecast.Forecast}}</div>
</div>

<div class=box style="width: 300px; top: 66px; left: 475px">
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

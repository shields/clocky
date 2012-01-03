package clocky

import (
	"http"
	"io"

	"appengine"
)

const Lat, Lng = 37.79, -122.42

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

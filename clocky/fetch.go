package clocky

import (
	"http"
	"io"
	"io/ioutil"
	"log"
	"os"

	"appengine"
	"appengine/memcache"
	"appengine/urlfetch"
)

// http://www.sfmta.com/cms/asite/nextmunidata.htm
const NextBusURL = ("http://webservices.nextbus.com/service/publicXMLFeed?" +
	"command=predictionsForMultiStops&a=sf-muni" +
	"&stops=1|null|4016" +
	"&stop=1|null|6297" +
	"&stops=10|null|5859" +
	"&stops=12|null|5859" +
	"&stops=27|null|35165" +
	"&stops=47|null|6825" +
	"&stops=49|null|6825")

const WeatherURL = "http://forecast.weather.gov/MapClick.php?lat=37.79570&lon=-122.42100&FcstType=dwml&unit=1"

func fetch(r *http.Request, key, url string, expiration int32) os.Error {
	c := appengine.NewContext(r)

	client := urlfetch.Client(c)
	resp, err := client.Get(url)
	if err != nil {
		log.Print(err)
		return err
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return err
	}
	resp.Body.Close()

	item := &memcache.Item{
		Key:        key,
		Value:      contents,
		Expiration: expiration,
	}
	if err := memcache.Set(c, item); err != nil {
		log.Print(err)
		return err
	}

	log.Printf("fetched %d bytes of %s data", len(contents), key)
	return nil
}

const NextBusKey = "nextbus"
const WeatherKey = "weather"

// Normally called whenever a page refresh notices stale data.
func fetchNextBus(r *http.Request) os.Error {
	return fetch(r, NextBusKey, NextBusURL, 300)
}

// Normally called hourly by cron.
// http://graphical.weather.gov/xml/mdl/XML/Design/WebServicesUseGuildlines.php
func fetchWeather(r *http.Request) os.Error {
	return fetch(r, WeatherKey, WeatherURL, 4*3600)
}

func fetchNextBusHandler(w http.ResponseWriter, r *http.Request) {
	if err := fetchNextBus(r); err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, "ok\n")
}

func fetchWeatherHandler(w http.ResponseWriter, r *http.Request) {
	if err := fetchWeather(r); err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, "ok\n")
}

func warmup(w http.ResponseWriter, r *http.Request) {
	ch := make(chan os.Error)
	go func() { ch <- fetchNextBus(r) }()
	go func() { ch <- fetchWeather(r) }()
	for i := 0; i < 2; i++ {
		if err := <-ch; err != nil {
			http.Error(w, err.String(), http.StatusInternalServerError)
			return
		}
	}
	io.WriteString(w, "ok\n")
}

func init() {
	http.HandleFunc("/fetch/nextbus", fetchNextBusHandler)
	http.HandleFunc("/fetch/weather", fetchWeatherHandler)
	http.HandleFunc("/_ah/warmup", warmup)
}

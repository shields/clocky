package clocky

import (
	"http"
	"io"
	"io/ioutil"

	"appengine"
	"appengine/memcache"
	"appengine/urlfetch"
)

const WeatherURL = "http://forecast.weather.gov/MapClick.php?lat=37.79570&lon=-122.42100&FcstType=dwml&unit=1"

func fetchWeather(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	client := urlfetch.Client(c)
	resp, err := client.Get(WeatherURL)
	if err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	resp.Body.Close()

	item := &memcache.Item{
		Key:        "weather",
		Value:      contents,
		Expiration: 4 * 3600,
	}
	if err := memcache.Set(c, item); err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}

	io.WriteString(w, "ok\n")
}

func init() {
	http.HandleFunc("/fetch/weather", fetchWeather)
}

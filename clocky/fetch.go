package clocky

import (
	"http"
	"io"
	"io/ioutil"
	"os"

	"appengine"
	"appengine/memcache"
	"appengine/urlfetch"
)

const WeatherURL = "http://forecast.weather.gov/MapClick.php?lat=37.79570&lon=-122.42100&FcstType=dwml&unit=1"

func fetchWeather(r *http.Request) os.Error {
	c := appengine.NewContext(r)

	client := urlfetch.Client(c)
	resp, err := client.Get(WeatherURL)
	if err != nil {
		return err
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()

	item := &memcache.Item{
		Key:        "weather",
		Value:      contents,
		Expiration: 4 * 3600,
	}
	if err := memcache.Set(c, item); err != nil {
		return err
	}

	return nil
}

func fetchWeatherHandler(w http.ResponseWriter, r *http.Request) {
	if err := fetchWeather(r); err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, "ok\n")
}

func init() {
	http.HandleFunc("/fetch/weather", fetchWeatherHandler)
	http.HandleFunc("/ah/_warmup", fetchWeatherHandler)
}

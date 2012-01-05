package clocky

import (
	"fmt"
	"http"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"appengine"
	"appengine/memcache"
	"appengine/taskqueue"
	"appengine/urlfetch"
)

type Source struct {
	URL                 string
	Refresh, Expiration int
}

var Sources = map[string]Source{
	"nextbus": Source{
		URL: ("http://webservices.nextbus.com/service/publicXMLFeed?" +
			"command=predictionsForMultiStops&a=sf-muni" +
			"&stops=47|null|6825" +
			"&stops=49|null|6825" +
			"&stops=90|null|6825" +
			"&stops=10|null|5859" +
			"&stops=12|null|5859" +
			"&stops=1|null|4016" +
			"&stops=1|null|6297" +
			"&stops=27|null|5165"),
		Refresh:    10,
		Expiration: 300,
	},
	"forecast": Source{
		URL: ("http://forecast.weather.gov/MapClick.php?" +
			"lat=37.79570&lon=-122.42100&FcstType=dwml&unit=1"),
		Refresh:    3600,
		Expiration: 8 * 3600,
	},
	// A buoy near Crissy Field.
	"conditions": Source{
		URL:        "http://www.weather.gov/xml/current_obs/display.php?stid=FTPC1",
		Refresh:    3600,
		Expiration: 4 * 3600,
	},
}

func fetch(c appengine.Context, key string) os.Error {
	s, ok := Sources[key]
	if !ok {
		return fmt.Errorf("%q not found", key)
	}

	c.Debugf("fetching %s data", key)
	transport := urlfetch.Transport{Context: c, DeadlineSeconds: 60}
	req, err := http.NewRequest("GET", s.URL, strings.NewReader(""))
	if err != nil {
		return err
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch: bad status %d for %s", resp.StatusCode, s.URL)
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	item := &memcache.Item{
		Key:        key,
		Value:      contents,
		Expiration: int32(s.Expiration),
	}
	if err := memcache.Set(c, item); err != nil {
		return err
	}

	// We keep the last updated time in memcache.  It's not
	// updated atomically with the page, so it's only used to
	// limit the rate of fetches from the data servers.  Don't use
	// it for display; use the data creation times in the data
	// instead.  It doesn't matter to the user that we fetched a
	// weather forecast 3 minutes ago if the forecast is 48
	// minutes old.
	item = &memcache.Item{
		Key:   key + "_fresh",
		Value: []byte(strconv.Itoa64(time.Seconds())),
	}
	if err := memcache.Set(c, item); err != nil {
		return err
	}

	c.Infof("cached %d bytes of %s data", len(contents), key)
	return nil
}

func freshen(c appengine.Context, key string) os.Error {
	s, ok := Sources[key]
	if !ok {
		return fmt.Errorf("%q not found", key)
	}

	item, err := memcache.Get(c, key+"_fresh")
	if err == memcache.ErrCacheMiss {
		return fetch(c, key)
	} else if err != nil {
		return err
	}
	fresh, err := strconv.Atoi64(string(item.Value))
	if err != nil {
		return err
	}
	stale := fresh + int64(s.Refresh)
	c.Debugf("%s stale = %s", key, time.SecondsToUTC(stale).Format(time.RFC3339))
	if stale > time.Seconds() {
		return nil
	}

	t := &taskqueue.Task{Path: "/fetch/" + key}
	if _, err := taskqueue.Add(c, t, "fetch-"+key); err != nil {
		return err
	}

	return nil
}

func freshenHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	ch := make(chan os.Error)
	for key, _ := range Sources {
		go func(key string) { ch <- freshen(c, key) }(key)
	}
	for _ = range Sources {
		if err := <-ch; err != nil {
			c.Errorf("%s", err)
			http.Error(w, err.String(), http.StatusInternalServerError)
			return
		}
	}
	io.WriteString(w, "ok\n")
}

func init() {
	http.HandleFunc("/freshen", freshenHandler)
	http.HandleFunc("/_ah/warmup", freshenHandler)

	for key, _ := range Sources {
		h := func(key string) func(w http.ResponseWriter, r *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {
				if err := fetch(appengine.NewContext(r), key); err != nil {
					http.Error(w, err.String(), http.StatusInternalServerError)
					return
				}
			}
		}(key)
		http.HandleFunc("/fetch/"+key, h)
	}
}

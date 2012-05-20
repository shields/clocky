// Copyright 2012 Michael Shields
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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
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
	Refresh, Expiration time.Duration
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
		Refresh:    10 * time.Second,
		Expiration: 5 * time.Minute,
	},
	"forecast": Source{
		URL: ("http://forecast.weather.gov/MapClick.php?" +
			"lat=37.79570&lon=-122.42100&FcstType=dwml&unit=1"),
		Refresh:    1 * time.Hour,
		Expiration: 8 * time.Hour,
	},
	// NDBC latest observations for all points.  This file is much
	// smaller than the file for any individual station, because
	// the latter contains 45 days of 6-minute observations.
	// http://www.ndbc.noaa.gov/measdes.shtml
	"conditions": Source{
		URL:        "http://www.ndbc.noaa.gov/data/latest_obs/latest_obs.txt",
		Refresh:    6 * time.Minute,
		Expiration: 30 * time.Minute,
	},
}

func fetch(c appengine.Context, key string) error {
	s, ok := Sources[key]
	if !ok {
		return fmt.Errorf("%q not found", key)
	}

	c.Debugf("fetching %s data", key)
	transport := urlfetch.Transport{Context: c, Deadline: 60 * time.Second}
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
		Expiration: s.Expiration,
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
		Value: []byte(strconv.FormatInt(time.Now().Unix(), 10)),
	}
	if err := memcache.Set(c, item); err != nil {
		return err
	}

	c.Infof("cached %d bytes of %s data", len(contents), key)
	return nil
}

func freshen(c appengine.Context, key string) error {
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
	fresh, err := strconv.ParseInt(string(item.Value), 10, 64)
	if err != nil {
		return err
	}
	if time.Now().Unix() < fresh + int64(s.Refresh.Seconds()) {
		return nil
	}

	t := &taskqueue.Task{Path: "/fetch/" + key}
	if _, err := taskqueue.Add(c, t, "fetch-"+key); err != nil {
		return err
	}

	return nil
}

func freshenAll(c appengine.Context) error {
	ch := make(chan error)
	for key, _ := range Sources {
		go func(key string) { ch <- freshen(c, key) }(key)
	}
	for _ = range Sources {
		if err := <-ch; err != nil {
			return err
		}
	}
	return nil
}

func freshenAllHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	if err := freshenAll(c); err != nil {
		c.Errorf("%s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, "ok\n")
}

func init() {
	http.HandleFunc("/freshen", freshenAllHandler)
	http.HandleFunc("/_ah/warmup", freshenAllHandler)

	for key, _ := range Sources {
		h := func(key string) func(w http.ResponseWriter, r *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {
				c := appengine.NewContext(r)
				if err := fetch(c, key); err != nil {
					c.Errorf("%s", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				io.WriteString(w, "ok\n")
			}
		}(key)
		http.HandleFunc("/fetch/"+key, h)
	}
}

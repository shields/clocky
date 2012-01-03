package clocky

import (
	"http"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"appengine"
	"appengine/memcache"
	"appengine/urlfetch"
)

type Source struct {
	Key                 string
	URL                 string
	Refresh, Expiration int
}

var Sources = []*Source{
	&Source{
		Key: "nextbus",
		// http://www.sfmta.com/cms/asite/nextmunidata.htm
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
	&Source{
		Key: "forecast",
		URL: ("http://forecast.weather.gov/MapClick.php?" +
			"lat=37.79570&lon=-122.42100&FcstType=dwml&unit=1"),
		// http://graphical.weather.gov/xml/mdl/XML/Design/WebServicesUseGuildlines.php
		Refresh:    3600,
		Expiration: 8 * 3600,
	},
	// A buoy near Crissy Field.
	&Source{
		Key: "conditions",
		URL: "http://www.weather.gov/xml/current_obs/display.php?stid=FTPC1",
		// http://graphical.weather.gov/xml/mdl/XML/Design/WebServicesUseGuildlines.php
		Refresh:    3600,
		Expiration: 4 * 3600,
	},
}

func (s Source) Fetch(c appengine.Context) os.Error {
	c.Debugf("fetching %s data", s.Key)

	client := urlfetch.Client(c)
	resp, err := client.Get(s.URL)
	if err != nil {
		c.Errorf("%q", err)
		return err
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.Errorf("%q", err)
		return err
	}
	resp.Body.Close()

	item := &memcache.Item{
		Key:        s.Key,
		Value:      contents,
		Expiration: int32(s.Expiration),
	}
	if err := memcache.Set(c, item); err != nil {
		c.Errorf("%q", err)
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
		Key:   s.Key + "_fresh",
		Value: []byte(strconv.Itoa64(time.Seconds())),
	}
	if err := memcache.Set(c, item); err != nil {
		c.Errorf("%q", err)
		return err
	}

	c.Infof("fetched %d bytes of %s data", len(contents), s.Key)
	return nil
}

// Freshen calls Fetch iff the item is not known to be fresh.
func (s *Source) Freshen(c appengine.Context) os.Error {
	item, err := memcache.Get(c, s.Key+"_fresh")
	if err == memcache.ErrCacheMiss {
		return s.Fetch(c)
	} else if err != nil {
		c.Errorf("%q", err)
		// Something is wrong with memcache, so don't try to fetch.
		return err
	}

	fresh, err := strconv.Atoi64(string(item.Value))
	if err != nil {
		c.Errorf("%q", err)
		return s.Fetch(c)
	}

	stale := fresh + int64(s.Refresh)
	c.Debugf("%s stale = %s", s.Key, time.SecondsToUTC(stale).Format(time.RFC3339))
	if stale > time.Seconds() {
		return nil
	}
	return s.Fetch(c)
}

func warmup(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	ch := make(chan os.Error)
	for _, s := range Sources {
		go func(s *Source) { ch <- s.Freshen(c) }(s)
	}
	for _ = range Sources {
		if err := <-ch; err != nil {
			http.Error(w, err.String(), http.StatusInternalServerError)
			return
		}
	}
	io.WriteString(w, "ok\n")
}

func init() {
	http.HandleFunc("/_ah/warmup", warmup)
}

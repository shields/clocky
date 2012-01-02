package clocky

import (
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
			"&stops=1|null|4016" +
			"&stop=1|null|6297" +
			"&stops=10|null|5859" +
			"&stops=12|null|5859" +
			"&stops=27|null|35165" +
			"&stops=47|null|6825" +
			"&stops=49|null|6825"),
		Refresh:    10,
		Expiration: 300,
	},
	&Source{
		Key: "weather",
		URL: ("http://forecast.weather.gov/MapClick.php?" +
			"lat=37.79570&lon=-122.42100&FcstType=dwml&unit=1"),
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
	if stale > time.Seconds() {
		c.Debugf("not fetching %s until %d", s.Key, stale)
		return nil
	}
	c.Debugf("%s is stale since %d", s.Key, stale)
	return s.Fetch(c)
}

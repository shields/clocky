package clocky

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"xml"

	"appengine"
	"appengine/memcache"
)

func Conditions(c appengine.Context) map[string]string {
	item, err := memcache.Get(c, "conditions")
	if err != nil {
		c.Errorf("%q", err)
		return nil
	}
	cond := struct {
		Temp_c      string
		WindChill_c string
		Wind_Mph    string
	}{}

	// NWS serves XML in ISO-8859-1 for no reason; the data is really ASCII.
	p := xml.NewParser(strings.NewReader(string(item.Value)))
	p.CharsetReader = func(charset string, input io.Reader) (io.Reader, os.Error) {
		return input, nil
	}
	if err = p.Unmarshal(&cond, nil); err != nil {
		c.Errorf("%q", err)
		return nil
	}

	result := make(map[string]string)
	if cond.Temp_c != "" {
		result["Temp"] = cond.Temp_c + "°"
	}
	if cond.WindChill_c != "" {
		result["WindChill"] = "wind chill " + cond.WindChill_c + "°"
	}
	if cond.Wind_Mph != "" {
		mph, err := strconv.Atof64(cond.Wind_Mph)
		if err == nil {
			result["Wind"] = fmt.Sprintf("wind %d&thinsp;km/h", int(mph*1.609344))
		}
	}
	return result
}

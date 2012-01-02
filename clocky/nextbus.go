package clocky

import (
	//"fmt"

	"appengine"
	//"appengine/memcache"
)

func NextBus(c appengine.Context) map[string]string {
	return map[string]string{"NextBus": dummyNextBus}
}

const dummyNextBus = `
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

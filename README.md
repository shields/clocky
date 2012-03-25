Clocky displays the time, weather, and predicted bus arrivals.

It's a web app, but the intention is not to use it in the web browser on
your desktop.  The intention is to have a dedicated device that displays
this information all the time, like a clock.  A clock has one of the
world's greatest UIs: if you want to know the time, you look at it, and
if you don't, you don't.  I wanted a UI like that for more than the time.

It looks like this: http://clocky.msrl.com/

Clocky is written in Go and runs on Google App Engine.  It doesn't use any
exotic features of GAE, and could easily run elsewhere with small changes
to use Go's native HTTP server, or on any machine with no changes using
the GAE dev_appserver.


Display device
==============

What I'm trying to use is a rooted Nook Simple Touch.  It has an e-ink
display, so it doesn't glow, and it doesn't look too much like a computer.
At $99, it's reasonably priced.

A basic Android app is included with Clocky.  It just disables sleep and
displays the web page without an address bar or any other functionality.
It attempts to run in landscape mode, which would look a lot better,
but this doesn't quite work yet.


Customization
=============

There is none.  It's not a service.  It displays the time and weather
and bus arrivals near my home in San Francisco.  If your home is not
mine, you'll want to edit Sources in clocky/fetch.go.  Fork and enjoy.


Data sources
============

Current time is from the system.

Sunrise and sunset are from lookup tables.

Current weather conditions are from a NWS C-MAN automated data buoy
located off Crissy Field.  Wind chill is calculated.  Considering how
diverse San Francisco's microclimates are, it's a really good idea to
use a very nearby weather station.

Weather forecast is from the NWS detailed point forecast program.

Bus arrival times are from NextMuni.  Their XML data is in milliseconds,
which makes sense because Muni is known for keeping to their schedule
with sub-second precision.  The prediction for a bus arriving in less
than ten minutes will be rounded down to the half minute, not a full
minute as Muni's official signs do.  Clocky distinguishes between buses
which are going to different destinations, such as the 49 stopping at
Market instead of continuing into the Mission.  It also notes when a
bus will be departing from the start of the line at a definite time,
and when the time is a predicted time it will pass by a stop.  Finally,
service messages are displayed, but not the boring ones about watching
your belongings.

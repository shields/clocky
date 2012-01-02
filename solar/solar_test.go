package solar

import (
	"testing"
	"time"
)

func TestRiseSet(t *testing.T) {
	cases := []struct {
		when, rise, set string
	}{
		{
			"2012-01-01T23:00:00-08:00",
			"2012-01-02T15:25:00Z",
			"2012-01-03T01:02:00Z",
		},
		{
			"2012-01-04T00:00:00-08:00",
			"2012-01-04T15:25:00Z",
			"2012-01-05T01:04:00Z",
		},
		{
			"2012-01-04T00:00:00Z",
			"2012-01-04T15:25:00Z",
			"2012-01-04T01:04:00Z",
		},
		{
			"2012-01-04T20:00:00Z",
			"2012-01-05T15:26:00Z",
			"2012-01-05T01:04:00Z",
		},
	}
	for _, tt := range cases {
		when, err := time.Parse(time.RFC3339, tt.when)
		if err != nil {
			t.Fatal(err)
		}
		rise := Rise(when, sfLat, sfLng)
		if rise.Format(time.RFC3339) != tt.rise {
			t.Errorf("\nrise: %s\nwant: %s\ngot:  %s",
				tt.when, tt.rise, rise.Format(time.RFC3339))
		}
		set := Set(when, sfLat, sfLng)
		if set.Format(time.RFC3339) != tt.set {
			t.Errorf("\nset:  %s\nwant: %s\ngot:  %s",
				tt.when, tt.set, set.Format(time.RFC3339))
		}
	}
}

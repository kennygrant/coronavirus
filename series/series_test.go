package series

import (
	"path/filepath"
	"testing"
	"time"
)

var formatTests = map[int]string{
	10:       "10",
	999:      "999",
	1000:     "1000",
	1101:     "1101",
	10101:    "10.1k",
	11101:    "11.1k",
	1000000:  "1m",
	1100000:  "1.1m",
	1100499:  "1.1m",
	22400499: "22.4m",
}

func TestFormat(t *testing.T) {
	d := &Data{}

	for k, v := range formatTests {
		r := d.Format(k)
		if r != v {
			t.Errorf("format: failed for:%d want:%s got:%s", k, v, r)
		}
	}

}

// TestLoadAreas tests loading our static test area file (with just a few areas in it)
func TestLoadAreas(t *testing.T) {
	p, _ := filepath.Abs("testdata/areas.csv")
	err := LoadAreas(p)
	if err != nil {
		t.Fatalf("areas: failed to load file:%s", err)
	}

	count := 8
	if len(dataset) != count {
		t.Fatalf("areas: count wrong want:%d got:%d", count, len(dataset))
	}

	// Fetch global, should not fail
	global, err := dataset.FetchSeries("", "")
	if err != nil {
		t.Fatalf("areas: global not in dataset: s:%v", global)
	}

	if global.Color != "#000000" {
		t.Fatalf("areas: failed to load global color got:%s", global.Color)
	}

	// Fetch US, should not fail
	us, err := dataset.FetchSeries("US", "")
	if err != nil {
		t.Fatalf("areas: failed to load us:%s %v", err, dataset)
	}

	if us.Country != "US" || us.Province != "" {
		t.Fatalf("areas: failed to load us country/province got:%s", us.Country)
	}

	if us.Population != 329527888 {
		t.Fatalf("areas: failed to load us population got:%d", us.Population)
	}

	if us.Latitude != 40.0 {
		t.Fatalf("areas: failed to load us lat got:%f", us.Latitude)
	}

	// Fetch UK province
	// United Kingdom,England,54,-2.0,55977178,#201234
	england, err := dataset.FetchSeries("United Kingdom", "England")
	if err != nil {
		t.Fatalf("areas: failed to load england,uk:%s %v", err, dataset)
	}

	if england.Longitude != -2.0 {
		t.Fatalf("areas: failed to load England lon got:%f", england.Longitude)
	}

	if england.Color != "#201234" {
		t.Fatalf("areas: failed to load England color got:%s", england.Color)
	}

	venezuela, err := dataset.FetchSeries("Venezuela", "")
	if err != nil {
		t.Fatalf("areas: failed to load venezuela:%s %v", err, dataset)
	}

	if venezuela.Population != 32219521 {
		t.Fatalf("areas: failed to load venezuela population got:%d", us.Population)
	}

}

// TestLoadAreas tests loading our static test area file (with just a few areas in it)
func TestLoadSeries(t *testing.T) {
	// First load areas, so that we have a dataset to work with
	TestLoadAreas(t)

	// Now load the test series
	p, _ := filepath.Abs("testdata/series.csv")
	err := Load(p)
	if err != nil {
		t.Fatalf("series: load failed:%s", err)
	}

	// Fetch UK, should not fail
	uk, err := dataset.FetchSeries("United Kingdom", "")
	if err != nil {
		t.Fatalf("series: uk not in dataset: s:%v", uk)
	}

	// Check the uk value for deaths on 26th March 2020
	date := time.Date(2020, 1, 22, 0, 0, 0, 0, time.UTC)
	deaths := uk.FetchDate(date, DataDeaths)
	want := 0
	if deaths != want {
		t.Fatalf("series: uk deaths incorrect on date:%v want:%d got:%d", date, want, deaths)
	}

	t.Logf("dataset:%v", dataset)

	date = time.Date(2020, 3, 26, 0, 0, 0, 0, time.UTC)
	deaths = uk.FetchDate(date, DataDeaths)
	want = 578
	if deaths != want {
		t.Fatalf("series: uk deaths incorrect on date:%v want:%d got:%d", date, want, deaths)
	}

	date = time.Date(2020, 3, 14, 0, 0, 0, 0, time.UTC)
	deaths = uk.FetchDate(date, DataDeaths)
	want = 21
	if deaths != want {
		t.Fatalf("series: uk deaths incorrect on date:%v want:%d got:%d", date, want, deaths)
	}

	// Test US
	us, err := dataset.FetchSeries("US", "")
	if err != nil {
		t.Fatalf("series: us not in dataset: s:%v", uk)
	}

	date = time.Date(2020, 3, 24, 0, 0, 0, 0, time.UTC)
	deaths = us.FetchDate(date, DataDeaths)
	want = 706
	if deaths != want {
		t.Fatalf("series: us deaths incorrect on date:%v want:%d got:%d", date, want, deaths)
	}

	// Test Wyoming, fake test data inserted
	wyoming, err := dataset.FetchSeries("US", "Wyoming")
	if err != nil {
		t.Fatalf("series: us not in dataset: s:%v", uk)
	}

	date = time.Date(2020, 3, 26, 0, 0, 0, 0, time.UTC)
	deaths = wyoming.FetchDate(date, DataDeaths)
	want = 99
	if deaths != want {
		t.Fatalf("series: us deaths incorrect on date:%v want:%d got:%d", date, want, deaths)
	}
}

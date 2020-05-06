package series

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Test historical update from uk json
func TestHistoricalUKJSON(t *testing.T) {

	// Load areas first
	p, _ := filepath.Abs("testdata/areas.csv")
	err := LoadAreas(p)
	if err != nil {
		t.Fatalf("areas: failed to load file:%s", err)
	}

	// Load series first
	p, _ = filepath.Abs("testdata/series.csv")
	err = Load(p)
	if err != nil {
		t.Fatalf("series: load failed:%s", err)
	}

	p, _ = filepath.Abs("testdata/uk.json")
	f, err := os.Open(p)
	if err != nil {
		t.Fatalf("json open err:%s", err)
	}

	jsonData := make(map[string]interface{})
	err = json.NewDecoder(f).Decode(&jsonData)
	if err != nil {
		t.Fatalf("json parse err:%s", err)
	}

	err = UpdateUKDeaths(jsonData)
	if err != nil {
		t.Fatalf("failed to parse UK json:%s", err)
	}

	t.Fatalf("FINI")

}

// Test parse of UK json
func TestUKJSON(t *testing.T) {

	// Open the UK JSON FILE
	p, _ := filepath.Abs("testdata/uk.json")
	f, err := os.Open(p)
	if err != nil {
		t.Fatalf("json open err:%s", err)
	}

	jsonData := make(map[string]interface{})
	err = json.NewDecoder(f).Decode(&jsonData)
	if err != nil {
		t.Fatalf("json parse err:%s", err)
	}
	/*
		stats, err := parseUKJSON(jsonData)
		if err != nil {
			t.Fatalf("failed to parse UK json:%s", err)
		}

		if stats.UKCases != 47806 {
			t.Errorf("ukjson: incorrect UK Deaths want:%d got:%d", 47806, stats.UKCases)
		}

		if stats.WalesDeaths != 166 {
			t.Errorf("ukjson: incorrect UK Deaths want:%d got:%d", 166, stats.UKCases)
		}

		if stats.NIDeaths != 56 {
			t.Errorf("ukjson: incorrect UK Deaths want:%d got:%d", 56, stats.UKCases)
		}
	*/
}

/*
// TestUKJSONAssign tests parse and update from uk json
func TestUKJSONAssign(t *testing.T) {
	// Load areas first
	p, _ := filepath.Abs("testdata/areas.csv")
	err := LoadAreas(p)
	if err != nil {
		t.Fatalf("areas: failed to load file:%s", err)
	}

	// Add days up to today so that we can update them
	days := int(time.Now().UTC().Sub(seriesStartDate).Hours() / 24)
	// For every series add the right number of days up to and including today
	for _, series := range dataset {
		series.AddDays(days)
	}

	// Now load json
	p, _ = filepath.Abs("testdata/uk.json")
	f, err := os.Open(p)
	if err != nil {
		t.Fatalf("json open err:%s", err)
	}

	jsonData := make(map[string]interface{})
	err = json.NewDecoder(f).Decode(&jsonData)
	if err != nil {
		t.Fatalf("json parse err:%s", err)
	}

	err = UpdateFromUKStats(jsonData)
	if err != nil {
		t.Fatalf("failed to update from UK json:%s", err)
	}

	// Test fetch of wales and value
	wales, err := dataset.FetchSeries("United Kingdom", "Wales")
	if err != nil {
		t.Fatalf("failed to fetch wales:%s", err)
	}
	if wales.TotalDeaths() != 166 {
		t.Errorf("wales deaths wrong want:%d got:%d", 166, wales.TotalDeaths())
	}

}
*/

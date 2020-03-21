package covid

import (
	"os"
	"testing"
	"time"
)

func TestFetchData(t *testing.T) {
	// change dir back to root
	os.Chdir("..")
	err := FetchData()
	if err != nil {
		t.Fatalf("error fetching data:%s", err)
	}
}

func TestLoadData(t *testing.T) {

	// Test with test data - we expect a dataset of at least 462 provinces/countries
	dataPath = "./testdata/"
	err := LoadData()
	if err != nil {
		t.Fatalf("error loading data:%s", err)
	}

	// Test data is older and fixed in length
	// We exclude US counties/cities as data is no longer accurate
	// We add some global country entries which are not already there
	if len(data) != 276 {
		t.Fatalf("test: load data failed wrong len:%d", len(data))
	}

	// Test fetching datum
	series, err := data.FetchSeries("United Kingdom", "")
	if err != nil {
		t.Fatalf("test: failed fetching day of UK:%s", err)
	}

	// Test stable test data
	if len(series.Deaths) != 57 {
		t.Fatalf("test: failed fetching day of UK wrong len for deaths got:%d", len(series.Deaths))
	}

	// Spot check random days
	date, _ := time.Parse("2006-01-02", "2020-01-22")
	value, err := data.FetchDate("Kyrgyzstan", "", DataDeaths, date)
	if err != nil {
		t.Fatalf("test: failed fetching day 0 of Kyrgyzstan:%s", err)
	}
	if value != 0 {
		t.Fatalf("test: failed fetching day 0 of Kyrgyzstan:%s", err)
	}
	date, _ = time.Parse("2006-01-02", "2020-03-16")
	value, err = data.FetchDate("United Kingdom", "", DataDeaths, date)
	if err != nil {
		t.Fatalf("test: failed fetching day of UK:%s", err)
	}
	if value != 55 {
		t.Errorf("test: failed fetching day of UK wanted:%d got:%d", 55, value)
	}

	date, _ = time.Parse("2006-01-02", "2020-01-22")
	value, err = data.FetchDate("China", "Hubei", DataDeaths, date)
	if err != nil {
		t.Fatalf("test: failed fetching day of UK:%s", err)
	}
	if value != 17 {
		t.Errorf("test: failed fetching day 0 of Hubei wanted:%d got:%d", 17, value)
	}

	date, _ = time.Parse("2006-01-02", "2020-03-01")
	value, err = data.FetchDate("US", "", DataConfirmed, date)
	if err != nil {
		t.Fatalf("test: failed fetching day of US:%s", err)
	}
	if value != 44 {
		t.Errorf("test: failed fetching day of US wanted:%d got:%d", 44, value)
	}

}

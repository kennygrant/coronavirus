package series

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// UpdateHistoricalUKData updates older data as a one-off
func UpdateHistoricalUKData() error {
	p, _ := filepath.Abs("series/testdata/uk.json")
	f, err := os.Open(p)
	if err != nil {
		return err
	}

	jsonData := make(map[string]interface{})
	err = json.NewDecoder(f).Decode(&jsonData)
	if err != nil {
		return err
	}

	err = UpdateHistoricalUKDeaths(jsonData)
	if err != nil {
		return err
	}

	return nil
}

// Generate days for each country?
// Perhaps just fetch series concerned and directly update in memory?

// UpdateHistoricalUKDeaths is used to update historical deaths for the uk
// a simpler update function can be used to update daily?
func UpdateHistoricalUKDeaths(jsonData map[string]interface{}) error {

	// Lock during add operation
	mutex.Lock()
	defer mutex.Unlock()

	ukDeaths := make(map[string]int)

	// Read JSON - two lists - one overview and one 'countries' for every country

	// Read the overview - uk data
	overview := jsonData["overview"].([]interface{})
	for _, e := range overview {
		entry := e.(map[string]interface{})
		deaths, _ := entry["cumulativeDeaths"].(float64)
		date, _ := entry["reportingDate"].(string)

		switch entry["areaName"] {
		case "United Kingdom":
			ukDeaths[date] = int(deaths)
		}
	}

	englandDeaths := make(map[string]int)
	walesDeaths := make(map[string]int)
	scotlandDeaths := make(map[string]int)
	niDeaths := make(map[string]int)

	// Read the countries - sub-uk data
	countries := jsonData["countries"].([]interface{})
	for _, e := range countries {
		entry := e.(map[string]interface{})
		deaths, _ := entry["cumulativeDeaths"].(float64)
		date, _ := entry["reportingDate"].(string)

		switch entry["areaName"] {
		case "England":
			englandDeaths[date] = int(deaths)
		case "Wales":
			walesDeaths[date] = int(deaths)
		case "Scotland":
			scotlandDeaths[date] = int(deaths)
		case "Northern Ireland":
			niDeaths[date] = int(deaths)
		}

	}

	// Make sure last day deaths are up to date too for these series

	// Fetch UK series
	uk, err := dataset.FetchSeries("United Kingdom", "")
	if err != nil || uk.Count() == 0 {
		return fmt.Errorf("failed to fetch uk series:%s", err)
	}
	// Walk the uk series and where we have a match on dates set the data
	for _, day := range uk.Days {
		deaths, ok := ukDeaths[day.DateMachine()]
		if ok {
			day.Deaths = deaths
		}
	}

	// Fetch England series
	series, err := dataset.FetchSeries("United Kingdom", "England")
	if err != nil || uk.Count() == 0 {
		return fmt.Errorf("failed to fetch england series:%s", err)
	}
	// Walk the uk series and where we have a match on dates set the data
	for _, day := range series.Days {
		deaths, ok := englandDeaths[day.DateMachine()]
		if ok {
			day.Deaths = deaths
		}
	}
	series.Days[len(series.Days)-1].Deaths = series.Days[len(series.Days)-2].Deaths

	// Fetch Wales series
	series, err = dataset.FetchSeries("United Kingdom", "Wales")
	if err != nil || uk.Count() == 0 {
		return fmt.Errorf("failed to fetch Wales series:%s", err)
	}
	// Walk the uk series and where we have a match on dates set the data
	for _, day := range series.Days {
		deaths, ok := walesDeaths[day.DateMachine()]
		if ok {
			day.Deaths = deaths
		}
	}
	series.Days[len(series.Days)-1].Deaths = series.Days[len(series.Days)-2].Deaths

	// Fetch Scotland series
	series, err = dataset.FetchSeries("United Kingdom", "Scotland")
	if err != nil || uk.Count() == 0 {
		return fmt.Errorf("failed to fetch Scotland series:%s", err)
	}
	// Walk the uk series and where we have a match on dates set the data
	for _, day := range series.Days {
		deaths, ok := scotlandDeaths[day.DateMachine()]
		if ok {
			day.Deaths = deaths
		}
	}
	series.Days[len(series.Days)-1].Deaths = series.Days[len(series.Days)-2].Deaths

	// Fetch NI series
	series, err = dataset.FetchSeries("United Kingdom", "Northern Ireland")
	if err != nil || uk.Count() == 0 {
		return fmt.Errorf("failed to fetch Northern Ireland series:%s", err)
	}
	// Walk the uk series and where we have a match on dates set the data
	for _, day := range series.Days {
		deaths, ok := niDeaths[day.DateMachine()]
		if ok {
			day.Deaths = deaths
		}
	}
	series.Days[len(series.Days)-1].Deaths = series.Days[len(series.Days)-2].Deaths

	return nil
}

/*
// UKStats contains daily death counts for uk countries for one day
type UKStats struct {
	UKDeaths       int
	EnglandDeaths  int
	ScotlandDeaths int
	WalesDeaths    int
	NIDeaths       int
}


// UpdateFromUKStats reads stats from the official UK source
// https://coronavirus.data.gov.uk/#countries
// daily deaths are now available as json broken down by country
// daily cases are only available for england
func UpdateFromUKStats(jsonData map[string]interface{}) error {

	stats, err := parseUKJSON(jsonData)
	if err != nil {
		return err
	}

	log.Printf("uk stats:%v", stats)

		// Lock during add operation
		mutex.Lock()
		defer mutex.Unlock()

		// Grab the series concerned and update them
		uk, err := dataset.FetchSeries("United Kingdom", "")
		if err != nil {
			return fmt.Errorf("failed to fetch uk series")
		}
		uk.UpdateToday(time.Now().UTC(), stats.UKDeaths, stats.UKCases, 0, 0)

		england, err := dataset.FetchSeries("United Kingdom", "England")
		if err != nil {
			return fmt.Errorf("failed to fetch England series")
		}
		england.UpdateToday(time.Now().UTC(), stats.EnglandDeaths, stats.EnglandCases, 0, 0)

		scotland, err := dataset.FetchSeries("United Kingdom", "Scotland")
		if err != nil {
			return fmt.Errorf("failed to fetch Scotland series")
		}
		scotland.UpdateToday(time.Now().UTC(), stats.ScotlandDeaths, stats.ScotlandCases, 0, 0)

		wales, err := dataset.FetchSeries("United Kingdom", "Wales")
		if err != nil {
			return fmt.Errorf("failed to fetch Wales series")
		}
		wales.UpdateToday(time.Now().UTC(), stats.WalesDeaths, stats.WalesCases, 0, 0)

		northernIreland, err := dataset.FetchSeries("United Kingdom", "Northern Ireland")
		if err != nil {
			return fmt.Errorf("failed to fetch NI series")
		}
		northernIreland.UpdateToday(time.Now().UTC(), stats.NIDeaths, stats.NICases, 0, 0)

	return nil
}

func parseUKJSON(jsonData map[string]interface{}) (UKStats, error) {

	var stats UKStats

	// Find the data we're interested in
	features, ok := jsonData["features"].([]interface{})
	if !ok {
		return stats, fmt.Errorf("json format unexpected:%v", jsonData["features"])
	}

	properties, ok := features[0].(map[string]interface{})["properties"].(map[string]interface{})
	if !ok {
		return stats, fmt.Errorf("json format unexpected:%v", jsonData["features"])
	}

	for k, v := range properties {
		if k == "TotalUKCases" {
			stats.UKCases = int(v.(float64))
		} else if k == "TotalUKDeaths" {
			stats.UKDeaths = int(v.(float64))
		} else if k == "EnglandCases" {
			stats.EnglandCases = int(v.(float64))
		} else if k == "EnglandDeaths" {
			stats.EnglandDeaths = int(v.(float64))
		} else if k == "ScotlandCases" {
			stats.ScotlandCases = int(v.(float64))
		} else if k == "ScotlandDeaths" {
			stats.ScotlandDeaths = int(v.(float64))
		} else if k == "WalesCases" {
			stats.WalesCases = int(v.(float64))
		} else if k == "WalesDeaths" {
			stats.WalesDeaths = int(v.(float64))
		} else if k == "NICases" {
			stats.NICases = int(v.(float64))
		} else if k == "NIDeaths" {
			stats.NIDeaths = int(v.(float64))
		}
	}

	return stats, nil
}

*/

package series

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// UpdateUKData updates older data as a one-off
func UpdateUKData() error {
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

	err = UpdateUKDeaths(jsonData)
	if err != nil {
		return err
	}

	return nil
}

// Generate days for each country?
// Perhaps just fetch series concerned and directly update in memory?

// UpdateUKDeaths is used to update historical deaths for the uk
// a simpler update function can be used to update daily?
func UpdateUKDeaths(jsonData map[string]interface{}) error {

	// Lock during add operation
	mutex.Lock()
	defer mutex.Unlock()

	ukDeaths := make(map[string]int)
	englandDeaths := make(map[string]int)
	walesDeaths := make(map[string]int)
	scotlandDeaths := make(map[string]int)
	niDeaths := make(map[string]int)

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

	log.Printf("series: update from UK Gov figures %d datapoints", len(overview))

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
	// NB this updates historical figures too
	updateUKSeries("United Kingdom", "", ukDeaths)
	updateUKSeries("United Kingdom", "England", englandDeaths)
	updateUKSeries("United Kingdom", "Wales", walesDeaths)
	updateUKSeries("United Kingdom", "Scotland", scotlandDeaths)
	updateUKSeries("United Kingdom", "Northern Ireland", niDeaths)
	return nil
}

func updateUKSeries(country, province string, deaths map[string]int) error {
	// Fetch the series
	series, err := dataset.FetchSeries(country, province)
	if err != nil || series.Count() == 0 {
		return fmt.Errorf("failed to fetch %s,%s series:%s", err, province, country)
	}
	// Walk the uk series and where we have a match on dates set the data
	for _, day := range series.Days {
		deaths, ok := deaths[day.DateMachine()]
		if ok {
			day.Deaths = deaths
		}
	}
	// If we don't have last day deaths set correctly, set them
	if series.Days[len(series.Days)-1].Deaths < series.Days[len(series.Days)-2].Deaths {
		series.Days[len(series.Days)-1].Deaths = series.Days[len(series.Days)-2].Deaths
	}
	return nil
}

// UpdateUKTemp updates older data as a one-off
// this is used only to update the figures with latest historical stats for UK.
func UpdateUKTemp() error {

	// TMP - replace historical uk data
	err := UpdateUKData()
	if err != nil {
		log.Printf("update: failed to update uk series :%s", err)
		return err
	}

	// TMP - recalculate all global data and save again
	// Now update our global series which are unfortunteley not contained in this data
	err = CalculateGlobalSeriesData()
	if err != nil {
		log.Printf("update: failed to calculate global series :%s", err)
		return err
	}

	// Now save the series file to disk
	err = Save("data/series.csv")
	if err != nil {
		log.Printf("server: failed to save series data:%s", err)
		return err
	}

	return nil
}

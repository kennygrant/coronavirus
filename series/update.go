package series

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"
)

// UpdateFromJHUCountryCases updates from JHU country cases data files
// several files are required to get all data, all with different formats
// Cols: Country_Region,Last_Update,Lat,Long_,Confirmed,Deaths,Recovered,Active
func UpdateFromJHUCountryCases(rows [][]string) error {

	// Lock during add operation
	mutex.Lock()
	defer mutex.Unlock()

	log.Printf("series: update from JHU country cases %d rows", len(rows))

	// For each row in the input data, reject if admin2 completed
	for i, row := range rows {
		// Check format on row 0
		if i == 0 {
			if row[0] != "Country_Region" || row[1] != "Last_Update" || row[7] != "Active" {
				return fmt.Errorf("error reading JHU country cases - format invalid for row:%s", row)
			}
			continue
		}

		country := row[0]
		province := ""

		// Find the series for this row
		series, err := dataset.FetchSeries(country, province)
		if err != nil || series == nil {
			continue
		}

		// If we reach here we have a valid row and series - NB shuffled cols to match our default
		updated, deaths, confirmed, recovered, err := readJHURowData(row[1], row[5], row[4], row[6])
		if err != nil {
			continue
		}

		// We don't hav etested data from JHU so leave it unchanged
		series.UpdateToday(updated, deaths, confirmed, recovered, 0)

		log.Printf("update: %s u:%v d:%d c:%d r:%d", series, updated, deaths, confirmed, recovered)

	}

	return nil
}

// UpdateFromJHUStatesCases updates from JHU states cases data files
// several files are required to get all data, all with different formats
//  0    1    			2				3			4  5     	6		7		8		9
// FIPS,Province_State,Country_Region,Last_Update,Lat,Long_,Confirmed,Deaths,Recovered,Active
func UpdateFromJHUStatesCases(rows [][]string) error {

	// Lock during add operation
	mutex.Lock()
	defer mutex.Unlock()

	log.Printf("series: update from JHU states cases %d rows", len(rows))

	// For each row in the input data, reject if admin2 completed
	for i, row := range rows {
		// Check format on row 0
		if i == 0 {
			if row[0] != "FIPS" || row[3] != "Last_Update" || row[9] != "Active" {
				return fmt.Errorf("error reading JHU states cases - format invalid for row:%s", row)
			}
			continue
		}

		country := row[2]
		province := row[1]

		// Find the series concerned
		series, err := dataset.FetchSeries(country, province)
		if err != nil || series == nil {
			continue
		}

		// If we reach here we have a valid row and series - NB shuffled cols to match our default
		updated, deaths, confirmed, recovered, err := readJHURowData(row[3], row[7], row[6], row[8])
		if err != nil {
			continue
		}

		// We don't have tested data from JHU so leave it unchanged
		series.UpdateToday(updated, deaths, confirmed, recovered, 0)

		log.Printf("update: %s u:%v d:%d c:%d r:%d", series, updated, deaths, confirmed, recovered)

	}

	return nil
}

// Note csv col order is different from our standard order
func readJHURowData(updatedstr, deathsstr, confirmedstr, recoveredstr string) (time.Time, int, int, int, error) {

	// Dates are, remarkably, in two different formats in one file
	// Try first in the one true format
	updated, err := time.Parse("2006-01-02 15:04:05", updatedstr)
	if err != nil {
		return updated, 0, 0, 0, fmt.Errorf("load: error reading updated at series:%s error:%s", updatedstr, err)
	}

	deaths, err := strconv.Atoi(deathsstr)
	if err != nil {
		return updated, 0, 0, 0, fmt.Errorf("load: error reading deaths series:%s error:%s", deathsstr, err)
	}

	confirmed, err := strconv.Atoi(confirmedstr)
	if err != nil {
		return updated, 0, 0, 0, fmt.Errorf("load: error reading confirmed series:%s error:%s", confirmedstr, err)
	}

	recovered, err := strconv.Atoi(recoveredstr)
	if err != nil {
		return updated, 0, 0, 0, fmt.Errorf("load: error reading recovered series:%s error:%s", recoveredstr, err)
	}

	return updated, deaths, confirmed, recovered, nil
}

// UKStats stores stats read from UK gov json
type UKStats struct {
	UKCases        int
	UKDeaths       int
	EnglandCases   int
	EnglandDeaths  int
	ScotlandCases  int
	ScotlandDeaths int
	WalesCases     int
	WalesDeaths    int
	NICases        int
	NIDeaths       int
}

// UpdateFromUKStats reads stats from the official UK source
// unfortunately this changes every day to the latest day's stats
// there is no coherent UK historical record
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

// CalculateGlobalSeriesData adds some top level countries which are inexplicably missing from the original dataset
// presumably they calculate these on the fly
func CalculateGlobalSeriesData() error {

	// Lock during add operation
	mutex.Lock()
	defer mutex.Unlock()

	// Fetch series
	China, err := dataset.FetchSeries("China", "")
	if err != nil {
		return err
	}
	Australia, err := dataset.FetchSeries("Australia", "")
	if err != nil {
		return err
	}
	Canada, err := dataset.FetchSeries("Canada", "")
	if err != nil {
		return err
	}
	Global, err := dataset.FetchSeries("", "")
	if err != nil {
		return err
	}

	// Reset all these series as we're recalculating from scratch
	China.ResetDays()
	Australia.ResetDays()
	Canada.ResetDays()
	Global.ResetDays()

	// Add global country entries for countries with data broken down at province level
	// these are missing in the datasets from JHU for some reason, though US is now included
	for _, s := range dataset {

		// Build an overall China series
		if s.Country == "China" {
			err = China.MergeSeries(s)
			if err != nil {
				return err
			}
		}

		// Build an overall Australia series
		if s.Country == "Australia" {
			err = Australia.MergeSeries(s)
			if err != nil {
				return err
			}
		}

		// Build an overall Canada series
		if s.Country == "Canada" {
			err = Canada.MergeSeries(s)
			if err != nil {
				return err
			}
		}

		if s.ShouldIncludeInGlobal() {
			//	log.Printf("global:%s-%d", s.Country, s.TotalDeaths())
			err = Global.MergeSeries(s)
			if err != nil {
				return err
			}
		} else {
			if s.TotalDeaths() > 0 {
				//	log.Printf("ignore for global:%s deaths:%d", s, s.TotalDeaths())
			}
		}
	}

	// Sort entire dataset by deaths desc to get the right order
	sort.Stable(dataset)

	return nil
}

package series

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Mutex to protect access to dataset below
var mutex sync.RWMutex

// Store our dataset as a local global, use mutex to access - no direct access
var dataset Slice

// Data types for imported series
const (
	DataNone = iota
	DataDeaths
	DataConfirmed
	DataRecovered
	DataTested
)

// FIXME Now unused, remove
const (
	DataTodayState   = 20
	DataTodayCountry = 21
)

// LoadData reloads all data from our data files in dataPath
// Dataset is locked for writing inside functions below
func LoadData(dataPath string) error {

	// Sanitize input
	dataPath = filepath.Clean(dataPath)

	// First load the areas data - this sets up a series per area
	areaPath := filepath.Join(dataPath, "areas.csv")
	err := loadAreas(areaPath)
	if err != nil {
		return fmt.Errorf("data: error loading areas:%s data:%s", areaPath, err)
	}

	// Get a list of all csv files in the data path
	files, err := filepath.Glob(dataPath + "/*.csv")
	if err != nil {
		return err
	}

	// Now load any series files we find in data dir
	// these are identified by the series_ prefix
	for _, p := range files {
		name := filepath.Base(p)
		if strings.HasPrefix(name, "series_") {
			err = loadSeries(p)
			if err != nil {
				return err
			}
		}
	}

	// Finally sort the dataset by deaths, then alphabetically by country/province
	sort.Stable(dataset)

	log.Printf("END LOAD NEW DATA:%d\n\n\n", len(dataset))
	return nil
}

// loadAreas loads our areas from the specified areas file
func loadAreas(p string) error {
	// Open the areas CSV
	rows, err := loadCSV(p)
	if err != nil {
		return err
	}

	// Lock our dataset for write
	mutex.Lock()
	defer mutex.Unlock()

	// Walk rows reading area data (countries and provinces)
	// for each one we create a series
	for i, row := range rows {
		// validate header row
		if i == 0 {
			if row[0] != "country" || row[1] != "province" || row[6] != "colour" {
				return fmt.Errorf("areas: invalid header row in file:%s row:%s", p, row)
			}
			continue
		}

		s, err := NewData(row)
		if err != nil {
			return fmt.Errorf("areas: invalid row in file:%s row:%s error:%s", p, row, err)
		}

		dataset = append(dataset, s)
	}

	return nil
}

// loadSeries loads a series file for a given datum
// the file name is used to determine which datum to fill in
// Country and province names must match the areas file
func loadSeries(p string) error {
	// Decide on the datum based on file name
	dataType := dataTypeForPath(p)

	if dataType == DataNone {
		return fmt.Errorf("load: invalid data type for file:%s", p)
	}

	// Open the CSV file - one row per country
	rows, err := loadCSV(p)
	if err != nil {
		return err
	}

	// Make an assumption about the starting date for our data - checked below by checking header
	startDate := seriesStartDate

	log.Printf("load: loading series:%s", p)

	// Range rows loading data for each country from each row
	for i, row := range rows {
		// Validate header row
		if i == 0 {
			// We make assumptions about the start date rather than parsing the first date
			// we could instead parse this date to be more flexible
			if row[0] != "country" || row[1] != "province" || row[2] != "2020-01-22" {
				return fmt.Errorf("series: invalid header row in file:%s row:%s", p, row)
			}
			continue
		}

		// Read data row for country
		country := row[0]
		province := row[1]

		series, err := FetchSeries(country, province)
		if err != nil || series == nil {
			log.Printf("series: series not found %s, %s", country, province)
			continue
		}

		// Lock our dataset for writes
		mutex.Lock()

		// Add the data for this series
		values := intValues(row[2:])
		log.Printf("VALUES:%v %v", row[2:], values)
		series.AddData(startDate, dataType, values)

		mutex.Unlock()

	}

	return nil
}

// dataTypeForFile returns a data type for this file (e.g. deaths, confirmed)
func dataTypeForPath(p string) int {
	dataType := DataNone
	name := filepath.Base(p)
	switch name {
	case "series_deaths.csv":
		return DataDeaths
	case "series_confirmed.csv":
		return DataConfirmed
	case "series_recovered.csv":
		return DataRecovered
	case "series_tested.csv":
		return DataTested
	}

	return dataType
}

// intValues converts a list of strings to ints
func intValues(strings []string) (ints []int) {
	for _, s := range strings {
		v, err := strconv.Atoi(s)
		if err != nil {
			ints = append(ints, 0)
		} else {
			ints = append(ints, v)
		}
	}
	return ints
}

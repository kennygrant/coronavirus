package series

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Mutex to protect access to dataset below
var mutex sync.RWMutex

// Store our dataset as a local global, use mutex to access - no direct access
var dataset Slice

// FIXME unused except for import - move there
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
	start := time.Now().UTC()
	defer func() {
		log.Printf("series: loaded data in %s", time.Now().UTC().Sub(start))
	}()

	// Sanitize input
	dataPath = filepath.Clean(dataPath)

	// Lock during load operation
	mutex.Lock()
	defer mutex.Unlock()

	// Clear the existing data
	dataset = Slice{}

	// First load the areas data - this sets up a series per area
	areaPath := filepath.Join(dataPath, "areas.csv")
	err := LoadAreas(areaPath)
	if err != nil {
		return fmt.Errorf("data: error loading areas:%s data:%s", areaPath, err)
	}

	// Now load our main series file - this contains all historical data
	seriesPath := filepath.Join(dataPath, "series.csv")
	err = LoadSeries(seriesPath)
	if err != nil {
		return err
	}

	// Load a daily series file of incomplete data for today - TODO

	// Finally sort the dataset by deaths, then alphabetically by country/province
	sort.Stable(dataset)

	return nil
}

// LoadAreas loads our areas from the specified areas file
// dataset must be locked while performing this operation
func LoadAreas(p string) error {
	// Open the areas CSV
	rows, err := loadCSV(p)
	if err != nil {
		return err
	}

	// Walk rows reading area data (countries and provinces)
	// for each one we create a series
	for i, row := range rows {
		// validate header row
		if i == 0 {
			if row[0] != "country" || row[1] != "province" || row[2] != "area_id" || row[7] != "colour" {
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

// LoadSeries loads our global series file
// this contains all data in the sparse format (no rows for zero data):
// day, area_id, deaths, confirmed, recovered, tested
// dataset must be locked while performing this operation
func LoadSeries(p string) error {

	// Make an assumption about the starting date for our data - checked below by checking header
	startDate := seriesStartDate
	// Make an assumption based on that start date of the length of our data file
	// we expect it to be
	days := int(time.Now().UTC().Sub(startDate).Hours() / 24)

	log.Printf("load: loading series:%s days:%d", p, days)

	// For every series add the right number of days up to and including today
	for _, series := range dataset {
		series.AddDays(days)
	}

	// Open the CSV file - one row per day per area
	rows, err := loadCSV(p)
	if err != nil {
		return err
	}

	// Range rows loading data for each country from each row
	for i, row := range rows {
		// Validate header row
		if i == 0 {
			// We make assumptions about the start date rather than parsing the first date
			// we could instead parse this date to be more flexible
			if row[0] != "day" || row[1] != "area_id" || row[5] != "tested" {
				return fmt.Errorf("series: invalid header row in file:%s row:%s", p, row)
			}
			continue
		}

		values := intValues(row)
		if len(values) != 6 {
			return fmt.Errorf("series: invalid row len for row:%s", row)
		}

		series, err := dataset.FindSeries(values[1])
		if err != nil || series == nil {
			log.Printf("series: series not found for id:%d", values[1])
			continue
		}

		//	log.Printf("series:%s day:%v", series, values)
		// Set the series data from this row
		series.SetDayData(values[0], values[2], values[3], values[4], values[5])
	}

	return nil
}

// MergeData on series may not be required either

/* This is no longer required - would go in import */

// dataTypeForFile returns a data type for this file (e.g. deaths, confirmed)
func dataTypeForPath(p string) int {

	name := filepath.Base(p)
	if strings.Contains(name, "deaths") {
		return DataDeaths
	} else if strings.Contains(name, "confirmed") {
		return DataConfirmed
	} else if strings.Contains(name, "recovered") {
		return DataRecovered
	} else if strings.Contains(name, "tested") {
		return DataTested
	}

	return DataNone
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

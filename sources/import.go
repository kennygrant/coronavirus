package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kennygrant/coronavirus/series"
)

func main() {
	err := processData()
	if err != nil {
		log.Fatalf("data: error:%s", err)
	}
}

// processData reads the source data files and outputs data in our preferred format
func processData() error {

	// First load the areas data - this sets up a series per area
	// this is loaded from the live data director
	areaPath := filepath.Join("..", "data", "areas.csv")
	err := series.LoadAreas(areaPath)
	if err != nil {
		return fmt.Errorf("data: error loading areas:%s data:%s", areaPath, err)
	}

	// Load all series in sources path
	sourcePath := "series"

	// Get a list of all csv files in the data path
	files, err := filepath.Glob(sourcePath + "/*.csv")
	if err != nil {
		return err
	}

	// Now load our series files in the data dir
	for _, p := range files {
		name := filepath.Base(p)
		if strings.HasPrefix(name, "time_series") {

			if strings.HasSuffix(name, "_US.csv") {
				err = loadUSJHUSeries(p)
				if err != nil {
					return err
				}
			} else {
				err = loadJHUSeries(p)
				if err != nil {
					return err
				}
			}

		}
	}

	// Now we've loaded all our files we're in theory ready to write out the historical series file which the app will use.
	// One thing we must do though is fill in global series not in the original dataset which is inconsistent in this regard
	// Various global indices must be added	before writing out
	err = series.CalculateGlobalSeriesData()
	if err != nil {
		return err
	}

	// Now print out some of the series to test we have the data we need?
	global, err := series.FetchSeries("", "")
	if err != nil {
		return err
	}

	log.Printf("GLOBAL:%s", global.LastDay())

	// Now write out a series.csv file which contains all our data in the desired format cumulative totals per area per day
	//writeHistoricSeries()

	p := filepath.Join("output", "series.csv")
	return series.Save(p)
}

// writeHistoricSeries writes out a series.csv file to the data dir
// which contains a row for each date/area combo with data for a given date
// rows which would be all zero are ignored
// format: day, area_id, deaths, confirmed, recovered, tested
func writeHistoricSeries() error {
	data := series.DataSet()

	var seriesData [][]int
	var dayData [][]int

	// Open the file for writing

	// for every series, write out the data to a series.csv file
	for _, s := range data {
		// For each day, check if it is non-zero, if so write it out
		for i, d := range s.Days {
			if !d.IsZero() {
				// Day number is days since 2020-01-22 start of dataset
				dayNumber := i + 1
				seriesData = append(seriesData, []int{dayNumber, s.ID, d.Deaths, d.Confirmed, d.Recovered, d.Tested})

				if dayNumber == 73 {
					dayData = append(dayData, []int{dayNumber, s.ID, d.Deaths, d.Confirmed, d.Recovered, d.Tested})
				}
				// Write a string for log
				//log.Printf("%d,%d,%d,%d,%d,%d", dayNo, s.ID, d.Deaths, d.Confirmed, d.Recovered, d.Tested)
			}

		}
	}

	// Write the data out to files - our data is simple so we write directly
	headerRow := fmt.Sprintf("day,area_id,deaths,confirmed,recovered,tested\n")
	var row string

	// SERIES DATA
	f, err := os.Create(filepath.Join("output", "series.csv"))
	if err != nil {
		fmt.Println(err)
	}

	// Write header
	_, err = f.WriteString(headerRow)
	if err != nil {
		return fmt.Errorf("failed to write series file:%s", err)
	}
	// Write days
	for _, d := range seriesData {
		row = fmt.Sprintf("%d,%d,%d,%d,%d,%d\n", d[0], d[1], d[2], d[3], d[4], d[5])
		_, err = f.WriteString(row)
		if err != nil {
			return fmt.Errorf("failed to write day file:%s", err)
		}
	}

	// DAILY DATA (required?)
	/*
		f, err = os.Create(filepath.Join("output", "day-73.csv"))
		if err != nil {
			fmt.Println(err)
		}
		// Write header
		_, err = f.WriteString(headerRow)
		if err != nil {
			return fmt.Errorf("failed to write day file:%s", err)
		}
		// Write days
		for _, d := range dayData {
			row = fmt.Sprintf("%d,%d,%d,%d,%d,%d\n", d[0], d[1], d[2], d[3], d[4], d[5])
			_, err = f.WriteString(row)
			if err != nil {
				return fmt.Errorf("failed to write day file:%s", err)
			}
		}
	*/
	return nil
}

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

// seriesStartDate is our default start date
var seriesStartDate = time.Date(2020, 1, 22, 0, 0, 0, 0, time.UTC)

// LoadJHUSeries loads a series file for a given datum
// the file name is used to determine which datum to fill in
// This reliased on the areas.csv file being loaded first
func loadJHUSeries(p string) error {
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

	log.Printf("load: loading JHU series:%s", p)

	// Fetch otherSeries for use with cruise ships etc
	otherSeries, err := series.FetchSeries("Other", "Cruise ships etc")
	if err != nil {
		return err
	}

	// Range rows loading data for each country from each row
	for i, row := range rows {
		// Validate header row
		if i == 0 {
			// We make assumptions about the start date rather than parsing the first date
			// we could instead parse this date to be more flexible
			if row[0] != "Province/State" || row[1] != "Country/Region" || row[4] != "1/22/20" {
				return fmt.Errorf("series: invalid header row in file:%s row:%s", p, row)
			}
			continue
		}

		// Read data row for country
		country := row[1]
		province := row[0]

		// Transform countries
		switch country {
		case "Burma":
			country = "Myanmar"
		case "Taiwan*":
			country = "Taiwan"
		case "Korea, South":
			country = "South Korea"
		case "United Kingdom":
			if province == "British Virgin Islands" {
				province = "Virgin Islands"
			}
			if province == "Falkland Islands (Islas Malvinas)" {
				province = "Falkland Islands"
			}
		}

		// FIXME - if in series could lock dataset for write

		series, err := series.FetchSeries(country, province)
		if err != nil || series == nil {
			// Check if this is a known other, if not log it
			logSeriesNotFound(country, province)

			// MERGE with the other series instead and continue
			values := intValues(row[4:])
			otherSeries.MergeData(startDate, dataType, values)

		} else {
			// SET the data for this series
			values := intValues(row[4:])
			series.SetData(startDate, dataType, values)
		}

	}

	return nil
}

func logSeriesNotFound(country, province string) {

	// We know about some of these, they silently go into the Other category
	// log any we don't know about though
	switch country {
	case "Diamond Princess":
		return
	case "MS Zaandam":
		return
	}

	switch province {
	case "Grand Princess":
		return
	case "Diamond Princess":
		return
	case "MS Zaandam":
		return
	case "Recovered": // bogus data
		return
	}

	if province == "" {
		log.Printf("series: series not found %s", country)
	} else {
		log.Printf("series: series not found %s-%s", country, province)
	}
}

// State level data format varies
// UID,iso2,iso3,code3,FIPS,Admin2,Province_State,Country_Region,Lat,Long_,Combined_Key,1/22/20,

// LoadUSJHUSeries loads a series file for use county/state level data
// this includes unattributed data, and we only care about US states, not counties
// it doesn't include state level data, so we must sum all counties below states + unattributed
// the file name is used to determine which datum to fill in
// This reliased on the areas.csv file being loaded first
func loadUSJHUSeries(p string) error {
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

	log.Printf("load: loading JHU US States series:%s", p)

	// Yet more data inconsistencies - US deaths table has a population figure in it
	// adjust for this
	dateIndex := 11
	if strings.HasSuffix(p, "time_series_covid19_deaths_US.csv") {
		dateIndex = 12
	}

	// Range rows loading data for each country from each row
	for i, row := range rows {
		// Validate header row
		if i == 0 {
			// We make assumptions about the start date rather than parsing the first date
			// we could instead parse this date to be more flexible
			if row[6] != "Province_State" || row[7] != "Country_Region" || row[dateIndex] != "1/22/20" {
				return fmt.Errorf("series: invalid header row in us state file:%s row:%s", p, row)
			}
			continue
		}

		// Read data row for country
		country := row[7]
		province := row[6]

		series, err := series.FetchSeries(country, province)
		if err != nil || series == nil {
			// Check if this is a known other, if not log it
			logSeriesNotFound(country, province)

		} else {
			// Add the data for this series
			values := intValues(row[dateIndex:])
			//log.Printf("VALUES:%v %v", row[dateIndex:], values)
			series.MergeData(startDate, dataType, values)
		}

	}

	return nil
}

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

// loadCSV loads the given CSV file into memory
func loadCSV(p string) ([][]string, error) {
	p = filepath.Clean(p)
	log.Printf("data: loading file at path:%v", p)

	// Open the file at path
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

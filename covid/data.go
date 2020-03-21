package covid

import (
	"encoding/csv"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LoadData the data from the CSV files in our data dir
func LoadData(path string) (SeriesSlice, error) {

	log.Printf("data: loading data from path %s", path)

	// Get a list of csv files in the data path
	files, err := filepath.Glob(path + "/*.csv")
	if err != nil {
		return nil, err
	}

	var data SeriesSlice

	for _, fp := range files {
		data, err = loadCSVFile(fp, data)
		if err != nil {
			return nil, err
		}
	}

	// Post-process the data after loading (it doesn't include global US counts for example)
	data = processData(data)

	return data, nil
}

// loadCSVFile loads the data in file into the given data (which may be empty)
func loadCSVFile(path string, data SeriesSlice) (SeriesSlice, error) {

	log.Printf("load: loading file at path:%v", path)

	// Open the file at path
	f, err := os.Open(path)
	if err != nil {
		return data, err
	}
	r := csv.NewReader(f)
	csvData, err := r.ReadAll()
	if err != nil {
		return data, err
	}

	// Set the data type depending on file name
	dataType := DataDeaths
	if strings.HasSuffix(path, "Confirmed.csv") {
		dataType = DataConfirmed
	} else if strings.HasSuffix(path, "Recovered.csv") {
		dataType = DataRecovered
	}

	return data.MergeCSV(csvData, dataType)
}

// processData post-processes the data
// at present it just adds a series for the entire US
// based on all state-level US data
func processData(data SeriesSlice) SeriesSlice {

	// Set all series with a province matching country to blank province instead
	// so that countries don't have a duplicate province set - the dataset is inconsistent in this regard
	// At present this is France, Denmark, United Kingdom, Netherlands
	for _, s := range data {
		if s.Province == s.Country {
			s.Province = ""
		}
	}

	// Generate extra series not include in the data
	startDate := time.Date(2020, 1, 22, 0, 0, 0, 0, time.UTC)

	// Build a China series
	China := &Series{
		Country:  "China",
		Province: "",
		StartsAt: startDate,
	}

	// Build a US series
	US := &Series{
		Country:  "US",
		Province: "",
		StartsAt: startDate,
	}

	// Build an Australia series
	Australia := &Series{
		Country:  "Australia",
		Province: "",
		StartsAt: startDate,
	}

	// Build an Australia series
	Canada := &Series{
		Country:  "Canada",
		Province: "",
		StartsAt: startDate,
	}

	// Build a Global series
	Global := &Series{
		Country:  "",
		Province: "",
		StartsAt: startDate,
	}

	// Add global country entries for countries with data broken down at province level
	// Add a global dataset from all other datasets combined
	for _, s := range data {

		// Build an overall China series
		if s.Country == "China" {
			China.Merge(s)
		}

		// Build an overall US series
		// NB this ignores the US sub-state level data which we exclude from the dataset as it is no longer accurate
		// for newer dates this data is zeroed anyway
		if s.Country == "US" {
			US.Merge(s)
		}

		// Build an overall Australia series
		if s.Country == "Australia" {
			Australia.Merge(s)
		}

		// Build an overall Canada series
		if s.Country == "Canada" {
			Canada.Merge(s)
		}

		// Build a global series
		Global.Merge(s)

	}

	//	log.Printf("Added China Series:%s %s %v", China.Country, China.Province, China.Confirmed)
	data = append(data, China)

	//	log.Printf("Added US Series:%s %s %v", US.Country, US.Province, US.Confirmed)
	data = append(data, US)

	//	log.Printf("Added Australia Series:%s %s %v", Australia.Country, Australia.Province, Australia.Confirmed)
	data = append(data, Australia)

	//	log.Printf("Added Canada Series:%s %s %v", Canada.Country, Canada.Province, Canada.Confirmed)
	data = append(data, Canada)

	log.Printf("Added Global Series:%s %s %v", Global.Country, Global.Province, Global.Deaths)
	data = append(data, Global)

	// Sort the data alphabetically by country and then province
	sort.Sort(data)

	// TODO - should we sum data for countries like UK, to include dependencies for consistency?
	// the original dataset has some like China done this way, but others like UK seem to not. Check data.

	return data
}

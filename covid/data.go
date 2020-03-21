package covid

import (
	"encoding/csv"
	"log"
	"os"
	"path/filepath"
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
	// so that countries don't have a duplicate province set
	for _, s := range data {
		if s.Province == s.Country {
			s.Province = ""
			log.Printf("Blanked:%s", s.Country)
		}
	}

	// Generate extra series not include in the data
	startDate := time.Date(2020, 1, 22, 0, 0, 0, 0, time.UTC)

	// Build a US series
	US := &Series{
		Country:  "US",
		Province: "",
		StartsAt: startDate,
	}

	// Build a Global series
	global := &Series{
		Country:  "",
		Province: "",
		StartsAt: startDate,
	}

	// Walk the existing series and combine all US state-level data (including dependencies, excluding cruise ships etc)
	// may need to finesse this
	for _, s := range data {
		if s.Country == "US" && !s.USCity() {
			//		log.Printf("US State:%s %v", s.Province, s.Confirmed)
			US.Merge(s)
		}
	}

	log.Printf("Added US Series:%s %s %v", US.Country, US.Province, US.Confirmed)
	data = append(data, US)

	log.Printf("Added Global Series:%s %s %v", global.Country, global.Province, global.Confirmed)
	data = append(data, global)

	for _, s := range data {
		if s.Country == "United Kingdom" {
			log.Printf("SERIES:%v", s)
		}
	}

	return data
}

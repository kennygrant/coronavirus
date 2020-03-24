package covid

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Recovered data is no longer available, it was here:
//	"https://raw.githubusercontent.com/CSSEGISandData/COVID-19/master/csse_covid_19_data/csse_covid_19_time_series/time_series_19-covid-Recovered.csv",
// I think this is because US is not reporting recovered cases

var dataPath = "./data"
var dailyDataFiles = []string{
	"https://raw.githubusercontent.com/CSSEGISandData/COVID-19/master/csse_covid_19_data/csse_covid_19_time_series/time_series_covid19_confirmed_global.csv",
	"https://raw.githubusercontent.com/CSSEGISandData/COVID-19/master/csse_covid_19_data/csse_covid_19_time_series/time_series_covid19_deaths_global.csv",
	"https://raw.githubusercontent.com/CSSEGISandData/COVID-19/web-data/data/cases_state.csv",
	"https://raw.githubusercontent.com/CSSEGISandData/COVID-19/web-data/data/cases_country.csv",
}

var hourlyDataFiles = []string{
	"https://raw.githubusercontent.com/CSSEGISandData/COVID-19/web-data/data/cases_state.csv",
	"https://raw.githubusercontent.com/CSSEGISandData/COVID-19/web-data/data/cases_country.csv",
}

// LoadData the data from the CSV files in our data dir
func LoadData() error {

	start := time.Now()
	log.Printf("data: loading data from path %s", dataPath)

	// Get a list of csv files in the data path
	files, err := filepath.Glob(dataPath + "/*.csv")
	if err != nil {
		return err
	}

	mutex.Lock()
	defer mutex.Unlock()

	// Need to clear previous data in case we are reloading
	data = SeriesSlice{}

	// Load all our time series data files - must be loaded and processed first
	for _, fp := range files {
		name := filepath.Base(fp)
		if strings.HasPrefix(name, "time_series") {
			data, err = loadCSVFile(fp, data)
			if err != nil {
				return err
			}
		}
	}

	// Process the data after loading (it doesn't include global US counts for example)
	data = processData(data)

	// Load all our daily data files - must be loaded after main series are inserted for countries
	for _, fp := range files {
		name := filepath.Base(fp)
		if strings.HasPrefix(name, "cases_") {
			data, err = loadCSVFile(fp, data)
			if err != nil {
				return err
			}
		}
	}

	// Update the global dates
	updateGlobal(data)

	// Sort the data by deaths, then alphabetically by country
	sort.Stable(data)

	log.Printf("server: loaded data in %s len:%d", time.Now().Sub(start), len(data))

	// For Debug, output a series
	data.PrintSeries("United Kingdom", "")

	return nil
}

// loadCSVFile loads the data in file into the given data (which may be empty)
// call via LoadData above
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
	if strings.Contains(path, "confirmed") {
		dataType = DataConfirmed
	} else if strings.HasSuffix(path, "cases_state.csv") {
		dataType = DataTodayState
	} else if strings.HasSuffix(path, "cases_country.csv") {
		dataType = DataTodayCountry
	}

	return data.MergeCSV(csvData, dataType)
}

// processData post-processes the data
// adds a global data series
// adds some country level data series which are missing
func processData(data SeriesSlice) SeriesSlice {

	// Set all series with a province matching country to blank province instead
	// so that countries don't have a duplicate province set - the dataset is inconsistent in this regard
	// At present this is France, Denmark, United Kingdom, Netherlands
	for _, s := range data {
		if s.Province == s.Country {
			s.Province = ""
		}
		// Fix country names - inconsistent between all data sets!
		if s.Country == "The Bahamas" || s.Country == "Bahamas, The" {
			s.Country = "Bahamas"
		}
		if s.Country == "The Gambia" || s.Country == "Gambia, The" {
			s.Country = "Gambia"
		}
		if s.Country == "East Timor" {
			s.Country = "Timor-Leste"
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
	/*
		// Build a US series
		US := &Series{
			Country:  "US",
			Province: "",
			StartsAt: startDate,
		}
	*/
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

		// The dataset now includes a US global entry
		// Build an overall US series
		// NB this ignores the US sub-state level data which we exclude from the dataset as it is no longer accurate
		// for newer dates this data is zeroed anyway
		/*
			if s.Country == "US" {
				US.Merge(s)
			}
		*/

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
	//	data = append(data, US)

	//	log.Printf("Added Australia Series:%s %s %v", Australia.Country, Australia.Province, Australia.Confirmed)
	data = append(data, Australia)

	//	log.Printf("Added Canada Series:%s %s %v", Canada.Country, Canada.Province, Canada.Confirmed)
	data = append(data, Canada)

	log.Printf("Added Global Series:%s %s %v", Global.Country, Global.Province, Global.Deaths)
	data = append(data, Global)

	// Sort the data by deaths, then alphabetically by country
	sort.Stable(data)

	// TODO - should we sum data for countries like UK, to include dependencies for consistency?
	// the original dataset has some like China done this way, but others like UK seem to not. Check data.

	return data
}

// updateGlobal updates the global data with the updated at date
func updateGlobal(data SeriesSlice) {

	global, err := data.FetchSeries("", "")
	if err != nil {
		log.Printf("err:%s", err)
		return
	}

	// Add a blank day to global
	global.Deaths = append(global.Deaths, 0)
	global.Confirmed = append(global.Confirmed, 0)
	global.DeathsDaily = append(global.DeathsDaily, 0)
	global.ConfirmedDaily = append(global.ConfirmedDaily, 0)

	// Add global country entries for countries with data broken down at province level
	// Add a global dataset from all other datasets combined
	for _, s := range data {
		// Add final day for each series to global totals, ignoring our synthetic globals not in orgiginal dataset
		// US, Global etc
		if s.AddToGlobal() {
			global.MergeFinalDay(s)
		} else {
			//log.Printf("skipping global:%s %d", s.Country, s.TotalDeaths())
		}
	}

}

// ScheduleDataFetch sets up a regular fetch of data from data sources
func ScheduleDataFetch() {
	// Set up a scheduled time every day at 8PM UTC
	now := time.Now()
	when := time.Date(now.Year(), now.Month(), now.Day(), 3, 33, 0, 0, time.UTC)
	daily := time.Hour * 24 // daily
	hourly := time.Hour     // hourly

	// For debug, test straight away
	//when = now.Add(5 * time.Second)

	// Schedule the fetch for daily data
	ScheduleAt(FetchDataDaily, when, daily)

	// Schedule an hourly fetch for hourly data
	when = time.Date(now.Year(), now.Month(), now.Day(), 0, 5, 0, 0, time.UTC)

	ScheduleAt(FetchDataHourly, when, hourly)

}

// FetchDataDaily is called on a schedule
func FetchDataDaily() {
	log.Printf("schedule: fetching daily data from data source")

	// First download the files we need from github master branch to our data dir
	err := DownloadFiles(dailyDataFiles, dataPath)
	if err != nil {
		log.Printf("schedule: error fetching daily data from data source:%s", err)
	}

	// Add a pause after requests
	time.Sleep(1 * time.Second)

	// Trigger a reload of the data from our standard data path
	err = LoadData()
	if err != nil {
		log.Printf("schedule: error loading daily data from data source:%s", err)
	}
}

// FetchDataHourly is called on a schedule
func FetchDataHourly() {
	log.Printf("schedule: fetching hourly data from data source")

	// First download hourly data files
	err := DownloadFiles(hourlyDataFiles, dataPath)
	if err != nil {
		log.Printf("schedule: error fetching daily data from data source:%s", err)
	}

	// Add a pause after requests
	time.Sleep(1 * time.Second)

	// Trigger a reload of the data from our standard data path
	err = LoadData()
	if err != nil {
		log.Printf("schedule: error loading daily data from data source:%s", err)
	}
}

// FetchData fetches data from our data sources
// for more frequent updates, we could look at downloading the case data
func FetchData() error {

	// First download the 3 x files we need from github master branch to our data dir
	err := DownloadFiles(dailyDataFiles, dataPath)
	if err != nil {
		return err
	}

	// Allow a pause after requests to save data to disk
	time.Sleep(1 * time.Second)

	// Trigger a reload of the data from our standard data path
	return LoadData()
}

// DownloadFiles downloads the specified url to the specified file path
// requires csv files
func DownloadFiles(urls []string, dataPath string) error {

	for _, url := range urls {
		log.Printf("schedule: downloading file %s", url)

		// Get the data
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("data: error fetching data url:%s error:%s", url, err)
		}
		defer resp.Body.Close()

		name := filepath.Clean(filepath.Base(url))
		if !strings.HasSuffix(name, ".csv") {
			return fmt.Errorf("data: error csv not supplied:%s", name)
		}
		path := filepath.Join(dataPath, name)
		// Open or Create the file locally if required
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
		if err != nil {
			return fmt.Errorf("data: error opening file:%s", err)
		}
		defer f.Close()

		// Write the body to file
		_, err = io.Copy(f, resp.Body)
		if err != nil {
			return fmt.Errorf("data: error copying data url:%s path:%s error:%s", url, path, err)
		}

		// Allow a pause between requests
		time.Sleep(1 * time.Second)
	}
	return nil
}

// ScheduleAt schedules execution for a particular time and at intervals thereafter.
// If interval is 0, the function will be called only once.
// Callers should call close(task) before exiting the app or to stop repeating the action.
func ScheduleAt(f func(), t time.Time, i time.Duration) chan struct{} {
	task := make(chan struct{})
	now := time.Now().UTC()

	// Check that t is not in the past, if it is increment it by interval until it is not
	for now.Sub(t) > 0 {
		t = t.Add(i)
	}

	// We ignore the timer returned by AfterFunc - so no cancelling, perhaps rethink this
	tillTime := t.Sub(now)
	time.AfterFunc(tillTime, func() {
		// Call f at the time specified
		go f()

		// If we have an interval, call it again repeatedly after interval
		// stopping if the caller calls stop(task) on returned channel
		if i > 0 {
			ticker := time.NewTicker(i)
			go func() {
				for {
					select {
					case <-ticker.C:
						go f()
					case <-task:
						ticker.Stop()
						return
					}
				}
			}()
		}
	})

	return task // call close(task) to stop executing the task for repeated tasks
}

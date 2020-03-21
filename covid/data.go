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

var dataPath = "./data"
var dataFiles = []string{
	"https://raw.githubusercontent.com/CSSEGISandData/COVID-19/master/csse_covid_19_data/csse_covid_19_time_series/time_series_19-covid-Confirmed.csv",
	"https://raw.githubusercontent.com/CSSEGISandData/COVID-19/master/csse_covid_19_data/csse_covid_19_time_series/time_series_19-covid-Deaths.csv",
	"https://raw.githubusercontent.com/CSSEGISandData/COVID-19/master/csse_covid_19_data/csse_covid_19_time_series/time_series_19-covid-Recovered.csv",
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

	// Load all our data files
	for _, fp := range files {
		data, err = loadCSVFile(fp, data)
		if err != nil {
			return err
		}
	}

	// Post-process the data after loading (it doesn't include global US counts for example)
	data = processData(data)

	log.Printf("server: loaded data in %s len:%d", time.Now().Sub(start), len(data))

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
// call via LoadData above
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

// ScheduleDataFetch sets up a regular fetch of data from data sources
func ScheduleDataFetch() {
	// Set up a scheduled time every day at 8PM UTC
	now := time.Now()
	when := time.Date(now.Year(), now.Month(), now.Day(), 3, 33, 0, 0, time.UTC)
	daily := time.Hour * 24 // daily

	// For debug, test straight away
	//when = now.Add(5 * time.Second)

	// Schedule the fetch
	ScheduleAt(FetchDataDaily, when, daily)
}

// FetchDataDaily is called on a schedule
func FetchDataDaily() {
	log.Printf("daily: fetching data from data source")
	err := FetchData()
	if err != nil {
		log.Printf("daily: error fetching data from data source:%s", err)
	}
}

// FetchData fetches data from our data sources
// for more frequent updates, we could look at downloading the case data
func FetchData() error {

	// First download the 3 x files we need from github master branch to our data dir
	err := DownloadFiles(dataFiles, dataPath)
	if err != nil {
		return err
	}

	// Trigger a reload of the data from our standard data path
	return LoadData()
}

// DownloadFiles downloads the specified url to the specified file path
// requires csv files
func DownloadFiles(urls []string, dataPath string) error {

	for _, url := range urls {
		log.Printf("daily: downloading file %s", url)

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
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0700)
		if err != nil {
			return fmt.Errorf("data: error opening file:%s", err)
		}
		defer f.Close()

		// Write the body to file
		_, err = io.Copy(f, resp.Body)
		if err != nil {
			return fmt.Errorf("data: error copying data url:%s path:%s error:%s", url, path, err)
		}

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

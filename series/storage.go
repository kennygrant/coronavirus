package series

import (
	"fmt"
	"log"
	"os"
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

// AddToday adds a day to our dataset
// usually called after zero hours
func AddToday() error {

	// If we don't have it already, add a set of data for today
	// Lock during add operation
	mutex.Lock()
	err := dataset.AddToday()
	if err != nil {
		return fmt.Errorf("series: failed to add today on series data:%s", err)
	}
	mutex.Unlock()

	err = Save("data/series.csv")
	if err != nil {
		return fmt.Errorf("series: failed to save series data:%s", err)
	}

	return nil
}

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
	err = Load(seriesPath)
	if err != nil {
		return err
	}

	// Add today if we don't have it
	err = dataset.AddToday()
	if err != nil {
		return fmt.Errorf("series: failed to add today on series data:%s", err)
	}

	// Finally sort the dataset by deaths, then alphabetically by country/province
	sort.Stable(dataset)

	// For debug, print today's data after load
	//dataset.PrintToday()

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

// Save saves the existing series to a file at the path given
// this is used for automatic updates of data from data sources
func Save(p string) error {

	if len(dataset) == 0 {
		return fmt.Errorf("series: save on empty data set")
	}

	days := len(dataset[0].Days)
	if days == 0 {
		return fmt.Errorf("series: save on empty data set")
	}

	// Lock during save operation
	mutex.Lock()
	defer mutex.Unlock()

	var seriesData [][]int

	// Sort the dataset by id for saving
	sort.Slice(dataset, func(i, j int) bool {
		return dataset[i].ID < dataset[j].ID
	})

	// We save the series per day rather than every series at once
	// so that additional days are at the end of the file
	// For every series, save the data to an array (if non-zero)
	// it would perhaps be more intuitive to order by area_id instead first
	for i := 0; i < days; i++ {
		dayNumber := i + 1
		for _, s := range dataset {
			d := s.Days[i]
			if !d.IsZero() {
				seriesData = append(seriesData, []int{dayNumber, s.ID, d.Deaths, d.Confirmed, d.Recovered, d.Tested})
			}
		}
	}

	// Resort the dataset in our preferred order
	sort.Sort(dataset)

	// Write the data out to files - our data is simple so we write directly
	headerRow := fmt.Sprintf("day,area_id,deaths,confirmed,recovered,tested\n")
	var row string

	// SERIES DATA file is saved at path given, over existing file if required
	f, err := os.Create(p)
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

	return nil
}

// Load loads our global series file
// this contains all data in the sparse format (no rows for zero data):
// day, area_id, deaths, confirmed, recovered, tested
// dataset must be locked while performing this operation
func Load(p string) error {

	// Open the CSV file - one row per day per area
	rows, err := loadCSV(p)
	if err != nil {
		return err
	}

	// Check the day number on the last row in the series - we want this many days to load into
	// we assume we start from 1 up to this day number
	// this may or may not include today
	days, err := strconv.Atoi(rows[len(rows)-1][0])
	if err != nil {
		// If dayno read fails, fall back to days up to but not including today
		days = int(time.Now().UTC().Sub(seriesStartDate).Hours() / 24)
	}

	log.Printf("load: loading series:%s days:%d", p, days)

	// For every series add the right number of days up to but not including today
	// these days are initially zeroed out before loading from the file
	for _, series := range dataset {
		series.AddDays(days)
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
			log.Printf("series: series not found for id:%d index:%d row:%v", values[1], i, row)
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

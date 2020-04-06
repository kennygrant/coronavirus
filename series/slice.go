package series

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// Slice is a collection of Series - stored in our pkg global dataset variable
type Slice []*Data

func (slice Slice) Len() int      { return len(slice) }
func (slice Slice) Swap(i, j int) { slice[i], slice[j] = slice[j], slice[i] }

// Sort first on number of deaths, then on alpha order
func (slice Slice) Less(i, j int) bool {
	if slice[i].TotalDeaths() > 0 || slice[j].TotalDeaths() > 0 {
		return slice[i].TotalDeaths() > slice[j].TotalDeaths()
	}
	return slice[i].Country < slice[j].Country
}

// AddToday adds a day for today's date to the end of the dataset
// if today already exists on the global slice, it does nothing
func (slice Slice) AddToday() error {

	if len(slice) < 1 {
		log.Printf("series: addtoday empty dataset")
		return fmt.Errorf("series: attempt to add today to empty dataset")
	}

	// Work out whether we already have today in the first slice data
	// NB we assume a certain start date for today
	days := int(time.Now().UTC().Sub(seriesStartDate).Hours()/24) + 1
	if days <= len(slice[0].Days) {
		log.Printf("series: addtoday have enough days:%d global days:%d", days, len(slice[0].Days))
		return nil
	}

	// If here we still need to add today to all our slices, based on data from yesterday
	for _, s := range slice {
		s.AddToday()
	}

	return nil
}

// FetchDate fetches the datapoint for a given datum and date
func (slice Slice) FetchDate(country, province string, datum int, date time.Time) (int, error) {
	// Find the series, if none found return 0
	series, err := slice.FetchSeries(country, province)
	if err != nil {
		return 0, err
	}
	if !series.Valid() {
		return 0, fmt.Errorf("series: no such series")
	}

	return series.FetchDate(date, datum), nil
}

// FetchSeries returns a series (if found) for this combination of country and province
func (slice Slice) FetchSeries(country string, province string) (*Data, error) {

	for _, s := range slice {
		if s.Match(country, province) {
			return s, nil
		}
	}

	return &Data{}, fmt.Errorf("series: not found")
}

// FindSeries returns a series (if found) for this ID
func (slice Slice) FindSeries(seriesID int) (*Data, error) {

	for _, s := range slice {
		if s.ID == seriesID {
			return s, nil
		}
	}

	return &Data{}, fmt.Errorf("series: not found")
}

// PrintSeries uses our stored data to fetch a series
func (slice Slice) PrintSeries(country string, province string) error {
	s, err := slice.FetchSeries(country, province)
	if err != nil {
		log.Printf("error: series err:%s %s", country, err)
		return err
	}
	log.Printf("series:%s,%s %v", s.Country, s.Province, s.Days)
	return nil
}

// PrintToday prints the data for today for debug purposes
func (slice Slice) PrintToday() {
	for _, s := range slice {
		day := s.LastDay()
		log.Printf("series:%s last day:%v", s, day)
	}
}

// CountryOptions returns a set of options for the country dropdown (including a global one)
func (slice Slice) CountryOptions() (options []Option) {

	options = append(options, Option{Name: "Global", Value: ""})

	for _, s := range slice {
		if s.Province == "" && s.Country != "" {
			name := s.Country
			if s.TotalDeaths() > 0 {
				name = fmt.Sprintf("%s (%d Deaths)", s.Country, s.TotalDeaths())
			}
			options = append(options, Option{Name: name, Value: s.Key(s.Country)})
		}
	}

	return options
}

// ProvinceOptions returns a set of options for the province dropdown
// this should probably be based on the current country selection, and filtered from there
// to avoid inconsistency
// for now just show all which have province filled in.
func (slice Slice) ProvinceOptions(country string) (options []Option) {

	options = append(options, Option{Name: "All Areas", Value: ""})

	// Some countries don't have complete data (France, Netherlands)
	// but we'll leave them in even though they don't have a proper regional breakdown
	// as the outlying areas are otherwise hidden from the dataset
	// Typically this is dependencies and former colonies (France etc)

	for _, s := range slice {
		if s.Country == country && s.Province != "" {
			name := s.Province
			if s.TotalDeaths() > 0 {
				name = fmt.Sprintf("%s (%d Deaths)", s.Province, s.TotalDeaths())
			}
			options = append(options, Option{Name: name, Value: s.Key(s.Province)})
		}
	}

	return options
}

// MergeCSV merges the data in this CSV with the data we already have in the Slice
func (slice Slice) MergeCSV(records [][]string, dataType int) (Slice, error) {

	// If daily data, merge it to existing last date
	switch dataType {
	case DataTodayCountry:
		return slice.mergeDailyCountryCSV(records, dataType)
	case DataTodayState:
		return slice.mergeDailyStateCSV(records, dataType)
	}

	return slice.mergeTimeSeriesCSV(records, dataType)
}

// mergeTimeSeriesCSV merges the data in this time series CSV with the data we already have in the Slice
func (slice Slice) mergeTimeSeriesCSV(records [][]string, dataType int) (Slice, error) {

	// Make an assumption about the starting date (checked below on header row)
	date := time.Date(2020, 1, 22, 0, 0, 0, 0, time.UTC)

	for i, row := range records {

		// Check header to see this is the file we expect, if not skip
		if i == 0 {
			// We just check a few cols - we assume the start date of the data won't change
			if row[0] != "Province/State" || row[1] != "Country/Region" || row[2] != "Lat" || row[4] != "1/22/20" {
				return slice, fmt.Errorf("load: error loading file - time series csv data format invalid")
			}

		} else {

			// Fetch data to match series
			country := row[1]
			province := row[0]

			// We ignore rows which match ,CA etc
			// these are US sub-state level data which is no longer included in the dataset and is zeroed out
			if country == "US" && strings.Contains(province, ", ") {
				//	fmt.Printf("ignoring series:%s %s\n", country, province)
				continue
			}

			// Ignore duplicate Virgin Islands
			if province == "Virgin Islands, U.S." {
				continue
			}

			// Fetch the series
			var series *Data
			series, _ = slice.FetchSeries(country, province)

			// If we don't have one yet, create one
			if !series.Valid() {
				series = &Data{
					Country:  country,
					Province: province,
				}
				slice = append(slice, series)
			}

			// Walk through row, reading days data after col 3 (longitude)
			for ii, d := range row {
				if ii < 4 {
					continue
				}
				var v int
				var err error
				if d != "" {
					v, err = strconv.Atoi(d)
					if err != nil {
						log.Printf("load: error loading series:%s row:%d col:%d row:\n%s\nerror:%s", country, i+1, ii+1, row, err)
						return slice, fmt.Errorf("load: error loading row %d - csv day data invalid:%s", i+1, err)
					}
				} else {
					// This is typically a clerical error - in this case invalid rows ending in ,
					// So just quietly ignore it
					log.Printf("load: missing data for series:%s row:%d col:%d", country, i, ii)
				}
				deaths, confirmed, recovered, tested := -1, -1, -1, -1

				switch dataType {
				case DataDeaths:
					deaths = v
					series.AddDay(date, deaths, confirmed, recovered, tested)
					//series.Deaths = append(series.Deaths, v)
				case DataConfirmed:
					confirmed = v
					series.AddDay(date, deaths, confirmed, recovered, tested)
					//series.Confirmed = append(series.Confirmed, v)
				}
			}
			date = date.AddDate(0, 0, 1)
		}

	}
	return slice, nil
}

// mergeDailyCountryCSV merges the data in this country daily series CSV with the data we already have in the Slice
func (slice Slice) mergeDailyCountryCSV(records [][]string, dataType int) (Slice, error) {

	//	log.Printf("load: merge daily country csv")

	// Make an assumption about the starting date - if this changes update
	date := time.Date(2020, 1, 22, 0, 0, 0, 0, time.UTC)

	// Calculate index in series given shared StartsAt vs today (we assume data in these files is for today)
	days := time.Now().UTC().Sub(date)
	dayIndex := int(days.Hours() / 24)

	// Bounds check index
	if dayIndex < 0 {
		return nil, fmt.Errorf("day index out of bounds")
	}

	for i, row := range records {
		// Check header to see this is the file we expect, if not skip
		if i == 0 {
			//Country_Region,Last_Update,Lat,Long_,Confirmed,Deaths,Recovered,Active
			// We just check a few cols - we assume the start date of the data won't change
			if row[0] != "Country_Region" || row[1] != "Last_Update" || row[2] != "Lat" || row[4] != "Confirmed" {
				return slice, fmt.Errorf("load: error loading file - daily country csv data format invalid")
			}

		} else {

			// Fetch data to match series
			country := row[0]
			province := ""

			// Fix data
			if country == "Taiwan*" {
				country = "Taiwan"
			}

			// Fetch the series
			series, err := slice.FetchSeries(country, province)
			if err != nil {
				log.Printf("load: warning reading daily series:%s error:%s", row[0], err)
				//return nil, fmt.Errorf("load: error reading daily series:%s error:%s", row[0], err)
			}

			// Get the series data from the row
			updated, confirmed, deaths, err := readCountryRow(row)
			if err != nil {
				return nil, fmt.Errorf("load: error reading row series:%s error:%s", row[0], err)
			}
			series.SetUpdated(updated)
			// blank for now - not sure date will be right here?
			recovered, tested := -1, -1
			series.AddDay(date, deaths, confirmed, recovered, tested)
			date = date.AddDate(0, 0, 1)

		}

	}
	return slice, nil
}

// mergeDailyStateCSV merges the data in this state daily series CSV with the data we already have in the Slice
func (slice Slice) mergeDailyStateCSV(records [][]string, dataType int) (Slice, error) {

	//log.Printf("load: merge daily state csv")

	// Make an assumption about the starting date - if this changes update
	date := time.Date(2020, 1, 22, 0, 0, 0, 0, time.UTC)

	// Calculate index in series given shared StartsAt vs today (we assume data in these files is for today)
	days := time.Now().UTC().Sub(date)
	dayIndex := int(days.Hours() / 24)

	// Bounds check index
	if dayIndex < 0 {
		return nil, fmt.Errorf("day index out of bounds")
	}

	for i, row := range records {
		// Check header to see this is the file we expect, if not skip
		if i == 0 {
			//FIPS,Province_State,Country_Region,Last_Update,Lat,Long_,Confirmed,Deaths,Recovered,Active
			// We just check a few cols - we assume the start date of the data won't change
			if row[0] != "FIPS" || row[1] != "Province_State" || row[2] != "Country_Region" || row[6] != "Confirmed" {
				return slice, fmt.Errorf("load: error loading file - daily country csv data format invalid")
			}

		} else {

			// Fetch data to match series
			country := row[2]
			province := row[1]

			if province == "Virgin Islands, U.S" {
				continue
			}

			// There are several province series with bad names or dates which are duplicated in the state level dataset
			// we therefore ignore them here as the data seems to be out of date anyway

			// Fetch the series
			series, err := slice.FetchSeries(country, province)
			if err != nil {
				//	log.Printf("load: warning reading daily series:%s error:%s", row[1], err)
				continue
			}

			// Get the series data from the row
			updated, confirmed, deaths, err := readStateRow(row)
			if err != nil {
				return nil, fmt.Errorf("load: error reading row series:%s error:%s", row[1], err)
			}

			// Update updated at date
			series.SetUpdated(updated)

			// blank for now
			recovered, tested := -1, -1
			series.AddDay(date, deaths, confirmed, recovered, tested)
			date = date.AddDate(0, 0, 1)
		}

	}
	return slice, nil
}

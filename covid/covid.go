package covid

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Data types for imported series data
const (
	DataDeaths = iota
	DataConfirmed
	DataRecovered
)

// Series stores data for one country or province within a country
type Series struct {
	// The Country or Region
	Country string
	// The Province or State - may be blank for countries
	Province string
	// The date at which the series starts - all datasets must be the same length
	StartsAt time.Time
	// Total Deaths, Confirmed or Recovered by day (cumulative)
	Deaths    []int
	Confirmed []int
	Recovered []int
}

// Title returns a display title for this series
func (s *Series) Title() string {
	if s.Country == "" && s.Province == "" {
		return "Global"
	} else if s.Province == "" {
		return s.Country
	}

	return fmt.Sprintf("%s (%s)", s.Province, s.Country)
}

// Dates returns a set of date labels as an array of strings
// for every datapoint in this series
func (s *Series) Dates() (dates []string) {
	d := s.StartsAt
	for range s.Deaths {
		dates = append(dates, d.Format("Jan 2"))
		d = d.AddDate(0, 0, 1)
	}
	return dates
}

// FetchDate retusn the data for the given data from datum
func (s *Series) FetchDate(datum int, date time.Time) int {

	// Calculate index in series given StartsAt
	days := date.Sub(s.StartsAt)
	i := int(days.Hours() / 24)

	// Bounds check index
	if i < 0 || i > len(s.Deaths)-1 {
		return 0
	}

	// Fetch the data at index
	switch datum {
	case DataDeaths:
		return s.Deaths[i]
	case DataConfirmed:
		return s.Confirmed[i]
	case DataRecovered:
		return s.Recovered[i]
	}
	return 0
}

// Valid returns true if this series is valid
// a series without a start date set is considered invalid
func (s *Series) Valid() bool {
	return !s.StartsAt.IsZero()
}

// Key converts a value into one suitable for use in urls
func (s *Series) Key(v string) string {
	return strings.Replace(strings.ToLower(v), " ", "-", -1)
}

// Match returns true if this series matches data from a row
// performs a case insensitive match
func (s *Series) Match(country string, province string) bool {
	return s.Key(s.Country) == s.Key(country) && s.Key(s.Province) == s.Key(province)
}

// Merge the data from the incoming series with ours
func (s *Series) Merge(series *Series) {
	// If we are len 0 make sure we have enough space
	if len(s.Deaths) == 0 {
		s.Deaths = make([]int, len(series.Deaths))
		s.Confirmed = make([]int, len(series.Confirmed))
		s.Recovered = make([]int, len(series.Recovered))
	}

	// Then add to the data we have (if any)
	for i, d := range series.Deaths {
		s.Deaths[i] += d
	}
	for i, d := range series.Confirmed {
		s.Confirmed[i] += d
	}
	for i, d := range series.Recovered {
		s.Recovered[i] += d
	}
}

// Format formats a given number for display and returns a string
func (s *Series) Format(i int) string {
	if i < 1000 {
		return fmt.Sprintf("%d", i)
	}
	if i < 1000000 {
		return fmt.Sprintf("%.2fk", float64(i)/1000)
	}
	return fmt.Sprintf("%.2fm", float64(i)/1000000)
}

// DeathsDisplay returns a string representation of TotalDeaths
func (s *Series) DeathsDisplay() string {
	return s.Format(s.TotalDeaths())
}

// ConfirmedDisplay returns a string representation of TotalConfirmed
func (s *Series) ConfirmedDisplay() string {
	return s.Format(s.TotalConfirmed())
}

// RecoveredDisplay returns a string representation of TotalRecovered
func (s *Series) RecoveredDisplay() string {
	return s.Format(s.TotalRecovered())
}

// TotalDeaths returns the cumulative death due to COVID-19 for this series
func (s *Series) TotalDeaths() int {
	if len(s.Deaths) == 1 || len(s.Deaths) > 60 {
		return s.Deaths[len(s.Deaths)-1]
	}
	return s.Deaths[len(s.Deaths)-1] - s.Deaths[0]
}

// TotalConfirmed returns the cumulative confirmed cases of COVID-19 for this series
func (s *Series) TotalConfirmed() int {
	if len(s.Confirmed) == 1 || len(s.Confirmed) > 60 {
		return s.Confirmed[len(s.Confirmed)-1]
	}
	return s.Confirmed[len(s.Confirmed)-1] - s.Confirmed[0]
}

// TotalRecovered returns the cumulative confirmed cases of COVID-19 for this series
func (s *Series) TotalRecovered() int {
	if len(s.Recovered) == 1 || len(s.Recovered) > 60 {
		return s.Recovered[len(s.Recovered)-1]
	}
	return s.Recovered[len(s.Recovered)-1] - s.Recovered[0]
}

// Days returns a copy of this series for just the given number of days in the past
func (s *Series) Days(days int) *Series {
	i := len(s.Deaths) - days
	return &Series{
		Country:   s.Country,
		Province:  s.Province,
		StartsAt:  s.StartsAt.AddDate(0, 0, days),
		Deaths:    s.Deaths[i:],
		Confirmed: s.Confirmed[i:],
		Recovered: s.Recovered[i:],
	}
}

// SLICE OF Series

// SeriesSlice is a collection of Series
type SeriesSlice []*Series

func (slice SeriesSlice) Len() int      { return len(slice) }
func (slice SeriesSlice) Swap(i, j int) { slice[i], slice[j] = slice[j], slice[i] }

// Sort first on number of deaths, then on alpha order
func (slice SeriesSlice) Less(i, j int) bool {
	if slice[i].TotalDeaths() > 0 {
		return slice[i].TotalDeaths() > slice[j].TotalDeaths()
	}
	return slice[i].Country < slice[j].Country
}

// FetchDate fetches the datapiont for a given datum and date
func (slice SeriesSlice) FetchDate(country, province string, datum int, date time.Time) (int, error) {
	// Find the series, if none found return 0
	series, err := slice.FetchSeries(country, province)
	if err != nil {
		return 0, err
	}
	if !series.Valid() {
		return 0, fmt.Errorf("series: no such series")
	}

	return series.FetchDate(datum, date), nil
}

// FetchSeries returns a series (if found) for this combination of country and province
func (slice SeriesSlice) FetchSeries(country string, province string) (*Series, error) {

	for _, s := range slice {
		if s.Match(country, province) {
			return s, nil
		}
	}

	return &Series{}, fmt.Errorf("series: not found")
}

// Option is used to generate options for selects in the view
type Option struct {
	Name  string
	Value string
}

// PeriodOptions returns a set of options for period filters
func (slice SeriesSlice) PeriodOptions() (options []Option) {

	options = append(options, Option{Name: "All Time", Value: "0"})
	options = append(options, Option{Name: "1 Day", Value: "1"})
	options = append(options, Option{Name: "2 Days", Value: "2"})
	options = append(options, Option{Name: "3 Days", Value: "3"})
	options = append(options, Option{Name: "7 Days", Value: "7"})
	options = append(options, Option{Name: "14 Days", Value: "14"})
	options = append(options, Option{Name: "28 Days", Value: "28"})

	return options
}

// CountryOptions returns a set of options for the country dropdown (including a global one)
func (slice SeriesSlice) CountryOptions() (options []Option) {

	options = append(options, Option{Name: "Global", Value: ""})

	for _, s := range slice {
		if s.Province == "" && s.Country != "" {
			options = append(options, Option{Name: s.Country, Value: s.Key(s.Country)})
		}
	}

	return options
}

// ProvinceOptions returns a set of options for the province dropdown
// this should probably be based on the current country selection, and filtered from there
// to avoid inconsistency
// for now just show all which have province filled in.
func (slice SeriesSlice) ProvinceOptions(country string) (options []Option) {

	options = append(options, Option{Name: "Areas", Value: ""})

	for _, s := range slice {
		if s.Country == country && s.Province != "" {
			options = append(options, Option{Name: s.Province, Value: s.Key(s.Province)})
		}
	}

	return options
}

// MergeCSV merges the data in this CSV with data
func (slice SeriesSlice) MergeCSV(records [][]string, dataType int) (SeriesSlice, error) {

	// Make an assumption about the starting date (checked below on header row)
	startDate := time.Date(2020, 1, 22, 0, 0, 0, 0, time.UTC)

	for i, row := range records {
		// Check header to see this is the file we expect, if not skip
		if i == 0 {
			// We just check a few cols - we assume the start date of the data won't change
			if row[0] != "Province/State" || row[1] != "Country/Region" || row[2] != "Lat" || row[4] != "1/22/20" {
				return slice, fmt.Errorf("load: error loading file - csv data format invalid")
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

			// Fetch the series
			var series *Series
			series, _ = slice.FetchSeries(country, province)

			// If we don't have one yet, create one
			if !series.Valid() {
				series = &Series{
					Country:  country,
					Province: province,
					StartsAt: startDate,
				}
				slice = append(slice, series)
			}

			// Walk through row, reading days data after col 3 (longitude)
			for i, d := range row {
				if i < 4 {
					continue
				}
				v, err := strconv.Atoi(d)
				if err != nil {
					return slice, fmt.Errorf("load: error loading row %d - csv day data invalid:%s", i, err)
				}
				switch dataType {
				case DataDeaths:
					series.Deaths = append(series.Deaths, v)
				case DataConfirmed:
					series.Confirmed = append(series.Confirmed, v)
				case DataRecovered:
					series.Recovered = append(series.Recovered, v)
				}
			}

		}

	}
	return slice, nil
}

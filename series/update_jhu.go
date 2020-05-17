package series

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

// UpdateFromJHUCountryCases updates from JHU country cases data files
// several files are required to get all data, all with different formats
// Cols: Country_Region,Last_Update,Lat,Long_,Confirmed,Deaths,Recovered,Active
func UpdateFromJHUCountryCases(rows [][]string) error {

	// Lock during add operation
	mutex.Lock()
	defer mutex.Unlock()

	log.Printf("series: update from JHU country cases %d rows", len(rows))

	// For each row in the input data, reject if admin2 completed
	for i, row := range rows {
		// Check format on row 0
		if i == 0 {
			if row[0] != "Country_Region" || row[1] != "Last_Update" || row[7] != "Active" {
				return fmt.Errorf("error reading JHU country cases - format invalid for row:%s", row)
			}
			continue
		}

		country := row[0]
		province := ""

		// Skip some countries for which we have data from a country provider
		if country == "United Kingdom" {
			continue
		}

		// Transform countries
		switch country {
		case "Burma":
			country = "Myanmar"
		case "Taiwan*":
			country = "Taiwan"
		case "Korea, South":
			country = "South Korea"
		}
		// Find the series for this row
		series, err := dataset.FetchSeries(country, province)
		if err != nil || series == nil {
			continue
		}

		// If we reach here we have a valid row and series - NB shuffled cols to match our default
		updated, deaths, confirmed, recovered, err := readJHURowData(row[1], row[5], row[4], row[6])
		if err != nil {
			log.Printf("update: error updating series:%s error:%s", series, err)
			continue
		}

		// We don't hav etested data from JHU so leave it unchanged
		series.UpdateToday(updated, deaths, confirmed, recovered, 0)

		log.Printf("update: %s u:%v d:%d c:%d r:%d", series, updated, deaths, confirmed, recovered)

	}

	return nil
}

// UpdateFromJHUStatesCases updates from JHU states cases data files
// several files are required to get all data, all with different formats
//  0    1    			2				3			4  5     	6		7		8		9
// FIPS,Province_State,Country_Region,Last_Update,Lat,Long_,Confirmed,Deaths,Recovered,Active
func UpdateFromJHUStatesCases(rows [][]string) error {

	// Lock during add operation
	mutex.Lock()
	defer mutex.Unlock()

	log.Printf("series: update from JHU states cases %d rows", len(rows))

	// For each row in the input data, reject if admin2 completed
	for i, row := range rows {
		// Check format on row 0
		if i == 0 {
			if row[0] != "Province_State" || row[1] != "Country_Region" || row[2] != "Last_Update" || row[8] != "Active" {
				return fmt.Errorf("error reading JHU states cases - format invalid for row:%s", row)
			}
			continue
		}

		country := row[1]
		province := row[0]

		// Rename or ignore some series
		switch province {
		case "Falkland Islands (Malvinas)":
			province = "Falkland Islands"
		case "British Virgin Islands":
			province = "Virgin Islands"
		case "Grand Princess":
			continue
		case "Diamond Princess":
			continue
		case "Recovered":
			continue
		}

		// Find the series concerned
		series, err := dataset.FetchSeries(country, province)
		if err != nil || series == nil {
			log.Printf("series: state series not found for:%s,%s", country, province)
			continue
		}

		// If we reach here we have a valid row and series - NB shuffled cols to match our default
		updated, deaths, confirmed, recovered, err := readJHURowData(row[2], row[6], row[5], row[7])
		if err != nil {
			log.Printf("series: error reading state row:%s\n\terror:%s", row, err)
			continue
		}

		// We don't have tested data from JHU so leave it unchanged
		series.UpdateToday(updated, deaths, confirmed, recovered, 0)

		//	log.Printf("update province: %s u:%v d:%d c:%d r:%d", series, updated, deaths, confirmed, recovered)

	}

	return nil
}

// Note csv col order is different from our standard order
func readJHURowData(updatedstr, deathsstr, confirmedstr, recoveredstr string) (time.Time, int, int, int, error) {

	var err error
	var d, c, r float64
	var deaths, confirmed, recovered int
	updated := time.Now().UTC()

	// Ignore dates which are not present (this applies to all non-use series now)
	if updatedstr != "" {
		updated, err := time.Parse("2006-01-02 15:04:05", updatedstr)
		if err != nil {
			return updated, 0, 0, 0, fmt.Errorf("load: error reading updated at series:%s error:%s", updatedstr, err)
		}
	}

	// Deal with inconsistent data like empty entries
	// Inexplicably, the data from JHU now comes as floats (so you can half a death presumably)

	if deathsstr != "" {
		d, err = strconv.ParseFloat(deathsstr, 32)
		if err != nil {
			return updated, 0, 0, 0, fmt.Errorf("load: error reading deaths series:%s error:%s", deathsstr, err)
		}
		deaths = int(d)
	}

	if confirmedstr != "" {
		c, err = strconv.ParseFloat(confirmedstr, 32)
		if err != nil {
			return updated, 0, 0, 0, fmt.Errorf("load: error reading confirmed series:%s error:%s", confirmedstr, err)
		}
		confirmed = int(c)
	}

	if recoveredstr != "" {
		r, err = strconv.ParseFloat(recoveredstr, 32)
		if err != nil {
			return updated, 0, 0, 0, fmt.Errorf("load: error reading recovered series:%s error:%s", recoveredstr, err)
		}
		recovered = int(r)
	}

	return updated, deaths, confirmed, recovered, nil
}

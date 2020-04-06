package series

import ()

// FetchSeries uses our stored dataset to fetch a series
func FetchSeries(country string, province string) (*Data, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return dataset.FetchSeries(country, province)
}

// FindSeries uses our stored dataset to fetch a series by series id
func FindSeries(seriesID int) (*Data, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return dataset.FindSeries(seriesID)
}

// SelectedEuropeanSeries selects a set of comparative series of interest from Europe
func SelectedEuropeanSeries(country string, n int) Slice {
	mutex.RLock()
	defer mutex.RUnlock()

	// Fetch all top series
	var count int
	var collection Slice
	for _, s := range dataset {
		if count >= n {
			break
		}

		// Exclude global series
		if s.IsGlobal() {
			continue
		}

		// Exclude provinces
		if s.IsProvince() {
			continue
		}

		// Always include the country
		if s.Country == country {
			collection = append(collection, s)
			count++
		} else if s.Country == "Italy" || s.Country == "Spain" || s.Country == "France" || s.Country == "Switzerland" || s.Country == "Germany" || s.Country == "United Kingdom" {
			collection = append(collection, s)
			count++
		}

	}

	return collection

}

// SelectedSeries selects a set of comparative series of interest
func SelectedSeries(country string, n int) Slice {
	mutex.RLock()
	defer mutex.RUnlock()

	// Fetch all top series
	var count int
	var collection Slice
	for _, s := range dataset {
		if count >= n {
			break
		}

		// Exclude global series
		if s.IsGlobal() {
			continue
		}

		// Exclude provinces for now
		if s.IsProvince() {
			continue
		}

		// Always include country
		if s.Country == country {
			collection = append(collection, s)
			count++
		} else if country == "Spain" || country == "US" || country == "United Kingdom" || country == "China" || country == "Japan" {
			collection = append(collection, s)
			count++
		}

	}

	return collection

}

// TopSeries selects the top n series by deaths
func TopSeries(country string, n int) Slice {
	mutex.RLock()
	defer mutex.RUnlock()

	// Fetch all top series
	var count int
	var collection Slice
	for _, s := range dataset {
		if count >= n {
			break
		}

		// Exclude global series
		if s.IsGlobal() {
			continue
		}

		// Append all *countries* if global series is given
		if country == "" && !s.IsProvince() {
			collection = append(collection, s)
			count++
		} else if s.MatchCountry(country) && s.IsProvince() {
			// Append any provinces for this country if a country series is given
			collection = append(collection, s)
			count++
		}

	}

	return collection
}

// DataSet - REMOVE AFTER SETUP FIXME this is not thread safe - it is only called to construct initial series data from JHU historical data
func DataSet() Slice {
	return dataset
}

package series

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
		} else if s.Country == "Italy" || s.Country == "Spain" || s.Country == "France" || s.Country == "Switzerland" || s.Country == "Germany" || s.Country == "United Kingdom" || s.Country == "Sweden" || s.Country == "Netherlands" {
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

	var count int
	var collection Slice

	// Always include this country in the selected series
	countrySeries, err := FetchSeries(country, "")
	if err == nil {
		collection = append(collection, countrySeries)
	}

	// Fetch all top series
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

		// Skip country already added
		if s == countrySeries {
			continue
		}

		switch s.Country {
		case "Italy":
			fallthrough
		case "US":
			fallthrough
		case "Japan":
			fallthrough
		case "Brazil":
			fallthrough
		case "Germany":
			fallthrough
		case "Iran":
			fallthrough
		case "Sweden":
			fallthrough
		case "United Kingdom":
			collection = append(collection, s)
			count++
		}
	}

	return collection

}

// TopSeriesGlobal selects the top n series by deaths for global page
func TopSeriesGlobal(country string, n int) Slice {
	mutex.RLock()
	defer mutex.RUnlock()

	// Fetch all top series
	var count int
	var collection Slice
	for _, s := range dataset {
		if count >= n {
			break
		}
		// Exclude China - distorts chart and figures unreliable
		if s.Country == "China" {
			continue
		}

		// Exclude global series
		if s.IsGlobal() {
			continue
		}

		// Append all *countries* up to count
		if !s.IsProvince() {
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

		// Append all provinces of this country
		if s.MatchCountry(country) && s.IsProvince() {
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

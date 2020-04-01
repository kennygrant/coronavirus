package series

import ()

// FetchSeries uses our stored dataset to fetch a series
func FetchSeries(country string, province string) (*Data, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return dataset.FetchSeries(country, province)
}

// TopSeries selects the top n series by deaths
func TopSeries(country string, n int) Slice {
	mutex.RLock()
	defer mutex.RUnlock()

	var count int

	// Need to get those matching country
	var collection Slice
	for _, s := range dataset {
		if count >= n {
			break
		}

		// Exclude global series
		if s.Global() {
			continue
		}

		if country == "" && s.Province == "" {
			// Append all *countries* if country is blank
			collection = append(collection, s)
			count++
		} else if s.Country == country && s.Province != "" {
			// Else in a country so append any provinces for that country
			collection = append(collection, s)
			count++
		}
	}

	return collection
}

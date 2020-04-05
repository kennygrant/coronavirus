package series

// Option is used to generate options for selects in the view
type Option struct {
	Name  string
	Value string
}

// PeriodOptions returns a set of options for period filters
func PeriodOptions() (options []Option) {

	options = append(options, Option{Name: "All Time", Value: "-1"})
	options = append(options, Option{Name: "112 Days", Value: "112"})
	options = append(options, Option{Name: "56 Days", Value: "56"})
	options = append(options, Option{Name: "28 Days", Value: "28"})
	options = append(options, Option{Name: "14 Days", Value: "14"})
	options = append(options, Option{Name: "7 Days", Value: "7"})
	options = append(options, Option{Name: "3 Days", Value: "3"})

	return options
}

// CountryOptions uses our stored dataset to fetch country options
func CountryOptions() (options []Option) {
	mutex.RLock()
	defer mutex.RUnlock()
	return dataset.CountryOptions()
}

// ProvinceOptions uses our stored dataset to fetch province options for a country
func ProvinceOptions(country string) (options []Option) {
	mutex.RLock()
	defer mutex.RUnlock()
	return dataset.ProvinceOptions(country)
}

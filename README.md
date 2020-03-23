
# Covid-19 (Novel Coronavirus) Global Charts

This project uses data from to product charts of deaths, confirmed cases and recovered cases from countries globally, based on data provided by <a href="https://systems.jhu.edu/research/public-health/ncov/">John Hopkins University</a>. The source data is available <a href="https://github.com/CSSEGISandData/COVID-19">here</a> on github and is updated daily from that source.  

This project and all code and data transformations are public domain and free in every sense, corrections and contributions are welcome. 

# View the data

You can see the data at https://coronavirus.projectpage.app/

You can also access a json feed for any of the series at urls like this: https://coronavirus.projectpage.app/global.json

Data is held in memory on the server so response times should be fast even under load. 

# Notes on Data
There are some inconsistencies in the source data, which where possible have been corrected. All changes made to the data are outlined below. 

* Added Global time series which is a sum of all other series 
* Added China overall data - a sum of all other series with country China
* Added US overall data - a sum of all other series with country US
* Added Austrilia overall data - a sum of all other series with country Austrilia
* Added Canada overall data - a sum of all other series with country Canada

Known problems: 

* US county time series data is no longer updated and has been omitted
* US recovered cases appear to no longer be updated, but have been left in the data as-is
* UK does not have a full area breakdown at present


# Installation 

There are no external requirements except a working go install to build, data is read from CSV files and stored in memory. The server can be compiled and run locally with: 

COVID=dev go run main.go 

Today's data is updated hourly from the data source, historical time series data is updated once a day (for corrections). 

# License 

This code and any modified data is released as open source in the public domain, use it as you see fit. 

The author hereby disclaims any and all representations and warranties with respect to the code and website, including accuracy, fitness for use, and merchantability. Reliance on the website for medical guidance or use of the website in commerce is strictly prohibited.
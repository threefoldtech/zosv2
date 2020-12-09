package collectors

// Metric struct
type Metric struct {
	Name        string
	Descritpion string
}

//Collector interface
type Collector interface {
	// Collect run collection logic
	Collect() error
	// Metrics returns a list with all provided keys
	Metrics() []Metric
}

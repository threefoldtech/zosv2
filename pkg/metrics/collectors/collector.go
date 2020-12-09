package collectors

//Collector interface
type Collector interface {
	// Collect run collection logic
	Collect() error
	// Metrics returns a list with all provided keys
	Metrics() []string
}

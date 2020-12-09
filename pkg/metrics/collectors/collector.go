package collectors

//Collector interface
type Collector interface {
	Collect() error
}

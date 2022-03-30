package counter

//Count of routines
type Counter struct {
	N int64
}

func (c *Counter) Add() {
	c.N++
}

func (c *Counter) Sub() {
	c.N--
}

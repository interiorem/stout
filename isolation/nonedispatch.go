package isolation

import "fmt"

type noneDispatch struct{}

func (d *noneDispatch) Handle(*message) (Dispatcher, error) {
	return d, fmt.Errorf("no transitions from NonDispatch")
}

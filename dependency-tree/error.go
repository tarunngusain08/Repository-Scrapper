package dependency_tree

import "log"

func (d *DependencyTree) errorHandler() {
	defer func() {
		d.wg.Done()
		log.Println("returning from errorHandler")
		close(d.ErrorChannel)
	}()
	for err := range d.ErrorChannel {
		log.Println(err)
	}
}

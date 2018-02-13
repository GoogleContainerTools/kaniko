package util

func Until(f func(), stop chan bool) {
	for {
		select {
		case <-stop:
			return
		default:
			f()
		}
	}
}

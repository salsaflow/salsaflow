package github

const MaxConcurrentRequests = 10

var requestSemaphore = make(chan struct{}, MaxConcurrentRequests)

func withRequestAllocated(body func()) {
	requestSemaphore <- struct{}{}
	defer func() {
		<-requestSemaphore
	}()

	body()
}

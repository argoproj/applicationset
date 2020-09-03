package refresher

import (
	"github.com/google/go-cmp/cmp"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

type Refresher struct {
	queue []ctrl.Request
}

func (r *Refresher) Start(stop <-chan struct{}, rateLimiter workqueue.RateLimiter) {
	ticker := time.NewTicker(3 * time.Minute)

	go func() {
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				for _, item := range r.queue {
					rateLimiter.When(item)
				}
			}
		}
	}()

}

func (r *Refresher) Add(item ctrl.Request) {
	exists := false
	for _, obj := range r.queue{
		if cmp.Equal(obj, item) {
			exists = true
			break
		}
	}

	if !exists {
		r.queue = append(r.queue, item)
	}

}
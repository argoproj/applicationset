package refresher

import (
	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"time"
)

type Refresher interface {
	Add(item ctrl.Request)
}


type refresher struct {
	queue []ctrl.Request
}

func Start(duration time.Duration, stop <-chan struct{}, events chan event.GenericEvent) Refresher {
	refresher := refresher{}
	ticker := time.NewTicker(duration)

	go func() {
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				log.WithFields(log.Fields{"queue": refresher.queue, "total": len(refresher.queue)}).Info("ticker event")
				for _, item := range refresher.queue {
					events <- event.GenericEvent{
						Meta: &metav1.ObjectMeta{
							Name:                       item.Name,
							Namespace:                  item.Namespace,
						},
					}
				}
			}
		}
	}()

	return &refresher
}

func (r *refresher) Add(item ctrl.Request) {
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

	log.WithFields(log.Fields{"queue": r.queue, "itme": item, "added": exists}).Debug("Refresher add item")
}
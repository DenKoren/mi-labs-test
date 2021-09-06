package registry

import (
	"log"
	"time"

	"github.com/denkoren/mi-labs-test/internal/core"
)

const defaultNotificationTimeout = 500 * time.Nanosecond
const defaultStatusSubscriptionCapacity = 2

type Subscription struct {
	C <-chan core.ContainerStatus

	id        int
	container *ContainerInfo
}

func (s Subscription) Unsubscribe() {
	s.container.Lock()
	defer s.container.Unlock()

	log.Printf("[Registry] removing container status change events subscription: Container: '%s', subscription: '%d'",
		s.container.ID,
		s.id,
	)

	sub, ok := s.container.subscribers[s.id]
	if !ok {
		return
	}

	close(sub)
	delete(s.container.subscribers, s.id)

	log.Printf("[Registry] container status change events subscription removed. Container: '%s', Subscripton: '%d'. Subscriptions count: '%d'",
		s.container.ID,
		s.id,
		len(s.container.subscribers),
	)

	go func() {
		// Read rest of events from subscription to prevent memory leaks.
		// Nobody should already listen this channel anyway
		for range s.C {
		}
	}()
}

type subscriber chan<- core.ContainerStatus

func (c *ContainerInfo) Subscribe() Subscription {
	c.Lock()
	defer c.Unlock()

	return c.subscribe()
}

func (c *ContainerInfo) subscribe() Subscription {
	subCh := make(chan core.ContainerStatus, defaultStatusSubscriptionCapacity)
	sub := Subscription{
		C: subCh,

		id:        c.nextSubscriberID,
		container: c,
	}

	c.subscribers[sub.id] = subCh
	c.nextSubscriberID++

	log.Printf("[Registry] new container status change events subscription. Container: '%s', Subscription: '%d'. Subscriptions count: '%d'",
		c.ID,
		sub.id,
		len(c.subscribers),
	)

	return sub
}

func (c *ContainerInfo) notifySubscribers(status core.ContainerStatus) {
	log.Printf("[Registry] notifying '%d' container subscribers about status change...", len(c.subscribers))
	for _, subCh := range c.subscribers {
		select {
		case subCh <- status:
		default:
			// Don't block on notifications. It is better to lose one, than to catch deadlock
			log.Printf("[Registry] status change notification skipped!")
		}
	}
}

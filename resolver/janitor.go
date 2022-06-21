package resolver

import "time"

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *janitor) Run(c *Resolver) {
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func stopJanitor(c *Resolver) {
	c.janitor.stop <- true
}

func runJanitor(c *Resolver, ci time.Duration) {
	j := &janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	c.janitor = j
	go j.Run(c)
}

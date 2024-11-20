package shared

import "time"

type TimerFunc func()

type Timer struct {
	ticker *time.Ticker
	f      TimerFunc
}

func NewTimer(period time.Duration, f TimerFunc) *Timer {
	timer := &Timer{
		ticker: time.NewTicker(period),
		f:      f,
	}
	go timer.run()

	return timer
}

func (t *Timer) Cancel() {
	t.ticker.Stop()
}

func (t *Timer) run() {
	for {
		select {
		case <-t.ticker.C:
			t.f()
		}
	}
}

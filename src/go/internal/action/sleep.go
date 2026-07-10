package action

import "time"

// SleepAction 延时等待
type SleepAction struct {
	seconds float64
}

func (a *SleepAction) Execute(eng Engine) error {
	time.Sleep(time.Duration(a.seconds * float64(time.Second)))
	return nil
}

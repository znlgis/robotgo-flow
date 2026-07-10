package action

// ScrollAction 滚动操作
type ScrollAction struct {
	amount int // >0 down, <0 up
}

func (a *ScrollAction) Execute(eng Engine) error {
	if a.amount > 0 {
		return eng.ScrollDown(a.amount)
	}
	if a.amount < 0 {
		return eng.ScrollUp(-a.amount)
	}
	return nil
}

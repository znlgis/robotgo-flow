package action

// PressKeyAction 按键操作
type PressKeyAction struct {
	key string
}

func (a *PressKeyAction) Execute(eng Engine) error {
	return eng.PressKey(a.key)
}

// PressComboAction 组合键操作
type PressComboAction struct {
	keys []string
}

func (a *PressComboAction) Execute(eng Engine) error {
	return eng.PressCombo(a.keys...)
}

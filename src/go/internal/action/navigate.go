package action

// OpenURLAction 打开 URL
type OpenURLAction struct {
	url string
}

func (a *OpenURLAction) Execute(eng Engine) error {
	eng.OpenURL(a.url)
	return nil
}

// RefreshAction 刷新页面
type RefreshAction struct{}

func (a *RefreshAction) Execute(eng Engine) error {
	return eng.RefreshPage()
}

// BackAction 后退
type BackAction struct{}

func (a *BackAction) Execute(eng Engine) error {
	return eng.Back()
}

// ForwardAction 前进
type ForwardAction struct{}

func (a *ForwardAction) Execute(eng Engine) error {
	return eng.Forward()
}

// SwitchTabAction 切换标签页
type SwitchTabAction struct {
	index int
}

func (a *SwitchTabAction) Execute(eng Engine) error {
	return eng.SwitchTab(a.index)
}

package action

import (
	"robotgo-flow/internal/geom"
)

// MockCall 记录 MockEngine 上的单次方法调用。
type MockCall struct {
	Name string
	Args []interface{}
}

// MockEngine 提供供测试使用的假 Engine。所有字段公开以便测试配置。
type MockEngine struct {
	Calls          []MockCall
	FindResults    map[string]geom.Point
	DefaultFind    geom.Point
	FindError      error
	ClickError     error
	DoubleClickErr error
	RightClickErr  error
	DragError      error
	TypeTextError  error
	PressKeyError  error
	PressComboErr  error
	ScrollDownErr  error
	ScrollUpErr    error
	RefreshErr     error
	BackErr        error
	ForwardErr     error
	SwitchTabErr   error
	BrowserErr     error
	WaitErr        error
	WaitGoneErr    error
	Screenshots    []string
	OpenedURLs     []string
}

// Called 返回指定名称的方法是否被调用过至少一次。
func (m *MockEngine) Called(methodName string) bool {
	for _, c := range m.Calls {
		if c.Name == methodName {
			return true
		}
	}
	return false
}

// CallCount 返回指定名称的方法被调用的次数。
func (m *MockEngine) CallCount(methodName string) int {
	count := 0
	for _, c := range m.Calls {
		if c.Name == methodName {
			count++
		}
	}
	return count
}

// OnFindElement 配置 FindElement 对特定模板路径返回 pt。
func (m *MockEngine) OnFindElement(path string, pt geom.Point) {
	if m.FindResults == nil {
		m.FindResults = make(map[string]geom.Point)
	}
	m.FindResults[path] = pt
}

// OnAllFindElement 配置 FindElement 对不在 FindResults 中的任意模板返回 pt。
func (m *MockEngine) OnAllFindElement(pt geom.Point) {
	m.DefaultFind = pt
}

// SetFindError 强制 FindElement 忽略 FindResults/DefaultFind 并直接返回 err。
func (m *MockEngine) SetFindError(err error) {
	m.FindError = err
}

// SetClickError 强制 Click 返回 err。
func (m *MockEngine) SetClickError(err error) {
	m.ClickError = err
}

// FindCall 返回指定方法名的第一次调用记录，未找到返回 nil。
func (m *MockEngine) FindCall(methodName string) *MockCall {
	for i := range m.Calls {
		if m.Calls[i].Name == methodName {
			return &m.Calls[i]
		}
	}
	return nil
}

// LastCall 返回指定方法名的最后一次调用记录，未找到返回 nil。
func (m *MockEngine) LastCall(methodName string) *MockCall {
	for i := len(m.Calls) - 1; i >= 0; i-- {
		if m.Calls[i].Name == methodName {
			return &m.Calls[i]
		}
	}
	return nil
}

func (m *MockEngine) record(name string, args ...interface{}) {
	m.Calls = append(m.Calls, MockCall{Name: name, Args: args})
}

// Click 实现 Engine 接口。
func (m *MockEngine) Click(x, y int) error {
	m.record("Click", x, y)
	return m.ClickError
}

// DoubleClick 实现 Engine 接口。
func (m *MockEngine) DoubleClick(x, y int) error {
	m.record("DoubleClick", x, y)
	return m.DoubleClickErr
}

// RightClick 实现 Engine 接口。
func (m *MockEngine) RightClick(x, y int) error {
	m.record("RightClick", x, y)
	return m.RightClickErr
}

// Drag 实现 Engine 接口。
func (m *MockEngine) Drag(fromX, fromY, toX, toY int) error {
	m.record("Drag", fromX, fromY, toX, toY)
	return m.DragError
}

// TypeText 实现 Engine 接口。
func (m *MockEngine) TypeText(text string) error {
	m.record("TypeText", text)
	return m.TypeTextError
}

// PressKey 实现 Engine 接口。
func (m *MockEngine) PressKey(key string) error {
	m.record("PressKey", key)
	return m.PressKeyError
}

// PressCombo 实现 Engine 接口。
func (m *MockEngine) PressCombo(keys ...string) error {
	args := make([]interface{}, len(keys))
	for i, k := range keys {
		args[i] = k
	}
	m.record("PressCombo", args...)
	return m.PressComboErr
}

// FindElement 实现 Engine 接口。
func (m *MockEngine) FindElement(templatePath string) (geom.Point, error) {
	m.record("FindElement", templatePath)
	if m.FindError != nil {
		return geom.Point{}, m.FindError
	}
	if m.FindResults != nil {
		if pt, ok := m.FindResults[templatePath]; ok {
			return pt, nil
		}
	}
	return m.DefaultFind, nil
}

// WaitForElement 实现 Engine 接口。
func (m *MockEngine) WaitForElement(templatePath string, timeoutSec int) error {
	m.record("WaitForElement", templatePath, timeoutSec)
	return m.WaitErr
}

// WaitForElementGone 实现 Engine 接口。
func (m *MockEngine) WaitForElementGone(templatePath string, timeoutSec int) error {
	m.record("WaitForElementGone", templatePath, timeoutSec)
	return m.WaitGoneErr
}

// ScrollDown 实现 Engine 接口。
func (m *MockEngine) ScrollDown(amount int) error {
	m.record("ScrollDown", amount)
	return m.ScrollDownErr
}

// ScrollUp 实现 Engine 接口。
func (m *MockEngine) ScrollUp(amount int) error {
	m.record("ScrollUp", amount)
	return m.ScrollUpErr
}

// RefreshPage 实现 Engine 接口。
func (m *MockEngine) RefreshPage() error {
	m.record("RefreshPage")
	return m.RefreshErr
}

// Back 实现 Engine 接口。
func (m *MockEngine) Back() error {
	m.record("Back")
	return m.BackErr
}

// Forward 实现 Engine 接口。
func (m *MockEngine) Forward() error {
	m.record("Forward")
	return m.ForwardErr
}

// SwitchTab 实现 Engine 接口。
func (m *MockEngine) SwitchTab(index int) error {
	m.record("SwitchTab", index)
	return m.SwitchTabErr
}

// OpenURL 实现 Engine 接口。
func (m *MockEngine) OpenURL(url string) {
	m.record("OpenURL", url)
	m.OpenedURLs = append(m.OpenedURLs, url)
}

// FindBrowserWindow 实现 Engine 接口。
func (m *MockEngine) FindBrowserWindow() error {
	m.record("FindBrowserWindow")
	return m.BrowserErr
}

// Wait 实现 Engine 接口。
func (m *MockEngine) Wait(ms int) {
	m.record("Wait", ms)
}

// CaptureScreen 实现 Engine 接口。
func (m *MockEngine) CaptureScreen(filename string) string {
	m.record("CaptureScreen", filename)
	m.Screenshots = append(m.Screenshots, filename)
	return filename
}

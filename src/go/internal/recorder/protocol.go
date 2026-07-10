package recorder

// RecorderEvent 录制器发给 GUI 的事件（JSON-Line，一行一个对象）
type RecorderEvent struct {
	Type                string `json:"type"`
	StepIndex           int    `json:"step_index,omitempty"`
	ActionIndex         int    `json:"action_index,omitempty"`
	Template            string `json:"template,omitempty"`
	CoordOptions        any    `json:"coord_options,omitempty"`
	IsVariableSupported bool   `json:"is_variable_supported,omitempty"`
	TemplateName        string `json:"template_name,omitempty"`
	TemplatePath        string `json:"template_path,omitempty"`
	Width               int    `json:"width,omitempty"`
	Height              int    `json:"height,omitempty"`
	OutputPath          string `json:"output_path,omitempty"`
	StepCount           int    `json:"step_count,omitempty"`
	Message             string `json:"message,omitempty"`
}

// RecorderCommand GUI 发给录制器的命令（JSON-Line，一行一个对象）
type RecorderCommand struct {
	Type         string       `json:"type"`
	Name         string       `json:"name,omitempty"`
	Description  string       `json:"description,omitempty"`
	ActionType   string       `json:"action_type,omitempty"`
	TemplatePath string       `json:"template_path,omitempty"`
	X            int          `json:"x,omitempty"`
	Y            int          `json:"y,omitempty"`
	Text         string       `json:"text,omitempty"`
	Variable     *VariableDef `json:"variable,omitempty"`
	Key          string       `json:"key,omitempty"`
	Keys         string       `json:"keys,omitempty"`
	URL          string       `json:"url,omitempty"`
	Amount       int          `json:"amount,omitempty"`
	Seconds      float64      `json:"seconds,omitempty"`
}

// VariableDef 运行时变量定义
type VariableDef struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
	Mask     bool   `json:"mask"`
}

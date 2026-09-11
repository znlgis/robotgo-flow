package recorder

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"robotgo-flow/internal/capture"
	"robotgo-flow/internal/config"
	"robotgo-flow/internal/encoding"
)

// Recorder 逐步引导录制器
type Recorder struct {
	templateDir string
	wf          *config.Workflow
	in          *bufio.Reader
	outputPath  string
}

// New 创建录制器。in 用于读取用户输入（通常为 os.Stdin，测试时可替换为 strings.Reader）。
func New(outputPath, templateDir string, in io.Reader) *Recorder {
	return &Recorder{
		templateDir: templateDir,
		wf: &config.Workflow{
			Settings: config.Settings{ElementTimeout: 10},
		},
		in:         bufio.NewReader(in),
		outputPath: outputPath,
	}
}

// Run 启动录制流程
func (r *Recorder) Run() error {
	fmt.Println("═══════════════════════════════════")
	fmt.Println("  robotgo-flow 录制模式")
	fmt.Println("═══════════════════════════════════")
	fmt.Println()

	// 1. 询问元信息
	r.askMeta()

	// 2. 确保模板目录存在
	os.MkdirAll(r.templateDir, 0755)

	// 3. 录制步骤
	fmt.Println("━━━ 开始录制步骤 ━━━")
	fmt.Println("(输入空步骤名结束录制)")
	fmt.Println()

	for {
		name, err := r.readLine()
		if err != nil || name == "" {
			break
		}

		step := config.Step{Name: name}
		for {
			action := r.askAction()
			if action == nil {
				break
			}
			step.Actions = append(step.Actions, *action)
		}
		r.wf.Steps = append(r.wf.Steps, step)
		fmt.Printf("步骤 %q 已记录 (%d 个操作)\n\n", name, len(step.Actions))
	}

	// 4. 保存
	return r.save()
}

func (r *Recorder) askMeta() {
	fmt.Print("工作流名称: ")
	name, _ := r.readLine()
	r.wf.Name = strings.TrimSpace(name)

	fmt.Print("工作流描述 (可选): ")
	desc, _ := r.readLine()
	r.wf.Description = strings.TrimSpace(desc)
	fmt.Println()
}

// makeTemplateRel 将模板名称转换为相对于 YAML 文件所在目录的路径，保证可移植性
// 例如：以 -tpl ./templates -out ./workflow.yaml 录制 "button" → "templates/button.png"
func (r *Recorder) makeTemplateRel(name string) string {
	abs := filepath.Join(r.templateDir, name+".png")
	if rel, err := filepath.Rel(filepath.Dir(r.outputPath), abs); err == nil {
		return rel
	}
	return abs
}

func (r *Recorder) askAction() *config.Action {
	for {
		fmt.Println("  操作类型:")
		fmt.Println("    1.点击  2.双击  3.右键  4.拖拽  5.输入文本")
		fmt.Println("    6.按键  7.组合键 8.等待出现 9.等待消失 10.滚动")
		fmt.Println("    11.打开网址 12.刷新 13.后退 14.前进 15.延时  0.完成")
		fmt.Print("  选择 [0-15]: ")

		choice, err := r.readLine()
		if err == io.EOF {
			return nil // stdin closed, exit gracefully
		}
		choice = strings.TrimSpace(choice)

		switch choice {
		case "0":
			return nil
		case "1":
			if act := r.recordClick(); act != nil {
				return act
			}
		case "2":
			if act := r.recordDoubleClick(); act != nil {
				return act
			}
		case "3":
			if act := r.recordRightClick(); act != nil {
				return act
			}
		case "4":
			if act := r.recordDrag(); act != nil {
				return act
			}
		case "5":
			if act := r.recordType(); act != nil {
				return act
			}
		case "6":
			return r.recordPressKey()
		case "7":
			return r.recordPressCombo()
		case "8":
			if act := r.recordWait(); act != nil {
				return act
			}
		case "9":
			if act := r.recordWaitGone(); act != nil {
				return act
			}
		case "10":
			if act := r.recordScroll(); act != nil {
				return act
			}
		case "11":
			return r.recordOpenURL()
		case "12":
			return &config.Action{Refresh: true}
		case "13":
			return &config.Action{Back: true}
		case "14":
			return &config.Action{Forward: true}
		case "15":
			if act := r.recordSleep(); act != nil {
				return act
			}
		default:
			fmt.Println("  无效选择，请重试")
		}
	}
}

// askTemplateOrCoord 询问用户使用模板还是坐标
func (r *Recorder) askTemplateOrCoord() any {
	fmt.Printf("  定位方式: 1.模板截图  2.屏幕坐标\n")
	fmt.Print("  选择 [1-2]: ")
	choice, _ := r.readLine()
	choice = strings.TrimSpace(choice)

	switch choice {
	case "2":
		fmt.Print("  输入坐标 (格式: x,y): ")
		coordStr, _ := r.readLine()
		var x, y int
		if n, _ := fmt.Sscanf(strings.TrimSpace(coordStr), "%d,%d", &x, &y); n < 2 {
			fmt.Println("  坐标格式无效，已取消")
			return nil
		}
		return map[string]interface{}{"x": x, "y": y}
	default:
		fmt.Print("  模板名称: ")
		name, _ := r.readLine()
		name = strings.TrimSpace(name)
		templatePath := r.makeTemplateRel(name)

		// 检查模板是否存在（用绝对路径）
		absPath := filepath.Join(r.templateDir, name+".png")
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			for {
				fmt.Printf("  模板 %s 不存在，请框选截图\n", name)
				if err := capture.CaptureInteractive(name, r.templateDir); err != nil {
					fmt.Printf("  截图失败: %v\n", err)
					fmt.Print("  重试? (y/n): ")
					retry, _ := r.readLine()
					if strings.TrimSpace(strings.ToLower(retry)) != "y" {
						return nil
					}
					continue
				}
				break
			}
		}
		return templatePath
	}
}

func (r *Recorder) recordClick() *config.Action {
	target := r.askTemplateOrCoord()
	if target == nil {
		return nil
	}
	return &config.Action{Click: target}
}

func (r *Recorder) recordDoubleClick() *config.Action {
	target := r.askTemplateOrCoord()
	if target == nil {
		return nil
	}
	return &config.Action{DoubleClick: target}
}

func (r *Recorder) recordRightClick() *config.Action {
	target := r.askTemplateOrCoord()
	if target == nil {
		return nil
	}
	return &config.Action{RightClick: target}
}

func (r *Recorder) recordDrag() *config.Action {
	fmt.Print("  起点模板名称: ")
	fromName, _ := r.readLine()
	fromName = strings.TrimSpace(fromName)
	fromPath := r.makeTemplateRel(fromName)
	if err := r.ensureTemplate(fromName); err != nil {
		fmt.Printf("  截图失败: %v\n", err)
	}

	fmt.Print("  终点模板名称: ")
	toName, _ := r.readLine()
	toName = strings.TrimSpace(toName)
	toPath := r.makeTemplateRel(toName)
	if err := r.ensureTemplate(toName); err != nil {
		fmt.Printf("  截图失败: %v\n", err)
	}

	return &config.Action{Drag: &config.DragSpec{From: fromPath, To: toPath}}
}

func (r *Recorder) recordType() *config.Action {
	fmt.Print("  目标输入框模板名称: ")
	name, _ := r.readLine()
	name = strings.TrimSpace(name)
	templatePath := r.makeTemplateRel(name)
	if err := r.ensureTemplate(name); err != nil {
		fmt.Printf("  截图失败: %v\n", err)
	}

	fmt.Print("  输入文本: ")
	text, _ := r.readLine()
	text = strings.TrimSpace(text)

	// 询问该文本是固定值还是运行时变量
	fmt.Print("  该文本是固定值(1)还是运行时变量(2)? [默认1]: ")
	choice, _ := r.readLine()
	choice = strings.TrimSpace(choice)

	if choice == "2" {
		// 采集变量元数据
		fmt.Print("  变量名称: ")
		varName, _ := r.readLine()
		varName = strings.TrimSpace(varName)

		fmt.Print("  显示标签: ")
		label, _ := r.readLine()
		label = strings.TrimSpace(label)

		fmt.Print("  是否必填? (y/n) [默认y]: ")
		reqStr, _ := r.readLine()
		required := strings.TrimSpace(strings.ToLower(reqStr)) != "n"

		fmt.Print("  是否密码遮罩? (y/n) [默认n]: ")
		maskStr, _ := r.readLine()
		mask := strings.TrimSpace(strings.ToLower(maskStr)) == "y"

		// 将变量加入工作流输入列表
		r.wf.Inputs = append(r.wf.Inputs, config.InputSpec{
			Name:     varName,
			Label:    label,
			Required: required,
			Mask:     mask,
		})

		// 在动作中使用 $input.<name> 占位符
		text = "$input." + varName
	}

	return &config.Action{Type: &config.TypeSpec{Into: templatePath, Text: text}}
}

func (r *Recorder) recordPressKey() *config.Action {
	fmt.Print("  按键 (enter/tab/escape/backspace/...): ")
	key, _ := r.readLine()
	return &config.Action{Press: strings.TrimSpace(key)}
}

func (r *Recorder) recordPressCombo() *config.Action {
	fmt.Print("  组合键 (用逗号分隔, 如: ctrl,c): ")
	comboStr, _ := r.readLine()
	parts := strings.Split(strings.TrimSpace(comboStr), ",")
	keys := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			keys = append(keys, p)
		}
	}
	return &config.Action{Combo: keys}
}

func (r *Recorder) recordWait() *config.Action {
	fmt.Print("  模板名称: ")
	name, _ := r.readLine()
	name = strings.TrimSpace(name)
	templatePath := r.makeTemplateRel(name)
	if err := r.ensureTemplate(name); err != nil {
		fmt.Printf("  截图失败: %v\n", err)
	}
	return &config.Action{Wait: templatePath}
}

func (r *Recorder) recordWaitGone() *config.Action {
	fmt.Print("  模板名称: ")
	name, _ := r.readLine()
	name = strings.TrimSpace(name)
	templatePath := r.makeTemplateRel(name)
	if err := r.ensureTemplate(name); err != nil {
		fmt.Printf("  截图失败: %v\n", err)
	}
	return &config.Action{WaitGone: templatePath}
}

func (r *Recorder) recordScroll() *config.Action {
	fmt.Print("  滚动量 (正数=下, 负数=上): ")
	input, _ := r.readLine()
	var amount int
	if n, _ := fmt.Sscanf(strings.TrimSpace(input), "%d", &amount); n < 1 || amount == 0 {
		fmt.Println("  无效输入，请输入非零整数")
		return nil
	}
	return &config.Action{Scroll: amount}
}

func (r *Recorder) recordOpenURL() *config.Action {
	fmt.Print("  网址: ")
	url, _ := r.readLine()
	return &config.Action{OpenURL: strings.TrimSpace(url)}
}

func (r *Recorder) recordSleep() *config.Action {
	fmt.Print("  等待秒数: ")
	input, _ := r.readLine()
	var secs float64
	if n, _ := fmt.Sscanf(strings.TrimSpace(input), "%f", &secs); n < 1 || secs <= 0 {
		fmt.Println("  无效输入，请输入正数")
		return nil
	}
	return &config.Action{Sleep: secs}
}

// readLine 从 stdin 读取一行并自动转码 GBK→UTF-8，自动去除尾部的 \r 和 \n
func (r *Recorder) readLine() (string, error) {
	line, err := r.in.ReadString('\n')
	if err != nil {
		return strings.TrimRight(line, "\r\n"), err
	}
	// ReadString 的返回值包含定界符 '\n'，需先剥离再交给编码转换，
	// 否则空输入（直接回车）会得到 "\r\n" 而非 ""，导致退出判断失效。
	line = strings.TrimRight(line, "\r\n")
	return encoding.ToUTF8String(line), nil
}

// ensureTemplate 检查模板是否已存在，不存在则调用交互式截图
func (r *Recorder) ensureTemplate(name string) error {
	absPath := filepath.Join(r.templateDir, name+".png")
	if _, err := os.Stat(absPath); !os.IsNotExist(err) {
		return nil
	}
	return capture.CaptureInteractive(name, r.templateDir)
}

// save 保存工作流配置到 YAML 文件
func (r *Recorder) save() error {
	if err := saveWorkflow(r.wf, r.outputPath); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("━━━ 录制完成 ━━━")
	fmt.Printf("工作流已保存: %s\n", r.outputPath)
	fmt.Printf("共 %d 个步骤\n", len(r.wf.Steps))
	return nil
}

// saveWorkflow 将工作流序列化并写入文件（包内共享）
func saveWorkflow(wf *config.Workflow, outputPath string) error {
	data, err := yaml.Marshal(wf)
	if err != nil {
		return fmt.Errorf("序列化 YAML 失败: %w", err)
	}
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	return nil
}

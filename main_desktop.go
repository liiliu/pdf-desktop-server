package main

import (
	"fmt"
	"image/color"
	"io/ioutil"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/BurntSushi/toml"
)

// Logger 日志管理器
type Logger struct {
	logWidget *widget.Entry
}

func NewLogger(logWidget *widget.Entry) *Logger {
	return &Logger{logWidget: logWidget}
}

func (l *Logger) Log(msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMsg := fmt.Sprintf("[%s] %s\n", timestamp, msg)
	l.logWidget.SetText(l.logWidget.Text + logMsg)
	// 自动滚动到底部
	l.logWidget.CursorRow = len(strings.Split(l.logWidget.Text, "\n")) - 1
}

func (l *Logger) Clear() {
	l.logWidget.SetText("")
}

// ChineseTheme 支持中文的自定义主题
type ChineseTheme struct {
	fyne.Theme
	fontData fyne.Resource
}

func (t *ChineseTheme) Font(style fyne.TextStyle) fyne.Resource {
	if t.fontData != nil {
		return t.fontData
	}
	return theme.DefaultTheme().Font(style)
}

func (t *ChineseTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// 自定义按钮颜色 - 更深的蓝色
	if name == theme.ColorNamePrimary {
		return color.NRGBA{R: 33, G: 150, B: 243, A: 255} // 深蓝色
	}
	if name == theme.ColorNameButton {
		return color.NRGBA{R: 25, G: 118, B: 210, A: 255} // 更深的蓝色
	}
	if name == theme.ColorNameHover {
		return color.NRGBA{R: 21, G: 101, B: 192, A: 255} // 悬停时更深
	}

	// 加深文字颜色 - 让文字更清晰易读
	if name == theme.ColorNameForeground {
		return color.NRGBA{R: 33, G: 33, B: 33, A: 255} // 深灰色文字（几乎黑色）
	}
	if name == theme.ColorNamePlaceHolder {
		return color.NRGBA{R: 117, G: 117, B: 117, A: 255} // 占位符文字（深灰）
	}
	if name == theme.ColorNameDisabled {
		return color.NRGBA{R: 66, G: 66, B: 66, A: 255} // 禁用状态文字（深灰）
	}
	if name == theme.ColorNameInputBackground {
		return color.NRGBA{R: 250, G: 250, B: 250, A: 255} // 输入框背景（浅灰白）
	}

	// 其他颜色使用默认主题
	return t.Theme.Color(name, variant)
}

// setupChineseFont 设置中文字体
func setupChineseFont(myApp fyne.App) {
	// 尝试加载项目中的中文字体
	fontFiles := []string{
		"resources/fonts/PingFang Regular_0.ttf",
		"resources/fonts/PingFang Semibold.ttf",
	}

	var fontData []byte
	var err error

	for _, fontPath := range fontFiles {
		fontData, err = ioutil.ReadFile(fontPath)
		if err == nil {
			// 成功加载字体
			fontResource := fyne.NewStaticResource("chinese-font", fontData)
			customTheme := &ChineseTheme{
				Theme:    theme.LightTheme(),
				fontData: fontResource,
			}
			myApp.Settings().SetTheme(customTheme)
			return
		}
	}

	// 如果没有找到字体文件，使用默认主题
	// Windows 系统会自动使用系统中文字体
	myApp.Settings().SetTheme(theme.LightTheme())
}

func main() {
	// 加载配置
	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		panic(err)
	}

	// 创建 Fyne 应用
	myApp := app.New()

	// 设置中文字体
	setupChineseFont(myApp)

	myWindow := myApp.NewWindow("打印工具 - 桌面版")
	myWindow.Resize(fyne.NewSize(1200, 700))

	// 设置窗口图标
	icon, err := fyne.LoadResourceFromPath("resources/images/favicon.ico")
	if err == nil {
		myWindow.SetIcon(icon)
	}

	// 创建日志显示区域
	logWidget := widget.NewMultiLineEntry()
	logWidget.SetPlaceHolder("日志信息将在这里显示...")
	logWidget.Disable() // 只读
	logger := NewLogger(logWidget)

	// 创建清空日志按钮
	clearLogBtn := widget.NewButton("🗑️ 清空日志", func() {
		logger.Clear()
	})
	clearLogBtn.Importance = widget.LowImportance

	// Tab 1: 单个/成对设备号打印
	tab1Content := createPrintTab(logger)

	// Tab 2: 批量设备号打印
	tab2Content := createMultiPrintTab(logger)

	// Tab 3: 产品标签打印
	tab3Content := createTagPrintTab(logger)

	// 创建 Tab 容器
	tabs := container.NewAppTabs(
		container.NewTabItem("设备号打印", tab1Content),
		container.NewTabItem("批量打印", tab2Content),
		container.NewTabItem("标签打印", tab3Content),
	)

	// 日志区域 - 放在右侧，支持滚动
	logTitle := widget.NewLabelWithStyle("📋 日志信息", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	logHeader := container.NewBorder(nil, nil, nil, clearLogBtn, logTitle)
	
	// 将日志widget放入滚动容器，支持自动扩展和滚动
	logScroll := container.NewScroll(logWidget)
	logContainer := container.NewBorder(logHeader, nil, nil, nil, logScroll)

	// 主布局：左侧是功能区，右侧是日志
	// 使用 HSplit 可以让用户调整分割比例
	mainContent := container.NewHSplit(
		tabs,
		logContainer,
	)
	// 设置初始分割比例：左侧60%，右侧40%
	mainContent.SetOffset(0.6)

	myWindow.SetContent(mainContent)
	logger.Log("打印工具已启动")
	myWindow.ShowAndRun()
}

// createPrintTab 创建单个/成对设备号打印界面
func createPrintTab(logger *Logger) fyne.CanvasObject {
	// 输入框 - 初始显示6行
	deviceNosEntry := widget.NewMultiLineEntry()
	deviceNosEntry.SetPlaceHolder("输入设备号，多个设备号用逗号分隔\n例如: 12345,67890,11111,22222")
	deviceNosEntry.SetMinRowsVisible(6)         // 初始显示6行
	deviceNosEntry.Wrapping = fyne.TextWrapWord // 启用换行

	// 监听内容变化，动态调整显示行数
	deviceNosEntry.OnChanged = func(content string) {
		lines := strings.Count(content, "\n") + 1
		if lines > 6 {
			deviceNosEntry.SetMinRowsVisible(lines)
		} else {
			deviceNosEntry.SetMinRowsVisible(6)
		}
		deviceNosEntry.Refresh()
	}

	// 打印按钮
	printBtn := widget.NewButton("开始打印", func() {
		deviceNos := strings.TrimSpace(deviceNosEntry.Text)
		if deviceNos == "" {
			logger.Log("❌ 错误: 请输入设备号")
			return
		}

		// 拆分设备号
		deviceNoArr := strings.Split(deviceNos, ",")

		// 检查是否有重复
		duplicate := findDuplicate(deviceNoArr)
		if len(duplicate) > 0 {
			logger.Log(fmt.Sprintf("❌ 错误: 设备号重复: %s", strings.Join(duplicate, ",")))
			return
		}

		logger.Log(fmt.Sprintf("✓ 开始打印 %d 个设备号", len(deviceNoArr)))

		// 异步打印，避免阻塞 UI
		go func() {
			length := len(deviceNoArr)
			if length%2 == 0 {
				for i := 0; i < length; i += 2 {
					logger.Log(fmt.Sprintf("正在打印: %s, %s", deviceNoArr[i], deviceNoArr[i+1]))
					GenerateDoublePdf(strings.TrimSpace(deviceNoArr[i]), strings.TrimSpace(deviceNoArr[i+1]), config.AdobePath, config.PrintInterval)
				}
			} else {
				for i := 0; i < length-1; i += 2 {
					logger.Log(fmt.Sprintf("正在打印: %s, %s", deviceNoArr[i], deviceNoArr[i+1]))
					GenerateDoublePdf(strings.TrimSpace(deviceNoArr[i]), strings.TrimSpace(deviceNoArr[i+1]), config.AdobePath, config.PrintInterval)
				}
				logger.Log(fmt.Sprintf("正在打印: %s", deviceNoArr[length-1]))
				GeneratePdf(strings.TrimSpace(deviceNoArr[length-1]), config.AdobePath, config.PrintInterval)
			}
			logger.Log("✓ 所有打印任务完成")
		}()
	})

	// 设置按钮样式
	printBtn.Importance = widget.HighImportance

	// 布局 - 简单的垂直布局，输入框随内容自动扩展
	title := widget.NewLabelWithStyle("📝 输入设备号", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	form := container.NewVBox(
		title,
		widget.NewSeparator(),
		deviceNosEntry, // 输入框会随内容自动扩展
		container.NewPadded(printBtn),
	)

	// 整个表单可以滚动，内容多时向下扩展
	return container.NewScroll(container.NewPadded(form))
}

// createMultiPrintTab 创建批量打印界面
func createMultiPrintTab(logger *Logger) fyne.CanvasObject {
	// 输入框 - 初始显示6行
	deviceNosEntry := widget.NewMultiLineEntry()
	deviceNosEntry.SetPlaceHolder("输入设备号，多个设备号用逗号分隔\n例如: 12345,67890,11111")
	deviceNosEntry.SetMinRowsVisible(6)         // 初始显示6行
	deviceNosEntry.Wrapping = fyne.TextWrapWord // 启用换行

	// 监听内容变化，动态调整显示行数
	deviceNosEntry.OnChanged = func(content string) {
		lines := strings.Count(content, "\n") + 1
		if lines > 6 {
			deviceNosEntry.SetMinRowsVisible(lines)
		} else {
			deviceNosEntry.SetMinRowsVisible(6)
		}
		deviceNosEntry.Refresh()
	}

	// 打印按钮 - 设置为高优先级按钮
	printBtn := widget.NewButton("🖨️ 开始批量打印", func() {
		deviceNos := strings.TrimSpace(deviceNosEntry.Text)
		if deviceNos == "" {
			logger.Log("❌ 错误: 请输入设备号")
			return
		}

		logger.Log("✓ 开始生成批量二维码")

		// 异步打印
		go func() {
			GenerateMultiPdf(deviceNos, config.AdobePath, config.PrintInterval)
			logger.Log("✓ 批量打印完成")
		}()
	})

	// 设置按钮样式
	printBtn.Importance = widget.HighImportance

	// 布局 - 简单的垂直布局，输入框随内容自动扩展
	title := widget.NewLabelWithStyle("📦 输入设备号（批量）", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	form := container.NewVBox(
		title,
		widget.NewSeparator(),
		deviceNosEntry, // 输入框会随内容自动扩展
		container.NewPadded(printBtn),
	)

	// 整个表单可以滚动，内容多时向下扩展
	return container.NewScroll(container.NewPadded(form))
}

// createTagPrintTab 创建产品标签打印界面
func createTagPrintTab(logger *Logger) fyne.CanvasObject {
	// 创建输入框
	productNameEntry := widget.NewEntry()
	productNameEntry.SetPlaceHolder("产品名称")

	productColorEntry := widget.NewEntry()
	productColorEntry.SetPlaceHolder("产品颜色")

	productDateEntry := widget.NewEntry()
	productDateEntry.SetPlaceHolder("生产日期 (例如: 2024-01-01)")
	productDateEntry.SetText(time.Now().Format("2006-01-02"))

	productNumEntry := widget.NewEntry()
	productNumEntry.SetPlaceHolder("产品数量")

	grossWeightEntry := widget.NewEntry()
	grossWeightEntry.SetPlaceHolder("毛重 (KG)")

	netWeightEntry := widget.NewEntry()
	netWeightEntry.SetPlaceHolder("净重 (KG)")

	barCode69TypeEntry := widget.NewEntry()
	barCode69TypeEntry.SetPlaceHolder("条码类型 (例如: 401)")
	barCode69TypeEntry.SetText("401")

	boxNumEntry := widget.NewEntry()
	boxNumEntry.SetPlaceHolder("箱号")

	deviceNosEntry := widget.NewMultiLineEntry()
	deviceNosEntry.SetPlaceHolder("设备号\n每行一个设备号，或用逗号、竖线等分隔\n例如:\n12345\n67890\n或: 12345,67890")
	deviceNosEntry.SetMinRowsVisible(4)         // 初始显示4行，保证按钮可见
	deviceNosEntry.Wrapping = fyne.TextWrapWord // 启用换行

	// 监听内容变化，动态调整显示行数（无上限，整个页面可滚动）
	deviceNosEntry.OnChanged = func(content string) {
		lines := strings.Count(content, "\n") + 1
		if lines > 4 {
			deviceNosEntry.SetMinRowsVisible(lines) // 随内容扩展，无上限
		} else {
			deviceNosEntry.SetMinRowsVisible(4) // 至少显示4行
		}
		deviceNosEntry.Refresh()
	}

	// 打印按钮 - 设置为高优先级按钮
	printBtn := widget.NewButton("🏷️ 打印产品标签", func() {
		// 验证输入
		excelData := &ExcelData{
			ProductName:   strings.TrimSpace(productNameEntry.Text),
			ProductColor:  strings.TrimSpace(productColorEntry.Text),
			ProductDate:   strings.TrimSpace(productDateEntry.Text),
			ProductNum:    strings.TrimSpace(productNumEntry.Text),
			GrossWeight:   strings.TrimSpace(grossWeightEntry.Text),
			NetWeight:     strings.TrimSpace(netWeightEntry.Text),
			BarCode69Type: strings.TrimSpace(barCode69TypeEntry.Text),
			BoxNum:        strings.TrimSpace(boxNumEntry.Text),
			DeviceNos:     strings.TrimSpace(deviceNosEntry.Text),
		}

		// 验证必填项
		if excelData.ProductName == "" {
			logger.Log("❌ 错误: 请输入产品名称")
			return
		}
		if excelData.ProductColor == "" {
			logger.Log("❌ 错误: 请输入产品颜色")
			return
		}
		if excelData.ProductDate == "" {
			logger.Log("❌ 错误: 请输入生产日期")
			return
		}
		if excelData.ProductNum == "" {
			logger.Log("❌ 错误: 请输入产品数量")
			return
		}
		if excelData.GrossWeight == "" {
			logger.Log("❌ 错误: 请输入毛重")
			return
		}
		if excelData.NetWeight == "" {
			logger.Log("❌ 错误: 请输入净重")
			return
		}
		if excelData.BarCode69Type == "" {
			logger.Log("❌ 错误: 请输入条码类型")
			return
		}
		if excelData.BoxNum == "" {
			logger.Log("❌ 错误: 请输入箱号")
			return
		}
		if excelData.DeviceNos == "" {
			logger.Log("❌ 错误: 请输入设备号")
			return
		}

		// 处理条码类型
		excelData.BarCode69Type = fmt.Sprintf("%s-69.png", excelData.BarCode69Type)

		// 处理箱号拼接 - 根据设备号中是否有逗号来决定拼接方式
		if excelData.BoxNum != "" {
			if strings.Contains(excelData.DeviceNos, ",") {
				// 设备号中有逗号 → 箱号用逗号拼接
				excelData.DeviceNos = excelData.BoxNum + "," + excelData.DeviceNos
			} else {
				// 设备号中没有逗号 → 箱号用换行拼接
				excelData.DeviceNos = excelData.BoxNum + "\n" + excelData.DeviceNos
			}
		}
		// 不做任何格式转换，完全使用用户输入的格式

		logger.Log(fmt.Sprintf("✓ 开始打印标签: 箱号 %s", excelData.BoxNum))

		// 异步打印
		go func() {
			GenerateMultiTagPdf(excelData)
			logger.Log(fmt.Sprintf("✓ 标签打印完成: 箱号 %s", excelData.BoxNum))
		}()
	})

	// 设置按钮样式
	printBtn.Importance = widget.HighImportance

	// 布局 - 使用分组和更好的视觉层次
	productInfoTitle := widget.NewLabelWithStyle("📦 产品信息", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	weightInfoTitle := widget.NewLabelWithStyle("⚖️ 重量信息", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	barcodeInfoTitle := widget.NewLabelWithStyle("📊 条码和箱号", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	deviceInfoTitle := widget.NewLabelWithStyle("🔢 设备号", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	form := container.NewVBox(
		productInfoTitle,
		productNameEntry,
		productColorEntry,
		productDateEntry,
		productNumEntry,
		widget.NewSeparator(),

		weightInfoTitle,
		container.NewGridWithColumns(2, grossWeightEntry, netWeightEntry),
		widget.NewSeparator(),

		barcodeInfoTitle,
		container.NewGridWithColumns(2, barCode69TypeEntry, boxNumEntry),
		widget.NewSeparator(),

		deviceInfoTitle,
		deviceNosEntry, // 输入框会随内容自动扩展

		container.NewPadded(printBtn),
	)

	// 整个表单可以滚动
	return container.NewScroll(form)
}

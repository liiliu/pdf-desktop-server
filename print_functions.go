package main

import (
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/ean"
	"github.com/boombuler/barcode/qr"
	"github.com/flopp/go-findfont"
	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
	qrcode2 "github.com/skip2/go-qrcode"
	"github.com/tealeg/xlsx"
	"image/jpeg"
	"image/png"
	"math"
	"math/rand"
	"net/http"

	"image"
	"image/draw"
	_ "image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AdobePath     string
	PrintInterval int
	Name          string
	Baud          int
	ImageDir      string
	PdfDir        string
}

var config *Config

// mainWeb 启动 Web 服务器版本 (已废弃，使用桌面版)
func mainWeb() {
	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		panic(err)
	}

	// 注册helloHandler处理函数，对应"/hello"路径的GET请求
	http.HandleFunc("/print", printHandler)
	http.HandleFunc("/printMulti", printMultiHandler)
	http.HandleFunc("/printMultiTag", printMultiTagHandler)

	// 启动Web服务器，监听在13008端口
	fmt.Println("Starting server on port 13008...")
	if err := http.ListenAndServe("0.0.0.0:13008", nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}

// mainExcel Excel 批量处理版本 (已废弃，使用桌面版)
func mainExcel() {
	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		panic(err)
	}
	//接收输入的文件名参数
	if len(os.Args) < 2 {
		fmt.Println("用法: main.exe <filename>")
		return
	}
	fileName := os.Args[1]
	fmt.Println("文件名:", fileName)
	//解析excel 文件
	data, err := ParseExcel(fileName)
	if err != nil {
		return
	}
	for _, excelData := range data {
		GenerateMultiPdfByExcel(excelData)
	}
}

// Response 结构体保持不变
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// 找出字符串切片中的重复元素
func findDuplicate(strSlice []string) []string {
	encountered := map[string]bool{}
	var duplicate []string
	for v := range strSlice {
		if encountered[strSlice[v]] == true {
			duplicate = append(duplicate, strSlice[v])
		} else {
			encountered[strSlice[v]] = true
		}
	}
	return duplicate
}

func printHandler(w http.ResponseWriter, r *http.Request) {
	// 设置响应的内容类型为JSON
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 获取查询字符串
	queryParams := r.URL.Query()
	deviceNos := queryParams.Get("deviceNos") // 通过键名获取参数值，如果不存在则返回空字符串
	// 创建一个Response实例
	resp := &Response{
		Code:    0,
		Message: "",
	}

	if deviceNos != "" {
		resp.Message = fmt.Sprintf("Hello, %s! This is your personalized API response.", deviceNos)
	} else {
		resp.Code = -1
		resp.Message = "Hello, this is your API! Please provide a 'deviceNos' parameter for a personalized greeting."
		json.NewEncoder(w).Encode(resp)
		return
	}

	//打印数据
	//拆分字符串
	deviceNoArr := strings.Split(deviceNos, ",")

	//检查是否有重复的设备号
	duplicate := findDuplicate(deviceNoArr)
	if len(duplicate) > 0 {
		resp.Code = -1
		resp.Message = "设备号重复: " + strings.Join(duplicate, ",")
		json.NewEncoder(w).Encode(resp)
		return
	}

	length := len(deviceNoArr)
	if length%2 == 0 {
		for i := 0; i < length; i += 2 {
			//生成二维码
			GenerateDoublePdf(strings.TrimSpace(deviceNoArr[i]), strings.TrimSpace(deviceNoArr[i+1]), config.AdobePath, config.PrintInterval)
		}
	} else {
		for i := 0; i < length-1; i += 2 {
			//生成二维码
			GenerateDoublePdf(strings.TrimSpace(deviceNoArr[i]), strings.TrimSpace(deviceNoArr[i+1]), config.AdobePath, config.PrintInterval)
		}
		//生成二维码
		GeneratePdf(strings.TrimSpace(deviceNoArr[length-1]), config.AdobePath, config.PrintInterval)
	}

	// 将Response实例编码为JSON并写入响应体
	json.NewEncoder(w).Encode(resp)
}

func GenerateDoublePdf(deviceNo, deviceNo1, adobePath string, printInterval int) {
	imagePath := fmt.Sprintf("%s/%s.jpeg", config.ImageDir, deviceNo)
	// Create the barcode
	qrCode, _ := qr.Encode(deviceNo, qr.H, qr.Auto)

	// Scale the barcode to 200x200 pixels
	qrCode, _ = barcode.Scale(qrCode, 320, 320)

	// create the output file
	file, _ := os.Create(imagePath)
	defer file.Close()

	// encode the barcode as png
	jpeg.Encode(file, qrCode, nil)

	imagePath1 := fmt.Sprintf("%s/%s.jpeg", config.ImageDir, deviceNo1)
	// Create the barcode
	qrCode1, _ := qr.Encode(deviceNo1, qr.H, qr.Auto)

	// Scale the barcode to 200x200 pixels
	qrCode1, _ = barcode.Scale(qrCode1, 320, 320)

	// create the output file
	file1, _ := os.Create(imagePath1)
	defer file1.Close()

	// encode the barcode as png
	jpeg.Encode(file1, qrCode1, nil)

	// 初始化一个pdf
	// param1: P/L 横屏或者竖屏
	// param2: mm, cm ... 间距单位,一般是mm
	// param3: A4,A3,A5 纸张大小
	// param4: 特定的字体样式
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: "L",
		UnitStr:        "mm",
		SizeStr:        "A4",
		Size: gofpdf.SizeType{
			Wd: 400,
			Ht: 800,
		},
		FontDirStr: "",
	})
	// 添加新的空白页,在写入内容之前,一定要添加一个空白页
	pdf.AddPage()
	// 设置特定的字体,如果是中文字体的话,需要使用其它方法,下面介绍
	// param1: 字体样式名称,可以引入自己下载的
	// param2: 字体样式,B,I,U,S,分别是粗体,斜体,下划线,删除线,或者其它
	// param3: 字体大小
	pdf.SetFont("Arial", "B", 112)
	// 新建一个表格单元,也就是一块区域,可以理解为一行
	// param1: 单元格宽,这里40表示40mm,与new里面的单位有关
	// param2: 单元格高
	// param3: 显示的内容
	// 实际例子中使用的是pdf.CellFormat() 这个api,可以在一行画多个单元格
	//pdf.Cell(40, 10, "Hello, world")

	//将图片放入到 pdf 文档中
	//ImageOptions(src, x, y, width, height, flow, options, link, linkStr)
	pdf.ImageOptions(
		imagePath,
		20, 0,
		320, 320,
		false,
		gofpdf.ImageOptions{ImageType: "jpeg", ReadDpi: false, AllowNegativePosition: true},
		0,
		"",
	)
	pdf.Text(45, 370, deviceNo)
	pdf.ImageOptions(
		imagePath1,
		460, 0,
		320, 320,
		false,
		gofpdf.ImageOptions{ImageType: "jpeg", ReadDpi: false, AllowNegativePosition: true},
		0,
		"",
	)
	pdf.Text(485, 370, deviceNo1)
	pdfPath := fmt.Sprintf("%s/%s_%s.pdf", config.PdfDir, deviceNo, deviceNo1)
	herr := pdf.OutputFileAndClose(pdfPath)
	if herr != nil {
		fmt.Println(herr.Error())
	}

	fmt.Println("设备号[", deviceNo, deviceNo1, "]开始打印")
	pwd, _ := os.Getwd()
	// 打印二维码
	//cmd := exec.Command(adobePath, "/p", "/h", "/n", "/s", "/o", pdfPath)
	CmdBlockExec("cmd", "/k", "start", adobePath, "/p", "/h", filepath.Join(pwd, pdfPath))
	//CmdSyncExec("cmd", "/k", "start", adobePath, "/p", "/h", pdfPath)
	time.Sleep(time.Duration(printInterval) * time.Second)

	// 删除pdf
	//os.Remove(filepath.Join(pwd, pdfPath))
	// 删除图片
	//os.Remove(filepath.Join(pwd, imagePath))
	// 删除图片
	//os.Remove(filepath.Join(pwd, imagePath1))
	fmt.Println("设备号[", deviceNo, deviceNo1, "]打印完成")
}

func GeneratePdf(deviceNo, adobePath string, printInterval int) {
	imagePath := fmt.Sprintf("%s/%s.jpeg", config.ImageDir, deviceNo)
	// Create the barcode
	qrCode, _ := qr.Encode(deviceNo, qr.H, qr.Auto)

	// Scale the barcode to 200x200 pixels
	qrCode, _ = barcode.Scale(qrCode, 320, 320)

	// create the output file
	file, _ := os.Create(imagePath)
	defer file.Close()

	// encode the barcode as png
	jpeg.Encode(file, qrCode, nil)

	// 初始化一个pdf
	// param1: P/L 横屏或者竖屏
	// param2: mm, cm ... 间距单位,一般是mm
	// param3: A4,A3,A5 纸张大小
	// param4: 特定的字体样式
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: "L",
		UnitStr:        "mm",
		SizeStr:        "A4",
		Size: gofpdf.SizeType{
			Wd: 400,
			Ht: 800,
		},
		FontDirStr: "",
	})
	// 添加新的空白页,在写入内容之前,一定要添加一个空白页
	pdf.AddPage()
	// 设置特定的字体,如果是中文字体的话,需要使用其它方法,下面介绍
	// param1: 字体样式名称,可以引入自己下载的
	// param2: 字体样式,B,I,U,S,分别是粗体,斜体,下划线,删除线,或者其它
	// param3: 字体大小
	pdf.SetFont("Arial", "B", 112)
	// 新建一个表格单元,也就是一块区域,可以理解为一行
	// param1: 单元格宽,这里40表示40mm,与new里面的单位有关
	// param2: 单元格高
	// param3: 显示的内容
	// 实际例子中使用的是pdf.CellFormat() 这个api,可以在一行画多个单元格
	//pdf.Cell(40, 10, "Hello, world")

	//将图片放入到 pdf 文档中
	//ImageOptions(src, x, y, width, height, flow, options, link, linkStr)
	pdf.ImageOptions(
		imagePath,
		20, 0,
		320, 320,
		false,
		gofpdf.ImageOptions{ImageType: "jpeg", ReadDpi: false, AllowNegativePosition: true},
		0,
		"",
	)
	pdf.Text(45, 370, deviceNo)
	pdf.ImageOptions(
		imagePath,
		460, 0,
		320, 320,
		false,
		gofpdf.ImageOptions{ImageType: "jpeg", ReadDpi: false, AllowNegativePosition: true},
		0,
		"",
	)
	pdf.Text(485, 370, deviceNo)
	pdfPath := fmt.Sprintf("%s/%s.pdf", config.PdfDir, deviceNo)
	herr := pdf.OutputFileAndClose(pdfPath)
	if herr != nil {
		fmt.Println(herr.Error())
	}

	fmt.Println("设备号[", deviceNo, "]开始打印")
	pwd, _ := os.Getwd()
	// 打印二维码
	//cmd := exec.Command(adobePath, "/p", "/h", "/n", "/s", "/o", pdfPath)
	//go CmdBlockExec("cmd", "/k", "start", adobePath, "/p", "/h", pdfPath)
	CmdBlockExec("cmd", "/k", "start", adobePath, "/p", "/h", filepath.Join(pwd, pdfPath))
	time.Sleep(time.Duration(printInterval) * time.Second)

	// 删除pdf
	//os.Remove(filepath.Join(pwd, pdfPath))
	// 删除图片
	//os.Remove(filepath.Join(pwd, imagePath))
	fmt.Println("设备号[", deviceNo, "]打印完成")
}

// printHandler 是处理GET请求的函数
func printMultiHandler(w http.ResponseWriter, r *http.Request) {
	// 设置响应的内容类型为JSON
	w.Header().Set("Content-Type", "application/json")

	// 获取查询字符串
	queryParams := r.URL.Query()
	deviceNos := queryParams.Get("deviceNos") // 通过键名获取参数值，如果不存在则返回空字符串

	// 创建一个Response实例
	resp := &Response{
		Code:    0,
		Message: "",
	}

	if deviceNos != "" {
		resp.Message = fmt.Sprintf("Hello, %s! This is your personalized API response.", deviceNos)
	} else {
		resp.Code = -1
		resp.Message = "Hello, this is your API! Please provide a 'deviceNos' parameter for a personalized greeting."
	}

	//生成二维码
	go GenerateMultiPdf(strings.TrimSpace(deviceNos), config.AdobePath, config.PrintInterval)

	// 将Response实例编码为JSON并写入响应体
	json.NewEncoder(w).Encode(resp)
}

func GenerateMultiPdf(deviceNo, adobePath string, printInterval int) {
	// 将设备号的逗号替换为换行符
	deviceNo = strings.ReplaceAll(deviceNo, ",", "\n")
	// 创建二维码图片的文件名
	fileName := fmt.Sprintf("multiCode_%d", time.Now().UnixMilli())
	imagePath := fmt.Sprintf("%s/%s.jpeg", config.ImageDir, fileName)
	// Create the barcode
	qrCode, err := qr.Encode(deviceNo, qr.H, qr.Auto)
	if err != nil {
		fmt.Println("生成二维码失败:", err.Error())
	}

	// Scale the barcode to 200x200 pixels
	qrCode, _ = barcode.Scale(qrCode, 600, 600) // 800，800

	// create the output file
	file, _ := os.Create(imagePath)
	defer file.Close()

	// encode the barcode as png
	jpeg.Encode(file, qrCode, nil)

	// 初始化一个pdf
	// param1: P/L 横屏或者竖屏
	// param2: mm, cm ... 间距单位,一般是mm
	// param3: A4,A3,A5 纸张大小
	// param4: 特定的字体样式
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: "L",
		UnitStr:        "mm",
		SizeStr:        "A4",
		Size: gofpdf.SizeType{
			Wd: 840,
			Ht: 840,
		},
		FontDirStr: "",
	})
	// 添加新的空白页,在写入内容之前,一定要添加一个空白页
	pdf.AddPage()
	// 设置特定的字体,如果是中文字体的话,需要使用其它方法,下面介绍
	// param1: 字体样式名称,可以引入自己下载的
	// param2: 字体样式,B,I,U,S,分别是粗体,斜体,下划线,删除线,或者其它
	// param3: 字体大小
	pdf.SetFont("Arial", "B", 112)
	// 新建一个表格单元,也就是一块区域,可以理解为一行
	// param1: 单元格宽,这里40表示40mm,与new里面的单位有关
	// param2: 单元格高
	// param3: 显示的内容
	// 实际例子中使用的是pdf.CellFormat() 这个api,可以在一行画多个单元格
	//pdf.Cell(40, 10, "Hello, world")

	//将图片放入到 pdf 文档中
	//ImageOptions(src, x, y, width, height, flow, options, link, linkStr)
	pdf.ImageOptions(
		imagePath,
		120, 120, // 20，20
		600, 600, // 800，800
		false,
		gofpdf.ImageOptions{ImageType: "jpeg", ReadDpi: false, AllowNegativePosition: true},
		0,
		"",
	)
	//pdf.Text(45, 370, deviceNo)
	//pdf.ImageOptions(
	//	imagePath,
	//	460, 0,
	//	320, 320,
	//	false,
	//	gofpdf.ImageOptions{ImageType: "jpeg", ReadDpi: false, AllowNegativePosition: true},
	//	0,
	//	"",
	//)
	//pdf.Text(485, 370, deviceNo)
	pdfPath := fmt.Sprintf("%s/%s.pdf", config.PdfDir, fileName)
	herr := pdf.OutputFileAndClose(pdfPath)
	if herr != nil {
		fmt.Println(herr.Error())
	}

	fmt.Println("设备号[", fileName, "]开始打印")
	pwd, _ := os.Getwd()
	// 打印二维码
	//cmd := exec.Command(adobePath, "/p", "/h", "/n", "/s", "/o", pdfPath)
	//go CmdBlockExec("cmd", "/k", "start", adobePath, "/p", "/h", pdfPath)
	CmdBlockExec("cmd", "/k", "start", adobePath, "/p", "/h", filepath.Join(pwd, pdfPath))
	time.Sleep(time.Duration(printInterval) * time.Second)

	// 删除pdf
	//os.Remove(filepath.Join(pwd, pdfPath))
	// 删除图片
	//os.Remove(filepath.Join(pwd, imagePath))
	fmt.Println("设备号[", fileName, "]打印完成")
}

// printHandler 是处理GET请求的函数
func printMultiTagHandler(w http.ResponseWriter, r *http.Request) {
	// 设置响应的内容类型为JSON
	w.Header().Set("Content-Type", "application/json")

	excelData := new(ExcelData)
	// 获取查询字符串
	queryParams := r.URL.Query()
	excelData.ProductName = queryParams.Get("productName")
	excelData.ProductColor = queryParams.Get("productColor")
	excelData.ProductDate = queryParams.Get("productDate")
	excelData.ProductNum = queryParams.Get("productNum")
	excelData.GrossWeight = queryParams.Get("grossWeight")
	excelData.NetWeight = queryParams.Get("netWeight")
	excelData.BarCode69Type = queryParams.Get("barCode69Type")
	excelData.BoxNum = queryParams.Get("boxNum")
	excelData.DeviceNos = queryParams.Get("deviceNos") // 通过键名获取参数值，如果不存在则返回空字符串

	// 创建一个Response实例
	resp := &Response{
		Code:    0,
		Message: "",
	}
	if excelData.ProductName == "" {
		resp.Message = "请输入产品名称"
	} else if excelData.ProductColor == "" {
		resp.Message = "请输入产品颜色"
	} else if excelData.ProductDate == "" {
		resp.Message = "请输入产品生产日期"
	} else if excelData.ProductNum == "" {
		resp.Message = "请输入产品数量"
	} else if excelData.GrossWeight == "" {
		resp.Message = "请输入产品毛重"
	} else if excelData.NetWeight == "" {
		resp.Message = "请输入产品净重"
	} else if excelData.BarCode69Type == "" {
		resp.Message = "请输入条码类型"
	} else if excelData.BoxNum == "" {
		resp.Message = "请输入箱数"
	} else if excelData.DeviceNos == "" {
		resp.Message = "请输入设备号"
	}
	if resp.Message != "" {
		resp.Code = -1
		json.NewEncoder(w).Encode(resp)
	} else {
		excelData.BarCode69Type = fmt.Sprintf("%s-69.png", excelData.BarCode69Type)

		//设备号前面拼接箱号，连接符取设备号的连接符,或者\n,没有箱号时，则使用设备号
		if excelData.BoxNum != "" {
			if strings.Contains(excelData.DeviceNos, "|") {
				excelData.DeviceNos = fmt.Sprintf("%s|%s", excelData.BoxNum, excelData.DeviceNos)
			} else {
				excelData.DeviceNos = fmt.Sprintf("%s,%s", excelData.BoxNum, excelData.DeviceNos)
			}
		}
		excelData.DeviceNos = strings.ReplaceAll(excelData.DeviceNos, "|", "\n")
		//生成二维码
		go GenerateMultiTagPdf(excelData)

		// 将Response实例编码为JSON并写入响应体
		json.NewEncoder(w).Encode(resp)
	}
}

func GenerateMultiTagPdf(excelData *ExcelData) {
	barcodePath, err := BarCode(excelData.BoxNum)
	if err != nil {
		fmt.Println("生成条形码失败:", err.Error())
		return
	}

	// 创建二维码图片的文件名
	codeFileName := fmt.Sprintf("qrcode_%s", excelData.BoxNum)
	imagePath := fmt.Sprintf("%s/%s.png", config.ImageDir, codeFileName)

	// 1. 创建二维码对象
	qr2, err := qrcode2.New(excelData.DeviceNos, qrcode.Medium) // Medium 纠错等级
	if err != nil {
		fmt.Println("生成二维码失败:", err.Error())
		return
	}

	// 2. 去掉边距（默认是 4 模块宽）
	qr2.DisableBorder = true

	// 3. 写入文件，指定图片像素大小
	err = qr2.WriteFile(1000, imagePath)
	if err != nil {
		fmt.Println("生成二维码失败:", err.Error())
		return
	}

	//err = qrcode2.WriteFile(excelData.DeviceNos, qrcode.Medium, 1000, imagePath)
	//if err != nil {
	//	fmt.Println("生成二维码失败:", err.Error())
	//	return
	//}

	// 初始化一个pdf
	// param1: P/L 横屏或者竖屏
	// param2: mm, cm ... 间距单位,一般是mm
	// param3: A4,A3,A5 纸张大小
	// param4: 特定的字体样式
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
		Size: gofpdf.SizeType{
			Wd: 1000,
			Ht: 600,
		},
		FontDirStr: "",
	})

	// 添加中文字体支持
	fontPaths := findfont.List()
	var fontPath string
	for _, path := range fontPaths {
		if strings.Contains(path, "msyh.ttf") || strings.Contains(path, "simhei.ttf") || strings.Contains(path, "simsun.ttc") || strings.Contains(path, "simkai.ttf") {
			fontPath = path
			break
		}
	}
	//fontPath = "./PingFang Regular_0.ttf"

	if fontPath != "" {
		// 添加UTF-8字体支持
		pdf.AddUTF8Font("微软雅黑", "", fontPath)
		pdf.AddUTF8Font("微软雅黑", "B", fontPath)
		// 设置字体为微软雅黑
		pdf.SetFont("微软雅黑", "", 100)
	} else {
		// 如果找不到中文字体，则使用默认字体
		pdf.SetFont("Arial", "B", 100)
	}

	// 添加新的空白页,在写入内容之前,一定要添加一个空白页
	pdf.AddPage()

	pdf.ImageOptions(
		imagePath,
		640, 240, // 20，20
		340, 340, // 800，800
		false,
		gofpdf.ImageOptions{ImageType: "png", ReadDpi: false, AllowNegativePosition: true},
		0,
		"",
	)

	pdf.ImageOptions(
		"resources/images/"+excelData.BarCode69Type,
		20, 230,
		580, 165, // 600，170
		false,
		gofpdf.ImageOptions{ImageType: "png", ReadDpi: false, AllowNegativePosition: true},
		0,
		"",
	)

	pdf.ImageOptions(
		barcodePath,
		20, 410, // 20，20
		560, 110, // 800，800
		false,
		gofpdf.ImageOptions{ImageType: "png", ReadDpi: false, AllowNegativePosition: true},
		0,
		"",
	)

	pdf.Text(40, 60, "产品名称: "+excelData.ProductName)
	pdf.Text(40, 120, "产品颜色: "+excelData.ProductColor)
	pdf.Text(40, 180, "产品日期: "+excelData.ProductDate)

	pdf.Text(570, 60, "产品数量: "+excelData.ProductNum+"PCS")
	pdf.Text(570, 120, "净    重: "+excelData.NetWeight+"KG")
	pdf.Text(570, 180, "毛    重: "+excelData.GrossWeight+"KG")

	pdf.Text(640, 230, "SN:")

	pdf.Text(90, 560, "箱号:"+excelData.BoxNum)

	pdfPath := fmt.Sprintf("%s/%s.pdf", config.PdfDir, excelData.BoxNum)
	herr := pdf.OutputFileAndClose(pdfPath)
	if herr != nil {
		fmt.Println(herr.Error())
	}

	fmt.Println("箱号[", excelData.BoxNum, "]开始打印")
	pwd, _ := os.Getwd()
	// 打印二维码
	CmdBlockExec("cmd", "/k", "start", config.AdobePath, "/p", "/h", filepath.Join(pwd, pdfPath))
	time.Sleep(time.Duration(config.PrintInterval) * time.Second)
	fmt.Println("箱号[", excelData.BoxNum, "]标签生成成功")
}

func GenerateMultiPdfByExcel(excelData *ExcelData) {
	barcodePath, err := BarCode(excelData.BoxNum)
	if err != nil {
		fmt.Println("生成条形码失败:", err.Error())
		return
	}

	// 创建二维码图片的文件名
	codeFileName := fmt.Sprintf("qrcode_%s", excelData.BoxNum)
	imagePath := fmt.Sprintf("%s/%s.png", config.ImageDir, codeFileName)

	// 1. 创建二维码对象
	qr2, err := qrcode2.New(excelData.DeviceNos, qrcode.Medium) // Medium 纠错等级
	if err != nil {
		fmt.Println("生成二维码失败:", err.Error())
		return
	}

	// 2. 去掉边距（默认是 4 模块宽）
	qr2.DisableBorder = true

	// 3. 写入文件，指定图片像素大小
	err = qr2.WriteFile(1000, imagePath)
	if err != nil {
		fmt.Println("生成二维码失败:", err.Error())
		return
	}

	//err = qrcode2.WriteFile(excelData.DeviceNos, qrcode.Medium, 1000, imagePath)
	//if err != nil {
	//	fmt.Println("生成二维码失败:", err.Error())
	//	return
	//}

	// 初始化一个pdf
	// param1: P/L 横屏或者竖屏
	// param2: mm, cm ... 间距单位,一般是mm
	// param3: A4,A3,A5 纸张大小
	// param4: 特定的字体样式
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
		Size: gofpdf.SizeType{
			Wd: 1000,
			Ht: 600,
		},
		FontDirStr: "",
	})

	// 添加中文字体支持
	fontPaths := findfont.List()
	var fontPath string
	for _, path := range fontPaths {
		if strings.Contains(path, "msyh.ttf") || strings.Contains(path, "simhei.ttf") || strings.Contains(path, "simsun.ttc") || strings.Contains(path, "simkai.ttf") {
			fontPath = path
			break
		}
	}
	//fontPath = "./PingFang Regular_0.ttf"

	if fontPath != "" {
		// 添加UTF-8字体支持
		pdf.AddUTF8Font("微软雅黑", "", fontPath)
		pdf.AddUTF8Font("微软雅黑", "B", fontPath)
		// 设置字体为微软雅黑
		pdf.SetFont("微软雅黑", "", 100)
	} else {
		// 如果找不到中文字体，则使用默认字体
		pdf.SetFont("Arial", "B", 100)
	}

	// 添加新的空白页,在写入内容之前,一定要添加一个空白页
	pdf.AddPage()

	pdf.ImageOptions(
		imagePath,
		640, 240, // 20，20
		340, 340, // 800，800
		false,
		gofpdf.ImageOptions{ImageType: "png", ReadDpi: false, AllowNegativePosition: true},
		0,
		"",
	)

	pdf.ImageOptions(
		"resources/images/"+excelData.BarCode69Type,
		10, 230,
		580, 165, // 600，170
		false,
		gofpdf.ImageOptions{ImageType: "png", ReadDpi: false, AllowNegativePosition: true},
		0,
		"",
	)

	pdf.ImageOptions(
		barcodePath,
		26, 410, // 20，20
		560, 110, // 800，800
		false,
		gofpdf.ImageOptions{ImageType: "png", ReadDpi: false, AllowNegativePosition: true},
		0,
		"",
	)

	pdf.Text(40, 60, "产品名称："+excelData.ProductName)
	pdf.Text(40, 120, "产品颜色："+excelData.ProductColor)
	pdf.Text(40, 180, "产品日期："+excelData.ProductDate)

	pdf.Text(570, 60, "产品数量："+excelData.ProductNum+"PCS")
	pdf.Text(570, 120, "净    重："+excelData.NetWeight+"KG")
	pdf.Text(570, 180, "毛    重："+excelData.GrossWeight+"KG")

	pdf.Text(640, 230, "SN：")

	pdf.Text(90, 560, "箱号："+excelData.BoxNum)

	pdfPath := fmt.Sprintf("%s/%s.pdf", config.PdfDir, excelData.FileName+"_"+excelData.BoxNum)
	herr := pdf.OutputFileAndClose(pdfPath)
	if herr != nil {
		fmt.Println(herr.Error())
	}

	//fmt.Println("设备号[", fileName, "]开始打印")
	//pwd, _ := os.Getwd()
	// 打印二维码
	//CmdBlockExec("cmd", "/k", "start", config.AdobePath, "/p", "/h", filepath.Join(pwd, pdfPath))
	//time.Sleep(time.Duration(config.PrintInterval) * time.Second)

	////cmd := exec.Command(adobePath, "/p", "/h", "/n", "/s", "/o", pdfPath)
	////go CmdBlockExec("cmd", "/k", "start", adobePath, "/p", "/h", pdfPath)

	//后台打印二维码
	////cmd := exec.Command("cmd","/c", "start","/b","", config.AdobePath, "/p", "/h", filepath.Join(pwd, pdfPath))
	////err = cmd.Start() // Start 不会阻塞
	////if err != nil {
	////	fmt.Println("打印启动失败:", err)
	////}

	// 删除pdf
	//os.Remove(filepath.Join(pwd, pdfPath))
	// 删除图片
	//os.Remove(filepath.Join(pwd, imagePath))
	fmt.Println("箱号[", excelData.BoxNum, "]标签生成成功")
}

// CmdSyncExec 协程执行命令
func CmdSyncExec(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	if err := cmd.Start(); err != nil {
		//fmt.Println(err.Error())
	}
	fmt.Println(cmd.String())
	//wait for command to finishing ...
	if err := cmd.Wait(); err != nil {
		//fmt.Println(err.Error())
	}
	//os.Exit(-1)
	//time.Sleep(5 * time.Second)
}

// CmdBlockExec 阻塞执行命令
func CmdBlockExec(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	if err := cmd.Run(); err != nil {
		//fmt.Println(err.Error())
	}
	fmt.Println(cmd.String())

	defer cmd.Process.Kill()
}

// StringToInt string 转 int
func StringToInt(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return num
}

func BarCode(content string) (imagePath string, err error) {

	// 生成条形码
	var barCode barcode.Barcode
	barCode, err = code128.Encode(content)
	if err != nil {
		fmt.Println("生成条形码失败：", err.Error())
		return "", err
	}

	// 可选：调整条形码大小
	barCode, err = barcode.Scale(barCode, 200, 50) // 宽200px，高50px
	if err != nil {
		fmt.Println("调整条形码大小失败：", err.Error())
		return "", err
	}

	// 创建输出文件
	filePath := config.ImageDir + "/" + content + ".png"
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Println("创建目录失败：", err.Error())
		return "", err
	}

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("创建输出文件失败：", err.Error())
		return "", err
	}
	defer file.Close()

	// 将条形码转换为标准RGBA图像以避免16位色深问题
	bounds := barCode.Bounds()
	img := image.NewRGBA(bounds)
	draw.Draw(img, bounds, barCode, bounds.Min, draw.Src)

	// 将条形码编码为 PNG 并写入文件
	err = png.Encode(file, img)
	if err != nil {
		fmt.Println("将条形码编码为 PNG 失败：", err.Error())
		return "", err
	}

	println("条形码已生成：" + filePath)
	return filePath, nil
}

// BarCode69 生成69码(EAN-13)条形码
func BarCode69(content string) (imagePath string, err error) {
	// 验证输入
	if len(content) != 13 {
		return "", fmt.Errorf("69码内容必须是13位数字")
	}

	if !strings.HasPrefix(content, "69") {
		return "", fmt.Errorf("69码必须以69开头")
	}

	// 验证是否全部为数字
	for _, r := range content {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("69码只能包含数字")
		}
	}

	// 生成EAN-13条形码
	var barCode barcode.Barcode
	barCode, err = ean.Encode(content)
	if err != nil {
		fmt.Println("生成69码失败：", err.Error())
		return "", err
	}

	// 调整条形码大小
	barCode, err = barcode.Scale(barCode, 300, 100) // 宽300px，高100px
	if err != nil {
		fmt.Println("调整69码大小失败：", err.Error())
		return "", err
	}

	// 创建输出文件
	filePath := "barcode/69_" + content + ".png"
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Println("创建目录失败：", err.Error())
		return "", err
	}

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("创建输出文件失败：", err.Error())
		return "", err
	}
	defer file.Close()

	// 将条形码转换为标准RGBA图像以避免16位色深问题
	bounds := barCode.Bounds()
	img := image.NewRGBA(bounds)
	draw.Draw(img, bounds, barCode, bounds.Min, draw.Src)

	// 将条形码编码为 PNG 并写入文件
	err = png.Encode(file, img)
	if err != nil {
		fmt.Println("将69码编码为 PNG 失败：", err.Error())
		return "", err
	}

	println("69码已生成：" + filePath)
	return filePath, nil
}

type ExcelData struct {
	//产品名称
	ProductName string `json:"productName"`
	//产品颜色
	ProductColor string `json:"productColor"`
	//生产日期
	ProductDate string `json:"productDate"`
	//产品数量
	ProductNum string `json:"productNum"`
	//净重
	NetWeight string `json:"netWeight"`
	//毛重
	GrossWeight string `json:"grossWeight"`
	//设备号
	DeviceNos string `json:"deviceNos"`
	//箱号
	BoxNum string `json:"boxNum"`
	//69码类型
	BarCode69Type string `json:"barCode69Type"`
	//文件名
	FileName string `json:"fileName"`
}

// ParseExcel 解析导入excel文件
func ParseExcel(fileName string) (data []*ExcelData, err error) {
	xlFile, err := xlsx.OpenFile("./" + fileName)
	if err != nil {
		fmt.Println("解析excel文件失败：", err.Error())
		return
	}
	data = make([]*ExcelData, 0)
	for _, sheet := range xlFile.Sheets {
		if len(sheet.Rows) > 1001 {
			err = fmt.Errorf("单次最多导入1000条记录")
			fmt.Println("单次最多导入1000条记录")
			return
		}
		if len(sheet.Rows) == 0 {
			err = fmt.Errorf("文件内容为空")
			fmt.Println("文件内容为空")
			return
		}
		for rowIndex, row := range sheet.Rows {
			if rowIndex == 0 {
				continue
			}
			if len(row.Cells) == 0 {
				continue
			}
			excelData := new(ExcelData)
			for cellIndex, cell := range row.Cells {
				value := strings.TrimSpace(cell.String())
				switch cellIndex {
				case 0: //产品名称
					excelData.ProductName = value
				case 1: //产品颜色
					excelData.ProductColor = value
				case 2: //设备型号
					excelData.BarCode69Type = fmt.Sprintf("%s-69.png", value)
				case 3: //生产日期
					excelData.ProductDate = value
				case 4: //产品数量
					excelData.ProductNum = value
				case 5: //净重
					excelData.NetWeight = value
				case 6: //毛重
					excelData.GrossWeight = value
				case 7: //设备号
					excelData.DeviceNos = value
				}
			}
			if excelData.BoxNum == "" {
				excelData.BoxNum = "C" + time.Now().Format("0102150405") + GenerateRandomNumber(7)
			}
			excelData.FileName = strings.ReplaceAll(fileName, ".xlsx", "")
			if excelData.BarCode69Type == "" {
				excelData.BarCode69Type = "401-69.png"
			}
			data = append(data, excelData)
		}
	}
	if len(data) == 0 {
		err = fmt.Errorf("文件数据为空")
		fmt.Println("文件数据为空")
	}
	return
}

// GenerateRandomNumber 生成指定长度的数字随机数
func GenerateRandomNumber(length int) string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%0"+strconv.Itoa(length)+"d", rand.Intn(int(math.Pow10(length))))
}

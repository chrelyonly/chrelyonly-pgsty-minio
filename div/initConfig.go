package div

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/minio/minio/internal/color"
)

type Person struct {
	UserName     string `json:"userName"`
	Password     string `json:"password"`
	ApiProd      int    `json:"apiProd"`
	WebProd      int    `json:"webProd"`
	FtpProd      int    `json:"ftpProd"`
	FtpProdRange string `json:"ftpProdRange"`
	ResourceData string `json:"resourceData"`
}

func Optimize(args []string) []string {
	//打印当前时间
	currentTime := time.Now()
	fmt.Println(color.GreenBold("当前时间: " + currentTime.Format("2006-01-02 15:04:05")))
	var configPath = ""
	//判断是否传入配置文件路径
	if len(args) > 1 {
		configPath = args[1] + "/config.json"
	} else {
		//获取当前程序目录下的config.json文件
		configPath = "./config.json"
	}
	//判断文件是否存在
	_, err := os.Stat(configPath)
	fmt.Println(color.GreenBold("读取到配置文件,路径: " + configPath))
	//配置信息
	var userName string
	var password string
	var apiProd int
	var webProd int
	var ftpProd int
	var ftpProdRange string
	//文件储存位置为 当前目录下的data文件夹
	var resourceData string
	//新的参数
	var newArgs []string
	//文件对象
	var config *os.File
	if err != nil {
		fmt.Println(color.RedBold("当前配置文件不存在,将进行初始化配置"))
		//创建文件
		config, err = os.Create(configPath)
		if err != nil {
			fmt.Println(color.RedBold("配置文件创建失败,请检查是否有权限"))
			os.Exit(1)
		}
		fmt.Println(color.GreenBold("请输入管理员用户名(默认: minioadmin))"))
		_, err := fmt.Scanln(&userName)
		if err != nil {
			userName = "minioadmin"
			fmt.Println(color.FgWhite("当前用户名: minioadmin"))
		}
		fmt.Println(color.Green("请输入管理员密码(默认: minioadmin)"))
		_, err = fmt.Scanln(&password)
		if err != nil {
			password = "minioadmin"
			fmt.Println(color.FgWhite("当前密码: minioadmin"))
		}
		fmt.Println(color.Green("请输入API接口端口(默认: 30000)"))
		_, err = fmt.Scanln(&apiProd)
		if err != nil {
			apiProd = 30000
			fmt.Println(color.FgWhite("当前API端口: 30000"))
		}
		fmt.Println(color.Green("请输入web控制台端口(默认: 30001)"))
		_, err = fmt.Scanln(&webProd)
		if err != nil {
			webProd = 30001
			fmt.Println(color.FgWhite("当前web控制台端口: 30001"))
		}
		fmt.Println(color.Green("请输入FTP端口(默认: 30002)"))
		_, err = fmt.Scanln(&ftpProd)
		if err != nil {
			ftpProd = 30002
			fmt.Println(color.FgWhite("当前FTP端口: 30002"))
		}
		fmt.Println(color.Green("请输入FTP被动模式端口范围(默认: 40000-50000)"))
		_, err = fmt.Scanln(&ftpProdRange)
		if err != nil {
			ftpProdRange = "40000-50000"
			fmt.Println(color.FgWhite("当前FTP被动模式端口范围: 40000-50000"))
		}
		fmt.Println(color.Green("输入储存目录(默认: data)"))
		_, err = fmt.Scanln(&resourceData)
		if err != nil {
			resourceData = "data"
			fmt.Println(color.FgWhite("当前输入储存目录: data"))
		}
		//将配置信息保存
		person := Person{
			UserName:     userName,
			Password:     password,
			ApiProd:      apiProd,
			WebProd:      webProd,
			FtpProd:      ftpProd,
			FtpProdRange: ftpProdRange,
			ResourceData: resourceData,
		}
		//将配置信息写入文件
		encoder := json.NewEncoder(config)
		err = encoder.Encode(person)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		//	如果存在文件则,从文件中读取配置
		fmt.Println(color.GreenBold("当前配置文件存在,将从配置文件中读取配置"))
		//读取配置文件,json格式
		config, err = os.OpenFile(configPath, os.O_RDWR, 0666)
		if err != nil {
			fmt.Println(color.RedBold("配置文件读取失败,请检查配置文件是否存在"))
			os.Exit(1)
		}
		// 从文件中读取 JSON 数据
		decoder := json.NewDecoder(config)
		var person Person
		err = decoder.Decode(&person)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		userName = person.UserName
		password = person.Password
		apiProd = person.ApiProd
		webProd = person.WebProd
		ftpProd = person.FtpProd
		ftpProdRange = person.FtpProdRange
		resourceData = person.ResourceData
	}

	//关闭文件
	err = config.Close()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//账号密码
	err = os.Setenv("MINIO_ROOT_USER", userName)
	if err != nil {
		fmt.Println("账号密码设置失败")
		os.Exit(1)
	}
	//账号密码
	err = os.Setenv("MINIO_ROOT_PASSWORD", password)
	if err != nil {
		fmt.Println("账号密码设置失败")
		os.Exit(1)
	}
	appName := filepath.Base(args[0])
	args = append(newArgs, appName, "server", resourceData,
		"--console-address", ":"+strconv.Itoa(webProd),
		"--address", ":"+strconv.Itoa(apiProd),
		"--ftp=address=:"+strconv.Itoa(ftpProd),
		"--ftp=passive-port-range="+ftpProdRange)
	return args
}

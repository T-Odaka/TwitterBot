package main

import (
	"bufio"
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/sclevine/agouti"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const osWindows string = "windows"
const osMac string = "darwin"
const osLinux string = "linux"

var pathSeparate string = ":"
var fileSeparate string = "/"

type Param struct {
	Id int `json: "id"`
	XPath string `json: "xpath"`
	Control string `json: "control"`
	Text string `json: "text"`
	URL string `json: "url"`
}

type Params struct {
	Data []Param `json: "data"`
}

type Pm map[int]Param

func indexHandler(c echo.Context) error {
	data := struct {
		IP string
	}{
		IP: c.Request().Host,
	}

	return c.Render(http.StatusOK, "index", data)
}

func (rhs *runHandleStruct) runHandler(c echo.Context) error {
	// Pmの初期化
	param := new(Pm)

	if err := c.Bind(param); err != nil {
		log.Fatal(err)
	}

	// 取得したパラメータをrunHandleStructチャネルへ送信
	rhs.param <- *param

	// HTMLにレスポンスを返送
	return c.JSON(http.StatusOK, param)
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// runハンドラ実行結果を取得するための構造体
// Pm型のparamフィールドのチャネルで通信します。
type runHandleStruct struct {
	param chan Pm
}

func main() {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())

	// アプリのディレクトリを取得する
	dir, err := os.Executable()

	if err = setENV(dir); err != nil {
		log.Fatal(err)
	}

	t := &Template{templates: template.Must(template.ParseGlob(fmt.Sprintf("%s/public/*.html",getPublicFilePath(dir))))}
	e.Renderer = t

	rhs := runHandleStruct{param: make(chan Pm)}

	// Webサーバの開始用のゴルーチン
	// メインゴルーチンで実行してしまうと、他の処理をブロックしてしまうため、ゴルーチンを分けています。
	go func(e *echo.Echo, rhs runHandleStruct) {
		e.GET("/", indexHandler)
		e.POST("/run", rhs.runHandler)
		e.Logger.Fatal(e.Start(":1323"))
	}(e, rhs)

	// Webドライバの初期化（Chromeの場合）
	driver := agouti.ChromeDriver(
		agouti.ChromeOptions("args", []string{
			// ヘッドレスモード（ブラウザ非表示）でChrome起動の設定
			//"--headless",
			// Windowのサイズを1280x720にする
			"--window-size=1280,720",
			// ログイン情報等を保存するディレクトリを指定。これによって、Twitterのログイン情報を保持することができる
			"--user-data-dir=" + getUserDataPath(dir),
		}),
	)

	err = driver.Start()
	defer func() {
		err = driver.Stop()
		if err != nil {
			log.Println(err)
		}
	}()

	if err != nil {
		log.Fatal(err)
	}

	// クロームを起動。page型の返り値（セッション）を返す。
	page, err := driver.NewPage(agouti.Browser("chrome"))
	if err != nil {
		log.Fatal(err)
	}

	// runHandleStructを宣言、paramフィールドにPmのチャネルを宣言
	//rhs := runHandleStruct{param: make(chan Pm)}

	// paramフィールドのチャネルを受信するためのゴルーチンを開始
	go getParameterChan(rhs, page)

	err = page.Navigate("localhost:1323") // 指定したurlにアクセスする
	if err != nil {
		log.Fatal(err)
	}

	// finという名前の構造体チャネルを作成
	fin := make(chan struct{}, 1)
	// mainプログラム（メインゴルーチン）終了時に、チャネルを開放する（チャネル使用時は開放は必須。）
	defer close(fin)

	// 別のプロセス（ゴルーチン）で実行する（並列処理）。
	go finishCheck(fin)

	// finチャネルに値が送信するまで待つ
	<-fin
}

// runHandlerに送信されてきたリクエスト毎にブラウザの操作
func getParameterChan(rhs runHandleStruct, page *agouti.Page) {
	for{
		p := <-rhs.param
		if err := page.Navigate(p[0].URL); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("chan: %v\n",p)

		for i, _ := range p {
			switch p[i].Control {
			case "データ抽出":
				time.Sleep(1 * time.Second)
				s, err := getByXPath(page, p[i].XPath)
				file, err := os.Create("out.txt")

				defer func(){
					if err := file.Close(); err != nil {
						log.Fatal(err)
					}
				}()

				if err != nil {
					log.Fatal(err)
				}

				if _, err = file.Write([]byte(s)); err != nil {
					log.Fatal(err)
				}
			case "クリック":
				time.Sleep(1 * time.Second)
				if err := clickByXPath(page, p[i].XPath); err != nil {
					log.Fatal(err)
				}
			case "入力":
				time.Sleep(1 * time.Second)
				if err := inputByXPath(page, p[i].XPath, p[i].Text); err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}

func finishCheck(fin chan<- struct{}) {
	// 標準入力を取得する
	sc := bufio.NewScanner(os.Stdin)
	for {
		// 1秒待つ
		time.Sleep(1 * time.Second)
		// １行取得（文字の末尾に改行が入っていること）
		sc.Scan()
		// 入力された文字がquitまたはexitだった場合は、finチャネルに空構造体を送信する
		if sc.Text() == "quit" || sc.Text() == "exit" {
			fin <- struct{}{}
		}
	}
}

func getByXPath(page *agouti.Page, xpath string) (string, error) {
	item := page.FindByXPath(xpath)
	return item.Text()
}

func getAllByClass(page *agouti.Page, className string) ([]string, error) {
	// ニュース蘭の情報を取得する
	item := page.AllByClass(className)
	count, err := item.Count()

	if err != nil {
		return nil, err
	}

	var result []string = []string{}

	for i := 0; i < count; i++ {
		text, err := item.At(i).Text()
		if err != nil {
			return nil, err
		}
		result = append(result, text)
	}

	return result, nil
}

func inputByXPath(page *agouti.Page, xpath, input string) error {
	if err := page.FindByXPath(xpath).Fill(input); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return nil
}

func clickByXPath(page *agouti.Page, xpath string) error {
	if err := page.FindByXPath(xpath).Click(); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return nil
}

func setENV(dir string) error {
	// ファイルドライバのパスを取得する
	pathEnv := []string{os.Getenv("PATH"), getDriverPath(dir)}
	// 環境変数PATHに、ファイルドライバのパスを設定する
	err := os.Setenv("PATH", strings.Join(pathEnv, pathSeparate))

	return err
}

func getDriverPath(dir string) (path string) {
	var twitterBotPath string = strings.Split(dir, "TwitterBot")[0]

	switch runtime.GOOS {
	case osWindows:
		pathSeparate = ";"
		fileSeparate = "\\"
		path = fmt.Sprintf("%s%s", twitterBotPath, filepath.FromSlash("TwitterBot/drivers/win32"))
	case osMac:
		path = fmt.Sprintf("%s%s", twitterBotPath, "TwitterBot/drivers/mac")
	case osLinux:
		path = fmt.Sprintf("%s%s", twitterBotPath, "TwitterBot/drivers/linux")
	default:
		log.Fatal("OS could not be determined.")
	}
	return
}

func getUserDataPath(dir string) (path string) {
	var twitterBotPath string = strings.Split(dir, "TwitterBot")[0]

	switch runtime.GOOS {
	case osWindows:
		path = fmt.Sprintf("%s%s", twitterBotPath, filepath.FromSlash("TwitterBot/ChromeUserData"))
	case osMac:
		path = fmt.Sprintf("%s%s", twitterBotPath, "TwitterBot/ChromeUserData")
	case osLinux:
		path = fmt.Sprintf("%s%s", twitterBotPath, "TwitterBot/ChromeUserData")
	default:
		log.Fatal("OS could not be determined.")
	}
	return
}

func getPublicFilePath(dir string) (path string) {
	var twitterBotPath string = strings.Split(dir, "TwitterBot")[0]

	switch runtime.GOOS {
	case osWindows:
		path = fmt.Sprintf("%s%s", twitterBotPath, filepath.FromSlash("TwitterBot/sample/Sample"))
	case osMac:
		path = fmt.Sprintf("%s%s", twitterBotPath, "TwitterBot/sample/Sample")
	case osLinux:
		path = fmt.Sprintf("%s%s", twitterBotPath, "TwitterBot/sample/Sample")
	default:
		log.Fatal("OS could not be determined.")
	}
	return
}
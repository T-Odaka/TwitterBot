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

const url string = "https://www.yahoo.co.jp/"
const osWindows string = "windows"
const osMac string = "darwin"
const osLinux string = "linux"

var pathSeparate string = ":"
var fileSeparate string = "/"

func indexHandler(c echo.Context) error {
	data := struct {
		IP string
	}{
		IP: c.Request().Host,
	}

	return c.Render(http.StatusOK, "index", data)
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())

	t := &Template{templates: template.Must(template.ParseGlob("public/*.html"))}
	e.Renderer = t

	go func(e *echo.Echo) {
		e.GET("/", indexHandler)
		e.Logger.Fatal(e.Start(":1323"))
	}(e)

	// アプリのディレクトリを取得する
	dir, err := os.Getwd()

	if err = setENV(dir); err != nil {
		log.Fatal(err)
	}

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

	err = page.Navigate(url) // 指定したurlにアクセスする
	if err != nil {
		log.Fatal(err)
	}

	// ニュース蘭の情報を取得する
	classes, err := getAllByClass(page, "_2j0udhv5jERZtYzddeDwcv")

	for _, class := range classes {
		fmt.Println(class)
	}

	time.Sleep(1 * time.Second)

	// XPathにFillに"Golang"を入力
	if err = inputByXPath(page, "/html/body/div/div[1]/header/section[1]/div/form/fieldset/span/input", "Golang"); err != nil {
		log.Println(err)
	}

	// XPathに指定された要素をクリック
	if err = clickByXPath(page, "/html/body/div/div[1]/header/section[1]/div/form/fieldset/span/button/span"); err != nil {
		log.Println(err)
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

func getAllByClass(page *agouti.Page, className string) ([]string, error) {
	// ニュース蘭の情報を取得する
	item := page.AllByClass(className)
	count, err := item.Count()

	var result []string = []string{}

	for i := 0; i < count; i++ {
		text, err := item.At(i).Text()
		if err != nil {
			break
		}
		result = append(result, text)
	}

	return result, err
}

func inputByXPath(page *agouti.Page, xpath, input string) error {
	err := page.FindByXPath(xpath).Fill(input)
	time.Sleep(1 * time.Second)
	return err
}

func clickByXPath(page *agouti.Page, xpath string) error {
	err := page.FindByXPath(xpath).Click()
	time.Sleep(1 * time.Second)
	return err
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

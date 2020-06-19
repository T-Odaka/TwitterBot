package main

import (
	"bufio"
	"fmt"
	"github.com/sclevine/agouti"
	"log"
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

func getDriverPath(dir string) string {
	var path string

	switch runtime.GOOS {
	case osWindows:
		pathSeparate = ";"
		fileSeparate = "\\"
		path = fmt.Sprintf("%s%s", strings.Split(dir, "TwitterBot")[0], filepath.FromSlash("TwitterBot/drivers/win32"))
	case osMac:
		path = fmt.Sprintf("%s%s", strings.Split(dir, "TwitterBot")[0], "TwitterBot/drivers/mac")
	case osLinux:
		path = fmt.Sprintf("%s%s", strings.Split(dir, "TwitterBot")[0], "TwitterBot/drivers/linux")
	default:
		log.Fatal("OS could not be determined.")
	}
	return path
}

func main() {
	// アプリのディレクトリを取得する
	dir, _ := os.Getwd()
	fmt.Println(dir)

	// ファイルドライバのパスを取得する
	pathEnv := []string{os.Getenv("PATH"), getDriverPath(dir)}

	// 環境変数PATHに、ファイルドライバのパスを設定する
	_ = os.Setenv("PATH", strings.Join(pathEnv, pathSeparate))
	fmt.Println(os.Getenv("PATH"))

	const url string = "https://www.yahoo.co.jp/"

	driver := agouti.ChromeDriver(
		agouti.ChromeOptions("args", []string{
			// ヘッドレスモード（ブラウザ非表示）でChrome起動の設定
			//"--headless",
			// Windowのサイズを1280x720にする
			"--window-size=1280,720",
			// ログイン情報等を保存するディレクトリを指定。これによって、Twitterのログイン情報を保持することができる
			"--user-data-dir=./ChromeUserData",
		}),
	)

	err := driver.Start()
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
	s := page.AllByClass("_2j0udhv5jERZtYzddeDwcv")
	max, _ := s.Count()

	for i := 0; i < max; i++ {
		fmt.Println(s.At(i).Text())
	}

	time.Sleep(1 * time.Second)
	// XPathにFillに"Golang"を入力
	err = page.FindByXPath("/html/body/div/div[1]/header/section[1]/div/form/fieldset/span/input").Fill("Golang")
	if err != nil {
		log.Println(err)
	}

	time.Sleep(1 * time.Second)
	// XPathに指定された要素をクリック
	err = page.FindByXPath("/html/body/div/div[1]/header/section[1]/div/form/fieldset/span/button/span").Click()
	if err != nil {
		log.Println(err)
	}

	// finという名前の構造体チャネルを作成
	fin := make(chan struct{}, 1)
	// mainプログラム（メインゴルーチン）終了時に、チャネルを開放する（チャネル使用時は開放は必須。）
	defer close(fin)

	// 別のプロセス（ゴルーチン）で無名関数を実行する（並列処理）。
	go func(fin chan<- struct{}) {
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
	}(fin)

	// finチャネルに値が送信するまで待つ
	<- fin
}

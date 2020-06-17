package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sclevine/agouti"
)

func main() {
	const url string = "https://www.yahoo.co.jp/"

	// current dirにChromeUserDataというディレクトリが存在するか確認し、なければ作成する
	if _, err := os.Stat("./ChromeUserData"); os.IsExist(err) {
		fmt.Println("creating ChromeUserData files.")
		if err := os.Mkdir("ChromeUserData", 0777); err != nil {
			log.Fatal(err)
		}
	}

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
		log.Fatal("Fatal to start driver:")
	}

	// クロームを起動。page型の返り値（セッション）を返す。
	page, err := driver.NewPage(agouti.Browser("chrome"))
	if err != nil {
		log.Printf("Failed to open page: %v", err)
	}

	err = page.Navigate(url) // 指定したurlにアクセスする
	if err != nil {
		log.Printf("Failed to navigate: %v", err)
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

	// Twitterのログイン確認用に２分間のWaitを入れる
	time.Sleep(2 * time.Minute)
}

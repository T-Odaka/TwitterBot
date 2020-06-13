package main

import (
	"fmt"
	"log"
	"time"

	"github.com/sclevine/agouti"
)

func main() {
	const url string = "https://www.yahoo.co.jp/"

	driver := agouti.ChromeDriver(
		agouti.ChromeOptions("args", []string{
			// ヘッドレスモード（ブラウザ非表示）でChrome起動の設定
			"--headless",
		}),
	)
	err := driver.Start()
	defer driver.Stop()
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

	for i:=0;i<max;i++ {
		fmt.Println(s.At(i).Text())
	}

	time.Sleep(1 * time.Second)
	// XPathにFillに"Golang"を入力
	page.FindByXPath("/html/body/div/div[1]/header/section[1]/div/form/fieldset/span/input").Fill("Golang")
	time.Sleep(1 * time.Second)
	// XPathに指定された要素をクリック
	page.FindByXPath("/html/body/div/div[1]/header/section[1]/div/form/fieldset/span/button/span").Click()
	time.Sleep(5 * time.Second)
}

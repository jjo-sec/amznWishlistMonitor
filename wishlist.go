package main

import (
	"flag"
	"fmt"
	"github.com/gookit/color"
	"github.com/sclevine/agouti"
	log "github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	log.SetOutput(os.Stdout)

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetLevel(log.WarnLevel)
	var wishlistId string
	flag.StringVar(&wishlistId, "w", "", "Wishlist ID to check")
	flag.Parse()
	if wishlistId == "" {
		log.Fatal("Wishlist ID must be set via -w")
	}
	driver := agouti.ChromeDriver(agouti.ChromeOptions("args", []string{"--headless", "window-size=1440x900", "--disable-gpu", "--no-sandbox"}))
	if err := driver.Start(); err != nil {
		log.Fatal("Failed to start driver:", err)
	}

	page, err := driver.NewPage()
	if err != nil {
		log.Fatal("Failed to open page:", err)
	}
	if err := page.Navigate(fmt.Sprintf("https://www.amazon.com/gp/registry/wishlist/%s/ref=nav_wishlist_lists_1", wishlistId)); err != nil {
		log.Fatal("Failed to navigate:", err)
	}
	for count := 0; count < 5; count++ {
		var value string
		var err error
		err = page.RunScript(`window.scrollBy(0,1400);`, nil, &value)
		if err != nil {
			log.Error(err)
		}
		err = page.RunScript(`window.scrollBy(0,1400);`, nil, &value)
		if err != nil {
			log.Error(err)
		}
		time.Sleep(time.Second * 3)
		eolM := page.FindByID(`endOfListMarker`)
		ec, _ := eolM.Count()
		if ec > 0 {
			break
		}
	}
	books := page.AllByClass(`g-item-sortable`)
	bookChan := make(chan *agouti.Selection)
	finishedChan := make(chan bool)
	cnt, _ := books.Count()
	go PrintBook(bookChan, finishedChan)
	for i := 0; i < cnt; i++ {
		bookChan <- books.At(i)
	}
	close(bookChan)
	<-finishedChan
	close(finishedChan)

	if err := driver.Stop(); err != nil {
		log.Fatal("Failed to close pages and stop WebDriver:", err)
	}

}

func PrintBook(bookChan chan *agouti.Selection, finishedChan chan bool) {
	error_ := color.RGB(255, 0, 0)
	warn := color.RGB(255, 95, 0)
	info := color.RGB(135, 255, 0)
	spam := color.RGB(0, 95, 0)
	notice := color.RGB(255, 215, 0)
	critical := color.Style{color.FgWhite, color.BgRed, color.Bold}

	for {
		book, more := <-bookChan
		if !more {
			finishedChan <- true
			break
		}
		bkId, _ := book.Attribute(`data-itemid`)
		bkTitle, _ := book.Find(fmt.Sprintf("a[id=itemName_%s]", bkId)).Text()
		bkAuthor, _ := book.Find(fmt.Sprintf("span[id=item-byline-%s]", bkId)).Text()
		prDrop, err := book.FindByClass("itemPriceDrop").Text()
		if err != nil {
			prDrop = ""
		}
		sPrice, _ := book.Attribute(`data-price`)
		price, _ := strconv.ParseFloat(sPrice, 64)
		if prDrop != "" {
			regex := regexp.MustCompile("Price dropped (?P<drop_percent>[0-9]+)%")
			match := regex.FindStringSubmatch(prDrop)
			var drpPct = 0
			if len(match) == 2 {
				drpPct, _ = strconv.Atoi(match[1])
			}
			bkStr := fmt.Sprintf("%s %s $%.02f - %s", bkTitle, bkAuthor, price, prDrop)
			if drpPct >= 70 || price < 5 {
				PrintBookInfo("CRITICAL", critical.Sprint(bkStr))
			} else if drpPct >= 50 {
				PrintBookInfo("ERROR", color.Bold.Sprint(error_.Sprint(bkStr)))
			} else if drpPct >= 25 {
				PrintBookInfo("WARNING", color.Bold.Sprint(warn.Sprint(bkStr)))
			} else if drpPct >= 10 {
				PrintBookInfo("NOTICE", color.Bold.Sprint(notice.Sprint(bkStr)))
			} else {
				PrintBookInfo("INFO", color.Bold.Sprint(info.Sprint(bkStr)))
			}
		} else if price < 5 {
			PrintBookInfo("CRITICAL", critical.Sprintf("%s %s $%.02f", bkTitle, bkAuthor, price))
		} else {
			PrintBookInfo("SPAM", spam.Sprintf("%s %s $%.02f", bkTitle, bkAuthor, price))
		}
	}
}

func PrintBookInfo(level string, message string) {
	tmStr := color.RGB(0, 155, 0).Sprintf("[%s]", time.Now().Format("2006-01-02 15:04:05"))
	levelFmt := color.Style{color.FgDarkGray, color.OpBold}

	fmt.Println(tmStr, levelFmt.Sprintf("[%s]", strings.ToUpper(level)), message)
}

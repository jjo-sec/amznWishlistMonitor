package main

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/sclevine/agouti"
	log "github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strconv"
	"time"
)

func main() {
	//log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Only log the warning severity or above.
	log.SetLevel(log.TraceLevel)
	driver := agouti.ChromeDriver(agouti.ChromeOptions("args", []string{"--headless", "window-size=1440x900", "--disable-gpu", "--no-sandbox"}),)
	if err := driver.Start(); err != nil {
		log.Fatal("Failed to start driver:", err)
	}

	page, err := driver.NewPage()
	if err != nil {
		log.Fatal("Failed to open page:", err)
	}
	if err := page.Navigate("https://www.amazon.com/gp/registry/wishlist/1NI6JGK58RCSE/ref=nav_wishlist_lists_1"); err != nil {
		log.Fatal("Failed to navigate:", err)
	}
	for count := 0; count < 5; count++ {
		var value string
		page.RunScript(`window.scrollBy(0,1400);`, nil, &value)
		page.RunScript(`window.scrollBy(0,1400);`, nil, &value)
		time.Sleep(time.Second * 3)
		eolM := page.FindByID(`endOfListMarker`)
		ec, _ := eolM.Count()
		if ec > 0 {
			break
		}
	}
	books := page.AllByClass(`g-item-sortable`)
	cnt, _ := books.Count()
	for i:=0; i < cnt; i++ {
		bkId, _ := books.At(i).Attribute(`data-itemid`)
		bkTitle, _ := books.At(i).Find(fmt.Sprintf("a[id=itemName_%s]",bkId)).Text()
		bkAuthor, _ := books.At(i).Find(fmt.Sprintf("span[id=item-byline-%s]", bkId)).Text()
		prDrop, err := books.At(i).FindByClass("itemPriceDrop").Text()
		if err != nil {
			prDrop = ""
		}
		sPrice, _ := books.At(i).Attribute(`data-price`)
		price, _ := strconv.ParseFloat(sPrice, 64)
		if (prDrop != "") {
			regex := regexp.MustCompile("Price dropped (?P<drop_percent>[0-9]+)%")
			match := regex.FindStringSubmatch(prDrop)
			var drpPct = 0
			if (len(match) == 2) {
				drpPct, _ = strconv.Atoi(match[1])
			}
			bkStr := fmt.Sprintf("%s %s $%.02f - %s", bkTitle, bkAuthor, price, prDrop)
			if drpPct >= 70 || price < 5 {
				log.Error(aurora.BgBrightRed(aurora.Bold(aurora.White(bkStr))))
			} else if drpPct >= 50 {
				log.Error(aurora.BrightRed(aurora.Bold(bkStr)))
			} else if drpPct >= 25 {
				log.Warn(aurora.Bold(aurora.BrightBlue(bkStr)))
			} else if drpPct >= 10 {
				log.Info(aurora.Bold(aurora.BrightYellow(bkStr)))
			} else {
				log.Debug(aurora.BrightGreen(aurora.Bold(bkStr)))
			}
		} else if (price < 5) {
			log.Error(aurora.BgBrightRed(aurora.Bold(aurora.White(fmt.Sprintf("%s %s $%.02f", bkTitle, bkAuthor, price)))))
		} else {
			log.Trace(fmt.Sprintf("%s %s $%.02f", bkTitle, bkAuthor, price))
		}
	}
	/*amzn_html, _ := page.HTML()
	r := strings.NewReader(amzn_html)
	parser := html.NewTokenizer(r)*/
	if err := driver.Stop(); err != nil {
		log.Fatal("Failed to close pages and stop WebDriver:", err)
	}

}

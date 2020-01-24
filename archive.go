package keralalottery

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

// Downloader is the function to handle draw all results download for a single lottery
type Downloader func(Draw)

// GetLotteriesList get the list of all lotteries listed
func GetLotteriesList(domain string) []Lottery {
	var lotteries []Lottery
	url := fmt.Sprintf("%s/lottery/detailsofdrawweb.php", domain)
	c := colly.NewCollector()

	// Find and visit all links
	c.OnHTML("select#lotterydet option", func(e *colly.HTMLElement) {
		lotteries = append(lotteries, Lottery{strings.TrimSpace(e.Text), e.Attr("value")})
	})

	c.Visit(url)
	return lotteries
}

// GetLotteryDraws get complete list of draws of the lottery
func GetLotteryDraws(domain string, index string, downloaderFunc Downloader) []Draw {
	var draws []Draw
	res, err := http.PostForm(domain+"/lottery/detailsofdrawweb.php",
		url.Values{"lotterydet": {index}})
	if err != nil {
		// handle error
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	doc.Find("table table table tr").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		columns := s.Children()
		if columns.First().Text() != "LOTTERY" {
			name := columns.Get(1).FirstChild.Data
			url, _ := columns.Find("a").Attr("href")
			draw := Draw{name, url}
			draws = append(draws, draw)
			// If downloader function is provided, start download
			if downloaderFunc != nil {
				downloaderFunc(draw)
			}
		}
	})
	return draws
}

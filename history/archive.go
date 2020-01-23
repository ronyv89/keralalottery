package history

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

type Lottery struct {
	Name  string
	Index string
}

type Draw struct {
	Name string
	URL  string
}

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
func GetLotteryDraws(domain string, index string) []Draw {
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
			draws = append(draws, Draw{name, url})
		}
	})
	return draws
}

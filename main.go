package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gocolly/colly"
)

func main() {
	c := colly.NewCollector()
	c.OnHTML("div.contentpane iframe[src]", func(e *colly.HTMLElement) {
		findResults(e.Attr("src"))
	})
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})
	c.Visit("http://keralalotteries.in/index.php/quick-view/result")
}

func findResults(url string) {
	d := colly.NewCollector()
	count := 1
	host := ""
	d.OnHTML("#form1 table tbody table tr", func(f *colly.HTMLElement) {
		if count != 1 {
			elCount := 1
			var name, date, filePtr string
			f.ForEach("td", func(_ int, el *colly.HTMLElement) {
				switch elCount {
				case 1:
					name = el.Text
				case 2:
					date = el.Text
				case 3:
					filePtr = extractFilePtr(el.ChildAttr("a", "href"))
				}
				elCount++
			})
			if filePtr != "" {
				fileURL := host + "tmp" + filePtr + ".pdf"
				dir, _ := os.Getwd()
				date = strings.ReplaceAll(date, "/", "-")
				if err := downloadFile(dir+"/"+date+".pdf", fileURL); err != nil {
					panic(err)
				}
				fmt.Println(name, date, filePtr)
			}

		}
		// fmt.Println(count, f)
		count++
	})
	d.OnRequest(func(r *colly.Request) {
		host = r.URL.Scheme + "://" + r.URL.Hostname()
		if r.URL.Port() != "" {
			host += ":" + r.URL.Port()
		}
		host += "/lottery/reports/draw/"
		fmt.Println("Visiting", r.URL.String())
	})
	d.Visit(url)
}

func extractFilePtr(jsScript string) string {
	re := regexp.MustCompile(`\d+`)
	return re.FindString(jsScript)
}

func downloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

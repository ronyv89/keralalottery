package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
	"github.com/ledongthuc/pdf"
)

type ConsolationPrize struct {
	PrizeAmount string
	Winners     []string
}

type Prize struct {
	PrizeAmount string
	Winners     []string
	Consolation ConsolationPrize
}

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
			var fullName, code, name, date, filePtr string
			f.ForEach("td", func(_ int, el *colly.HTMLElement) {
				switch elCount {
				case 1:
					fullName = el.Text
					re := regexp.MustCompile(`^(.+) \((.+)\)$`)
					matched := re.FindStringSubmatch(fullName)
					if len(matched) == 3 {
						name = matched[1]
						code = matched[2]
					}

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
				filepath := dir + "/" + name + " " + code + " " + date + ".pdf"
				if err := downloadFile(filepath, fileURL); err != nil {
					panic(err)
				}
				prizes, err := readPdf(filepath)
				if err != nil {
					panic(err)
				}
				fmt.Println(name, prizes)
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

func readPdf(path string) ([]Prize, error) {
	f, r, err := pdf.Open(path)
	defer func() {
		_ = f.Close()
	}()
	if err != nil {
		return nil, err
	}
	totalPage := r.NumPage()
	prizeCount := 1
	prizeStarted := false
	consolationStarted := false
	prizeStopped := true
	var prizes []Prize
	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}
		rows, _ := p.GetTextByRow()
		for _, row := range rows {
			for _, word := range row.Content {
				trimmed := strings.TrimSpace(word.S)
				if trimmed != "" {
					if !prizeStarted {
						re := regexp.MustCompile(prizeString(prizeCount) + ` Prize- (.+)`)
						match := re.FindStringSubmatch(trimmed)
						// First prize
						if len(match) != 0 {
							prizeStarted = true
							prizeStopped = false
							var prize Prize
							prize.PrizeAmount = match[2]
							prizes = append(prizes, prize)
						}
						// fmt.Println(prizeCount, match)
					} else if !prizeStopped {
						re1 := regexp.MustCompile(`^(\w\w \d+)$`)
						matched1 := re1.FindStringSubmatch(trimmed)
						if len(matched1) == 0 {
							re2 := regexp.MustCompile(`^(\d+)$`)
							matched2 := re2.FindStringSubmatch(trimmed)
							if len(matched2) == 0 {
								re3 := regexp.MustCompile(`^\(\w+\)$`)
								matched3 := re3.FindStringSubmatch(trimmed)
								if len(matched3) == 0 {
									consolationStarted = false
									re := regexp.MustCompile(prizeString(prizeCount+1) + ` Prize- (.+)`)
									match := re.FindStringSubmatch(trimmed)
									if len(match) == 0 {
										return prizes, nil
									}

									if match[1] == "Consolation" {
										consolationStarted = true
									} else {
										prizeCount++
										var prize Prize
										prize.PrizeAmount = match[2]
										prizes = append(prizes, prize)
									}
									prizeStarted = true
									prizeStopped = false
								}
								// re2 := regexp.MustCompile(`^(\d+)$`)
							} else {
								if consolationStarted {
									prizes[len(prizes)-1].Consolation.Winners = append(prizes[len(prizes)-1].Consolation.Winners, matched2[0])
								} else {
									prizes[len(prizes)-1].Winners = append(prizes[len(prizes)-1].Winners, matched2[0])
								}
							}
						} else {
							if consolationStarted {
								prizes[len(prizes)-1].Consolation.Winners = append(prizes[len(prizes)-1].Consolation.Winners, matched1[0])
							} else {
								prizes[len(prizes)-1].Winners = append(prizes[len(prizes)-1].Winners, matched1[0])
							}
						}

					}
				}

			}
		}
	}
	return prizes, nil
}

func addPrizeWinner(prices []Prize) {

}
func prizeString(prizeCount int) string {
	var postfix string
	prizeCountString := strconv.Itoa(prizeCount)
	switch prizeCountString[len(prizeCountString)-1:] {
	case "1":
		postfix = "st"
	case "2":
		postfix = "nd"
	case "3":
		postfix = "rd"
	default:
		postfix = "th"
	}
	return `(` + prizeCountString + postfix + `|Consolation)`
}
func readPlainTextFromPDF(pdfpath string) (text string, err error) {
	f, r, err := pdf.Open(pdfpath)
	defer f.Close()
	if err != nil {
		return
	}

	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return
	}

	buf.ReadFrom(b)
	text = buf.String()
	return
}

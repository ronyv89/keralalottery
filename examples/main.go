package main

import (
	"bytes"
	"fmt"
	"io"
	"keralalottery"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gocolly/colly"
	"github.com/ledongthuc/pdf"
	"golang.org/x/net/context"
)

func main() {
	// dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(dir)
	// // opt := option.WithCredentialsFile("../serviceAccountKey.json")
	// parsed, err := keralalottery.ParseLocalPDF("/home/rony/go/src/github.com/ronyv89/keralalottery/03-01-2020.pdf")
	// fmt.Println(parsed)
	// ctx := context.Background()
	// app, err := firebase.NewApp(ctx, nil, opt)
	// if err != nil {
	// 	fmt.Errorf("error initializing app: %v", err)
	// } else {
	// 	client, err := app.Firestore(ctx)
	// 	if err != nil {
	// 		log.Fatalln(err)
	// 	}
	// 	defer client.Close()
	// 	c := colly.NewCollector()
	// 	c.OnHTML("div.contentpane iframe[src]", func(e *colly.HTMLElement) {
	// 		findResults(e.Attr("src"), client, ctx)
	// 	})
	// 	c.OnRequest(func(r *colly.Request) {
	// 		fmt.Println("Visiting", r.URL.String())
	// 	})
	// 	c.Visit("http://keralalotteries.in/index.php/quick-view/result")
	// }

}

func findResults(url string, client *firestore.Client, ctx context.Context) {
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
				_, err := client.Collection("lotteries").Doc(name).Set(ctx, map[string]interface{}{
					"name": name,
				})
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
				_, nerr := client.Collection("results").Doc(code).Set(ctx, map[string]interface{}{
					"draw":   code,
					"date":   date,
					"prizes": prizes,
				})
				if nerr != nil {
					fmt.Println("Error storing results")
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

func readPdf(path string) ([]keralalottery.Prize, error) {
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
	var prizes []keralalottery.Prize
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
						re := regexp.MustCompile(prizeString(prizeCount) + ` keralalottery.Prize- (.+)`)
						match := re.FindStringSubmatch(trimmed)
						// First prize
						if len(match) != 0 {
							prizeStarted = true
							prizeStopped = false
							var prize keralalottery.Prize
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
									re := regexp.MustCompile(prizeString(prizeCount+1) + ` keralalottery.Prize- (.+)`)
									match := re.FindStringSubmatch(trimmed)
									if len(match) == 0 {
										if trimmed != "FOR THE TICKETS ENDING WITH THE FOLLOWING NUMBERS" {
											return prizes, nil
										}
									} else {
										if match[1] == "Consolation" {
											consolationStarted = true
											prizes[len(prizes)-1].ConsolationPresent = true
											prizes[len(prizes)-1].Consolation.PrizeAmount = match[2]
										} else {
											prizeCount++
											var prize keralalottery.Prize
											prize.PrizeAmount = match[2]
											prizes = append(prizes, prize)
										}
										prizeStarted = true
										prizeStopped = false
									}

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

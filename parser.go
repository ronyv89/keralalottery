package keralalottery

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
)

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

// ParseLocalPDF extracts lottery results from a local pdf file
func ParseLocalPDF(path string) ([]Prize, error) {
	file, reader, err := pdf.Open(path)
	defer func() {
		_ = file.Close()
	}()
	if err != nil {
		return nil, err
	}
	return ParsePDFContents(reader)
}

// ParsePDFContents extracts lottery results from PDF contents
func ParsePDFContents(reader *pdf.Reader) ([]Prize, error) {
	totalPage := reader.NumPage()
	prizeCount := 1
	prizeStarted := false
	consolationStarted := false
	prizeStopped := true
	var prizes []Prize
	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		p := reader.Page(pageIndex)
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
											var prize Prize
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

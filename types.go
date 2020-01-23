package keralalottery

type ConsolationPrize struct {
	PrizeAmount string
	Winners     []string
}

type Prize struct {
	PrizeAmount        string
	Winners            []string
	ConsolationPresent bool
	Consolation        ConsolationPrize
}

type Lottery struct {
	Name  string
	Index string
}

type Draw struct {
	Name string
	URL  string
}

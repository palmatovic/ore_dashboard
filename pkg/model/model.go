package model

type Obj struct {
	MinersCoal       []Miner          `json:"miners_coal"`
	MinersOre        []Miner          `json:"miners_ore"`
	MinerCoalSummary MinerCoalSummary `json:"miner_coal_summary"`
	MinerOreSummary  MinerOreSummary  `json:"miner_ore_summary"`
	WalletSummary    WalletSummary    `json:"wallet_summary"`
	Wallets          []Wallet         `json:"wallets"`
	Errors           []string         `json:"errors"`
}

type WalletSummary struct {
	Quantity int    `json:"quantity"`
	Best     string `json:"best"`
	Value    string `json:"value"`
}

type MinerCoalSummary struct {
	CoalPrice string `json:"coal_price"`
	Unclaimed string `json:"unclaimed"`
	Value     string `json:"value"`
}
type MinerOreSummary struct {
	OrePrice  string `json:"ore_price"`
	Unclaimed string `json:"unclaimed"`
	Value     string `json:"value"`
}

type Miner struct {
	Miner     string `json:"miner"`
	Unclaimed string `json:"unclaimed"`
	Value     string `json:"value"`
}

type Token struct {
	Name    string  `json:"name"`
	Address *string `json:"address,omitempty"`
	Balance string  `json:"balance"`
	Value   string  `json:"value"`
	Price   string  `json:"price"`
}

type Wallet struct {
	WalletId string  `json:"wallet"`
	Address  string  `json:"address"`
	Value    string  `json:"value"`
	Tokens   []Token `json:"tokens"`
}

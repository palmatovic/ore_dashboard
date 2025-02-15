package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type CLI int64

const (
	Coal CLI = iota
	Ore
)

type UnclaimedData struct {
	coalCli string
	oreCli  string
}

func NewUnclaimedData(coalCli string, oreCli string) *UnclaimedData {
	return &UnclaimedData{
		coalCli: coalCli,
		oreCli:  oreCli,
	}
}

func (u *UnclaimedData) Get(c CLI, keypair string) (float64, error) {
	cli := u.coalCli
	if c == Ore {
		cli = u.oreCli
	}
	cmd := exec.Command(cli, "balance", "--keypair", keypair)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return 0, err
	}
	var err error
	var unclaimed float64
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if parts[0] == "Stake:" {
			valueStr := parts[1]
			var value float64
			value, err = strconv.ParseFloat(valueStr, 64)
			if err != nil {
				return 0, err
			}
			unclaimed = value
			break
		}
	}
	return unclaimed, nil
}

type TokenPrice struct {
	jupApiUrl string
}

func NewTokensPrice(jupApiUrl string) *TokenPrice {
	return &TokenPrice{
		jupApiUrl: jupApiUrl,
	}
}

type Data struct {
	ID            string  `json:"id"`
	MintSymbol    string  `json:"mintSymbol"`
	VsToken       string  `json:"vsToken"`
	VsTokenSymbol string  `json:"vsTokenSymbol"`
	Price         float64 `json:"price"`
}

type Response struct {
	Data      map[string]Data `json:"data"`
	TimeTaken float64         `json:"timeTaken"`
}

func (o *TokenPrice) Get(tokenAddress string) (tokenData Data, err error) {
	var resp *http.Response
	resp, err = http.Get(o.jupApiUrl + tokenAddress)
	if err != nil {
		return Data{}, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return Data{}, fmt.Errorf("got %d http status from Jupiter for %s", resp.StatusCode, tokenAddress)
	}
	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return Data{}, err
	}
	var response Response
	if err = json.Unmarshal(body, &response); err != nil {
		return Data{}, err
	}
	if _, ok := response.Data[tokenAddress]; ok {
		return response.Data[tokenAddress], nil
	} else {
		return Data{}, err
	}
}

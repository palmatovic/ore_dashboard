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

type UnclaimedData struct {
	oreCli string
}

func NewOreUnclaimedData(oreCli string) *UnclaimedData {
	return &UnclaimedData{
		oreCli: oreCli,
	}
}

func (u *UnclaimedData) Get(keypair string) (float64, error) {
	cmd := exec.Command(u.oreCli, "rewards", "--keypair", keypair)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return 0, err
	}
	unclaimed, err := strconv.ParseFloat(strings.TrimSpace(strings.Split(stdout.String(), " ")[0]), 64)
	if err != nil {
		return 0, err
	}
	return unclaimed, nil
}

type TokenPrice struct {
	jupOreApiUrl string
}

func NewTokensPrice(jupOreApiUrl string) *TokenPrice {
	return &TokenPrice{
		jupOreApiUrl: jupOreApiUrl,
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
	resp, err = http.Get(o.jupOreApiUrl + tokenAddress)
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

package service

import (
	"Ore/pkg/model"
	"Ore/pkg/util"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/blocto/solana-go-sdk/types"
	"github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Service struct {
	unclaimedData     *util.UnclaimedData
	tokenPrice        *util.TokenPrice
	keyPairFolderPath string
	rpcUrl            string
	tokensToSearch    map[string]string
	solanaCli         string
}

func NewService(unclaimedData *util.UnclaimedData, orePrice *util.TokenPrice, keyPairFolderPath string, rpcUrl string, tokensToSearch map[string]string, solanaCli string) *Service {
	return &Service{unclaimedData, orePrice, keyPairFolderPath, rpcUrl, tokensToSearch, solanaCli}
}

func (s *Service) GenerateData() *model.Obj {
	var dataOre, dataOrz []model.Miner
	var errs []string
	var totalUnclaimedOre, totalUnclaimedOrz float64
	var wallets []model.Wallet

	var oreData util.Data
	var solData util.Data
	var orzData util.Data
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		var err error
		oreData, err = s.tokenPrice.Get(s.tokensToSearch["ORE"])
		if err != nil {
			errs = append(errs, fmt.Sprintf("cannot get ORE token prices: %s", err.Error()))
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		solData, err = s.tokenPrice.Get(s.tokensToSearch["SOL"])
		if err != nil {
			errs = append(errs, fmt.Sprintf("cannot get SOL token prices: %s", err.Error()))
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		orzData, err = s.tokenPrice.Get(s.tokensToSearch["ORZ"])
		if err != nil {
			errs = append(errs, fmt.Sprintf("cannot get SOL token prices: %s", err.Error()))
		}
	}()

	wg.Wait()

	pattern := regexp.MustCompile(`^id_(\d+)\.json$`)

	files, err := os.ReadDir(s.keyPairFolderPath)
	if err != nil {
		logrus.WithError(err).Error("cannot read directory")
		errs = append(errs, fmt.Sprintf("cannot read keypaird folder:  %s", err.Error()))
	}

	var bestWalletValue, totalWalletValue float64
	var bestWallet string
	var mux, eMux, wMux sync.Mutex

	for _, file := range files {
		matches := pattern.FindStringSubmatch(file.Name())
		if len(matches) == 2 {
			wg.Add(1)
			go func(file os.DirEntry) {
				defer wg.Done()
				i := matches[1]
				keyPairFilepath := fmt.Sprintf("%s/id_%s.json", s.keyPairFolderPath, i)
				minerId, _ := strconv.Atoi(i)

				var unclaimedOre float64
				if unclaimedOre, err = s.unclaimedData.Get(util.OreCLI, keyPairFilepath); err != nil {
					eMux.Lock()
					defer eMux.Unlock()
					errs = append(errs, fmt.Sprintf("cannot get miner unclaimed data error:  %s", err.Error()))
					return
				}
				var minerOre = model.Miner{
					Miner:     fmt.Sprintf("#%d", minerId),
					Unclaimed: fmt.Sprintf("%.6f ORE", unclaimedOre),
					Value:     fmt.Sprintf("%.2f $", unclaimedOre*oreData.Price),
				}
				totalUnclaimedOre += unclaimedOre

				var unclaimedOrz float64
				if unclaimedOrz, err = s.unclaimedData.Get(util.OrzCLI, keyPairFilepath); err != nil {
					eMux.Lock()
					defer eMux.Unlock()
					errs = append(errs, fmt.Sprintf("cannot get miner unclaimed data error:  %s", err.Error()))
					return
				}
				var minerOrz = model.Miner{
					Miner:     fmt.Sprintf("#%d", minerId),
					Unclaimed: fmt.Sprintf("%.6f ORZ", unclaimedOrz),
					Value:     fmt.Sprintf("%.2f $", unclaimedOrz*orzData.Price),
				}
				totalUnclaimedOrz += unclaimedOrz

				mux.Lock()
				dataOre = append(dataOre, minerOre)
				dataOrz = append(dataOrz, minerOrz)
				mux.Unlock()

				var walletData *model.Wallet
				var v float64
				walletData, v, err = s.getWalletData(minerId, keyPairFilepath, solData.Price)
				if err != nil {
					eMux.Lock()
					defer eMux.Unlock()
					errs = append(errs, fmt.Sprintf("cannot get wallet data error:  %s", err.Error()))
					return
				}

				totalWalletValue += v
				if v > bestWalletValue {
					bestWalletValue = v
					bestWallet = walletData.WalletId
				}
				wMux.Lock()
				wallets = append(wallets, *walletData)
				wMux.Unlock()
			}(file)
		}
	}

	wg.Wait()

	sort.Slice(dataOre, func(i, j int) bool {
		id1, _ := strconv.Atoi(dataOre[i].Miner[1:])
		id2, _ := strconv.Atoi(dataOre[j].Miner[1:])
		return id1 < id2
	})
	sort.Slice(dataOrz, func(i, j int) bool {
		id1, _ := strconv.Atoi(dataOrz[i].Miner[1:])
		id2, _ := strconv.Atoi(dataOrz[j].Miner[1:])
		return id1 < id2
	})

	sort.Slice(wallets, func(i, j int) bool {
		id1, _ := strconv.Atoi(wallets[i].WalletId[1:])
		id2, _ := strconv.Atoi(wallets[j].WalletId[1:])
		return id1 < id2
	})

	return &model.Obj{
		MinersOre: dataOre,
		MinersOrz: dataOrz,
		MinerOreSummary: model.MinerOreSummary{
			OrePrice:  fmt.Sprintf("%.2f $", oreData.Price),
			Unclaimed: fmt.Sprintf("%.6f ORE", totalUnclaimedOre),
			Value:     fmt.Sprintf("%.2f $", totalUnclaimedOre*oreData.Price),
		},
		MinerOrzSummary: model.MinerOrzSummary{
			OrzPrice:  fmt.Sprintf("%.2f $", orzData.Price),
			Unclaimed: fmt.Sprintf("%.6f ORZ", totalUnclaimedOrz),
			Value:     fmt.Sprintf("%.2f $", totalUnclaimedOrz*orzData.Price),
		},
		Errors:  errs,
		Wallets: wallets,
		WalletSummary: model.WalletSummary{
			Quantity: len(wallets),
			Best:     bestWallet,
			Value:    fmt.Sprintf("%.2f $", totalWalletValue),
		},
	}
}

func (s *Service) getWalletData(minerId int, keyPairFilepath string, solPrice float64) (*model.Wallet, float64, error) {
	fileContent, err := os.ReadFile(keyPairFilepath)
	if err != nil {
		return nil, 0, err
	}

	var byteSlice []byte
	err = json.Unmarshal(fileContent, &byteSlice)
	if err != nil {
		return nil, 0, err
	}

	account, err := types.AccountFromBytes(byteSlice)
	if err != nil {
		return nil, 0, err
	}

	cmd := exec.Command(path.Join(s.solanaCli, "spl-token"), "accounts", "--url", s.rpcUrl, "--owner", account.PublicKey.String(), "--output", "json")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return nil, 0, err
	}

	type TokenAccount struct {
		Address     string `json:"address"`
		Mint        string `json:"mint"`
		TokenAmount struct {
			UIAmount float64 `json:"uiAmount"`
		} `json:"tokenAmount"`
	}

	type TokenAccountsResponse struct {
		Accounts []TokenAccount `json:"accounts"`
	}

	var response TokenAccountsResponse
	if err = json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return nil, 0, err
	}

	var tokens []model.Token
	var value float64
	for i, acc := range response.Accounts {
		var tokenData util.Data
		tokenData, err = s.tokenPrice.Get(acc.Mint)
		if err != nil {
			logrus.WithError(err).Errorf("cannot get token price for %d", i)
			continue
		}
		tokens = append(tokens, model.Token{
			Name:    tokenData.MintSymbol,
			Address: &acc.Mint,
			Balance: fmt.Sprintf("%.4f %s", acc.TokenAmount.UIAmount, tokenData.MintSymbol),
			Value:   fmt.Sprintf("%.2f $", acc.TokenAmount.UIAmount*tokenData.Price),
			Price:   fmt.Sprintf("%.4f $", tokenData.Price),
		})
		value += acc.TokenAmount.UIAmount * tokenData.Price
	}

	// Command to execute
	cmd2 := exec.Command(path.Join(s.solanaCli, "solana"), "balance", account.PublicKey.String())
	var stdout2 bytes.Buffer
	cmd2.Stdout = &stdout2
	cmd2.Stderr = os.Stderr
	if err = cmd2.Run(); err != nil {
		return nil, 0, err
	}

	balanceStr := strings.Split(strings.TrimSpace(stdout2.String()), " ")[0]
	balance, _ := strconv.ParseFloat(balanceStr, 64)
	value += balance * solPrice
	tokens = append(tokens, model.Token{
		Name:    "SOL",
		Balance: fmt.Sprintf("%.4f SOL", balance),
		Value:   fmt.Sprintf("%.2f $", balance*solPrice),
		Price:   fmt.Sprintf("%.2f $", solPrice),
	})

	qrCode, err := qrcode.Encode(account.PublicKey.String(), qrcode.Medium, 256)
	if err != nil {
		return nil, 0, err
	}

	base64QR := base64.StdEncoding.EncodeToString(qrCode)

	return &model.Wallet{
		WalletId: fmt.Sprintf("#%d", minerId),
		Address:  base64QR,
		Tokens:   tokens,
		Value:    fmt.Sprintf("%.2f $", value),
	}, value, nil
}

package main

import (
	"Ore/pkg/service"
	"Ore/pkg/util"
	"encoding/json"
	"fmt"
	"github.com/caarlos0/env/v6"
	"github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)

type Environment struct {
	OreCli            string `env:"ORE_CLI,required"`
	KeyPairFolderPath string `env:"KEY_PAIR_FOLDER_PATH,required"`
	JupApiUrl         string `env:"JUP_API_URL,required"`
	ServerPort        int    `env:"SERVER_PORT,required"`
	RpcUrl            string `env:"RPC_URL,required"`
	SolCli            string `env:"SOLANA_CLI,required"`
}

func main() {
	var err error
	var cfg Environment
	if err = env.Parse(&cfg); err != nil {
		logrus.WithError(err).Panic("cannot configure environment variables")
	}

	currentDir, err := os.Getwd()
	if err != nil {
		logrus.WithError(err).Panic("cannot get current directory")
	}

	var tokenMap = map[string]string{
		"ORE": "oreoU2P8bN6jkk3jbaiVxYnG1dCXcYxwhwyK9jSybcp",
		"SOL": "So11111111111111111111111111111111111111112",
	}

	s := service.NewService(util.NewUnclaimedData(cfg.OreCli), util.NewTokensPrice(cfg.JupApiUrl), cfg.KeyPairFolderPath, cfg.RpcUrl, tokenMap, cfg.SolCli)

	templateDir := filepath.Join(currentDir, "pkg", "templates")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var tmpl *template.Template
		tmpl, err = template.ParseFiles(templateDir + "/index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err = tmpl.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err = json.NewEncoder(w).Encode(s.GenerateData()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	staticDir := filepath.Join(currentDir, "pkg", "templates", "static")

	fs := http.FileServer(http.Dir(staticDir))

	http.Handle("/static/", http.StripPrefix("/static/", fs))

	if err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.ServerPort), nil); err != nil {
		logrus.WithError(err).Panic("cannot init server")
	}

}

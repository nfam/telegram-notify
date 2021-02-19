package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

func main() {
	err := run(parseConfig())
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
}

func run(c config) error {
	url := "https://api.telegram.org/bot" + c.token + "/sendMessage"
	ctype := "Content-Type: application/json"
	queue := make(chan message, 100)

	mux := http.NewServeMux()
	mux.Handle("/notify", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			from := strings.TrimSpace(q.Get("from"))
			sound := strings.TrimSpace(q.Get("sound")) != "on"

			ids, ok := c.rules[from]
			if !ok {
				ids, _ = c.rules[""]
			}
			if len(ids) == 0 {
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				println("Error reading body: %v", err)
				http.Error(w, "can't read body", http.StatusBadRequest)
				return
			}
			if len(body) == 0 {
				return
			}
			text := string(body)
			mode := strings.TrimSpace(q.Get("mode"))
			if mode != "html" && mode != "markdown" {
				mode = c.mode
				if mode != "html" && mode != "markdown" {
					mode = ""
				}
			}
			if from != "" {
				switch mode {
				case "html":
					text = "<b>" + from + ":</b> " + text
				case "markdown":
					text = "*" + from + ":* " + text
				default:
					text = from + ": " + text
				}
			}
			for _, id := range ids {
				select {
				case queue <- message{
					ChatID:                id,
					Text:                  text,
					ParseMode:             mode,
					DisableWebPagePreview: true,
					DisableNotification:   sound,
				}:
				default:
					http.Error(w, "max capacity reached", 503)
					return
				}
			}
		},
	))
	server := http.Server{
		Addr:    c.listen,
		Handler: mux,
	}

	var wg sync.WaitGroup

	graceful := false
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for msg := range queue {
			data, _ := json.Marshal(&msg)
			http.Post(url, ctype, bytes.NewReader(data))
		}
		graceful = true
		println("Stop server")
	}()
	go func() {
		<-sigChan
		_ = server.Close()
		close(queue)
	}()

	println("Start listening to " + c.listen)
	err := server.ListenAndServe()
	if !graceful || err != http.ErrServerClosed {
		return err
	}
	wg.Wait()
	return nil
}

type message struct {
	ChatID                int64  `json:"chat_id,string"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
	DisableNotification   bool   `json:"disable_notification,omitempty"`
}

type config struct {
	listen string
	token  string
	mode   string
	rules  map[string][]int64
}

func parseConfig() config {
	var c config
	flag.StringVar(
		&c.listen,
		"l",
		envStr("LISTEN", ":8000"),
		"[addres]:port for the web server to listen to",
	)
	flag.StringVar(
		&c.token,
		"t",
		envStr("TOKEN", ""),
		"Telegram bot token",
	)
	flag.StringVar(
		&c.mode,
		"m",
		envStr("MODE", "text"),
		"parse_mode of telegram message (text, html, markdown)",
	)
	var rulestr string
	flag.StringVar(
		&rulestr,
		"r",
		envStr("RULE", ""),
		"Rule to forward message to users by format [{from:}{id,...};]{id,...}",
	)
	flag.Parse()

	c.rules = map[string][]int64{}
	rules := strings.Split(rulestr, ";")
	for _, rule := range rules {
		from, ids, err := parseRule(rule)
		if err != nil {
			panic(err)
		}
		c.rules[from] = appendID(c.rules[from], ids...)
	}

	return c
}

func parseRule(rule string) (string, []int64, error) {
	p := strings.Split(rule, ":")
	switch len(p) {
	case 1:
		ids, err := parseIDs(p[0])
		return "", ids, err
	case 2:
		ids, err := parseIDs(p[1])
		return strings.TrimSpace(p[0]), ids, err
	default:
		return "", nil, errors.New("invalid rule")
	}
}

func parseIDs(str string) ([]int64, error) {
	var ids []int64
	ss := strings.Split(str, ",")
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = appendID(ids, i)
	}
	return ids, nil
}

func appendID(slice []int64, ids ...int64) []int64 {
	for _, id := range ids {
		var exits bool
		for _, i := range slice {
			if i == id {
				exits = true
				break
			}
		}
		if !exits {
			slice = append(slice, id)
		}
	}
	return slice
}

func envStr(name string, value string) string {
	v := os.Getenv(name)
	if v != "" {
		return v
	}
	return value
}

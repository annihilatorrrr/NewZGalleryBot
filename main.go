package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PuerkitoBio/goquery"
	"github.com/redis/go-redis/v9"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"
)

type newsItem struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Description string `json:"description"`
}

func fetchNDTVNews() []newsItem {
	req, err := http.NewRequest("GET", "https://www.ndtv.com/latest#pfrom=home-ndtv_nav_wap", nil)
	if err != nil {
		return []newsItem{}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/png,image/svg+xml,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "")
	req.Header.Set("Referer", "https://www.ndtv.com/")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Priority", "u=0, i")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []newsItem{}
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return []newsItem{}
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return []newsItem{}
	}
	var news []newsItem
	doc.Find("div").Each(func(i int, s *goquery.Selection) {
		if class, _ := s.Attr("class"); class == "news_Itm" {
			title := s.Find("a").Text()
			link, _ := s.Find("a").Attr("href")
			description := s.Find(".newsCont").Text()
			if title != "" && link != "" {
				news = append(news, newsItem{
					Title:       title,
					Link:        link,
					Description: description,
				})
			}
		}
	})
	if len(news) == 0 {
		return []newsItem{}
	}
	return news
}

func callrestarter(slp bool) {
	if slp {
		time.Sleep(time.Second * 21600)
	}
	self, err := os.Executable()
	if err != nil {
		log.Println(err.Error())
		return
	}
	_ = syscall.Exec(self, os.Args, os.Environ())
}

func reverseNewsItems(items []newsItem) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func waitAndSend(b *gotgbot.Bot, meth string, params map[string]string, data map[string]gotgbot.FileReader) bool {
	if b.BotClient == nil {
		return false
	}
	var waitFor int64
	for {
		if waitFor != 0 {
			time.Sleep(time.Duration(waitFor) * time.Second)
		}
		if _, err := b.Request(meth, params, data, &gotgbot.RequestOpts{Timeout: time.Minute}); err != nil {
			var tgErr *gotgbot.TelegramError
			if errors.As(err, &tgErr) {
				if tgErr.Code != 429 || tgErr.ResponseParams.RetryAfter == 0 {
					break
				}
				waitFor = tgErr.ResponseParams.RetryAfter + 1
			} else {
				break
			}
		} else {
			return true
		}
	}
	return false
}

func worker(b *gotgbot.Bot, db *redis.Client, cotx context.Context) {
	for {
		time.Sleep(time.Minute)
		if data := fetchNDTVNews(); len(data) > 0 {
			if db.SIsMember(cotx, "newsold", data[0].Title).Val() {
				continue
			}
			var (
				newnews []string
				counter int
			)
			for _, x := range data {
				if db.SIsMember(cotx, "newsold", x.Title).Val() {
					break
				}
				counter++
				newnews = append(newnews, x.Title)
			}
			data = data[:counter]
			reverseNewsItems(data)
			v := map[string]string{}
			v["chat_id"] = strconv.FormatInt(-1002493739515, 10)
			v["parse_mode"] = "html"
			for _, x := range data {
				v["text"] = fmt.Sprintf("<b>Title:</b> %s\n<b>Description:</b> %s\n<b>Link:</b> %s\n\n<b>©️ @Memers_Gallery</b>", x.Title, x.Description, x.Link)
				waitAndSend(b, "sendMessage", v, nil)
			}
			if len(newnews) > 0 {
				db.Del(cotx, "newsold")
				db.SAdd(cotx, "newsold", newnews)
			}
		}
	}
}

func main() {
	token := os.Getenv("TOKEN")
	if token == "" {
		token = ""
	}
	wurl := os.Getenv("URL")
	if wurl == "" {
		log.Fatal("No webhook url was found!")
	}
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("No port was found to bind!")
	}
	b, err := gotgbot.NewBot(token, nil)
	if err != nil {
		log.Fatal(err.Error())
	}
	opt, err := redis.ParseURL(os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err.Error())
	}
	db := redis.NewClient(opt)
	if err = db.Ping(context.Background()).Err(); err != nil {
		log.Fatal(err.Error())
	}
	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(_ *gotgbot.Bot, _ *ext.Context, err error) ext.DispatcherAction {
			_, _ = b.SendMessage(1594433798, err.Error(), nil)
			return ext.DispatcherActionNoop
		},
		MaxRoutines: -1,
	})
	updater := ext.NewUpdater(dispatcher, nil)
	dispatcher.AddHandler(handlers.NewCommand("start", func(b *gotgbot.Bot, ctx *ext.Context) error {
		_, _ = ctx.EffectiveMessage.Reply(b, "I'm alive!\nBy @Annihilatorrrr", nil)
		return ext.EndGroups
	}))
	if _, err = b.SetWebhook(wurl+token, &gotgbot.SetWebhookOpts{
		MaxConnections:     40,
		DropPendingUpdates: false,
		AllowedUpdates:     []string{"message"},
		SecretToken:        "xyzzz",
	}); err != nil {
		log.Fatal(err.Error())
	}
	if err = updater.StartWebhook(b,
		token,
		ext.WebhookOpts{
			ListenAddr:  "0.0.0.0:" + port,
			SecretToken: "xyzzz",
		},
	); err != nil {
		log.Fatalf(err.Error())
	}
	go worker(b, db, context.Background())
	go callrestarter(true)
	log.Println(b.User.FirstName, "has been started!")
	updater.Idle()
	_ = db.Close()
	log.Println("Bye!")
}

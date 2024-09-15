package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/redis/go-redis/v9"
	"io"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"
)

type newsItem struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Description string `json:"description"`
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

func getJSONResponse() []newsItem {
	resp, err := http.Get("https://igpdl.vercel.app/news?key=" + os.Getenv("KEY"))
	if err != nil {
		return []newsItem{}
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []newsItem{}
	}
	var jdata []newsItem
	if err = json.Unmarshal(body, &jdata); err != nil {
		return []newsItem{}
	}
	return jdata
}

func worker(b *gotgbot.Bot, db *redis.Client, cotx context.Context) {
	for {
		time.Sleep(time.Minute)
		if data := getJSONResponse(); len(data) > 0 {
			if db.SIsMember(cotx, "newsold", data[0].Title).Val() {
				continue
			}
			reverseNewsItems(data)
			var newnews []string
			for _, x := range data {
				if db.SIsMember(cotx, "newsold", x.Title).Val() {
					break
				}
				newnews = append(newnews, x.Title)
				_, _ = b.SendMessage(-1002493739515, fmt.Sprintf("<b>Title:</b> %s\n<b>Description:</b> %s\n<b>Link:</b> %s\n\n<b>©️ @Memers_Gallery</b>", x.Title, x.Description, x.Link), &gotgbot.SendMessageOpts{ParseMode: "html"})
				time.Sleep(time.Second)
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

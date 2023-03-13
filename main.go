package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/akuzia/mattermost-redmine-bot/mattermost"
	"github.com/akuzia/mattermost-redmine-bot/redmine"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

func configInit() {
	viper.AutomaticEnv()

	if _, err := os.Stat(".env"); err == nil {
		viper.SetConfigFile(".env")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatal(fmt.Errorf("error in config file: %w", err))
		}
	}
}

func init() {
	configInit()
}

func main() {
	redmine := redmine.New(
		viper.GetString("redmine_url"),
		viper.GetString("redmine_api_key"),
		cast.ToIntSlice(strings.Split(viper.GetString("redmine_closed_statuses"), ",")),
		cast.ToIntSlice(strings.Split(viper.GetString("redmine_high_priorities"), ",")),
	)

	viper.SetDefault("mattermost_channel_join_minutes", 15)

	baseUrl, err := url.Parse(viper.GetString("mattermost_url"))
	if err != nil {
		log.Fatal(err.Error())
	}

	mm := mattermost.New(
		baseUrl,
		viper.GetString("mattermost_token"),
		redmine,
	)

	watchSignals(mm)

	mm.Listen()
}

func watchSignals(mm *mattermost.Client) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)
	signal.Notify(signalChan, syscall.SIGTERM)

	duration := viper.GetInt64("mattermost_channel_join_minutes")
	ticker := time.NewTicker(time.Duration(duration) * time.Minute)

	go func() {
	out:
		for {
			select {
			case <-signalChan:
				mm.Close()
				break out
			case <-ticker.C:
				mm.JoinChannels()
			}
		}
	}()
}

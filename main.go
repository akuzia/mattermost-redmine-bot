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

	"github.com/akuzia/mattermost-redmine-bot/logger"
	"github.com/akuzia/mattermost-redmine-bot/mattermost"
	"github.com/akuzia/mattermost-redmine-bot/redmine"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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
	logger, err := logger.New(viper.GetViper())
	if err != nil {
		log.Fatal(err)
	}

	redmine := redmine.New(
		viper.GetString("redmine_url"),
		viper.GetString("redmine_api_key"),
		cast.ToIntSlice(strings.Split(viper.GetString("redmine_closed_statuses"), ",")),
		cast.ToIntSlice(strings.Split(viper.GetString("redmine_high_priorities"), ",")),
	)

	viper.SetDefault("mattermost_channel_join_minutes", 15)

	baseUrl, err := url.Parse(viper.GetString("mattermost_url"))
	if err != nil {
		logger.Fatal(
			"cannot get mattermost url",
			zap.String("url", viper.GetString("mattermost_url")),
		)
	}

	mm, err := mattermost.New(
		baseUrl,
		viper.GetString("mattermost_token"),
		redmine,
		logger,
	)
	if err != nil {
		logger.Fatal(
			"cannot create mattermos client",
			zap.Error(err),
		)
	}

	watchSignals(mm, logger)

	mm.Listen()
}

func watchSignals(
	mm *mattermost.Client,
	logger *zap.Logger,
) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)
	signal.Notify(signalChan, syscall.SIGTERM)

	duration := viper.GetInt64("mattermost_channel_join_minutes")
	ticker := time.NewTicker(time.Duration(duration) * time.Minute)

	go func() {
	out:
		for {
			select {
			case s := <-signalChan:
				logger.Info(
					"recieved signal",
					zap.String("signal", s.String()),
				)
				mm.Close()
				break out
			case <-ticker.C:
				mm.JoinChannels()
			}
		}
	}()
}

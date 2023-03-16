package mattermost

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/akuzia/mattermost-redmine-bot/redmine"
	"github.com/mattermost/mattermost-server/v5/model"
)

const (
	botLogo      = "http://download.retailcrm.pro/redmine_fluid_icon.png"
	issuePattern = "(#|%s\\/issues\\/)(\\d+)"
)

type Client struct {
	url             *url.URL
	token           string
	client          *model.Client4
	websocketClient *model.WebSocketClient
	redmine         *redmine.Client
	pattern         *regexp.Regexp
	user            model.User
}

func New(
	baseUrl *url.URL,
	token string,
	redmineClient *redmine.Client,
) *Client {
	client := model.NewAPIv4Client(baseUrl.String())
	client.SetToken(token)

	user, _ := client.GetMe("")

	wsUrl := *baseUrl
	wsUrl.Scheme = "ws"
	wsUrl.Host = strings.Join([]string{wsUrl.Hostname(), "8065"}, ":")

	webSocketClient, err := model.NewWebSocketClient4(wsUrl.String(), token)
	if err != nil {
		log.Fatal(err.Error())
	}

	return &Client{
		baseUrl,
		token,
		client,
		webSocketClient,
		redmineClient,
		regexp.MustCompile(fmt.Sprintf(issuePattern, regexp.QuoteMeta(redmineClient.Url))),
		*user,
	}
}

func (s *Client) sendMessage(issue *redmine.Issue, channel string, rootId string) (err error) {
	var preText string
	if issue.Tracker != nil {
		preText = fmt.Sprintf("%s #%d: %s", issue.Tracker.Name, issue.Id, issue.Subject)
	} else {
		preText = fmt.Sprintf("#%d: %s", issue.Id, issue.Subject)
	}
	preText = fmt.Sprintf("[%s](%s)\n", preText, s.redmine.GetIssueUrl(issue))

	var text []string

	text = append(text, fmt.Sprintf("**Project**: %s", issue.Project.Name))
	text = append(text, fmt.Sprintf("**Status**: %s", issue.Status.Name))

	if issue.Category != nil {
		text = append(text, fmt.Sprintf("**Category**: %s", issue.Category.Name))
	}
	if issue.Version != nil {
		text = append(text, fmt.Sprintf("**Version**: %s", issue.Version.Name))
	}
	if issue.AssignedTo != nil {
		text = append(text, fmt.Sprintf("**Assigned to**: %s", issue.AssignedTo.Name))
	}
	if s.redmine.IssueInHighPriority(issue) {
		text = append(text, fmt.Sprintf("**Priority**: %s", issue.Priority.Name))
	}

	s.client.CreatePost(&model.Post{
		ChannelId: channel,
		RootId:    rootId,
		Props: map[string]interface{}{
			"attachments": []map[string]interface{}{
				{
					"color": "#C0C0C0",
					"text":  preText + strings.Join(text, "\t"),
				},
			},
		},
	})

	return err
}

func (s *Client) processEvent(event *model.WebSocketEvent) {
	processed := make(map[string]bool)

	post := model.PostFromJson(strings.NewReader(event.GetData()["post"].(string)))
	if post == nil {
		log.Printf("cannot decode post: %v", event.GetData())

		return
	}

	matches := s.pattern.FindAllStringSubmatch(post.Message, -1)

	for _, v := range matches {
		issueNumber := v[2]

		if _, ok := processed[issueNumber]; ok {
			continue
		}

		issue, err := s.redmine.GetIssue(issueNumber)
		processed[issueNumber] = true

		if err != nil {
			fmt.Printf("cannot fetch issue: %s", err.Error())
			continue
		}

		err = s.sendMessage(issue, post.ChannelId, post.RootId)

		if err != nil {
			fmt.Printf("cannot send message: %s", err.Error())
		}
	}
}

func (s *Client) Listen() {
	s.websocketClient.Listen()

	log.Println("listener started")
	for event := range s.websocketClient.EventChannel {
		if event.EventType() != model.WEBSOCKET_EVENT_POSTED {
			continue
		}

		s.processEvent(event)
	}
	log.Println("listener stopped")
}

func (s *Client) Close() {
	s.websocketClient.Close()
}

func (s *Client) JoinChannels() {
	log.Println("joining available channels")
	teams, _ := s.client.GetTeamsForUser(s.user.Id, "")
	for _, t := range teams {
		channels, _ := s.client.GetPublicChannelsForTeam(t.Id, 0, 100, "")
		for _, c := range channels {
			s.client.AddChannelMember(c.Id, s.user.Id)
		}
	}
}

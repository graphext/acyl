package slacknotifier

import (
	"fmt"

	"github.com/nlopes/slack"
)

const (
	slackUsername = "Dynamic QA Notifier"
	slackIconURL  = "http://lorempixel.com/48/48"
)

// ChatNotifier destribes an object capable of pushing notifications to a chat service
type ChatNotifier interface {
	Creating(name string, rd *RepoRevisionData) error
	Destroying(name string, rd *RepoRevisionData, reason QADestroyReason) error
	Updating(name string, rd *RepoRevisionData, commitmsg string) error
	Success(name string, hostname string, svcports map[string]int64, k8sNamespace string, rd *RepoRevisionData) error
	Failure(name string, rd *RepoRevisionData, msg string) error
	LockError(rd *RepoRevisionData, msg string) error
}

// SlackNotifier is an object that pushes notifications to Slack
type SlackNotifier struct {
	channel string
	api     SlackAPIClient
	params  *slack.PostMessageParameters
	mapper  SlackUsernameMapper
}

// SlackAPIClient describes the methods we use on slack.Client
type SlackAPIClient interface {
	PostMessage(channel, text string, params slack.PostMessageParameters) (string, string, error)
	GetUsers() ([]*slack.User, error)
}

// NewSlackNotifier returns a SlackNotifier using the provided API token
func NewSlackNotifier(channel string, api SlackAPIClient, usernameMapper SlackUsernameMapper) *SlackNotifier {
	params := slack.NewPostMessageParameters()
	params.Username = slackUsername
	params.IconURL = slackIconURL
	return &SlackNotifier{
		channel: channel,
		api:     api,
		params:  &params,
		mapper:  usernameMapper,
	}
}

// Creating sends a Slack notification that a new QA is being created
func (sn *SlackNotifier) Creating(name string, rd *RepoRevisionData) error {
	p := *sn.params
	p.Text = ":wrench: Creating QA"
	p.Attachments = []slack.Attachment{
		slack.Attachment{
			Color:      "good",
			Text:       fmt.Sprintf("%v\nPR #%v: %v :arrow_right: %v", rd.Repo, rd.PullRequest, rd.SourceBranch, rd.BaseBranch),
			Title:      name,
			MarkdownIn: []string{"text", "pretext"},
		},
	}

	return sn.postMessageToUserAndNotificationChannel(rd, p)
}

// Destroying sends a Slack notification that an existing QA is being explicitly destroyed
func (sn *SlackNotifier) Destroying(name string, rd *RepoRevisionData, reason QADestroyReason) error {
	p := *sn.params
	p.Text = ":bomb: Destroying QA"
	p.Attachments = []slack.Attachment{
		slack.Attachment{
			Color:      "warning",
			Text:       fmt.Sprintf("%v\nPR #%v: %v :arrow_right: %v (%s)", rd.Repo, rd.PullRequest, rd.SourceBranch, rd.BaseBranch, reason),
			Title:      name,
			MarkdownIn: []string{"text", "pretext"},
		},
	}

	return sn.postMessageToUserAndNotificationChannel(rd, p)
}

// Updating sends a Slack notification that an existing environment is getting replaced by one with new(er) code
func (sn *SlackNotifier) Updating(name string, rd *RepoRevisionData, commitmsg string) error {
	commiturl := fmt.Sprintf("https://github.com/%v/commit/%v", rd.Repo, rd.SourceSHA)
	p := *sn.params
	p.Text = ":vertical_traffic_light: Updating QA"
	p.Attachments = []slack.Attachment{
		slack.Attachment{
			Color:      "warning",
			Title:      name,
			Text:       fmt.Sprintf("%v\nPR #%v: %v :arrow_right: %v\nUpdating to commit:\n%v\n\"%v\" - _%v_", rd.Repo, rd.PullRequest, rd.SourceBranch, rd.BaseBranch, commiturl, commitmsg, rd.User),
			MarkdownIn: []string{"text", "pretext"},
		},
	}

	return sn.postMessageToUserAndNotificationChannel(rd, p)
}

// Success sends a Slack notification that an existing QA is up and ready
func (sn *SlackNotifier) Success(name string, hostname string, svcports map[string]int64, k8sNamespace string, rd *RepoRevisionData) error {
	p := *sn.params
	p.Text = ":checkered_flag: QA Success"
	p.Attachments = []slack.Attachment{
		slack.Attachment{
			Color:      "good",
			Text:       fmt.Sprintf("%v\nPR #%v: %v :arrow_right: %v", rd.Repo, rd.PullRequest, rd.SourceBranch, rd.BaseBranch),
			Title:      name,
			MarkdownIn: []string{"text", "pretext"},
		},
		slack.Attachment{
			Color: "good",
			Text:  fmt.Sprintf("%v (%v)", hostname, k8sNamespace),
		},
		slack.Attachment{
			Color: "good",
			Text:  fmt.Sprintf("Service Ports: %v", svcports),
		},
	}

	return sn.postMessageToUserAndNotificationChannel(rd, p)
}

// LockError sends a Slack notification that an action for a repo/PR combination couldn't be performed because the lock couldn't be acquired
func (sn *SlackNotifier) LockError(rd *RepoRevisionData, msg string) error {
	p := *sn.params
	p.Text = ":lock: Lock Error"
	p.Attachments = []slack.Attachment{
		slack.Attachment{
			Color:      "danger",
			Text:       fmt.Sprintf("%v\nPR #%v: %v :arrow_right: %v", rd.Repo, rd.PullRequest, rd.SourceBranch, rd.BaseBranch),
			MarkdownIn: []string{"text", "pretext"},
		},
		slack.Attachment{
			Color: "danger",
			Text:  msg,
		},
	}

	return sn.postMessageToUserAndNotificationChannel(rd, p)
}

// Failure sends a Slack notification that a QA operation failed
func (sn *SlackNotifier) Failure(name string, rd *RepoRevisionData, msg string) error {
	p := *sn.params
	p.Text = "(╯°□°）╯︵ ┻━┻    QA Failed"
	p.Attachments = []slack.Attachment{
		slack.Attachment{
			Color:      "danger",
			Text:       fmt.Sprintf("%v\nPR #%v: %v :arrow_right: %v", rd.Repo, rd.PullRequest, rd.SourceBranch, rd.BaseBranch),
			Title:      name,
			MarkdownIn: []string{"text", "pretext"},
		},
		slack.Attachment{
			Color: "danger",
			Text:  msg,
		},
	}

	return sn.postMessageToUserAndNotificationChannel(rd, p)
}

func (sn *SlackNotifier) postMessageToNotificationChannel(p slack.PostMessageParameters) error {
	_, _, err := sn.api.PostMessage(sn.channel, p.Text, p)
	return err
}

func (sn *SlackNotifier) postMessageToUserAndNotificationChannel(rd *RepoRevisionData, p slack.PostMessageParameters) error {
	errs := []error{}
	channels := []string{sn.channel}
	userChannel, chanErr := sn.userChannelForGithubUsername(rd.User)
	if chanErr != nil {
		errs = append(errs, chanErr)
	}

	if len(userChannel) > 0 {
		channels = append(channels, userChannel)
	}

	postErr := sn.fanoutPostMessage(channels, p)
	if postErr != nil {
		errs = append(errs, postErr)
	}

	var err error
	if len(errs) > 0 {
		err = fmt.Errorf("postMessageToUserAndNotificationChannel errors: %v", errs)
	}

	return err
}

func (sn *SlackNotifier) userChannelForGithubUsername(githubUsername string) (string, error) {
	un, err := sn.mapper.UsernameFromGithubUsername(githubUsername)
	if err != nil {
		return "", err
	}

	if len(un) == 0 {
		return "", nil
	}

	channel, uerr := sn.userIDfromUsername(un)
	if uerr != nil {
		return "", err
	}
	if len(channel) == 0 {
		return "", nil
	}

	return channel, nil
}

func (sn *SlackNotifier) userIDfromUsername(slackUsername string) (string, error) {
	users, err := sn.api.GetUsers()
	if err != nil {
		return "", err
	}

	if len(users) == 0 {
		return "", nil
	}

	var userid string
	for _, v := range users {
		if v.Name == slackUsername {
			userid = v.ID
		}
	}
	return userid, nil
}

func (sn *SlackNotifier) fanoutPostMessage(channels []string, p slack.PostMessageParameters) error {
	errs := []error{}
	for _, c := range channels {
		_, _, err := sn.api.PostMessage(c, p.Text, p)
		if err != nil {
			errs = append(errs, err)
		}
	}

	var err error
	if len(errs) > 0 {
		err = fmt.Errorf("fanoutPostMessage errors: %v", errs)
	}

	return err
}

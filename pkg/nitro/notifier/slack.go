package notifier

import (
	multierror "github.com/hashicorp/go-multierror"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

// SlackAPIClient describes the methods we use on slack.Client
type SlackAPIClient interface {
	PostMessage(channel, text string, params slack.PostMessageParameters) (string, string, error)
}

// SlackBackend is an object that uses API to send notifications to Users and Channels
type SlackBackend struct {
	Username, IconURL string
	API               SlackAPIClient
	Users, Channels   []string
}

var _ Backend = &SlackBackend{}

// Send broadcasts n to all configured users/channels
func (sb *SlackBackend) Send(n Notification) error {
	p, err := sb.render(n)
	if err != nil || p == nil {
		return errors.Wrap(err, "error rendering notification")
	}
	p.Username = sb.Username
	p.IconURL = sb.IconURL
	var merr *multierror.Error
	for _, c := range sb.Channels {
		if len(c) > 0 {
			_, _, err = sb.API.PostMessage(c, p.Text, *p)
			if err != nil {
				merr = multierror.Append(merr, err)
			}
		}
	}
	for _, u := range sb.Users {
		if len(u) > 0 {
			_, _, err = sb.API.PostMessage("@"+u, p.Text, *p)
			if err != nil {
				merr = multierror.Append(merr, err)
			}
		}
	}
	return merr.ErrorOrNil()
}

func (sb *SlackBackend) render(n Notification) (*slack.PostMessageParameters, error) {
	rn, err := n.Template.Render(n.Data)
	if err != nil {
		return nil, errors.Wrap(err, "error rendering template")
	}
	out := &slack.PostMessageParameters{Text: rn.Title, Attachments: make([]slack.Attachment, len(rn.Sections))}
	for i, s := range rn.Sections {
		if s.Title == "" {
			s.Title = "<empty>"
		}
		if s.Text == "" {
			s.Text = "<empty>"
		}
		out.Attachments[i].Title = s.Title
		out.Attachments[i].Text = s.Text
		out.Attachments[i].Color = s.Style
	}
	return out, nil
}

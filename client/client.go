// Package client provides a ntfy client to publish and subscribe to topics.
package client

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"heckel.io/ntfy/v2/log"
	"heckel.io/ntfy/v2/util"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	// MessageEvent identifies a message event in the JSON stream.
	MessageEvent = "message"
)

const (
	maxResponseBytes = 4096
)

var (
	topicRegex = regexp.MustCompile(`^[-_A-Za-z0-9]{1,64}$`) // Same as in server/server.go
)

// Client is the ntfy client that can be used to publish and subscribe to ntfy topics.
type Client struct {
	// Messages is a channel that receives new messages for subscribed topics.
	Messages      chan *Message
	config        *Config
	subscriptions map[string]*subscription
	mu            sync.Mutex
}

// Message represents a ntfy message.
type Message struct { // TODO combine with server.message
	// ID is the unique identifier of the message.
	ID         string
	// Event is the type of event (e.g., "message", "open", "keepalive").
	Event      string
	// Time is the timestamp of the message.
	Time       int64
	// Topic is the topic name.
	Topic      string
	// Message is the message body.
	Message    string
	// Title is the title of the message.
	Title      string
	// Priority is the priority of the message (1-5).
	Priority   int
	// Tags is a list of tags associated with the message.
	Tags       []string
	// Click is a URL to open when the notification is clicked.
	Click      string
	// Icon is a URL to an icon to display with the notification.
	Icon       string
	// Attachment contains information about an attachment, if present.
	Attachment *Attachment

	// Additional fields
	
	// TopicURL is the full URL of the topic.
	TopicURL       string
	// SubscriptionID is the ID of the subscription that received this message.
	SubscriptionID string
	// Raw is the raw JSON representation of the message.
	Raw            string
}

// Attachment represents a message attachment.
type Attachment struct {
	// Name is the name of the attachment.
	Name    string `json:"name"`
	// Type is the MIME type of the attachment.
	Type    string `json:"type,omitempty"`
	// Size is the size of the attachment in bytes.
	Size    int64  `json:"size,omitempty"`
	// Expires is the timestamp when the attachment expires.
	Expires int64  `json:"expires,omitempty"`
	// URL is the URL to download the attachment.
	URL     string `json:"url"`
	// Owner is the IP address of uploader, used for rate limiting.
	Owner   string `json:"-"` 
}

type subscription struct {
	ID       string
	topicURL string
	cancel   context.CancelFunc
}

// New creates a new Client using a given Config.
//
// Parameters:
//   - config: The configuration object for the client.
//
// Returns:
//   - A new Client instance.
func New(config *Config) *Client {
	return &Client{
		Messages:      make(chan *Message, 50), // Allow reading a few messages
		config:        config,
		subscriptions: make(map[string]*subscription),
	}
}

// Publish sends a message to a specific topic, optionally using options.
// See PublishReader for details.
//
// Parameters:
//   - topic: The topic to publish to.
//   - message: The message content.
//   - options: Optional configuration for the publish request (e.g., title, priority).
//
// Returns:
//   - The published Message object, or an error if the request failed.
func (c *Client) Publish(topic, message string, options ...PublishOption) (*Message, error) {
	return c.PublishReader(topic, strings.NewReader(message), options...)
}

// PublishReader sends a message to a specific topic using an io.Reader as the body, optionally using options.
//
// A topic can be either a full URL (e.g. https://myhost.lan/mytopic), a short URL which is then prepended https://
// (e.g. myhost.lan -> https://myhost.lan), or a short name which is expanded using the default host in the
// config (e.g. mytopic -> https://ntfy.sh/mytopic).
//
// To pass title, priority and tags, check out WithTitle, WithPriority, WithTagsList, WithDelay, WithNoCache,
// WithNoFirebase, and the generic WithHeader.
//
// Parameters:
//   - topic: The topic to publish to.
//   - body: The message body as an io.Reader.
//   - options: Optional configuration for the publish request.
//
// Returns:
//   - The published Message object, or an error if the request failed.
func (c *Client) PublishReader(topic string, body io.Reader, options ...PublishOption) (*Message, error) {
	topicURL, err := c.expandTopicURL(topic)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", topicURL, body)
	if err != nil {
		return nil, err
	}
	for _, option := range options {
		if err := option(req); err != nil {
			return nil, err
		}
	}
	log.Debug("%s Publishing message with headers %s", util.ShortTopicURL(topicURL), req.Header)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(strings.TrimSpace(string(b)))
	}
	m, err := toMessage(string(b), topicURL, "")
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Poll queries a topic for all (or a limited set) of messages. Unlike Subscribe, this method only polls for
// messages and does not subscribe to messages that arrive after this call.
//
// A topic can be either a full URL (e.g. https://myhost.lan/mytopic), a short URL which is then prepended https://
// (e.g. myhost.lan -> https://myhost.lan), or a short name which is expanded using the default host in the
// config (e.g. mytopic -> https://ntfy.sh/mytopic).
//
// By default, all messages will be returned, but you can change this behavior using a SubscribeOption.
// See WithSince, WithSinceAll, WithSinceUnixTime, WithScheduled, and the generic WithQueryParam.
//
// Parameters:
//   - topic: The topic to poll.
//   - options: Optional configuration for the poll request.
//
// Returns:
//   - A list of messages, or an error if the request failed.
func (c *Client) Poll(topic string, options ...SubscribeOption) ([]*Message, error) {
	topicURL, err := c.expandTopicURL(topic)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	messages := make([]*Message, 0)
	msgChan := make(chan *Message)
	errChan := make(chan error)
	log.Debug("%s Polling from topic", util.ShortTopicURL(topicURL))
	options = append(options, WithPoll())
	go func() {
		err := performSubscribeRequest(ctx, msgChan, topicURL, "", options...)
		close(msgChan)
		errChan <- err
	}()
	for m := range msgChan {
		messages = append(messages, m)
	}
	return messages, <-errChan
}

// Subscribe subscribes to a topic to listen for newly incoming messages. The method starts a connection in the
// background and returns new messages via the Messages channel.
//
// A topic can be either a full URL (e.g. https://myhost.lan/mytopic), a short URL which is then prepended https://
// (e.g. myhost.lan -> https://myhost.lan), or a short name which is expanded using the default host in the
// config (e.g. mytopic -> https://ntfy.sh/mytopic).
//
// By default, only new messages will be returned, but you can change this behavior using a SubscribeOption.
// See WithSince, WithSinceAll, WithSinceUnixTime, WithScheduled, and the generic WithQueryParam.
//
// The method returns a unique subscriptionID that can be used in Unsubscribe.
//
// Parameters:
//   - topic: The topic to subscribe to.
//   - options: Optional configuration for the subscription.
//
// Returns:
//   - A subscription ID, or an error if the subscription failed.
//
// Example:
//
//	c := client.New(client.NewConfig())
//	subscriptionID, _ := c.Subscribe("mytopic")
//	for m := range c.Messages {
//	  fmt.Printf("New message: %s", m.Message)
//	}
func (c *Client) Subscribe(topic string, options ...SubscribeOption) (string, error) {
	topicURL, err := c.expandTopicURL(topic)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	subscriptionID := util.RandomString(10)
	log.Debug("%s Subscribing to topic", util.ShortTopicURL(topicURL))
	ctx, cancel := context.WithCancel(context.Background())
	c.subscriptions[subscriptionID] = &subscription{
		ID:       subscriptionID,
		topicURL: topicURL,
		cancel:   cancel,
	}
	go handleSubscribeConnLoop(ctx, c.Messages, topicURL, subscriptionID, options...)
	return subscriptionID, nil
}

// Unsubscribe unsubscribes from a topic that has been previously subscribed to using the unique
// subscriptionID returned in Subscribe.
//
// Parameters:
//   - subscriptionID: The ID of the subscription to cancel.
func (c *Client) Unsubscribe(subscriptionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sub, ok := c.subscriptions[subscriptionID]
	if !ok {
		return
	}
	delete(c.subscriptions, subscriptionID)
	sub.cancel()
}

func (c *Client) expandTopicURL(topic string) (string, error) {
	if strings.HasPrefix(topic, "http://") || strings.HasPrefix(topic, "https://") {
		return topic, nil
	} else if strings.Contains(topic, "/") {
		return fmt.Sprintf("https://%s", topic), nil
	}
	if !topicRegex.MatchString(topic) {
		return "", fmt.Errorf("invalid topic name: %s", topic)
	}
	return fmt.Sprintf("%s/%s", c.config.DefaultHost, topic), nil
}

func handleSubscribeConnLoop(ctx context.Context, msgChan chan *Message, topicURL, subcriptionID string, options ...SubscribeOption) {
	for {
		// TODO The retry logic is crude and may lose messages. It should record the last message like the
		//      Android client, use since=, and do incremental backoff too
		if err := performSubscribeRequest(ctx, msgChan, topicURL, subcriptionID, options...); err != nil {
			log.Warn("%s Connection failed: %s", util.ShortTopicURL(topicURL), err.Error())
		}
		select {
		case <-ctx.Done():
			log.Info("%s Connection exited", util.ShortTopicURL(topicURL))
			return
		case <-time.After(10 * time.Second): // TODO Add incremental backoff
		}
	}
}

func performSubscribeRequest(ctx context.Context, msgChan chan *Message, topicURL string, subscriptionID string, options ...SubscribeOption) error {
	streamURL := fmt.Sprintf("%s/json", topicURL)
	log.Debug("%s Listening to %s", util.ShortTopicURL(topicURL), streamURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return err
	}
	for _, option := range options {
		if err := option(req); err != nil {
			return err
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		if err != nil {
			return err
		}
		return errors.New(strings.TrimSpace(string(b)))
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		messageJSON := scanner.Text()
		m, err := toMessage(messageJSON, topicURL, subscriptionID)
		if err != nil {
			return err
		}
		log.Trace("%s Message received: %s", util.ShortTopicURL(topicURL), messageJSON)
		if m.Event == MessageEvent {
			msgChan <- m
		}
	}
	return nil
}

func toMessage(s, topicURL, subscriptionID string) (*Message, error) {
	var m *Message
	if err := json.NewDecoder(strings.NewReader(s)).Decode(&m); err != nil {
		return nil, err
	}
	m.TopicURL = topicURL
	m.SubscriptionID = subscriptionID
	m.Raw = s
	return m, nil
}

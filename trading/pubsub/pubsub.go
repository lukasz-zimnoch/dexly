package pubsub

import (
	"cloud.google.com/go/pubsub"
	"context"
)

type Client struct {
	notificationsTopic *pubsub.Topic
}

func NewClient(
	ctx context.Context,
	projectID,
	notificationsTopicID string,
) (*Client, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &Client{
		notificationsTopic: client.Topic(notificationsTopicID),
	}, nil
}

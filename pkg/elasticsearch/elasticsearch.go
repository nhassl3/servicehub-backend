package elasticsearch

import (
	"context"
	"fmt"

	"github.com/elastic/go-elasticsearch/v9"
)

func New(ctx context.Context, hosts []string, username, password string) (*elasticsearch.Client, error) {
	client, err := elasticsearch.New(
		elasticsearch.WithAddresses(hosts...),
		elasticsearch.WithBasicAuth(username, password),
		elasticsearch.WithRetry(3),
	)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: new client: %w", err)
	}

	resp, err := client.Ping(client.Ping.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: ping: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("elasticsearch: ping: status %d", resp.StatusCode)
	}

	return client, nil
}

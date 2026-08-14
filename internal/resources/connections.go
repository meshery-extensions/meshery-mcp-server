package resources

import "time"

type Connection struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type ConnectionsResponse struct {
	Timestamp   time.Time    `json:"timestamp"`
	Connections []Connection `json:"connections"`
}

func GetConnections() (ConnectionsResponse, error) {
	return ConnectionsResponse{
		Timestamp: time.Now(),
		Connections: []Connection{
			{
				ID:     "1",
				Name:   "local-cluster",
				Type:   "kubernetes",
				Status: "connected",
			},
		},
	}, nil
}

package resources

import "time"

type Environment struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Connections []string `json:"connections"`
}

type EnvironmentsResponse struct {
	Timestamp    time.Time     `json:"timestamp"`
	Environments []Environment `json:"environments"`
}

func GetEnvironments() (EnvironmentsResponse, error) {
	return EnvironmentsResponse{
		Timestamp: time.Now(),
		Environments: []Environment{
			{
				ID:          "env-1",
				Name:        "development",
				Connections: []string{"conn-1"},
			},
			{
				ID:          "env-2",
				Name:        "production",
				Connections: []string{"conn-2"},
			},
		},
	}, nil
}

package resources

import "time"

type Provider struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	URL    string `json:"url"`
}

type ProvidersResponse struct {
	Timestamp time.Time  `json:"timestamp"`
	Providers []Provider `json:"providers"`
}

func GetProviders() (ProvidersResponse, error) {
	return ProvidersResponse{
		Timestamp: time.Now(),
		Providers: []Provider{
			{
				Name:   "Meshery",
				Status: "active",
				URL:    "http://localhost",
			},
		},
	}, nil
}

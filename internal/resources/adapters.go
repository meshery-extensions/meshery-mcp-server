package resources

import "time"

type Adapter struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

type AdaptersResponse struct {
	Timestamp time.Time `json:"timestamp"`
	Adapters  []Adapter `json:"adapters"`
}

func GetAdapters() (AdaptersResponse, error) {
	return AdaptersResponse{
		Timestamp: time.Now(),
		Adapters: []Adapter{
			{
				Name:    "Istio",
				Version: "1.18",
				Status:  "running",
			},
			{
				Name:    "Linkerd",
				Version: "2.13",
				Status:  "stopped",
			},
		},
	}, nil
}

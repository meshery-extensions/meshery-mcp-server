package resources

import "time"

type HealthResponse struct {
	Timestamp  time.Time         `json:"timestamp"`
	Status     string            `json:"status"`
	Components map[string]string `json:"components"`
}

func GetHealth() (HealthResponse, error) {
	return HealthResponse{
		Timestamp: time.Now(),
		Status:    "healthy",
		Components: map[string]string{
			"server":   "ok",
			"database": "ok",
			"adapters": "ok",
		},
	}, nil
}

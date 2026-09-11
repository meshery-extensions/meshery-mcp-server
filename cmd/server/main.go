package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type Request struct {
	Method string `json:"method"`
}

type Response struct {
	Result string `json:"result"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		var req Request
		err := json.Unmarshal(scanner.Bytes(), &req)
		if err != nil {
			fmt.Println(`{"error": "invalid request"}`)
			continue
		}

		if req.Method == "ping" {
			res := Response{Result: "pong"}
			bytes, _ := json.Marshal(res)
			fmt.Println(string(bytes))
		}
	}
}

package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
)

func Test_data(t *testing.T) {
	f, _ := os.Open("data.json")
	bytes, _ := io.ReadAll(f)

	var data [][]interface{}

	_ = json.Unmarshal(bytes, &data)

	for _, v := range data {
		fmt.Printf("%s ", v[2])
	}
	fmt.Println()
}

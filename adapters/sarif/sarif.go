package sarif

import (
	"encoding/json"
	"os"
)

type Result struct {
	Runs     int            `json:"runs"`
	Findings int            `json:"findings"`
	Levels   map[string]int `json:"levels"`
}

func Parse(path string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	var document struct {
		Runs []struct {
			Results []struct {
				Level string `json:"level"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err = json.Unmarshal(data, &document); err != nil {
		return Result{}, err
	}
	result := Result{Runs: len(document.Runs), Levels: map[string]int{}}
	for _, run := range document.Runs {
		for _, finding := range run.Results {
			result.Findings++
			level := finding.Level
			if level == "" {
				level = "warning"
			}
			result.Levels[level]++
		}
	}
	return result, nil
}

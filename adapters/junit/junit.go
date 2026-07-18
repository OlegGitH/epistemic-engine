package junit

import (
	"encoding/xml"
	"os"
)

type Result struct {
	Tests    int    `json:"tests"`
	Failures int    `json:"failures"`
	Errors   int    `json:"errors"`
	Skipped  int    `json:"skipped"`
	Name     string `json:"name,omitempty"`
}

type suite struct {
	XMLName  xml.Name `xml:"testsuite"`
	Name     string   `xml:"name,attr"`
	Tests    int      `xml:"tests,attr"`
	Failures int      `xml:"failures,attr"`
	Errors   int      `xml:"errors,attr"`
	Skipped  int      `xml:"skipped,attr"`
	Suites   []suite  `xml:"testsuite"`
}

func Parse(path string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	var root suite
	if err = xml.Unmarshal(data, &root); err != nil {
		return Result{}, err
	}
	result := Result{Tests: root.Tests, Failures: root.Failures, Errors: root.Errors, Skipped: root.Skipped, Name: root.Name}
	for _, child := range root.Suites {
		result.Tests += child.Tests
		result.Failures += child.Failures
		result.Errors += child.Errors
		result.Skipped += child.Skipped
	}
	return result, nil
}

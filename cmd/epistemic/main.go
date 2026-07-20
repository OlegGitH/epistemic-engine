package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	artifactadapter "github.com/OlegGitH/epistemic-engine/adapters/artifact"
	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
	fileprovider "github.com/OlegGitH/epistemic-engine/sdk/go/providers/file"
	"github.com/OlegGitH/epistemic-engine/sdk/go/providers/local"
	"github.com/OlegGitH/epistemic-engine/sdk/go/providers/noop"
	"github.com/OlegGitH/epistemic-engine/sdk/go/providers/remote"
)

type config struct {
	APIVersion string `yaml:"api_version"`
	Mode       string `yaml:"mode"`
	Decision   struct {
		ID             string `yaml:"id"`
		Recommendation string `yaml:"recommendation"`
		ActionType     string `yaml:"action_type"`
		SubjectType    string `yaml:"subject_type"`
		SubjectID      string `yaml:"subject_id"`
		RiskLevel      string `yaml:"risk_level"`
		Approved       bool   `yaml:"approved"`
		ApprovedBy     string `yaml:"approved_by"`
	} `yaml:"decision"`
	Provider struct {
		Type     string `yaml:"type"`
		Endpoint string `yaml:"endpoint"`
		Path     string `yaml:"path"`
	} `yaml:"provider"`
	Requirements []epistemic.Requirement  `yaml:"requirements"`
	Sources      []artifactadapter.Source `yaml:"sources"`
	Outputs      struct {
		Certificate string `yaml:"certificate"`
		Result      string `yaml:"result"`
		Report      string `yaml:"report"`
	} `yaml:"outputs"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(4)
	}
	switch os.Args[1] {
	case "evaluate":
		os.Exit(evaluate(os.Args[2:]))
	case "version":
		fmt.Println("epistemic protocol", epistemic.Version)
	default:
		usage()
		os.Exit(4)
	}
}

func evaluate(arguments []string) int {
	flags := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	configPath := flags.String("config", ".epistemic.yaml", "configuration path")
	endpoint := flags.String("endpoint", "", "override remote endpoint")
	if err := flags.Parse(arguments); err != nil {
		return 4
	}
	configuration, err := loadConfig(*configPath)
	if err != nil {
		return fail(err)
	}
	if *endpoint != "" {
		configuration.Provider.Endpoint = *endpoint
		configuration.Provider.Type = "remote"
	}
	provider, err := buildProvider(configuration)
	if err != nil {
		return fail(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	defer provider.Shutdown(context.Background())
	decisionID := configuration.Decision.ID
	if decisionID == "" {
		decisionID = fmt.Sprintf("decision-%x", time.Now().UnixNano())
	}
	runID := "run-" + decisionID
	protocolContext := epistemic.Context{DecisionID: decisionID, RunID: runID, Correlation: decisionID}
	events := make([]epistemic.Event, 0, len(configuration.Sources))
	for index, source := range configuration.Sources {
		event, eventErr := artifactadapter.Event(source, protocolContext, int64(index+1))
		if eventErr != nil {
			return fail(fmt.Errorf("source %s: %w", source.Path, eventErr))
		}
		events = append(events, event)
		if err = provider.Emit(ctx, event); err != nil {
			return fail(err)
		}
	}
	mode := configuration.Mode
	if mode == "" {
		mode = "advise"
	}
	request := epistemic.DecisionRequest{SpecVersion: epistemic.Version, DecisionID: decisionID, Recommendation: configuration.Decision.Recommendation, Action: epistemic.Action{Type: configuration.Decision.ActionType, Subject: epistemic.Subject{Type: configuration.Decision.SubjectType, ID: configuration.Decision.SubjectID}, RiskLevel: configuration.Decision.RiskLevel}, Context: protocolContext, Mode: mode, Requirements: configuration.Requirements, Events: events, Approval: epistemic.Approval{Approved: configuration.Decision.Approved, Actor: configuration.Decision.ApprovedBy}}
	result, err := provider.Evaluate(ctx, request)
	if err != nil {
		return fail(err)
	}
	if result.Certificate == nil {
		certificate, certErr := createCertificate(result)
		if certErr != nil {
			return fail(certErr)
		}
		result.Certificate = &certificate
	}
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
	if configuration.Outputs.Result != "" {
		if err = writeJSON(configuration.Outputs.Result, result); err != nil {
			return fail(err)
		}
	}
	if configuration.Outputs.Certificate != "" {
		if err = writeJSON(configuration.Outputs.Certificate, result.Certificate); err != nil {
			return fail(err)
		}
	}
	if configuration.Outputs.Report != "" {
		report, reportErr := epistemic.HumanReport(*result.Certificate)
		if reportErr != nil {
			return fail(reportErr)
		}
		if err = writeText(configuration.Outputs.Report, report); err != nil {
			return fail(err)
		}
	}
	if err = provider.Flush(ctx); err != nil {
		return fail(err)
	}
	if mode != "enforce" {
		return 0
	}
	switch result.Status {
	case "allow":
		if result.ActionAllowed {
			return 0
		}
		return 2
	case "block":
		return 2
	case "indeterminate":
		return 3
	default:
		return 4
	}
}

func loadConfig(path string) (config, error) {
	var value config
	data, err := os.ReadFile(path)
	if err != nil {
		return value, err
	}
	if err = yaml.Unmarshal(data, &value); err != nil {
		return value, err
	}
	if value.APIVersion != "epistemic.dev/v1alpha1" {
		return value, fmt.Errorf("unsupported api_version %q", value.APIVersion)
	}
	if value.Decision.Recommendation == "" || value.Decision.ActionType == "" || value.Decision.SubjectType == "" || value.Decision.SubjectID == "" {
		return value, errors.New("decision recommendation, action_type, subject_type, and subject_id are required")
	}
	if value.Mode != "" && value.Mode != "observe" && value.Mode != "advise" && value.Mode != "enforce" {
		return value, fmt.Errorf("unsupported mode %q", value.Mode)
	}
	return value, nil
}

func buildProvider(configuration config) (epistemic.Provider, error) {
	switch configuration.Provider.Type {
	case "remote":
		if configuration.Provider.Endpoint == "" {
			return nil, errors.New("remote provider requires endpoint")
		}
		return remote.New(configuration.Provider.Endpoint), nil
	case "file":
		path := configuration.Provider.Path
		if path == "" {
			path = ".epistemic/events.jsonl"
		}
		return fileprovider.New(path), nil
	case "noop":
		value := noop.New()
		return value, nil
	case "local":
		return local.New(), nil
	default:
		return nil, fmt.Errorf("unsupported provider type %q", configuration.Provider.Type)
	}
}

func createCertificate(result epistemic.DecisionResult) (epistemic.Certificate, error) {
	copyResult := result
	copyResult.Certificate = nil
	data, err := json.Marshal(copyResult)
	if err != nil {
		return epistemic.Certificate{}, err
	}
	certificate := epistemic.Certificate{SpecVersion: epistemic.Version, ID: "proof-" + result.DecisionID, DecisionID: result.DecisionID, Result: data, ArtifactHashes: []string{}, IssuedAt: time.Now().UTC(), Proof: epistemic.Proof{Algorithm: "SHA-256"}}
	digest, err := epistemic.Hash(certificate)
	if err != nil {
		return certificate, err
	}
	certificate.Proof.Digest = digest
	return certificate, nil
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
func writeText(path, value string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if !strings.HasSuffix(value, "\n") {
		value += "\n"
	}
	return os.WriteFile(path, []byte(value), 0o600)
}
func fail(err error) int { fmt.Fprintln(os.Stderr, "epistemic:", err); return 4 }
func usage()             { fmt.Fprintln(os.Stderr, "usage: epistemic <evaluate|version> [options]") }

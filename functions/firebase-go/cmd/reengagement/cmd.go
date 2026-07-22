package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultDelay = 250 * time.Millisecond

type recipient struct {
	Name        string
	Email       string
	SubmittedAt string
}

type campaignConfig struct {
	Environment string
	ResultsPath string
	ProjectID   string
	DatabaseID  string
	ContinueURL string
	LinkDomain  string
	VehicleURL  string
	CampaignID  string
	Send        bool
	Confirm     int
	Delay       time.Duration
	LogLevel    string
	Input       io.Reader
	Output      io.Writer
}

type resultRow struct {
	Recipient recipient
	Status    string
	Detail    string
	ResendID  string
}

type environmentSettings struct {
	ProjectID   string
	DatabaseID  string
	Host        string
	ContinueURL string
	VehicleURL  string
}

var settingsByEnvironment = map[string]environmentSettings{
	"staging": {
		ProjectID: "ipace-owners-staging", DatabaseID: "ipace-owners-staging", Host: "stage.ipace-owners.org",
		ContinueURL: "https://stage.ipace-owners.org/member/account/", VehicleURL: "https://stage.ipace-owners.org/member/vehicle/",
	},
	"production": {
		ProjectID: "ipace-owners-production", DatabaseID: "ipace-owners-production", Host: "ipace-owners.org",
		ContinueURL: "https://ipace-owners.org/member/account/", VehicleURL: "https://ipace-owners.org/member/vehicle/",
	},
}

func commandConfig() campaignConfig {
	config := campaignConfig{}
	flag.StringVar(&config.Environment, "env", "", "Firebase environment: staging or production")
	flag.StringVar(&config.ResultsPath, "results", "", "new CSV path for per-recipient results")
	flag.StringVar(&config.ProjectID, "project", "", "Firebase project override")
	flag.StringVar(&config.DatabaseID, "database", "", "Firestore database override")
	flag.StringVar(&config.ContinueURL, "continue-url", "", "post-sign-in account URL override")
	flag.StringVar(&config.LinkDomain, "link-domain", "", "Firebase email action link domain override")
	flag.StringVar(&config.VehicleURL, "vehicle-url", "", "secondary vehicle-data URL override")
	flag.StringVar(&config.CampaignID, "campaign-id", "", "Resend idempotency namespace; generated from environment and UTC date when omitted")
	flag.BoolVar(&config.Send, "send", false, "send emails; omitted means preflight-only dry run")
	flag.IntVar(&config.Confirm, "confirm-count", 0, "exact eligible recipient count required with --send")
	flag.DurationVar(&config.Delay, "delay", defaultDelay, "delay between sends (default 250ms)")
	flag.StringVar(&config.LogLevel, "log-level", "info", "logging level: info or debug (debug prints personal data)")
	flag.Parse()
	config.Input, config.Output = os.Stdin, os.Stdout
	return config
}

func validateConfig(config campaignConfig) error {
	if config.Environment != "staging" && config.Environment != "production" {
		return errors.New("--env must be staging or production")
	}
	for label, value := range map[string]string{
		"--results": config.ResultsPath, "--project": config.ProjectID, "--database": config.DatabaseID,
		"--continue-url": config.ContinueURL, "--link-domain": config.LinkDomain, "--vehicle-url": config.VehicleURL,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
	}
	for _, path := range []string{config.ResultsPath, manifestPath(config.ResultsPath)} {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("refusing to overwrite existing campaign file %s", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	for label, rawURL := range map[string]string{"--continue-url": config.ContinueURL, "--vehicle-url": config.VehicleURL} {
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
			return fmt.Errorf("%s must be an absolute HTTPS URL", label)
		}
	}
	if config.Send {
		if config.Confirm < 1 {
			return errors.New("--confirm-count must be positive with --send")
		}
		if strings.TrimSpace(os.Getenv("RESEND_API_KEY")) == "" || strings.TrimSpace(os.Getenv("RESEND_FROM")) == "" {
			return errors.New("RESEND_API_KEY and RESEND_FROM are required with --send")
		}
	}
	if config.Delay < 200*time.Millisecond {
		return errors.New("--delay must be at least 200ms to respect Resend's default 5 requests/second limit")
	}
	if config.LogLevel != "info" && config.LogLevel != "debug" {
		return errors.New("--log-level must be info or debug")
	}
	return nil
}

func resolveEnvironment(config campaignConfig) campaignConfig {
	config.Environment = strings.ToLower(strings.TrimSpace(config.Environment))
	settings, ok := settingsByEnvironment[config.Environment]
	if !ok {
		return config
	}
	if config.ProjectID == "" {
		config.ProjectID = settings.ProjectID
	}
	if config.DatabaseID == "" {
		config.DatabaseID = settings.DatabaseID
	}
	if config.ContinueURL == "" {
		config.ContinueURL = settings.ContinueURL
	}
	if config.VehicleURL == "" {
		config.VehicleURL = settings.VehicleURL
	}
	if config.LinkDomain == "" {
		config.LinkDomain = settings.Host
	}
	if config.CampaignID == "" {
		config.CampaignID = fmt.Sprintf("join-account-verification-%s-%s", config.Environment, time.Now().UTC().Format("2006-01-02"))
	}
	if config.Input == nil {
		config.Input = os.Stdin
	}
	if config.Output == nil {
		config.Output = os.Stdout
	}
	return config
}

func logCandidates(config campaignConfig, eligible []recipient) {
	fmt.Fprintf(config.Output, "campaign=%s eligible recipients=%d (use --log-level=debug to print personal data)\n", config.CampaignID, len(eligible))
	if config.LogLevel != "debug" {
		return
	}
	for _, person := range eligible {
		fmt.Fprintf(config.Output, "candidate name=%q email=%q submitted=%s\n", person.Name, person.Email, person.SubmittedAt)
	}
}

func confirmSend(config campaignConfig, count int) error {
	expected := fmt.Sprintf("SEND %s %d", config.Environment, count)
	fmt.Fprintf(config.Output, "Type %q to send %d emails for campaign %q: ", expected, count, config.CampaignID)
	response, err := bufio.NewReader(config.Input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read confirmation: %w", err)
	}
	if strings.TrimSpace(response) != expected {
		return errors.New("confirmation did not match; no emails sent")
	}
	return nil
}

func manifestPath(resultsPath string) string {
	return resultsPath + ".manifest.json"
}

func writeCampaignManifest(config campaignConfig, joined, authAccounts, authIdentities, matched, eligible int) error {
	manifest := struct {
		CampaignID     string        `json:"campaignId"`
		Environment    string        `json:"environment"`
		Mode           string        `json:"mode"`
		ProjectID      string        `json:"projectId"`
		DatabaseID     string        `json:"databaseId"`
		ContinueURL    string        `json:"continueUrl"`
		LinkDomain     string        `json:"linkDomain"`
		VehicleURL     string        `json:"vehicleUrl"`
		Delay          time.Duration `json:"delay"`
		Joined         int           `json:"joined"`
		AuthAccounts   int           `json:"authAccounts"`
		AuthIdentities int           `json:"authIdentities"`
		Matched        int           `json:"matched"`
		Eligible       int           `json:"eligible"`
		GeneratedAt    time.Time     `json:"generatedAt"`
	}{
		CampaignID: config.CampaignID, Environment: config.Environment, Mode: map[bool]string{true: "send", false: "dry-run"}[config.Send],
		ProjectID: config.ProjectID, DatabaseID: config.DatabaseID, ContinueURL: config.ContinueURL, LinkDomain: config.LinkDomain,
		VehicleURL: config.VehicleURL, Delay: config.Delay, Joined: joined, AuthAccounts: authAccounts, AuthIdentities: authIdentities,
		Matched: matched, Eligible: eligible, GeneratedAt: time.Now().UTC(),
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(manifestPath(config.ResultsPath), data, 0600)
}

func writeResults(path string, rows []resultRow) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"name", "email", "submitted_at", "status", "detail", "resend_id"}); err != nil {
		file.Close()
		return err
	}
	for _, row := range rows {
		if err := writer.Write([]string{row.Recipient.Name, row.Recipient.Email, row.Recipient.SubmittedAt, row.Status, row.Detail, row.ResendID}); err != nil {
			file.Close()
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

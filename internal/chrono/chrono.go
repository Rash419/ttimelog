package chrono

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rash419/ttimelog/internal/config"
	"github.com/Rash419/ttimelog/internal/timelog"
	"github.com/Rash419/ttimelog/internal/treeview"
)

func ParseProjectList(filePath string) (*treeview.TreeNode, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Error("Failed to close file", "error", err)
		}
	}()

	scanner := bufio.NewScanner(file)

	hiddenRoot := treeview.TreeNode{
		Expanded: true,
		Label: "Projects",
	}

	for scanner.Scan() {
		line := scanner.Text()
		ignoreLine := line == "" || strings.HasPrefix(line, "#") || strings.Contains(line, "*")
		if ignoreLine {
			continue
		}

		tokens := strings.Split(line, ":")
		if len(tokens) != 4 {
			continue
		}

		treeview.AppendPath(&hiddenRoot, tokens, 0)
	}
	return &hiddenRoot, nil
}

// SubmitTimesheet posts today's entries to the report_to_url endpoint.
func SubmitTimesheet(entries []timelog.Entry, appConfig *config.AppConfig) error {
	if appConfig.Gtimelog.ReportToURL == "" {
		return fmt.Errorf("report_to_url is not configured")
	}

	// Group today's entries by date, skipping arrived markers
	grouped := make(map[string]string)
	for _, entry := range entries {
		if !entry.Today {
			continue
		}
		if timelog.IsArrivedMessage(entry.Description) {
			continue
		}
		dateKey := entry.EndTime.Format("2006-01-02")
		line := fmt.Sprintf("%s %s\n", timelog.FormatDurationShort(entry.Duration), entry.Description)
		grouped[dateKey] += line
	}

	if len(grouped) == 0 {
		return fmt.Errorf("no entries to submit")
	}

	encoded := url.Values{}
	for key, val := range grouped {
		encoded.Set(key, val)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", appConfig.Gtimelog.ReportToURL, strings.NewReader(encoded.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", appConfig.Gtimelog.AuthHeader)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func FetchProjectList(appConfig *config.AppConfig) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", appConfig.Gtimelog.TaskListURL, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", appConfig.Gtimelog.AuthHeader)
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	projectListPath := filepath.Join(appConfig.TimeLogDirPath, config.ProjectListFile)
	projectListFile, err := os.Create(projectListPath)
	if err != nil {
		return fmt.Errorf("failed to create project-list[%s] with error[%v]", projectListPath, err)
	}
	_, err = projectListFile.Write(body)
	if err != nil {
		return fmt.Errorf("failed to write to project-list[%s] with error[%v]", projectListPath, err)
	}

	return nil
}

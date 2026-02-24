package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

const reportQueueSize = 64

// ErrorReportService handles error reporting to Forgejo issues.
type ErrorReportService struct {
	client              *ForgejoClient
	log                 *slog.Logger
	enabled             bool
	autoReportLabelID   int64
	userFeedbackLabelID int64
	mu                  sync.RWMutex // protects label IDs
	reportChan          chan ErrorReport
	done                chan struct{}
}

// ErrorReport represents an incoming error report.
type ErrorReport struct {
	Type             string `json:"type"`
	ErrorType        string `json:"error_type"`
	Message          string `json:"message"`
	Stack            string `json:"stack"`
	Fingerprint      string `json:"fingerprint"`
	URL              string `json:"url"`
	Component        string `json:"component"`
	UserAgent        string `json:"user_agent"`
	AppVersion       string `json:"app_version"`
	Description      string `json:"description"`
	StepsToReproduce string `json:"steps_to_reproduce"`
}

// ErrorReportResult is the response returned to the client.
type ErrorReportResult struct {
	Accepted bool `json:"accepted"`
}

// forgejoLabel represents a Forgejo label.
type forgejoLabel struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// forgejoIssue represents a Forgejo issue (minimal fields).
type forgejoIssue struct {
	Number int64  `json:"number"`
	Title  string `json:"title"`
}

// NewErrorReportService creates a new ErrorReportService.
// If any of forgejoURL, repoOwner, repoName, or apiToken is empty, the service is disabled.
func NewErrorReportService(forgejoURL, repoOwner, repoName, apiToken string, log *slog.Logger) *ErrorReportService {
	enabled := forgejoURL != "" && repoOwner != "" && repoName != "" && apiToken != ""

	if log == nil {
		log = slog.Default()
	}

	if enabled {
		log.Info("Error reporting enabled", "forgejo_url", forgejoURL, "repo", repoOwner+"/"+repoName)
	} else {
		log.Info("Error reporting disabled (missing FORGEJO_URL, FORGEJO_REPO, or FORGEJO_API_TOKEN)")
	}

	var client *ForgejoClient
	if enabled {
		client = NewForgejoClient(strings.TrimRight(forgejoURL, "/"), repoOwner, repoName, apiToken, log)
	}

	return &ErrorReportService{
		client:     client,
		log:        log,
		enabled:    enabled,
		reportChan: make(chan ErrorReport, reportQueueSize),
		done:       make(chan struct{}),
	}
}

// Start launches the background worker that processes queued error reports.
// Call Close() to shut down gracefully.
func (s *ErrorReportService) Start() {
	if !s.enabled {
		return
	}
	go s.processQueue()
}

// Close drains the report queue and stops the background worker.
func (s *ErrorReportService) Close() {
	close(s.reportChan)
	<-s.done
}

// EnqueueReport accepts an error report for async processing.
// Returns immediately. Reports are dropped if the queue is full.
func (s *ErrorReportService) EnqueueReport(report ErrorReport) ErrorReportResult {
	if !s.enabled {
		return ErrorReportResult{Accepted: false}
	}

	select {
	case s.reportChan <- report:
		return ErrorReportResult{Accepted: true}
	default:
		s.log.Warn("error report queue full, dropping report", "fingerprint", report.Fingerprint)
		return ErrorReportResult{Accepted: false}
	}
}

func (s *ErrorReportService) processQueue() {
	defer close(s.done)
	for report := range s.reportChan {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if _, err := s.SubmitReport(ctx, report); err != nil {
			s.log.Error("async error report failed", "error", err, "fingerprint", report.Fingerprint)
		}
		cancel()
	}
}

// IsEnabled returns true if error reporting is configured and available.
func (s *ErrorReportService) IsEnabled() bool {
	return s.enabled
}

// EnsureLabels resolves or creates the auto-report and user-feedback labels on Forgejo.
func (s *ErrorReportService) EnsureLabels(ctx context.Context) error {
	if !s.enabled {
		return nil
	}

	autoID, err := s.resolveOrCreateLabel(ctx, "auto-report", "#e11d48")
	if err != nil {
		return fmt.Errorf("failed to ensure auto-report label: %w", err)
	}

	feedbackID, err := s.resolveOrCreateLabel(ctx, "user-feedback", "#2563eb")
	if err != nil {
		return fmt.Errorf("failed to ensure user-feedback label: %w", err)
	}

	s.mu.Lock()
	s.autoReportLabelID = autoID
	s.userFeedbackLabelID = feedbackID
	s.mu.Unlock()

	s.log.Info("Error report labels ensured", "auto_report_id", autoID, "user_feedback_id", feedbackID)
	return nil
}

// SubmitReport submits an error report to Forgejo as an issue or comment.
func (s *ErrorReportService) SubmitReport(ctx context.Context, report ErrorReport) (ErrorReportResult, error) {
	if !s.enabled {
		return ErrorReportResult{Accepted: false}, nil
	}

	// Determine label IDs for this report
	s.mu.RLock()
	typeLabelID := s.autoReportLabelID
	if report.Type == "manual" {
		typeLabelID = s.userFeedbackLabelID
	}
	s.mu.RUnlock()

	// Search for existing issue with the same fingerprint
	existingIssue, err := s.searchExistingIssue(ctx, report.Fingerprint)
	if err != nil {
		s.log.Error("failed to search for existing issue", "error", err, "fingerprint", report.Fingerprint)
		// Continue to create a new issue — don't fail completely
	}

	if existingIssue != nil {
		// Add comment to existing issue
		commentBody := s.buildCommentBody(report)
		if err := s.client.AddComment(ctx, existingIssue.Number, commentBody); err != nil {
			s.log.Error("failed to add comment to existing issue", "error", err, "issue", existingIssue.Number)
			return ErrorReportResult{Accepted: false}, err
		}
		s.log.Info("added comment to existing issue", "issue", existingIssue.Number, "fingerprint", report.Fingerprint)
		return ErrorReportResult{Accepted: true}, nil
	}

	// Create fingerprint label
	fpLabelID, err := s.createFingerprintLabel(ctx, report.Fingerprint)
	if err != nil {
		s.log.Error("failed to create fingerprint label", "error", err, "fingerprint", report.Fingerprint)
		return ErrorReportResult{Accepted: false}, err
	}

	// Create new issue
	title := s.buildIssueTitle(report)
	body := s.buildIssueBody(report)
	labelIDs := []int64{typeLabelID, fpLabelID}

	for attempt := 0; attempt < 2; attempt++ {
		err = s.client.CreateIssue(ctx, title, body, labelIDs)
		if err == nil {
			break
		}
		// If label-related error (404/422), try re-resolving labels once
		if attempt == 0 && (strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "422")) {
			s.log.Warn("issue creation failed, re-resolving labels", "error", err)
			if resolveErr := s.EnsureLabels(ctx); resolveErr != nil {
				s.log.Error("failed to re-resolve labels", "error", resolveErr)
				return ErrorReportResult{Accepted: false}, err
			}
			// Update type label ID after re-resolve
			s.mu.RLock()
			if report.Type == "manual" {
				labelIDs[0] = s.userFeedbackLabelID
			} else {
				labelIDs[0] = s.autoReportLabelID
			}
			s.mu.RUnlock()
			continue
		}
		s.log.Error("failed to create issue", "error", err)
		return ErrorReportResult{Accepted: false}, err
	}

	s.log.Info("created new error report issue", "fingerprint", report.Fingerprint, "type", report.Type)
	return ErrorReportResult{Accepted: true}, nil
}

// searchExistingIssue searches for an open issue with the given fingerprint.
// Tries label-based search first, then falls back to body search.
func (s *ErrorReportService) searchExistingIssue(ctx context.Context, fingerprint string) (*forgejoIssue, error) {
	fpLabelName := "fp:" + fingerprint

	// Primary: search by fingerprint label
	issues, err := s.client.SearchIssuesByLabel(ctx, fpLabelName)
	if err != nil {
		return nil, err
	}
	if len(issues) > 0 {
		return &issues[0], nil
	}

	// Fallback: search by fingerprint marker in body
	searchQuery := fmt.Sprintf("fingerprint:%s", fingerprint)
	issues, err = s.client.SearchIssuesByQuery(ctx, searchQuery)
	if err != nil {
		return nil, err
	}
	if len(issues) > 0 {
		return &issues[0], nil
	}

	return nil, nil
}

func (s *ErrorReportService) resolveOrCreateLabel(ctx context.Context, name, color string) (int64, error) {
	labels, err := s.client.FetchLabels(ctx)
	if err != nil {
		return 0, err
	}

	for _, l := range labels {
		if l.Name == name {
			return l.ID, nil
		}
	}

	// Label not found — create it
	label, err := s.client.CreateLabel(ctx, name, color)
	if err != nil {
		return 0, err
	}
	if label == nil {
		// Concurrent creation — resolve again
		s.log.Warn("label creation conflict, attempting to resolve", "name", name)
		return s.resolveLabel(ctx, name)
	}
	return label.ID, nil
}

// resolveLabel finds a label by name (used as fallback after create conflict).
func (s *ErrorReportService) resolveLabel(ctx context.Context, name string) (int64, error) {
	labels, err := s.client.FetchLabels(ctx)
	if err != nil {
		return 0, err
	}

	for _, l := range labels {
		if l.Name == name {
			return l.ID, nil
		}
	}

	return 0, fmt.Errorf("label %q not found after creation conflict", name)
}

// createFingerprintLabel creates a fingerprint label (fp:xxxx) for dedup.
func (s *ErrorReportService) createFingerprintLabel(ctx context.Context, fingerprint string) (int64, error) {
	name := "fp:" + fingerprint
	label, err := s.client.CreateLabel(ctx, name, "#6b7280")
	if err != nil {
		return 0, err
	}
	if label == nil {
		// Concurrent creation — resolve
		return s.resolveLabel(ctx, name)
	}
	return label.ID, nil
}

func (s *ErrorReportService) buildIssueTitle(report ErrorReport) string {
	title := report.Message
	if len(title) > 100 {
		title = title[:97] + "..."
	}
	if report.Type == "manual" {
		return "[Feedback] " + title
	}
	prefix := "Error"
	if report.ErrorType != "" {
		prefix = report.ErrorType
	}
	return fmt.Sprintf("[%s] %s", prefix, title)
}

func (s *ErrorReportService) buildIssueBody(report ErrorReport) string {
	var b strings.Builder

	b.WriteString("<!-- fingerprint:")
	b.WriteString(report.Fingerprint)
	b.WriteString(" -->\n\n")

	if report.Type == "manual" {
		b.WriteString("## User Feedback\n\n")
		b.WriteString(report.Message)
		b.WriteString("\n")
		if report.StepsToReproduce != "" {
			b.WriteString("\n### Steps to Reproduce\n\n")
			b.WriteString(report.StepsToReproduce)
			b.WriteString("\n")
		}
	} else {
		b.WriteString("## Error Details\n\n")
		b.WriteString(fmt.Sprintf("**Type:** `%s`\n", report.ErrorType))
		b.WriteString(fmt.Sprintf("**Message:** %s\n", report.Message))
		if report.Stack != "" {
			b.WriteString("\n### Stack Trace\n\n```\n")
			b.WriteString(report.Stack)
			b.WriteString("\n```\n")
		}
	}

	b.WriteString("\n### Context\n\n")
	if report.URL != "" {
		b.WriteString(fmt.Sprintf("- **Page:** `%s`\n", report.URL))
	}
	if report.Component != "" {
		b.WriteString(fmt.Sprintf("- **Component:** `%s`\n", report.Component))
	}
	if report.UserAgent != "" {
		b.WriteString(fmt.Sprintf("- **User-Agent:** `%s`\n", report.UserAgent))
	}
	if report.AppVersion != "" {
		b.WriteString(fmt.Sprintf("- **Version:** `%s`\n", report.AppVersion))
	}
	b.WriteString(fmt.Sprintf("- **Fingerprint:** `%s`\n", report.Fingerprint))

	return b.String()
}

func (s *ErrorReportService) buildCommentBody(report ErrorReport) string {
	var b strings.Builder

	b.WriteString("### Duplicate Report\n\n")

	if report.Type == "manual" {
		b.WriteString("**User Feedback:**\n")
		b.WriteString(report.Message)
		b.WriteString("\n")
		if report.StepsToReproduce != "" {
			b.WriteString("\n**Steps to Reproduce:**\n")
			b.WriteString(report.StepsToReproduce)
			b.WriteString("\n")
		}
	} else {
		b.WriteString(fmt.Sprintf("**Message:** %s\n", report.Message))
		if report.Stack != "" {
			b.WriteString(fmt.Sprintf("\n```\n%s\n```\n", report.Stack))
		}
	}

	if report.UserAgent != "" {
		b.WriteString(fmt.Sprintf("\n- **User-Agent:** `%s`\n", report.UserAgent))
	}
	if report.AppVersion != "" {
		b.WriteString(fmt.Sprintf("- **Version:** `%s`\n", report.AppVersion))
	}

	return b.String()
}

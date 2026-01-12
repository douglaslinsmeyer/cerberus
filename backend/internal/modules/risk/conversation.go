package risk

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// ConversationService handles conversation thread and message operations
type ConversationService struct {
	repo RepositoryInterface
}

// NewConversationService creates a new conversation service
func NewConversationService(repo *Repository) *ConversationService {
	return &ConversationService{
		repo: repo,
	}
}

// CreateThread creates a new conversation thread
func (s *ConversationService) CreateThread(ctx context.Context, req CreateThreadRequest) (uuid.UUID, error) {
	if req.RiskID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("risk_id is required")
	}
	if req.CreatedBy == uuid.Nil {
		return uuid.Nil, fmt.Errorf("created_by is required")
	}
	if req.Title == "" {
		return uuid.Nil, fmt.Errorf("title is required")
	}

	// Validate thread type
	if !isValidThreadType(req.ThreadType) {
		return uuid.Nil, fmt.Errorf("invalid thread_type: must be discussion, status_update, decision, or escalation")
	}

	// Verify risk exists
	_, err := s.repo.GetRiskByID(ctx, req.RiskID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("risk not found: %w", err)
	}

	// Create thread
	threadID := uuid.New()
	thread := &ConversationThread{
		ThreadID:     threadID,
		RiskID:       req.RiskID,
		Title:        req.Title,
		ThreadType:   req.ThreadType,
		IsResolved:   false,
		MessageCount: 0,
		CreatedBy:    req.CreatedBy,
		CreatedAt:    time.Now(),
	}

	err = s.repo.CreateThread(ctx, thread)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create thread: %w", err)
	}

	return threadID, nil
}

// GetThread retrieves a thread by ID
func (s *ConversationService) GetThread(ctx context.Context, threadID uuid.UUID) (*ConversationThread, error) {
	if threadID == uuid.Nil {
		return nil, fmt.Errorf("thread_id is required")
	}

	thread, err := s.repo.GetThreadByID(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	return thread, nil
}

// GetThreadWithMessages retrieves a thread with all its messages
func (s *ConversationService) GetThreadWithMessages(ctx context.Context, threadID uuid.UUID) (*ThreadWithMessages, error) {
	if threadID == uuid.Nil {
		return nil, fmt.Errorf("thread_id is required")
	}

	threadWithMessages, err := s.repo.GetThreadWithMessages(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread with messages: %w", err)
	}

	return threadWithMessages, nil
}

// ListThreads retrieves all threads for a risk
func (s *ConversationService) ListThreads(ctx context.Context, riskID uuid.UUID) ([]ConversationThread, error) {
	if riskID == uuid.Nil {
		return nil, fmt.Errorf("risk_id is required")
	}

	threads, err := s.repo.ListThreadsByRisk(ctx, riskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list threads: %w", err)
	}

	return threads, nil
}

// ResolveThread marks a thread as resolved
func (s *ConversationService) ResolveThread(ctx context.Context, threadID, resolvedBy uuid.UUID) error {
	if threadID == uuid.Nil {
		return fmt.Errorf("thread_id is required")
	}
	if resolvedBy == uuid.Nil {
		return fmt.Errorf("resolved_by is required")
	}

	// Verify thread exists
	_, err := s.repo.GetThreadByID(ctx, threadID)
	if err != nil {
		return fmt.Errorf("thread not found: %w", err)
	}

	err = s.repo.ResolveThread(ctx, threadID, resolvedBy)
	if err != nil {
		return fmt.Errorf("failed to resolve thread: %w", err)
	}

	return nil
}

// DeleteThread soft-deletes a thread
func (s *ConversationService) DeleteThread(ctx context.Context, threadID uuid.UUID) error {
	if threadID == uuid.Nil {
		return fmt.Errorf("thread_id is required")
	}

	err := s.repo.DeleteThread(ctx, threadID)
	if err != nil {
		return fmt.Errorf("failed to delete thread: %w", err)
	}

	return nil
}

// AddMessage adds a message to a thread
func (s *ConversationService) AddMessage(ctx context.Context, req CreateMessageRequest) (uuid.UUID, error) {
	if req.ThreadID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("thread_id is required")
	}
	if req.CreatedBy == uuid.Nil {
		return uuid.Nil, fmt.Errorf("created_by is required")
	}
	if req.MessageText == "" {
		return uuid.Nil, fmt.Errorf("message_text is required")
	}

	// Validate message format
	if req.MessageFormat == "" {
		req.MessageFormat = "markdown" // Default to markdown
	}
	if !isValidMessageFormat(req.MessageFormat) {
		return uuid.Nil, fmt.Errorf("invalid message_format: must be markdown or plain_text")
	}

	// Verify thread exists
	_, err := s.repo.GetThreadByID(ctx, req.ThreadID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("thread not found: %w", err)
	}

	// Parse mentions from message
	mentionedUserIDs := s.parseMentions(req.MessageText)

	// Create message
	messageID := uuid.New()
	message := &ConversationMessage{
		MessageID:        messageID,
		ThreadID:         req.ThreadID,
		MessageText:      req.MessageText,
		MessageFormat:    req.MessageFormat,
		MentionedUserIDs: mentionedUserIDs,
		CreatedBy:        req.CreatedBy,
		CreatedAt:        time.Now(),
	}

	err = s.repo.CreateMessage(ctx, message)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create message: %w", err)
	}

	return messageID, nil
}

// GetMessages retrieves all messages for a thread
func (s *ConversationService) GetMessages(ctx context.Context, threadID uuid.UUID) ([]ConversationMessage, error) {
	if threadID == uuid.Nil {
		return nil, fmt.Errorf("thread_id is required")
	}

	messages, err := s.repo.GetThreadMessages(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	return messages, nil
}

// DeleteMessage soft-deletes a message
func (s *ConversationService) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	if messageID == uuid.Nil {
		return fmt.Errorf("message_id is required")
	}

	err := s.repo.DeleteMessage(ctx, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// parseMentions extracts user mentions from message text
// Supports @username and @[UUID] formats
func (s *ConversationService) parseMentions(messageText string) []uuid.UUID {
	// Pattern to match @[UUID] format
	uuidPattern := regexp.MustCompile(`@\[([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\]`)

	matches := uuidPattern.FindAllStringSubmatch(messageText, -1)

	var mentionedUserIDs []uuid.UUID
	seen := make(map[uuid.UUID]bool)

	for _, match := range matches {
		if len(match) > 1 {
			if userID, err := uuid.Parse(match[1]); err == nil {
				// Deduplicate mentions
				if !seen[userID] {
					mentionedUserIDs = append(mentionedUserIDs, userID)
					seen[userID] = true
				}
			}
		}
	}

	return mentionedUserIDs
}

// FormatMentionForUser creates a mention string for a user ID
func FormatMentionForUser(userID uuid.UUID) string {
	return fmt.Sprintf("@[%s]", userID.String())
}

// Validation helpers

func isValidThreadType(threadType string) bool {
	validTypes := map[string]bool{
		"discussion":    true,
		"status_update": true,
		"decision":      true,
		"escalation":    true,
	}
	return validTypes[threadType]
}

func isValidMessageFormat(format string) bool {
	validFormats := map[string]bool{
		"markdown":   true,
		"plain_text": true,
	}
	return validFormats[format]
}

// GetUnresolvedThreads retrieves all unresolved threads for a risk
func (s *ConversationService) GetUnresolvedThreads(ctx context.Context, riskID uuid.UUID) ([]ConversationThread, error) {
	if riskID == uuid.Nil {
		return nil, fmt.Errorf("risk_id is required")
	}

	// Get all threads
	threads, err := s.repo.ListThreadsByRisk(ctx, riskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list threads: %w", err)
	}

	// Filter for unresolved
	var unresolved []ConversationThread
	for _, thread := range threads {
		if !thread.IsResolved {
			unresolved = append(unresolved, thread)
		}
	}

	return unresolved, nil
}

// GetThreadStatistics returns statistics for a thread
func (s *ConversationService) GetThreadStatistics(ctx context.Context, threadID uuid.UUID) (*ThreadStatistics, error) {
	if threadID == uuid.Nil {
		return nil, fmt.Errorf("thread_id is required")
	}

	// Get thread with messages
	threadWithMessages, err := s.repo.GetThreadWithMessages(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	// Calculate statistics
	stats := &ThreadStatistics{
		ThreadID:      threadID,
		TotalMessages: len(threadWithMessages.Messages),
		IsResolved:    threadWithMessages.IsResolved,
		CreatedAt:     threadWithMessages.CreatedAt,
	}

	// Set LastMessageAt if valid
	if threadWithMessages.LastMessageAt.Valid {
		stats.LastMessageAt.Time = threadWithMessages.LastMessageAt.Time
		stats.LastMessageAt.Valid = true
	}

	// Count unique participants
	participants := make(map[uuid.UUID]bool)
	for _, msg := range threadWithMessages.Messages {
		participants[msg.CreatedBy] = true
	}
	stats.UniqueParticipants = len(participants)

	// Count mentions
	totalMentions := 0
	for _, msg := range threadWithMessages.Messages {
		totalMentions += len(msg.MentionedUserIDs)
	}
	stats.TotalMentions = totalMentions

	return stats, nil
}

// ThreadStatistics represents statistics for a conversation thread
type ThreadStatistics struct {
	ThreadID           uuid.UUID    `json:"thread_id"`
	TotalMessages      int          `json:"total_messages"`
	UniqueParticipants int          `json:"unique_participants"`
	TotalMentions      int          `json:"total_mentions"`
	IsResolved         bool         `json:"is_resolved"`
	CreatedAt          time.Time    `json:"created_at"`
	LastMessageAt      floatNullTime `json:"last_message_at,omitempty"`
}

// floatNullTime is a helper type for nullable time
type floatNullTime struct {
	Time  time.Time
	Valid bool
}

// MarshalJSON implements json.Marshaler for floatNullTime
func (t floatNullTime) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return []byte("null"), nil
	}
	return t.Time.MarshalJSON()
}

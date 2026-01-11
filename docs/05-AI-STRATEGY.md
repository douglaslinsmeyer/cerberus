# Cerberus AI Integration Strategy

This document defines the comprehensive strategy for integrating Anthropic's Claude API into Cerberus, covering implementation patterns, prompt engineering, cost optimization, and operational best practices.

---

## AI Philosophy

**AI-First, Not AI-Bolted-On**

Cerberus is designed from the ground up with AI as a core capability, not an afterthought. Every module leverages Claude API for:
- **Intelligence:** Extract meaning from unstructured data
- **Automation:** Reduce manual analysis time by 50%+
- **Proactivity:** Detect risks and issues before they escalate
- **Context-Awareness:** Every AI query leverages full program history

**Human-in-the-Loop**

AI augments human decision-making, never replaces it:
- Confidence scores on all AI outputs
- User can accept, edit, or reject AI suggestions
- Critical decisions require human approval
- Feedback loop improves AI accuracy over time

---

## Claude API Integration Architecture

### Model Selection Strategy

**Two Models, Strategic Usage**

| Model | Use Cases | Rationale | Cost |
|-------|-----------|-----------|------|
| **Claude Opus 4.5** | Complex analysis, deep reasoning | Superior performance on multi-step reasoning | $15/MTok input, $75/MTok output |
| **Claude Sonnet 4.5** | Frequent operations, summaries | Fast, cost-effective, still highly capable | $3/MTok input, $15/MTok output |

**Decision Matrix:**

```
Use Opus 4.5 for:
- Risk impact analysis (multiple dimensions, cascading effects)
- Change request impact assessment (cross-module analysis)
- Financial forensics (complex variance investigations)
- Decision impact analysis (strategic implications)
- Health score calculation (multi-source synthesis)

Use Sonnet 4.5 for:
- Artifact metadata extraction (routine, frequent)
- Invoice data extraction (structured extraction)
- Communication drafting (template-following)
- Meeting summarization (straightforward)
- Simple validation tasks
```

**Estimated Cost Breakdown:**
- Artifact analysis (Sonnet): $0.20-0.30 per document
- Deep analysis (Opus): $0.80-1.20 per analysis
- With 90% caching: $0.30-0.50 per artifact (blended average)

### Go Client Implementation

```go
// internal/platform/ai/client.go

package ai

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type Client struct {
    apiKey      string
    baseURL     string
    httpClient  *http.Client
    cache       CacheManager
    metrics     MetricsCollector
    rateLimiter RateLimiter
}

func NewClient(apiKey string, cache CacheManager, metrics MetricsCollector) *Client {
    return &Client{
        apiKey:  apiKey,
        baseURL: "https://api.anthropic.com/v1",
        httpClient: &http.Client{
            Timeout: 120 * time.Second,
        },
        cache:       cache,
        metrics:     metrics,
        rateLimiter: NewRateLimiter(100), // 100 req/min
    }
}

type Request struct {
    Model         string          `json:"model"`
    MaxTokens     int             `json:"max_tokens"`
    System        string          `json:"system,omitempty"`
    Messages      []Message       `json:"messages"`
    Temperature   float64         `json:"temperature,omitempty"`
    Stream        bool            `json:"stream,omitempty"`
}

type Message struct {
    Role    string          `json:"role"`
    Content []ContentBlock  `json:"content"`
}

type ContentBlock struct {
    Type         string        `json:"type"`
    Text         string        `json:"text,omitempty"`
    CacheControl *CacheControl `json:"cache_control,omitempty"`
}

type CacheControl struct {
    Type string `json:"type"` // "ephemeral"
}

type Response struct {
    ID           string      `json:"id"`
    Content      []Content   `json:"content"`
    Usage        Usage       `json:"usage"`
    StopReason   string      `json:"stop_reason"`
}

type Content struct {
    Type string `json:"type"`
    Text string `json:"text"`
}

type Usage struct {
    InputTokens         int `json:"input_tokens"`
    OutputTokens        int `json:"output_tokens"`
    CacheCreationTokens int `json:"cache_creation_input_tokens"`
    CacheReadTokens     int `json:"cache_read_input_tokens"`
}

func (c *Client) Request(ctx context.Context, req *Request) (*Response, error) {
    // Check cache first
    cacheKey := c.generateCacheKey(req)
    if cached, found := c.cache.Get(ctx, cacheKey); found {
        return cached.(*Response), nil
    }

    // Rate limit
    if err := c.rateLimiter.Wait(ctx); err != nil {
        return nil, err
    }

    // Make API request
    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    httpReq, err := http.NewRequestWithContext(
        ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("x-api-key", c.apiKey)
    httpReq.Header.Set("anthropic-version", "2023-06-01")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
    }

    var apiResp Response
    if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
        return nil, err
    }

    // Calculate cost
    cost := c.calculateCost(req.Model, &apiResp.Usage)

    // Track metrics
    c.metrics.Track(ctx, &Metrics{
        Model:        req.Model,
        InputTokens:  apiResp.Usage.InputTokens,
        OutputTokens: apiResp.Usage.OutputTokens,
        CachedTokens: apiResp.Usage.CacheReadTokens,
        Cost:         cost,
        Duration:     time.Since(time.Now()),
    })

    // Cache response (1 hour TTL)
    c.cache.Set(ctx, cacheKey, &apiResp, time.Hour)

    return &apiResp, nil
}

func (c *Client) calculateCost(model string, usage *Usage) float64 {
    var inputCost, outputCost float64

    switch model {
    case "claude-opus-4-5-20251101":
        inputCost = 15.0 / 1_000_000  // $15 per MTok
        outputCost = 75.0 / 1_000_000 // $75 per MTok
    case "claude-sonnet-4-5-20250929":
        inputCost = 3.0 / 1_000_000   // $3 per MTok
        outputCost = 15.0 / 1_000_000 // $15 per MTok
    }

    // Cached tokens cost 10% of normal
    cachedCost := float64(usage.CacheReadTokens) * inputCost * 0.1
    normalInputCost := float64(usage.InputTokens-usage.CacheReadTokens) * inputCost
    outputCostTotal := float64(usage.OutputTokens) * outputCost

    return cachedCost + normalInputCost + outputCostTotal
}
```

---

## AI Processing Pipeline: Artifact Ingestion

### Ingestion Flow

```
User Upload → Store File → Extract Content → Chunk Document →
Queue AI Job → Background Worker → AI Analysis → Parse Response →
Store Metadata → Publish Events → Notify User
```

### Chunking Strategy

**Why Chunking?**
- Large documents (100+ pages) exceed single prompt capacity
- Semantic chunking preserves context better than arbitrary splits
- Parallel processing of chunks speeds up analysis

**Implementation:**

```go
// internal/modules/artifacts/chunking.go

type ChunkingStrategy struct {
    MaxTokens     int     // 4000-8000 tokens per chunk
    OverlapTokens int     // 200 tokens overlap
    PreserveStructure bool // Respect sections/headings
}

func (cs *ChunkingStrategy) ChunkDocument(content string) []Chunk {
    // Use tiktoken for accurate token counting
    tokens := cs.tokenize(content)

    chunks := []Chunk{}
    start := 0

    for start < len(tokens) {
        end := min(start+cs.MaxTokens, len(tokens))

        // Try to break at natural boundaries (paragraph, section)
        if end < len(tokens) && cs.PreserveStructure {
            end = cs.findNaturalBreak(tokens, start, end)
        }

        chunkTokens := tokens[start:end]
        chunks = append(chunks, Chunk{
            Index:       len(chunks),
            Text:        cs.detokenize(chunkTokens),
            StartOffset: start,
            EndOffset:   end,
        })

        // Overlap for context continuity
        start = end - cs.OverlapTokens
    }

    return chunks
}

func (cs *ChunkingStrategy) findNaturalBreak(tokens []int, start, end int) int {
    // Look for paragraph breaks, section headings
    // within last 10% of chunk
    searchStart := end - (cs.MaxTokens / 10)

    for i := end - 1; i >= searchStart; i-- {
        if cs.isNaturalBreak(tokens[i]) {
            return i
        }
    }

    return end // No natural break found, use max
}
```

### AI Analysis Service

```go
// internal/modules/artifacts/ai_service.go

type AIService struct {
    client  *ai.Client
    prompts *PromptLibrary
}

func (s *AIService) AnalyzeArtifact(
    ctx context.Context,
    artifact *domain.Artifact,
    programContext *ProgramContext,
) (*ArtifactMetadata, error) {

    // Build prompt with context
    prompt := s.prompts.Get("artifact_analysis").
        WithContext(programContext).
        WithArtifact(artifact)

    // Make API request
    resp, err := s.client.Request(ctx, &ai.Request{
        Model:     "claude-sonnet-4-5-20250929",
        MaxTokens: 4096,
        System:    prompt.SystemPrompt,
        Messages: []ai.Message{{
            Role: "user",
            Content: []ai.ContentBlock{
                {
                    Type: "text",
                    Text: prompt.ProgramContext,
                    CacheControl: &ai.CacheControl{Type: "ephemeral"}, // Cache this
                },
                {
                    Type: "text",
                    Text: fmt.Sprintf("Artifact: %s\n\n%s", artifact.Filename, artifact.RawContent),
                },
            },
        }},
        Temperature: 0.0, // Deterministic for extraction
    })

    if err != nil {
        return nil, err
    }

    // Parse structured JSON response
    var metadata ArtifactMetadata
    if err := json.Unmarshal([]byte(resp.Content[0].Text), &metadata); err != nil {
        return nil, err
    }

    return &metadata, nil
}
```

---

## Prompt Engineering Templates

### Template Architecture

```go
// internal/platform/ai/prompts.go

type PromptTemplate struct {
    ID              string
    Version         string
    Module          string
    Purpose         string
    SystemPrompt    string
    UserPromptFmt   string
    OutputSchema    *JSONSchema
    RequiredContext []string
    Model           string
    Temperature     float64
}

type PromptLibrary struct {
    templates map[string]*PromptTemplate
}

func (pl *PromptLibrary) Get(id string) *PromptTemplate {
    return pl.templates[id]
}

// Prompt template with variable substitution
type CompiledPrompt struct {
    SystemPrompt   string
    ProgramContext string
    Variables      map[string]interface{}
}

func (pt *PromptTemplate) WithContext(ctx *ProgramContext) *CompiledPrompt {
    return &CompiledPrompt{
        SystemPrompt:   pt.SystemPrompt,
        ProgramContext: ctx.ToPromptString(),
        Variables:      make(map[string]interface{}),
    }
}
```

### Module-Specific Prompts

#### 1. Artifact Analysis (CORE)

```go
var ArtifactAnalysisPrompt = &PromptTemplate{
    ID:      "artifact_analysis_v1",
    Module:  "artifacts",
    Purpose: "Extract structured metadata from any artifact",
    Model:   "claude-sonnet-4-5-20250929",
    Temperature: 0.0,

    SystemPrompt: `You are an expert document analyst for enterprise program management.

Your role is to analyze any uploaded artifact and extract structured metadata to build program knowledge.

Extract with high accuracy:
1. Executive summary (2-3 sentences)
2. Key topics/themes using program taxonomy
3. People mentioned (names, roles, context)
4. Important facts (dates, amounts, metrics, commitments)
5. Decisions or action items
6. Risk indicators or concerns
7. Financial data
8. Links to existing program context

Be thorough but concise. Provide confidence scores (0.0-1.0) for all extractions.`,

    UserPromptFmt: `Program Context:
- Program: {{.ProgramName}}
- Taxonomy: {{.CustomTaxonomy}}
- Key Stakeholders: {{.StakeholderList}}
- Active Risks: {{.RiskSummary}}

Task: Analyze this artifact and extract structured metadata.

Output as JSON matching this schema:
{
  "summary": "string",
  "key_topics": [{"topic": "string", "confidence": 0.95}],
  "persons_mentioned": [{"name": "string", "role": "string", "context": "string", "confidence": 0.9}],
  "facts": [{"type": "date|amount|metric", "key": "string", "value": "string", "confidence": 0.85}],
  "insights": [{"type": "risk|opportunity|action", "title": "string", "description": "string", "severity": "string", "confidence": 0.8}],
  "financial_data": {"amounts": [], "vendors": []},
  "sentiment": "positive|neutral|concern",
  "priority": 1-5
}

Artifact Content:
{{.ArtifactContent}}`,

    OutputSchema: &JSONSchema{/* Schema definition */},
}
```

#### 2. Invoice Validation (Financial)

```go
var InvoiceValidationPrompt = &PromptTemplate{
    ID:          "invoice_validation_v1",
    Module:      "financial",
    Purpose:     "Validate invoice against rate cards and detect variances",
    Model:       "claude-sonnet-4-5-20250929",
    Temperature: 0.0,

    SystemPrompt: `You are a financial analyst expert in program governance.

Your role is to analyze invoices against approved rate cards and schedules, identifying discrepancies and categorizing expenditures.

Be precise with numbers and conservative with variance flagging.`,

    UserPromptFmt: `Program Context:
- Rate Cards: {{.RateCards}}
- Approved Schedules: {{.Schedules}}
- Budget Status: {{.BudgetSummary}}

Task: Analyze this invoice and:
1. Extract all line items (description, quantity, rate, amount)
2. Validate each line item against rate cards (flag mismatches)
3. Check dates/hours against approved schedules
4. Categorize spend into program taxonomy
5. Calculate variance from expected costs
6. Identify any red flags or anomalies

Output as JSON:
{
  "validation_status": "approved|flagged|rejected",
  "line_items": [
    {
      "description": "string",
      "quantity": 100,
      "unit_rate": 200,
      "amount": 20000,
      "expected_rate": 180,
      "variance": 20,
      "variance_percentage": 11.1,
      "flag": "rate_overage",
      "spend_category": "labor"
    }
  ],
  "total_variance": 5000,
  "variance_percentage": 10.5,
  "flags": ["rate_overage_line_3", "unusual_vendor"],
  "spend_categories": {"labor": 50000, "materials": 10000},
  "recommended_action": "approve_with_review|reject|request_clarification"
}

Invoice Content:
{{.InvoiceContent}}`,
}
```

#### 3. Risk Detection (Risk Management)

```go
var RiskDetectionPrompt = &PromptTemplate{
    ID:          "risk_detection_v1",
    Module:      "risk",
    Purpose:     "Identify potential risks from artifacts",
    Model:       "claude-opus-4-5-20251101", // Use Opus for deep analysis
    Temperature: 0.1,

    SystemPrompt: `You are a risk management specialist for enterprise programs.

Your role is to identify potential risks from new information and connect them to existing program risks.

Be proactive but not alarmist. Focus on actionable risk identification.`,

    UserPromptFmt: `Existing Risks:
{{.ActiveRisks}}

Recent Program Context:
{{.RecentArtifactsSummary}}

Task: Given this new artifact, identify:
1. New potential risks (with severity, likelihood)
2. Links to existing risks (does this provide new context, escalate, or mitigate existing risks?)
3. Suggested risk categories and ownership
4. Recommended immediate actions if critical

Be specific about:
- What could go wrong
- Why it matters (impact)
- How likely it is
- What evidence supports this risk

Output as JSON:
{
  "new_risks": [
    {
      "title": "string",
      "description": "string",
      "category": "technical|financial|schedule|resource|external",
      "probability": "very_low|low|medium|high|very_high",
      "impact": "very_low|low|medium|high|very_high",
      "evidence": "string",
      "suggested_owner": "string",
      "recommended_action": "string",
      "confidence": 0.85
    }
  ],
  "existing_risk_updates": [
    {
      "risk_id": "existing_risk_reference",
      "update_type": "new_context|escalation|mitigation",
      "description": "string",
      "confidence": 0.9
    }
  ]
}

Artifact Content:
{{.ArtifactContent}}`,
}
```

#### 4. Communication Drafting (Communications)

```go
var CommunicationDraftingPrompt = &PromptTemplate{
    ID:          "communication_drafting_v1",
    Module:      "communications",
    Purpose:     "Draft communications following templates",
    Model:       "claude-sonnet-4-5-20250929",
    Temperature: 0.3, // Slightly creative for natural writing

    SystemPrompt: `You are a professional communications specialist for program governance.

Your role is to draft communications following established templates and incorporating program context.

Maintain appropriate tone for audience, follow template structure, and include actionable information.`,

    UserPromptFmt: `Communication Plan: {{.CommunicationPlan}}

Template:
{{.Template}}

Program Status:
- Health Score: {{.HealthScore}}
- Recent Risks: {{.RecentRisks}}
- Budget Status: {{.BudgetStatus}}
- Upcoming Milestones: {{.UpcomingMilestones}}
- Recent Decisions: {{.RecentDecisions}}

Task: Draft a {{.CommunicationType}} for {{.Audience}} that:
- Follows the template structure
- Incorporates relevant program updates
- Maintains appropriate tone ({{.ToneGuidance}})
- Highlights key decisions/milestones/risks
- Includes actionable next steps

Provide 2 versions:
1. Concise (executive summary)
2. Detailed (full context)

Output as JSON:
{
  "subject": "string",
  "concise_version": "string",
  "detailed_version": "string",
  "key_points": ["point1", "point2"],
  "calls_to_action": ["action1", "action2"]
}`,
}
```

#### 5. Decision Extraction (Decision Log)

```go
var DecisionExtractionPrompt = &PromptTemplate{
    ID:          "decision_extraction_v1",
    Module:      "decision",
    Purpose:     "Extract decisions from meeting notes",
    Model:       "claude-sonnet-4-5-20250929",
    Temperature: 0.0,

    SystemPrompt: `You are a program decision analyst.

Your role is to extract decisions from meeting notes and analyze their impacts.

Be precise about what was decided, by whom, and what the implications are.`,

    UserPromptFmt: `Program Context:
{{.ProgramOverview}}

Recent Decisions:
{{.RecentDecisions}}

Task: From this meeting note:
1. Extract all decisions made (who, what, when, why)
2. Identify impacted areas (scope, budget, schedule, risk, quality)
3. Link to existing decisions or artifacts
4. Assess decision urgency and significance
5. Suggest follow-up actions or approvals needed

Output as JSON:
{
  "decisions": [
    {
      "title": "string",
      "description": "string",
      "decision_maker": "string",
      "participants": ["person1", "person2"],
      "rationale": "string",
      "alternatives_considered": "string",
      "impact_areas": ["budget", "schedule"],
      "impact_description": "string",
      "urgency": "critical|high|medium|low",
      "requires_approval": true,
      "confidence": 0.9
    }
  ]
}

Meeting Notes:
{{.MeetingNotes}}`,
}
```

#### 6. Health Scoring (Dashboard)

```go
var HealthScoringPrompt = &PromptTemplate{
    ID:          "health_scoring_v1",
    Module:      "dashboard",
    Purpose:     "Calculate program health score with insights",
    Model:       "claude-opus-4-5-20251101", // Use Opus for synthesis
    Temperature: 0.1,

    SystemPrompt: `You are a program health analyst.

Your role is to generate health scores and predictive alerts based on comprehensive program data.

Be data-driven, specific, and actionable in your assessments.`,

    UserPromptFmt: `Program Data:
- Financial: {{.FinancialSummary}}
  - Budget: {{.BudgetStatus}}
  - Recent Invoices: {{.RecentInvoices}}
  - Variance Trend: {{.VarianceTrend}}

- Schedule: {{.ScheduleStatus}}
  - Milestones Achieved: {{.MilestonesAchieved}}
  - Milestones At Risk: {{.MilestonesAtRisk}}
  - Velocity: {{.Velocity}}

- Risks: {{.RiskProfile}}
  - Open Risks: {{.OpenRisks}}
  - Critical Risks: {{.CriticalRisks}}
  - Risk Trend: {{.RiskTrend}}

- Quality: {{.QualityMetrics}}

- Stakeholder Sentiment: {{.StakeholderSentiment}}

Recent Decisions: {{.RecentDecisions}}

Task: Generate:
1. Overall health score (0-100) with clear reasoning
2. Category scores (financial, schedule, risk, quality, stakeholder)
3. Trend analysis (improving, stable, declining)
4. Predictive alerts (potential issues in next 30/60/90 days)
5. Recommended leadership actions (top 3 priorities)

Be specific about:
- What's working well
- What's concerning
- What needs immediate attention
- What to watch

Output as JSON:
{
  "overall_score": 82,
  "overall_status": "yellow",
  "trend": "declining",
  "category_scores": {
    "financial": {"score": 75, "status": "yellow", "reasoning": "..."},
    "schedule": {"score": 90, "status": "green", "reasoning": "..."},
    "risk": {"score": 70, "status": "yellow", "reasoning": "..."},
    "quality": {"score": 85, "status": "green", "reasoning": "..."},
    "stakeholder": {"score": 88, "status": "green", "reasoning": "..."}
  },
  "summary": "Program health declined 8 points this week primarily due to budget variance...",
  "key_concerns": ["Budget overrun in Phase 2", "Vendor delivery delays", "3 new critical risks"],
  "positive_trends": ["Milestone M5 achieved ahead of schedule", "Stakeholder engagement high"],
  "predictive_alerts": [
    {
      "horizon_days": 45,
      "type": "budget",
      "severity": "high",
      "description": "Predicted Phase 3 budget overrun based on current burn rate",
      "confidence": 0.85
    }
  ],
  "recommended_actions": [
    "Review vendor contract for Phase 2 deliverables",
    "Conduct budget review with finance team",
    "Escalate critical risks to steering committee"
  ]
}`,
}
```

#### 7. Change Impact Analysis (Change Control)

```go
var ChangeImpactAnalysisPrompt = &PromptTemplate{
    ID:          "change_impact_analysis_v1",
    Module:      "changecontrol",
    Purpose:     "Analyze change request impacts across program",
    Model:       "claude-opus-4-5-20251101", // Complex multi-dimensional analysis
    Temperature: 0.0,

    SystemPrompt: `You are a change impact analyst for enterprise programs.

Your role is to assess how proposed changes affect all program dimensions: scope, schedule, budget, resources, risks, quality.

Be thorough and conservative in impact assessment.`,

    UserPromptFmt: `Program Artifacts Summary:
{{.ArtifactsSummary}}

Current Baselines:
- Scope: {{.ScopeBaseline}}
- Schedule: {{.ScheduleBaseline}}
- Budget: {{.BudgetBaseline}}
- Resources: {{.ResourceBaseline}}

Active Risks: {{.ActiveRisks}}

Change Request:
{{.ChangeRequestDetails}}

Task: Assess impact across:
1. Schedule (which milestones affected, critical path impact, estimated delay)
2. Budget (cost increase/decrease, funding sources, contingency impact)
3. Resources (team capacity, skill requirements, vendor contracts)
4. Risks (new risks introduced, existing risks mitigated/escalated)
5. Quality (quality implications, testing impact)
6. Dependencies (other changes, decisions, approvals required)

Provide:
- Specific impact estimates (not vague)
- Confidence levels for estimates
- Approval pathway recommendation
- Conditions for approval (if applicable)

Output as JSON:
{
  "schedule_impact": {
    "affected_milestones": ["M7", "M9"],
    "delay_days": 21,
    "critical_path_affected": true,
    "description": "...",
    "confidence": 0.8
  },
  "budget_impact": {
    "cost_increase_usd": 75000,
    "affected_categories": ["labor", "software"],
    "funding_source_suggestion": "contingency",
    "description": "...",
    "confidence": 0.85
  },
  "resource_impact": {
    "additional_fte": 0.5,
    "skill_requirements": ["security specialist"],
    "vendor_changes": false,
    "description": "...",
    "confidence": 0.75
  },
  "risk_impact": {
    "new_risks": ["Security compliance delay"],
    "mitigated_risks": [],
    "escalated_risks": ["R-042"],
    "overall_risk_change": "increased",
    "description": "...",
    "confidence": 0.9
  },
  "quality_impact": {
    "quality_improvement": true,
    "testing_impact": "Additional security testing required",
    "description": "...",
    "confidence": 0.8
  },
  "dependencies": {
    "requires_decisions": ["Security vendor selection"],
    "requires_approvals": ["CISO approval"],
    "related_changes": [],
    "description": "..."
  },
  "recommendation": {
    "action": "approve_with_conditions",
    "rationale": "Strategic value justifies cost and delay, but requires security vendor decision first",
    "conditions": ["Secure CISO approval", "Complete vendor selection", "Defer Feature X to Phase 3"],
    "approval_level": "executive_steering_committee",
    "confidence": 0.85
  }
}`,
}
```

---

## Context Assembly Strategy

### Program Context Structure

```go
// internal/platform/ai/context.go

type ProgramContext struct {
    ProgramID       string
    ProgramName     string
    ProgramCode     string

    // Taxonomy
    CustomTaxonomy  map[string][]string

    // Stakeholders
    KeyStakeholders []StakeholderSummary

    // Current State
    HealthScore     int
    BudgetStatus    BudgetSummary
    ScheduleStatus  ScheduleSummary
    ActiveRisks     []RiskSummary
    RecentDecisions []DecisionSummary

    // Artifacts (recent/relevant)
    RecentArtifacts []ArtifactSummary

    // Last updated
    ContextTimestamp time.Time
}

func (pc *ProgramContext) ToPromptString() string {
    // Serialize to human-readable format for AI
    var sb strings.Builder

    sb.WriteString(fmt.Sprintf("Program: %s (%s)\n", pc.ProgramName, pc.ProgramCode))
    sb.WriteString(fmt.Sprintf("Health Score: %d\n", pc.HealthScore))
    sb.WriteString(fmt.Sprintf("Budget Status: %s\n", pc.BudgetStatus.Summary()))
    sb.WriteString(fmt.Sprintf("Active Risks: %d (%d critical)\n",
        len(pc.ActiveRisks), pc.countCriticalRisks()))

    sb.WriteString("\nKey Stakeholders:\n")
    for _, s := range pc.KeyStakeholders {
        sb.WriteString(fmt.Sprintf("- %s (%s)\n", s.Name, s.Role))
    }

    sb.WriteString("\nCustom Taxonomy:\n")
    for category, values := range pc.CustomTaxonomy {
        sb.WriteString(fmt.Sprintf("- %s: %s\n", category, strings.Join(values, ", ")))
    }

    return sb.String()
}
```

### Context Builder with Caching

```go
// internal/platform/ai/context_builder.go

type ContextBuilder struct {
    db    *sql.DB
    cache *redis.Client
}

func (cb *ContextBuilder) BuildContext(
    ctx context.Context,
    programID string,
    purpose ContextPurpose,
) (*ProgramContext, error) {

    // Check if we have cached program context (refreshed daily)
    cacheKey := fmt.Sprintf("program_context:%s:%s",
        programID, time.Now().Format("2006-01-02"))

    if cached, err := cb.cache.Get(ctx, cacheKey).Result(); err == nil {
        var pc ProgramContext
        json.Unmarshal([]byte(cached), &pc)
        return &pc, nil
    }

    // Build fresh context
    pc := &ProgramContext{
        ProgramID:       programID,
        ContextTimestamp: time.Now(),
    }

    // Load program details
    cb.loadProgramDetails(ctx, pc)

    // Load taxonomy
    cb.loadTaxonomy(ctx, pc)

    // Load key stakeholders
    cb.loadStakeholders(ctx, pc)

    // Load current state
    cb.loadCurrentState(ctx, pc)

    // Load recent artifacts (last 30 days, top 50 by relevance)
    cb.loadRecentArtifacts(ctx, pc, 30, 50)

    // Cache for 24 hours
    cached, _ := json.Marshal(pc)
    cb.cache.Set(ctx, cacheKey, cached, 24*time.Hour)

    return pc, nil
}

// Purpose-specific context assembly
func (cb *ContextBuilder) BuildContextForPurpose(
    ctx context.Context,
    programID string,
    purpose ContextPurpose,
    additionalParams map[string]interface{},
) (*CompiledPrompt, error) {

    baseContext := cb.BuildContext(ctx, programID, purpose)

    switch purpose {
    case FinancialAnalysis:
        return cb.buildFinancialContext(ctx, baseContext, additionalParams)
    case RiskIdentification:
        return cb.buildRiskContext(ctx, baseContext, additionalParams)
    case CommunicationDrafting:
        return cb.buildCommsContext(ctx, baseContext, additionalParams)
    default:
        return cb.buildGenericContext(ctx, baseContext)
    }
}
```

---

## Cost Optimization Techniques

### 1. Prompt Caching (70-90% Savings)

**Strategy:** Cache static program context that changes infrequently.

```go
func (s *AIService) RequestWithCaching(
    ctx context.Context,
    staticContext string,  // Cached
    dynamicContext string, // Not cached
) (*ai.Response, error) {

    return s.client.Request(ctx, &ai.Request{
        Model:     "claude-sonnet-4-5-20250929",
        MaxTokens: 4096,
        Messages: []ai.Message{{
            Role: "user",
            Content: []ai.ContentBlock{
                {
                    Type: "text",
                    Text: staticContext,
                    CacheControl: &ai.CacheControl{Type: "ephemeral"}, // ⭐ CACHE THIS
                },
                {
                    Type: "text",
                    Text: dynamicContext, // Not cached
                },
            },
        }},
    })
}
```

**What to Cache:**
- Program overview (changes rarely)
- Custom taxonomy (changes rarely)
- Key stakeholders (changes weekly)
- Rate cards (changes monthly)
- Communication templates (changes rarely)

**What NOT to Cache:**
- New artifact content (always unique)
- Specific analysis requests (always unique)
- Real-time data (health scores, budgets)

**Expected Savings:** 70-90% on repeated queries

### 2. Response Caching (Redis, 1-hour TTL)

**Strategy:** Cache identical AI requests for 1 hour.

```go
func (c *Client) generateCacheKey(req *Request) string {
    // Hash the request to generate cache key
    h := sha256.New()
    json.NewEncoder(h).Encode(req)
    return fmt.Sprintf("ai_response:%x", h.Sum(nil))
}
```

**Expected Savings:** 80-100% on cache hits (varies by usage pattern)

### 3. Lazy Analysis

**Strategy:** Perform deep analysis only when user requests it.

```
On Upload:
- Basic metadata extraction (Sonnet) ✓
- Quick confidence scoring ✓

On User Click "Deep Analysis":
- Full analysis (Opus) ✓
- Extended insights ✓
```

**Expected Savings:** 50% (many artifacts never need deep analysis)

### 4. Batch Processing

**Strategy:** Process multiple similar items in one request.

```
Instead of:
- 5 API calls for 5 invoices = $2.50

Do:
- 1 API call for all 5 invoices = $0.80
```

**Expected Savings:** 40-60% on bulk operations

### 5. Smart Context Assembly

**Strategy:** Include only relevant artifacts, not entire history.

```go
// Bad: Include all 500 artifacts (exceeds context limit)
allArtifacts := fetchAllArtifacts(programID)

// Good: Include top 20 most relevant
relevantArtifacts := fetchRelevantArtifacts(programID, query, limit=20)
```

**Expected Savings:** 30-50% on token usage

---

## Cost Monitoring & Alerting

### Usage Tracking

```go
// internal/platform/ai/metrics.go

type Metrics struct {
    ProgramID    string
    Module       string
    JobType      string
    Model        string
    InputTokens  int
    OutputTokens int
    CachedTokens int
    Cost         float64
    Duration     time.Duration
    Timestamp    time.Time
}

func (m *MetricsCollector) Track(ctx context.Context, metrics *Metrics) {
    // Store in database
    _, err := m.db.ExecContext(ctx, `
        INSERT INTO ai_usage (
            program_id, module, job_type, model,
            tokens_input, tokens_output, tokens_cached,
            cost_usd, duration_ms, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
    `, metrics.ProgramID, metrics.Module, metrics.JobType, metrics.Model,
       metrics.InputTokens, metrics.OutputTokens, metrics.CachedTokens,
       metrics.Cost, metrics.Duration.Milliseconds())

    // Check if daily limit exceeded
    if m.isDailyLimitExceeded(ctx, metrics.ProgramID) {
        m.sendAlert(ctx, metrics.ProgramID)
    }
}

func (m *MetricsCollector) isDailyLimitExceeded(ctx context.Context, programID string) bool {
    var dailyCost float64
    m.db.QueryRowContext(ctx, `
        SELECT COALESCE(SUM(cost_usd), 0)
        FROM ai_usage
        WHERE program_id = $1
          AND created_at >= CURRENT_DATE
    `, programID).Scan(&dailyCost)

    return dailyCost > 100.00 // $100/day limit
}
```

### Cost Analytics Dashboard

**Metrics to Track:**
- Daily cost per program
- Cost per module
- Token usage trends
- Cache hit rates
- Average cost per artifact

**Alerts:**
- Daily cost > $100 (email program admin)
- Monthly cost projection > $3000 (escalate)
- Cache hit rate < 50% (investigate)
- Single request > $5 (log for review)

---

## Operational Best Practices

### Error Handling

```go
func (s *AIService) AnalyzeWithRetry(
    ctx context.Context,
    artifact *Artifact,
) (*Metadata, error) {

    var lastErr error

    for attempt := 0; attempt < 3; attempt++ {
        result, err := s.analyze(ctx, artifact)

        if err == nil {
            return result, nil
        }

        // Retry on transient errors
        if isTransientError(err) {
            time.Sleep(time.Duration(attempt+1) * time.Second)
            lastErr = err
            continue
        }

        // Don't retry on permanent errors
        return nil, err
    }

    return nil, fmt.Errorf("failed after 3 attempts: %w", lastErr)
}

func isTransientError(err error) bool {
    // Rate limit, timeout, 5xx errors
    return strings.Contains(err.Error(), "rate_limit") ||
           strings.Contains(err.Error(), "timeout") ||
           strings.Contains(err.Error(), "500") ||
           strings.Contains(err.Error(), "503")
}
```

### Rate Limiting

```go
type RateLimiter struct {
    limiter *rate.Limiter
    burst   int
}

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
    return &RateLimiter{
        limiter: rate.NewLimiter(rate.Limit(requestsPerMinute/60), requestsPerMinute),
        burst:   requestsPerMinute,
    }
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
    return rl.limiter.Wait(ctx)
}
```

### Graceful Degradation

```go
// If AI fails, don't block user workflow
func (s *AIService) AnalyzeOrMarkPending(
    ctx context.Context,
    artifact *Artifact,
) error {

    err := s.analyze(ctx, artifact)

    if err != nil {
        // Log error, mark for manual review
        log.Error("AI analysis failed", "artifact_id", artifact.ID, "error", err)

        artifact.ProcessingStatus = "failed_manual_review_needed"
        s.repo.Update(ctx, artifact)

        // Notify user
        s.notifier.Send(ctx, &Notification{
            UserID: artifact.UploadedBy,
            Type:   "ai_analysis_failed",
            Message: fmt.Sprintf("Artifact %s requires manual review", artifact.Filename),
        })

        return nil // Don't fail workflow
    }

    return nil
}
```

---

## Testing & Validation

### AI Output Validation

```go
func validateAIMetadata(metadata *ArtifactMetadata) error {
    // Validate structure
    if metadata.Summary == "" {
        return errors.New("missing summary")
    }

    // Validate confidence scores
    for _, topic := range metadata.Topics {
        if topic.Confidence < 0 || topic.Confidence > 1 {
            return fmt.Errorf("invalid confidence score: %f", topic.Confidence)
        }
    }

    // Validate required fields based on artifact type
    if metadata.DetectedType == "invoice" {
        if len(metadata.FinancialData.Amounts) == 0 {
            return errors.New("invoice must have amounts")
        }
    }

    return nil
}
```

### Prompt Testing Framework

```go
// Test prompts with known inputs
func TestInvoiceValidationPrompt(t *testing.T) {
    testCases := []struct{
        invoice string
        expected ValidationResult
    }{
        {
            invoice: loadTestInvoice("valid_invoice.pdf"),
            expected: ValidationResult{
                Status: "approved",
                Variance: 0,
            },
        },
        {
            invoice: loadTestInvoice("overage_invoice.pdf"),
            expected: ValidationResult{
                Status: "flagged",
                Variance: 15.5,
                Flags: []string{"rate_overage"},
            },
        },
    }

    for _, tc := range testCases {
        result := analyzeInvoice(tc.invoice)
        assert.Equal(t, tc.expected.Status, result.Status)
    }
}
```

---

## Summary: Expected Performance

### Cost Estimates (with optimization)

**Per Artifact:**
- Basic analysis (Sonnet): $0.20-0.30
- Deep analysis (Opus): $0.80-1.20
- With 90% caching: $0.30-0.50 average

**Per Program (10 artifacts/day):**
- Daily: $3-5
- Monthly: $90-150

**Cost Optimization Impact:**
- Without optimization: $500-800/month per program
- With optimization: $90-150/month per program
- **Savings: 70-85%**

### Performance Targets

- **Artifact Processing:** <30 seconds
- **AI Response Time:** <10 seconds (Sonnet), <20 seconds (Opus)
- **Cache Hit Rate:** >70%
- **Accuracy:** >85% precision on extractions
- **Uptime:** >99.9% (AI availability)

---

**Document Version:** 1.0
**Last Updated:** 2026-01-10
**Document Owner:** Technical Lead
**Status:** Approved for Implementation

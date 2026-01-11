# Cerberus Module Specifications

This document provides detailed specifications for all 10 core modules in the Cerberus enterprise program governance system.

---

## Module 1: Program Artifacts (CORE CONTEXT ENGINE) ⭐

### Purpose
The **central intelligence hub** of Cerberus. This module ingests any file type, extracts structured metadata via AI, and provides context to all other modules. Think of it as the "brain" that feeds intelligence to the entire system.

### Key Features

**Universal File Ingestion**
- Upload any file type: PDF, Excel, Word, PowerPoint, images, CSV, text, meeting transcriptions
- Drag-and-drop interface
- Bulk upload support
- File deduplication via content hash
- Version tracking for updated artifacts

**AI-Powered Analysis**
- Automatic content extraction (OCR for PDFs/images)
- Intelligent chunking (4K-8K tokens per chunk with overlap)
- Metadata extraction:
  - **Summary:** Executive summary (2-3 sentences)
  - **Topics:** Classification using program taxonomy
  - **Persons:** Named entity recognition with roles and context
  - **Facts:** Dates, amounts, commitments, metrics
  - **Insights:** Risks, opportunities, anomalies, action items
  - **Decisions:** Automatically captured from meeting notes
  - **Sentiment:** Overall document tone (positive, neutral, concern)
  - **Priority:** AI-assigned priority level (1-5)
- Vector embeddings for semantic search (pgvector)

**Context Provision**
- Expose artifact metadata to all modules via API
- Event publishing when artifacts are analyzed
- Semantic search across all program documents
- Related artifacts recommendation

### User Workflows

**Primary: Upload & Review**
1. User uploads document (invoice, meeting notes, contract)
2. System extracts content and queues for AI analysis
3. Background worker processes document with Claude API
4. User receives notification when analysis complete
5. User reviews extracted metadata (topics, persons, insights)
6. User can accept, edit, or dismiss AI suggestions
7. Metadata is indexed and available to other modules

**Secondary: Search & Discover**
1. User searches for "budget variance Q2"
2. System performs semantic search across all artifacts
3. Results ranked by relevance with highlighted snippets
4. User can view full artifact or extracted metadata

**Tertiary: Link & Contextualize**
1. User creates risk "Vendor delivery delay"
2. System suggests related artifacts (vendor emails, contracts)
3. User links artifacts to provide context
4. Risk automatically inherits context from linked artifacts

### Data Model

```sql
-- Core artifact table
CREATE TABLE artifacts (
    artifact_id UUID PRIMARY KEY,
    program_id UUID NOT NULL,
    filename VARCHAR(500),
    file_type VARCHAR(100),
    file_size_bytes BIGINT,
    storage_path TEXT,
    content_hash VARCHAR(64) UNIQUE,
    raw_content TEXT,
    processing_status VARCHAR(50), -- pending, processing, completed, failed
    ai_model_version VARCHAR(50),
    uploaded_by UUID,
    uploaded_at TIMESTAMPTZ,
    processed_at TIMESTAMPTZ
);

-- AI-extracted topics
CREATE TABLE artifact_topics (
    topic_id UUID PRIMARY KEY,
    artifact_id UUID REFERENCES artifacts,
    topic_name VARCHAR(255),
    confidence_score DECIMAL(5,4),
    parent_topic_id UUID REFERENCES artifact_topics
);

-- AI-extracted persons
CREATE TABLE artifact_persons (
    person_id UUID PRIMARY KEY,
    artifact_id UUID REFERENCES artifacts,
    person_name VARCHAR(255),
    person_role VARCHAR(255),
    context_snippets JSONB,
    confidence_score DECIMAL(5,4),
    stakeholder_id UUID REFERENCES stakeholders -- Link to stakeholder module
);

-- AI-extracted facts
CREATE TABLE artifact_facts (
    fact_id UUID PRIMARY KEY,
    artifact_id UUID REFERENCES artifacts,
    fact_type VARCHAR(100), -- date, amount, metric, commitment
    fact_key VARCHAR(255),
    fact_value TEXT,
    normalized_value_numeric DECIMAL(20,4),
    normalized_value_date DATE,
    unit VARCHAR(50),
    confidence_score DECIMAL(5,4)
);

-- AI-generated insights
CREATE TABLE artifact_insights (
    insight_id UUID PRIMARY KEY,
    artifact_id UUID REFERENCES artifacts,
    insight_type VARCHAR(100), -- risk, opportunity, action_required, anomaly
    title VARCHAR(500),
    description TEXT,
    severity VARCHAR(50),
    suggested_action TEXT,
    impacted_modules VARCHAR(100)[], -- Array: which modules should see this
    confidence_score DECIMAL(5,4),
    is_dismissed BOOLEAN DEFAULT false
);

-- Vector embeddings for semantic search
CREATE TABLE artifact_embeddings (
    embedding_id UUID PRIMARY KEY,
    artifact_id UUID REFERENCES artifacts,
    chunk_index INTEGER,
    chunk_text TEXT,
    embedding vector(1536) -- pgvector
);
```

### API Endpoints

```
POST   /api/v1/programs/:programId/artifacts/upload
GET    /api/v1/programs/:programId/artifacts
GET    /api/v1/programs/:programId/artifacts/:id
GET    /api/v1/programs/:programId/artifacts/:id/metadata
GET    /api/v1/programs/:programId/artifacts/:id/download
POST   /api/v1/programs/:programId/artifacts/:id/reanalyze
POST   /api/v1/programs/:programId/artifacts/search
DELETE /api/v1/programs/:programId/artifacts/:id
```

### AI Integration

**Prompt Template:**
```
System: You are an expert document analyst for enterprise program management.

Program Context:
- Program: {{program_name}}
- Taxonomy: {{custom_taxonomy}}
- Key Stakeholders: {{stakeholder_list}}
- Active Risks: {{risk_summary}}

Task: Analyze this artifact and extract:
1. Executive summary (2-3 sentences)
2. Key topics/themes using program taxonomy
3. People mentioned (names, roles, context)
4. Decisions or action items
5. Risk indicators or concerns
6. Financial data or commitments
7. Links to existing program context

Output as structured JSON.
```

**Model Selection:** Claude Sonnet 4.5 (fast, cost-effective for routine extraction)

### Success Metrics
- **Processing Time:** <30 seconds for typical document
- **Accuracy:** >85% precision on metadata extraction
- **User Acceptance:** >70% of AI suggestions accepted without edits
- **Search Relevance:** >80% of searches return useful results in top 5

---

## Module 2: Program Financial Reporting

### Purpose
AI-powered invoice validation, variance detection, and categorical spend tracking. Automatically flags budget anomalies and ensures rate card compliance.

### Key Features

**Invoice Management**
- Upload invoices (PDF, Excel, images)
- OCR extraction of invoice data
- Line-item level detail capture
- Vendor tracking
- Payment status tracking

**Rate Card Compliance**
- Define approved rate cards (by role, seniority, skill)
- Automatic line-item validation against rates
- Flag overages and discrepancies
- Historical rate comparison

**Variance Analysis**
- Compare actual vs expected costs
- Identify budget anomalies
- Calculate burn rates and projections
- Category-level variance tracking

**Categorical Spend Tracking**
- Automatic spend categorization (labor, materials, software, travel)
- Multi-dimensional analysis (by milestone, workstream, vendor, category)
- Budget vs actual tracking per category
- Trend visualization

### User Workflows

**Primary: Invoice Validation**
1. User uploads vendor invoice
2. AI extracts invoice details (line items, amounts, dates)
3. System checks each line item against rate cards
4. Flags found: "Senior Developer billed at $250/hr, rate card is $200/hr"
5. System calculates expected cost vs actual
6. AI suggests spend categories for each line item
7. User reviews, approves, or disputes invoice
8. If variance exceeds threshold, AI recommends creating risk

**Secondary: Budget Analysis**
1. User views financial dashboard
2. See categorical spend breakdown (pie chart)
3. Identify budget overages by category
4. Drill down into specific invoices
5. Export financial report for executives

### Data Model

```sql
CREATE TABLE rate_cards (
    rate_card_id UUID PRIMARY KEY,
    program_id UUID,
    name VARCHAR(255),
    effective_start_date DATE,
    effective_end_date DATE,
    currency VARCHAR(3) DEFAULT 'USD'
);

CREATE TABLE rate_card_items (
    item_id UUID PRIMARY KEY,
    rate_card_id UUID REFERENCES rate_cards,
    role_title VARCHAR(255),
    rate_type VARCHAR(50), -- hourly, daily, fixed
    rate_amount DECIMAL(15,2),
    seniority_level VARCHAR(50)
);

CREATE TABLE invoices (
    invoice_id UUID PRIMARY KEY,
    program_id UUID,
    artifact_id UUID REFERENCES artifacts,
    invoice_number VARCHAR(100),
    vendor_name VARCHAR(255),
    invoice_date DATE,
    total_amount DECIMAL(15,2),
    payment_status VARCHAR(50),
    approval_status VARCHAR(50)
);

CREATE TABLE invoice_line_items (
    line_item_id UUID PRIMARY KEY,
    invoice_id UUID REFERENCES invoices,
    description TEXT,
    quantity DECIMAL(15,4),
    unit_rate DECIMAL(15,4),
    line_amount DECIMAL(15,2),
    spend_category VARCHAR(100),
    budget_category_id UUID
);

CREATE TABLE budget_categories (
    category_id UUID PRIMARY KEY,
    program_id UUID,
    category_name VARCHAR(255),
    budgeted_amount DECIMAL(15,2),
    fiscal_year INTEGER,
    fiscal_quarter INTEGER
);
```

### AI Integration

**Invoice Extraction Prompt:**
```
System: You are a financial analyst expert in program governance.

Context:
- Rate Cards: {{rate_cards}}
- Budget Status: {{budget_summary}}

Task: Analyze this invoice and:
1. Extract all line items (description, quantity, rate, amount)
2. Validate against rate cards (flag mismatches)
3. Categorize spend (labor, materials, software, travel, other)
4. Calculate variance from expected costs
5. Identify anomalies or red flags

Output structured JSON with findings.
```

**Model:** Claude Sonnet 4.5

### Success Metrics
- **Invoice Processing:** 95% accuracy in data extraction
- **Variance Detection:** 100% of >10% variances flagged
- **Time Savings:** 50% reduction in invoice review time
- **Cost Avoidance:** $50K+ in billing errors caught per year

---

## Module 3: Risk & Issue Management

### Purpose
Proactive risk identification through AI-powered analysis of all program artifacts. Maintains risk register with conversation threads and mitigation tracking.

### Key Features

**Risk Register**
- Manual risk entry with scoring (probability × impact)
- Risk categorization (technical, financial, schedule, resource, external)
- Ownership assignment
- Status tracking (open, monitoring, mitigated, closed, realized)
- Mitigation strategy documentation

**AI Risk Detection**
- Automatically identify risks from newly uploaded artifacts
- Severity assessment based on context
- Link risks to source artifacts for traceability
- Suggest mitigation strategies

**Conversation Threads**
- Discussion threads per risk
- Stakeholder mentions and notifications
- Attachment of supporting artifacts
- Status change tracking

**Impact Analysis**
- Cross-reference with milestones, budget, decisions
- Identify cascading risks
- Track risk trends over time

### User Workflows

**Primary: AI-Detected Risk**
1. User uploads meeting notes mentioning "vendor may miss deadline"
2. AI detects risk indicator during artifact analysis
3. System creates risk suggestion with:
   - Title: "Vendor XYZ delivery delay risk"
   - Probability: High
   - Impact: Medium
   - Source: Linked to meeting notes artifact
4. User reviews AI suggestion
5. User accepts → creates risk in register
6. User assigns owner, sets mitigation strategy
7. System monitors for related artifacts automatically

**Secondary: Manual Risk Creation**
1. User creates risk "Budget overrun in Q2"
2. System suggests related artifacts (invoices, budget reports)
3. User links artifacts to provide context
4. User adds comment thread with mitigation discussion
5. Stakeholders mentioned are notified
6. Weekly review: user updates risk status

### Data Model

```sql
CREATE TABLE risks (
    risk_id UUID PRIMARY KEY,
    program_id UUID,
    risk_number VARCHAR(50) UNIQUE,
    title VARCHAR(500),
    description TEXT,
    risk_category VARCHAR(100),
    probability VARCHAR(50), -- very_low to very_high
    impact VARCHAR(50),
    risk_score INTEGER, -- Calculated: probability × impact (1-25)
    status VARCHAR(50),
    owner_id UUID,
    mitigation_strategy TEXT,
    is_ai_suggested BOOLEAN,
    ai_confidence_score DECIMAL(5,4),
    artifact_ids UUID[], -- Links to source artifacts
    identified_date DATE,
    target_closure_date DATE
);

CREATE TABLE issues (
    issue_id UUID PRIMARY KEY,
    program_id UUID,
    issue_number VARCHAR(50),
    title VARCHAR(500),
    description TEXT,
    priority VARCHAR(50),
    severity VARCHAR(50),
    status VARCHAR(50),
    related_risk_id UUID REFERENCES risks,
    resolution TEXT
);

-- Unified conversation pattern (used across modules)
CREATE TABLE conversation_threads (
    thread_id UUID PRIMARY KEY,
    program_id UUID,
    entity_type VARCHAR(50), -- 'risk', 'issue', 'decision', etc.
    entity_id UUID,
    is_locked BOOLEAN DEFAULT false
);

CREATE TABLE conversation_messages (
    message_id UUID PRIMARY KEY,
    thread_id UUID REFERENCES conversation_threads,
    message_body TEXT,
    message_type VARCHAR(50), -- comment, status_change, ai_suggestion
    mentioned_user_ids UUID[],
    posted_by UUID,
    posted_at TIMESTAMPTZ
);
```

### AI Integration

**Risk Detection Prompt:**
```
System: You are a risk management specialist.

Existing Risks: {{active_risks}}
Recent Context: {{recent_artifacts_summary}}

Task: Given this new artifact, identify:
1. New potential risks (with severity, likelihood)
2. Links to existing risks (mitigation, escalation, new context)
3. Suggested risk categories and ownership
4. Recommended immediate actions if critical

Be proactive but not alarmist.
```

**Model:** Claude Opus 4.5 (deep analysis required)

### Success Metrics
- **Proactive Detection:** 30+ days earlier risk identification on average
- **AI Accuracy:** >75% of AI-suggested risks deemed valid
- **Coverage:** Zero missed risks due to buried information
- **Resolution:** 90% of risks mitigated or closed within target date

---

## Module 4: Communications Module

### Purpose
Template-based communication authoring with AI assistance. Ensures consistent, context-aware messaging to stakeholders while maintaining records of all program communications.

### Key Features

**Communication Plans**
- Define routine communications (weekly status, monthly steering committee)
- Set frequency, audience, channel
- Schedule upcoming communications
- Track completion

**Template Library**
- Create templates with structure guidelines
- Define must-do / never-do rules
- Variable placeholders ({{stakeholder_name}}, {{report_date}})
- Tone and style specifications

**AI-Assisted Drafting**
- Generate drafts from templates incorporating program context
- Pull latest data from financial, risk, milestone modules
- Adapt tone for audience (executive vs tactical)
- Provide 2 versions: concise and detailed

**Communication Records**
- Store all sent communications
- Link to stakeholder recipients
- Track open/read status
- Search historical communications

**Template Creation**
- Create new template from any finished communication
- Extract structure and style patterns

### User Workflows

**Primary: Routine Communication**
1. System notifies: "Weekly status update due Friday"
2. User clicks "Draft communication"
3. AI generates draft incorporating:
   - Latest health score
   - New risks identified this week
   - Budget status
   - Upcoming milestones
4. User reviews, edits tone or adds details
5. User selects stakeholder groups to receive
6. Send or schedule for later
7. Communication stored with links to source data

**Secondary: Ad-Hoc Communication**
1. User needs to notify executives about critical risk
2. Selects template "Executive Alert"
3. AI drafts message with risk details, impact, recommended actions
4. User edits, selects executive stakeholders
5. Send immediately
6. Communication linked to risk in system

### Data Model

```sql
CREATE TABLE communication_plans (
    plan_id UUID PRIMARY KEY,
    program_id UUID,
    name VARCHAR(255),
    description TEXT,
    frequency VARCHAR(50), -- daily, weekly, monthly, quarterly
    next_scheduled_date DATE,
    status VARCHAR(50)
);

CREATE TABLE communication_templates (
    template_id UUID PRIMARY KEY,
    program_id UUID,
    name VARCHAR(255),
    template_type VARCHAR(50), -- email, report, presentation
    subject_template TEXT,
    body_template TEXT,
    template_variables JSONB, -- Variable definitions
    style_guidelines TEXT,
    is_default BOOLEAN
);

CREATE TABLE communications (
    communication_id UUID PRIMARY KEY,
    program_id UUID,
    plan_id UUID REFERENCES communication_plans,
    template_id UUID REFERENCES communication_templates,
    subject TEXT,
    body TEXT,
    communication_type VARCHAR(50),
    status VARCHAR(50), -- draft, scheduled, sent
    ai_generated BOOLEAN,
    scheduled_send_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    created_by UUID
);

CREATE TABLE communication_recipients (
    communication_id UUID REFERENCES communications,
    stakeholder_id UUID REFERENCES stakeholders,
    status VARCHAR(50), -- pending, sent, opened
    opened_at TIMESTAMPTZ
);
```

### AI Integration

**Communication Drafting Prompt:**
```
System: You are a professional communications specialist for program governance.

Communication Plan: {{comm_plan}}
Template: {{template}}
Program Status: {{current_status}}

Task: Draft a {{communication_type}} for {{audience}} that:
- Follows the template structure
- Incorporates relevant program updates
- Maintains appropriate tone for audience
- Highlights key decisions/milestones/risks
- Includes actionable next steps

Provide concise and detailed versions.
```

**Model:** Claude Sonnet 4.5

### Success Metrics
- **Time Savings:** 60% reduction in communication drafting time
- **Consistency:** 100% adherence to template structure
- **Accuracy:** <5% of AI-drafted facts require correction
- **Engagement:** 80%+ open rate on executive communications

---

## Module 5: Stakeholder Management

### Purpose
Maintain comprehensive stakeholder dossiers with taxonomy, context notes, and engagement tracking. Automatically extract stakeholder mentions from artifacts.

### Key Features

**Stakeholder Profiles**
- Basic info (name, email, title, organization)
- Classification (internal, external, vendor, customer)
- Influence level (high, medium, low)
- Interest level (high, medium, low)
- Communication preferences

**Taxonomy & Groups**
- Group stakeholders (Executive Steering Committee, Project Team, Vendors)
- Multiple group memberships
- Group-level communication enrollment

**Context Notes**
- Add notes about stakeholder (concerns, preferences, history)
- Link notes to specific artifacts or communications
- AI sentiment analysis from communications

**Engagement Tracking**
- Communication enrollment (which routine comms they receive)
- Communication history per stakeholder
- Engagement metrics (responsiveness, sentiment)

**AI Extraction**
- Automatically extract person mentions from artifacts
- Match to existing stakeholders or suggest new entries
- Suggest classifications based on context

### User Workflows

**Primary: Auto-Discovery**
1. User uploads meeting notes mentioning "John Smith from Acme Corp expressed concerns about timeline"
2. AI extracts person: John Smith, Acme Corp
3. System checks if stakeholder exists
4. Not found → suggests creating stakeholder
5. User accepts, AI suggests: Classification: Vendor, Influence: High
6. User adds note: "Concerned about timeline, needs weekly updates"
7. User enrolls in "Weekly Vendor Status" communication plan

**Secondary: Manual Management**
1. User creates stakeholder "Jane Doe, Executive Sponsor"
2. Classifies as: Internal, Influence: High, Interest: Medium
3. Adds to group "Executive Steering Committee"
4. Enrolls in "Monthly Steering Committee Report"
5. Adds context note: "Prefers data-driven insights, avoid technical jargon"

### Data Model

```sql
CREATE TABLE stakeholders (
    stakeholder_id UUID PRIMARY KEY,
    program_id UUID,
    full_name VARCHAR(255),
    email VARCHAR(255),
    title VARCHAR(200),
    organization VARCHAR(255),
    stakeholder_type VARCHAR(100), -- internal, external, vendor, customer
    influence_level VARCHAR(50),
    interest_level VARCHAR(50),
    communication_preferences JSONB,
    user_id UUID REFERENCES users -- If they're also a system user
);

CREATE TABLE stakeholder_groups (
    group_id UUID PRIMARY KEY,
    program_id UUID,
    name VARCHAR(255),
    description TEXT,
    group_type VARCHAR(100)
);

CREATE TABLE stakeholder_group_members (
    group_id UUID REFERENCES stakeholder_groups,
    stakeholder_id UUID REFERENCES stakeholders,
    added_at TIMESTAMPTZ,
    removed_at TIMESTAMPTZ
);

CREATE TABLE stakeholder_notes (
    note_id UUID PRIMARY KEY,
    stakeholder_id UUID REFERENCES stakeholders,
    note_text TEXT,
    note_type VARCHAR(50), -- general, meeting, concern, preference
    artifact_ids UUID[],
    created_by UUID,
    created_at TIMESTAMPTZ
);

CREATE TABLE communication_enrollments (
    enrollment_id UUID PRIMARY KEY,
    plan_id UUID REFERENCES communication_plans,
    stakeholder_id UUID REFERENCES stakeholders,
    enrolled_at TIMESTAMPTZ,
    unsubscribed_at TIMESTAMPTZ
);
```

### AI Integration

**Person Extraction Prompt:**
```
System: You are an expert in stakeholder analysis.

Existing Stakeholders: {{stakeholders}}

Task: From this artifact:
1. Extract all person mentions with context
2. Match to existing stakeholders (if applicable)
3. Suggest stakeholder classification
4. Identify influence level and engagement frequency
5. Recommend engagement approach

Output structured JSON.
```

**Model:** Claude Sonnet 4.5

### Success Metrics
- **Auto-Discovery:** 70% of stakeholders identified automatically from artifacts
- **Classification Accuracy:** >80% of AI classifications accepted
- **Engagement:** 90%+ of key stakeholders have documented notes
- **Coverage:** Zero missed stakeholders in governance meetings

---

## Module 6: Decision Log

### Purpose
Capture and track all program decisions with impact analysis. AI automatically extracts decisions from meeting notes and links them to affected program areas.

### Key Features

**Decision Capture**
- Manual decision entry
- AI extraction from meeting notes
- Decision numbering (DEC-001, DEC-002)
- Decision metadata (date, maker, participants, rationale)

**Impact Tracking**
- Document impacted areas (scope, schedule, budget, risk)
- Link to artifacts that influenced decision
- Link to affected milestones, risks, change requests
- Alternatives considered

**Approval Workflow**
- Multi-level approvals for major decisions
- Participant tracking (consulted, informed, approver)
- Approval status and notes

**AI Impact Analysis**
- Assess decision impacts across program dimensions
- Identify dependencies on other decisions
- Suggest review dates

### User Workflows

**Primary: AI Extraction**
1. User uploads steering committee meeting notes
2. AI extracts decision: "Approved budget increase of $500K for Phase 2"
3. System creates decision suggestion with:
   - Decision maker: Executive Sponsor
   - Impact: Budget (+$500K), Schedule (no change)
   - Rationale: "Additional security requirements"
4. User reviews and accepts
5. Decision added to log, linked to meeting notes artifact
6. Impacted budget categories automatically adjusted

**Secondary: Manual Entry**
1. User creates decision "Defer Feature X to Phase 3"
2. Documents rationale, alternatives considered
3. Links to risk "Resource constraints" that drove decision
4. Tags impacted milestones
5. Sets review date (end of Phase 2)
6. Notifies affected stakeholders

### Data Model

```sql
CREATE TABLE decisions (
    decision_id UUID PRIMARY KEY,
    program_id UUID,
    decision_number VARCHAR(50) UNIQUE,
    title VARCHAR(500),
    description TEXT,
    decision_category VARCHAR(100), -- technical, financial, organizational, strategic
    rationale TEXT,
    alternatives_considered TEXT,
    decision_maker_id UUID,
    approved_by_ids UUID[],
    impact_description TEXT,
    impacted_modules VARCHAR(100)[],
    decision_status VARCHAR(50), -- proposed, approved, rejected, superseded
    decision_date DATE,
    effective_date DATE,
    review_date DATE,
    artifact_ids UUID[],
    related_risk_ids UUID[],
    superseded_by UUID REFERENCES decisions
);

CREATE TABLE decision_participants (
    participant_id UUID PRIMARY KEY,
    decision_id UUID REFERENCES decisions,
    stakeholder_id UUID REFERENCES stakeholders,
    participation_role VARCHAR(50), -- consulted, informed, approver
    approval_status VARCHAR(50),
    approval_date TIMESTAMPTZ,
    approval_notes TEXT
);
```

### AI Integration

**Decision Extraction Prompt:**
```
System: You are a program decision analyst.

Program Context: {{program_overview}}
Recent Decisions: {{recent_decisions}}

Task: From this meeting note:
1. Extract all decisions made (who, what, when)
2. Identify impacted areas (scope, budget, schedule, risk, quality)
3. Link to existing decisions or artifacts
4. Assess decision urgency and significance
5. Suggest follow-up actions or approvals needed

Output structured JSON.
```

**Model:** Claude Sonnet 4.5

### Success Metrics
- **Extraction Accuracy:** >80% of decisions captured automatically
- **Traceability:** 100% of decisions linked to source artifacts
- **Impact Visibility:** 90%+ of decision impacts documented
- **Compliance:** Zero governance audit findings on decision documentation

---

## Module 7: Executive Dashboard & Health Metrics

### Purpose
Real-time program health visibility with predictive alerts. AI-powered health scoring across multiple dimensions with natural language insights.

### Key Features

**Program Health Score**
- Overall score (0-100) with red/yellow/green status
- Dimensional scores: Financial, Schedule, Risk, Quality, Stakeholder
- Trend analysis (improving, stable, declining)
- Historical tracking

**KPI Management**
- Define custom KPIs (budget variance, milestone adherence, risk count)
- Target values and thresholds
- Actual vs target tracking
- Visualization (gauges, trend lines)

**Predictive Alerts**
- Early warning system for potential issues
- AI-powered predictions (30/60/90 day horizon)
- Alert prioritization (critical, high, medium)
- Recommended leadership actions

**AI-Generated Insights**
- Natural language summaries of program status
- Key concerns highlighted
- Positive trends celebrated
- Executive action recommendations

### User Workflows

**Primary: Executive Check-In**
1. Executive opens dashboard
2. Sees overall health score: 78 (Yellow, declining)
3. AI insight: "Financial health dropped 12 points due to Q2 budget overrun. Schedule remains on track. 3 new critical risks identified this week."
4. Drill down into Financial dimension
5. See categorical spend chart highlighting overages
6. Click alert: "Risk of budget overrun in Phase 3"
7. Review AI-recommended actions
8. Assign action item to program manager

**Secondary: Predictive Alert**
1. System generates alert: "Predicted milestone delay (Milestone M5) in 45 days based on current velocity"
2. Program manager reviews
3. AI provides analysis: "Last 3 milestones delayed average 2 weeks. Vendor XYZ consistently late on deliverables."
4. Manager creates mitigation plan
5. Links plan to risk register

### Data Model

```sql
CREATE TABLE kpi_definitions (
    kpi_id UUID PRIMARY KEY,
    program_id UUID,
    kpi_name VARCHAR(255),
    kpi_code VARCHAR(50),
    unit_of_measure VARCHAR(50),
    target_value DECIMAL(15,4),
    threshold_red DECIMAL(15,4),
    threshold_yellow DECIMAL(15,4),
    threshold_green DECIMAL(15,4),
    higher_is_better BOOLEAN,
    measurement_frequency VARCHAR(50)
);

CREATE TABLE kpi_measurements (
    measurement_id UUID PRIMARY KEY,
    kpi_id UUID REFERENCES kpi_definitions,
    measurement_date DATE,
    measured_value DECIMAL(15,4),
    status VARCHAR(50), -- green, yellow, red
    notes TEXT,
    recorded_by UUID
);

CREATE TABLE health_scores (
    score_id UUID PRIMARY KEY,
    program_id UUID,
    score_date DATE,
    overall_score INTEGER CHECK (overall_score BETWEEN 0 AND 100),
    overall_status VARCHAR(50),
    schedule_score INTEGER,
    budget_score INTEGER,
    risk_score INTEGER,
    quality_score INTEGER,
    trend VARCHAR(50), -- improving, stable, declining
    ai_summary TEXT,
    key_concerns TEXT[]
);
```

### AI Integration

**Health Scoring Prompt:**
```
System: You are a program health analyst.

Program Data:
- Financial: {{financial_summary}}
- Schedule: {{schedule_status}}
- Risks: {{risk_profile}}
- Decisions: {{recent_decisions}}

Task: Generate:
1. Overall health score (0-100) with reasoning
2. Category scores (financial, schedule, risk, quality)
3. Trend analysis (improving, stable, declining)
4. Predictive alerts (potential issues in next 30/60/90 days)
5. Recommended leadership actions

Be data-driven and specific.
```

**Model:** Claude Opus 4.5 (deep analysis)

### Success Metrics
- **Accuracy:** Health scores correlate >90% with actual program outcomes
- **Early Warning:** Alerts predict issues 30+ days in advance
- **Executive Adoption:** 100% of executives check dashboard weekly
- **Actionability:** >80% of AI recommendations acted upon

---

## Module 8: Milestone & Phase Management

### Purpose
High-level program timeline management with phases, milestones, and gates. Focus on strategic oversight, not detailed task management.

### Key Features

**Phase Management**
- Define major program phases (Initiation, Planning, Execution, Closure)
- Phase sequencing with dependencies
- Planned vs actual dates
- Phase status tracking

**Milestone Tracking**
- Key deliverables, gates, reviews, approvals
- Target dates and actual completion
- Success criteria
- Owner assignment
- Status (pending, at_risk, achieved, missed)

**Gate Management**
- Go/no-go decision points
- Entry and exit criteria
- Gate status (pending, passed, failed, conditional)
- Link to decisions

**Dependencies**
- Cross-milestone dependencies
- Critical path identification

### User Workflows

**Primary: Milestone Tracking**
1. User views program timeline
2. Milestone "M3: Security Review" is marked at_risk
3. User clicks milestone for details
4. See: Target date March 31, current date March 28, status: Not started
5. AI alert: "Risk of delay based on pending security documentation"
6. User creates action item, assigns to security lead
7. When completed, mark milestone achieved

**Secondary: Gate Review**
1. Gate G2 scheduled for end of Phase 2
2. System compiles gate package:
   - Phase 2 deliverables status
   - Budget health
   - Risk profile
   - Stakeholder feedback
3. Steering committee reviews
4. Decision recorded: "Pass with conditions: address security concerns in Phase 3"
5. Gate status updated, program advances to Phase 3

### Data Model

```sql
CREATE TABLE phases (
    phase_id UUID PRIMARY KEY,
    program_id UUID,
    phase_name VARCHAR(255),
    phase_number INTEGER,
    description TEXT,
    planned_start_date DATE,
    planned_end_date DATE,
    actual_start_date DATE,
    actual_end_date DATE,
    status VARCHAR(50), -- planned, active, completed, on_hold
    predecessor_phase_id UUID REFERENCES phases
);

CREATE TABLE milestones (
    milestone_id UUID PRIMARY KEY,
    program_id UUID,
    phase_id UUID REFERENCES phases,
    milestone_name VARCHAR(500),
    description TEXT,
    milestone_type VARCHAR(50), -- deliverable, gate, review, approval
    planned_date DATE,
    actual_date DATE,
    status VARCHAR(50), -- pending, at_risk, achieved, missed
    success_criteria TEXT,
    owner_id UUID,
    artifact_ids UUID[]
);

CREATE TABLE gates (
    gate_id UUID PRIMARY KEY,
    program_id UUID,
    phase_id UUID REFERENCES phases,
    gate_name VARCHAR(255),
    gate_type VARCHAR(50), -- go_no_go, stage_gate, quality_gate
    scheduled_date DATE,
    actual_date DATE,
    entry_criteria TEXT,
    exit_criteria TEXT,
    gate_status VARCHAR(50), -- pending, passed, failed, conditional
    decision_id UUID REFERENCES decisions
);
```

### AI Integration

**Schedule Risk Prompt:**
```
System: You are a program schedule analyst.

Program Timeline: {{timeline}}
Recent Milestones: {{milestone_history}}
Current Velocity: {{completion_rates}}

Task: Analyze schedule health:
1. Identify at-risk milestones
2. Predict likely delays with confidence
3. Assess critical path impact
4. Recommend mitigation actions

Output structured risk assessment.
```

**Model:** Claude Sonnet 4.5

### Success Metrics
- **Milestone Achievement:** >85% of milestones achieved on time
- **Prediction Accuracy:** >70% of predicted delays materialize
- **Visibility:** Zero surprises at gates (all issues flagged in advance)
- **Decision Quality:** 100% of gate decisions documented

---

## Module 9: Governance Framework & Compliance

### Purpose
Track governance cadence and ensure audit readiness. Maintains compliance requirements with evidence collection and audit trail analysis.

### Key Features

**Governance Cadence**
- Define recurring governance meetings (steering committee, reviews)
- Track meeting schedules and attendance
- Link meeting notes artifacts
- Action item tracking from meetings

**Compliance Requirements**
- Define compliance obligations (SOX, GDPR, internal policies)
- Track compliance status per requirement
- Evidence collection and linking
- Audit readiness scoring

**Audit Trail**
- Universal audit log of all system changes
- Search and filter by entity, user, date
- Compliance reporting
- Export for external audits

**Meeting Management**
- Meeting templates and agendas
- Attendance tracking
- Action items with owners and due dates
- Meeting notes AI summarization

### User Workflows

**Primary: Governance Meeting**
1. Monthly steering committee meeting scheduled
2. System generates agenda from template:
   - Health score review
   - Risk updates
   - Decision approvals
   - Budget status
3. Meeting held, notes uploaded as artifact
4. AI extracts:
   - Attendance
   - Decisions made
   - Action items
5. System links decisions to decision log
6. Action items assigned to owners
7. Meeting marked complete

**Secondary: Compliance Tracking**
1. User defines compliance requirement "SOC 2 - Access Control"
2. Specifies evidence needed: Access logs, user reviews
3. Links evidence artifacts as they're created
4. Quarterly audit: system shows compliance percentage per requirement
5. Export compliance report for auditors

### Data Model

```sql
CREATE TABLE governance_cadences (
    cadence_id UUID PRIMARY KEY,
    program_id UUID,
    name VARCHAR(255), -- "Monthly Steering Committee"
    frequency VARCHAR(50), -- weekly, monthly, quarterly
    meeting_duration_minutes INTEGER,
    required_participants UUID[],
    agenda_template TEXT,
    is_active BOOLEAN
);

CREATE TABLE governance_meetings (
    meeting_id UUID PRIMARY KEY,
    cadence_id UUID REFERENCES governance_cadences,
    program_id UUID,
    meeting_date DATE,
    attendee_user_ids UUID[],
    artifact_id UUID REFERENCES artifacts, -- Meeting notes
    status VARCHAR(50), -- scheduled, completed, cancelled
    ai_summary TEXT
);

CREATE TABLE compliance_requirements (
    requirement_id UUID PRIMARY KEY,
    program_id UUID,
    requirement_code VARCHAR(50),
    requirement_name VARCHAR(255),
    description TEXT,
    compliance_domain VARCHAR(100), -- sox, gdpr, iso27001
    validation_frequency VARCHAR(50),
    status VARCHAR(50),
    evidence_artifact_ids UUID[],
    last_audit_date DATE,
    next_audit_date DATE
);

CREATE TABLE audit_logs (
    audit_id UUID PRIMARY KEY,
    program_id UUID,
    entity_type VARCHAR(50),
    entity_id UUID,
    action VARCHAR(50), -- created, updated, deleted
    changed_fields JSONB,
    changed_by UUID,
    changed_at TIMESTAMPTZ,
    ip_address INET
);
```

### AI Integration

**Meeting Summarization Prompt:**
```
System: You are a meeting analyst.

Meeting Notes: {{meeting_artifact}}

Task: Extract:
1. Key decisions made
2. Action items with owners
3. Risks or concerns discussed
4. Budget or schedule impacts mentioned
5. Attendance (if mentioned)

Provide executive summary.
```

**Model:** Claude Sonnet 4.5

### Success Metrics
- **Governance Adherence:** 100% of scheduled meetings held or documented
- **Compliance:** >95% compliance score on all requirements
- **Audit Readiness:** Zero findings in external audits
- **Traceability:** 100% of decisions traceable to governance meetings

---

## Module 10: Change Control Board

### Purpose
Manage program-level changes (scope, schedule, budget) with AI-powered impact analysis and approval workflows.

### Key Features

**Change Request Management**
- Submit change requests with justification
- Change categorization (scope, schedule, budget, resource)
- Change type (major, minor, emergency)
- Priority assignment

**AI Impact Analysis**
- Automated analysis across all program dimensions
- Schedule impact assessment
- Budget impact calculation
- Risk assessment
- Dependency identification

**Approval Workflow**
- Multi-level approvals based on change magnitude
- Approval status tracking
- Conditional approvals
- Approval notes and rationale

**Implementation Tracking**
- Implementation plan documentation
- Status tracking
- Verification that change was implemented

### User Workflows

**Primary: Change Request**
1. User submits change request: "Add Feature Y to Phase 2"
2. Documents justification: "Customer requirement, strategic value"
3. AI performs impact analysis:
   - Schedule: +3 weeks to Phase 2 completion
   - Budget: +$150K estimated
   - Risks: Resource conflict with Feature X
   - Dependencies: Requires decision on vendor selection
4. System routes to approvers based on impact magnitude
5. CCB reviews with AI analysis
6. Decision: Approved with condition "Defer Feature X to Phase 3"
7. Decision logged, milestones adjusted, budget updated
8. Implementation plan created

**Secondary: Emergency Change**
1. Critical security patch needed immediately
2. User submits emergency change request
3. AI fast-tracks analysis
4. Alerts executive approver
5. Approved within 4 hours
6. Change implemented, post-implementation review scheduled

### Data Model

```sql
CREATE TABLE change_requests (
    change_request_id UUID PRIMARY KEY,
    program_id UUID,
    change_number VARCHAR(50) UNIQUE,
    title VARCHAR(500),
    description TEXT,
    change_category VARCHAR(100), -- scope, schedule, budget, resource
    change_type VARCHAR(50), -- major, minor, emergency
    priority VARCHAR(50),
    business_justification TEXT,
    schedule_impact_days INTEGER,
    budget_impact_amount DECIMAL(15,2),
    resource_impact TEXT,
    risk_impact TEXT,
    impacted_milestones UUID[],
    requestor_id UUID,
    status VARCHAR(50), -- submitted, under_review, pending_approval, approved, rejected
    decision_id UUID REFERENCES decisions,
    artifact_ids UUID[]
);

CREATE TABLE change_request_approvers (
    approver_id UUID PRIMARY KEY,
    change_request_id UUID REFERENCES change_requests,
    user_id UUID REFERENCES users,
    approval_level INTEGER,
    approval_status VARCHAR(50), -- pending, approved, rejected
    approval_date TIMESTAMPTZ,
    approval_notes TEXT,
    required BOOLEAN DEFAULT true
);
```

### AI Integration

**Impact Analysis Prompt:**
```
System: You are a change impact analyst.

Program Artifacts: {{artifacts_summary}}
Current Baselines: {{baselines}}
Change Request: {{change_details}}

Task: Assess impact across:
1. Schedule (milestones affected, critical path)
2. Budget (cost increase/decrease, funding sources)
3. Resources (team capacity, vendor contracts)
4. Risks (new risks introduced, existing risks mitigated)
5. Dependencies (other changes, decisions, approvals)

Recommend approval pathway and conditions.
```

**Model:** Claude Opus 4.5 (complex analysis)

### Success Metrics
- **Analysis Accuracy:** >85% of AI impact assessments validated by CCB
- **Approval Velocity:** <5 days average approval time for non-emergency changes
- **Decision Quality:** <10% of approved changes require corrective action
- **Traceability:** 100% of changes linked to decisions and impacts documented

---

## Cross-Module Integration

### Event-Driven Integration

All modules subscribe to events from the Artifacts module and publish their own events:

```
Artifacts → artifact.analyzed
  ↓
Financial → Checks for invoices → financial.variance_detected
  ↓
Risk → Creates risk from variance → risk.identified
  ↓
Dashboard → Updates health score → dashboard.health_updated
  ↓
Stakeholder → Notifies budget owner → communication.sent
```

### Shared Patterns

**Conversation Threads**
- Risks, Issues, Decisions, Change Requests all use unified conversation pattern
- Consistent UX for discussions across modules

**Artifact Linking**
- All entities can link to artifacts via `artifact_ids UUID[]`
- Provides context and traceability

**Audit Trail**
- All changes logged in centralized `audit_logs` table
- Universal compliance and debugging

---

**Document Version:** 1.0
**Last Updated:** 2026-01-10
**Document Owner:** Product Owner
**Status:** Approved for Implementation

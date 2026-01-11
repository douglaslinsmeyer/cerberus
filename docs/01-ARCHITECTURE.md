# Cerberus Technical Architecture

## Technology Stack

### Backend
- **Language:** Go 1.22+
- **Framework:** Standard library + chi router
- **API Style:** RESTful with JSON
- **Authentication:** JWT tokens

### Frontend
- **Framework:** ReactJS 18+ with TypeScript
- **State Management:** React Context + TanStack Query (React Query)
- **UI Library:** Tailwind CSS + Headless UI
- **Build Tool:** Vite

### Database & Storage
- **Primary Database:** PostgreSQL 16
- **Extensions:** pgvector (vector embeddings), pg_trgm (fuzzy search)
- **Object Storage:** MinIO (S3-compatible, local dev) / AWS S3 (production)
- **Cache:** Redis 7
- **Search:** PostgreSQL full-text search (Phase 1), optional Elasticsearch (Phase 2+)

### AI / LLM
- **Provider:** Anthropic Claude API
- **Models:**
  - Claude Opus 4.5 (complex analysis, deep reasoning)
  - Claude Sonnet 4.5 (frequent operations, summaries)
- **Features:** Streaming, prompt caching, structured outputs

### Infrastructure
- **Local Development:** Docker Compose
- **Container Registry:** Docker Hub (public images) / ECR (custom images)
- **Orchestration (Production):** Kubernetes or ECS
- **CI/CD:** GitHub Actions
- **Monitoring:** Prometheus + Grafana (future)

---

## System Architecture: Modular Monolith

### Architectural Decision: Why Modular Monolith?

**Chosen:** Single deployable application with clear module boundaries

**Rationale:**
1. **Simplicity:** Easier to deploy, debug, and monitor than microservices
2. **Performance:** In-process communication is faster than network calls
3. **Transactions:** Database transactions across modules without distributed transactions
4. **Shared Context:** Artifacts module benefits from tight integration
5. **Future-Proof:** Clear module boundaries enable extraction to microservices when needed

**Trade-offs Accepted:**
- Less independent scaling per module (mitigated: horizontal scaling of entire app)
- Requires discipline to maintain module boundaries (mitigated: code organization, linting)

### Directory Structure

```
cerberus/
├── backend/
│   ├── cmd/
│   │   ├── api/
│   │   │   └── main.go              # API server entrypoint
│   │   ├── worker/
│   │   │   └── main.go              # Background AI worker
│   │   └── migrate/
│   │       └── main.go              # Database migrations CLI
│   │
│   ├── internal/
│   │   ├── modules/                 # 10 Feature Modules
│   │   │   ├── artifacts/           # ⭐ CORE CONTEXT ENGINE
│   │   │   │   ├── handler.go       # HTTP handlers
│   │   │   │   ├── service.go       # Business logic
│   │   │   │   ├── repository.go    # Data access
│   │   │   │   ├── ai.go            # AI-specific logic
│   │   │   │   └── events.go        # Event publishers/subscribers
│   │   │   ├── financial/
│   │   │   ├── risk/
│   │   │   ├── communications/
│   │   │   ├── stakeholder/
│   │   │   ├── decision/
│   │   │   ├── dashboard/
│   │   │   ├── milestone/
│   │   │   ├── governance/
│   │   │   └── changecontrol/
│   │   │
│   │   ├── platform/                # Shared Infrastructure
│   │   │   ├── ai/                  # Claude API client
│   │   │   │   ├── client.go
│   │   │   │   ├── streaming.go
│   │   │   │   ├── caching.go
│   │   │   │   └── prompts.go
│   │   │   ├── events/              # Event bus
│   │   │   │   ├── bus.go
│   │   │   │   └── types.go
│   │   │   ├── storage/             # File storage abstraction
│   │   │   │   ├── storage.go
│   │   │   │   └── minio.go
│   │   │   ├── auth/                # Authentication
│   │   │   │   ├── jwt.go
│   │   │   │   └── middleware.go
│   │   │   ├── db/                  # Database utilities
│   │   │   │   ├── connection.go
│   │   │   │   └── transactions.go
│   │   │   └── observability/       # Logging, metrics
│   │   │       ├── logger.go
│   │   │       └── metrics.go
│   │   │
│   │   ├── domain/                  # Domain Models
│   │   │   ├── program.go
│   │   │   ├── artifact.go
│   │   │   ├── risk.go
│   │   │   └── ...
│   │   │
│   │   └── api/                     # API Layer
│   │       ├── router.go
│   │       ├── middleware.go
│   │       └── responses.go
│   │
│   ├── migrations/                  # Database migrations
│   │   ├── 001_foundation.sql
│   │   ├── 002_artifacts.sql
│   │   └── ...
│   │
│   ├── pkg/                         # Public libraries (if needed)
│   │
│   ├── go.mod
│   └── go.sum
│
├── frontend/
│   ├── src/
│   │   ├── modules/                 # Feature modules
│   │   │   ├── artifacts/
│   │   │   │   ├── components/
│   │   │   │   ├── hooks/
│   │   │   │   ├── services/
│   │   │   │   └── pages/
│   │   │   ├── financial/
│   │   │   └── ...
│   │   │
│   │   ├── layouts/                 # Layout components
│   │   │   └── CockpitLayout/
│   │   │       ├── index.tsx
│   │   │       ├── Sidebar.tsx
│   │   │       ├── CommandBar.tsx
│   │   │       ├── ContextPanel.tsx
│   │   │       └── StatusBar.tsx
│   │   │
│   │   ├── components/              # Shared components
│   │   │   ├── Button/
│   │   │   ├── Modal/
│   │   │   ├── Table/
│   │   │   └── ...
│   │   │
│   │   ├── contexts/                # React contexts
│   │   │   ├── AuthContext.tsx
│   │   │   ├── ProgramContext.tsx
│   │   │   └── AIContext.tsx
│   │   │
│   │   ├── services/                # API clients
│   │   │   └── api.ts
│   │   │
│   │   ├── App.tsx
│   │   └── main.tsx
│   │
│   ├── package.json
│   └── tsconfig.json
│
├── docker/
│   ├── Dockerfile.api
│   ├── Dockerfile.worker
│   └── Dockerfile.web
│
├── docker-compose.yml
│
└── docs/
    ├── 00-OVERVIEW.md
    ├── 01-ARCHITECTURE.md
    └── ...
```

---

## Module Integration: Event-Driven Context Propagation

### Core Principle

**Artifacts module is the HUB** that publishes events when documents are processed. Other modules are SPOKES that subscribe and react.

### Event Flow Diagram

```
┌──────────────────────────────────────────────┐
│        ARTIFACTS MODULE (CORE HUB)           │
│                                              │
│  1. User uploads invoice PDF                 │
│  2. AI extracts metadata (vendor, amount)    │
│  3. Publishes: artifact.analyzed             │
└────────────┬─────────────────────────────────┘
             │
             ├─────────────────┬────────────────┬─────────────────┐
             ▼                 ▼                ▼                 ▼
    ┌────────────────┐ ┌───────────────┐ ┌──────────────┐ ┌─────────────┐
    │   FINANCIAL    │ │   RISK        │ │  STAKEHOLDER │ │  DASHBOARD  │
    │                │ │               │ │              │ │             │
    │ Subscribes to  │ │ Subscribes to │ │ Subscribes to│ │ Subscribes  │
    │ artifact.      │ │ artifact.     │ │ artifact.    │ │ to all      │
    │ analyzed       │ │ analyzed      │ │ analyzed     │ │ events      │
    │                │ │               │ │              │ │             │
    │ Checks: Is     │ │ Checks: Any   │ │ Checks: Any  │ │ Updates     │
    │ this an        │ │ risk          │ │ persons      │ │ program     │
    │ invoice?       │ │ indicators?   │ │ mentioned?   │ │ health      │
    │                │ │               │ │              │ │ score       │
    │ YES →          │ │ YES →         │ │ YES →        │ │             │
    │ Create         │ │ Suggest new   │ │ Link to      │ │             │
    │ invoice        │ │ risk          │ │ stakeholder  │ │             │
    │ record         │ │               │ │              │ │             │
    │                │ │               │ │              │ │             │
    │ Detect         │ │               │ │              │ │             │
    │ variance →     │ │               │ │              │ │             │
    │ Publish:       │ │               │ │              │ │             │
    │ financial.     │ │               │ │              │ │             │
    │ variance       │ │               │ │              │ │             │
    │ _detected      │ │               │ │              │ │             │
    └────────────────┘ └───────────────┘ └──────────────┘ └─────────────┘
```

### Event Types

```go
// internal/platform/events/types.go

type EventType string

const (
    // Artifact events
    ArtifactUploaded          EventType = "artifact.uploaded"
    ArtifactAnalyzed          EventType = "artifact.analyzed"
    ArtifactMetadataExtracted EventType = "artifact.metadata_extracted"

    // Financial events
    InvoiceProcessed        EventType = "financial.invoice_processed"
    VarianceDetected        EventType = "financial.variance_detected"
    BudgetThresholdExceeded EventType = "financial.budget_exceeded"

    // Risk events
    RiskIdentified  EventType = "risk.identified"
    RiskEscalated   EventType = "risk.escalated"
    IssueCreated    EventType = "risk.issue_created"

    // Decision events
    DecisionExtracted EventType = "decision.extracted"
    DecisionApproved  EventType = "decision.approved"

    // Change events
    ChangeProposed EventType = "change.proposed"
    ChangeApproved EventType = "change.approved"
)

type Event struct {
    ID            string                 `json:"id"`
    Type          EventType              `json:"type"`
    ProgramID     string                 `json:"program_id"`
    Timestamp     time.Time              `json:"timestamp"`
    Source        string                 `json:"source"`        // Module name
    Payload       map[string]interface{} `json:"payload"`
    CorrelationID string                 `json:"correlation_id"`// Trace workflows
    Metadata      EventMetadata          `json:"metadata"`
}

type EventMetadata struct {
    AIGenerated  bool     `json:"ai_generated"`
    Confidence   float64  `json:"confidence"`
    ArtifactRefs []string `json:"artifact_refs"`
}
```

### Event Bus Implementation

**Phase 1 (MVP):** PostgreSQL-backed event store
- Events stored in `events` table
- Background worker polls for unprocessed events
- Simple, reliable, no additional infrastructure

**Phase 2 (Scale):** NATS or Redis Streams
- Real-time event streaming
- Better concurrency for high-volume processing

```go
// internal/platform/events/bus.go

type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(eventType EventType, handler EventHandler) error
    Start(ctx context.Context) error
}

type EventHandler func(ctx context.Context, event Event) error

// PostgreSQL implementation
type PostgresEventBus struct {
    db       *sql.DB
    handlers map[EventType][]EventHandler
    mu       sync.RWMutex
}

func (b *PostgresEventBus) Publish(ctx context.Context, event Event) error {
    // Insert event into events table
    _, err := b.db.ExecContext(ctx, `
        INSERT INTO events (id, type, program_id, source, payload, correlation_id, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, event.ID, event.Type, event.ProgramID, event.Source,
       event.Payload, event.CorrelationID, event.Metadata)
    return err
}

func (b *PostgresEventBus) Subscribe(eventType EventType, handler EventHandler) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.handlers[eventType] = append(b.handlers[eventType], handler)
    return nil
}

func (b *PostgresEventBus) Start(ctx context.Context) error {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            b.processEvents(ctx)
        }
    }
}

func (b *PostgresEventBus) processEvents(ctx context.Context) {
    // Poll for unprocessed events
    rows, _ := b.db.QueryContext(ctx, `
        SELECT id, type, program_id, source, payload, correlation_id, metadata
        FROM events
        WHERE processed = false
        ORDER BY timestamp ASC
        LIMIT 100
    `)

    // Process each event by calling registered handlers
    // Mark as processed when done
}
```

---

## Claude API Integration Architecture

### Centralized AI Platform Layer

```
┌─────────────────────────────────────────────────┐
│         PLATFORM AI LAYER                        │
│      (internal/platform/ai/)                     │
│                                                  │
│  ┌────────────────────────────────────────────┐ │
│  │    Claude API Client (Singleton)           │ │
│  │  • HTTP client with retry logic            │ │
│  │  • Rate limiting (per program)             │ │
│  │  • Cost tracking                           │ │
│  │  • Response caching (Redis)                │ │
│  │  • Streaming support                       │ │
│  └────────────────────────────────────────────┘ │
│                     │                            │
│  ┌──────────────────┴──────────────────────────┐│
│  │    Prompt Template Library                  ││
│  │  • Versioned prompts per module             ││
│  │  • Variable substitution                    ││
│  │  • Output schema definitions                ││
│  └─────────────────────────────────────────────┘│
│                     │                            │
│  ┌──────────────────┴──────────────────────────┐│
│  │    Context Builder                          ││
│  │  • Assemble relevant artifacts              ││
│  │  • Apply prompt caching markers             ││
│  │  • Optimize context size                    ││
│  └─────────────────────────────────────────────┘│
└──────────────────────────────────────────────────┘
          │              │              │
          ▼              ▼              ▼
    ┌──────────┐  ┌──────────┐  ┌──────────┐
    │Artifacts │  │Financial │  │   Risk   │
    │AI Logic  │  │AI Logic  │  │AI Logic  │
    └──────────┘  └──────────┘  └──────────┘
```

### AI Client Interface

```go
// internal/platform/ai/client.go

type Client interface {
    Request(ctx context.Context, req *AIRequest) (*AIResponse, error)
    Stream(ctx context.Context, req *AIRequest) (<-chan *AIChunk, error)
}

type AIRequest struct {
    Model          string           // "claude-opus-4-5" or "claude-sonnet-4-5"
    SystemPrompt   string
    Messages       []Message
    MaxTokens      int
    Temperature    float64
    CacheControl   *CacheControl    // For prompt caching
}

type AIResponse struct {
    Content      string
    TokensUsed   TokenUsage
    CacheHit     bool
    Cost         float64
    Metadata     map[string]interface{}
}

type TokenUsage struct {
    Input          int
    Output         int
    CachedInput    int  // Tokens served from cache
}
```

### Cost Optimization Implementation

**1. Prompt Caching:**
```go
func (c *ContextBuilder) BuildPromptWithCaching(
    programID string,
    purpose string,
) (*AIRequest, error) {

    // Load static program context (refreshed daily)
    staticContext := c.cache.Get(fmt.Sprintf("program:%s:context", programID))

    return &AIRequest{
        Messages: []Message{
            {
                Role: "user",
                Content: []ContentBlock{
                    {
                        Type: "text",
                        Text: staticContext,
                        CacheControl: &CacheControl{Type: "ephemeral"}, // ⭐ Cache this
                    },
                    {
                        Type: "text",
                        Text: "NEW ARTIFACT: ...", // Not cached
                    },
                },
            },
        },
    }
}
```

**2. Response Caching (Redis):**
```go
func (c *ClaudeClient) Request(ctx context.Context, req *AIRequest) (*AIResponse, error) {
    // Generate cache key from request
    cacheKey := generateCacheKey(req)

    // Check cache
    if cached, found := c.redis.Get(cacheKey); found {
        return parseCachedResponse(cached), nil
    }

    // Make API call
    resp, err := c.httpClient.Post("https://api.anthropic.com/v1/messages", ...)

    // Cache response (1 hour TTL)
    c.redis.Set(cacheKey, resp, 1*time.Hour)

    return resp, nil
}
```

**3. Cost Tracking:**
```go
func (c *ClaudeClient) trackUsage(programID, module string, usage TokenUsage) {
    cost := calculateCost(usage)

    // Store in database
    c.db.Exec(`
        INSERT INTO ai_usage (program_id, module, tokens_input, tokens_output, cost_usd)
        VALUES ($1, $2, $3, $4, $5)
    `, programID, module, usage.Input, usage.Output, cost)

    // Check daily limit
    if c.isDailyLimitExceeded(programID) {
        c.sendAlert(programID, "Daily AI cost limit exceeded")
    }
}
```

---

## API Design

### RESTful API Structure

```
/api/v1
├── /auth
│   ├── POST   /login
│   ├── POST   /logout
│   └── POST   /refresh
│
├── /programs
│   ├── GET    /                              # List user's programs
│   ├── POST   /                              # Create program
│   ├── GET    /:programId                    # Get program details
│   ├── PATCH  /:programId                    # Update program
│   └── DELETE /:programId                    # Delete program
│
├── /programs/:programId/artifacts
│   ├── GET    /                              # List artifacts
│   ├── POST   /upload                        # Upload artifact
│   ├── GET    /:artifactId                   # Get artifact details
│   ├── GET    /:artifactId/download          # Download file
│   ├── POST   /:artifactId/analyze           # Trigger AI re-analysis
│   ├── GET    /:artifactId/metadata          # Get extracted metadata
│   ├── POST   /search                        # Semantic search
│   └── DELETE /:artifactId                   # Delete artifact
│
├── /programs/:programId/financial
│   ├── GET    /invoices                      # List invoices
│   ├── POST   /invoices                      # Create invoice
│   ├── GET    /invoices/:id                  # Get invoice
│   ├── GET    /rate-cards                    # List rate cards
│   ├── POST   /rate-cards                    # Create rate card
│   ├── GET    /variance-analysis             # AI variance report
│   └── GET    /spend-summary                 # Categorical spend
│
├── /programs/:programId/risks
│   ├── GET    /                              # List risks
│   ├── POST   /                              # Create risk
│   ├── GET    /:id                           # Get risk details
│   ├── PATCH  /:id                           # Update risk
│   ├── POST   /:id/comments                  # Add comment
│   ├── GET    /:id/comments                  # Get comments
│   └── POST   /suggest                       # AI risk suggestions
│
├── /programs/:programId/communications
│   ├── GET    /plans                         # List comm plans
│   ├── POST   /plans                         # Create plan
│   ├── GET    /templates                     # List templates
│   ├── POST   /templates                     # Create template
│   ├── POST   /                              # Create communication
│   ├── POST   /generate                      # AI-generate draft
│   └── POST   /:id/send                      # Send communication
│
├── /programs/:programId/stakeholders
│   ├── GET    /                              # List stakeholders
│   ├── POST   /                              # Create stakeholder
│   ├── GET    /:id                           # Get stakeholder
│   ├── PATCH  /:id                           # Update stakeholder
│   ├── GET    /groups                        # List groups
│   └── POST   /groups                        # Create group
│
├── /programs/:programId/decisions
│   ├── GET    /                              # List decisions
│   ├── POST   /                              # Create decision
│   ├── GET    /:id                           # Get decision
│   ├── POST   /:id/impact-analysis           # AI impact analysis
│   └── POST   /extract                       # Extract from artifacts
│
├── /programs/:programId/dashboard
│   ├── GET    /health                        # Program health score
│   ├── GET    /kpis                          # KPI values
│   ├── GET    /insights                      # AI insights
│   └── GET    /alerts                        # Predictive alerts
│
├── /programs/:programId/milestones
│   ├── GET    /phases                        # List phases
│   ├── POST   /phases                        # Create phase
│   ├── GET    /milestones                    # List milestones
│   └── POST   /milestones                    # Create milestone
│
├── /programs/:programId/governance
│   ├── GET    /cadences                      # List cadences
│   ├── GET    /meetings                      # List meetings
│   ├── GET    /compliance                    # Compliance status
│   └── GET    /audit-trail                   # Audit trail
│
├── /programs/:programId/changes
│   ├── GET    /                              # List change requests
│   ├── POST   /                              # Create change
│   ├── GET    /:id                           # Get change details
│   ├── POST   /:id/impact-analysis           # AI impact analysis
│   └── POST   /:id/approve                   # Approve change
│
└── /ai
    ├── POST   /chat                          # AI chat interface
    ├── GET    /jobs/:id                      # AI job status
    └── GET    /usage                         # Usage & costs
```

### Response Format

```json
{
  "success": true,
  "data": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "Example Artifact",
    "metadata": {
      "topics": ["budget", "risk"],
      "persons": [...]
    }
  },
  "meta": {
    "timestamp": "2026-01-10T20:00:00Z",
    "request_id": "req_abc123"
  }
}
```

---

## Frontend Architecture: Cockpit UX

### Cockpit Layout

```
┌─────────────────────────────────────────────────────────┐
│ COMMAND BAR                                             │
│ [🔍 Search] [💬 AI Chat] [⬆️ Upload] [👤 User]      │
└─────────────────────────────────────────────────────────┘
┌──────┬────────────────────────────────────┬─────────────┐
│      │                                    │             │
│  S   │                                    │   CONTEXT   │
│  I   │          MAIN CONTENT              │   PANEL     │
│  D   │                                    │             │
│  E   │  (Module-specific views)           │ AI Insights │
│  B   │                                    │ Related     │
│  A   │                                    │ Artifacts   │
│  R   │                                    │ Alerts      │
│      │                                    │             │
│      │                                    │             │
└──────┴────────────────────────────────────┴─────────────┘
┌─────────────────────────────────────────────────────────┐
│ STATUS BAR: 🟢 Program Health: 85  📊 Budget: 65%      │
└─────────────────────────────────────────────────────────┘
```

### Component Hierarchy

```tsx
<CockpitLayout>
  <CommandBar>
    <GlobalSearch />
    <AIChat />
    <QuickUpload />
    <UserMenu />
  </CommandBar>

  <Sidebar>
    <ModuleNav modules={[
      {icon: "📁", name: "Artifacts", path: "/artifacts"},
      {icon: "💰", name: "Financial", path: "/financial", health: "red"},
      {icon: "⚠️", name: "Risks", path: "/risks", badge: 3},
      // ... other modules
    ]} />
  </Sidebar>

  <MainContent>
    <Outlet /> {/* React Router nested routes */}
  </MainContent>

  <ContextPanel>
    <AIInsights />
    <RelatedArtifacts />
    <ActiveAlerts />
  </ContextPanel>

  <StatusBar>
    <HealthIndicator score={85} />
    <BudgetStatus />
    <Notifications />
  </StatusBar>
</CockpitLayout>
```

### State Management Strategy

**React Context for Global State:**
```tsx
// src/contexts/ProgramContext.tsx

interface ProgramContextType {
  currentProgram: Program | null;
  setCurrentProgram: (program: Program) => void;
  healthScore: number;
  aiInsights: AIInsight[];
}

export const ProgramProvider: React.FC = ({ children }) => {
  const [currentProgram, setCurrentProgram] = useState<Program | null>(null);
  // ...

  return (
    <ProgramContext.Provider value={{ ... }}>
      {children}
    </ProgramContext.Provider>
  );
};
```

**TanStack Query for Server State:**
```tsx
// src/modules/artifacts/hooks/useArtifacts.ts

export function useArtifacts(programId: string) {
  return useQuery({
    queryKey: ['artifacts', programId],
    queryFn: () => fetchArtifacts(programId),
    staleTime: 30000, // 30 seconds
  });
}

export function useUploadArtifact() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: uploadArtifact,
    onSuccess: () => {
      // Invalidate and refetch
      queryClient.invalidateQueries({ queryKey: ['artifacts'] });
    },
  });
}
```

---

## Docker Compose Local Development

### docker-compose.yml

```yaml
version: '3.9'

services:
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_DB: cerberus
      POSTGRES_USER: cerberus
      POSTGRES_PASSWORD: cerberus_dev
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U cerberus"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: cerberus
      MINIO_ROOT_PASSWORD: cerberus_dev
    ports:
      - "9000:9000"   # API
      - "9001:9001"   # Console
    volumes:
      - minio_data:/data

  api:
    build:
      context: ./backend
      dockerfile: ../docker/Dockerfile.api
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_NAME: cerberus
      DB_USER: cerberus
      DB_PASSWORD: cerberus_dev
      REDIS_URL: redis:6379
      STORAGE_ENDPOINT: minio:9000
      STORAGE_ACCESS_KEY: cerberus
      STORAGE_SECRET_KEY: cerberus_dev
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started
      minio:
        condition: service_started
    volumes:
      - ./backend:/app

  worker:
    build:
      context: ./backend
      dockerfile: ../docker/Dockerfile.worker
    environment:
      # Same as API
      DB_HOST: postgres
      DB_PORT: 5432
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
      # ...
    depends_on:
      - postgres
      - redis

  web:
    build:
      context: ./frontend
      dockerfile: ../docker/Dockerfile.web
    environment:
      VITE_API_URL: http://localhost:8080/api/v1
    ports:
      - "3000:3000"
    volumes:
      - ./frontend:/app
      - /app/node_modules
    depends_on:
      - api

volumes:
  postgres_data:
  redis_data:
  minio_data:
```

### Usage

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f api

# Run migrations
docker-compose exec api /app/migrate up

# Stop all services
docker-compose down

# Clean everything (including volumes)
docker-compose down -v
```

---

## Security Architecture

### Authentication & Authorization

**JWT-Based Authentication:**
```go
type JWTClaims struct {
    UserID    string   `json:"user_id"`
    Email     string   `json:"email"`
    ProgramIDs []string `json:"program_ids"` // Programs user can access
    Role      string   `json:"role"`         // global role
    jwt.StandardClaims
}

// Middleware checks JWT validity
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract token from Authorization header
        // Validate and parse JWT
        // Inject claims into context
        next.ServeHTTP(w, r)
    })
}
```

**Program-Level Authorization:**
```go
func RequireProgram Access(programID string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := GetClaims(r.Context())
            if !contains(claims.ProgramIDs, programID) {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Data Encryption

- **In Transit:** TLS 1.3 for all API communication
- **At Rest:** PostgreSQL encryption, MinIO encryption
- **Sensitive Fields:** AES-256 encryption for stakeholder PII

### Audit Trail

Every modification logged in `audit_logs` table:
```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    program_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,  -- 'created', 'updated', 'deleted'
    changed_fields JSONB,
    changed_by UUID NOT NULL,
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);
```

---

## Observability

### Structured Logging

```go
logger.Info("artifact analyzed",
    "artifact_id", artifactID,
    "program_id", programID,
    "duration_ms", duration.Milliseconds(),
    "tokens_used", tokensUsed,
    "cost_usd", cost,
)
```

### Metrics (Future)

- Request latency (p50, p95, p99)
- AI processing time per artifact
- Cost per program per day
- Cache hit rates
- Error rates

### Health Checks

```
GET /health/live   # Is the service running?
GET /health/ready  # Is it ready to serve traffic?
```

---

**Document Version:** 1.0
**Last Updated:** 2026-01-10
**Document Owner:** Technical Lead
**Status:** Approved for Implementation

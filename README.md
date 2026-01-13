# Cerberus - Enterprise Program Governance System

An AI-powered strategic "cockpit" for program leaders to import, assimilate, analyze, and execute from. Cerberus transforms unstructured program artifacts into actionable intelligence through AI.

## Project Status

🎉 **Phase 3: Financial & Risk Modules** - Complete!

Current implementation status:
- ✅ Phase 1: Foundation infrastructure (8 services, auto-migrations)
- ✅ Phase 2: Artifacts Module with AI-powered analysis (semantic search, pgvector)
- ✅ Phase 3: Financial & Risk modules (invoice validation, risk identification)
- ✅ 3/10 core modules operational
- ✅ Complete backend with Claude API integration
- ✅ React frontend with comprehensive UI (15+ pages)
- ✅ NATS event bus connecting modules
- ✅ JWT authentication foundation
- ⏳ Remaining modules (communications, stakeholders, decisions, etc.) (planned for Phase 4-7)

## Architecture

### Technology Stack

**Backend:**
- Go 1.22+ with chi router
- PostgreSQL 16 with pgvector
- NATS for event streaming
- Redis for caching

**Frontend:**
- React 18+ with TypeScript
- Vite build tool
- Tailwind CSS + Headless UI
- TanStack Query (React Query)

**Storage:**
- RustFS (custom Rust-based file storage)

**AI:**
- Anthropic Claude API (Opus 4.5 & Sonnet 4.5)

### Services

1. **postgres** - PostgreSQL database with pgvector extension
2. **redis** - Cache and session storage
3. **nats** - Event streaming and message queue
4. **rustfs** - File storage service
5. **api** - Go REST API server
6. **worker** - Background AI processing worker
7. **web** - React frontend
8. **adminer** - Database administration UI

## Getting Started

### Prerequisites

- Docker and Docker Compose
- (Optional) Go 1.22+ for local development
- (Optional) Node.js 20+ for local frontend development

### Quick Start

1. **Clone the repository**
```bash
git clone <repository-url>
cd cerberus
```

2. **Set up environment variables**
```bash
cp .env.example .env
# Edit .env and add your ANTHROPIC_API_KEY
```

3. **Start all services**
```bash
docker-compose up -d
```

4. **Run database migrations**
```bash
docker-compose exec api ./migrate -cmd up
```

5. **Access the application**
- Frontend: http://localhost:3000
- API: http://localhost:8080
- Adminer (DB UI): http://localhost:8081
- NATS Monitoring: http://localhost:8222
- RustFS: http://localhost:9000

### Development Workflow

**Backend Development:**
```bash
cd backend

# Download dependencies
go mod download

# Run migrations
go run cmd/migrate/main.go -cmd up

# Run API server
go run cmd/api/main.go

# Run worker
go run cmd/worker/main.go
```

**Frontend Development:**
```bash
cd frontend

# Install dependencies
npm install

# Start dev server
npm run dev

# Build for production
npm run build
```

**RustFS Development:**
```bash
cd rustfs

# Build and run
cargo run

# Build release
cargo build --release
```

### Database Migrations

Run migrations:
```bash
docker-compose exec api ./migrate -cmd up
```

Check migration status:
```bash
docker-compose exec api ./migrate -cmd status
```

## Project Structure

```
cerberus/
├── backend/                 # Go backend
│   ├── cmd/                # Application entry points
│   │   ├── api/           # API server
│   │   ├── worker/        # Background worker
│   │   └── migrate/       # Migration CLI
│   ├── internal/
│   │   ├── modules/       # 10 feature modules
│   │   ├── platform/      # Shared infrastructure
│   │   ├── domain/        # Domain models
│   │   └── api/           # API layer
│   └── migrations/        # SQL migrations
│
├── frontend/              # React frontend
│   ├── src/
│   │   ├── modules/      # Feature modules
│   │   ├── layouts/      # Layout components
│   │   ├── components/   # Shared components
│   │   ├── contexts/     # React contexts
│   │   └── services/     # API clients
│   └── package.json
│
├── rustfs/               # Rust file storage service
│   ├── src/
│   │   └── main.rs
│   └── Cargo.toml
│
├── docker/               # Dockerfiles
│   ├── Dockerfile.api
│   ├── Dockerfile.worker
│   └── Dockerfile.web
│
├── docs/                 # Documentation
│   ├── 00-OVERVIEW.md
│   ├── 01-ARCHITECTURE.md
│   ├── 02-MODULES.md
│   ├── 03-ROADMAP.md
│   ├── 04-DATA-MODEL.md
│   └── 05-AI-STRATEGY.md
│
└── docker-compose.yml    # Service orchestration
```

## 10 Core Modules

**Operational (3/10):**
1. ✅ **Program Artifacts** ⭐ - AI-powered document analysis and metadata extraction (Phase 2)
2. ✅ **Financial Reporting** - Invoice validation, variance analysis, budget tracking (Phase 3)
3. ✅ **Risk & Issue Management** - Proactive risk identification with conversation threads (Phase 3)

**Coming Soon (7/10):**
4. ⏳ **Communications** - AI-assisted communication drafting (Phase 4)
5. ⏳ **Stakeholder Management** - Stakeholder tracking and engagement (Phase 4)
6. ⏳ **Decision Log** - Decision capture and impact analysis (Phase 5)
7. ⏳ **Executive Dashboard** - Health scoring and predictive alerts (Phase 5)
8. ⏳ **Milestone & Phase Management** - Timeline and gate management (Phase 6)
9. ⏳ **Governance Framework** - Compliance and audit tracking (Phase 6)
10. ⏳ **Change Control Board** - Change impact analysis (Phase 7)

## API Documentation

API endpoints will be documented as they are implemented. Base URL: `http://localhost:8080/api/v1`

**Health Check:**
```bash
curl http://localhost:8080/health
```

## Contributing

This is a strategic enterprise project. Contribution guidelines will be established as the project progresses.

## License

[To be determined]

## Roadmap

See [docs/03-ROADMAP.md](docs/03-ROADMAP.md) for the complete 32-week implementation plan.

**Phase 1 (Weeks 1-4):** Foundation - Current phase
**Phase 2 (Weeks 5-8):** Artifacts Module (CORE) - Next phase
**Phase 3-8:** Additional modules and optimization

## Support

For questions or issues, please refer to the project documentation in the `docs/` directory.

---

**Last Updated:** 2026-01-11
**Project Status:** Phase 1 - In Development
**Version:** 0.1.0 (Foundation)

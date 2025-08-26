# 🏗️ ExpenseTracker Architecture Documentation

## Table of Contents

1. [System Overview](#system-overview)
2. [Architecture Layers](#architecture-layers)
3. [Data Model](#data-model)
4. [Application Flow](#application-flow)
5. [Deployment Architecture](#deployment-architecture)
6. [Technology Stack](#technology-stack)
7. [Security Architecture](#security-architecture)
8. [Performance Considerations](#performance-considerations)
9. [Scalability Patterns](#scalability-patterns)
10. [Development Guidelines](#development-guidelines)

---

## System Overview

ExpenseTracker is a modern, collaborative expense management application built with a clean layered architecture. The system follows Domain-Driven Design (DDD) principles and implements a RESTful API with real-time capabilities.

### Core Principles

- **Separation of Concerns**: Clear layer boundaries with distinct responsibilities
- **Domain-Driven Design**: Business logic encapsulated in domain services
- **Progressive Web App**: Modern web technologies for native-like experience
- **Real-time Collaboration**: WebSocket-powered live updates
- **Security First**: JWT authentication, input validation, audit trails
- **Cloud-Ready**: Containerized with multiple deployment options

---

## Architecture Layers

The application follows a traditional N-tier architecture with clear separation between presentation, business logic, and data layers:

```mermaid
graph TB
    subgraph "Client Layer"
        PWA["Progressive Web App<br/>HTML5 + CSS3 + JavaScript ES6+"]
        SW["Service Worker<br/>Offline Caching"]
        Manifest["Web Manifest<br/>App Installation"]
    end

    subgraph "Presentation Layer"
        Login["Login & Registration<br/>JWT Authentication"]
        Dashboard["Dashboard<br/>Events Overview"]
        EventDetails["Event Management<br/>Expenses & Settlements"]
        RealTime["Real-time Updates<br/>WebSocket Connection"]
    end

    subgraph "API Gateway Layer"
        Router["Gin Router<br/>HTTP/WebSocket Handler"]
        Middleware["Middleware Stack"]
        Auth["JWT Auth Middleware"]
        CORS["CORS Middleware"]
        RateLimit["Rate Limiting"]
        Logger["Request Logger"]
    end

    subgraph "Controller Layer"
        AuthController["Authentication<br/>Controller"]
        EventController["Event Management<br/>Controller"]
        ExpenseController["Expense Management<br/>Controller"]
        SettlementController["Settlement<br/>Controller"]
        WebSocketController["WebSocket<br/>Controller"]
    end

    subgraph "Service Layer"
        SettlementService["Settlement Service<br/>Balance Calculations<br/>Optimization Algorithms"]
        ValidationService["Business Logic<br/>Validation"]
        NotificationService["Real-time<br/>Notifications"]
    end

    subgraph "Data Access Layer"
        GORM["GORM ORM<br/>Database Abstraction"]
        Models["Data Models<br/>User, Event, Expense<br/>Settlement, Audit"]
    end

    subgraph "Storage Layer"
        SQLite["SQLite Database<br/>Local File Storage"]
        Static["Static Files<br/>CSS, JS, Assets"]
    end

    subgraph "Infrastructure"
        Docker["Docker Container"]
        Traefik["Reverse Proxy<br/>SSL Termination"]
        Backup["Automated Backups<br/>Data Protection"]
    end

    %% Client connections
    PWA --> Router
    SW --> PWA
    Manifest --> PWA

    %% Presentation connections
    Login --> AuthController
    Dashboard --> EventController
    EventDetails --> ExpenseController
    EventDetails --> SettlementController
    RealTime --> WebSocketController

    %% Middleware flow
    Router --> Middleware
    Middleware --> Auth
    Middleware --> CORS
    Middleware --> RateLimit
    Middleware --> Logger

    %% Controller connections
    Middleware --> AuthController
    Middleware --> EventController
    Middleware --> ExpenseController
    Middleware --> SettlementController
    Middleware --> WebSocketController

    %% Service connections
    SettlementController --> SettlementService
    ExpenseController --> ValidationService
    WebSocketController --> NotificationService

    %% Data access
    AuthController --> GORM
    EventController --> GORM
    ExpenseController --> GORM
    SettlementController --> GORM
    SettlementService --> GORM

    %% Storage connections
    GORM --> Models
    Models --> SQLite
    Router --> Static

    %% Infrastructure
    Docker --> Router
    Traefik --> Docker
    Backup --> SQLite
```

### Layer Responsibilities

#### 1. Client Layer (Progressive Web App)
- **Progressive Web App**: Modern web technologies providing native-like experience
- **Service Worker**: Offline functionality and intelligent caching
- **Web Manifest**: App installation and home screen integration

#### 2. Presentation Layer (Frontend)
- **Authentication UI**: Login, registration, and profile management
- **Dashboard**: Events overview and quick actions
- **Event Management**: Detailed expense tracking and settlement views
- **Real-time Updates**: WebSocket-powered live collaboration

#### 3. API Gateway Layer (Middleware)
- **Gin Router**: HTTP request routing and WebSocket upgrade handling
- **Authentication**: JWT token validation and user context
- **CORS**: Cross-origin resource sharing configuration
- **Rate Limiting**: Request throttling and abuse prevention
- **Logging**: Structured request/response logging

#### 4. Controller Layer (Request Handlers)
- **AuthController**: User authentication and profile management
- **EventController**: Event creation, management, and participation
- **ExpenseController**: Expense submission, approval, and categorization
- **SettlementController**: Settlement generation and management
- **WebSocketController**: Real-time communication handling

#### 5. Service Layer (Business Logic)
- **SettlementService**: Complex balance calculations and optimization algorithms
- **ValidationService**: Business rule validation and data integrity
- **NotificationService**: Real-time event broadcasting

#### 6. Data Access Layer (ORM)
- **GORM**: Object-relational mapping with PostgreSQL/SQLite support
- **Models**: Strongly-typed data models with relationships

#### 7. Storage Layer
- **SQLite Database**: Lightweight, file-based database for development/small deployments
- **Static Files**: CSS, JavaScript, images, and other assets

#### 8. Infrastructure Layer
- **Docker**: Containerization for consistent deployments
- **Reverse Proxy**: SSL termination and load balancing
- **Backup Services**: Automated database backups and retention

---

## Data Model

The application uses a well-normalized relational data model with proper foreign key relationships and constraints:

```mermaid
erDiagram
    User ||--o{ Event : creates
    User ||--o{ Participation : "participates in"
    User ||--o{ Expense : submits
    User ||--o{ Contribution : contributes
    User ||--o{ Settlement : "owes/owed"
    User ||--o{ AuditLog : "activity tracked"

    Event ||--o{ Participation : "has participants"
    Event ||--o{ Expense : "contains expenses"
    Event ||--o{ Contribution : "receives contributions"
    Event ||--o{ Settlement : "generates settlements"

    Expense ||--o{ ExpenseShare : "split among users"
    Expense }|--|| ExpenseCategory : "categorized as"
    Expense }|--|| User : "reviewed by"

    ExpenseShare }|--|| User : "assigned to"

    Settlement }|--|| User : "from user"
    Settlement }|--|| User : "to user"

    User {
        uint id PK
        string username UK
        string email UK
        string password
        string currency
        string first_name
        string last_name
        string display_name
        string avatar
        boolean notifications
        string theme
        string language
        string timezone
        timestamp created_at
        timestamp updated_at
    }

    Event {
        uint id PK
        string name
        string description
        string code UK
        uint created_by FK
        string currency
        string status
        decimal total_contributions
        decimal total_expenses
        timestamp end_date
        boolean require_approval
        decimal auto_approval_limit
        timestamp created_at
        timestamp updated_at
    }

    Participation {
        uint id PK
        uint user_id FK
        uint event_id FK
        string role
        timestamp joined_at
    }

    Expense {
        uint id PK
        uint event_id FK
        uint submitted_by FK
        uint category_id FK
        decimal amount
        string currency
        string description
        date date
        string status
        uint reviewed_by FK
        timestamp reviewed_at
        string rejection_reason
        string receipt_url
        string location
        string vendor
        string notes
        string split_type
        timestamp submitted_at
    }

    ExpenseShare {
        uint id PK
        uint expense_id FK
        uint user_id FK
        decimal amount
        decimal percentage
    }

    ExpenseCategory {
        uint id PK
        string name UK
        string icon
    }

    Contribution {
        uint id PK
        uint event_id FK
        uint user_id FK
        decimal amount
        string currency
        string notes
        timestamp timestamp
    }

    Settlement {
        uint id PK
        uint event_id FK
        uint from_user_id FK
        uint to_user_id FK
        decimal amount
        string currency
        string status
        string method
        string payment_reference
        timestamp settled_at
        timestamp created_at
    }

    AuditLog {
        uint id PK
        string table_name
        uint record_id
        string action
        text old_values
        text new_values
        uint changed_by FK
        timestamp changed_at
        string ip_address
        string user_agent
        string session_id
    }
```

### Key Entities

#### Core Entities
- **User**: System users with profiles and preferences
- **Event**: Group expense events (trips, parties, etc.)
- **Participation**: User roles within events
- **Expense**: Individual expense entries
- **Settlement**: Optimized payment transactions

#### Supporting Entities
- **ExpenseShare**: Expense splitting configuration
- **ExpenseCategory**: Predefined expense categories
- **Contribution**: Money contributed to events
- **AuditLog**: Complete activity audit trail

### Data Relationships

1. **User-Event Relationship**: Many-to-many through Participation
2. **Event-Expense Relationship**: One-to-many with cascading operations
3. **Expense-Share Relationship**: One-to-many for flexible splitting
4. **Settlement Optimization**: Calculated from expense shares and contributions

---

## Application Flow

The following sequence diagram illustrates the typical user interaction flow:

```mermaid
sequenceDiagram
    participant U as User Browser
    participant PWA as PWA Client
    participant API as Gin Router
    participant Auth as Auth Middleware
    participant EC as Event Controller
    participant SC as Settlement Controller
    participant SS as Settlement Service
    participant DB as SQLite Database
    participant WS as WebSocket

    Note over U,WS: User Authentication Flow
    U->>PWA: Login Request
    PWA->>API: POST /api/v1/auth/login
    API->>DB: Validate Credentials
    DB-->>API: User Data
    API-->>PWA: JWT Token
    PWA->>PWA: Store Token Locally
    PWA-->>U: Login Success

    Note over U,WS: Event Creation & Management
    U->>PWA: Create Event
    PWA->>API: POST /api/v1/events/
    API->>Auth: Validate JWT
    Auth-->>API: User Authenticated
    API->>EC: Process Event Creation
    EC->>DB: Create Event & Participation
    DB-->>EC: Event Created
    EC-->>API: Event Response
    API-->>PWA: Event Data
    PWA-->>U: Event Created

    Note over U,WS: Expense Submission & Approval
    U->>PWA: Submit Expense
    PWA->>API: POST /api/v1/expenses/event/:id/
    API->>Auth: Validate JWT
    Auth-->>API: User Authenticated
    API->>DB: Create Expense & Shares
    DB-->>API: Expense Saved
    API->>WS: Broadcast Expense Created
    WS-->>PWA: Real-time Update
    PWA-->>U: Expense Submitted

    Note over U,WS: Settlement Generation & Optimization
    U->>PWA: Generate Settlements
    PWA->>API: POST /api/v1/settlements/event/:id/generate
    API->>Auth: Validate JWT
    Auth-->>API: User Authenticated
    API->>SC: Generate Settlements
    SC->>SS: Calculate Balances
    SS->>DB: Query Expenses & Contributions
    DB-->>SS: Financial Data
    SS->>SS: Optimize Settlement Transactions
    SS-->>SC: Optimized Settlements
    SC->>DB: Save Settlements
    DB-->>SC: Settlements Saved
    SC-->>API: Settlement Response
    API-->>PWA: Settlement Data
    PWA-->>U: Settlements Generated

    Note over U,WS: Real-time Updates
    API->>WS: Event Update
    WS-->>PWA: WebSocket Message
    PWA->>PWA: Update UI
    PWA-->>U: Live Update Displayed
```

### Key Workflows

#### 1. Authentication Flow
- JWT-based stateless authentication
- Secure password hashing with bcrypt
- Token refresh mechanism
- Session management in localStorage

#### 2. Event Management
- Unique event codes for easy joining
- Role-based permissions (owner, admin, participant)
- Real-time participant updates

#### 3. Expense Processing
- Multi-step approval workflow
- Flexible splitting algorithms (equal, percentage, custom)
- Receipt upload and categorization

#### 4. Settlement Optimization
- Complex balance calculation algorithms
- Minimal transaction optimization
- Real-time settlement updates

---

## Deployment Architecture

The application supports multiple deployment strategies from development to enterprise scale:

```mermaid
graph LR
    subgraph "Development Environment"
        Dev["Local Development<br/>go run main.go<br/>Port 8080"]
        HotReload["Hot Reload<br/>Air Tool<br/>Auto-restart"]
    end

    subgraph "Docker Containerization"
        DockerFile["Dockerfile<br/>Multi-stage Build<br/>Alpine Linux"]
        DockerCompose["Docker Compose<br/>App + Proxy + Backup"]
        Volume["Persistent Volumes<br/>Database & Backups"]
    end

    subgraph "Cloud Deployment Options"
        AWS["AWS ECS<br/>Fargate Tasks<br/>Load Balancer"]
        DigitalOcean["DigitalOcean<br/>App Platform<br/>Auto-scaling"]
        Railway["Railway<br/>Git-based Deploy<br/>Auto HTTPS"]
        Fly["Fly.io<br/>Global Edge<br/>SQLite Replicas"]
    end

    subgraph "Reverse Proxy & SSL"
        Traefik["Traefik v2.10<br/>Auto HTTPS<br/>Let's Encrypt"]
        Nginx["Nginx Alternative<br/>Manual Config<br/>Custom Certs"]
    end

    subgraph "Database & Backups"
        SQLiteFile["SQLite File<br/>expense_tracker.db<br/>Local Storage"]
        AutoBackup["Automated Backups<br/>Daily Cron Job<br/>Compressed Storage"]
        BackupRetention["Backup Retention<br/>30 Days Default<br/>Configurable"]
    end

    subgraph "Monitoring & Health"
        HealthCheck["Health Endpoint<br/>/health<br/>Database Status"]
        Logging["Request Logging<br/>Structured Logs<br/>Audit Trail"]
        Metrics["Application Metrics<br/>Response Times<br/>Error Rates"]
    end

    %% Development flow
    Dev --> DockerFile
    HotReload --> Dev

    %% Docker flow
    DockerFile --> DockerCompose
    DockerCompose --> Volume

    %% Deployment options
    DockerCompose --> AWS
    DockerCompose --> DigitalOcean
    DockerCompose --> Railway
    DockerCompose --> Fly

    %% Proxy connections
    DockerCompose --> Traefik
    Traefik --> Nginx

    %% Database connections
    Volume --> SQLiteFile
    Volume --> AutoBackup
    AutoBackup --> BackupRetention

    %% Monitoring connections
    DockerCompose --> HealthCheck
    DockerCompose --> Logging
    DockerCompose --> Metrics
```

### Deployment Options

#### Local Development
```bash
# Quick start
go run main.go

# With hot reload
air

# Custom configuration
PORT=3000 JWT_SECRET=dev-secret go run main.go
```

#### Docker Deployment
```bash
# Build and run
docker-compose -f deployment/docker/docker-compose.yml up -d

# With custom environment
JWT_SECRET=production-secret docker-compose up -d
```

#### Cloud Platforms
- **AWS ECS**: Fargate containers with Application Load Balancer
- **DigitalOcean**: App Platform with automatic scaling
- **Railway**: Git-based deployment with automatic HTTPS
- **Fly.io**: Global edge deployment with SQLite replication

---

## Technology Stack

### Backend Technologies
- **Language**: Go 1.21+ with strong typing and concurrency
- **Web Framework**: Gin for high-performance HTTP routing
- **ORM**: GORM for database abstraction and migrations
- **Database**: SQLite for development, PostgreSQL for production
- **Authentication**: JWT tokens with bcrypt password hashing
- **Real-time**: WebSocket for live collaboration

### Frontend Technologies
- **Progressive Web App**: Modern web standards
- **Styling**: Swiss Design system with CSS custom properties
- **JavaScript**: ES6+ with async/await patterns
- **Icons**: Lucide icon library
- **Caching**: Service Worker with Cache API

### Infrastructure Technologies
- **Containerization**: Docker with multi-stage builds
- **Reverse Proxy**: Traefik v2.10 with automatic HTTPS
- **Backups**: Automated SQLite backups with compression
- **Monitoring**: Health checks and structured logging

---

## Security Architecture

### Authentication & Authorization
- **JWT Tokens**: Stateless authentication with 24-hour expiration
- **Password Security**: bcrypt hashing with salt rounds
- **Role-Based Access**: Granular permissions within events
- **Session Management**: Secure token storage and refresh

### Data Protection
- **Input Validation**: Server-side validation for all inputs
- **SQL Injection Prevention**: GORM parameterized queries
- **XSS Protection**: Content Security Policy headers
- **Rate Limiting**: Request throttling per IP address

### Audit & Compliance
- **Audit Trail**: Complete activity logging with user context
- **Data Retention**: Configurable backup retention policies
- **GDPR Compliance**: User data export and deletion capabilities

---

## Performance Considerations

### Database Optimization
- **Indexes**: Strategic indexing on frequently queried columns
- **Query Optimization**: GORM query optimization and eager loading
- **Connection Pooling**: Efficient database connection management

### Caching Strategy
- **Service Worker**: Intelligent client-side caching
- **Static Assets**: Browser caching with cache busting
- **API Responses**: Conditional requests with ETags

### Real-time Performance
- **WebSocket Connections**: Efficient message broadcasting
- **Connection Management**: Automatic reconnection handling
- **Message Queuing**: Buffered updates for offline users

---

## Scalability Patterns

### Horizontal Scaling
- **Stateless Design**: No server-side session storage
- **Load Balancing**: Multiple instance deployment
- **Database Scaling**: Read replicas and connection pooling

### Vertical Scaling
- **Resource Optimization**: Efficient memory and CPU usage
- **Goroutine Management**: Concurrent request handling
- **Database Tuning**: Query optimization and indexing

### Microservices Migration Path
- **Service Extraction**: Settlement service as independent service
- **API Gateway**: Centralized routing and authentication
- **Event-Driven Architecture**: Asynchronous communication patterns

---

## Development Guidelines

### Code Organization
```
├── controllers/          # Request handlers and routing logic
├── middleware/           # Cross-cutting concerns (auth, CORS, logging)
├── models/              # Data models and database schema
├── services/            # Business logic and domain services
├── database/            # Database configuration and migrations
├── static/              # Frontend assets (CSS, JS, images)
├── templates/           # HTML templates
├── deployment/          # Docker and deployment configurations
└── main.go             # Application entry point
```

### Best Practices
- **Error Handling**: Consistent error responses with proper HTTP status codes
- **Testing**: Unit tests for business logic, integration tests for APIs
- **Documentation**: Clear API documentation with examples
- **Logging**: Structured logging with appropriate log levels
- **Configuration**: Environment-based configuration management

### Development Workflow
1. **Local Development**: Use `air` for hot reloading
2. **Testing**: Run tests before committing changes
3. **Code Review**: Peer review for all changes
4. **Deployment**: Automated deployment through CI/CD pipelines

---

## Future Enhancements

### Planned Features
- **Mobile Apps**: React Native or Flutter applications
- **Advanced Analytics**: Expense trends and insights
- **Integration APIs**: Third-party service integrations
- **Multi-tenancy**: Organization-level isolation

### Technical Improvements
- **Microservices**: Service decomposition for better scalability
- **Event Sourcing**: Complete audit trail with event replay
- **GraphQL API**: Flexible data fetching for mobile clients
- **Kubernetes**: Container orchestration for enterprise deployments

---

*This architecture documentation is maintained alongside the codebase and should be updated with any significant architectural changes.*

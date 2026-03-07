# gofer

<div align="center">
  <img src="./.github/assets/gofer.png" alt="Tally Mascot" width="500" />
</div>

## Problem

When integrating with third-party services that send HTTP requests to your endpoints (Stripe payment events, GitHub pushes, etc), it's not always clear what the shape of the request will be. OTel solves this for systems you control---instrument your HTTP client and you see outbound request details in traces. But you cannot add OTel to Stripe's HTTP client.

## Solution

With **gofer**, you deploy it to a public endpoint, hand the URL to the third-party service, and inspect every request that arrives in real time in your browser. 

Gofer is a single Go binary with an embedded web UI. Create a bin, get a public capture URL, point the third-party sender at it, and watch requests arrive via SSE. 

- **SQLite** with WAL mode for storage
- **SSE** for real-time push to the browser 
- **OTel traces** exported to any OTLP destination

## Architecture

### Flowchart

```mermaid
graph TB
    subgraph "The Internet"
        STRIPE["Third-party sender<br/>(Stripe, GitHub, etc.)"]
        BROWSER["Developer's Browser"]
    end

    subgraph "gofer"

        subgraph "Inbound Adapters"
            WEB_UI["Web UI Handlers<br/>GET / — home<br/>GET /bins/:slug — inspection"]
            API["API Handlers<br/>POST /api/bins<br/>GET /api/bins/:slug"]
            CAPTURE["Capture Handler<br/>ANY /gofer/:slug/*"]
            SSE_EP["SSE Endpoint<br/>GET /sse/:slug"]
            HEALTH["GET /healthz"]
        end

        subgraph "Service Layer"
            UC["Use Cases<br/>CreateBin / CaptureRequest<br/>ViewBin / ViewRequest<br/>CleanupExpiredBins"]
        end

        subgraph "Domain Layer"
            ENT["Entities: Bin, CapturedRequest<br/>Value Objects: Slug, RawPayload<br/>Logic: IsExpired, NewSlug, Validate"]
        end

        subgraph "Outbound Adapters"
            PORTS["Ports (interfaces)<br/>BinRepository<br/>RequestRepository<br/>EventPublisher"]
            SQLITE["SQLite WAL<br/>implements BinRepository<br/>implements RequestRepository"]
            HUB["SSE Hub<br/>implements EventPublisher"]
            OTEL["OTel Tracer<br/>OTLP exporter"]
        end

        CLEANUP["Cleanup Goroutine<br/>time.Ticker — deletes expired bins"]
    end

    STRIPE -->|"POST /gofer/a3xB7k/webhook"| CAPTURE
    BROWSER -->|"Create/inspect bins"| WEB_UI
    BROWSER -->|"AJAX"| API
    BROWSER <-->|"EventSource"| SSE_EP

    WEB_UI --> UC
    API --> UC
    CAPTURE --> UC
    SSE_EP --> HUB

    UC --> ENT
    UC --> PORTS

    SQLITE -.->|implements| PORTS
    HUB -.->|implements| PORTS
    CLEANUP --> UC
```

### ER Diagram

```mermaid
erDiagram
    Bin {
        uuid id PK "Internal — never exposed"
        Slug slug UK "External nanoid — URL-safe"
        timestamp created_at
        timestamp expires_at
    }

    CapturedRequest {
        uuid id PK "Internal — never exposed"
        uuid bin_id FK "References Bin"
        int sequence_num "Monotonic per bin — total order"
        string method
        string path
        jsonb headers
        jsonb query_params
        int body_size
        string content_type
        string remote_addr
        timestamp captured_at
        RawPayload raw_payload "Exact bytes received"
    }

    Bin ||--o{ CapturedRequest : "has many"
```

## Usage

Coming soon!
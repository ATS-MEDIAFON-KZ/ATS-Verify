---
name: ats-verify-findings
type: knowledge-base
description: Persistent technical decisions, database schemas, external API limits, and parsing strategies.
---

# findings.md

## 1. Tracking Data Logic (Marketplace)
* **Primary Key:** `track_number`.
* **State Machine:** Track numbers have a boolean state: `used`. Overwrites allowed only if `used=false`.

## 2. IMEI Extraction & Matching Logic
* **PDF Target:** Customs Declaration "Graph 31".
* **Match Condition:** Search for 14-digit IMEI (CSV) within 15-digit sequences (PDF text). Regex: `\b\d{14}\d?\b`.

## 3. Support Ticket Data Structure (Kanban)
Replaces the legacy Google Sheet for rejected applications.
* **Database Table:** `support_tickets`
* **Fields Required:**
  * `iin` (ИИН - String)
  * `full_name` (ФИО - String)
  * `support_ticket_id` (ID заявки в СП - String)
  * `application_number` (номер заявки - String)
  * `document_number` (номер документа - String)
  * `rejection_reason` (причина отклонения заявки - Text)
  * `attachments` (подтверждающие документы - Array of URLs/Paths)
  * `support_comment` (комментарий СП - Text)
  * `customs_comment` (комментарий ГО - Text, updated by Customs upon resolution)
  * `status` (Enum: `To Do`, `In Progress`, `Completed/Resolved`)

## 4. External APIs
* **Kazpost:** `https://open.post.kz/services/details/15`
* **CDEK:** `https://www.cdek.ru/ru/tracking/`

## 5. Phase 2: Dependency Research (Context7 Verified)

### CDEK API v2 (✅ Verified via Context7)
* **Base:** `https://api.cdek.ru/v2/`
* **Auth:** OAuth2 Bearer Token (`POST /v2/oauth/token`)
* **Tracking Endpoint:** `GET /v2/orders?cdek_number={number}` or `?im_number={number}`
* **Response:** JSON with `entity.uuid`, `entity.cdek_number`, `entity.number`, location data, package details, services.
* **Rate Limits:** Not explicitly documented. Recommend per-request caching in Redis.

### Kazpost API (⚠️ Not in Context7 — Niche KZ API)
* **Docs:** `https://open.post.kz/services/details/15`
* **Strategy:** Manual REST integration. Will need to reverse-engineer request/response format from docs or test calls.
* **Fallback:** Scrape `https://post.kz/tracking` if API is unstable.

### Go PDF Library: `ledongthuc/pdf` (✅ Verified)
* **Import:** `github.com/ledongthuc/pdf`
* **Usage:** `pdf.Open(filePath)` → `reader.GetPlainText()` → `io.Reader` → full text.
* **Limitation:** Simple text extraction. May struggle with complex PDF layouts (Graph 31). Fallback: Python `PyMuPDF` sidecar.
* **Decision:** Use `ledongthuc/pdf` as primary. Add Python sidecar only if extraction quality is insufficient.

### WebSocket (Real-time Kanban)
* **Library:** `github.com/gorilla/websocket` (de facto Go standard)
* **Strategy:** Server-side hub pattern — broadcast ticket status changes to connected Customs clients via WebSocket.

### React DnD (Frontend Kanban)
* **Library:** `@dnd-kit/core` + `@dnd-kit/sortable`
* **Rationale:** Active maintenance, accessible, lightweight, hooks-based. Better DX vs `react-beautiful-dnd` (deprecated) and `@hello-pangea/dnd`.
---
name: ats-verify-task-plan
type: project-state
description: Active Kanban board and phase checklist based on the BLAST framework.
---

# task_plan.md

## Protocol 0: Sync

- [x] Ingest GOALS.md and functional requirements.
- [x] Initialize project memory files with YAML frontmatter.

## Phase 1: Blueprint (Logic & Schema)

- [x] Define PostgreSQL schema for Users, Parcels, Risk_Flags, Support_Tickets, and Tracking_Events.
- [x] Draft data structs in Go (`internal/models`).
- [x] Define the exact output format struct for the IMEI verification report.

## Phase 2: Link (Dependencies)

- [x] Verify Kazpost tracking API specs via Context7. *(Not in Context7 — manual integration planned)*
- [x] Verify CDEK tracking API specs via Context7. *(`GET /v2/orders`, OAuth2 Bearer)*
- [x] Select and test Go PDF parsing library → `ledongthuc/pdf` (`GetPlainText()`).
- [x] **Kanban Integration:** WebSocket → `gorilla/websocket` (hub pattern).
- [x] Select Frontend Drag-and-Drop library → `@dnd-kit/core` + `@dnd-kit/sortable`.

## Phase 3: Architect (Implementation)

- [x] **Auth:** JWT role-based middleware. *(Already existed and is solid)*
- [x] **Auth (Registration):** Add `is_approved` field, endpoints for Registration and Admin user approval. Include auto-approval logic for `@ats-mediafon.kz`.
- [x] **Ingestion:** Marketplace CSV parser with missing data skipping and `used` boolean upsert logic. *(Already existed)*
- [x] **Smart Ingestion:** Overhaul CSV parser with strict field validation (null-checks) and dynamically detect marketplace column handling based on `user.Role`.
- [x] **Tracking:** Implement unified tracking interface for Kazpost/CDEK. *(Tracker interface + stubs)*
- [x] **Risk Engine:** Implement Admin logic to analyze document reuse and assign Red/Yellow/Green flags to IIN/BINs. *(3 algorithms + auto-flagging)*
- [x] **Verification:** Implement PDF "Graph 31" extraction and 14-digit IMEI substring matching logic. *(ledongthuc/pdf integrated)*
- [x] **Kanban Board:** Implement drag-and-drop UI and WebSocket event handling for status updates. *(TicketsPage with @dnd-kit)*
- [x] **Ticket Service:** Create CRUD API for Support Tickets. Implement status update endpoint for Kanban drag-and-drop.

## Phase 4: Style & Output

- [ ] Implement the specific IMEI text report generator (Total stats, examples, line-by-line mapping).
- [x] Apply premium design tokens (Navy, Electric Blue) to React/Vite frontend. *(Inter font, design system in index.css)*
- [x] Implement Dashboard UI (Statistics, Task Calendar) based on reference designs. *(Stat cards, ticket breakdown, activity feed)*
- [x] Implement Kanban Board UI with draggable cards (ATS creates, Customs moves/edits). *(TicketsPage + @dnd-kit)*

## Phase 5: Trigger

- [x] Write Dockerfile and docker-compose.yml. *(Multi-stage: Node→Go→Alpine, SPA serving, non-root)*
- [ ] Final end-to-end testing.

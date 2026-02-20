---
name: ats-verify-goals
type: project-memory
description: Executive vision, core business rules, role definitions, and the primary bottleneck for the ATS-Verify platform, including the ATS-to-Customs Kanban workflow.
---

# GOALS.md

## Project: ATS-Verify

**Last Updated:** 2026-02-19
**Status:** Active Development

## 1. Executive Vision

**Objective:** Build a robust, role-based SaaS microservice platform for logistics and customs verification. The system validates imported devices, tracks parcels, performs risk analysis on IIN/BINs, automatically verifies IMEI codes against Customs Declaration PDFs, and facilitates seamless dispute resolution between ATS Support and Customs via an integrated Kanban board.

## 2. Current Bottleneck (The Riskiest Part)

- **Status:** 🔴 UNSOLVED
- **Description:** Accurately extracting text from "Graph 31" of varied PDF declarations and executing a high-performance 14-digit substring match against 15-digit sequences (ignoring the Luhn check digit) from a multi-column CSV.
- **Proposed Solution:** Implement a dedicated parsing service in Go. If Go PDF libraries (`ledongthuc/pdf`) fail on complex layouts, fallback to a Python microservice/sidecar (`PyMuPDF`) specifically for text extraction.
- **Secondary Bottleneck:** Ensuring real-time or optimistic UI updates on the Kanban board without overloading the PostgreSQL database during concurrent drag-and-drop operations by multiple Customs officers.


## 3. Core Roles & Permissions

1.  **Admin:** Full access. Downloads CSV reports by date. Uploads Risk CSVs. Assigns Risk levels (Red/Yellow/Green) to IIN/BINs. Approves user registrations.
2.  **Customs:** Views parcel tables. Sets `used = true` flag. Runs IMEI PDF verification. **Manages and resolves support tickets on the Kanban Board.**
3.  **Marketplace:** Uploads daily parcel CSVs.
4.  **ATS Staff:** Checks track number existence and `used` status. **Creates review tickets for Customs based on user appeals (rejected applications).**
5.  **Paid Users:** Tracks parcels and runs IMEI PDF verification.

*Registration & Approval Workflow:*
- Automatic: Users registering with an `@ats-mediafon.kz` email are automatically assigned the `ats_staff` role and are approved (`is_approved = true`).
- Manual: Other users default to `is_approved = false` and must be explicitly approved by an Admin via the approve endpoint.

## 4. Business Logic Rules

- **Marketplace Ingestion:** CSV format: `marketplace | country | brand | name | track number | SNT | date`. Skip rows with missing data.
  - _Upsert Logic:_ If track exists and `used=false`, warn user but allow overwrite. If `used=true`, ERROR.
- **Risk Analysis (Admin):** Analyzes `Date | AppId | IIN/BIN | doc | User | Org | Status | Reject | Reason` to flag doc reuse, flip-flop statuses, and high-frequency IIN/BINs.
- **IMEI Verification (Customs/Paid):** Validates 14-digit IMEIs from CSV (`Imei1`..`Imei4`) against 15-digit values in PDF. Must output a specific statistical report and line-by-line result mapping.
- **Support Ticketing (Kanban):** ATS Staff creates a ticket replacing the legacy Google Sheet. Customs officers move tickets across statuses (e.g., `To Do` -> `In Progress` -> `Resolved`) and leave final comments ("комментарий ГО").

## 5. Architecture & Constraints

- **Stack:** Go (Backend), PostgreSQL (DB), React/Vite (Frontend).
- **Tracking Integrations:** Kazpost & CDEK APIs.
- **Architecture:** Strict Clean Architecture (`internal/core`, `internal/repository`, `internal/transport`).

## 6. Definition of Done (DoD)

1. Code compiles without errors.
2. Business logic is isolated in `internal/core/` and covered by Unit Tests.
3. No hardcoded secrets (use `.env`).
4. Task plan and findings are updated.

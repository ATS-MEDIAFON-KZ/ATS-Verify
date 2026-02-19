---
name: ats-verify-design
type: ui-tokens
description: UI/UX guidelines, semantic color mapping, and layout structures for the frontend, including the Kanban Board.
---

# design.md

## 1. Design Tokens
* **Theme:** Premium, Professional Corporate Dashboard.
* **Background:** Light/Clean aesthetic for dashboards (e.g., `#F8FAFC` background with white `#FFFFFF` cards) or Deep Navy based on user preference toggle.
* **Accents:** Electric Blue (`#1D4ED8`) or Teal/Mint (like in provided references) for primary actions.
* **Typography:** `Inter` or `Outfit` (sans-serif, clean readability for numbers).

## 2. Semantic Colors (Risk & Status)
* 🔴 **High Risk / High Priority (Red):** IIN/BIN risk, or Urgent Kanban Tasks.
* 🟡 **Medium Risk / In Progress (Yellow):** Verification attempts, or Tasks under review.
* 🟢 **Low Risk / Completed (Green):** Normal behavior, or Resolved Tasks.

## 3. Key Interface Views

### A. Dashboard Overview
* **Widgets:** Total Tasks, Pending Tasks, Completed Tasks with mini-charts (bar/line).
* **Task Calendar:** Horizontal timeline view showing upcoming deadlines.
* **Recent Activity:** List of latest verification uploads or ticket updates.

### B. Customs Kanban Board (ATS to Customs Workflow)
* **Columns:** `To Do` (Backlog), `In Progress`, `Completed`.
* **Card Details (Mini-view):** * Priority Tag (High/Medium/Low based on ATS input).
  * Application Number & IIN.
  * Preview of Rejection Reason.
  * Assignee Avatar (Customs Officer).
  * Attachments count icon.
* **Card Modal (Expanded):** Displays all fields from the `Support_Tickets` schema. Allows ATS to edit `support_comment` and Customs to edit `customs_comment` and change `status`.

### C. IMEI Verification Report Modal
* Top: Statistics (e.g., "Imei1 — found X of Y").
* Middle: Highlighted 15-digit matches.
* Bottom: Paginated exact line-by-line verification table.
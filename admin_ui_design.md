# Shukisha Admin Panel: UI Design Specification

This document outlines the layout, component architecture, and detailed role-based views for the Shukisha Admin Dashboard.

## 🎨 Design Philosophy (Modern SaaS — Luminare-Inspired)

The Admin Dashboard adopts a **Premium Modern SaaS** aesthetic inspired by the Luminare dashboard reference — warm, readable, and data-dense:

*   **KPI-First Layout**: Top row of rounded cards showing critical metrics (Revenue, Parcels, Active Drivers, SLA Breaches) with colored icon circles and percentage change badges.
*   **Pill Navigation**: Top nav bar with pill-shaped active tab indicator — clean, minimal, instantly scannable role switching.
*   **Warm Tones**: Off-white/warm grey backgrounds (`#F5F5F0`) with pure white cards and subtle shadow elevation (`0 1px 3px rgba(0,0,0,0.04)`).
*   **Rounded Everything**: Cards use `border-radius: 16px`, buttons use `10px`, pills use `999px`. No harsh edges.
*   **Data-Dense Kanban**: Living pipeline columns (Assigned → In Transit → SLA Breach) with inline status badges and monospaced IDs.
*   **Bar Charts**: Simple CSS-driven bar charts with hover tooltips for platform performance tracking.

---

## 🏗️ Global Layout & Triage Layer

*   **Shell Architecture**: 
    1.  Top bar (52px)
    2.  Role bar (40px)
    3.  Sidebar (220px)
    4.  Main content area
*   **Role Switcher**: The role tab bar sits at the shell level. It switches the entire sidebar navigation and KPI set without a page reload. Admins can hot-swap between Transport, Logistics, Finance, and Support.
*   **Triage Alert Strip**: Every page opens with an alert strip (typically Amber/Red) localized to that role if there are active issues. Action over information.

---

## 🖌️ Design System Tokens (Tailwind)

| Token | Hex | Used for |
|---|---|---|
| Deep Black | `#0F0F0F` | Primary text, massive headers, thick borders, circular stamps |
| Paper/Cream | `#F4F4F0` | Main shell background, negative space |
| Stark White | `#FFFFFF` | Core data cards (to slightly contrast against Paper) |
| Monospace Gray| `#6B7280` | Log outputs, timestamps, secondary labels |
| Accent Emerald| `#10B981` | Required for operational clarity (Delivered/Online) but applied minimally as a harsh pixel block or thin line. |
| Accent Red | `#EF4444` | Required for SLA breaches and Offline states. |

**CSS Typography Directives:**
*   `font-display`: Massive, dense, uppercase sans-serif (e.g., `font-black uppercase tracking-tighter leading-none`).
*   `font-mono`: Tiny, spaced-out monospace for data and IDs (e.g., `text-[10px] tracking-widest uppercase`).

---

## 📦 1. Logistics Admin (Parcels)

**Goal**: Eradicate delivery bottlenecks and SLA breaches.

*   **Triage Alert Strip**: E.g., `[Red] 3 Parcels Exceeded Delivery SLA (> 3 Hrs)`
*   **Primary View**: The Parcel Lifecycle Kanban/Table.
    *   A clean data table organized by status tabs: `AWAITING_PAYMENT`, `ASSIGNED`, `PICKED_UP`, `IN_TRANSIT`, `PENDING_DELIVERY`.
*   **Signature Component: The Parcel Timer Ring**.
    *   A visual SVG circle path using `stroke-dashoffset` bound to the elapsed time.
    *   **Green Ring ( < 1hr )**: Moving on time.
    *   **Amber Ring ( 1 - 3hr )**: Nearing threshold, stroke width slightly pulses on hover.
*   **Action Flows**:
    1.  **Handover Override Modal**: A highly secure dialog to "Force Complete" a delivery if the receiver's phone is off but delivery is confirmed. Requires the Admin to input a mandatory text `Reason` for the audit log.
    2.  **Re-assignment Drop**: Ability to unassign a stalled driver and re-assign the parcel payload to another online driver active on the same route. 
    3.  **Dispute Triage Console**: A split-screen review deck for dealing with reported issues.
        *   **Left Column**: Timeline of the parcel journey, sender complaint text, and exact geolocation points of the driver.
        *   **Right Column**: Courier proof of delivery (uploaded photo) and secure action buttons (`Refund Sender`, `Dismiss Dispute`, `Compensate Driver`). Admin must supply a mandatory text reason.

---

## 🚐 2. Transport Admin (Fleet Operations)

**Goal**: Monitor real-time driver availability, route saturation, and fleet compliance.

*   **Triage Alert Strip**: E.g., `[Amber] 5 Drivers idle on Route 125 for 45+ mins`
*   **Primary View**: Live Fleet Map (Mapbox GL JS).
    *   Map dominates the viewport. Pulsing dot markers (Emerald = Free, Blue = Active Parcel/Stop, Red = Offline/Disconnected heartbeat).
    *   **Sidebar Drawer (default)**: Route filters. Admins can toggle "Show Route 125 only".
*   **Action Flows**:
    1.  **Driver Comm Drawer**: Clicking a driver marker slides out the entity sidebar. It overtakes the UI to show Driver Name, Plate, SACCO, Current Speed, and Last Heartbeat. 
    3.  **Path Tracing**: Toggle "Show Trail" to draw the polyline of the driver's last 30 minutes of logged heartbeats to verify deviations.
    4.  **Route Setup Console**: A specialized data-entry view to configure system routing.
        *   **Visual Map Editor**: Click on the map to define matatu stages and draw the polyline path between them.
        *   **Parameter Table**: Input the Route Name, Base Fare Rule, Distance Multiplier, and Active Status. Real-time API `POST /v1/admin/routes` invocation.

---

## 💰 3. Finance Admin (Revenue & Ledger)

**Goal**: Ensure automated financial reconciliation, manage payout liquidity, and configure platform commissions.

*   **Triage Alert Strip**: E.g., `[Red] M-Pesa B2C Balance Dropping Below KES 10,000` or `[Amber] 15 Payouts Pending Execution`.
*   **Primary View**: Payout Ledger & Revenue Analytics.
    *   Top row features massive numerical KPI cards (Today's Revenue, Outstanding Payout Liability, M-Pesa Float).
    *   Line chart (D3/Recharts) tracking daily successful M-Pesa STK volumes.
*   **Signature Component: 3-Step M-Pesa Payout Ceremony**.
    *   Because payouts are irreversible, the UI enforces friction.
    *   *Step 1*: Filter and Select target drivers in the Payout Queue (e.g., Drivers with balances > KES 5,000).
    *   *Step 2*: Summary Review Modal outlining total transaction deduction vs. current M-Pesa B2C Float.
    *   *Step 3 - The Lock*: A text input field requires the Admin to literally type `"CONFIRM"` into an empty box. Only then does the greyed-out button turn **Red** and activate the API call.
*   **Commission Adjuster**: A globally permissioned slider module to globally tweak platform fee percentages applied at checkout.

---

## 🛡️ 4. Support Admin (User Success)

**Goal**: Unblock users experiencing auth failures, payment issues, or disputes.

*   **Triage Alert Strip**: E.g., `[Red] 48 Failed OTP requests in the last hour` or `[Amber] 3 New Dispute Tickets`.
*   **Primary View**: The 360° Search Console.
    *   A massive, globally focused input field centered on the page: `Search Phone Number, OTP Code, Parcel ID, or Plate...`
*   **Signature Component: The Activity Timeline**.
    *   Once a user is located, the UI lists a literal vertical timeline of their existence. E.g.:
        *   `14:02 - Sent SMS OTP (Twilio)`
        *   `14:05 - Failed Login (Rate Limit Exceeded)`
        *   `14:15 - Admin bypassed OTP`.
*   **Action Flows**:
    1.  **Account Rescue**: If a user is blocked by OTP limits or telecom drops, the Support Admin clicks "Rescue Auth". This creates a scoped, 5-minute manual verification token bypassing SMS, generating a fixed PIN to give the user over the phone.
    2.  **Audit Flagging**: If a malicious user/driver is found, a red toggle triggers `Suspend User`. It opens a mandatory text area for the reason, immediately logging out all their active sessions via the `internal/identity` module.

---

## 🛠️ Tech Stack Recommendation

*   **Frontend Shell**: SvelteKit + Vite (No SPA loading lag, highly reactive).
*   **Styling**: Tailwind CSS (Leveraging utility classes for ultra-thin 0.5px borders and strict typographic scaling).
*   **Charts & Vectors**: Native Svelte SVG manipulation for the Parcel Timer ring and minimal charts.
*   **Mapping**: Mapbox GL JS for real-time tracking (leveraging PostGIS).
*   **Icons**: Lucide (Clean, modern 1.5px stroke symbols).

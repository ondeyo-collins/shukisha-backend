# Shukisha MVP — Logistics Department Scope (Operations)

**Status:** Draft (implementation-aligned)  
**Version:** 1.0  
**Date:** 2026-04-07  

## 1) Department purpose

The Logistics Department owns the **operational lifecycle of parcels** moving through Shukisha.

Success means parcels:
- are visible end-to-end
- progress through the pipeline on time
- are delivered with verifiable proof
- have exceptions resolved quickly (delays, failed delivery, disputes)

## 2) Ownership boundaries (what Logistics owns / does not own)

### Logistics owns
- Monitoring parcel flow (pipeline visibility)
- Operational intervention for stuck/delayed parcels
- Delivery exception handling (attempts, follow-up, escalation)
- Operational overrides **when policy allows** (always audited)
- Dispute/claim intake (MVP v1.1)
- Operational notes (MVP v1.1)

### Logistics does NOT own
- Driver onboarding / verification (Transport)
- Routes & stops creation (Transport)
- Payout execution / reconciliation (Finance)
- Account recovery / user unlock (Support)
- Admin user management and global configuration (SUPER_ADMIN)

## 3) Role and RBAC policy

### Role: `LOGISTICS`
Can:
- view all parcels and their timelines
- filter/search parcels by status, date, receiver phone, route, driver
- perform allowed operational actions:
  - override/force-complete (MVP v1: exists) **with mandatory reason** (audited)
  - driver notification triggers (MVP v1.1)
  - claims/disputes intake (MVP v1.1)
  - internal notes (MVP v1.1)

Cannot:
- manage admin users
- modify routes/stops
- execute payouts
- verify drivers

### Role: `SUPER_ADMIN`
- Full access including all Logistics actions.

## 4) Data model: STATUS vs TRACKING EVENTS (critical distinction)

Shukisha represents parcel lifecycle using **two related concepts**:

### 4.1 Canonical parcel status (`parcels.parcel.status`)
This is the **current state** used for filtering/queues and operational decisions.

**MVP v1 (current backend implementation)**
- `DRAFT`
- `AWAITING_PAYMENT`
- `PAID`
- `ASSIGNED`
- `PICKED_UP`
- `ARRIVED_AT_DROPOFF`
- `DELIVERY_ATTEMPTED`
- `DELIVERED`

> Rule (locked): `parcel.status` must be a fixed enum. Do not set status to arbitrary tracking event strings.

### 4.2 Tracking events (`parcels.parcel_tracking_events.event_type`)
This is an **append-only operational history** (timeline).

**MVP v1 (current backend examples)**
- `BOOKED`
- `PAID`
- `DRIVER_ASSIGNED`
- `PICKUP_CONFIRMED`
- `CHECKPOINT_REACHED`
- `ARRIVED_AT_DROPOFF`
- `DELIVERY_ATTEMPTED`
- `DELIVERED`
- `DELIVERED_OVERRIDE` (admin override)

> Rule (locked): events are history. They should not automatically overwrite `parcel.status` unless there is an explicit mapping.

## 5) Security & data minimization (Logistics)

### 5.1 OTP handling (locked)
- Admin UI **must never display** pickup or delivery OTP values.
- Admin UI may display only:
  - “OTP verified: yes/no”
  - timestamps and actor (driver/admin) where relevant

> Note: the current backend `Parcel` model includes `pickup_otp` and `delivery_otp` fields. Admin-facing responses should remove these in a follow-up hardening step.

### 5.2 PII visibility
- Parcel list: mask phone numbers by default (e.g. `+2547•••1234`)
- Parcel detail: full phone may be shown for Logistics operators
- All sensitive actions must be audited

## 6) Logistics dashboards (Admin UI requirements)

### 6.1 Logistics home route
- `LOGISTICS` role home: `/logistics`

### 6.2 Main dashboard components

**Pipeline view**
- Table/Kanban of active parcels with:
  - parcel ID
  - created time / last updated time
  - current `status`
  - payment status (if available)
  - assigned driver (if available)
  - route (if available)
  - exception flags (delay / attempts / override)

**Filters**
- by `status`
- by date range
- by receiver phone
- by driver ID / plate (if available)
- by route ID (if available)
- by “stuck threshold” (e.g. > 30m since last update)

**KPIs**
- total active parcels
- parcels awaiting assignment (e.g. `PAID`)
- parcels in transit (`PICKED_UP` etc.)
- delivery attempts
- overrides performed (today/week)

### 6.3 Parcel detail drawer
Must show:
- parcel identifiers (ID, route)
- sender/receiver (phone)
- assigned driver info
- current status
- timeline (`tracking_events`) with timestamps + remarks
- payment info (amount, status)
- override history + reason (if overridden)
- action buttons limited to Logistics role permissions

## 7) Operational workflows owned by Logistics

### Workflow A — Active monitoring (MVP v1)
- Open `/logistics`
- Identify “PAID but not assigned” parcels
- Identify stale parcels based on last update
- Drill into parcel detail timeline

### Workflow B — Stuck parcel intervention (MVP v1.1)
- Filter by stale threshold
- Review timeline and assigned driver
- Take action:
  - record internal note
  - notify driver
  - escalate to Transport/SUPER_ADMIN

### Workflow C — Delivery exception handling (MVP v1)
- Review attempts and `DELIVERY_ATTEMPTED` events
- Decide to:
  - request retry (v1.1)
  - open dispute/claim (v1.1)
  - override delivery if policy allows (v1 exists)

### Workflow D — Secure override (MVP v1)
- Require:
  - mandatory reason
  - audit log entry
- Effect:
  - mark parcel delivered operationally
  - append `DELIVERED_OVERRIDE` tracking event
  - (future) notify affected parties

## 8) Backend support for Logistics

### 8.1 Existing endpoints (MVP v1)
- `GET /v1/admin/parcels`  
  Returns parcel list (currently includes tracking events + payment preloaded)
- `POST /v1/admin/parcels/{id}/override`  
  Force-complete delivery with a reason (must be audited)

### 8.2 Required additions (MVP v1.1 recommended)

**Parcel detail**
- `GET /v1/admin/parcels/{id}` (full parcel record + timeline)

**Events**
- `GET /v1/admin/parcels/{id}/events` (timeline only)

**Notes**
- `POST /v1/admin/parcels/{id}/notes` (internal notes, not visible to users)

**Notify driver**
- `POST /v1/admin/parcels/{id}/notify-driver` (records action + dispatches notification)

**Claims/disputes**
- `POST /v1/admin/parcels/{id}/claims`
- `GET /v1/admin/claims?status=...`
- `PATCH /v1/admin/claims/{id}`

## 9) Audit requirements (locked)

Every sensitive Logistics action must create an immutable audit event:
- parcel override (force-complete)
- dispute/claim opening
- manual state correction
- escalation
- driver notification triggered by admin
- any action that reveals sensitive information (if ever added)

Audit record must include:
- `admin_id`
- action name (e.g. `PARCEL_OVERRIDE_DELIVERED`)
- `entity_id` = parcel ID
- human-readable details (reason, context)

## 10) Acceptance checklist (Logistics MVP)

- LOGISTICS user can:
  - load `/logistics` successfully
  - view parcels list and statuses
  - open parcel detail and see timeline
  - perform override with a mandatory reason
- LOGISTICS user cannot:
  - access finance endpoints (403)
  - access transport endpoints (403)
  - access superadmin endpoints like `/admin/management` (403)
- Admin UI:
  - never displays OTP values
  - shows a stable sidebar (no flicker/redirect loops)


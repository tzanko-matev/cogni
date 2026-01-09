# Technical Specification: High-Performance Rate Limiting Service

**Version:** 1.0 (Draft) | **Language:** Go | **Backend:** TigerBeetle / Memory

## 1. Executive Summary

We are building a generic Rate Limiting Proxy for LLM workloads.
Unlike standard rate limiters, our system must handle **variable costs** (unknown token usage) and **multi-resource dependencies** (a request needs both User Credits and Provider Quota).

**Key Constraints:**

1. **Throughput First:** We must maximize usage. A stalled resource (e.g., OpenAI) must never block requests to an available resource (e.g., Anthropic).
2. **Deployment Flexibility:** The system must run in **Distributed Mode** (using TigerBeetle for scale) AND **Standalone Mode** (single binary using in-memory maps).
3. **Risk-First Approach:** We will build 3 throw-away prototypes to validate our assumptions before writing the final server code.

---

## 2. System Architecture

The system follows a **Proxy/Middleware** pattern. Clients do not connect to the database directly. They connect to our Go Service, which aggregates requests and manages the accounting backend.

### 2.1 The "Optimistic" Lifecycle

Because we don't know the final token count of an LLM request, we use a two-phase commit strategy:

1. **Phase 1: Entry (Reservation)**
* **Action:** Reserve `Estimated_Cost` (Input tokens + Buffer).
* **Mechanism:** Atomically debit *both* the `User` and the `Provider` accounts.
* **Result:** If successful, the request proceeds. If failed, return 429.


2. **Phase 2: Settlement (Async)**
* **Action:** After the request, calculate `Delta = Actual_Cost - Estimated_Cost`.
* **Mechanism:** If `Delta > 0`, debit the difference.
* **Fallback:** If the User runs out of funds during Settlement, we move the remaining charge to a "Debt Account" (negative balance allowed) so the system tracks the overage without crashing.



---

## 3. The Implementation Strategy (Phased)

We will not build the full API yet. We will build **3 Isolated Prototypes** to prove the core mechanics.

### Prototype 1: The Data Integrity Check

**Goal:** Prove we can strictly enforce multi-resource limits using TigerBeetle's Linked Transfers.
**Scope:** A simple Go script (`cmd/proto1/main.go`).

**Requirements:**

1. **Deterministic Hashing:** Implement `StringToU128(s string)`. It must be consistent (e.g., FNV-1a).
2. **Scenario:**
* Create Account A (Provider) with **1 credit**.
* Create Account B (User) with **100 credits**.


3. **The Atomic Test:**
* Attempt to debit **2 credits** from A and **2 credits** from B in a *single linked chain*.
* **Expected Result:** The transfer fails because A has insufficient funds. **Crucially:** Account B must remain at 100. If B drops to 98, the test fails.



### Prototype 2: The "Fair" Multiplexer

**Goal:** Prove we can solve Head-of-Line Blocking.
**Scope:** A Go benchmark (`cmd/proto2/main.go`). **No TigerBeetle required.**

**The Problem:**
If we use a single `chan Request` for all traffic, 10,000 requests for a slow provider will block 1 request for a fast provider.

**Requirements:**

1. **Mock Backends:** Create a "Slow" mock (sleeps 100ms) and a "Fast" mock (sleeps 1ms).
2. **The Solution (The Multiplexer):**
* Implement a `map[string]chan Request` (one queue per provider).
* Run a background worker that performs **Weighted Round Robin**:
* Take 5 items from Queue A -> Add to batch.
* Take 5 items from Queue B -> Add to batch.
* Process batch.




3. **Validation:**
* Flood the "Slow" queue with 1,000 requests.
* Push 1 request to the "Fast" queue.
* **Success:** The Fast request completes in <20ms (verifying it "skipped the line").



### Prototype 3: The Pluggable Interface

**Goal:** Enable "Single Binary" distribution.
**Scope:** Go Interface Design (`pkg/ledger/`).

**Requirements:**
Define the contract that abstracts the storage engine:

```go
type Ledger interface {
    // Reserve checks balances and holds funds. 
    // Returns a reservationID (string) for later settlement.
    Reserve(ctx context.Context, userID, providerID string, amount uint64) (string, error)

    // Settle finalizes the transaction.
    Settle(ctx context.Context, reservationID string, actualCost uint64) error
}

```

**Task:**

1. Implement `MemoryLedger`: A thread-safe struct using `map[string]int64` and `sync.Mutex`.
2. Implement `TigerBeetleLedger`: A wrapper around the logic from Prototype 1.
3. **Main Switch:** Create a `main.go` that selects the backend based on a flag:
* `./app --mode=local` (Uses Memory)
* `./app --mode=cluster` (Uses TigerBeetle)



---

## 4. Technical Reference (For the Final Build)

*Only proceed to this after the prototypes are approved.*

### 4.1 TigerBeetle Schema

We use 3 Account Types (distinguished by the `Code` field):

| Name | Code | Flags | Notes |
| --- | --- | --- | --- |
| **Limit Account** | 1 | `linked` | `debits_must_not_exceed_credits` |
| **Debt Account** | 2 | `linked` | Unbounded. Tracks "Overage" if Settlement fails. |
| **System Sink** | 99 | None | Where spent credits are sent. |

### 4.2 The Settlement Logic (Pseudo-code)

The `Settle` function in the Proxy handles the "Overage" edge case:

```go
func (s *Service) Settle(resID string, actual uint64) {
    diff := actual - s.getEstimated(resID)
    if diff <= 0 { return } // Already paid enough

    // Try to pay the difference normally
    err := s.ledger.Debit(user, diff) 
    
    // EDGE CASE: User ran out of money during the call
    if err == ErrExceedsCredits {
        // Force the debit into the Debt Account (no limits)
        // so we record that they owe us money.
        s.ledger.DebitDebtAccount(user, diff)
    }
}

```

# ADR 0011: No Authn/Authz in v1 (Network Isolation Only)

## Status

- Proposed

## Context

- We need a working v1 rate limiter quickly.
- Adding authentication and tenant scoping requires extra infrastructure (keys, identity, policy).

## Decision

- v1 has no authentication or authorization on `ratelimiterd` endpoints.
- Deployment must ensure the service is reachable only by trusted clients (private network, firewall rules).

## Specification

- All endpoints (`/v1/reserve`, `/v1/complete`, `/v1/admin/limits`, batch variants) accept requests without auth headers.
- Security relies on network isolation and environment-level controls.

## Consequences

- Positive: Faster implementation and easier local usage.
- Negative: Not safe for untrusted networks; must be wrapped by infra controls.

## Alternatives considered

- API key per tenant (rejected for v1 complexity).
- mTLS between client and server (rejected for v1 complexity).

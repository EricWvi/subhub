# Phase 2: Model Transformation Functional Spec

## Purpose

This document defines what Phase 2 of SubHub must deliver for future contributors. It is intentionally focused on product behavior, contributor expectations, and completion criteria rather than implementation details. The goal is to make the intended user-facing behavior unambiguous before contributors expand the data model and provider management experience.

## Background

Phase 1 establishes the basic ingest-to-output foundation: provider data can be fetched, normalized into a Mihomo-native representation, and served as a usable unified output. That foundation proves that SubHub can collect and republish subscription content reliably.

Phase 2 exists because a unified output alone is not enough for meaningful management. Contributors now need to support three product-level capabilities that make provider data manageable and inspectable:

- users can define their own provider abbreviations from the web page
- the providers page can show subscription usage and limit information derived from `Subscription-Userinfo`
- each proxy node is stored as its own managed record rather than existing only inside a provider snapshot

Together, these changes move SubHub from provider-level caching toward a node-aware product model that later phases can build on for grouping, rule automation, and scoring.

## Phase Outcome

At the end of Phase 2, SubHub should be able to:

- let a user assign a custom abbreviation to each provider through the web page
- preserve abbreviations even when multiple providers choose the same value
- display provider-related subscription usage information on the providers page when upstream metadata is available
- retain individual proxy nodes as separate stored entities associated with their source provider
- treat stored nodes as the canonical basis for later transformation and output workflows

Phase 2 is successful when contributors can point to a provider management experience that is no longer limited to whole-provider snapshots and can instead support provider metadata plus node-level persistence.

## Product Principles

Phase 2 work should follow these principles:

- user-defined labels should be simple and predictable
- provider metadata should be visible when available and non-disruptive when unavailable
- node-level persistence should improve clarity rather than introduce hidden transformation rules
- provider identity and node identity should remain understandable to contributors and users
- Phase 2 should enable later roadmap work without prematurely introducing grouping, scoring, or automation logic

## In Scope

Phase 2 includes three product capabilities.

### 1. Custom Provider Abbreviations

SubHub must allow users to define a custom abbreviation for each provider directly from the web page.

Expected behavior:

- each provider has an editable abbreviation field in the web interface
- abbreviations are user-defined rather than system-generated
- abbreviations accept only uppercase letters
- abbreviations do not need to be unique across providers
- a provider can still be understood and managed even if another provider uses the same abbreviation
- invalid abbreviation input is rejected clearly rather than silently changed into another value
- abbreviations are fields displayed with names, not replacing

### 2. Provider Subscription Usage Visibility

The providers page must display subscription usage and limit information derived from `Subscription-Userinfo` whenever that information is available from the upstream provider response.

Expected behavior:

- provider entries can show relevant subscription usage details associated with the latest known provider state
- usage information is presented as provider metadata, not as a separate advanced analytics system
- if upstream usage metadata is unavailable, malformed, or absent, the providers page remains usable and communicates that the information is not available
- usage information shown on the page reflects the latest known usable provider data rather than requiring a live fetch during page view

This capability exists to make provider management more transparent without expanding Phase 2 into full monitoring or billing logic.

### 3. Individual Proxy Node Storage

SubHub must persist each proxy node separately rather than treating transformed nodes only as a single bulk payload attached to a provider snapshot.

Expected behavior:

- each fetched provider can contribute multiple individually stored proxy nodes
- stored nodes remain associated with their source provider
- the normalized yaml of each provider remains untouched
- node persistence reflects the transformed node representation that SubHub intends to manage going forward
- contributors can reason about node-level changes without parsing an entire provider payload as one opaque object
- later workflows can use stored nodes as the basis for output generation and future matching logic

This capability exists to establish the product’s first durable node-level model.

## Core Workflows

### Workflow 1: Set a Provider Abbreviation

1. A user opens the providers page.
2. The user edits the abbreviation for a provider.
3. The system validates the input against the uppercase-letters-only rule.
4. lowercase letters are automatically transformed to uppercase letters when inputing.
5. If the input is valid, the updated abbreviation is saved and shown as part of that provider’s displayed state.
6. If the input is invalid, the change is not accepted and the user is informed that only uppercase letters are allowed.

Notes:

- this workflow does not require abbreviation uniqueness
- the provider remains identifiable by its broader provider record even if another provider shares the same abbreviation

### Workflow 2: Review Subscription Usage on the Providers Page

1. A user opens the providers page.
2. The page shows provider details, including subscription usage information when it exists in the latest known provider metadata.
3. The user can review usage-related values without triggering a new refresh.
4. If usage information is unavailable for a provider, the page communicates that the information is unavailable instead of showing misleading placeholder values.

Notes:

- this workflow is about visibility, not quota enforcement
- the page should distinguish between known values and unavailable values

### Workflow 3: Refresh a Provider and Persist Nodes Individually

1. A provider is refreshed through the normal refresh path.
2. The refreshed provider content is transformed into the product’s managed node representation.
3. Each resulting proxy node is stored as its own record associated with the source provider.
4. The provider’s latest subscription metadata and node set become the latest known usable state for later product workflows.
5. Future output-oriented workflows can rely on those stored individual nodes rather than only on a bulk snapshot blob.

Notes:

- this workflow does not require advanced deduplication across providers
- this workflow does not require grouping, scoring, or rule assignment

## Functional Requirements

Phase 2 should satisfy the following requirements:

1. A user can enter and update a custom abbreviation for each provider from the web page.
2. Abbreviations accept uppercase letters only.
3. Two or more providers may share the same abbreviation without causing either provider to become invalid.
4. Invalid abbreviation input is rejected clearly.
5. The providers page displays `Subscription-Userinfo` related information when the latest known provider data includes it.
6. The absence of `Subscription-Userinfo` data does not block provider display or management.
7. Each transformed proxy node is stored separately as an individual managed record.
8. Individually stored nodes remain linked to their source provider.
9. Node-level persistence becomes the product basis for later transformation and output workflows in this phase and beyond.

## Out of Scope

To protect Phase 2 from expanding into later roadmap work, the following are explicitly out of scope:

- uniqueness rules for provider abbreviations
- automatic abbreviation generation or suggestion logic
- advanced subscription analytics, forecasting, alerting, or billing workflows
- cross-provider node deduplication or intelligent merge policies
- regex grouping, rule injection, or policy assignment
- health checks, scoring, ranking, or node quality decisions
- multi-user permission models or audit history for abbreviation changes

Contributors should avoid turning Phase 2 into an intelligence or automation phase. The goal is better structure and visibility, not advanced decision-making.

## Quality Expectations

Minimum quality expectations for Phase 2:

- abbreviation behavior should be consistent everywhere it appears
- invalid abbreviation input should fail clearly and predictably
- provider usage information should never imply certainty when the upstream data is missing or malformed
- node-level persistence should be dependable enough for later phases to treat individual nodes as stable product entities
- the providers page should remain understandable even when some providers have incomplete metadata

## Acceptance Criteria

Phase 2 can be considered complete when all of the following are true:

- a contributor can demonstrate that each provider has a user-editable abbreviation on the web page
- abbreviation input accepts uppercase letters only
- two different providers can be saved successfully with the same abbreviation
- invalid abbreviation input is rejected with clear feedback
- the providers page shows `Subscription-Userinfo` related data for providers whose latest known state includes it
- providers that do not expose usable `Subscription-Userinfo` data still display correctly on the page
- after a provider refresh, the resulting proxy nodes are persisted as separate records rather than only as one bulk transformed payload
- each stored node can be traced back to its source provider
- later contributor work can treat individually stored nodes as the basis for downstream transformation and output behavior

## Definition of Done

Phase 2 is done when SubHub can support provider-level management and visibility with a node-aware data model:

- providers can carry user-defined abbreviations
- subscription usage metadata can be reviewed from the providers page
- transformed nodes are persisted individually and associated with their provider

If those three things work reliably and within scope, future contributors can move on to Phase 3 without needing to redesign the provider-facing model introduced in Phase 2.

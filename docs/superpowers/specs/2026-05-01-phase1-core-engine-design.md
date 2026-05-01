# Phase 1: Core Engine Product Spec

## Purpose

This document defines what Phase 1 of SubHub must deliver for future contributors. It is intentionally focused on product behavior, scope, and quality expectations rather than implementation details. The goal is to make it clear what "done" means before deeper engineering work begins.

Phase 1 establishes the minimum viable foundation of SubHub: take raw provider subscriptions, turn them into a reliable Mihomo-native representation, and produce a usable Mihomo-first subscription output.

## Phase Outcome

At the end of Phase 1, SubHub should be able to:

- accept one or more provider sources as inputs
- refresh those sources on a recurring basis
- retain the most recent usable provider data locally
- normalize diverse provider formats into a single Mihomo-native proxy representation
- generate a basic but valid unified subscription output for Mihomo

Phase 1 is successful when contributors can point to a complete ingest-to-output flow, even if the system is still minimal and not yet intelligent.

## Why This Phase Exists

SubHub cannot deliver advanced grouping, rule automation, scoring, or UI management until it can reliably do three things:

1. collect provider data without requiring manual handling
2. understand inconsistent upstream formats well enough to standardize them
3. publish a stable merged output that downstream clients can consume

Phase 1 is therefore about trust and foundation, not sophistication.

## Primary Audience

This spec is written for future contributors who will build Phase 1. It should help them answer:

- what problem this phase solves
- what behaviors are required
- what is intentionally out of scope
- what standards must be met before Phase 1 can be considered complete

## Product Principles

Phase 1 work should follow these principles:

- reliability over feature breadth
- Mihomo compatibility before broader ecosystem support
- normalization before customization
- predictable output over clever transformation
- explicit scope control to avoid leaking Phase 2 and later concerns into the foundation

## In Scope

Phase 1 includes three product capabilities.

### 1. Provider Intake and Refresh

SubHub must be able to work from external provider sources and keep them reasonably current without requiring manual retrieval each time.

Expected behavior:

- contributors can define provider sources that SubHub will treat as upstream inputs
- SubHub can refresh those sources repeatedly over time
- if a provider temporarily fails, previously usable data should not be discarded without replacement
- refresh behavior should be predictable enough that users can trust the output is based on recent provider data

This capability exists to ensure SubHub behaves like an aggregator rather than a one-off conversion tool.

### 2. Provider Parsing and Normalization

SubHub must convert inconsistent upstream subscription content into a single Mihomo-native representation of nodes.

Expected behavior:

- SubHub can accept the common provider payload styles anticipated for early Mihomo/Clash usage
- upstream formatting differences do not leak into the rest of the product
- the same Mihomo-compatible proxy map shape is produced regardless of which supported provider format supplied it
- malformed or incomplete provider data is handled in a controlled way rather than silently corrupting the output

This capability is the heart of Phase 1. Without normalization, later features such as grouping, scoring, and rule logic cannot be built cleanly.

### 3. Unified Mihomo Output

SubHub must expose a basic unified output that combines normalized nodes into a usable subscription result.

Expected behavior:

- the output is structured for Mihomo first
- multiple provider inputs can contribute nodes to one combined result
- the generated output is stable and predictable for repeated consumer use
- the output includes the essential node data required for client consumption
- the output can be based on a fixed template as long as the merged node content is correct

Phase 1 does not need to produce a highly customized configuration. It only needs to prove that the full pipeline works and yields a practical result.

## Out of Scope

To protect Phase 1 from expanding into later roadmap work, the following are explicitly out of scope:

- automatic proxy grouping by regex, category, or intent
- rule management, injection, or validation beyond what is minimally necessary for a basic output
- health checks, scoring, ranking, or automatic node quality decisions
- user dashboards or visual management tools
- authentication, multi-user concerns, or deployment hardening
- advanced traffic analysis such as ingress, egress, or location verification
- custom scripting or webhook automation

Contributors should resist adding "almost related" features from later phases unless they are strictly required to make the Phase 1 pipeline function.

## Functional Requirements

Phase 1 should satisfy the following requirements:

1. A contributor can identify at least one provider source and have SubHub retrieve it repeatedly over time.
2. SubHub preserves a usable local copy of provider data so output generation is not wholly dependent on a live upstream response at request time.
3. Supported provider input formats are transformed into one standard Mihomo-native node representation.
4. Unsupported or broken provider content does not produce misleading output.
5. SubHub can generate one unified Mihomo-oriented configuration result from normalized nodes.
6. The generated result remains understandable and predictable even when multiple providers contribute data.
7. Failure in one provider should not make all healthy provider data unusable by default.

## Quality Expectations

Even without advanced features, contributors should treat Phase 1 as production-like in behavior.

Minimum quality expectations:

- refresh and output behavior should be deterministic enough to debug
- invalid upstream data should be visible as an error or rejection, not hidden
- repeated output generation from unchanged inputs should produce equivalent results
- the system should favor preserving a last known usable state over returning obviously broken output
- contributor-facing behavior should be simple enough that future phases can build on it without rethinking core assumptions

## Mihomo-First Requirement

Phase 1 should prioritize Mihomo-specific compatibility and assumptions. This does not mean other Clash-family formats are forbidden forever; it means contributors should optimize early product decisions around the client family that gives SubHub the most flexibility for future aggregation features.

When a choice must be made between generic compatibility and a cleaner Mihomo-first foundation, Phase 1 should prefer the Mihomo-first path unless it creates obvious future lock-in.

## Acceptance Criteria

Phase 1 can be considered complete when all of the following are true:

- SubHub can ingest more than one provider source
- provider data can be refreshed without manual intervention
- a usable prior state is retained when an upstream provider is temporarily unavailable
- supported provider payloads are normalized into one Mihomo-native node model
- bad provider input is rejected or surfaced clearly
- a single unified Mihomo-oriented output can be generated from normalized nodes
- the resulting output is stable enough for a contributor to treat it as the baseline for later phases

## What Phase 1 Does Not Need to Prove

Phase 1 does not need to prove that SubHub is already smart. It only needs to prove that SubHub is dependable at the core workflow.

It is acceptable at the end of Phase 1 if:

- output grouping remains simple
- configuration templating remains static
- provider support is intentionally narrow
- failure reporting is basic but clear
- operational controls are minimal and contributor-oriented

## Risks to Watch During Implementation

Contributors should be alert to the following scope and product risks:

- allowing provider-specific quirks to leak into the normalized model
- building output logic that assumes only one provider exists
- treating temporary upstream failure as total data loss
- over-designing for future phases before Phase 1 is stable
- expanding the output surface before the normalization contract is trustworthy

## Definition of Done

Phase 1 is done when SubHub has a dependable end-to-end foundation:

- upstream provider data can be collected
- that data can be standardized into one Mihomo-native model
- the standardized data can be published as one basic Mihomo-ready output

If those three things work reliably and within the intended scope, future contributors can move on to Phase 2 without needing to redesign the foundation.

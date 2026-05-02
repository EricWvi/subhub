# Phase 4: Subscription Services Functional Spec

## Purpose

This document defines what Phase 4 of SubHub must deliver for future contributors. It is intentionally limited to product behavior, user expectations, and completion criteria. It does not prescribe implementation details, storage design, API shapes, or specific technologies.

## Background

Earlier phases establish the managed building blocks that SubHub now owns:

- providers can be fetched and normalized into managed proxy nodes
- internal proxy groups can organize those managed proxy nodes
- manual rules can target those internal proxy groups

By the end of Phase 3.5, SubHub can model providers, nodes, internal proxy groups, and rules, but it still does not expose reusable subscription products built from that managed state.

Phase 4 exists to turn those internal building blocks into user-consumable subscription services. Users need three related outputs:

- a full Clash or Mihomo configuration subscription for direct client use
- a proxy provider subscription that publishes only the proxy nodes selected by one internal proxy group
- a rule provider subscription that publishes only the rules selected by one internal proxy group

This phase introduces a clear distinction between internal proxy groups and output-facing `proxy-group` entries inside a Clash config subscription.

- internal proxy groups are SubHub-managed source selections created in earlier phases
- output-facing `proxy-group` entries are the `proxy-group` objects a user assembles when defining a Clash config subscription

Phase 4 is successful when contributors can clearly understand how those layers relate and can build subscription outputs without redefining the meaning of internal proxy groups, output-facing `proxy-group` entries, proxy nodes, or rules.

## Phase Outcome

At the end of Phase 4, SubHub should be able to:

- let a user create and manage a Clash config subscription from the web page
- let that user assemble output-facing `proxy-group` entries for that subscription
- let each output-facing `proxy-group` reference other output-facing `proxy-group` entries
- let each output-facing `proxy-group` also flatten one or more internal proxy groups as whole-group members
- require each output-facing `proxy-group` to bind its rules through one internal proxy group
- expose a proxy provider subscription that represents the proxy nodes selected by one internal proxy group
- expose a rule provider subscription that represents the rules selected by one internal proxy group
- keep internal proxy groups as the shared source-of-selection model across all three subscription types

Phase 4 is successful when contributors can point to a coherent subscription product model rather than three unrelated output ideas.

## Product Principles

Phase 4 work should follow these principles:

- internal proxy groups remain the shared selection unit for subscription outputs
- output-facing `proxy-group` entries are assembled by users and are distinct from internal proxy groups
- a Clash config subscription should be composable without requiring users to edit raw YAML
- output-facing `proxy-group` entries should be able to reference other output-facing `proxy-group` entries in the same subscription directly
- flattening should expand an internal proxy group as a whole without changing the meaning of that internal proxy group
- rule selection should be explicit and traceable through internal proxy groups
- proxy provider and rule provider subscriptions should stay narrow and reusable rather than becoming alternate full-config products
- future contributors should be able to explain every served subscription in terms of managed internal groups, managed proxy nodes, and managed rules

## In Scope

Phase 4 includes four product capabilities.

### 1. Clash Config Subscription Management

SubHub must let users create and manage a full Clash or Mihomo configuration subscription from the web page.

Expected behavior:

- a user can create a new Clash config subscription
- a user can view the current structure of that subscription from the web page
- a user can update the subscription definition later
- a user can remove a subscription definition that is no longer needed
- the subscription is understood as a direct-client configuration product rather than only as an internal preview

### 2. Output-Facing `proxy-group` Assembly

SubHub must let users assemble output-facing `proxy-group` entries for a Clash config subscription.

Expected behavior:

- a Clash config subscription can contain one or more output-facing `proxy-group` entries
- each output-facing `proxy-group` is assembled on the web page
- an output-facing `proxy-group` may reference one or more other output-facing `proxy-group` entries in the same subscription
- an output-facing `proxy-group` may also flatten one or more internal proxy groups
- flattening an internal proxy group means using the proxy nodes selected by that internal proxy group as direct members of the output-facing `proxy-group`
- contributors can clearly distinguish between referencing an output-facing `proxy-group` and flattening an internal proxy group

### 3. Rule Binding by Internal Proxy Group

SubHub must require each output-facing `proxy-group` in a Clash config subscription to bind rules through one internal proxy group.

Expected behavior:

- each output-facing `proxy-group` has one explicit rule binding
- that rule binding points to one internal proxy group
- the rules for that output-facing `proxy-group` come from the rules selected by the bound internal proxy group
- the rule binding is independent from whether the output-facing `proxy-group` uses referenced output-facing `proxy-group` entries, flattened internal proxy groups, or both for its proxy membership
- contributors can trace the rules used by any output-facing `proxy-group` back to one internal proxy group

### 4. Internal-Group-Based Provider Subscriptions

SubHub must expose proxy provider and rule provider subscriptions as direct projections of internal proxy groups.

Expected behavior:

- a proxy provider subscription is defined by one internal proxy group
- that proxy provider subscription publishes the proxy nodes selected by that internal proxy group
- a rule provider subscription is defined by one internal proxy group
- that rule provider subscription publishes the rules selected by that internal proxy group
- neither provider subscription type requires a separate manual selection model beyond choosing the internal proxy group
- contributors can explain both provider subscription types as reusable exports of existing internal group state

## Core Workflows

### Workflow 1: Create a Clash Config Subscription

1. A user starts the create-subscription flow from the web page.
2. The user chooses to create a Clash config subscription.
3. The user begins assembling one or more output-facing `proxy-group` entries for that subscription.
4. The system saves the subscription successfully when the required pieces are complete.
5. The new subscription becomes available as a direct-client Clash or Mihomo configuration product.

Notes:

- this workflow defines the full-config subscription as a first-class product
- creating the subscription is separate from creating internal proxy groups or manual rules in earlier phases

### Workflow 2: Build an Output-Facing `proxy-group` From Referenced `proxy-group` Entries

1. A user adds or edits an output-facing `proxy-group` inside a Clash config subscription.
2. The user chooses one or more existing output-facing `proxy-group` entries to reference.
3. The system records those `proxy-group` references as part of the current output-facing `proxy-group` definition.
4. The saved subscription reflects those referenced `proxy-group` relationships.

Notes:

- this workflow defines how output-facing `proxy-group` entries can be composed from other output-facing `proxy-group` entries
- referencing a `proxy-group` is different from flattening an internal proxy group

### Workflow 3: Flatten an Internal Proxy Group Into an Output-Facing `proxy-group`

1. A user adds or edits an output-facing `proxy-group` inside a Clash config subscription.
2. The user chooses the flatten-internal-group path.
3. The user selects one internal proxy group as the source for flattening.
4. The system uses the proxy nodes currently selected by that internal proxy group as direct members of the output-facing `proxy-group`.
5. The saved output-facing `proxy-group` includes that flattened internal proxy group as part of its membership definition.

Notes:

- flattening is a whole-group expansion workflow, not a per-node picking workflow
- flattening does not redefine the internal proxy group; it reuses that internal proxy group’s selected nodes as direct members in the output-facing `proxy-group`

### Workflow 4: Bind Rules to an Output-Facing `proxy-group`

1. A user creates or edits an output-facing `proxy-group` inside a Clash config subscription.
2. The user chooses one internal proxy group as that output-facing `proxy-group`’s rule binding.
3. The system associates the output-facing `proxy-group` with the rules selected by the bound internal proxy group.
4. The saved subscription reflects that rule binding as part of the output-facing proxy group definition.

Notes:

- rule binding is required for each output-facing `proxy-group`
- rule binding is based on internal proxy groups, not on free-form rule lists created directly inside the subscription flow

### Workflow 5: Publish a Proxy Provider Subscription

1. A user starts the create-subscription flow from the web page.
2. The user chooses to create a proxy provider subscription.
3. The user selects one internal proxy group.
4. The system creates a proxy provider subscription for that internal proxy group.
5. The resulting subscription publishes the proxy nodes selected by that internal proxy group.

Notes:

- this workflow is intentionally narrower than full Clash config assembly
- the internal proxy group is the complete source definition for the published proxy list

### Workflow 6: Publish a Rule Provider Subscription

1. A user starts the create-subscription flow from the web page.
2. The user chooses to create a rule provider subscription.
3. The user selects one internal proxy group.
4. The system creates a rule provider subscription for that internal proxy group.
5. The resulting subscription publishes the rules selected by that internal proxy group.

Notes:

- this workflow exports rules through an internal proxy group selection, not through a separate ad hoc rule picker
- the internal proxy group is the complete source definition for the published rule set

## Functional Requirements

Phase 4 should satisfy the following requirements:

1. A user can create a Clash config subscription from the web page.
2. A user can view and update an existing Clash config subscription definition.
3. A user can delete a Clash config subscription that is no longer needed.
4. A Clash config subscription can contain one or more output-facing `proxy-group` entries.
5. Output-facing `proxy-group` entries are distinct from internal proxy groups.
6. An output-facing `proxy-group` can reference one or more other output-facing `proxy-group` entries.
7. An output-facing `proxy-group` can flatten one or more internal proxy groups.
8. Flattening an internal proxy group means using the proxy nodes selected by that internal proxy group as direct members of the output-facing `proxy-group`.
9. The product must not require per-node picking when flattening an internal proxy group.
10. Each output-facing `proxy-group` must bind rules through one internal proxy group.
11. The rules associated with an output-facing `proxy-group` are the rules selected by its bound internal proxy group.
12. Rule binding remains required even when the output-facing `proxy-group` uses referenced `proxy-group` entries, flattened internal proxy groups, or both for membership.
13. A user can create a proxy provider subscription by selecting one internal proxy group.
14. A proxy provider subscription publishes the proxy nodes selected by that internal proxy group.
15. A user can create a rule provider subscription by selecting one internal proxy group.
16. A rule provider subscription publishes the rules selected by that internal proxy group.
17. Proxy provider subscriptions and rule provider subscriptions do not introduce separate manual node or rule selection models beyond the chosen internal proxy group.
18. Future contributors must be able to explain every Phase 4 subscription output in terms of internal proxy groups, managed proxy nodes, and managed rules.

## Out of Scope

To keep Phase 4 focused, the following are explicitly out of scope:

- authentication or access-control policy for subscription URLs
- health scoring, latency ranking, or node-quality-based filtering
- automatic rule generation or third-party rule-feed ingestion
- advanced validation or linting beyond what later roadmap phases define
- cross-subscription analytics, usage dashboards, or subscription-sharing features
- collaborative editing, approval flows, or audit history for subscription changes
- redefining how internal proxy groups choose nodes
- redefining how manual rules are authored or stored

Contributors should avoid turning Phase 4 into a performance, security, or automation phase. The goal is to publish coherent subscription products from the models that earlier phases already establish.

## Quality Expectations

Minimum quality expectations for Phase 4:

- contributors should be able to tell the difference between an internal proxy group and an output-facing `proxy-group` without ambiguity
- the Clash config subscription flow should make source selection explicit rather than hiding it in implicit defaults
- `proxy-group` references should be understandable as composition between output-facing `proxy-group` entries
- flattening should be understandable as expanding an internal proxy group as a whole
- rule binding should be explicit and traceable for every output-facing `proxy-group`
- proxy provider and rule provider subscriptions should behave like reusable exports of internal group state rather than like partially hidden config builders

## Acceptance Criteria

Phase 4 can be considered complete when all of the following are true:

- a contributor can demonstrate creating a Clash config subscription from the web page
- a contributor can demonstrate adding at least one output-facing `proxy-group` to that subscription
- a contributor can demonstrate configuring an output-facing `proxy-group` to reference another output-facing `proxy-group`
- a contributor can demonstrate configuring an output-facing `proxy-group` to flatten one chosen internal proxy group as a whole
- a contributor can demonstrate that each output-facing `proxy-group` requires a rule binding to one internal proxy group
- a contributor can demonstrate that the rules associated with an output-facing `proxy-group` are determined by its bound internal proxy group
- a contributor can demonstrate creating a proxy provider subscription by selecting one internal proxy group
- a contributor can demonstrate that the proxy provider subscription publishes the proxy nodes selected by that internal proxy group
- a contributor can demonstrate creating a rule provider subscription by selecting one internal proxy group
- a contributor can demonstrate that the rule provider subscription publishes the rules selected by that internal proxy group
- a contributor can demonstrate that proxy provider and rule provider subscriptions do not require separate manual node or rule picking beyond choosing the internal proxy group
- future contributors can read the spec and clearly understand that internal proxy groups are the shared source model across full-config, proxy-provider, and rule-provider subscription outputs

## Definition of Done

Phase 4 is done when SubHub supports a coherent subscription-services model built on internal proxy groups:

- users can create a Clash config subscription for direct client use
- users can assemble output-facing `proxy-group` entries by referencing other output-facing `proxy-group` entries and by flattening internal proxy groups as whole-group sources
- each output-facing `proxy-group` binds its rules through one internal proxy group
- users can publish proxy provider and rule provider subscriptions by selecting one internal proxy group

If those conditions hold reliably and within scope, future contributors can move into later validation, scoring, and optimization phases without redesigning the subscription model introduced here.

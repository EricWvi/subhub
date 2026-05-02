# Phase 3.5: Rule Aggregator Functional Spec

## Purpose

This document defines what Phase 3.5 of SubHub must deliver for future contributors. It is intentionally limited to product behavior, user expectations, and completion criteria. It does not prescribe implementation details, storage design, transport choices, or UI framework decisions.

## Background

Earlier phases establish the provider model, node model, and proxy group model. By the end of Phase 3, users can organize proxies into named proxy groups, but they still cannot define how traffic should be matched against those groups through product-managed rules.

Phase 3.5 exists to introduce manual rule management as a first-class capability. Users need to create and maintain explicit routing rules that connect a match condition to a chosen target group. This phase is limited to manual rule lists only. It does not include external rule-provider sources, imported rule sets, or automation around third-party rule feeds.

For this phase, a rule is defined by exactly three product fields:

- `rule_type`
- `pattern`
- `proxy_group`

Those three fields together represent one complete routing rule from the user’s perspective.

This phase also defines the selectable values that appear on the web page while editing a rule:

- `proxy_group` is a finite set made up of `DIRECT`, `REJECT`, and the proxy groups that already exist in SubHub
- `rule_type` includes `DOMAIN-SUFFIX`, `DOMAIN-KEYWORD`, and a custom type input for user-supplied rule types

Together, these capabilities move SubHub from group definition into user-managed rule assignment while keeping the scope narrow enough for future contributors to extend safely.

This phase also depends on a clear lifecycle rule between proxy groups and manual rules: if a proxy group is deleted, any manual rules that target that proxy group are deleted as well. Contributors should treat those rules as dependent on the continued existence of their target proxy group.

## Phase Outcome

At the end of Phase 3.5, SubHub should be able to:

- let users create, view, update, and delete manual rules
- treat each rule as a distinct record composed of `rule_type`, `pattern`, and `proxy_group`
- let users choose `proxy_group` from a finite set on the web page
- always include `DIRECT` and `REJECT` in that `proxy_group` set
- include all existing proxy groups in that same `proxy_group` set
- present `DOMAIN-SUFFIX` and `DOMAIN-KEYWORD` as built-in `rule_type` choices on the web page
- allow users to supply a custom `rule_type` when the built-in choices are not sufficient
- ensure manual rules are removed when their target proxy group is deleted
- display rule lists on the web page in descending order
- support paginated rule-list retrieval in the backend

Phase 3.5 is successful when contributors can point to a complete manual rule management experience rather than only a future roadmap idea.

## Product Principles

Phase 3.5 work should follow these principles:

- a rule is always understood as `rule_type + pattern + proxy_group`
- manual rule editing is a first-class product workflow, not a temporary fallback
- the web page should make finite choices explicit where the product already knows them
- built-in rule-type shortcuts should reduce friction without blocking advanced users
- future contributors should be able to understand the rule model without inferring hidden fields or derived behavior
- Phase 3.5 should stay focused on manual rule lists and should not expand into imported rule-provider systems
- rules that target a user-defined proxy group should not outlive that group
- rule-list browsing should remain usable as the number of rules grows

## In Scope

Phase 3.5 includes seven product capabilities.

### 1. Manual Rule CRUD

SubHub must let users create, view, update, and delete manual rules.

Expected behavior:

- a user can create a new rule
- a user can view the list of existing rules
- a user can inspect an individual rule and understand its current values
- a user can update an existing rule
- a user can delete a rule that is no longer needed
- a deleted rule no longer appears as an active rule in the product

### 2. Rule Structure

SubHub must treat each rule as exactly three fields: `rule_type`, `pattern`, and `proxy_group`.

Expected behavior:

- every rule contains one `rule_type`
- every rule contains one `pattern`
- every rule contains one `proxy_group`
- contributors can describe a rule completely using only those three fields
- the product does not require additional user-facing fields to define a manual rule in this phase

### 3. Finite Proxy Group Choices on the Web Page

SubHub must present `proxy_group` as a finite selectable set on the web page.

Expected behavior:

- the set always contains `DIRECT`
- the set always contains `REJECT`
- the set includes all proxy groups currently defined in SubHub
- users choose from the available set rather than entering arbitrary free-form `proxy_group` values on the web page
- contributors can reason about rule targets as a bounded set at the time of editing

### 4. Rule Type Choices Plus Custom Input

SubHub must present built-in `rule_type` choices on the web page while also supporting custom rule types.

Expected behavior:

- the web page includes `DOMAIN-SUFFIX` as a built-in choice
- the web page includes `DOMAIN-KEYWORD` as a built-in choice
- the web page also includes a custom type input path for user-supplied `rule_type` values
- users are not forced to misuse `DOMAIN-SUFFIX` or `DOMAIN-KEYWORD` when another valid rule type is needed
- contributors can treat built-in choices as convenience options rather than as an exhaustive list

### 5. Proxy Group Deletion Cascade

SubHub must remove dependent manual rules when a user-defined proxy group is deleted.

Expected behavior:

- if a proxy group is deleted, every manual rule whose `proxy_group` points to that deleted group is also deleted
- rules targeting `DIRECT` or `REJECT` are unaffected by deletion of user-defined proxy groups
- contributors can reason about rule validity without handling orphaned rules that point to missing proxy groups

### 6. Descending Rule List Order on the Web Page

SubHub must display rule lists on the web page in descending order.

Expected behavior:

- when a user views the rule list, rules are shown in descending order
- the ordering is consistent across normal rule-list browsing on the web page
- contributors can reason about which rules appear first without relying on unspecified list behavior

### 7. Backend Pagination for Rule Lists

SubHub must support pagination for rule-list retrieval in the backend.

Expected behavior:

- backend rule-list retrieval supports paginated access
- contributors can request a subset of the total rule list rather than requiring the full list every time
- pagination support exists as part of the product contract for rule-list browsing
- pagination does not change the descending order expectation for the returned rule list

## Core Workflows

### Workflow 1: Create a Rule With a Built-In Rule Type

1. A user starts the create-rule flow.
2. The user selects a built-in `rule_type` such as `DOMAIN-SUFFIX` or `DOMAIN-KEYWORD`.
3. The user enters a `pattern`.
4. The user selects a `proxy_group` from the web page.
5. The system creates the rule successfully.
6. The new rule appears as a complete rule made of `rule_type`, `pattern`, and `proxy_group`.

Notes:

- this workflow should feel like the default path for common rule creation
- the rule is incomplete until all three fields are provided

### Workflow 2: Create a Rule With a Custom Rule Type

1. A user starts the create-rule flow.
2. The user chooses the custom type input path instead of a built-in rule type.
3. The user enters a custom `rule_type`.
4. The user enters a `pattern`.
5. The user selects a `proxy_group` from the available set on the web page.
6. The system creates the rule successfully.
7. The new rule appears with the user-supplied custom `rule_type`.

Notes:

- the custom path exists to support valid rule types beyond the built-in shortcuts
- custom type entry does not change the three-field rule structure

### Workflow 3: Select a Proxy Group Target

1. A user creates or edits a rule.
2. The web page presents the available `proxy_group` choices.
3. The user sees `DIRECT` and `REJECT` in the list.
4. The user also sees the proxy groups that currently exist in SubHub.
5. The user selects one value from that finite set.
6. The saved rule reflects the selected `proxy_group`.

Notes:

- this workflow defines the product contract for selectable rule targets
- `proxy_group` selection is bounded even though the set can expand as users create more proxy groups

### Workflow 4: Edit a Rule

1. A user opens an existing rule.
2. The user changes one or more of `rule_type`, `pattern`, or `proxy_group`.
3. The system saves the updated rule.
4. The rule continues to exist as the same logical rule with updated values.

Notes:

- editing may switch between a built-in `rule_type` and a custom `rule_type`
- editing one rule must not implicitly change other rules

### Workflow 5: Delete a Rule

1. A user selects an existing rule for deletion.
2. The system completes the delete action.
3. The deleted rule is no longer shown as an active rule.

Notes:

- deletion removes that rule from the managed manual rule list
- deleting a rule does not remove any proxy group

### Workflow 6: Delete a Proxy Group That Is Referenced by Rules

1. A user deletes an existing proxy group.
2. The system identifies manual rules whose `proxy_group` points to that proxy group.
3. The system deletes the proxy group.
4. The system also deletes the related manual rules.
5. The deleted proxy group no longer appears as an available `proxy_group` choice on the web page.
6. The related deleted rules no longer appear as active rules in the product.

Notes:

- this is a product-level cascade, not an optional cleanup step
- the goal is to prevent orphaned rules that target a missing proxy group

### Workflow 7: Browse the Rule List on the Web Page

1. A user opens the rule list on the web page.
2. The system shows the available rules in descending order.
3. The user reviews the rules starting from the first visible entries.
4. If more rules exist than fit in one page of results, the user continues through the list using paginated browsing.

Notes:

- descending order is the default list presentation on the web page
- pagination should preserve predictable ordering as the user moves through the rule list

## Functional Requirements

Phase 3.5 should satisfy the following requirements:

1. A user can create a manual rule.
2. A user can view existing manual rules.
3. A user can update an existing manual rule.
4. A user can delete an existing manual rule.
5. Every rule is defined by exactly three fields: `rule_type`, `pattern`, and `proxy_group`.
6. A rule is not considered complete unless all three fields are present.
7. The web page presents `proxy_group` as a finite set of choices.
8. That `proxy_group` set always includes `DIRECT`.
9. That `proxy_group` set always includes `REJECT`.
10. That `proxy_group` set includes the proxy groups that already exist in SubHub.
11. The web page presents `DOMAIN-SUFFIX` as a built-in `rule_type` choice.
12. The web page presents `DOMAIN-KEYWORD` as a built-in `rule_type` choice.
13. The web page supports a custom type input for user-supplied `rule_type` values.
14. A rule created with a custom `rule_type` is still treated as a normal rule composed of the same three fields.
15. Changing one rule does not implicitly change other rules.
16. If a user-defined proxy group is deleted, all manual rules targeting that proxy group are deleted.
17. No active manual rule may remain with a `proxy_group` that refers to a deleted user-defined proxy group.
18. The web page displays rule lists in descending order.
19. Backend rule-list retrieval supports pagination.
20. Paginated rule-list retrieval preserves descending order.

## Out of Scope

To keep Phase 3.5 focused, the following are explicitly out of scope:

- external rule-provider sources
- importing remote rule sets
- syncing or refreshing third-party rule feeds
- implementation details of how rules are stored or rendered
- rule evaluation engine internals
- rule ordering semantics beyond basic manual list management
- validation rules for every possible custom rule type
- permissions, collaboration, or audit-history features for rule changes

Contributors should avoid expanding Phase 3.5 into a general rule-provider integration phase. The goal is to define and manage manual rule lists clearly.

## Quality Expectations

Minimum quality expectations for Phase 3.5:

- the three-field rule model should be consistent across create, read, update, and delete flows
- built-in and custom `rule_type` paths should feel like two ways to produce the same kind of rule
- `proxy_group` choices on the web page should be understandable and bounded at the moment of editing
- contributors should be able to understand the manual rule model without reading implementation code
- the product should communicate that manual rules are intentional, supported data rather than an advanced edge case
- deleting a proxy group should leave the remaining rule list in a valid, non-orphaned state
- rule-list browsing should remain predictable and manageable even when many rules exist

## Acceptance Criteria

Phase 3.5 can be considered complete when all of the following are true:

- a contributor can demonstrate creating a manual rule successfully
- a contributor can demonstrate viewing existing manual rules
- a contributor can demonstrate updating an existing manual rule successfully
- a contributor can demonstrate deleting an existing manual rule successfully
- a contributor can demonstrate that a rule is represented by exactly `rule_type`, `pattern`, and `proxy_group`
- a contributor can demonstrate that `DIRECT` appears as a selectable `proxy_group` on the web page
- a contributor can demonstrate that `REJECT` appears as a selectable `proxy_group` on the web page
- a contributor can demonstrate that existing proxy groups from SubHub appear as selectable `proxy_group` values on the web page
- a contributor can demonstrate that `DOMAIN-SUFFIX` appears as a built-in `rule_type` choice on the web page
- a contributor can demonstrate that `DOMAIN-KEYWORD` appears as a built-in `rule_type` choice on the web page
- a contributor can demonstrate creating or editing a rule by supplying a custom `rule_type`
- a contributor can demonstrate that deleting a proxy group also deletes manual rules targeting that proxy group
- a contributor can demonstrate that no active rule remains pointing at a deleted user-defined proxy group
- a contributor can demonstrate that the web page displays the rule list in descending order
- a contributor can demonstrate that backend rule-list retrieval supports pagination
- a contributor can demonstrate that paginated rule-list results remain in descending order
- future contributors can read this spec and clearly understand that Phase 3.5 supports manual rule lists only, not external rule-provider integration

## Definition of Done

Phase 3.5 is done when SubHub supports a complete manual rule management model that future contributors can build on:

- manual rules can be created, viewed, updated, and deleted
- every manual rule is defined as `rule_type + pattern + proxy_group`
- `proxy_group` selection on the web page is limited to `DIRECT`, `REJECT`, and the proxy groups already defined in SubHub
- `rule_type` entry on the web page supports both built-in choices and a custom type input
- deleting a proxy group also removes manual rules that target it
- the web page presents manual rules in descending order
- the backend supports paginated rule-list retrieval without breaking that order

If those conditions hold reliably and within scope, future contributors can build later output and routing work without redefining what a manual rule is in SubHub.

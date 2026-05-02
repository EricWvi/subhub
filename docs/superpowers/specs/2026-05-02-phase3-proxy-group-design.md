# Phase 3: Proxy Group Functional Spec

## Purpose

This document defines what Phase 3 of SubHub must deliver for future contributors. It is intentionally limited to product behavior, user expectations, and completion criteria. It does not prescribe implementation details, storage design, execution engines, or specific technologies.

## Background

Earlier phases establish the provider and node foundations of SubHub. Provider data can be fetched, normalized, and exposed as managed proxy nodes that later product features can build on.

Phase 3 exists because users need a way to organize those managed proxy nodes into meaningful groups. A flat list of nodes is not enough once users want to separate traffic or use cases such as streaming, AI services, regional routing, or custom operational categories.

This phase introduces proxy groups as a user-managed concept. Each group represents a named bucket of proxy nodes defined by the user. A group may also include a user-provided script that controls how nodes are selected or assigned for that group, but scripting is optional. A group must still be valid and usable even when no script is provided.

Together, these capabilities move SubHub from node storage into user-directed grouping behavior, while keeping the model understandable for future contributors.

## Phase Outcome

At the end of Phase 3, SubHub should be able to:

- let users create, view, update, and delete proxy groups
- treat each proxy group as a distinct user-managed object with its own name and configuration
- allow a user script to be attached to one proxy group without affecting other groups
- allow proxy groups to exist with no user script at all
- preserve a clear relationship between a proxy group and the nodes it includes
- provide a contributor-facing product model that can support later automation and output work without redefining what a proxy group is

Phase 3 is successful when contributors can point to a complete, understandable proxy group management experience rather than only a concept in the roadmap.

## Product Principles

Phase 3 work should follow these principles:

- proxy groups are user-managed and explicitly named
- each group stands on its own rather than inheriting hidden behavior from other groups
- a user script belongs to exactly one group at a time
- a missing script is a normal and supported state
- group behavior should be understandable from the group itself, without requiring contributors to infer hidden coupling
- Phase 3 should enable later automation without forcing users to adopt scripting before they need it

## In Scope

Phase 3 includes three product capabilities.

### 1. Proxy Group CRUD

SubHub must let users create, view, update, and delete proxy groups.

Expected behavior:

- a user can create a new proxy group
- a user can view the list of existing proxy groups
- a user can inspect an individual group and understand its current state
- a user can update an existing group's editable fields
- a user can delete a proxy group that is no longer needed
- a deleted group no longer appears as an active group in the product
- proxy groups remain distinguishable as separate user-managed records

### 2. User Script Is Per Group

SubHub must treat user scripts as group-specific configuration rather than as shared global behavior.

Expected behavior:

- a script can be attached to one proxy group
- different groups can have different scripts
- changing the script for one group affects only that group
- removing or replacing a script on one group does not change the script state of any other group
- contributors can reason about a group’s script behavior by looking at that group alone

### 3. User Script Is Optional

SubHub must support proxy groups that do not have a user script.

Expected behavior:

- a proxy group can be created without a script
- a proxy group without a script is still considered valid
- a user may add a script later to a group that was initially created without one
- a user may remove an existing script from a group and keep the group
- the product does not force a placeholder script, default script, or dummy script value just to make a group valid

## Core Workflows

### Workflow 1: Create a Proxy Group Without a Script

1. A user starts the create-group flow.
2. The user provides the required group information.
3. The user leaves the script field empty.
4. The system creates the proxy group successfully.
5. The new group appears as a valid proxy group with no script attached.

Notes:

- this is a first-class supported workflow, not an exception path
- the absence of a script should be represented clearly rather than as an error state

### Workflow 2: Create a Proxy Group With a Script

1. A user starts the create-group flow.
2. The user provides the required group information.
3. The user provides a script for that group.
4. The system creates the proxy group successfully.
5. The new group appears with its script attached to that group only.

Notes:

- the script belongs to the newly created group
- creating one scripted group does not create or alter scripts for any other group

### Workflow 3: Update a Group’s Script

1. A user opens an existing proxy group.
2. The user edits the script attached to that group.
3. The system saves the updated group state.
4. The group reflects the new script.
5. Other groups remain unchanged.

Notes:

- this workflow applies whether the group previously had a script or not
- the key product rule is isolation between groups

### Workflow 4: Remove a Group’s Script

1. A user opens an existing proxy group that currently has a script.
2. The user removes the script from that group.
3. The system saves the updated group state.
4. The group remains valid and continues to exist without a script.

Notes:

- removing a script does not imply deleting the group
- a scriptless group remains a supported end state

### Workflow 5: Edit General Group Information

1. A user opens an existing proxy group.
2. The user updates editable group information.
3. The system saves the changes.
4. The group continues to exist as the same logical group with updated details.

Notes:

- updating group information should not implicitly change the script state unless the user changes the script directly

### Workflow 6: Delete a Proxy Group

1. A user selects an existing proxy group for deletion.
2. The system completes the delete action.
3. The deleted group is no longer shown as an active proxy group.

Notes:

- deletion removes the group as a user-managed object
- once deleted, the group should no longer participate in normal group workflows

## Functional Requirements

Phase 3 should satisfy the following requirements:

1. A user can create a proxy group.
2. A user can view existing proxy groups.
3. A user can update an existing proxy group.
4. A user can delete an existing proxy group.
5. Each proxy group is represented as its own distinct user-managed object.
6. A proxy group may have a user script associated with it.
7. A user script, when present, belongs to one specific proxy group.
8. Updating a script for one group does not update any other group.
9. Different proxy groups may have different scripts.
10. A proxy group may exist with no script.
11. A proxy group without a script is still valid and manageable.
12. A user can add a script to a previously scriptless group.
13. A user can remove a script from a group without deleting the group.
14. The product must not require a placeholder or default script to validate a group.

## Out of Scope

To keep Phase 3 focused, the following are explicitly out of scope:

- implementation details of how scripts are executed
- programming language or runtime choices for scripts
- security model details for script execution
- advanced script authoring tools, templates, or debugging environments
- rule-provider aggregation or rule editing
- node health scoring, ranking, or selection quality logic
- multi-group inheritance, shared script libraries, or global script defaults
- collaborative editing, audit history, or permissions models for group changes

Contributors should avoid expanding Phase 3 into a general automation platform. The goal is to define and manage proxy groups cleanly, not to solve all future scripting needs at once.

## Quality Expectations

Minimum quality expectations for Phase 3:

- proxy group lifecycle behavior should be consistent across create, read, update, and delete flows
- the presence or absence of a script should be easy to understand for each group
- group-specific script behavior should remain isolated from other groups
- contributors should be able to reason about proxy groups without inferring hidden cross-group dependencies
- scriptless groups should feel intentional and supported, not like incomplete data

## Acceptance Criteria

Phase 3 can be considered complete when all of the following are true:

- a contributor can demonstrate creating a new proxy group successfully
- a contributor can demonstrate viewing existing proxy groups
- a contributor can demonstrate updating an existing proxy group successfully
- a contributor can demonstrate deleting an existing proxy group successfully
- a contributor can demonstrate creating a proxy group without a script and show that it remains valid
- a contributor can demonstrate creating a proxy group with a script attached to that group
- a contributor can demonstrate updating the script for one group without changing any other group
- a contributor can demonstrate removing a script from a group while keeping the group valid and present
- a contributor can demonstrate that different groups can hold different script content
- a contributor can demonstrate that the product does not require a script in order to create or retain a proxy group
- future contributors can read the product behavior and clearly understand that scripts are per-group and optional

## Definition of Done

Phase 3 is done when SubHub supports a complete proxy group management model that future contributors can build on:

- proxy groups can be created, viewed, updated, and deleted
- each group can carry its own script independently of other groups
- groups remain valid whether a script is present or absent

If those conditions hold reliably and within scope, future contributors can build later rule, automation, and output features without redefining the core behavior of proxy groups.

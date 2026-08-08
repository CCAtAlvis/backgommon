---
description: "Rule for generating a standardized Product Requirements Document (PRD) from initial user input."
globs: []
alwaysApply: false
---

# PRD Creation Rule

## Purpose

This rule defines the process for generating a standardized Product Requirements Document (PRD) based on initial user requirements, including necessary clarification steps.

## Workflow (AI Instructions)

The AI assistant MUST follow these steps to generate a PRD:

1.  **Step 1: Gather Initial Requirements**
    *   Receive the user's initial thoughts, goals, or requirements for the feature/task.

2.  **Step 2: Clarify & Elaborate (If Necessary)**
    *   Analyze the initial input against the Standard PRD Template sections (see below).
    *   If the input lacks sufficient detail to populate key sections (e.g., Background, Scope, Specific Requirements), prompt the user with targeted questions to gather the missing information. Example questions:
        *   "What is the primary motivation or problem this feature solves?" (for Background)
        *   "What are the specific functional requirements? What should the user be able to do?" (for Feature Overview/Requirements)
        *   "Are there any specific technical constraints or non-functional requirements (performance, security)?"
        *   "What parts of the system will this likely interact with?" (for Scope/Implementation)
    *   Continue clarification until enough detail is gathered to draft a comprehensive PRD.

3.  **Step 3: Generate Standardized PRD**
    *   Structure the gathered information according to the **Standard PRD Template** provided below.
    *   Use the structure and formatting style exemplified by existing PRDs like `docs/prd/api-logging-prd.md` and `docs/prd/fee-collection.md`.
    *   Ensure all relevant sections from the template are included.
    *   Save the generated PRD file in the designated feature directory (e.g., `docs/prd/{feature-name}/{feature-name}-prd.md`).
    *   Output the path to the created PRD file.

## Standard PRD Template

```markdown
# {Feature Name} Requirements (PRD)

**created by:** {User Name/Handle}
**created on:** YYYY-MM-DD
**last updated:** YYYY-MM-DD

## Background & Motivation

(Explain the "why" behind this feature. What problem does it solve? What is the context?)

## Scope

-   **Functionality:** (Describe the core features and functionality included.)
-   **Chains/Platforms:** (Specify applicable networks, platforms, etc.)
-   **Providers/Integrations:** (Mention relevant third-party services or internal modules involved.)
-   **Out of Scope:** (Clearly define what is *not* included in this phase/feature.)

## Feature Overview / Requirements

(Provide a detailed breakdown of how the feature works. Use numbered steps for workflows, bullet points for specific requirements.)
1.  **Step 1:** ...
2.  **Step 2:** ...
    -   Requirement 2a
    -   Requirement 2b

## Data Structures / Models

(Define or reference any new or significantly modified data structures or database models. Include code blocks if helpful.)

```typescript
export type NewModel = {
  // ... fields ...
};
```

## Example Flow / Use Case

(Provide a concrete example of the feature in action to illustrate its usage and expected outcome.)

## UI/UX Considerations (Optional)

(Describe any specific user interface or user experience requirements or mockups if applicable.)

## API Changes Required (If Applicable)

(List any new API endpoints to be created or existing ones to be modified.)
-   `POST /api/new-endpoint`: Creates {resource}. Requires {params}.
-   `GET /api/existing-endpoint/:id`: Add {new_field} to response.

## Implementation Details (Optional Initial Sketch)

(High-level thoughts on implementation approach, key components, potential challenges. This section might be expanded later or covered entirely by the separate Execution Plan.)

### Relevant Files (Initial List)
*(This list can be refined during planning/implementation)*
- `path/to/file1.ts`
- `path/to/file2.ts`

### Implementation Gist
*(Brief summary of the technical approach)*

```

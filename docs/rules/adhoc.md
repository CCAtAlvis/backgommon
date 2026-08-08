---
description: "Standard process for handling small, ad-hoc tasks and bug fixes using AI assistance."
globs: []
alwaysApply: false
---

# Ad-Hoc Task & Bug Fix Workflow

## Purpose

This document defines a streamlined workflow for addressing small, self-contained tasks or bug fixes that do not require a full PRD and execution plan. The focus is on quick identification, planning, execution, and logging.

## Trigger

This workflow is initiated when a user describes a specific, relatively small task or bug fix, rather than a large feature requiring extensive planning. Examples:
*   "Fix the off-by-one error in the pagination logic in `file.ts`."
*   "Refactor the `getUser` function in `service.ts` to also return the user's role."
*   "Add input validation for the email field on the settings page form."

## Workflow (AI Instructions)

The AI assistant MUST follow these steps for ad-hoc tasks/bugs:

1.  **Step 1: Understand Task/Bug**
    *   Receive the user's description of the task or bug.
    *   If it's a bug, ensure context like affected files, reproduction steps (if provided), and expected vs. actual behavior are noted.
    *   Ask clarifying questions if the request is ambiguous (e.g., "Which specific function needs refactoring?", "What validation rules should be applied?").

2.  **Step 2: Identify Relevant Code**
    *   Based on the description and clarifications, identify the relevant files and specific code sections (functions, classes) that need modification.
    *   Perform codebase searches (grep, semantic, file) if necessary.

3.  **Step 3: Propose Action Plan**
    *   Outline a concise action plan (typically 1-5 steps) to address the task/bug. Examples:
        *   For a bug: "1. Read function X in `file.ts`. 2. Apply fix Y. 3. Add/update unit test Z. 4. Verify."
        *   For a small task: "1. Modify function A in `service.ts`. 2. Update call sites in `controller.ts`. 3. Add relevant tests."
    *   Present the plan to the user for confirmation or modification.

4.  **Step 4: Execute Plan Step-by-Step**
    *   Once the plan is confirmed, proceed with executing the *first* step.
    *   After completing each step, inform the user and confirm readiness to proceed to the next.

5.  **Step 5: Log Task & Progress**
    *   After confirming the action plan (before execution), create or append an entry to a dedicated log file: `docs/engineering/adhoc-log.md`.
    *   Use the **Ad-Hoc Task Log Entry Template** (see below). Fill in the initial details (Reported, Relevant Files, Plan).
    *   After completing *each step* of the plan, update the corresponding task marker `[ ]` to `[x]` in the log file.
    *   Add any relevant notes encountered during execution (e.g., "Refactoring required slight adjustment to parameter order").
    *   Once all steps are done, update the entry's status to `(Status: Completed)`.

## Ad-Hoc Task Log Entry Template (`docs/engineering/adhoc-log.md`)

```markdown
---

## {YYYY-MM-DD} - {Brief Task/Bug Title} - (Status: In Progress)

-   **Reported:** {User's initial description or link to conversation/issue tracker}
-   **Relevant Files:**
    -   `path/to/affected/file1.ts`
    -   `path/to/affected/file2.ts`
-   **Plan:**
    -   [ ] Step 1: Action description
    -   [ ] Step 2: Action description
    -   [ ] Step 3: Action description
-   **Notes:** *(Add notes during execution)*

```

*(Append new entries to this file, separated by `---`)*

## Comparison to Feature Initiation Workflow

This ad-hoc workflow differs significantly:
*   **No formal PRD:** Relies on user description and AI clarification.
*   **No separate Execution Plan file:** The concise plan is logged directly.
*   **No Checkpoint file:** Progress is tracked within the central `adhoc-log.md`.
*   **Focus:** Rapid execution and logging for smaller changes. 

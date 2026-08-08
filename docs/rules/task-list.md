---
description: "Guidelines for creating and managing task list files to track complex feature implementation."
globs: []
alwaysApply: false
---

# Task List File Creation & Management

## Purpose

Task List files serve as a central reference point for large, complex features or tasks that may take multiple sessions to complete and require careful tracking. They help both humans and AI assistants maintain context, track progress, identify relevant files, and manage the implementation plan.

## Task List File Creation

1.  **File Naming:** Create the task list file in the relevant feature directory (e.g., `docs/prd/{feature-name}/`). Use the naming convention `{feature-name}-task-list.md`.
2.  **Initial Generation:** When starting a task, the AI assistant should propose creating a task list file based on the initial requirements (e.g., a PRD file) and user request. When initiating a task via the Feature Initiation Workflow (`./feature-workflow.md`), the AI assistant will automatically generate this file based on the finalized PRD and Execution Plan.
3.  **Core Sections:** The task list file MUST include the following sections:

    ```markdown
    # Task List: {Feature Name} Implementation

    **Date Created:** {Current Date}
    **Last Updated:** {Current Date}

    ## 1. Objective

    (Briefly state the primary goal of the feature/task being implemented. Copied from PRD.)

    ## 2. Requirements Summary

    (Summarize the key functional and non-functional requirements derived from the PRD or user instructions. Use bullet points.)

    ## 3. Relevant Files & PRDs

    (List all relevant files identified initially or during implementation. Include PRDs, models, services, controllers, utilities, configuration files, etc. Copied initially from PRD/Plan analysis.)
    - `path/to/prd.md` - Main requirements document
    - `path/to/plan.md` - Execution plan
    - `path/to/model.ts` - Data structures
    - `path/to/service.ts` - Core logic implementation
    - ...

    ## 4. Implementation Plan (Tasks)

    (Break down the implementation into logical phases and detailed steps. Use markdown task lists `[ ]` for each actionable step. This plan should be derived from the requirements and initial analysis or copied from Execution Plan.)

    **Phase 1: {Phase Name}**
    - [ ] Step 1.1: Description
    - [ ] Step 1.2: Description

    **Phase 2: {Phase Name}**
    - [ ] Step 2.1: Description
    - [ ] Step 2.2: Description
    ...

    ## 5. Current Status (as of {Current Date})

    (Describe the current state of the implementation. Indicate which phase/step is next.)
    - **Phase X:** Brief status update.
    - **Next Step:** Describe the immediate next task from the implementation plan.

    ## 6. Notes & Challenges

    (A section for recording important decisions, potential issues, technical complexities, alternative approaches considered, or any other relevant context discovered during implementation.)
    ```

## Task List Maintenance (AI Instructions)

When working on a task associated with a task list file, the AI assistant MUST:

1.  **Consult the Task List:** Before starting work on a sub-task, consult the task list file to understand the current status and identify the next step in the implementation plan.
2.  **Update After Progress:** After successfully completing a step (or a set of related steps) from the implementation plan:
    *   Update the "Last Updated" timestamp.
    *   Mark the completed task(s) in the "Implementation Plan (Tasks)" section by changing `[ ]` to `[x]`.
    *   Update the "Current Status" section to reflect the progress made and identify the new next step.
    *   Add any relevant findings, decisions, or challenges encountered to the "Notes & Challenges" section.
    *   Ensure the "Relevant Files" section is up-to-date if new files were created or significant changes were made to existing ones.
3.  **Propose Updates:** Explicitly state the intention to update the task list file after making progress.

## Example Update Flow

**Before Task:** AI checks task list file, identifies "[ ] Step 1.2: Implement API endpoint" as the next step.
**AI Action:** Implements the API endpoint (e.g., edits controller, service, adds route).
**After Task:** AI proposes updating the task list file:
*   Changes "[ ] Step 1.2: Implement API endpoint" to "[x] Step 1.2: Implement API endpoint" in section 4.
*   Updates "Current Status" to reflect completion of Step 1.2 and identifies Step 2.1 as next.
*   Adds a note like "Used standard REST principles for the endpoint" to section 6 if relevant.
*   Updates "Last Updated" timestamp. 

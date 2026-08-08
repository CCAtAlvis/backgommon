---
description: "Defines the end-to-end workflow for initiating a new feature, from requirements gathering to creating planning and tracking documents."
globs: []
alwaysApply: false
---

# Feature Initiation Workflow Rule

## Purpose

This rule orchestrates the standardized process for initiating a new feature or complex task. It leverages specific, modular rules for generating the PRD, Execution Plan, and Checkpoint File, ensuring a consistent and efficient start to implementation.

## Workflow (AI Instructions)

When a user requests to start working on a new feature based on initial ideas:

1.  **Step 1: Invoke PRD Creation**
    *   Follow the workflow defined in `./prd-creation.md` to gather detailed requirements and generate the standardized PRD.
    *   Ensure the PRD is saved in the correct feature-specific directory: `docs/prd/{feature-name}/{feature-name}-prd.md`.
    *   Keep track of the generated PRD file path.

2.  **Step 2: Invoke Execution Plan Generation**
    *   Once the PRD is created and its path is known, follow the workflow defined in `./execution-plan.md`.
    *   Provide the path to the generated PRD as input to this rule.
    *   Ensure the finalized execution plan is saved in the same feature-specific directory: `docs/prd/{feature-name}/{feature-name}-plan.md`.
    *   Keep track of the generated execution plan file path.

3.  **Step 3: Invoke Checkpoint File Creation**
    *   Using the generated PRD and the finalized Execution Plan, create the initial Checkpoint file.
    *   Follow the structure and guidelines defined in `./task-list.md`.
    *   Populate the required sections:
        *   Objective (from PRD)
        *   Requirements Summary (from PRD)
        *   Relevant Files (from PRD and execution plan analysis)
        *   Implementation Plan (copy the steps from the finalized execution plan, using `[ ]` task markers)
        *   Current Status (indicating the first step of the plan is next)
    *   Save the checkpoint file in the same feature-specific directory: `docs/prd/{feature-name}/{feature-name}-task-list.md`.
    *   Keep track of the generated checkpoint file path.

4.  **Step 4: Inform User**
    *   Notify the user that the PRD, Execution Plan, and Checkpoint file have been created.
    *   Provide the paths to all three files.
    *   State that the setup is complete and implementation can begin by following the steps outlined in the checkpoint file.

## Directory Structure Convention

All generated artifacts for a specific feature should reside within a dedicated subdirectory under `docs/prd/`:

```
docs/
└── prd/
    └── {feature-name}/
        ├── {feature-name}-prd.md        # The PRD
        ├── {feature-name}-plan.md       # The Execution Plan
        └── {feature-name}-task-list.md  # The Task List
``` 

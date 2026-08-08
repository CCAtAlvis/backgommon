---
description: "Rule for generating a detailed Execution Plan based on an existing PRD."
globs: []
alwaysApply: false
---

# Execution Plan Generation Rule

## Purpose

This rule defines the process for generating a detailed, phased execution plan for implementing a feature, assuming a complete Product Requirements Document (PRD) already exists.

## Prerequisites

-   A complete PRD document for the feature, following the standard template (e.g., `docs/prd/{feature-name}/{feature-name}-prd.md`).

## Workflow (AI Instructions)

The AI assistant MUST follow these steps to generate an Execution Plan:

1.  **Step 1: Analyze PRD & Scan Codebase**
    *   Receive the path to the relevant PRD file as input.
    *   Thoroughly analyze the requirements, scope, and technical details outlined in the PRD.
    *   Perform a codebase search (semantic, grep, file search) to identify existing files (models, services, controllers, utilities, components, etc.) relevant to the feature. Pay attention to files mentioned in the PRD's "Implementation Details" or "Relevant Files" sections, if present.

2.  **Step 2: Draft Execution Plan**
    *   Based on the PRD requirements and identified files, generate a detailed, phased execution plan.
    *   Structure the plan logically, typically including phases like:
        *   Setup / Configuration
        *   Data Modeling / Database Changes
        *   Backend Logic / Service Implementation
        *   API Endpoint Creation / Modification
        *   Frontend Implementation (if applicable)
        *   Testing (Unit, Integration, End-to-End)
        *   Deployment Preparation
    *   Within each phase, list specific, actionable steps with clear descriptions. Use sub-bullets for finer granularity if needed.
    *   The plan should outline *how* the requirements in the PRD will be technically implemented.

3.  **Step 3: User Review & Refinement**
    *   Present the generated execution plan draft to the user.
    *   Explicitly ask for review and feedback: "Please review the draft execution plan based on the PRD. Does it accurately reflect the necessary steps? Are there any missing steps, incorrect assumptions, or changes required in the phasing or details?"
    *   Modify the execution plan based on the user's feedback until it is approved. Iterate if necessary.

4.  **Step 4: Save Final Execution Plan**
    *   Save the finalized execution plan as a markdown file.
    *   The plan should be saved in the *same directory* as the corresponding PRD file, using the naming convention: `docs/prd/{feature-name}/{feature-name}-plan.md`.
    *   Output the path to the created execution plan file. 

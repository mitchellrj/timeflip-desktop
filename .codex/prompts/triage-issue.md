---
name: /triage-issue
id: triage-issue
category: Development
description: Pick and triage an issue from the issue tracker, preparing it for development against.
---

Triage an issue and determine next steps and inputs for development.

**Input**: The argument after `/triage-issue` is the ID of a known issue.

**Steps**

1. **If no ID is given, select an issue to work on.**

   Use the **Github tools** to pick the oldest issue from the current repository with the `needs-triage` label. Share the issue ID, link, and title, and ask the user if:
   * they'd like to work on this issue,
   * if they would like to specify a different issue ID, or
   * if they would like to skip to the next issue (ordered by age descending).

2. **Assess the type of ticket.**

   If the `bug` or `enhancement` labels are missing, parse the body of the issue and identify if it is a bug report or feature request.

3. **Assess based on ticket type.**

    1. **Bug tickets**

        **Use model**: GPT-5.4

        Parse the body of the issue, and any attachments. Check as a minimum for steps to reproduce, expected behaviour, and actual behaviour. If these are missing, the next action is **Needs more info**.

        If the affected area, impact, or effort estimates are missing, then attempt to infer these from the description and codebase. Only apply limited reasoning and effort. If more information is needed, proceed to **Needs more info**.

        Otherwise:
        1. Use the **Github tools** to update the issue.
        2. Use the **Github tools** to remove the `needs-triage` label from the issue and add the `ready-for-development` label.

    2. **Feature / enhancement requests**

        **Use model**: GPT-5.5

        Parse the body of the issue and any attachments. Check as a minimum for "problem or opportunity" and "desired outcome".

        If the affected area, impact, or effort estimates are missing, then attempt to infer these from the description and codebase. Use the **Github tools** to update the issue with the new detail.

        1. **Large and extra large issues**

            1. Prepare a requirements specification document based on information in the issue. Break the specification into sections: "Background", "Scope In", "Scope Out" and "Acceptance Criteria".
            2. Create a new git branch for the issue following the branch name spec: `feat/{ISSUE_ID}-{ISSUE_TITLE}`.
            3. Add the specification document to the `requirements`](../../requirements) folder, commit it, and push it.
            4. Use the **Github tools** to remove the `needs-triage` label from the issue and add the `needs-analysis` label.

        2. **Medium and small issues**

            1. Identify the best fit document in the [`spdd/prompts`](../../spdd/prompts) folder
            2. Run the [`/spdd-prompt-update`](./spdd-prompt-update.md) command against the prompt file with the new feature description.
            3. Create a new git branch for the issue following the branch name spec: `feat/{ISSUE_ID}-{ISSUE_TITLE}`.
            4. Commit and push the updated prompt file.
            5. Use the **Github tools** to remove the `needs-triage` label from the issue and add the `ready-for-development` label.

4. **If needs more detail**

    Use the **Github tools** to add a comment to the issue. Use the following template:

    ```
    @{REPORTER}

    Before we can address this, we need more information. Please provide:
    {DETAILED DESCRIPTION OF INFORMATION NEEDED}
    ```

    Limit comment request to: 5 bullets or 300 words maximum.

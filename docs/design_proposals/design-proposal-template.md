# Title

* Author(s): \<your name\>
* Reviewers: \<reviewer name\>

    If you are already working with someone mention their name.
    If not, please leave this empty, some one from the core team with assign it to themselves.
* Date: \<date\>
* Status: [Reviewed/Cancelled/Under implementation/Complete]

Here is a brief explanation of the Statuses

1. Reviewed: The proposal PR has been accepted, merged and ready for
   implementation.
2. Under implementation: An accepted proposal is being implemented by actual work.
   Note: The design might change in this phase based on issues during
   implementation.
3. Cancelled: During or before implementation the proposal was cancelled.
   It could be due to:
   * other features added which made the current design proposal obsolete.
   * No longer a priority.
4. Complete: This feature/change is implemented.

## Background

In this section, please mention and describe the new feature, redesign
or refactor.

Please provide a brief explanation for the following questions:

1. Why is this required?
2. If this is a redesign, what are the drawbacks of the current implementation?
3. Is there any another workaround, and if so, what are its drawbacks?
4. Mention related issues, if there are any.

Here is an example snippet for an enhancement:

___
Currently, Kaniko includes `build-args` when calculating layer cache key even if they are not used
in the corresponding dockerfile command.

This causes a 100% cache miss rate even if the layer contents are same. 
Change layer caching to include `build-args` in cache key computation only if they are used in command.
___

## Design

Please describe your solution. Please list any:

* new command line flags
* interface changes
* design assumptions

### Open Issues/Questions

Please list any open questions here in the following format:

**\<Question\>**

Resolution: Please list the resolution if resolved during the design process or
specify __Not Yet Resolved__

## Implementation plan
As a team, we've noticed that larger PRs can go unreviewed for long periods of
time. Small incremental changes get reviewed faster and are also easier for
reviewers.
___


## Integration test plan

Please describe what new test cases you are going to consider.
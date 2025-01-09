# Issue Lifecycle

To ensure a balance between work carried out by the NGINX engineering team while encouraging community involvement on
this project, we use the following issue lifecycle. (Note: The issue *creator* refers to the community member that
created the issue. The issue *owner* refers to the NGINX team member that is responsible for managing the issue
lifecycle.)

1. New issue created by community member.

2. Assign issue owner: All new issues are assigned an owner on the NGINX engineering team. This owner shepherds the
   issue through the subsequent stages in the issue lifecycle.

3. Determine issue type: This is done with automation where possible, and manually by the owner where necessary. The
   associated label is applied to the issue.

   Possible Issue Types:

   - `needs more info`: The owner should use the issue to request information from the creator. If we don't receive the
   needed information within 7 days, automation closes the issue.

   - `bug`: The implementation of a feature is not correct.

   - `proposal`: Request for a change. This can be a new feature, tackling technical debt, documentation changes, or
   improving existing features.

   - `question`: The owner converts the issue to a github discussion and engages the creator.

4. Determine milestone: The owner, in collaboration with the wider team (PM & engineering), determines what milestone to
   attach to an issue. Generally, milestones correspond to product releases - however there are two 'magic' milestones
   with special meanings (not tied to a specific release):

   - Issues assigned to backlog: Our team is in favour of implementing the feature request/fixing the issue, however the
     implementation is not yet assigned to a concrete release. If and when a `backlog` issue aligns well with our
     roadmap, it will be scheduled for a concrete iteration. We review and update our roadmap at least once every
     quarter. The `backlog` list helps us shape our roadmap, but it is not the only source of input. Therefore, some
     `backlog` items may eventually be closed as `out of scope`, or relabelled as `backlog candidate` once it becomes
     clear that they do not align with our evolving roadmap.

   - Issues assigned to `backlog candidate`: Our team does not intend to implement the feature/fix request described in
     the issue and wants the community to weigh in before we make our final decision.

    `backlog` issues can be labeled by the owner as `help wanted` and/or `good first issue` as appropriate.

5. Promotion of `backlog candidate` issue to `backlog` issue: If an issue labelled `backlog candidate` receives more
   than 30 upvotes within 60 days, we promote the issue by applying the `backlog` label. While issues promoted in this
   manner have not been committed to a particular release, we welcome PRs from the community on them.

   If an issue does not make our roadmap and has not been moved to a discussion, it is closed with the label `out of
   scope`. The goal is to get every issue in the issues list to one of the following end states:

   - An assigned release.
   - The `backlog` label.
   - Closed as `out of scope`.

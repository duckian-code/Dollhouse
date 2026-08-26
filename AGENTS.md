# Repository Agent Rules

## Git commits

- Use a lightweight Conventional Commits format for commit subjects:
  `type: concise description` or, when useful,
  `type(scope): concise description`.
- Prefer lowercase types such as `feat`, `fix`, `refactor`, `build`, `chore`,
  `docs`, `test`, `style`, `perf`, `ci`, and `revert`.
- Do not assume a branch. Use the branch specified by the user or the branch
  appropriate to the assigned work.

## Trello workflow and ticket scope

- Before implementing project work, find and read the current Google Docs named
  `Project Proposal - Dollhouse` and `Dollhouse API Contracts`. Use the proposal
  for product and architecture context and the API contracts for request,
  response, route, validation, and error-format requirements.
- Use the Google Docs to understand and implement the assigned ticket, but do
  not treat them as authorization to expand its scope. The target Trello card
  and the board's current state determine which work belongs to the ticket.
- Before implementing a ticket, inspect the Trello board and the full target
  card, including its description, list, dependencies, and checklist items.
- Review which related tickets are complete, in progress, abandoned, or still
  pending so the implementation respects the project's current state.
- Follow the board's Kanban work-in-progress limit. Before starting a ticket,
  inspect the configured limit and current occupancy of the `DOING` list.
- An agent may begin work only when `DOING` has available capacity. Move the
  assigned ticket into `DOING` before making implementation changes.
- If `DOING` is at its limit, do not start the assigned ticket. Check whether an
  existing ticket can legitimately move out of `DOING` based on its actual
  status. If the agent cannot resolve the capacity blocker accurately and within
  its authority, ask the user or responsible team member to resolve it and wait
  until space is available.
- Implement only the work belonging to the assigned ticket. Do not implement
  work reserved for other tickets.
- When the work is complete, move the target card to `DONE` and mark the card
  complete.
- Mark each checklist item complete only after its work has actually been
  implemented and verified.
- Leave blocked, deferred, unavailable, or otherwise incomplete checklist items
  unchecked. Do not mark them complete merely because the rest of the ticket is
  finished.
- Re-read the card and its checklist after updating Trello to verify that every
  status accurately reflects the completed work.

# Channel Skip Auto Test Design

## Goal
Allow a channel to remain enabled while being excluded from scheduled auto-test runs only.

## Design
- Add a persisted boolean field `skip_auto_test` to `model.Channel`, default false.
- Scheduled auto-test path skips channels where `skip_auto_test=true`.
- Manual "test all channels" keeps current behavior and still tests those channels.
- Expose the flag in the channel edit modal with explanatory help text.
- Cover backend behavior with tests for scheduled vs manual execution filtering.

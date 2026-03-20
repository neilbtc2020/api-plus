import test from 'node:test';
import assert from 'node:assert/strict';

import { getRecentResultSummary, shouldShowAvailabilityItem } from './modelAvailability.utils.js';

test('getRecentResultSummary marks default-green windows clearly', () => {
  const summary = getRecentResultSummary([
    { status: 'success', defaulted: true },
    { status: 'success', defaulted: true },
  ]);

  assert.equal(summary.hasOnlyDefaulted, true);
  assert.equal(summary.failCount, 0);
});

test('shouldShowAvailabilityItem respects keyword and only_failed', () => {
  const item = { model_name: 'gpt-4o', fail_count: 2, has_real_logs: true };

  assert.equal(
    shouldShowAvailabilityItem(item, { keyword: 'gpt', onlyFailed: true, onlyWithLogs: false }),
    true,
  );
  assert.equal(
    shouldShowAvailabilityItem(item, { keyword: 'claude', onlyFailed: true, onlyWithLogs: false }),
    false,
  );
});

export function getRecentResultSummary(results = []) {
  const failCount = results.filter((item) => item?.status === 'fail').length;
  const realCount = results.filter((item) => !item?.defaulted).length;

  return {
    failCount,
    realCount,
    hasOnlyDefaulted: realCount === 0,
  };
}

export function shouldShowAvailabilityItem(item, filters = {}) {
  const keyword = (filters.keyword || '').trim().toLowerCase();

  if (keyword && !(item?.model_name || '').toLowerCase().includes(keyword)) {
    return false;
  }
  if (filters.onlyFailed && (item?.fail_count || 0) === 0) {
    return false;
  }
  if (filters.onlyWithLogs && !item?.has_real_logs) {
    return false;
  }

  return true;
}

# Group Model Availability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a dedicated logged-in page that shows each visible group’s models with configuration availability, the latest 20 global request outcomes (green/red dots with default-green fill), manual refresh, single-model probe, and a homepage entry card.

**Architecture:** Keep DB access in `model/`, feature assembly and cache management in `service/`, and HTTP/probe orchestration in `controller/`. Reuse existing `abilities`, `logs`, and `testChannel(...)` logic; store snapshots and probe results in `pkg/cachex` hybrid caches so Redis is used when available and in-memory fallback still works when Redis is disabled.

**Tech Stack:** Go 1.25 + Gin + GORM + `pkg/cachex`/`samber/hot` on the backend; React 18 + Semi UI + axios + bun on the frontend.

---

## File Structure

### Backend

- Create: `model/model_availability.go`
  - Feature-specific DB queries for:
    - recent logs by group
    - enabled channels by `group + model`
    - visible group list for admin/non-admin consumers
- Create: `service/model_availability.go`
  - Snapshot DTOs, dot-window assembly, filtering, selected-group resolution, “default green” fill
- Create: `service/model_availability_cache.go`
  - Hybrid cache instances, namespaces, TTL helpers, cache read/write helpers
- Create: `service/model_availability_test.go`
  - Unit tests for recent-results assembly, filtering, selected-group fallback, probe outcome merge helpers
- Create: `controller/model_availability.go`
  - `GET /api/model-availability`
  - `POST /api/model-availability/refresh`
  - `POST /api/model-availability/probe`
  - controller-local probe runner that reuses `testChannel(...)`
- Create: `controller/model_availability_test.go`
  - Unit tests for probe summarization / controller helper logic
- Modify: `router/api-router.go`
  - Register authenticated model-availability routes

### Frontend

- Create: `web/src/pages/ModelAvailability/index.jsx`
  - Page wrapper, consistent with existing `pages/*/index.jsx` files
- Create: `web/src/components/model-availability/index.jsx`
  - Feature entry component
- Create: `web/src/components/model-availability/ModelAvailabilityLayout.jsx`
  - Left/right shell layout
- Create: `web/src/components/model-availability/GroupList.jsx`
  - Visible group switcher
- Create: `web/src/components/model-availability/ModelAvailabilityFilters.jsx`
  - keyword / only_failed / only_with_logs controls
- Create: `web/src/components/model-availability/ModelAvailabilityTable.jsx`
  - main table/list
- Create: `web/src/components/model-availability/RecentResultDots.jsx`
  - 20-dot renderer with default-green indicator
- Create: `web/src/components/model-availability/ProbeStatusTag.jsx`
  - probe status display
- Create: `web/src/components/model-availability/modelAvailability.utils.js`
  - pure frontend helpers for dot labels / filter counts / empty-state copy
- Create: `web/src/components/model-availability/modelAvailability.utils.test.js`
  - Node built-in tests for those pure helpers
- Create: `web/src/hooks/model-availability/useModelAvailabilityData.jsx`
  - API requests, local UI state, refresh / probe actions
- Create: `web/src/components/home/ModelAvailabilityEntryCard.jsx`
  - isolated homepage entry card to avoid bloating the already-large home page file
- Modify: `web/src/App.jsx`
  - add authenticated route for `/model-availability`
- Modify: `web/src/pages/Home/index.jsx`
  - render the new entry card in the default home page
- Modify: `web/src/i18n/locales/zh-CN.json`
  - add source keys for new UI strings
- Modify: `web/src/i18n/locales/en.json`
  - add English translations for the new UI strings

---

### Task 1: Build backend snapshot assembly helpers first

**Files:**
- Create: `service/model_availability.go`
- Create: `service/model_availability_test.go`

- [ ] **Step 1: Write the failing service tests for recent-result windows, filtering, and selected-group fallback**

```go
func TestBuildRecentResultsWindow_FillsMissingWithDefaultGreen(t *testing.T) {
    logs := []availabilityLogEntry{
        {ModelName: "gpt-4o", Success: false},
        {ModelName: "gpt-4o", Success: true},
    }

    window := buildRecentResultsWindow(logs, 5)

    require.Len(t, window, 5)
    require.Equal(t, availabilityResultFail, window[0].Status)
    require.False(t, window[0].Defaulted)
    require.Equal(t, availabilityResultSuccess, window[2].Status)
    require.True(t, window[2].Defaulted)
}

func TestFilterAvailabilityItems_OnlyFailedAndKeyword(t *testing.T) {
    items := []ModelAvailabilityItem{
        {ModelName: "gpt-4o", FailCount: 2, RecentResults: []RecentResult{{Status: availabilityResultFail}}},
        {ModelName: "claude-3-7-sonnet", FailCount: 0, RecentResults: []RecentResult{{Status: availabilityResultSuccess, Defaulted: true}}},
    }

    filtered := filterAvailabilityItems(items, "gpt", true, false)
    require.Len(t, filtered, 1)
    require.Equal(t, "gpt-4o", filtered[0].ModelName)
}

func TestResolveSelectedGroup_FallsBackToFirstVisible(t *testing.T) {
    groups := []VisibleGroup{
        {Name: "vip"},
        {Name: "default"},
    }

    selected := resolveSelectedGroup(groups, "not-exists")
    require.Equal(t, "vip", selected)
}
```

- [ ] **Step 2: Run the targeted backend tests to confirm they fail before implementation**

Run:

```bash
go test ./service -run 'TestBuildRecentResultsWindow|TestFilterAvailabilityItems|TestResolveSelectedGroup' -count=1
```

Expected: FAIL with undefined `buildRecentResultsWindow`, `filterAvailabilityItems`, and `resolveSelectedGroup`.

- [ ] **Step 3: Implement the pure snapshot-building types and helpers**

```go
type RecentResult struct {
    Status    string `json:"status"`
    Defaulted bool   `json:"defaulted"`
}

type ModelAvailabilityItem struct {
    ModelName        string         `json:"model_name"`
    ConfigAvailable  bool           `json:"config_available"`
    RecentResults    []RecentResult `json:"recent_results"`
    SuccessCount     int            `json:"success_count"`
    FailCount        int            `json:"fail_count"`
    HasRealLogs      bool           `json:"has_real_logs"`
    Probe            *ProbeStatus   `json:"probe,omitempty"`
}

func buildRecentResultsWindow(entries []availabilityLogEntry, windowSize int) []RecentResult {
    results := make([]RecentResult, 0, windowSize)
    for _, entry := range entries {
        if len(results) == windowSize {
            break
        }
        status := availabilityResultSuccess
        if !entry.Success {
            status = availabilityResultFail
        }
        results = append(results, RecentResult{Status: status})
    }
    for len(results) < windowSize {
        results = append(results, RecentResult{Status: availabilityResultSuccess, Defaulted: true})
    }
    return results
}
```

- [ ] **Step 4: Re-run the targeted backend tests**

Run:

```bash
go test ./service -run 'TestBuildRecentResultsWindow|TestFilterAvailabilityItems|TestResolveSelectedGroup' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the helper layer**

```bash
git add service/model_availability.go service/model_availability_test.go
git commit -m "feat: add model availability snapshot helpers"
```

---

### Task 2: Wire backend queries, caches, and read/refresh endpoints

**Files:**
- Create: `model/model_availability.go`
- Create: `service/model_availability_cache.go`
- Create: `controller/model_availability.go`
- Modify: `router/api-router.go`
- Test: `controller/model_availability_test.go`

- [ ] **Step 1: Write the failing controller/helper tests for refresh selection and stale-cache fallback behavior**

```go
func TestMarkSelectedGroup(t *testing.T) {
    groups := []service.VisibleGroup{{Name: "vip"}, {Name: "default"}}
    marked := markSelectedGroup(groups, "default")
    require.False(t, marked[0].Selected)
    require.True(t, marked[1].Selected)
}

func TestWithSnapshotWarning_UsesWarningWhenDataIsStale(t *testing.T) {
    snapshot := service.GroupAvailabilitySnapshot{SelectedGroup: "vip"}
    response := withSnapshotWarning(snapshot, true)
    require.Contains(t, response.Warning, "过期")
}
```

- [ ] **Step 2: Run the targeted controller tests to confirm the helpers do not exist yet**

Run:

```bash
go test ./controller -run 'TestMarkSelectedGroup|TestWithSnapshotWarning' -count=1
```

Expected: FAIL with undefined helper names.

- [ ] **Step 3: Implement the DB queries, hybrid caches, read endpoint, and refresh endpoint**

```go
// model/model_availability.go
func GetRecentAvailabilityLogsByGroup(group string, limit int) ([]AvailabilityLogRow, error) {
    rows := make([]AvailabilityLogRow, 0, limit)
    err := LOG_DB.Model(&Log{}).
        Select("id, type, model_name, created_at").
        Where("logs."+logGroupCol+" = ? AND logs.type IN ?", group, []int{LogTypeConsume, LogTypeError}).
        Order("logs.id DESC").
        Limit(limit).
        Find(&rows).Error
    return rows, err
}

// service/model_availability_cache.go
var modelAvailabilitySnapshotCache = cachex.NewHybridCache[GroupAvailabilitySnapshot](...)

// controller/model_availability.go
func GetModelAvailability(c *gin.Context) {
    snapshot, stale, err := service.LoadGroupAvailabilitySnapshot(c.GetInt("id"), c.GetInt("role"), c.Query("group"), c.Query("keyword"), c.Query("only_failed") == "true", c.Query("only_with_logs") == "true")
    if err != nil {
        common.ApiError(c, err)
        return
    }
    common.ApiSuccess(c, withSnapshotWarning(snapshot, stale))
}

func RefreshModelAvailability(c *gin.Context) {
    // Rebuild from DB and overwrite cache for the requested group.
}
```

- [ ] **Step 4: Run the targeted tests and a broader backend pass**

Run:

```bash
go test ./controller -run 'TestMarkSelectedGroup|TestWithSnapshotWarning' -count=1
go test ./service ./controller -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the read/refresh backend slice**

```bash
git add model/model_availability.go service/model_availability_cache.go controller/model_availability.go controller/model_availability_test.go router/api-router.go
git commit -m "feat: add model availability snapshot endpoints"
```

---

### Task 3: Add single-model probe support on top of existing channel test logic

**Files:**
- Modify: `model/model_availability.go`
- Modify: `controller/model_availability.go`
- Modify: `controller/model_availability_test.go`

- [ ] **Step 1: Write the failing controller tests for probe result summarization**

```go
func TestSummarizeProbeAttempts_AnySuccessWins(t *testing.T) {
    attempts := []probeAttemptSummary{
        {ChannelID: 1, Success: false, Message: "timeout"},
        {ChannelID: 2, Success: true, Message: "ok", ResponseTimeMs: 850},
    }

    result := summarizeProbeAttempts(attempts)
    require.Equal(t, "success", result.Status)
    require.EqualValues(t, 850, result.ResponseTimeMs)
}

func TestSummarizeProbeAttempts_AllFailKeepsLastMessage(t *testing.T) {
    attempts := []probeAttemptSummary{
        {ChannelID: 1, Success: false, Message: "timeout"},
        {ChannelID: 2, Success: false, Message: "upstream 500"},
    }

    result := summarizeProbeAttempts(attempts)
    require.Equal(t, "fail", result.Status)
    require.Contains(t, result.Message, "upstream 500")
}
```

- [ ] **Step 2: Run the targeted probe tests to confirm they fail**

Run:

```bash
go test ./controller -run 'TestSummarizeProbeAttempts' -count=1
```

Expected: FAIL with undefined `summarizeProbeAttempts`.

- [ ] **Step 3: Implement enabled-channel lookup plus probe execution**

```go
// model/model_availability.go
func GetEnabledChannelsByGroupModel(group, modelName string) ([]*Channel, error) {
    channels := make([]*Channel, 0)
    err := DB.Table("channels").
        Joins("JOIN abilities ON abilities.channel_id = channels.id").
        Where("abilities."+commonGroupCol+" = ? AND abilities.model = ? AND abilities.enabled = ? AND channels.status = ?", group, modelName, true, common.ChannelStatusEnabled).
        Order("channels.priority DESC, channels.id DESC").
        Find(&channels).Error
    return channels, err
}

// controller/model_availability.go
func ProbeModelAvailability(c *gin.Context) {
    // bind JSON -> load channels -> loop testChannel(channel, modelName, "", false)
    // summarize attempts -> cache result -> return ApiSuccess
}
```

- [ ] **Step 4: Re-run the probe tests and the backend package suite**

Run:

```bash
go test ./controller -run 'TestSummarizeProbeAttempts' -count=1
go test ./service ./controller ./model -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the probe implementation**

```bash
git add model/model_availability.go controller/model_availability.go controller/model_availability_test.go
git commit -m "feat: add model availability probe support"
```

---

### Task 4: Add frontend utility coverage for dot labels and filter behavior

**Files:**
- Create: `web/src/components/model-availability/modelAvailability.utils.js`
- Create: `web/src/components/model-availability/modelAvailability.utils.test.js`

- [ ] **Step 1: Write failing Node tests for the pure frontend helpers**

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import {
  getRecentResultSummary,
  shouldShowAvailabilityItem,
} from './modelAvailability.utils.js';

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
  assert.equal(shouldShowAvailabilityItem(item, { keyword: 'gpt', onlyFailed: true, onlyWithLogs: false }), true);
  assert.equal(shouldShowAvailabilityItem(item, { keyword: 'claude', onlyFailed: true, onlyWithLogs: false }), false);
});
```

- [ ] **Step 2: Run the Node tests to confirm failure**

Run:

```bash
cd web && node --test src/components/model-availability/modelAvailability.utils.test.js
```

Expected: FAIL with missing exports.

- [ ] **Step 3: Implement the pure helpers**

```js
export function getRecentResultSummary(results = []) {
  const failCount = results.filter((item) => item.status === 'fail').length;
  const realCount = results.filter((item) => !item.defaulted).length;
  return {
    failCount,
    realCount,
    hasOnlyDefaulted: realCount === 0,
  };
}

export function shouldShowAvailabilityItem(item, filters) {
  const keyword = (filters.keyword || '').trim().toLowerCase();
  if (keyword && !(item.model_name || '').toLowerCase().includes(keyword)) {
    return false;
  }
  if (filters.onlyFailed && (item.fail_count || 0) === 0) {
    return false;
  }
  if (filters.onlyWithLogs && !item.has_real_logs) {
    return false;
  }
  return true;
}
```

- [ ] **Step 4: Re-run the Node utility tests**

Run:

```bash
cd web && node --test src/components/model-availability/modelAvailability.utils.test.js
```

Expected: PASS.

- [ ] **Step 5: Commit the tested utility layer**

```bash
git add web/src/components/model-availability/modelAvailability.utils.js web/src/components/model-availability/modelAvailability.utils.test.js
git commit -m "feat: add model availability frontend utilities"
```

---

### Task 5: Build the new page, hook, and authenticated route

**Files:**
- Create: `web/src/pages/ModelAvailability/index.jsx`
- Create: `web/src/components/model-availability/index.jsx`
- Create: `web/src/components/model-availability/ModelAvailabilityLayout.jsx`
- Create: `web/src/components/model-availability/GroupList.jsx`
- Create: `web/src/components/model-availability/ModelAvailabilityFilters.jsx`
- Create: `web/src/components/model-availability/ModelAvailabilityTable.jsx`
- Create: `web/src/components/model-availability/RecentResultDots.jsx`
- Create: `web/src/components/model-availability/ProbeStatusTag.jsx`
- Create: `web/src/hooks/model-availability/useModelAvailabilityData.jsx`
- Modify: `web/src/App.jsx`

- [ ] **Step 1: Add the route and the data hook shell, then intentionally call the not-yet-wired component to force a build failure**

```jsx
// web/src/App.jsx
import ModelAvailabilityPage from './pages/ModelAvailability';

<Route
  path='/model-availability'
  element={
    <PrivateRoute>
      <ModelAvailabilityPage />
    </PrivateRoute>
  }
/>
```

- [ ] **Step 2: Run the frontend build so the missing page/component wiring fails loudly**

Run:

```bash
cd web && bun run build
```

Expected: FAIL with missing `./pages/ModelAvailability` or unresolved component exports.

- [ ] **Step 3: Implement the page and UI components using the tested frontend utilities**

```jsx
export const useModelAvailabilityData = () => {
  const [groups, setGroups] = useState([]);
  const [items, setItems] = useState([]);
  const [selectedGroup, setSelectedGroup] = useState('');

  const load = async (group = '') => {
    const res = await API.get('/api/model-availability', {
      params: {
        group,
        keyword,
        only_failed: onlyFailed,
        only_with_logs: onlyWithLogs,
      },
    });
    setGroups(res.data.data.groups || []);
    setItems(res.data.data.items || []);
    setSelectedGroup(res.data.data.selected_group || '');
  };

  return { groups, items, selectedGroup, load, refresh, probeModel };
};
```

- [ ] **Step 4: Re-run the frontend build**

Run:

```bash
cd web && bun run build
```

Expected: PASS.

- [ ] **Step 5: Commit the page route and UI**

```bash
git add web/src/App.jsx web/src/pages/ModelAvailability/index.jsx web/src/components/model-availability web/src/hooks/model-availability/useModelAvailabilityData.jsx
git commit -m "feat: add model availability page"
```

---

### Task 6: Add the homepage entry card, translations, and end-to-end verification

**Files:**
- Create: `web/src/components/home/ModelAvailabilityEntryCard.jsx`
- Modify: `web/src/pages/Home/index.jsx`
- Modify: `web/src/i18n/locales/zh-CN.json`
- Modify: `web/src/i18n/locales/en.json`

- [ ] **Step 1: Add the homepage entry component and translation keys**

```jsx
export default function ModelAvailabilityEntryCard() {
  const { t } = useTranslation();
  return (
    <Link to='/model-availability'>
      <div className='rounded-2xl border border-semi-color-border p-5 bg-semi-color-bg-0 hover:shadow-md transition-shadow'>
        <div className='text-lg font-semibold'>{t('模型可用性')}</div>
        <div className='text-semi-color-text-1 mt-2 text-sm'>
          {t('查看每个分组下各模型最近 20 次请求的成功与失败情况')}
        </div>
      </div>
    </Link>
  );
}
```

- [ ] **Step 2: Run the frontend build after wiring the homepage card**

Run:

```bash
cd web && bun run build
```

Expected: PASS.

- [ ] **Step 3: Run the targeted backend and frontend verification suite**

Run:

```bash
go test ./service ./controller ./model -count=1
go test ./... -count=1
cd web && node --test src/components/model-availability/modelAvailability.utils.test.js
cd web && bun run build
```

Expected:

- all Go tests PASS
- Node utility tests PASS
- Vite production build PASS

- [ ] **Step 4: Perform the manual smoke test**

Run the app, log in, then verify:

1. Home page shows the new “模型可用性” card.
2. Clicking it lands on `/model-availability`.
3. Group switching works.
4. Failed models appear when `only_failed` is enabled.
5. A model with no logs still shows green dots plus the default-green explanation.
6. Clicking “立即重测” updates the probe status tag.

- [ ] **Step 5: Commit the homepage entry and translation work**

```bash
git add web/src/components/home/ModelAvailabilityEntryCard.jsx web/src/pages/Home/index.jsx web/src/i18n/locales/zh-CN.json web/src/i18n/locales/en.json
git commit -m "feat: add homepage entry for model availability"
```

---

## Final Verification Checklist

- [ ] `go test ./service ./controller ./model -count=1`
- [ ] `go test ./... -count=1`
- [ ] `cd web && node --test src/components/model-availability/modelAvailability.utils.test.js`
- [ ] `cd web && bun run build`
- [ ] Manual smoke test on the logged-in homepage and `/model-availability`

## Notes for the Implementer

- Do **not** use window functions for the “latest 20 logs” query; keep it SQLite/MySQL/PostgreSQL compatible by fetching a bounded recent log slice and aggregating in memory.
- Reuse existing `testChannel(...)` behavior for probe attempts rather than inventing a second health-check path.
- Keep `auto` out of V1 group tabs; the approved design uses real `using_group` log buckets only.
- Treat “no logs” as **default green**, but always render the explanatory copy so users can distinguish it from real successful traffic.
- Prefer `pkg/cachex` hybrid caches over ad hoc Redis strings/maps so Redis-disabled deployments still work.

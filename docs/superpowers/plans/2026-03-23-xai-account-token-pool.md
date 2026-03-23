# xAI Account Token Pool Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an `account_token` auth mode to the existing `xAI` channel so admins can paste one Grok account token per line, reuse the current multi-key infrastructure, and get per-token disable/cooldown behavior without regressing the current `xAI API Key` mode.

**Architecture:** Keep `ChannelTypeXai` unchanged and add an explicit `xai_auth_mode` setting in `ChannelOtherSettings`. For `account_token`, branch inside `relay/channel/xai` to a token-pool transport that re-selects tokens from the channel on each attempt, persists permanent-disable state in `ChannelInfo`, persists cooldown timestamps in `OtherInfo`, and only bubbles a final error after the pool is exhausted. Add a dedicated xAI video task adaptor so both auth modes share the same xAI capability surface.

**Tech Stack:** Go 1.22+, Gin, GORM, existing relay/channel abstractions, React 18 + Vite + Semi UI, Bun.

---

## File Structure

### Backend files to modify
- Modify: `dto/channel_settings.go` — add xAI auth-mode enum/constants and persistable field on `ChannelOtherSettings`
- Modify: `controller/channel.go` — validate xAI auth mode and newline token input; keep add/update flows compatible with existing multi-key behavior
- Modify: `relay/channel/xai/adaptor.go` — branch request URL/header/request loop by auth mode; preserve existing API key behavior
- Modify: `relay/relay_adaptor.go` — register a new xAI video task adaptor

### Backend files to create
- Create: `controller/channel_xai_validation_test.go` — validate `xai_auth_mode`, newline token acceptance, and rejection cases
- Create: `relay/channel/xai/auth_mode.go` — normalize auth mode and provide defaults
- Create: `relay/channel/xai/account_pool.go` — parse account tokens, load/save cooldown state, pick next usable token, mark disable/cooldown, update request context when token changes
- Create: `relay/channel/xai/account_pool_test.go` — token parsing, cooldown recovery, disabled-token skipping, random/polling behavior
- Create: `relay/channel/xai/account_request.go` — account-mode request URL/header/cookie builders, retry loop, final error shaping
- Create: `relay/channel/xai/adaptor_test.go` — prove API key mode is unchanged and account-token mode uses pool transport
- Create: `relay/channel/task/xai/adaptor.go` — xAI video submit/fetch adaptor with auth-mode branching
- Create: `relay/channel/task/xai/dto.go` — request/response/task status structs for xAI video operations
- Create: `relay/channel/task/xai/adaptor_test.go` — task URL/header/body and polling tests

### Frontend files to modify
- Modify: `web/src/components/table/channels/modals/EditChannelModal.jsx` — add `xAI` auth-mode select, change secret prompt/extra help text, keep existing batch and multi-key controls working

### Verification commands to use during implementation
- `go test ./controller -run 'TestValidateChannelXAI|TestNormalizeXAIAuthMode' -v`
- `go test ./relay/channel/xai -run 'TestXAI' -v`
- `go test ./relay/channel/task/xai -run 'TestXAITask' -v`
- `go test ./relay/... ./middleware/... ./controller/... -run 'Test.*XAI|Test.*Channel|Test.*MultiKey' -v`
- `go test ./...`
- `cd web && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat ../VERSION) bun run build`

---

### Task 1: Add xAI auth-mode configuration and channel validation

**Files:**
- Modify: `dto/channel_settings.go`
- Modify: `controller/channel.go`
- Test: `controller/channel_xai_validation_test.go`

- [ ] **Step 1: Write the failing validation tests**

```go
func TestValidateChannelXAIAccountTokenModeAcceptsMultilineTokens(t *testing.T) {
	channel := &model.Channel{
		Type: constant.ChannelTypeXai,
		Key:  "token-a\n\n token-b ",
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{XAIAuthMode: dto.XAIAuthModeAccountToken})
	require.NoError(t, validateChannel(channel, true))
}

func TestValidateChannelXAIAccountTokenModeRejectsEmptyKey(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeXai, Key: "   "}
	channel.SetOtherSettings(dto.ChannelOtherSettings{XAIAuthMode: dto.XAIAuthModeAccountToken})
	require.Error(t, validateChannel(channel, true))
}
```

- [ ] **Step 2: Run the controller tests to verify they fail**

Run: `go test ./controller -run 'TestValidateChannelXAI|TestNormalizeXAIAuthMode' -v`
Expected: FAIL because `XAIAuthMode` constants/field do not exist and xAI-specific validation is missing.

- [ ] **Step 3: Implement the smallest backend config/validation changes**

```go
type XAIAuthMode string

const (
	XAIAuthModeAPIKey       XAIAuthMode = "api_key"
	XAIAuthModeAccountToken XAIAuthMode = "account_token"
)

type ChannelOtherSettings struct {
	// ...existing fields...
	XAIAuthMode XAIAuthMode `json:"xai_auth_mode,omitempty"`
}

func (s *ChannelOtherSettings) NormalizeXAIAuthMode() XAIAuthMode {
	if s == nil || s.XAIAuthMode == "" {
		return XAIAuthModeAPIKey
	}
	if s.XAIAuthMode == XAIAuthModeAccountToken {
		return XAIAuthModeAccountToken
	}
	return XAIAuthModeAPIKey
}
```

In `validateChannel`, add an xAI branch that:
- reads `channel.GetOtherSettings().NormalizeXAIAuthMode()`
- for `account_token`, splits `channel.Key` by newline, trims whitespace, drops blanks, and errors if no token remains
- leaves `api_key` behavior unchanged

- [ ] **Step 4: Re-run the controller tests to verify they pass**

Run: `go test ./controller -run 'TestValidateChannelXAI|TestNormalizeXAIAuthMode' -v`
Expected: PASS.

- [ ] **Step 5: Commit the config/validation slice**

```bash
git add dto/channel_settings.go controller/channel.go controller/channel_xai_validation_test.go
git commit -m "feat: add xai auth mode validation"
```

---

### Task 2: Build reusable xAI account-token pool helpers

**Files:**
- Create: `relay/channel/xai/auth_mode.go`
- Create: `relay/channel/xai/account_pool.go`
- Test: `relay/channel/xai/account_pool_test.go`

- [ ] **Step 1: Write failing pool tests for parse/select/cooldown behavior**

```go
func TestXAISelectAccountTokenSkipsDisabledAndCooldown(t *testing.T) {
	channel := &model.Channel{Key: "token-a\ntoken-b\ntoken-c"}
	channel.ChannelInfo = model.ChannelInfo{
		IsMultiKey:         true,
		MultiKeySize:       3,
		MultiKeyStatusList: map[int]int{0: common.ChannelStatusAutoDisabled},
	}
	channel.SetOtherInfo(map[string]any{"xai_account_cooldown_until": map[string]any{"1": float64(now + 60)}})

	token, idx, state := selectAccountToken(channel, now, constant.MultiKeyModePolling)
	require.Equal(t, "token-c", token)
	require.Equal(t, 2, idx)
	require.False(t, state.Exhausted)
}
```

- [ ] **Step 2: Run the xAI pool tests to verify they fail**

Run: `go test ./relay/channel/xai -run 'TestXAISelectAccountToken|TestXAIParseAccountTokens|TestXAIRecoverExpiredCooldown' -v`
Expected: FAIL because the helper files do not exist.

- [ ] **Step 3: Implement pool helpers with explicit state persistence rules**

```go
type accountPoolState struct {
	CooldownUntil map[int]int64
	AllDisabled    bool
	AllCoolingDown bool
}

func parseAccountTokens(raw string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
	}
	return out
}
```

Implementation notes:
- Keep all xAI-specific pool logic inside `relay/channel/xai`, not `model`.
- Read/write cooldowns under `OtherInfo["xai_account_cooldown_until"]`.
- Use `channel.SaveWithoutKey()` after mutating `OtherInfo`.
- When a cooldown has expired, delete it immediately (“lazy recovery”) before selection finishes.
- Add a helper that updates `gin.Context` + `RelayInfo` when an internal retry switches tokens:
  - `ContextKeyChannelKey`
  - `ContextKeyChannelMultiKeyIndex`
  - `info.ApiKey`
  - `info.ChannelMultiKeyIndex`

- [ ] **Step 4: Re-run the xAI pool tests to verify they pass**

Run: `go test ./relay/channel/xai -run 'TestXAISelectAccountToken|TestXAIParseAccountTokens|TestXAIRecoverExpiredCooldown' -v`
Expected: PASS.

- [ ] **Step 5: Commit the helper layer**

```bash
git add relay/channel/xai/auth_mode.go relay/channel/xai/account_pool.go relay/channel/xai/account_pool_test.go
git commit -m "feat: add xai account token pool helpers"
```

---

### Task 3: Integrate account-token transport into the xAI adaptor for chat/responses/images

**Files:**
- Modify: `relay/channel/xai/adaptor.go`
- Create: `relay/channel/xai/account_request.go`
- Test: `relay/channel/xai/adaptor_test.go`

- [ ] **Step 1: Write failing adaptor tests for auth-mode branching and retry behavior**

```go
func TestXAIAdaptorSetupRequestHeaderKeepsAPIKeyMode(t *testing.T) {
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ApiKey: "xai-api-key",
		ChannelOtherSettings: dto.ChannelOtherSettings{XAIAuthMode: dto.XAIAuthModeAPIKey},
	}}
	headers := http.Header{}
	err := (&Adaptor{}).SetupRequestHeader(gin.CreateTestContextOnly(nil, nil), &headers, info)
	require.NoError(t, err)
	require.Equal(t, "Bearer xai-api-key", headers.Get("Authorization"))
}
```

Also add a test that a `401` on token-1 leads to a retry with token-2 and a disable mark for token-1.

- [ ] **Step 2: Run the xAI adaptor tests to verify they fail**

Run: `go test ./relay/channel/xai -run 'TestXAIAdaptor|TestXAIAccountRequest' -v`
Expected: FAIL because the account-mode transport and retry loop do not exist.

- [ ] **Step 3: Implement auth-mode branching with minimal churn**

Implementation shape:

```go
func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	if normalizeXAIAuthMode(info) == dto.XAIAuthModeAccountToken {
		return setupAccountTokenHeader(c, req, info)
	}
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	if normalizeXAIAuthMode(info) == dto.XAIAuthModeAccountToken {
		return doAccountTokenRequest(a, c, info, requestBody)
	}
	return channel.DoApiRequest(a, c, info, requestBody)
}
```

Detailed requirements for `doAccountTokenRequest`:
- Load the full channel via `model.GetChannelById(info.ChannelId, true)` so internal retries can see every token, not just the middleware-selected one.
- Loop within the same xAI channel:
  - choose token from pool
  - rebuild request body reader for each attempt
  - call `channel.DoApiRequest`
  - inspect `resp.StatusCode`
  - `401/403` -> `model.UpdateChannelStatus(channelID, token, common.ChannelStatusAutoDisabled, "xai account token unauthorized")`
  - `429` -> persist cooldown until `now + cooldownWindow`
  - success -> return response immediately
- If all tokens are cooling down, return a skip-retry/saturated error that does **not** auto-disable the channel.
- If all tokens are permanently disabled, disable the channel before returning a terminal error.
- Keep existing `ConvertOpenAIRequest`, `ConvertOpenAIResponsesRequest`, and image conversion behavior unchanged.

- [ ] **Step 4: Re-run the xAI adaptor tests to verify they pass**

Run: `go test ./relay/channel/xai -run 'TestXAIAdaptor|TestXAIAccountRequest' -v`
Expected: PASS.

- [ ] **Step 5: Commit the xAI transport integration**

```bash
git add relay/channel/xai/adaptor.go relay/channel/xai/account_request.go relay/channel/xai/adaptor_test.go
git commit -m "feat: add xai account token transport"
```

---

### Task 4: Add xAI video task support with auth-mode branching

**Files:**
- Create: `relay/channel/task/xai/adaptor.go`
- Create: `relay/channel/task/xai/dto.go`
- Modify: `relay/relay_adaptor.go`
- Test: `relay/channel/task/xai/adaptor_test.go`

- [ ] **Step 1: Write failing task-adaptor tests for submit/fetch routing**

```go
func TestXAITaskAdaptorBuildRequestURLForSubmit(t *testing.T) {
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType: constant.ChannelTypeXai,
		ChannelBaseUrl: "https://api.x.ai",
		UpstreamModelName: "grok-imagine-video",
	}}
	a := &TaskAdaptor{}
	a.Init(info)
	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	require.Contains(t, url, "/v1/videos")
}
```

- [ ] **Step 2: Run the xAI task tests to verify they fail**

Run: `go test ./relay/channel/task/xai -run 'TestXAITask' -v`
Expected: FAIL because the xAI task adaptor package is missing and `relay/relay_adaptor.go` does not register it.

- [ ] **Step 3: Implement the task adaptor with shared auth-mode branching**

Implementation notes:
- Follow the `relay/channel/task/gemini` and `relay/channel/task/sora` layout.
- `Init` should copy `info.ChannelBaseUrl`, `info.ApiKey`, `info.ChannelOtherSettings`.
- `BuildRequestHeader` should branch just like the non-task xAI adaptor:
  - API key mode -> bearer header
  - account token mode -> account-token header/cookie builder
- `DoRequest` should delegate to `channel.DoTaskApiRequest` for API key mode and to an internal retry loop for account-token mode (same 401/403/429 semantics as Task 3).
- `FetchTask` must honor the same auth mode so follow-up polling uses the same credential strategy.
- Update `relay/relay_adaptor.go` to return the new task adaptor for `constant.ChannelTypeXai`.

- [ ] **Step 4: Re-run the xAI task tests to verify they pass**

Run: `go test ./relay/channel/task/xai -run 'TestXAITask' -v`
Expected: PASS.

- [ ] **Step 5: Commit the video-task slice**

```bash
git add relay/channel/task/xai/adaptor.go relay/channel/task/xai/dto.go relay/channel/task/xai/adaptor_test.go relay/relay_adaptor.go
git commit -m "feat: add xai video task adaptor"
```

---

### Task 5: Add the xAI auth-mode UI and keep existing editing flows intact

**Files:**
- Modify: `web/src/components/table/channels/modals/EditChannelModal.jsx`

- [ ] **Step 1: Write down the expected UI states before editing**

Capture these cases in comments or a scratch checklist before changing code:
- xAI + API key mode -> current behavior and prompt stay unchanged
- xAI + account token mode -> prompt becomes “每行一个 token” and multi-key/batch remains allowed
- editing an existing xAI account-token channel -> selected mode and prompt round-trip from `settings`

- [ ] **Step 2: Run a frontend build baseline**

Run: `cd web && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat ../VERSION) bun run build`
Expected: PASS before edits.

- [ ] **Step 3: Implement the minimal UI changes**

Add a dedicated xAI auth-mode field to the modal when `inputs.type === 48`:

```jsx
<Form.Select
  field='xai_auth_mode'
  label={t('鉴权模式')}
  optionList={[
    { label: 'API Key', value: 'api_key' },
    { label: '账号 Token', value: 'account_token' },
  ]}
  value={inputs.xai_auth_mode || 'api_key'}
  onChange={(value) => handleChannelOtherSettingsChange('xai_auth_mode', value)}
/>
```

Then update the xAI secret prompt logic so account-token mode returns a concrete multiline hint, e.g.:

```jsx
if (type === 48 && xaiAuthMode === 'account_token') {
  return '请输入账号 Token，一行一个';
}
```

Also ensure the existing form-data hydration path reads `settings.xai_auth_mode` back into `inputs` during edit mode.

- [ ] **Step 4: Re-run the frontend build to verify it passes**

Run: `cd web && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat ../VERSION) bun run build`
Expected: PASS.

- [ ] **Step 5: Commit the frontend slice**

```bash
git add web/src/components/table/channels/modals/EditChannelModal.jsx
git commit -m "feat: add xai auth mode selector"
```

---

### Task 6: Run focused regression checks and final verification

**Files:**
- Verify only; no planned file creation

- [ ] **Step 1: Run focused Go regressions for xAI and channel handling**

Run: `go test ./controller ./middleware ./relay/channel/xai ./relay/channel/task/xai -v`
Expected: PASS.

- [ ] **Step 2: Run broader relay regressions**

Run: `go test ./relay/... -v`
Expected: PASS.

- [ ] **Step 3: Run the full Go test suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 4: Run the final frontend build**

Run: `cd web && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat ../VERSION) bun run build`
Expected: PASS.

- [ ] **Step 5: Commit the verified final state**

```bash
git status
git add dto/channel_settings.go controller/channel.go controller/channel_xai_validation_test.go relay/channel/xai relay/channel/task/xai relay/relay_adaptor.go web/src/components/table/channels/modals/EditChannelModal.jsx
git commit -m "feat: add xai account token pool mode"
```

package operation_setting

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	ChannelSecurityRuleMatchTypeRegex   = "regex"
	ChannelSecurityRuleMatchTypeKeyword = "keyword"
	ChannelSecurityRiskLevelHighRisk    = "high_risk"
)

type ChannelSecurityRule struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	MatchType string `json:"match_type"`
	Pattern   string `json:"pattern"`
	RiskLevel string `json:"risk_level"`
	Reason    string `json:"reason"`
}

var ChannelSecurityEnabled = true
var ChannelSecurityRules = DefaultChannelSecurityRules()
var ChannelSecurityRulesJSONString = defaultChannelSecurityRulesJSONString()

func DefaultChannelSecurityRules() []ChannelSecurityRule {
	return []ChannelSecurityRule{
		{ID: "html_script_tag", Name: "Script 标签注入", Enabled: true, MatchType: ChannelSecurityRuleMatchTypeRegex, Pattern: `(?i)<script[\s>]`, RiskLevel: ChannelSecurityRiskLevelHighRisk, Reason: "命中高危脚本注入片段"},
		{ID: "html_js_protocol", Name: "JavaScript 协议注入", Enabled: true, MatchType: ChannelSecurityRuleMatchTypeRegex, Pattern: `(?i)javascript:`, RiskLevel: ChannelSecurityRiskLevelHighRisk, Reason: "命中高危脚本协议片段"},
		{ID: "html_event_handler", Name: "HTML 事件注入", Enabled: true, MatchType: ChannelSecurityRuleMatchTypeRegex, Pattern: `(?i)on(load|error)\s*=`, RiskLevel: ChannelSecurityRiskLevelHighRisk, Reason: "命中高危 HTML 事件注入片段"},
		{ID: "html_iframe_svg", Name: "富文本注入", Enabled: true, MatchType: ChannelSecurityRuleMatchTypeRegex, Pattern: `(?i)<(iframe|svg)[\s>]`, RiskLevel: ChannelSecurityRiskLevelHighRisk, Reason: "命中高危富文本注入片段"},
		{ID: "prompt_injection_ignore_instructions", Name: "覆盖前置指令", Enabled: true, MatchType: ChannelSecurityRuleMatchTypeRegex, Pattern: `(?i)(ignore|忽略).{0,24}(previous|prior|earlier|之前|前面).{0,24}(instruction|prompt|指令|提示)`, RiskLevel: ChannelSecurityRiskLevelHighRisk, Reason: "命中 prompt injection 指令覆盖片段"},
		{ID: "prompt_injection_system_prompt", Name: "系统提示泄露", Enabled: true, MatchType: ChannelSecurityRuleMatchTypeRegex, Pattern: `(?i)(reveal|leak|show|expose|泄露|显示).{0,24}(system prompt|prompt|系统提示|提示词)`, RiskLevel: ChannelSecurityRiskLevelHighRisk, Reason: "命中系统提示泄露诱导片段"},
		{ID: "command_execution_keywords", Name: "命令执行诱导", Enabled: true, MatchType: ChannelSecurityRuleMatchTypeRegex, Pattern: `(?i)\b(curl|wget|powershell)\b|bash\s+-c|rm\s+-rf|chmod\s+\+x`, RiskLevel: ChannelSecurityRiskLevelHighRisk, Reason: "命中命令执行诱导片段"},
		{ID: "obfuscation_base64", Name: "Base64 混淆载荷", Enabled: true, MatchType: ChannelSecurityRuleMatchTypeRegex, Pattern: `(?i)data:[^;]{0,40};base64,|(?:[A-Za-z0-9+/]{120,}={0,2})`, RiskLevel: ChannelSecurityRiskLevelHighRisk, Reason: "命中可疑混淆载荷片段"},
		{ID: "obfuscation_hex_escape", Name: "十六进制转义混淆", Enabled: true, MatchType: ChannelSecurityRuleMatchTypeRegex, Pattern: `(?i)(?:\\x[0-9a-f]{2}){6,}`, RiskLevel: ChannelSecurityRiskLevelHighRisk, Reason: "命中可疑十六进制转义片段"},
	}
}

func defaultChannelSecurityRulesJSONString() string {
	bytes, err := common.Marshal(DefaultChannelSecurityRules())
	if err != nil {
		return "[]"
	}
	return string(bytes)
}

func UpdateChannelSecurityRulesFromString(value string) error {
	rules, normalized, err := NormalizeChannelSecurityRules(value)
	if err != nil {
		return err
	}
	ChannelSecurityRules = rules
	ChannelSecurityRulesJSONString = normalized
	return nil
}

func NormalizeChannelSecurityRules(value string) ([]ChannelSecurityRule, string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		defaultRules := DefaultChannelSecurityRules()
		return defaultRules, defaultChannelSecurityRulesJSONString(), nil
	}

	var rules []ChannelSecurityRule
	if err := common.Unmarshal([]byte(trimmed), &rules); err != nil {
		return nil, "", err
	}
	if len(rules) == 0 {
		return nil, "", fmt.Errorf("channel security rules cannot be empty")
	}

	normalizedRules := make([]ChannelSecurityRule, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))
	for i, rule := range rules {
		rule.ID = strings.TrimSpace(rule.ID)
		rule.Name = strings.TrimSpace(rule.Name)
		rule.MatchType = strings.ToLower(strings.TrimSpace(rule.MatchType))
		rule.Pattern = strings.TrimSpace(rule.Pattern)
		rule.RiskLevel = strings.ToLower(strings.TrimSpace(rule.RiskLevel))
		rule.Reason = strings.TrimSpace(rule.Reason)

		if rule.ID == "" {
			return nil, "", fmt.Errorf("channel security rule[%d] id is required", i)
		}
		if _, ok := seen[rule.ID]; ok {
			return nil, "", fmt.Errorf("duplicate channel security rule id: %s", rule.ID)
		}
		seen[rule.ID] = struct{}{}
		if rule.Name == "" {
			rule.Name = rule.ID
		}
		if rule.Pattern == "" {
			return nil, "", fmt.Errorf("channel security rule[%d] pattern is required", i)
		}
		if rule.Reason == "" {
			return nil, "", fmt.Errorf("channel security rule[%d] reason is required", i)
		}
		switch rule.MatchType {
		case ChannelSecurityRuleMatchTypeRegex:
			if _, err := regexp.Compile(rule.Pattern); err != nil {
				return nil, "", fmt.Errorf("channel security rule[%d] regex invalid: %w", i, err)
			}
		case ChannelSecurityRuleMatchTypeKeyword:
		default:
			return nil, "", fmt.Errorf("channel security rule[%d] match_type invalid: %s", i, rule.MatchType)
		}
		if rule.RiskLevel == "" {
			rule.RiskLevel = ChannelSecurityRiskLevelHighRisk
		}
		if rule.RiskLevel != ChannelSecurityRiskLevelHighRisk {
			return nil, "", fmt.Errorf("channel security rule[%d] risk_level invalid: %s", i, rule.RiskLevel)
		}
		normalizedRules = append(normalizedRules, rule)
	}

	normalizedBytes, err := common.Marshal(normalizedRules)
	if err != nil {
		return nil, "", err
	}
	return normalizedRules, string(normalizedBytes), nil
}

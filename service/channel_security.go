package service

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const (
	ChannelSecurityRiskLevelSafe     = "safe"
	ChannelSecurityRiskLevelHighRisk = "high_risk"

	channelSecuritySampleLimit = 1024
)

type ChannelSecurityScanResult struct {
	Safe      bool
	RiskLevel string
	Reason    string
	RuleIDs   []string
	Sample    string
}

type channelSecurityRule struct {
	ID     string
	Reason string
	Regex  *regexp.Regexp
}

func ScanChannelSecurity(respBody []byte) ChannelSecurityScanResult {
	if !operation_setting.ChannelSecurityEnabled {
		return ChannelSecurityScanResult{
			Safe:      true,
			RiskLevel: ChannelSecurityRiskLevelSafe,
		}
	}

	content := strings.TrimSpace(string(respBody))
	if content == "" {
		return ChannelSecurityScanResult{
			Safe:      true,
			RiskLevel: ChannelSecurityRiskLevelSafe,
		}
	}

	matchedRuleIDs := make([]string, 0, 2)
	reasonSet := make(map[string]struct{})
	firstSample := ""

	for _, rule := range buildChannelSecurityRules(operation_setting.ChannelSecurityRules) {
		matchIndexes := rule.Regex.FindStringIndex(content)
		if matchIndexes == nil {
			continue
		}
		matchedRuleIDs = append(matchedRuleIDs, rule.ID)
		reasonSet[rule.Reason] = struct{}{}
		if firstSample == "" {
			firstSample = buildSecuritySample(content, matchIndexes[0], matchIndexes[1])
		}
	}

	if len(matchedRuleIDs) == 0 {
		return ChannelSecurityScanResult{
			Safe:      true,
			RiskLevel: ChannelSecurityRiskLevelSafe,
		}
	}

	reasons := make([]string, 0, len(reasonSet))
	for reason := range reasonSet {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	sort.Strings(matchedRuleIDs)

	return ChannelSecurityScanResult{
		Safe:      false,
		RiskLevel: ChannelSecurityRiskLevelHighRisk,
		Reason:    strings.Join(reasons, "；"),
		RuleIDs:   matchedRuleIDs,
		Sample:    firstSample,
	}
}

func buildChannelSecurityRules(configs []operation_setting.ChannelSecurityRule) []channelSecurityRule {
	rules := make([]channelSecurityRule, 0, len(configs))
	for _, config := range configs {
		if !config.Enabled {
			continue
		}
		pattern := config.Pattern
		if config.MatchType == operation_setting.ChannelSecurityRuleMatchTypeKeyword {
			pattern = `(?i)` + regexp.QuoteMeta(config.Pattern)
		}
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			common.SysError(fmt.Sprintf("invalid channel security rule ignored: id=%s err=%v", config.ID, err))
			continue
		}
		rules = append(rules, channelSecurityRule{
			ID:     config.ID,
			Reason: config.Reason,
			Regex:  compiled,
		})
	}
	return rules
}

func buildSecuritySample(content string, start int, end int) string {
	if len(content) <= channelSecuritySampleLimit {
		return content
	}
	snippetStart := start - 160
	if snippetStart < 0 {
		snippetStart = 0
	}
	snippetEnd := end + 160
	if snippetEnd > len(content) {
		snippetEnd = len(content)
	}
	sample := content[snippetStart:snippetEnd]
	if len(sample) > channelSecuritySampleLimit {
		sample = sample[:channelSecuritySampleLimit]
	}
	return sample
}

func applySecurityDisableMetadata(channel *model.Channel, scan ChannelSecurityScanResult, source string, detectedAt int64) {
	if channel == nil {
		return
	}

	channel.Status = common.ChannelStatusSecurityDisabled
	info := channel.GetOtherInfo()
	info["status_reason"] = fmt.Sprintf("security: %s", scan.Reason)
	info["status_time"] = detectedAt
	info["security_reason"] = scan.Reason
	info["security_rule_ids"] = scan.RuleIDs
	info["security_detected_at"] = detectedAt
	info["security_sample"] = scan.Sample
	info["security_source"] = source
	channel.SetOtherInfo(info)

	if !channel.ChannelInfo.IsMultiKey {
		return
	}

	keys := channel.GetKeys()
	if len(keys) == 0 {
		return
	}
	if channel.ChannelInfo.MultiKeyStatusList == nil {
		channel.ChannelInfo.MultiKeyStatusList = make(map[int]int, len(keys))
	}
	if channel.ChannelInfo.MultiKeyDisabledReason == nil {
		channel.ChannelInfo.MultiKeyDisabledReason = make(map[int]string, len(keys))
	}
	if channel.ChannelInfo.MultiKeyDisabledTime == nil {
		channel.ChannelInfo.MultiKeyDisabledTime = make(map[int]int64, len(keys))
	}
	channel.ChannelInfo.MultiKeySize = len(keys)
	for i := range keys {
		channel.ChannelInfo.MultiKeyStatusList[i] = common.ChannelStatusSecurityDisabled
		channel.ChannelInfo.MultiKeyDisabledReason[i] = scan.Reason
		channel.ChannelInfo.MultiKeyDisabledTime[i] = detectedAt
	}
}

func SecurityDisableChannel(channel *model.Channel, scan ChannelSecurityScanResult, source string) {
	if channel == nil {
		return
	}

	detectedAt := common.GetTimestamp()
	common.SysError(fmt.Sprintf("通道「%s」（#%d）命中安全巡检规则，准备安全禁用，原因：%s，规则：%v", channel.Name, channel.Id, scan.Reason, scan.RuleIDs))

	applySecurityDisableMetadata(channel, scan, source, detectedAt)
	if err := channel.SaveWithoutKey(); err != nil {
		common.SysError(fmt.Sprintf("failed to persist security disabled channel: channel_id=%d, error=%v", channel.Id, err))
		return
	}

	model.CacheUpdateChannel(channel)
	if err := model.UpdateAbilityStatus(channel.Id, false); err != nil {
		common.SysError(fmt.Sprintf("failed to update ability status for security disabled channel: channel_id=%d, error=%v", channel.Id, err))
	}

	subject := fmt.Sprintf("通道「%s」（#%d）已被安全禁用", channel.Name, channel.Id)
	content := fmt.Sprintf("通道「%s」（#%d）在定时巡检中命中安全风险，原因：%s，规则：%s", channel.Name, channel.Id, scan.Reason, strings.Join(scan.RuleIDs, ", "))
	NotifyRootUser(formatNotifyType(channel.Id, common.ChannelStatusSecurityDisabled), subject, content)
}

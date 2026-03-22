/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState, useRef } from 'react';
import { Button, Col, Form, Row, Space, Spin } from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
  parseHttpStatusCodeRules,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';
import HttpStatusCodeRulesInput from '../../../components/settings/HttpStatusCodeRulesInput';

const DEFAULT_CHANNEL_SECURITY_RULES = JSON.stringify(
  [
    {
      id: 'html_script_tag',
      name: 'Script 标签注入',
      enabled: true,
      match_type: 'regex',
      pattern: '(?i)<script[\\s>]',
      risk_level: 'high_risk',
      reason: '命中高危脚本注入片段',
    },
    {
      id: 'html_js_protocol',
      name: 'JavaScript 协议注入',
      enabled: true,
      match_type: 'regex',
      pattern: '(?i)javascript:',
      risk_level: 'high_risk',
      reason: '命中高危脚本协议片段',
    },
    {
      id: 'html_event_handler',
      name: 'HTML 事件注入',
      enabled: true,
      match_type: 'regex',
      pattern: '(?i)on(load|error)\\s*=',
      risk_level: 'high_risk',
      reason: '命中高危 HTML 事件注入片段',
    },
    {
      id: 'html_iframe_svg',
      name: '富文本注入',
      enabled: true,
      match_type: 'regex',
      pattern: '(?i)<(iframe|svg)[\\s>]',
      risk_level: 'high_risk',
      reason: '命中高危富文本注入片段',
    },
    {
      id: 'prompt_injection_ignore_instructions',
      name: '覆盖前置指令',
      enabled: true,
      match_type: 'regex',
      pattern:
        '(?i)(ignore|忽略).{0,24}(previous|prior|earlier|之前|前面).{0,24}(instruction|prompt|指令|提示)',
      risk_level: 'high_risk',
      reason: '命中 prompt injection 指令覆盖片段',
    },
    {
      id: 'prompt_injection_system_prompt',
      name: '系统提示泄露',
      enabled: true,
      match_type: 'regex',
      pattern:
        '(?i)(reveal|leak|show|expose|泄露|显示).{0,24}(system prompt|prompt|系统提示|提示词)',
      risk_level: 'high_risk',
      reason: '命中系统提示泄露诱导片段',
    },
    {
      id: 'command_execution_keywords',
      name: '命令执行诱导',
      enabled: true,
      match_type: 'regex',
      pattern: '(?i)\\b(curl|wget|powershell)\\b|bash\\s+-c|rm\\s+-rf|chmod\\s+\\+x',
      risk_level: 'high_risk',
      reason: '命中命令执行诱导片段',
    },
    {
      id: 'obfuscation_base64',
      name: 'Base64 混淆载荷',
      enabled: true,
      match_type: 'regex',
      pattern: '(?i)data:[^;]{0,40};base64,|(?:[A-Za-z0-9+/]{120,}={0,2})',
      risk_level: 'high_risk',
      reason: '命中可疑混淆载荷片段',
    },
    {
      id: 'obfuscation_hex_escape',
      name: '十六进制转义混淆',
      enabled: true,
      match_type: 'regex',
      pattern: '(?i)(?:\\\\x[0-9a-f]{2}){6,}',
      risk_level: 'high_risk',
      reason: '命中可疑十六进制转义片段',
    },
  ],
  null,
  2,
);

export default function SettingsMonitoring(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    ChannelDisableThreshold: '',
    QuotaRemindThreshold: '',
    AutomaticDisableChannelEnabled: false,
    AutomaticEnableChannelEnabled: false,
    AutomaticDisableKeywords: '',
    AutomaticDisableStatusCodes: '401',
    AutomaticRetryStatusCodes:
      '100-199,300-399,401-407,409-499,500-503,505-523,525-599',
    ChannelSecurityEnabled: true,
    ChannelSecurityRules: DEFAULT_CHANNEL_SECURITY_RULES,
    'monitor_setting.auto_test_channel_enabled': false,
    'monitor_setting.auto_test_channel_minutes': 10,
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);
  const parsedAutoDisableStatusCodes = parseHttpStatusCodeRules(
    inputs.AutomaticDisableStatusCodes || '',
  );
  const parsedAutoRetryStatusCodes = parseHttpStatusCodeRules(
    inputs.AutomaticRetryStatusCodes || '',
  );
  let channelSecurityRulesError = '';
  try {
    const parsed = JSON.parse(inputs.ChannelSecurityRules || '[]');
    if (!Array.isArray(parsed)) {
      channelSecurityRulesError = t('安全巡检规则必须是 JSON 数组');
    }
  } catch (error) {
    channelSecurityRulesError = error.message;
  }

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    if (!parsedAutoDisableStatusCodes.ok) {
      const details =
        parsedAutoDisableStatusCodes.invalidTokens &&
        parsedAutoDisableStatusCodes.invalidTokens.length > 0
          ? `: ${parsedAutoDisableStatusCodes.invalidTokens.join(', ')}`
          : '';
      return showError(`${t('自动禁用状态码格式不正确')}${details}`);
    }
    if (!parsedAutoRetryStatusCodes.ok) {
      const details =
        parsedAutoRetryStatusCodes.invalidTokens &&
        parsedAutoRetryStatusCodes.invalidTokens.length > 0
          ? `: ${parsedAutoRetryStatusCodes.invalidTokens.join(', ')}`
          : '';
      return showError(`${t('自动重试状态码格式不正确')}${details}`);
    }
    if (channelSecurityRulesError) {
      return showError(`${t('安全巡检规则 JSON 格式不正确')}: ${channelSecurityRulesError}`);
    }
    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
        value = String(inputs[item.key]);
      } else {
        const normalizedMap = {
          AutomaticDisableStatusCodes: parsedAutoDisableStatusCodes.normalized,
          AutomaticRetryStatusCodes: parsedAutoRetryStatusCodes.normalized,
        };
        value = normalizedMap[item.key] ?? inputs[item.key];
      }
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (requestQueue.length === 1) {
          if (res.includes(undefined)) return;
        } else if (requestQueue.length > 1) {
          if (res.includes(undefined))
            return showError(t('部分保存失败，请重试'));
        }
        showSuccess(t('保存成功'));
        props.refresh();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current.setValues(currentInputs);
  }, [props.options]);

  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('监控设置')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'monitor_setting.auto_test_channel_enabled'}
                  label={t('定时测试所有通道')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      'monitor_setting.auto_test_channel_enabled': value,
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('自动测试所有通道间隔时间')}
                  step={1}
                  min={1}
                  suffix={t('分钟')}
                  extraText={t('每隔多少分钟测试一次所有通道')}
                  placeholder={''}
                  field={'monitor_setting.auto_test_channel_minutes'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      'monitor_setting.auto_test_channel_minutes':
                        parseInt(value),
                    })
                  }
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('测试所有渠道的最长响应时间')}
                  step={1}
                  min={0}
                  suffix={t('秒')}
                  extraText={t(
                    '当运行通道全部测试时，超过此时间将自动禁用通道',
                  )}
                  placeholder={''}
                  field={'ChannelDisableThreshold'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      ChannelDisableThreshold: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('额度提醒阈值')}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  extraText={t('低于此额度时将发送邮件提醒用户')}
                  placeholder={''}
                  field={'QuotaRemindThreshold'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaRemindThreshold: String(value),
                    })
                  }
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'AutomaticDisableChannelEnabled'}
                  label={t('失败时自动禁用通道')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) => {
                    setInputs({
                      ...inputs,
                      AutomaticDisableChannelEnabled: value,
                    });
                  }}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'ChannelSecurityEnabled'}
                  label={t('启用渠道安全巡检')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      ChannelSecurityEnabled: value,
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'AutomaticEnableChannelEnabled'}
                  label={t('成功时自动启用通道')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      AutomaticEnableChannelEnabled: value,
                    })
                  }
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={16}>
                <HttpStatusCodeRulesInput
                  label={t('自动禁用状态码')}
                  placeholder={t('例如：401, 403, 429, 500-599')}
                  extraText={t(
                    '支持填写单个状态码或范围（含首尾），使用逗号分隔',
                  )}
                  field={'AutomaticDisableStatusCodes'}
                  onChange={(value) =>
                    setInputs({ ...inputs, AutomaticDisableStatusCodes: value })
                  }
                  parsed={parsedAutoDisableStatusCodes}
                  invalidText={t('自动禁用状态码格式不正确')}
                />
                <HttpStatusCodeRulesInput
                  label={t('自动重试状态码')}
                  placeholder={t('例如：401, 403, 429, 500-599')}
                  extraText={t(
                    '支持填写单个状态码或范围（含首尾），使用逗号分隔；504 和 524 始终不重试，不受此处配置影响',
                  )}
                  field={'AutomaticRetryStatusCodes'}
                  onChange={(value) =>
                    setInputs({ ...inputs, AutomaticRetryStatusCodes: value })
                  }
                  parsed={parsedAutoRetryStatusCodes}
                  invalidText={t('自动重试状态码格式不正确')}
                />
                <Form.TextArea
                  label={t('自动禁用关键词')}
                  placeholder={t('一行一个，不区分大小写')}
                  extraText={t(
                    '当上游通道返回错误中包含这些关键词时（不区分大小写），自动禁用通道',
                  )}
                  field={'AutomaticDisableKeywords'}
                  autosize={{ minRows: 6, maxRows: 12 }}
                  onChange={(value) =>
                    setInputs({ ...inputs, AutomaticDisableKeywords: value })
                  }
                />
                <Form.TextArea
                  label={t('安全巡检规则')}
                  placeholder={t('请填写 JSON 数组格式的全局安全巡检规则')}
                  extraText={t(
                    '全局生效；只影响返回内容扫描，不会对巡检请求额外拼接任何提示词。支持字段：id、name、enabled、match_type(keyword/regex)、pattern、risk_level、reason',
                  )}
                  field={'ChannelSecurityRules'}
                  autosize={{ minRows: 10, maxRows: 20 }}
                  validateStatus={channelSecurityRulesError ? 'error' : 'default'}
                  helpText={
                    channelSecurityRulesError
                      ? `${t('安全巡检规则 JSON 格式不正确')}: ${channelSecurityRulesError}`
                      : ''
                  }
                  onChange={(value) =>
                    setInputs({ ...inputs, ChannelSecurityRules: value })
                  }
                />
                <Space>
                  <Button
                    type='tertiary'
                    size='small'
                    onClick={() =>
                      setInputs({
                        ...inputs,
                        ChannelSecurityRules: DEFAULT_CHANNEL_SECURITY_RULES,
                      })
                    }
                  >
                    {t('恢复默认安全规则')}
                  </Button>
                  <Button
                    type='tertiary'
                    size='small'
                    onClick={() => {
                      try {
                        const formatted = JSON.stringify(
                          JSON.parse(inputs.ChannelSecurityRules || '[]'),
                          null,
                          2,
                        );
                        setInputs({ ...inputs, ChannelSecurityRules: formatted });
                      } catch (error) {
                        showError(
                          `${t('安全巡检规则 JSON 格式不正确')}: ${error.message}`,
                        );
                      }
                    }}
                  >
                    {t('格式化安全规则 JSON')}
                  </Button>
                </Space>
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存监控设置')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}

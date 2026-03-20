export const MODEL_MAPPING_EXAMPLE = {
  'gpt-3.5-turbo': 'gpt-3.5-turbo-0125',
  'gpt-5.4-openai-compact': {
    target_model: 'gpt-5.4',
    endpoint_type: 'openai-response',
  },
};

export const MODEL_MAPPING_ENDPOINT_TYPES = [
  'openai',
  'openai-response',
  'openai-response-compact',
  'anthropic',
  'gemini',
  'embeddings',
  'image-generation',
  'jina-rerank',
];

export function normalizeModelMappingConfig(modelMapping) {
  if (
    !modelMapping ||
    typeof modelMapping !== 'object' ||
    Array.isArray(modelMapping)
  ) {
    return modelMapping;
  }

  const normalized = {};
  for (const [sourceModel, rule] of Object.entries(modelMapping)) {
    if (typeof rule === 'string') {
      normalized[sourceModel] = rule;
      continue;
    }

    if (!rule || typeof rule !== 'object' || Array.isArray(rule)) {
      normalized[sourceModel] = rule;
      continue;
    }

    const targetModel = String(rule.target_model || '').trim();
    const endpointType = String(rule.endpoint_type || '').trim();

    normalized[sourceModel] = endpointType
      ? { target_model: targetModel, endpoint_type: endpointType }
      : { target_model: targetModel };
  }

  return normalized;
}

export function validateModelMappingConfig(modelMapping) {
  if (
    !modelMapping ||
    typeof modelMapping !== 'object' ||
    Array.isArray(modelMapping)
  ) {
    return '模型重定向必须是 JSON 对象';
  }

  for (const [sourceModel, rule] of Object.entries(modelMapping)) {
    const normalizedSourceModel = String(sourceModel || '').trim();
    if (!normalizedSourceModel) {
      return '模型重定向的键不能为空';
    }

    if (typeof rule === 'string') {
      if (!rule.trim()) {
        return `模型 ${normalizedSourceModel} 的重定向目标不能为空`;
      }
      continue;
    }

    if (!rule || typeof rule !== 'object' || Array.isArray(rule)) {
      return `模型 ${normalizedSourceModel} 的重定向值必须是字符串或对象`;
    }

    const targetModel = String(rule.target_model || '').trim();
    if (!targetModel) {
      return `模型 ${normalizedSourceModel} 的 target_model 不能为空`;
    }

    if (rule.endpoint_type !== undefined && rule.endpoint_type !== null) {
      const endpointType = String(rule.endpoint_type).trim();
      if (
        endpointType &&
        !MODEL_MAPPING_ENDPOINT_TYPES.includes(endpointType)
      ) {
        return `模型 ${normalizedSourceModel} 的 endpoint_type 不合法，可选值：${MODEL_MAPPING_ENDPOINT_TYPES.join(', ')}`;
      }
    }
  }

  return '';
}

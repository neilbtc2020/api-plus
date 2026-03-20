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

import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

export const useModelAvailabilityData = () => {
  const { t } = useTranslation();
  const [groups, setGroups] = useState([]);
  const [items, setItems] = useState([]);
  const [selectedGroup, setSelectedGroup] = useState('');
  const [warning, setWarning] = useState('');
  const [refreshedAt, setRefreshedAt] = useState(0);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [probingModel, setProbingModel] = useState('');
  const [keyword, setKeyword] = useState('');
  const [onlyFailed, setOnlyFailed] = useState(false);
  const [onlyWithLogs, setOnlyWithLogs] = useState(false);

  const applySnapshot = useCallback((payload = {}) => {
    setGroups(Array.isArray(payload.groups) ? payload.groups : []);
    setItems(Array.isArray(payload.items) ? payload.items : []);
    setSelectedGroup(payload.selected_group || '');
    setWarning(payload.warning || '');
    setRefreshedAt(payload.refreshed_at || 0);
  }, []);

  const load = useCallback(
    async (group = '') => {
      setLoading(true);
      try {
        const res = await API.get('/api/model-availability', {
          params: {
            group,
          },
        });
        const { success, message, data } = res.data || {};
        if (!success) {
          showError(message || t('加载模型可用性失败'));
          return;
        }
        applySnapshot(data || {});
      } catch (error) {
        showError(error);
      } finally {
        setLoading(false);
      }
    },
    [applySnapshot, t],
  );

  useEffect(() => {
    load();
  }, [load]);

  const changeGroup = useCallback(
    async (group) => {
      await load(group);
    },
    [load],
  );

  const refresh = useCallback(async () => {
    if (!selectedGroup) {
      return;
    }
    setRefreshing(true);
    try {
      const res = await API.post('/api/model-availability/refresh', {
        group: selectedGroup,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('刷新失败'));
        return;
      }
      applySnapshot(data || {});
      showSuccess(t('刷新成功'));
    } catch (error) {
      showError(error);
    } finally {
      setRefreshing(false);
    }
  }, [applySnapshot, selectedGroup, t]);

  const probeModel = useCallback(
    async (modelName) => {
      if (!selectedGroup || !modelName) {
        return;
      }
      setProbingModel(modelName);
      try {
        const res = await API.post('/api/model-availability/probe', {
          group: selectedGroup,
          model_name: modelName,
        });
        const { success, message, data } = res.data || {};
        if (!success) {
          showError(message || t('检测失败'));
          return;
        }
        setItems((currentItems) =>
          currentItems.map((item) =>
            item.model_name === modelName ? { ...item, probe: data } : item,
          ),
        );
        showSuccess(data?.message || t('检测完成'));
      } catch (error) {
        showError(error);
      } finally {
        setProbingModel('');
      }
    },
    [selectedGroup, t],
  );

  const clearFilters = useCallback(() => {
    setKeyword('');
    setOnlyFailed(false);
    setOnlyWithLogs(false);
  }, []);

  return {
    groups,
    items,
    selectedGroup,
    warning,
    refreshedAt,
    loading,
    refreshing,
    probingModel,
    keyword,
    onlyFailed,
    onlyWithLogs,
    setKeyword,
    setOnlyFailed,
    setOnlyWithLogs,
    clearFilters,
    load,
    changeGroup,
    refresh,
    probeModel,
  };
};

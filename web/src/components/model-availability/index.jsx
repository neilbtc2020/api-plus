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

import React, { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import Loading from '../common/ui/Loading';
import ModelAvailabilityLayout from './ModelAvailabilityLayout';
import GroupList from './GroupList';
import ModelAvailabilityFilters from './ModelAvailabilityFilters';
import ModelAvailabilityTable from './ModelAvailabilityTable';
import { shouldShowAvailabilityItem } from './modelAvailability.utils';
import { useModelAvailabilityData } from '../../hooks/model-availability/useModelAvailabilityData';

const ModelAvailability = () => {
  const { t } = useTranslation();
  const {
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
    changeGroup,
    refresh,
    probeModel,
  } = useModelAvailabilityData();

  const filteredItems = useMemo(
    () =>
      items.filter((item) =>
        shouldShowAvailabilityItem(item, {
          keyword,
          onlyFailed,
          onlyWithLogs,
        }),
      ),
    [items, keyword, onlyFailed, onlyWithLogs],
  );

  if (loading) {
    return <Loading />;
  }

  return (
    <ModelAvailabilityLayout
      title={t('模型可用性')}
      description={t('查看每个分组下各模型最近 20 次请求的成功与失败情况')}
      warning={warning}
      refreshedAt={refreshedAt}
      sidebar={
        <GroupList
          groups={groups}
          selectedGroup={selectedGroup}
          onSelect={changeGroup}
        />
      }
      filters={
        <ModelAvailabilityFilters
          keyword={keyword}
          onlyFailed={onlyFailed}
          onlyWithLogs={onlyWithLogs}
          refreshing={refreshing}
          onKeywordChange={setKeyword}
          onOnlyFailedChange={setOnlyFailed}
          onOnlyWithLogsChange={setOnlyWithLogs}
          onRefresh={refresh}
          onClear={clearFilters}
        />
      }
      table={
        <ModelAvailabilityTable
          items={filteredItems}
          probingModel={probingModel}
          onProbe={probeModel}
        />
      }
    />
  );
};

export default ModelAvailability;

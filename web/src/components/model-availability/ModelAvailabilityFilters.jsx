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

import React from 'react';
import { useTranslation } from 'react-i18next';

const ModelAvailabilityFilters = ({
  keyword,
  onlyFailed,
  onlyWithLogs,
  refreshing,
  onKeywordChange,
  onOnlyFailedChange,
  onOnlyWithLogsChange,
  onRefresh,
  onClear,
}) => {
  const { t } = useTranslation();

  return (
    <div className='space-y-3'>
      <div className='grid grid-cols-1 gap-3 lg:grid-cols-[minmax(0,1fr)_auto_auto]'>
        <input
          className='w-full rounded-xl border border-semi-color-border bg-white px-3 py-2 text-sm outline-none'
          placeholder={t('搜索模型名称')}
          value={keyword}
          onChange={(event) => onKeywordChange(event.target.value)}
        />
        <button
          type='button'
          className='rounded-xl bg-semi-color-fill-0 px-4 py-2 text-sm hover:bg-semi-color-fill-1'
          onClick={onClear}
        >
          {t('清空筛选')}
        </button>
        <button
          type='button'
          className='rounded-xl bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60'
          onClick={onRefresh}
          disabled={refreshing}
        >
          {refreshing ? t('刷新中...') : t('刷新快照')}
        </button>
      </div>

      <div className='flex flex-wrap gap-4 text-sm'>
        <label className='flex items-center gap-2'>
          <input
            type='checkbox'
            checked={onlyFailed}
            onChange={(event) => onOnlyFailedChange(event.target.checked)}
          />
          <span>{t('仅看失败模型')}</span>
        </label>
        <label className='flex items-center gap-2'>
          <input
            type='checkbox'
            checked={onlyWithLogs}
            onChange={(event) => onOnlyWithLogsChange(event.target.checked)}
          />
          <span>{t('仅看有真实日志')}</span>
        </label>
      </div>
    </div>
  );
};

export default ModelAvailabilityFilters;

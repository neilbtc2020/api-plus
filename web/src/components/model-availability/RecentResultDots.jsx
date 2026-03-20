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
import { getRecentResultSummary } from './modelAvailability.utils';

const dotClassName = (result) => {
  if (result?.status === 'fail') {
    return 'bg-red-500';
  }
  if (result?.defaulted) {
    return 'bg-green-200';
  }
  return 'bg-green-500';
};

const RecentResultDots = ({ results = [] }) => {
  const { t } = useTranslation();
  const summary = getRecentResultSummary(results);

  return (
    <div className='space-y-2'>
      <div className='flex flex-wrap gap-1'>
        {results.map((result, index) => (
          <span
            key={`${result?.status || 'success'}-${index}`}
            className={`inline-block h-2.5 w-2.5 rounded-full ${dotClassName(result)}`}
            title={result?.defaulted ? t('默认补绿') : result?.status}
          />
        ))}
      </div>
      <div className='text-xs text-semi-color-text-2'>
        {summary.hasOnlyDefaulted
          ? t('当前仅显示默认补绿窗口')
          : `${t('真实结果')} ${summary.realCount} / ${t('失败')} ${summary.failCount}`}
      </div>
    </div>
  );
};

export default RecentResultDots;

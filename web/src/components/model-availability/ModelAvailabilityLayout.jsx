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

const formatRefreshedAt = (refreshedAt) => {
  if (!refreshedAt) {
    return '-';
  }
  return new Date(refreshedAt * 1000).toLocaleString();
};

const ModelAvailabilityLayout = ({
  title,
  description,
  warning,
  refreshedAt,
  sidebar,
  filters,
  table,
}) => {
  return (
    <div className='space-y-4'>
      <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-0 p-5'>
        <div className='text-2xl font-semibold'>{title}</div>
        <div className='mt-2 text-sm text-semi-color-text-1'>{description}</div>
        <div className='mt-3 text-xs text-semi-color-text-2'>
          最近刷新：{formatRefreshedAt(refreshedAt)}
        </div>
        {warning ? (
          <div className='mt-3 rounded-xl border border-yellow-200 bg-yellow-50 px-3 py-2 text-sm text-yellow-700'>
            {warning}
          </div>
        ) : null}
      </div>

      <div className='grid grid-cols-1 gap-4 lg:grid-cols-[240px_minmax(0,1fr)]'>
        <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-0 p-4'>
          {sidebar}
        </div>
        <div className='space-y-4'>
          <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-0 p-4'>
            {filters}
          </div>
          <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-0 p-4'>
            {table}
          </div>
        </div>
      </div>
    </div>
  );
};

export default ModelAvailabilityLayout;

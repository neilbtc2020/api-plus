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

const GroupList = ({ groups = [], selectedGroup, onSelect }) => {
  const { t } = useTranslation();

  if (groups.length === 0) {
    return (
      <div className='text-sm text-semi-color-text-2'>
        {t('当前没有可见分组')}
      </div>
    );
  }

  return (
    <div>
      <div className='mb-3 text-sm font-medium text-semi-color-text-0'>
        {t('分组')}
      </div>
      <div className='space-y-2'>
        {groups.map((group) => {
          const active = group.name === selectedGroup;
          return (
            <button
              key={group.name}
              type='button'
              className={`w-full rounded-xl border px-3 py-2 text-left text-sm transition ${
                active
                  ? 'border-blue-500 bg-blue-50 text-blue-700'
                  : 'border-semi-color-border bg-semi-color-fill-0 hover:bg-semi-color-fill-1'
              }`}
              onClick={() => onSelect(group.name)}
            >
              {group.name}
            </button>
          );
        })}
      </div>
    </div>
  );
};

export default GroupList;

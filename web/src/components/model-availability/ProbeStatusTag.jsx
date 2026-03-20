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

const ProbeStatusTag = ({ probe }) => {
  const { t } = useTranslation();

  if (!probe) {
    return <span className='text-xs text-semi-color-text-2'>{t('未检测')}</span>;
  }

  const success = probe.status === 'success';

  return (
    <div className='space-y-1'>
      <span
        className={`inline-flex rounded-full px-2 py-1 text-xs font-medium ${
          success
            ? 'bg-green-100 text-green-700'
            : 'bg-red-100 text-red-700'
        }`}
      >
        {success ? t('成功') : t('失败')}
      </span>
      {probe.message ? (
        <div className='max-w-[220px] text-xs text-semi-color-text-2'>
          {probe.message}
        </div>
      ) : null}
      {probe.checked_at ? (
        <div className='text-xs text-semi-color-text-2'>
          {new Date(probe.checked_at * 1000).toLocaleString()}
        </div>
      ) : null}
    </div>
  );
};

export default ProbeStatusTag;

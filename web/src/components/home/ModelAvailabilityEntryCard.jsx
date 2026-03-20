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
import { Link } from 'react-router-dom';

export default function ModelAvailabilityEntryCard() {
  const { t } = useTranslation();

  return (
    <Link to='/model-availability'>
      <div className='rounded-2xl border border-semi-color-border p-5 bg-semi-color-bg-0 hover:shadow-md transition-shadow'>
        <div className='text-lg font-semibold'>{t('模型可用性')}</div>
        <div className='text-semi-color-text-1 mt-2 text-sm'>
          {t('查看每个分组下各模型最近 20 次请求的成功与失败情况')}
        </div>
      </div>
    </Link>
  );
}

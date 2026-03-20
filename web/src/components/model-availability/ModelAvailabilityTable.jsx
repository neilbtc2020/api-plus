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
import RecentResultDots from './RecentResultDots';
import ProbeStatusTag from './ProbeStatusTag';

const StatusBadge = ({ active, activeLabel, inactiveLabel }) => (
  <span
    className={`inline-flex rounded-full px-2 py-1 text-xs font-medium ${
      active
        ? 'bg-green-100 text-green-700'
        : 'bg-red-100 text-red-700'
    }`}
  >
    {active ? activeLabel : inactiveLabel}
  </span>
);

const ModelAvailabilityTable = ({ items = [], probingModel, onProbe }) => {
  const { t } = useTranslation();

  if (items.length === 0) {
    return (
      <div className='py-10 text-center text-sm text-semi-color-text-2'>
        {t('当前筛选条件下没有可展示的模型')}
      </div>
    );
  }

  return (
    <div className='overflow-x-auto'>
      <table className='min-w-full text-sm'>
        <thead>
          <tr className='border-b border-semi-color-border text-left text-semi-color-text-2'>
            <th className='pb-3 pr-4'>{t('模型')}</th>
            <th className='pb-3 pr-4'>{t('配置状态')}</th>
            <th className='pb-3 pr-4'>{t('最近结果')}</th>
            <th className='pb-3 pr-4'>{t('统计')}</th>
            <th className='pb-3 pr-4'>{t('Probe')}</th>
            <th className='pb-3'>{t('操作')}</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr
              key={item.model_name}
              className='border-b border-semi-color-border/70 align-top'
            >
              <td className='py-4 pr-4 font-medium'>{item.model_name}</td>
              <td className='py-4 pr-4'>
                <StatusBadge
                  active={item.config_available}
                  activeLabel={t('可用')}
                  inactiveLabel={t('不可用')}
                />
              </td>
              <td className='py-4 pr-4'>
                <RecentResultDots results={item.recent_results || []} />
              </td>
              <td className='py-4 pr-4 text-semi-color-text-1'>
                <div>{t('成功')}: {item.success_count || 0}</div>
                <div>{t('失败')}: {item.fail_count || 0}</div>
                <div>{item.has_real_logs ? t('有真实日志') : t('仅默认补绿')}</div>
              </td>
              <td className='py-4 pr-4'>
                <ProbeStatusTag probe={item.probe} />
              </td>
              <td className='py-4'>
                <button
                  type='button'
                  className='rounded-xl bg-semi-color-fill-0 px-3 py-2 text-sm hover:bg-semi-color-fill-1 disabled:cursor-not-allowed disabled:opacity-60'
                  onClick={() => onProbe(item.model_name)}
                  disabled={probingModel === item.model_name}
                >
                  {probingModel === item.model_name ? t('检测中...') : t('重新检测')}
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default ModelAvailabilityTable;

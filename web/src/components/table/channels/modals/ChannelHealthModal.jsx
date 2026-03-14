import React, { useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Input,
  Modal,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import {
  renderQuotaWithAmount,
  timestamp2string,
} from '../../../../helpers';
import { CHANNEL_OPTIONS } from '../../../../constants';

const getTypeMeta = (type, t) => {
  const match = CHANNEL_OPTIONS.find((item) => item.value === type);
  return match || { label: t('未知类型'), color: 'grey' };
};

const renderStatus = (status, t) => {
  switch (status) {
    case 1:
      return <Tag color='green'>{t('已启用')}</Tag>;
    case 2:
      return <Tag color='red'>{t('已禁用')}</Tag>;
    case 3:
      return <Tag color='yellow'>{t('自动禁用')}</Tag>;
    default:
      return <Tag color='grey'>{t('未知状态')}</Tag>;
  }
};

const renderAvailability = (record, t) => {
  if (record.available) {
    return <Tag color='green'>{t('可用')}</Tag>;
  }
  return (
    <Space spacing={4} vertical align='start'>
      <Tag color='red'>{t('不可用')}</Tag>
      {record.available_message ? (
        <Typography.Text type='tertiary' size='small'>
          {record.available_message}
        </Typography.Text>
      ) : null}
    </Space>
  );
};

const renderBalance = (record, t) => {
  if (!record.balance_supported) {
    return (
      <Space spacing={4} vertical align='start'>
        <Tag color='grey'>{t('不支持')}</Tag>
        {record.balance_message ? (
          <Typography.Text type='tertiary' size='small'>
            {record.balance_message}
          </Typography.Text>
        ) : null}
      </Space>
    );
  }
  if (record.balance === null || record.balance === undefined) {
    return (
      <Space spacing={4} vertical align='start'>
        <Tag color='red'>{t('查询失败')}</Tag>
        {record.balance_message ? (
          <Typography.Text type='tertiary' size='small'>
            {record.balance_message}
          </Typography.Text>
        ) : null}
      </Space>
    );
  }
  return <Tag color='white'>{renderQuotaWithAmount(record.balance)}</Tag>;
};

const renderResponseTime = (record, t) => {
  if (!record.response_time_ms) {
    return <Tag color='grey'>{t('未检测')}</Tag>;
  }
  const seconds = Number(record.response_time || 0).toFixed(2);
  const color =
    record.response_time_ms <= 1000
      ? 'green'
      : record.response_time_ms <= 3000
        ? 'lime'
        : record.response_time_ms <= 5000
          ? 'yellow'
          : 'red';
  return <Tag color={color}>{seconds + t(' 秒')}</Tag>;
};

const SummaryCard = ({ title, value, color }) => (
  <div
    style={{
      minWidth: 120,
      padding: 12,
      borderRadius: 12,
      border: '1px solid var(--semi-color-border)',
      background: 'var(--semi-color-bg-0)',
    }}
  >
    <Typography.Text type='tertiary'>{title}</Typography.Text>
    <div style={{ marginTop: 6 }}>
      <Typography.Text
        strong
        style={{ fontSize: 20, color: color || 'var(--semi-color-text-0)' }}
      >
        {value}
      </Typography.Text>
    </div>
  </div>
);

const ChannelHealthModal = ({
  visible,
  loading,
  items,
  summary,
  onCancel,
  onRefresh,
  t,
}) => {
  const [keyword, setKeyword] = useState('');

  const filteredItems = useMemo(() => {
    const lowerKeyword = keyword.trim().toLowerCase();
    if (!lowerKeyword) {
      return items || [];
    }
    return (items || []).filter((item) => {
      return (
        String(item.id).includes(lowerKeyword) ||
        (item.name || '').toLowerCase().includes(lowerKeyword) ||
        (item.type_name || '').toLowerCase().includes(lowerKeyword) ||
        (item.group || '').toLowerCase().includes(lowerKeyword) ||
        (item.tag || '').toLowerCase().includes(lowerKeyword)
      );
    });
  }, [items, keyword]);

  const columns = useMemo(
    () => [
      {
        title: t('ID'),
        dataIndex: 'id',
        width: 80,
      },
      {
        title: t('名称'),
        dataIndex: 'name',
        render: (text, record) => (
          <Space spacing={4} vertical align='start'>
            <Typography.Text strong>{text}</Typography.Text>
            {record.is_multi_key ? (
              <Tag color='blue'>{t('多密钥')}</Tag>
            ) : null}
          </Space>
        ),
      },
      {
        title: t('类型'),
        dataIndex: 'type',
        render: (type) => {
          const meta = getTypeMeta(type, t);
          return <Tag color={meta.color}>{meta.label}</Tag>;
        },
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        render: (status) => renderStatus(status, t),
      },
      {
        title: t('可行性'),
        dataIndex: 'available',
        render: (_, record) => renderAvailability(record, t),
      },
      {
        title: t('响应时间'),
        dataIndex: 'response_time_ms',
        render: (_, record) => renderResponseTime(record, t),
      },
      {
        title: t('可用额度'),
        dataIndex: 'balance',
        render: (_, record) => renderBalance(record, t),
      },
      {
        title: t('检测时间'),
        dataIndex: 'checked_at',
        render: (value) =>
          value ? timestamp2string(value) : <Tag color='grey'>{t('暂无')}</Tag>,
      },
    ],
    [t],
  );

  return (
    <Modal
      title={t('渠道健康检查')}
      visible={visible}
      onCancel={onCancel}
      width={1100}
      footer={
        <Space>
          <Button type='tertiary' onClick={onCancel}>
            {t('关闭')}
          </Button>
          <Button type='primary' loading={loading} onClick={onRefresh}>
            {t('重新检查')}
          </Button>
        </Space>
      }
    >
      <div className='flex flex-col gap-3'>
        <Banner
          type='info'
          closeIcon={null}
          description={t(
            '这里会实时检查渠道可用性，并尝试读取支持额度查询的渠道余额。',
          )}
        />

        <div className='flex flex-wrap gap-3'>
          <SummaryCard
            title={t('总渠道')}
            value={summary?.total || 0}
            color='var(--semi-color-text-0)'
          />
          <SummaryCard
            title={t('可用')}
            value={summary?.available_count || 0}
            color='var(--semi-color-success)'
          />
          <SummaryCard
            title={t('不可用')}
            value={summary?.unavailable_count || 0}
            color='var(--semi-color-danger)'
          />
          <SummaryCard
            title={t('支持额度')}
            value={summary?.balance_supported_count || 0}
            color='var(--semi-color-primary)'
          />
          <SummaryCard
            title={t('额度失败')}
            value={summary?.balance_failed_count || 0}
            color='var(--semi-color-warning)'
          />
        </div>

        <Input
          prefix={<IconSearch />}
          placeholder={t('搜索渠道名称、ID、类型、分组、标签')}
          value={keyword}
          onChange={setKeyword}
          showClear
        />

        <Table
          rowKey='id'
          loading={loading}
          columns={columns}
          dataSource={filteredItems}
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            pageSizeOpts: [10, 20, 50, 100],
          }}
          scroll={{ x: 'max-content' }}
        />
      </div>
    </Modal>
  );
};

export default ChannelHealthModal;

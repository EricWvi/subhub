import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Modal, Form, Input, Select, InputNumber, message, Popconfirm, Typography, Card, Descriptions, Tag } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { formatDate24h } from '../utils';

const { Title, Text } = Typography;

interface ProxyMember {
  type: string;
  value: string;
}

interface ProxyGroup {
  id: number;
  name: string;
  type: string;
  url: string;
  interval: number;
  proxies: ProxyMember[];
  bind_internal_proxy_group_id: number;
  is_system: boolean;
}

interface ClashConfigSubscription {
  id: number;
  name: string;
  providers: number[];
  proxy_groups: ProxyGroup[];
  created_at: string;
  updated_at: string;
}

interface Provider {
  id: number;
  name: string;
}

interface InternalGroup {
  id: number;
  name: string;
}

const ClashConfigSubscriptionManager: React.FC = () => {
  const [subscriptions, setSubscriptions] = useState<ClashConfigSubscription[]>([]);
  const [loading, setLoading] = useState(false);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [internalGroups, setInternalGroups] = useState<InternalGroup[]>([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingSub, setEditingSub] = useState<ClashConfigSubscription | null>(null);
  const [form] = Form.useForm();

  const fetchSubscriptions = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/subscriptions/clash-configs');
      const data = await response.json();
      setSubscriptions(data.subscriptions || []);
    } catch (error) {
      message.error('Failed to fetch subscriptions');
    } finally {
      setLoading(false);
    }
  };

  const fetchProviders = async () => {
    try {
      const response = await fetch('/api/providers');
      const data = await response.json();
      setProviders(data.providers || []);
    } catch (error) { /* ignore */ }
  };

  const fetchInternalGroups = async () => {
    try {
      const response = await fetch('/api/proxy-groups');
      const data = await response.json();
      setInternalGroups((data.groups || []).map((g: any) => ({ id: g.id, name: g.name })));
    } catch (error) { /* ignore */ }
  };

  useEffect(() => {
    fetchSubscriptions();
    fetchProviders();
    fetchInternalGroups();
  }, []);

  const handleAdd = () => {
    setEditingSub(null);
    form.resetFields();
    form.setFieldsValue({ name: '', providers: [], proxy_groups: [] });
    setModalVisible(true);
  };

  const handleEdit = (record: ClashConfigSubscription) => {
    setEditingSub(record);
    form.setFieldsValue({
      name: record.name,
      providers: record.providers,
      proxy_groups: record.proxy_groups.filter(pg => !pg.is_system).map(pg => ({
        name: pg.name,
        type: pg.type,
        url: pg.url,
        interval: pg.interval,
        bind_internal_proxy_group_id: pg.bind_internal_proxy_group_id,
        proxies: pg.proxies.map(p => ({ type: p.type, value: p.value })),
      })),
    });
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await fetch(`/api/subscriptions/clash-configs/${id}`, { method: 'DELETE' });
      message.success('Subscription deleted');
      fetchSubscriptions();
    } catch (error) {
      message.error('Failed to delete subscription');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const url = editingSub ? `/api/subscriptions/clash-configs/${editingSub.id}` : '/api/subscriptions/clash-configs';
      const method = editingSub ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(values),
      });

      if (response.ok) {
        message.success(`Subscription ${editingSub ? 'updated' : 'added'}`);
        setModalVisible(false);
        fetchSubscriptions();
      } else {
        const errorText = await response.text();
        message.error(`Operation failed: ${errorText}`);
      }
    } catch (error) { /* form validation */ }
  };

  const providerMap = Object.fromEntries(providers.map(p => [p.id, p.name]));
  const groupMap = Object.fromEntries(internalGroups.map(g => [g.id, g.name]));

  return (
    <div>
      <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={3}>Clash Config Subscriptions</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>Add Subscription</Button>
      </div>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={subscriptions}
        columns={[
          { title: 'Name', dataIndex: 'name', key: 'name' },
          {
            title: 'Providers',
            key: 'providers',
            render: (_: any, record: ClashConfigSubscription) => (
              <span>{record.providers.map(id => providerMap[id] || id).join(' → ')}</span>
            ),
          },
          { title: 'Updated At', dataIndex: 'updated_at', key: 'updated_at', render: (text: string) => formatDate24h(text) },
          {
            title: 'Action',
            key: 'action',
            render: (_: any, record: ClashConfigSubscription) => (
              <Space>
                <Button icon={<EditOutlined />} onClick={() => handleEdit(record)} />
                <Popconfirm title="Sure to delete?" onConfirm={() => handleDelete(record.id)}>
                  <Button danger icon={<DeleteOutlined />} />
                </Popconfirm>
              </Space>
            ),
          },
        ]}
        expandable={{
          expandedRowRender: (record) => (
            <div style={{ padding: '8px 40px' }}>
              <Title level={5}>Proxy Groups</Title>
              {record.proxy_groups.map(pg => (
                <Card key={pg.id} size="small" title={pg.name} style={{ marginBottom: 8 }}
                  extra={pg.is_system ? <Tag color="blue">System</Tag> : null}>
                  <Descriptions column={2} size="small">
                    <Descriptions.Item label="Type">{pg.type}</Descriptions.Item>
                    <Descriptions.Item label="Interval">{pg.interval || '-'}</Descriptions.Item>
                    {pg.bind_internal_proxy_group_id ? (
                      <Descriptions.Item label="Bound Internal Group">
                        {groupMap[pg.bind_internal_proxy_group_id] || pg.bind_internal_proxy_group_id}
                      </Descriptions.Item>
                    ) : null}
                    <Descriptions.Item label="Members">
                      {pg.proxies.map(m => (
                        <Tag key={`${m.type}:${m.value}`}>
                          {m.type === 'internal' ? (groupMap[Number(m.value)] || m.value) : m.value}
                        </Tag>
                      ))}
                    </Descriptions.Item>
                  </Descriptions>
                </Card>
              ))}
            </div>
          ),
        }}
      />

      <Modal
        title={editingSub ? 'Edit Clash Config Subscription' : 'Add Clash Config Subscription'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        width={720}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Name is required' }]}>
            <Input placeholder="Daily Config" />
          </Form.Item>
          <Form.Item name="providers" label="Providers" rules={[{ required: true, message: 'At least one provider is required' }]}>
            <Select
              mode="multiple"
              placeholder="Select providers"
              options={providers.map(p => ({ label: p.name, value: p.id }))}
            />
          </Form.Item>
          <Form.List name="proxy_groups">
            {(fields, { add, remove }) => (
              <>
                <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between' }}>
                  <Text strong>Proxy Groups</Text>
                  <Button size="small" onClick={() => add({ name: '', type: 'select', url: '', interval: 0, proxies: [], bind_internal_proxy_group_id: undefined })}>
                    Add Proxy Group
                  </Button>
                </div>
                {fields.map(field => (
                  <Card key={field.key} size="small" style={{ marginBottom: 8 }}
                    extra={<Button danger size="small" onClick={() => remove(field.name)}>Remove</Button>}>
                    <Form.Item name={[field.name, 'name']} label="Name" rules={[{ required: true }]}>
                      <Input placeholder="Media" />
                    </Form.Item>
                    <Form.Item name={[field.name, 'type']} label="Type" rules={[{ required: true }]}>
                      <Select options={[
                        { label: 'select', value: 'select' },
                        { label: 'url-test', value: 'url-test' },
                        { label: 'fallback', value: 'fallback' },
                      ]} />
                    </Form.Item>
                    <Form.Item name={[field.name, 'url']} label="URL">
                      <Input placeholder="https://cp.cloudflare.com/generate_204" />
                    </Form.Item>
                    <Form.Item name={[field.name, 'interval']} label="Interval (seconds)">
                      <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={[field.name, 'bind_internal_proxy_group_id']} label="Bound Internal Group">
                      <Select allowClear placeholder="Select internal group"
                        options={internalGroups.map(g => ({ label: g.name, value: g.id }))} />
                    </Form.Item>
                    <Form.List name={[field.name, 'proxies']}>
                      {(memberFields, { add: addMember, remove: removeMember }) => (
                        <>
                          <div style={{ marginBottom: 4 }}>
                            <Text strong>Members</Text>
                            <Button size="small" style={{ marginLeft: 8 }} onClick={() => addMember({ type: 'DIRECT', value: 'DIRECT' })}>
                              Add Member
                            </Button>
                          </div>
                          {memberFields.map(mf => (
                            <Space key={mf.key} style={{ display: 'flex', marginBottom: 4 }}>
                              <Form.Item name={[mf.name, 'type']} noStyle>
                                <Select style={{ width: 120 }} options={[
                                  { label: 'Internal', value: 'internal' },
                                  { label: 'Reference', value: 'reference' },
                                  { label: 'DIRECT', value: 'DIRECT' },
                                  { label: 'REJECT', value: 'REJECT' },
                                ]} />
                              </Form.Item>
                              <Form.Item name={[mf.name, 'value']} noStyle>
                                <Input placeholder="value" style={{ width: 200 }} />
                              </Form.Item>
                              <Button danger size="small" onClick={() => removeMember(mf.name)}>×</Button>
                            </Space>
                          ))}
                        </>
                      )}
                    </Form.List>
                  </Card>
                ))}
              </>
            )}
          </Form.List>
        </Form>
      </Modal>
    </div>
  );
};

export default ClashConfigSubscriptionManager;

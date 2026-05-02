import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Modal, Form, Input, Select, message, Popconfirm, Typography } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { formatDate24h } from '../utils';

const { Title } = Typography;

interface RuleProviderSubscription {
  id: number;
  name: string;
  providers: number[];
  internal_proxy_group_id: number;
  created_at: string;
  updated_at: string;
}

interface Provider { id: number; name: string; }
interface InternalGroup { id: number; name: string; }

const RuleProviderSubscriptionManager: React.FC = () => {
  const [subscriptions, setSubscriptions] = useState<RuleProviderSubscription[]>([]);
  const [loading, setLoading] = useState(false);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [internalGroups, setInternalGroups] = useState<InternalGroup[]>([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingSub, setEditingSub] = useState<RuleProviderSubscription | null>(null);
  const [form] = Form.useForm();

  const fetchData = async () => {
    setLoading(true);
    try {
      const [subRes, provRes, groupRes] = await Promise.all([
        fetch('/api/subscriptions/rule-providers'),
        fetch('/api/providers'),
        fetch('/api/proxy-groups'),
      ]);
      const subData = await subRes.json();
      const provData = await provRes.json();
      const groupData = await groupRes.json();
      setSubscriptions(subData.subscriptions || []);
      setProviders(provData.providers || []);
      setInternalGroups((groupData.groups || []).map((g: any) => ({ id: g.id, name: g.name })));
    } catch (error) {
      message.error('Failed to fetch data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchData(); }, []);

  const handleAdd = () => {
    setEditingSub(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: RuleProviderSubscription) => {
    setEditingSub(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await fetch(`/api/subscriptions/rule-providers/${id}`, { method: 'DELETE' });
      message.success('Subscription deleted');
      fetchData();
    } catch (error) {
      message.error('Failed to delete subscription');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const url = editingSub ? `/api/subscriptions/rule-providers/${editingSub.id}` : '/api/subscriptions/rule-providers';
      const method = editingSub ? 'PUT' : 'POST';
      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(values),
      });
      if (response.ok) {
        message.success(`Subscription ${editingSub ? 'updated' : 'added'}`);
        setModalVisible(false);
        fetchData();
      } else {
        message.error(`Failed: ${await response.text()}`);
      }
    } catch (error) { /* form validation */ }
  };

  const providerMap = Object.fromEntries(providers.map(p => [p.id, p.name]));
  const groupMap = Object.fromEntries(internalGroups.map(g => [g.id, g.name]));

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={3}>Rule Provider Subscriptions</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>Add</Button>
      </div>
      <Table
        rowKey="id" loading={loading} dataSource={subscriptions}
        columns={[
          { title: 'Name', dataIndex: 'name', key: 'name' },
          { title: 'Providers', key: 'providers', render: (_: any, r: RuleProviderSubscription) => r.providers.map(id => providerMap[id] || id).join(' → ') },
          { title: 'Internal Group', key: 'group', render: (_: any, r: RuleProviderSubscription) => groupMap[r.internal_proxy_group_id] || r.internal_proxy_group_id },
          { title: 'Updated', dataIndex: 'updated_at', key: 'updated_at', render: (t: string) => formatDate24h(t) },
          {
            title: 'Action', key: 'action',
            render: (_: any, r: RuleProviderSubscription) => (
              <Space>
                <Button icon={<EditOutlined />} onClick={() => handleEdit(r)} />
                <Popconfirm title="Sure to delete?" onConfirm={() => handleDelete(r.id)}>
                  <Button danger icon={<DeleteOutlined />} />
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />
      <Modal title={editingSub ? 'Edit' : 'Add'} open={modalVisible} onOk={handleModalOk} onCancel={() => setModalVisible(false)} destroyOnClose>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input placeholder="Rules Export" />
          </Form.Item>
          <Form.Item name="providers" label="Providers" rules={[{ required: true }]}>
            <Select mode="multiple" options={providers.map(p => ({ label: p.name, value: p.id }))} />
          </Form.Item>
          <Form.Item name="internal_proxy_group_id" label="Internal Proxy Group" rules={[{ required: true }]}>
            <Select options={internalGroups.map(g => ({ label: g.name, value: g.id }))} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default RuleProviderSubscriptionManager;

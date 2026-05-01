import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Modal, Form, Input, InputNumber, message, Popconfirm, Tag, Drawer, Typography } from 'antd';
import { PlusOutlined, ReloadOutlined, EditOutlined, DeleteOutlined, EyeOutlined } from '@ant-design/icons';
import { formatDate24h } from '../utils';

const { Title, Text } = Typography;

interface Provider {
  id: number;
  name: string;
  url: string;
  refresh_interval_minutes: number;
  created_at: string;
  updated_at: string;
  last_refresh_status?: string;
  last_refresh_message?: string;
}

interface Snapshot {
  id: number;
  provider_id: number;
  format: string;
  normalized_yaml: string;
  node_count: number;
  fetched_at: string;
}

const ProviderManager: React.FC = () => {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingProvider, setEditingProvider] = useState<Provider | null>(null);
  const [form] = Form.useForm();
  
  const [snapshotDrawerVisible, setSnapshotDrawerVisible] = useState(false);
  const [currentSnapshot, setCurrentSnapshot] = useState<Snapshot | null>(null);
  const [snapshotLoading, setSnapshotLoading] = useState(false);

  const fetchProviders = async () => {
    setLoading(true);
    try {
      const response = await fetch('/providers');
      const data = await response.json();
      setProviders(data.providers || []);
    } catch (error) {
      message.error('Failed to fetch providers');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProviders();
  }, []);

  const handleAdd = () => {
    setEditingProvider(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: Provider) => {
    setEditingProvider(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await fetch(`/providers/${id}`, { method: 'DELETE' });
      message.success('Provider deleted');
      fetchProviders();
    } catch (error) {
      message.error('Failed to delete provider');
    }
  };

  const handleRefresh = async (id: number) => {
    try {
      const response = await fetch(`/providers/${id}/refresh`, { method: 'POST' });
      if (response.ok) {
        message.success('Refresh triggered successfully');
        fetchProviders();
      } else {
        const errorText = await response.text();
        message.error(`Refresh failed: ${errorText}`);
      }
    } catch (error) {
      message.error('Failed to trigger refresh');
    }
  };

  const handleViewSnapshot = async (id: number) => {
    setSnapshotLoading(true);
    setSnapshotDrawerVisible(true);
    try {
      const response = await fetch(`/providers/${id}/snapshot`);
      if (response.ok) {
        const data = await response.json();
        setCurrentSnapshot(data.snapshot);
      } else {
        setCurrentSnapshot(null);
        if (response.status === 404) {
          message.info('No snapshot available for this provider');
        } else {
          message.error('Failed to fetch snapshot');
        }
      }
    } catch (error) {
      message.error('Failed to fetch snapshot');
    } finally {
      setSnapshotLoading(false);
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const url = editingProvider ? `/providers/${editingProvider.id}` : '/providers';
      const method = editingProvider ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(values),
      });

      if (response.ok) {
        message.success(`Provider ${editingProvider ? 'updated' : 'added'}`);
        setModalVisible(false);
        fetchProviders();
      } else {
        const errorText = await response.text();
        message.error(`Operation failed: ${errorText}`);
      }
    } catch (error) {
      // Form validation error
    }
  };

  const columns = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'URL',
      dataIndex: 'url',
      key: 'url',
      ellipsis: true,
      render: (text: string) => <a href={text} target="_blank" rel="noopener noreferrer">{text}</a>
    },
    {
      title: 'Interval (min)',
      dataIndex: 'refresh_interval_minutes',
      key: 'refresh_interval_minutes',
    },
    {
      title: 'Status',
      key: 'status',
      render: (_: any, record: Provider) => {
        if (!record.last_refresh_status) return <Tag>Never</Tag>;
        return record.last_refresh_status === 'success' ? (
          <Tag color="success">OK</Tag>
        ) : (
          <Tag color="error" title={record.last_refresh_message}>Failed</Tag>
        );
      },
    },
    {
      title: 'Updated At',
      dataIndex: 'updated_at',
      key: 'updated_at',
      render: (text: string) => formatDate24h(text),
    },
    {
      title: 'Action',
      key: 'action',
      render: (_: any, record: Provider) => (
        <Space size="middle">
          <Button 
            type="primary" 
            ghost 
            icon={<ReloadOutlined />} 
            onClick={() => handleRefresh(record.id)}
            title="Refresh Now"
          />
          <Button 
            icon={<EyeOutlined />} 
            onClick={() => handleViewSnapshot(record.id)}
            title="View Snapshot"
          />
          <Button 
            icon={<EditOutlined />} 
            onClick={() => handleEdit(record)}
          />
          <Popconfirm title="Sure to delete?" onConfirm={() => handleDelete(record.id)}>
            <Button danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2}>Provider Management</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          Add Provider
        </Button>
      </div>

      <Table 
        columns={columns} 
        dataSource={providers} 
        rowKey="id" 
        loading={loading} 
      />

      <Modal
        title={editingProvider ? 'Edit Provider' : 'Add Provider'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="Name"
            rules={[{ required: true, message: 'Please input provider name!' }]}
          >
            <Input placeholder="My Airport" />
          </Form.Item>
          <Form.Item
            name="url"
            label="URL"
            rules={[{ required: true, message: 'Please input provider URL!' }, { type: 'url', message: 'Please enter a valid URL!' }]}
          >
            <Input placeholder="https://example.com/sub?target=clash" />
          </Form.Item>
          <Form.Item
            name="refresh_interval_minutes"
            label="Refresh Interval (minutes)"
            initialValue={120}
            rules={[{ required: true, message: 'Please input refresh interval!' }]}
          >
            <InputNumber min={5} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title="Provider Snapshot"
        placement="right"
        width={800}
        onClose={() => setSnapshotDrawerVisible(false)}
        open={snapshotDrawerVisible}
        loading={snapshotLoading}
      >
        {currentSnapshot ? (
          <div>
            <Space direction="vertical" style={{ width: '100%' }} size="large">
              <div>
                <Text type="secondary">Fetched At: </Text>
                <Text strong>{formatDate24h(currentSnapshot.fetched_at)}</Text>
              </div>
              <div>
                <Text type="secondary">Format: </Text>
                <Tag color="blue">{currentSnapshot.format}</Tag>
                <Text type="secondary" style={{ marginLeft: '16px' }}>Node Count: </Text>
                <Text strong>{currentSnapshot.node_count}</Text>
              </div>
              
              <div>
                <Title level={5}>Normalized YAML</Title>
                <pre style={{ 
                  background: '#f5f5f5', 
                  padding: '12px', 
                  borderRadius: '4px', 
                  maxHeight: '600px', 
                  overflow: 'auto',
                  fontSize: '12px'
                }}>
                  {currentSnapshot.normalized_yaml}
                </pre>
              </div>
            </Space>
          </div>
        ) : (
          <div style={{ textAlign: 'center', marginTop: '40px' }}>
            <Text type="secondary">No snapshot data available.</Text>
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default ProviderManager;

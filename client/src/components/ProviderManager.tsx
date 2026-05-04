import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Modal, Form, Input, InputNumber, message, Popconfirm, Tag, Drawer, Typography, Progress, Switch } from 'antd';
import { PlusOutlined, ReloadOutlined, EditOutlined, DeleteOutlined, EyeOutlined } from '@ant-design/icons';
import Editor from '@monaco-editor/react';
import { formatDate24h, formatBytes, useMonacoTheme } from '../utils';

const { Title, Text } = Typography;

interface Provider {
  id: number;
  name: string;
  url: string;
  refresh_interval_minutes: number;
  abbrev: string;
  used: number;
  total: number;
  expire: number;
  created_at: string;
  updated_at: string;
  last_refresh_status?: string;
  last_refresh_message?: string;
}

interface ProxyNode {
  id: number;
  provider_id: number;
  name: string;
  raw_yaml: string;
  enabled: boolean;
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
  const monacoTheme = useMonacoTheme();
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(false);
  const [providerNodes, setProviderNodes] = useState<Record<number, ProxyNode[]>>({});
  const [nodesLoading, setNodesLoading] = useState<Record<number, boolean>>({});

  const fetchNodes = async (providerId: number) => {
    if (providerNodes[providerId]) return;
    setNodesLoading(prev => ({ ...prev, [providerId]: true }));
    try {
      const response = await fetch(`/api/providers/${providerId}/nodes`);
      if (response.ok) {
        const data = await response.json();
        setProviderNodes(prev => ({ ...prev, [providerId]: data.nodes || [] }));
      }
    } catch (error) {
      message.error('Failed to fetch nodes');
    } finally {
      setNodesLoading(prev => ({ ...prev, [providerId]: false }));
    }
  };

  const [modalVisible, setModalVisible] = useState(false);
  const [editingProvider, setEditingProvider] = useState<Provider | null>(null);
  const [form] = Form.useForm();
  
  const [snapshotDrawerVisible, setSnapshotDrawerVisible] = useState(false);
  const [currentSnapshot, setCurrentSnapshot] = useState<Snapshot | null>(null);
  const [snapshotLoading, setSnapshotLoading] = useState(false);

  const fetchProviders = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/providers');
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
      const response = await fetch(`/api/providers/${id}`, { method: 'DELETE' });
      if (response.ok) {
        message.success('Provider deleted');
        fetchProviders();
      } else {
        const errorText = await response.text();
        message.error(`Failed to delete: ${errorText}`);
      }
    } catch (error) {
      message.error('Failed to delete provider');
    }
  };

  const handleRefresh = async (id: number) => {
    try {
      const response = await fetch(`/api/providers/${id}/refresh`, { method: 'POST' });
      if (response.ok) {
        message.success('Refresh triggered successfully');
        setProviderNodes(prev => {
          const newState = { ...prev };
          delete newState[id];
          return newState;
        });
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
      const response = await fetch(`/api/providers/${id}/snapshot`);
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

  const handleToggleNode = async (providerId: number, nodeId: number) => {
    try {
      const response = await fetch(`/api/providers/${providerId}/nodes/toggle/${nodeId}`, { method: 'POST' });
      if (response.ok) {
        const data = await response.json();
        setProviderNodes(prev => {
          const nodes = prev[providerId];
          if (!nodes) return prev;
          return {
            ...prev,
            [providerId]: nodes.map(n => n.id === nodeId ? { ...n, enabled: data.enabled } : n),
          };
        });
      } else {
        message.error('Failed to toggle node');
      }
    } catch {
      message.error('Failed to toggle node');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const url = editingProvider ? `/api/providers/${editingProvider.id}` : '/api/providers';
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
      key: 'name',
      render: (_: any, record: Provider) => (
        <Space>
          {record.name}
          {record.abbrev && <Tag color="blue">{record.abbrev}</Tag>}
        </Space>
      ),
    },
    {
      title: 'Usage',
      key: 'usage',
      width: 200,
      render: (_: any, record: Provider) => {
        if (!record.total || record.total === 0) return <Text type="secondary">N/A</Text>;
        const percent = Math.min(100, Math.round((record.used / record.total) * 100));
        const expiryDate = record.expire > 0 ? formatDate24h(new Date(record.expire * 1000).toISOString()) : 'Never';
        return (
          <div style={{ width: '100%' }}>
            <Progress percent={percent} size="small" status={percent > 90 ? 'exception' : 'active'} />
            <div style={{ fontSize: '11px', display: 'flex', justifyContent: 'space-between', marginTop: '4px' }}>
              <span>{formatBytes(record.used)} / {formatBytes(record.total)}</span>
            </div>
            <div style={{ fontSize: '11px', color: '#8c8c8c' }}>
              Exp: {expiryDate}
            </div>
          </div>
        );
      }
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
        expandable={{
          expandedRowRender: (record) => (
            <div style={{ padding: '8px 40px' }}>
              <Title level={5}>Proxy Nodes ({providerNodes[record.id]?.length || 0})</Title>
              <Table
                size="small"
                columns={[
                  { title: 'Node Name', dataIndex: 'name', key: 'name' },
                  {
                    title: 'Enabled',
                    key: 'enabled',
                    width: 80,
                    render: (_, node: ProxyNode) => (
                      <Switch
                        size="small"
                        checked={node.enabled}
                        onChange={() => handleToggleNode(record.id, node.id)}
                      />
                    ),
                  },
                  {
                    title: 'Configuration',
                    key: 'raw',
                    render: (_, node: ProxyNode) => (
                      <pre style={{ margin: 0, fontSize: '12px', maxHeight: '120px', overflow: 'auto' }}>
                        {node.raw_yaml}
                      </pre>
                    ),
                  },
                ]}
                dataSource={providerNodes[record.id] || []}
                rowKey="id"
                pagination={false}
                loading={nodesLoading[record.id]}
              />
            </div>
          ),
          onExpand: (expanded, record) => {
            if (expanded) {
              fetchNodes(record.id);
            }
          }
        }}
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
            name="abbrev"
            label="Abbreviation"
            tooltip="Uppercase letters only (e.g. 'XYZ')"
            getValueFromEvent={(e) => e.target.value.toUpperCase().replace(/[^A-Z]/g, '')}
          >
            <Input placeholder="AIRPORT" />
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
        styles={{ body: { display: 'flex', flexDirection: 'column', overflow: 'hidden', flex: 1 } }}
      >
        {currentSnapshot ? (
          <div style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
            <div style={{ marginBottom: 16, flexShrink: 0 }}>
              <Space direction="vertical" style={{ width: '100%' }} size="small">
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
              </Space>
            </div>
            <Title level={5} style={{ flexShrink: 0 }}>Normalized YAML</Title>
            <div style={{ flex: 1, minHeight: 0 }}>
              <Editor
                height="100%"
                language="yaml"
                value={currentSnapshot.normalized_yaml}
                theme={monacoTheme}
                options={{
                  readOnly: true,
                  minimap: { enabled: false },
                  scrollBeyondLastLine: false,
                  fontSize: 14,
                  scrollbar: { verticalScrollbarSize: 8, horizontalScrollbarSize: 8 },
                }}
              />
            </div>
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

import React, { useCallback, useEffect, useState } from 'react';
import { Table, Button, Space, Modal, Form, Input, Checkbox, App, Popconfirm, Typography, Drawer } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, CopyOutlined, EyeOutlined } from '@ant-design/icons';
import Editor from '@monaco-editor/react';
import { formatDate24h, useMonacoTheme } from '../utils';
import { useApiClient } from '../api';

const { Title, Text } = Typography;

interface NodeProviderSubscription {
  id: number;
  name: string;
  nodes: number[];
  created_at: string;
  updated_at: string;
}

interface Node { id: number; name: string; }

const NodeProviderSubscriptionManager: React.FC = () => {
  const { message } = App.useApp();
  const apiClient = useApiClient();
  const monacoTheme = useMonacoTheme();
  const [subscriptions, setSubscriptions] = useState<NodeProviderSubscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [saving, setSaving] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingSub, setEditingSub] = useState<NodeProviderSubscription | null>(null);
  const [form] = Form.useForm();
  const [previewDrawerVisible, setPreviewDrawerVisible] = useState(false);
  const [previewContent, setPreviewContent] = useState<string>("");
  const [previewLoading, setPreviewLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [subRes, nodeRes] = await Promise.all([
        apiClient('/api/subscriptions/node-providers'),
        apiClient('/api/nodes'),
      ]);
      if (!subRes.ok || !nodeRes.ok) throw new Error('Failed to fetch data');
      const subData = await subRes.json();
      const nodeData = await nodeRes.json();
      setSubscriptions(subData.subscriptions || []);
      setNodes(nodeData.nodes || []);
    } catch {
      message.error('Failed to fetch data');
    } finally {
      setLoading(false);
    }
  }, [apiClient, message]);

  useEffect(() => {
    const timer = window.setTimeout(() => { void fetchData(); }, 0);
    return () => window.clearTimeout(timer);
  }, [fetchData]);

  const closeModal = () => {
    setModalVisible(false);
    setEditingSub(null);
    form.resetFields();
  };

  const handleAdd = () => {
    setEditingSub(null);
    form.resetFields();
    form.setFieldsValue({ name: '', nodes: [] });
    setModalVisible(true);
  };

  const handleEdit = (record: NodeProviderSubscription) => {
    setEditingSub(record);
    form.resetFields();
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      const response = await apiClient(`/api/subscriptions/node-providers/${id}`, { method: 'DELETE' });
      if (response.ok) {
        message.success('Subscription deleted');
        fetchData();
      } else {
        const errorText = await response.text();
        message.error(`Failed to delete: ${errorText}`);
      }
    } catch {
      message.error('Failed to delete subscription');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields().catch(() => null);
      if (!values) return;
      setSaving(true);
      const url = editingSub ? `/api/subscriptions/node-providers/${editingSub.id}` : '/api/subscriptions/node-providers';
      const method = editingSub ? 'PUT' : 'POST';
      const response = await apiClient(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(values),
      });
      if (response.ok) {
        message.success(`Subscription ${editingSub ? 'updated' : 'added'}`);
        closeModal();
        fetchData();
      } else {
        message.error(`Failed: ${await response.text()}`);
      }
    } catch {
      message.error('Failed to save subscription');
    } finally {
      setSaving(false);
    }
  };

  const handleCopy = async (id: number) => {
    const url = `${window.location.origin}/api/subscriptions/node-providers/${id}/content`;
    try {
      await navigator.clipboard.writeText(url);
      message.success('Subscription URL copied to clipboard');
    } catch {
      message.error('Failed to copy subscription URL');
    }
  };

  const handlePreview = async (id: number) => {
    setPreviewLoading(true);
    setPreviewDrawerVisible(true);
    try {
      const response = await apiClient(`/api/subscriptions/node-providers/${id}/content`);
      if (response.ok) {
        const text = await response.text();
        setPreviewContent(text);
      } else {
        setPreviewContent("");
        message.error("Failed to fetch content");
      }
    } catch {
      setPreviewContent("");
      message.error("Failed to fetch content");
    } finally {
      setPreviewLoading(false);
    }
  };

  const nodeMap = Object.fromEntries(nodes.map(n => [n.id, n.name]));

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={3}>Node Provider Subscriptions</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>Add</Button>
      </div>
      <Table
        rowKey="id" loading={loading} dataSource={subscriptions}
        pagination={{ pageSize: 20, showSizeChanger: false }}
        columns={[
          { title: 'Name', dataIndex: 'name', key: 'name' },
          { title: 'Nodes', key: 'nodes', render: (_: unknown, r: NodeProviderSubscription) => r.nodes.map(id => nodeMap[id] || id).join(', ') },
          { title: 'Updated', dataIndex: 'updated_at', key: 'updated_at', render: (t: string) => formatDate24h(t) },
          {
            title: 'Action', key: 'action',
            render: (_: unknown, r: NodeProviderSubscription) => (
              <Space>
                <Button icon={<CopyOutlined />} onClick={() => handleCopy(r.id)} title="Copy Link" />
                <Button icon={<EyeOutlined />} onClick={() => handlePreview(r.id)} title="Preview" />
                <Button icon={<EditOutlined />} onClick={() => handleEdit(r)} title="Edit" />
                <Popconfirm title="Sure to delete?" onConfirm={() => handleDelete(r.id)}>
                  <Button danger icon={<DeleteOutlined />} title="Delete" />
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />
      <Modal key={editingSub?.id || "add"} title={editingSub ? 'Edit' : 'Add'} open={modalVisible} confirmLoading={saving} onOk={handleModalOk} onCancel={closeModal} >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item name="name" label="Name" rules={[{ required: true, whitespace: true }]}>
            <Input placeholder="HK Export" />
          </Form.Item>
          <Form.Item name="nodes" label="Nodes" rules={[{ type: 'array', required: true, min: 1, message: 'Select at least one node' }]}>
            <Checkbox.Group
              options={nodes.map(n => ({ label: n.name, value: n.id }))}
              style={{ display: 'flex', flexDirection: 'column', gap: 8, maxHeight: 320, overflowY: 'auto' }}
            />
          </Form.Item>
          {nodes.length === 0 && <Text type="secondary">No nodes available. Add nodes in Nodes first.</Text>}
        </Form>
      </Modal>
      <Drawer
        title="Content Preview"
        placement="right"
        size="large"
        onClose={() => setPreviewDrawerVisible(false)}
        open={previewDrawerVisible}
        loading={previewLoading}
        styles={{ body: { display: 'flex', flexDirection: 'column', overflow: 'hidden', flex: 1 } }}
      >
        {previewContent ? (
          <div style={{ flex: 1, minHeight: 0 }}>
            <Editor
              height="100%"
              language="yaml"
              value={previewContent}
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
        ) : (
          <div style={{ textAlign: "center", marginTop: "40px" }}>
            <Text type="secondary">No content available.</Text>
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default NodeProviderSubscriptionManager;

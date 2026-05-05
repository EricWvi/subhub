import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Modal, Form, Input, App, Popconfirm, Typography, Tag } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import MonacoEditor from '@monaco-editor/react';
import { formatDate24h, useMonacoTheme } from '../utils';

const { Title } = Typography;

interface ProxyGroup {
  id: number;
  name: string;
  script: string;
  created_at: string;
  updated_at: string;
}

interface ProxyNodeView {
  id: number;
  providerName: string;
  name: string;
}

const MonacoEditorWrapper: React.FC = () => {
  const form = Form.useFormInstance();
  const value = Form.useWatch('script', form) ?? '';
  const monacoTheme = useMonacoTheme();

  return (
    <MonacoEditor
      height="300px"
      language="javascript"
      value={value}
      theme={monacoTheme}
      onChange={(val) => form.setFieldValue('script', val ?? '')}
      options={{
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        fontSize: 14,
        scrollbar: { verticalScrollbarSize: 8, horizontalScrollbarSize: 8 },
      }}
    />
  );
};

const ProxyGroupManager: React.FC = () => {
  const { message } = App.useApp();
  const monacoTheme = useMonacoTheme();
  const [groups, setGroups] = useState<ProxyGroup[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingGroup, setEditingGroup] = useState<ProxyGroup | null>(null);
  const [expandedRowKeys, setExpandedRowKeys] = useState<readonly React.Key[]>([]);
  const [form] = Form.useForm();

  const [groupNodes, setGroupNodes] = useState<Record<number, ProxyNodeView[]>>({});
  const [nodesLoading, setNodesLoading] = useState<Record<number, boolean>>({});

  const fetchGroups = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/proxy-groups');
      const data = await response.json();
      setGroups(data.groups || []);
    } catch (error) {
      message.error('Failed to fetch proxy groups');
    } finally {
      setLoading(false);
    }
  };

  const fetchGroupNodes = async (groupId: number, force = false) => {
    if (!force && groupNodes[groupId]) return;
    setNodesLoading(prev => ({ ...prev, [groupId]: true }));
    try {
      const response = await fetch(`/api/proxy-groups/${groupId}/nodes`);
      if (response.ok) {
        const data = await response.json();
        setGroupNodes(prev => ({ ...prev, [groupId]: data.nodes || [] }));
      }
    } catch (error) {
      message.error('Failed to fetch group nodes');
    } finally {
      setNodesLoading(prev => ({ ...prev, [groupId]: false }));
    }
  };

  useEffect(() => {
    fetchGroups();
  }, []);

  const handleAdd = () => {
    setEditingGroup(null);
    setModalVisible(true);
  };

  const handleEdit = (record: ProxyGroup) => {
    setEditingGroup(record);
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      const response = await fetch(`/api/proxy-groups/${id}`, { method: 'DELETE' });
      if (response.ok) {
        message.success('Proxy group deleted');
        fetchGroups();
      } else {
        const errorText = await response.text();
        message.error(`Failed to delete: ${errorText}`);
      }
    } catch (error) {
      message.error('Failed to delete proxy group');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const url = editingGroup ? `/api/proxy-groups/${editingGroup.id}` : '/api/proxy-groups';
      const method = editingGroup ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(values),
      });

      if (response.ok) {
        message.success(`Proxy group ${editingGroup ? 'updated' : 'added'}`);
        if (editingGroup) {
          const id = editingGroup.id;
          // Clear cache and re-fetch if it's currently expanded
          if (expandedRowKeys.includes(id)) {
            fetchGroupNodes(id, true);
          } else {
            setGroupNodes(prev => {
              const newState = { ...prev };
              delete newState[id];
              return newState;
            });
          }
        }
        setModalVisible(false);
        fetchGroups();
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
      title: 'Script',
      dataIndex: 'script',
      key: 'script',
      render: (text: string) => text ? <Tag color="green">Configured</Tag> : <Tag>None</Tag>
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
      render: (_: any, record: ProxyGroup) => (
        <Space size="middle">
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
    <div>
      <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2}>Proxy Group Management</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          Add Group
        </Button>
      </div>

      <Table 
        columns={columns} 
        dataSource={groups} 
        rowKey="id" 
        loading={loading}
        expandable={{
          expandedRowKeys,
          onExpandedRowsChange: (keys) => setExpandedRowKeys(keys),
          expandedRowRender: (record) => (
            <div style={{ padding: '8px 40px' }}>
              <Space orientation="vertical" style={{ width: '100%' }} size="large">
                {record.script && (
                  <div>
                    <Title level={5}>Script</Title>
                    <MonacoEditor
                      height="200px"
                      language="javascript"
                      value={record.script}
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
                )}
                <div>
                  <Title level={5}>Assigned Nodes ({groupNodes[record.id]?.length || 0})</Title>
                  <Table
                    size="small"
                    columns={[
                      { title: 'Provider', dataIndex: 'providerName', key: 'providerName' },
                      { title: 'Node Name', dataIndex: 'name', key: 'name' },
                    ]}
                    dataSource={groupNodes[record.id] || []}
                    rowKey="id"
                    pagination={false}
                    loading={nodesLoading[record.id]}
                  />
                </div>
              </Space>
            </div>
          ),
          onExpand: (expanded, record) => {
            if (expanded) {
              fetchGroupNodes(record.id);
            }
          }
        }}
      />

      <Modal key={editingGroup?.id || "add"}
        title={editingGroup ? 'Edit Proxy Group' : 'Add Proxy Group'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        
        width={800}
      >
        <Form form={form} layout="vertical" preserve={false} initialValues={editingGroup || {}}>
          <Form.Item
            name="name"
            label="Name"
            rules={[{ required: true, message: 'Please input group name!' }]}
          >
            <Input placeholder="Streaming" />
          </Form.Item>
          <Form.Item
            name="script"
            label="Script (Optional)"
            tooltip="User script to filter or transform nodes for this group."
            getValueFromEvent={() => undefined}
          >
            <MonacoEditorWrapper />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ProxyGroupManager;

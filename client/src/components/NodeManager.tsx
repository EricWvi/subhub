import React, { useCallback, useEffect, useState } from "react";
import { App, Button, Form, Input, Modal, Popconfirm, Space, Table, Tag, Typography } from "antd";
import { DeleteOutlined, EditOutlined, PlusOutlined } from "@ant-design/icons";
import Editor, { type OnMount } from "@monaco-editor/react";
import { useApiClient } from "../api";
import { claimKeyboardPriority } from "./keyboardPriority";
import { formatDate24h, useMonacoTheme } from "../utils";

const { Title } = Typography;

interface CustomNode {
  id: number;
  name: string;
  config: string;
  created_at: string;
  updated_at: string;
}

const ConfigEditor: React.FC = () => {
  const form = Form.useFormInstance();
  const value = Form.useWatch("config", form) ?? "";
  const monacoTheme = useMonacoTheme();

  const handleMount: OnMount = (editor) => {
    const release = claimKeyboardPriority(editor.getContainerDomNode());
    editor.onDidDispose(release);
  };

  return (
    <Editor
      height="360px"
      language="yaml"
      value={value}
      theme={monacoTheme}
      onMount={handleMount}
      onChange={(nextValue) => form.setFieldValue("config", nextValue ?? "")}
      options={{
        editContext: false,
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        fontSize: 14,
        scrollbar: { verticalScrollbarSize: 8, horizontalScrollbarSize: 8 },
      }}
    />
  );
};

const NodeManager: React.FC = () => {
  const { message } = App.useApp();
  const apiClient = useApiClient();
  const [nodes, setNodes] = useState<CustomNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingNode, setEditingNode] = useState<CustomNode | null>(null);
  const [form] = Form.useForm();

  const fetchNodes = useCallback(async () => {
    setLoading(true);
    try {
      const response = await apiClient("/api/nodes");
      const data = await response.json();
      setNodes(data.nodes || []);
    } catch {
      message.error("Failed to fetch nodes");
    } finally {
      setLoading(false);
    }
  }, [apiClient, message]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void fetchNodes();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [fetchNodes]);

  const closeModal = () => {
    setModalVisible(false);
    setEditingNode(null);
    form.resetFields();
  };

  const handleAdd = () => {
    setEditingNode(null);
    form.resetFields();
    form.setFieldsValue({ name: "", config: "" });
    setModalVisible(true);
  };

  const handleEdit = (record: CustomNode) => {
    setEditingNode(record);
    form.resetFields();
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      const response = await apiClient(`/api/nodes/${id}`, { method: "DELETE" });
      if (response.ok) {
        message.success("Node deleted");
        fetchNodes();
      } else {
        message.error(`Failed to delete: ${await response.text()}`);
      }
    } catch {
      message.error("Failed to delete node");
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const url = editingNode ? `/api/nodes/${editingNode.id}` : "/api/nodes";
      const method = editingNode ? "PUT" : "POST";
      const response = await apiClient(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(values),
      });
      if (response.ok) {
        message.success(`Node ${editingNode ? "updated" : "added"}`);
        closeModal();
        fetchNodes();
      } else {
        message.error(`Operation failed: ${await response.text()}`);
      }
    } catch {
      return;
    }
  };

  return (
    <div>
      <div style={{ marginBottom: 16, display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <Title level={2}>Nodes</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          Add Node
        </Button>
      </div>
      <Table
        rowKey="id"
        dataSource={nodes}
        loading={loading}
        pagination={{ pageSize: 20, showSizeChanger: false }}
        columns={[
          { title: "Name", dataIndex: "name", key: "name" },
          {
            title: "Config",
            dataIndex: "config",
            key: "config",
            render: (config: string) => config ? <Tag color="green">Configured</Tag> : <Tag>None</Tag>,
          },
          {
            title: "Updated At",
            dataIndex: "updated_at",
            key: "updated_at",
            render: (value: string) => formatDate24h(value),
          },
          {
            title: "Action",
            key: "action",
            render: (_: unknown, record: CustomNode) => (
              <Space>
                <Button icon={<EditOutlined />} onClick={() => handleEdit(record)} />
                <Popconfirm title="Sure to delete?" onConfirm={() => handleDelete(record.id)}>
                  <Button danger icon={<DeleteOutlined />} />
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />
      <Modal
        title={editingNode ? "Edit Node" : "Add Node"}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={closeModal}
        width={800}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true, message: "Name is required" }]}>
            <Input placeholder="My Node" />
          </Form.Item>
          <Form.Item
            name="config"
            label="Config"
            rules={[{ required: true, message: "Config is required" }]}
            getValueFromEvent={() => undefined}
          >
            <ConfigEditor />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default NodeManager;

import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Modal, Form, Input, Select, message, Popconfirm, Typography } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { formatDate24h } from '../utils';

const { Title } = Typography;

interface Rule {
  id: number;
  rule_type: string;
  pattern: string;
  proxy_group: string;
  created_at: string;
  updated_at: string;
}

const BUILT_IN_RULE_TYPES = ['DOMAIN-SUFFIX', 'DOMAIN-KEYWORD'];
const STATIC_PROXY_GROUPS = ['DIRECT', 'REJECT'];

const RuleManager: React.FC = () => {
  const [rules, setRules] = useState<Rule[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState<Rule | null>(null);
  const [proxyGroupOptions, setProxyGroupOptions] = useState<string[]>(STATIC_PROXY_GROUPS);
  const [customRuleType, setCustomRuleType] = useState(false);
  const [form] = Form.useForm();

  const fetchRules = async (nextPage = page, nextPageSize = pageSize) => {
    setLoading(true);
    try {
      const response = await fetch(`/api/rules?page=${nextPage}&page_size=${nextPageSize}`);
      const data = await response.json();
      setRules(data.rules || []);
      setPage(data.page || nextPage);
      setPageSize(data.page_size || nextPageSize);
      setTotal(data.total || 0);
    } catch (error) {
      message.error('Failed to fetch rules');
    } finally {
      setLoading(false);
    }
  };

  const fetchProxyGroupOptions = async () => {
    try {
      const response = await fetch('/api/proxy-groups');
      const data = await response.json();
      const dynamicNames = (data.groups || []).map((group: { name: string }) => group.name);
      setProxyGroupOptions([...STATIC_PROXY_GROUPS, ...dynamicNames]);
    } catch (error) {
      // keep static options
    }
  };

  useEffect(() => {
    fetchRules(1, pageSize);
    fetchProxyGroupOptions();
  }, []);

  const handleAdd = () => {
    setEditingRule(null);
    setCustomRuleType(false);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: Rule) => {
    setEditingRule(record);
    const isCustom = !BUILT_IN_RULE_TYPES.includes(record.rule_type);
    setCustomRuleType(isCustom);
    form.setFieldsValue({
      rule_type_selector: isCustom ? '__custom__' : record.rule_type,
      rule_type: isCustom ? record.rule_type : record.rule_type,
      pattern: record.pattern,
      proxy_group: record.proxy_group,
    });
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      const response = await fetch(`/api/rules/${id}`, { method: 'DELETE' });
      if (!response.ok) {
        message.error(`Delete failed: ${await response.text()}`);
        return;
      }
      message.success('Rule deleted');
      fetchRules(page, pageSize);
    } catch (error) {
      message.error('Failed to delete rule');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const submitValues = {
        rule_type: customRuleType ? values.rule_type : values.rule_type_selector,
        pattern: values.pattern,
        proxy_group: values.proxy_group,
      };

      const url = editingRule ? `/api/rules/${editingRule.id}` : '/api/rules';
      const method = editingRule ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(submitValues),
      });

      if (response.ok) {
        message.success(`Rule ${editingRule ? 'updated' : 'added'}`);
        setModalVisible(false);
        fetchRules(page, pageSize);
      } else {
        const errorText = await response.text();
        message.error(`Operation failed: ${errorText}`);
      }
    } catch (error) {
      // form validation error
    }
  };

  return (
    <div>
      <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2}>Rule Management</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          Add Rule
        </Button>
      </div>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={rules}
        columns={[
          { title: 'Rule Type', dataIndex: 'rule_type', key: 'rule_type' },
          { title: 'Pattern', dataIndex: 'pattern', key: 'pattern' },
          { title: 'Proxy Group', dataIndex: 'proxy_group', key: 'proxy_group' },
          { title: 'Updated At', dataIndex: 'updated_at', key: 'updated_at', render: (text: string) => formatDate24h(text) },
          {
            title: 'Action',
            key: 'action',
            render: (_: any, record: Rule) => (
              <Space>
                <Button icon={<EditOutlined />} onClick={() => handleEdit(record)} />
                <Popconfirm title="Sure to delete?" onConfirm={() => handleDelete(record.id)}>
                  <Button danger icon={<DeleteOutlined />} />
                </Popconfirm>
              </Space>
            ),
          },
        ]}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (nextPage, nextPageSize) => fetchRules(nextPage, nextPageSize),
        }}
      />

      <Modal
        title={editingRule ? 'Edit Rule' : 'Add Rule'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="rule_type_selector" label="Rule Type" rules={[{ required: true, message: 'Please select rule type' }]}>
            <Select
              options={[
                { label: 'DOMAIN-SUFFIX', value: 'DOMAIN-SUFFIX' },
                { label: 'DOMAIN-KEYWORD', value: 'DOMAIN-KEYWORD' },
                { label: 'Custom', value: '__custom__' },
              ]}
              onChange={(value) => {
                const useCustom = value === '__custom__';
                setCustomRuleType(useCustom);
                form.setFieldValue('rule_type', useCustom ? '' : value);
              }}
            />
          </Form.Item>

          <Form.Item name="rule_type" label="Custom Rule Type" hidden={!customRuleType} rules={[{ required: customRuleType, message: 'Please input custom rule type' }]}>
            <Input placeholder="GEOIP" />
          </Form.Item>

          <Form.Item name="pattern" label="Pattern" rules={[{ required: true, message: 'Please input pattern' }]}>
            <Input placeholder="example.com" />
          </Form.Item>

          <Form.Item name="proxy_group" label="Proxy Group" rules={[{ required: true, message: 'Please select proxy group' }]}>
            <Select
              showSearch
              options={proxyGroupOptions.map(name => ({ label: name, value: name }))}
              placeholder="Select proxy group"
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default RuleManager;

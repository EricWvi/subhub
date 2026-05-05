import React, { useEffect, useState } from "react";
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  App,
  Popconfirm,
  Typography,
  Tooltip,
  Switch,
} from "antd";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ImportOutlined,
  QuestionCircleOutlined,
} from "@ant-design/icons";
import { formatDate24h } from "../utils";

const { Title } = Typography;
const { TextArea } = Input;

interface Rule {
  id: number;
  rule_type: string;
  pattern: string;
  proxy_group: string;
  created_at: string;
  updated_at: string;
}

const BUILT_IN_RULE_TYPES = ["DOMAIN-SUFFIX", "DOMAIN-KEYWORD"];
const STATIC_PROXY_GROUPS = ["DIRECT", "REJECT"];

const IMPORT_HELP = (
  <div style={{ maxWidth: 320 }}>
    <p style={{ marginBottom: 0 }}>
      Paste rules in Clash format, one per line:
    </p>
    <code>{`<TYPE>,<pattern>,<target>`}</code>
    <p style={{ marginBottom: 0 }}>
      Target can be <code>DIRECT</code>, <code>REJECT</code>, or a proxy group
      name. Lines with unrecognized proxy groups will be skipped.
    </p>
  </div>
);

const RuleManager: React.FC = () => {
  const { message } = App.useApp();
  const [rules, setRules] = useState<Rule[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState<Rule | null>(null);
  const [proxyGroupOptions, setProxyGroupOptions] =
    useState<string[]>(STATIC_PROXY_GROUPS);
  const [customRuleType, setCustomRuleType] = useState(false);
  const [searchText, setSearchText] = useState("");
  const [importVisible, setImportVisible] = useState(false);
  const [importText, setImportText] = useState("");
  const [importReverse, setImportReverse] = useState(true);
  const [importing, setImporting] = useState(false);
  const [form] = Form.useForm();

  const fetchRules = async (
    nextPage = page,
    nextPageSize = pageSize,
    search = searchText,
  ) => {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: String(nextPage),
        page_size: String(nextPageSize),
      });
      if (search) params.set("search", search);
      const response = await fetch(`/api/rules?${params}`);
      const data = await response.json();
      setRules(data.rules || []);
      setPage(data.page || nextPage);
      setPageSize(data.page_size || nextPageSize);
      setTotal(data.total || 0);
    } catch (error) {
      message.error("Failed to fetch rules");
    } finally {
      setLoading(false);
    }
  };

  const fetchProxyGroupOptions = async () => {
    try {
      const response = await fetch("/api/proxy-groups");
      const data = await response.json();
      const dynamicNames = (data.groups || []).map(
        (group: { name: string }) => group.name,
      );
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
    setModalVisible(true);
  };

  const handleEdit = (record: Rule) => {
    setEditingRule(record);
    setCustomRuleType(!BUILT_IN_RULE_TYPES.includes(record.rule_type));
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      const response = await fetch(`/api/rules/${id}`, { method: "DELETE" });
      if (!response.ok) {
        message.error(`Delete failed: ${await response.text()}`);
        return;
      }
      message.success("Rule deleted");
      fetchRules(page, pageSize, searchText);
    } catch (error) {
      message.error("Failed to delete rule");
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const submitValues = {
        rule_type: customRuleType
          ? values.rule_type
          : values.rule_type_selector,
        pattern: values.pattern,
        proxy_group: values.proxy_group,
      };

      const url = editingRule ? `/api/rules/${editingRule.id}` : "/api/rules";
      const method = editingRule ? "PUT" : "POST";

      const response = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(submitValues),
      });

      if (response.ok) {
        message.success(`Rule ${editingRule ? "updated" : "added"}`);
        setModalVisible(false);
        fetchRules(page, pageSize, searchText);
      } else {
        const errorText = await response.text();
        message.error(`Operation failed: ${errorText}`);
      }
    } catch (error) {
      // form validation error
    }
  };

  const handleImport = async () => {
    if (!importText.trim()) {
      message.warning("Please paste rules first");
      return;
    }
    setImporting(true);
    try {
      const response = await fetch("/api/rules/import", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ rules: importText, reverse: importReverse }),
      });
      if (!response.ok) {
        message.error(`Import failed: ${await response.text()}`);
        return;
      }
      const result = await response.json();
      message.success(
        `Imported ${result.imported} rules${result.skipped > 0 ? `, skipped ${result.skipped}` : ""}`,
      );
      setImportVisible(false);
      setImportText("");
      fetchRules(1, pageSize, searchText);
    } catch (error) {
      message.error("Import failed");
    } finally {
      setImporting(false);
    }
  };

  return (
    <div>
      <div
        style={{
          marginBottom: "16px",
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
        }}
      >
        <Title level={2}>Rule Management</Title>
        <Space>
          <Input.Search
            placeholder="Search pattern..."
            allowClear
            onSearch={(value) => {
              setSearchText(value);
              fetchRules(1, pageSize, value);
            }}
            style={{ width: 250 }}
          />
          <Button
            icon={<ImportOutlined />}
            onClick={() => {
              setImportText("");
              setImportReverse(true);
              setImportVisible(true);
            }}
          />
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            Add Rule
          </Button>
        </Space>
      </div>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={rules}
        columns={[
          { title: "Rule Type", dataIndex: "rule_type", key: "rule_type" },
          { title: "Pattern", dataIndex: "pattern", key: "pattern" },
          {
            title: "Proxy Group",
            dataIndex: "proxy_group",
            key: "proxy_group",
          },
          {
            title: "Updated At",
            dataIndex: "updated_at",
            key: "updated_at",
            render: (text: string) => formatDate24h(text),
          },
          {
            title: "Action",
            key: "action",
            render: (_: any, record: Rule) => (
              <Space>
                <Button
                  icon={<EditOutlined />}
                  onClick={() => handleEdit(record)}
                />
                <Popconfirm
                  title="Sure to delete?"
                  onConfirm={() => handleDelete(record.id)}
                >
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
          showSizeChanger: false,
          onChange: (nextPage, nextPageSize) =>
            fetchRules(nextPage, nextPageSize, searchText),
        }}
      />

      <Modal
        key={editingRule ? `edit-${editingRule.id}` : "add"}
        title={editingRule ? "Edit Rule" : "Add Rule"}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
      >
        <Form 
          form={form} 
          layout="vertical" 
          preserve={false}
          initialValues={editingRule ? {
            rule_type_selector: !BUILT_IN_RULE_TYPES.includes(editingRule.rule_type) ? "__custom__" : editingRule.rule_type,
            rule_type: editingRule.rule_type,
            pattern: editingRule.pattern,
            proxy_group: editingRule.proxy_group,
          } : {
            rule_type_selector: "DOMAIN-SUFFIX",
            proxy_group: "DIRECT",
          }}
        >
          <Form.Item
            name="rule_type_selector"
            label="Rule Type"
            rules={[{ required: true, message: "Please select rule type" }]}
          >
            <Select
              options={[
                { label: "DOMAIN-SUFFIX", value: "DOMAIN-SUFFIX" },
                { label: "DOMAIN-KEYWORD", value: "DOMAIN-KEYWORD" },
                { label: "Custom", value: "__custom__" },
              ]}
              onChange={(value) => {
                const useCustom = value === "__custom__";
                setCustomRuleType(useCustom);
                form.setFieldValue("rule_type", useCustom ? "" : value);
              }}
            />
          </Form.Item>

          <Form.Item
            name="rule_type"
            label="Custom Rule Type"
            hidden={!customRuleType}
            rules={[
              {
                required: customRuleType,
                message: "Please input custom rule type",
              },
            ]}
          >
            <Input placeholder="GEOIP" />
          </Form.Item>

          <Form.Item
            name="pattern"
            label="Pattern"
            rules={[{ required: true, message: "Please input pattern" }]}
          >
            <Input placeholder="example.com" />
          </Form.Item>

          <Form.Item
            name="proxy_group"
            label="Proxy Group"
            rules={[{ required: true, message: "Please select proxy group" }]}
          >
            <Select
              showSearch
              options={proxyGroupOptions.map((name) => ({
                label: name,
                value: name,
              }))}
              placeholder="Select proxy group"
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="Import Rules"
        open={importVisible}
        onOk={handleImport}
        onCancel={() => setImportVisible(false)}
        confirmLoading={importing}
        okText="Import"
        width={640}
        
      >
        <div
          style={{
            marginBottom: 12,
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <span>
            Rules{" "}
            <Tooltip title={IMPORT_HELP}>
              <QuestionCircleOutlined
                style={{ color: "#1677ff", cursor: "pointer" }}
              />
            </Tooltip>
          </span>
          <Space>
            <span style={{ fontSize: 13 }}>Reverse order</span>
            <Tooltip title="Insert rules in reverse order so the first line in input gets the highest ID (matched first).">
              <QuestionCircleOutlined
                style={{ color: "#1677ff", cursor: "pointer" }}
              />
            </Tooltip>
            <Switch
              size="small"
              checked={importReverse}
              onChange={setImportReverse}
            />
          </Space>
        </div>
        <TextArea
          rows={12}
          value={importText}
          onChange={(e) => setImportText(e.target.value)}
          placeholder={`DOMAIN-SUFFIX,example.com,DIRECT\nDOMAIN-KEYWORD,google,REJECT\nDOMAIN,api.openai.com,MyProxy\n\nor\n\n- DOMAIN-SUFFIX,example.com,DIRECT\n- DOMAIN-KEYWORD,google,REJECT\n- DOMAIN,api.openai.com,MyProxy`}
          style={{ fontFamily: "monospace" }}
        />
      </Modal>
    </div>
  );
};

export default RuleManager;

import React, { useEffect, useState } from "react";
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  InputNumber,
  App,
  Popconfirm,
  Typography,
  Card,
  Descriptions,
  Tag,
  Collapse,
  Drawer,
  theme,
} from "antd";
import { PlusOutlined, EditOutlined, DeleteOutlined, CopyOutlined, EyeOutlined } from "@ant-design/icons";
import Editor from "@monaco-editor/react";
import { formatDate24h, useMonacoTheme } from "../utils";

const { Title, Text } = Typography;

interface ProxyMember {
  type: string;
  value: string;
}

interface ProxyGroup {
  id: number;
  name: string;
  type: string;
  position: number;
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

interface ProxyGroupFormValue {
  name: string;
  type: string;
  position: number;
  url: string;
  interval: number;
  proxies: ProxyMember[];
  bind_internal_proxy_group_id?: number;
  is_system: boolean;
}

const RESERVED_PROXY_GROUP_NAME = "Proxies";
const RESERVED_PROXY_GROUP_TYPE = "select";

const createReservedProxyGroup = (): ProxyGroupFormValue => ({
  name: RESERVED_PROXY_GROUP_NAME,
  type: RESERVED_PROXY_GROUP_TYPE,
  position: 0,
  url: "",
  interval: 0,
  proxies: [{ type: "DIRECT", value: "DIRECT" }],
  bind_internal_proxy_group_id: undefined,
  is_system: true,
});

const normalizeProxyGroupMembers = (
  proxies?: ProxyMember[],
): ProxyMember[] =>
  (proxies || []).map((proxy) => ({
    type: proxy.type,
    value: proxy.value,
  }));

const normalizeProxyGroups = (
  proxyGroups?: Partial<ProxyGroupFormValue>[],
): ProxyGroupFormValue[] => {
  const groups = [...(proxyGroups || [])].sort(
    (a, b) => (a.position ?? Number.MAX_SAFE_INTEGER) - (b.position ?? Number.MAX_SAFE_INTEGER),
  );
  const reservedGroup = groups.find(
    (group) => group.name === RESERVED_PROXY_GROUP_NAME,
  );
  const editableGroups = groups.filter(
    (group) => group.name !== RESERVED_PROXY_GROUP_NAME,
  );

  return [
    {
      ...createReservedProxyGroup(),
      bind_internal_proxy_group_id: reservedGroup?.bind_internal_proxy_group_id,
      proxies: normalizeProxyGroupMembers(reservedGroup?.proxies).length
        ? normalizeProxyGroupMembers(reservedGroup?.proxies)
        : createReservedProxyGroup().proxies,
    },
    ...editableGroups.map((group) => ({
      name: group.name || "",
      type: group.type || "select",
      position: 0,
      url: group.url || "",
      interval: group.interval || 0,
      proxies: normalizeProxyGroupMembers(group.proxies),
      bind_internal_proxy_group_id: group.bind_internal_proxy_group_id,
      is_system: false,
    })),
  ].map((group, index) => ({
    ...group,
    position: index,
    is_system: index === 0,
  }));
};

const ClashConfigSubscriptionManager: React.FC = () => {
  const { message } = App.useApp();
  const { token } = theme.useToken();
  const monacoTheme = useMonacoTheme();
  const [subscriptions, setSubscriptions] = useState<ClashConfigSubscription[]>(
    [],
  );
  const [loading, setLoading] = useState(false);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [internalGroups, setInternalGroups] = useState<InternalGroup[]>([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingSub, setEditingSub] = useState<ClashConfigSubscription | null>(
    null,
  );
  const [activeProxyGroupKey, setActiveProxyGroupKey] = useState<string[]>([
    "0",
  ]);
  const [form] = Form.useForm();
  const [previewDrawerVisible, setPreviewDrawerVisible] = useState(false);
  const [previewContent, setPreviewContent] = useState<string>("");
  const [previewLoading, setPreviewLoading] = useState(false);

  const fetchSubscriptions = async () => {
    setLoading(true);
    try {
      const response = await fetch("/api/subscriptions/clash-configs");
      const data = await response.json();
      setSubscriptions(data.subscriptions || []);
    } catch (error) {
      message.error("Failed to fetch subscriptions");
    } finally {
      setLoading(false);
    }
  };

  const fetchProviders = async () => {
    try {
      const response = await fetch("/api/providers");
      const data = await response.json();
      setProviders(data.providers || []);
    } catch (error) {
      /* ignore */
    }
  };

  const fetchInternalGroups = async () => {
    try {
      const response = await fetch("/api/proxy-groups");
      const data = await response.json();
      setInternalGroups(
        (data.groups || []).map((g: any) => ({ id: g.id, name: g.name })),
      );
    } catch (error) {
      /* ignore */
    }
  };

  useEffect(() => {
    fetchSubscriptions();
    fetchProviders();
    fetchInternalGroups();
  }, []);

  const closeModal = () => {
    setModalVisible(false);
    setEditingSub(null);
    setActiveProxyGroupKey(["0"]);
    form.resetFields();
  };

  const handleAdd = () => {
    setEditingSub(null);
    setModalVisible(true);
    setActiveProxyGroupKey(["0"]);
    setTimeout(() => {
      form.resetFields();
      form.setFieldsValue({
        name: "",
        providers: [],
        proxy_groups: [createReservedProxyGroup()],
      });
    }, 0);
  };

  const handleEdit = (record: ClashConfigSubscription) => {
    setEditingSub(record);
    const proxyGroups = normalizeProxyGroups(
      record.proxy_groups.map((pg) => ({
        name: pg.name,
        type: pg.type,
        position: pg.position,
        url: pg.url,
        interval: pg.interval,
        bind_internal_proxy_group_id: pg.bind_internal_proxy_group_id,
        is_system: pg.is_system,
        proxies: pg.proxies.map((p) => ({ type: p.type, value: p.value })),
      })),
    );
    setModalVisible(true);
    setActiveProxyGroupKey(["0"]);
    setTimeout(() => {
      form.setFieldsValue({
        name: record.name,
        providers: record.providers,
        proxy_groups: proxyGroups,
      });
    }, 0);
  };

  const handleDelete = async (id: number) => {
    try {
      const response = await fetch(`/api/subscriptions/clash-configs/${id}`, {
        method: "DELETE",
      });
      if (response.ok) {
        message.success("Subscription deleted");
        fetchSubscriptions();
      } else {
        const errorText = await response.text();
        message.error(`Failed to delete: ${errorText}`);
      }
    } catch (error) {
      message.error("Failed to delete subscription");
    }
  };

  const handleModalOk = async () => {
    try {
      await form.validateFields();
      const values = form.getFieldsValue(true);
      const payload = {
        ...values,
        proxy_groups: normalizeProxyGroups(values.proxy_groups).map((g) => {
          if (g.type !== "select" && !g.url) {
            g.url = "https://cp.cloudflare.com/generate_204";
          }
          return g;
        }),
      };
      const url = editingSub
        ? `/api/subscriptions/clash-configs/${editingSub.id}`
        : "/api/subscriptions/clash-configs";
      const method = editingSub ? "PUT" : "POST";

      const response = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (response.ok) {
        message.success(`Subscription ${editingSub ? "updated" : "added"}`);
        closeModal();
        fetchSubscriptions();
      } else {
        const errorText = await response.text();
        message.error(`Operation failed: ${errorText}`);
      }
    } catch (error) {
      /* form validation */
    }
  };

  const handleCopy = (id: number) => {
    const url = `${window.location.protocol}//${window.location.host}/api/subscriptions/clash-configs/${id}/content`;
    navigator.clipboard.writeText(url);
    message.success("Subscription URL copied to clipboard");
  };

  const handlePreview = async (id: number) => {
    setPreviewLoading(true);
    setPreviewDrawerVisible(true);
    try {
      const response = await fetch(`/api/subscriptions/clash-configs/${id}/content`);
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

  const providerMap = Object.fromEntries(providers.map((p) => [p.id, p.name]));
  const groupMap = Object.fromEntries(
    internalGroups.map((g) => [g.id, g.name]),
  );

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
        <Title level={3}>Clash Config Subscriptions</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          Add Subscription
        </Button>
      </div>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={subscriptions}
        pagination={{ pageSize: 20, showSizeChanger: false }}
        columns={[
          { title: "Name", dataIndex: "name", key: "name" },
          {
            title: "Providers",
            key: "providers",
            render: (_: any, record: ClashConfigSubscription) => (
              <span>
                {record.providers
                  .map((id) => providerMap[id] || id)
                  .join(" → ")}
              </span>
            ),
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
            render: (_: any, record: ClashConfigSubscription) => (
              <Space>
                <Button
                  icon={<CopyOutlined />}
                  onClick={() => handleCopy(record.id)}
                  title="Copy Subscription URL"
                />
                <Button
                  icon={<EyeOutlined />}
                  onClick={() => handlePreview(record.id)}
                  title="Preview"
                />
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
        expandable={{
          expandedRowRender: (record) => (
            <div style={{ padding: "8px 40px" }}>
              <Title level={5}>Proxy Groups</Title>
              {record.proxy_groups.map((pg) => (
                <Card
                  key={pg.id}
                  size="small"
                  title={pg.name}
                  style={{ marginBottom: 8 }}
                  extra={pg.is_system ? <Tag color="blue">System</Tag> : null}
                >
                  <Descriptions column={2} size="small">
                    <Descriptions.Item label="Type">
                      {pg.type}
                    </Descriptions.Item>
                    <Descriptions.Item label="Interval">
                      {pg.interval || "-"}
                    </Descriptions.Item>
                    {pg.bind_internal_proxy_group_id ? (
                      <Descriptions.Item label="Rules Bound Group">
                        {groupMap[pg.bind_internal_proxy_group_id] ||
                          pg.bind_internal_proxy_group_id}
                      </Descriptions.Item>
                    ) : null}
                    <Descriptions.Item label="Members">
                      {pg.proxies.map((m, idx) => (
                        <Tag key={`${m.type}:${m.value}:${idx}`}>
                          {m.type === "internal"
                            ? groupMap[Number(m.value)] || m.value
                            : m.value}
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
        title={
          editingSub
            ? "Edit Clash Config Subscription"
            : "Add Clash Config Subscription"
        }
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={closeModal}
        width={720}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="Name"
            rules={[{ required: true, message: "Name is required" }]}
          >
            <Input placeholder="Daily Config" />
          </Form.Item>
          <Form.Item
            name="providers"
            label="Providers"
            rules={[
              { required: true, message: "At least one provider is required" },
            ]}
          >
            <Select
              mode="multiple"
              placeholder="Select providers"
              options={providers.map((p) => ({ label: p.name, value: p.id }))}
            />
          </Form.Item>
          <Form.List name="proxy_groups">
            {(fields, { add, remove }) => (
              <>
                {/* UX target:
                - [ ] Add opens with a visible Proxies group already present
                - [ ] Proxies stays first, fixed-name/type, and non-deletable
                - [ ] Proxies keeps editable Rules Bound Group and Members
                - [ ] New user-created groups append after Proxies
                - [ ] Submit payload preserves list order via position and reserved-row is_system semantics
                */}
                <div
                  style={{
                    marginBottom: 8,
                    display: "flex",
                    justifyContent: "space-between",
                  }}
                >
                  <Text strong>Proxy Groups</Text>
                  <Button
                    size="small"
                    icon={<PlusOutlined />}
                    onClick={() => {
                      add({
                        name: "",
                        type: "select",
                        position: fields.length,
                        url: "",
                        interval: 0,
                        proxies: [],
                        bind_internal_proxy_group_id: undefined,
                        is_system: false,
                      });
                      setActiveProxyGroupKey([String(fields.length)]);
                    }}
                  >
                    Add Proxy Group
                  </Button>
                </div>
                <Collapse
                  accordion
                  activeKey={activeProxyGroupKey}
                  onChange={(key) =>
                    setActiveProxyGroupKey(
                      Array.isArray(key) ? key.map(String) : key ? [String(key)] : [],
                    )
                  }
                  items={fields.map((field) => {
                    const isReservedGroup = form.getFieldValue([
                      "proxy_groups",
                      field.name,
                      "is_system",
                    ]);
                    return {
                      key: String(field.name),
                      forceRender: true,
                      label: (
                        <Form.Item
                          noStyle
                          shouldUpdate={(prev, curr) =>
                            prev.proxy_groups?.[field.name]?.name !==
                            curr.proxy_groups?.[field.name]?.name
                          }
                        >
                          {() => (
                            <Space>
                              <Text strong>
                                {form.getFieldValue([
                                  "proxy_groups",
                                  field.name,
                                  "name",
                                ]) || "New Group"}
                              </Text>
                              {isReservedGroup && (
                                <Tag color="blue">System</Tag>
                              )}
                            </Space>
                          )}
                        </Form.Item>
                      ),
                      extra: !isReservedGroup && (
                        <Popconfirm
                          title="Remove this group?"
                          onConfirm={() => remove(field.name)}
                        >
                          <DeleteOutlined
                            style={{ color: token.colorError }}
                            onClick={(e) => e.stopPropagation()}
                          />
                        </Popconfirm>
                      ),
                      children: (
                        <>
                          <Form.Item name={[field.name, "is_system"]} hidden>
                            <Input />
                          </Form.Item>
                          <Form.Item name={[field.name, "position"]} hidden>
                            <InputNumber />
                          </Form.Item>
                          <Form.Item
                            name={[field.name, "name"]}
                            label="Name"
                            rules={[{ required: true }]}
                          >
                            <Input
                              placeholder="Media"
                              disabled={isReservedGroup}
                            />
                          </Form.Item>
                          <Form.Item
                            name={[field.name, "type"]}
                            label="Type"
                            rules={[{ required: true }]}
                          >
                            <Select
                              disabled={isReservedGroup}
                              options={[
                                { label: "select", value: "select" },
                                { label: "url-test", value: "url-test" },
                                { label: "fallback", value: "fallback" },
                              ]}
                            />
                          </Form.Item>

                          <Form.Item
                            noStyle
                            shouldUpdate={(prev, curr) =>
                              prev.proxy_groups?.[field.name]?.type !==
                              curr.proxy_groups?.[field.name]?.type
                            }
                          >
                            {() => {
                              const type = form.getFieldValue([
                                "proxy_groups",
                                field.name,
                                "type",
                              ]);
                              if (type === "select") return null;
                              return (
                                <>
                                  <Form.Item
                                    name={[field.name, "url"]}
                                    label="URL"
                                  >
                                    <Input placeholder="https://cp.cloudflare.com/generate_204" />
                                  </Form.Item>
                                  <Form.Item
                                    name={[field.name, "interval"]}
                                    label="Interval (seconds)"
                                  >
                                    <InputNumber
                                      min={0}
                                      style={{ width: "100%" }}
                                    />
                                  </Form.Item>
                                </>
                              );
                            }}
                          </Form.Item>

                          <Form.Item
                            noStyle
                            shouldUpdate={(prev, curr) =>
                              JSON.stringify(
                                (prev.proxy_groups || []).map(
                                  (g: ProxyGroupFormValue) => g.bind_internal_proxy_group_id,
                                ),
                              ) !==
                              JSON.stringify(
                                (curr.proxy_groups || []).map(
                                  (g: ProxyGroupFormValue) => g.bind_internal_proxy_group_id,
                                ),
                              )
                            }
                          >
                            {() => {
                              const allGroups: ProxyGroupFormValue[] =
                                form.getFieldValue("proxy_groups") || [];
                              const usedIds = allGroups
                                .filter((_: any, i: number) => i !== field.name)
                                .map((g: ProxyGroupFormValue) => g.bind_internal_proxy_group_id)
                                .filter(Boolean);
                              return (
                                <Form.Item
                                  name={[field.name, "bind_internal_proxy_group_id"]}
                                  label="Rules Bound Group"
                                >
                                  <Select
                                    allowClear
                                    placeholder="Select internal group"
                                    options={internalGroups
                                      .filter((g) => !usedIds.includes(g.id))
                                      .map((g) => ({ label: g.name, value: g.id }))}
                                  />
                                </Form.Item>
                              );
                            }}
                          </Form.Item>
                          <Form.List name={[field.name, "proxies"]}>
                            {(
                              memberFields,
                              { add: addMember, remove: removeMember },
                            ) => (
                              <>
                                <div
                                  style={{
                                    marginBottom: 4,
                                    display: "flex",
                                    justifyContent: "space-between",
                                    alignItems: "center",
                                  }}
                                >
                                  <Text strong>Members</Text>
                                  <Button
                                    size="small"
                                    icon={<PlusOutlined />}
                                    onClick={() =>
                                      addMember({
                                        type: "DIRECT",
                                        value: "DIRECT",
                                      })
                                    }
                                  >
                                    Add Member
                                  </Button>
                                </div>
                                {memberFields.map((mf) => (
                                  <Space
                                    key={mf.key}
                                    style={{ display: "flex", marginBottom: 4 }}
                                    align="baseline"
                                  >
                                    <Form.Item name={[mf.name, "type"]} noStyle>
                                      <Select
                                        style={{ width: 120 }}
                                        options={[
                                          {
                                            label: "Internal",
                                            value: "internal",
                                          },
                                          {
                                            label: "Reference",
                                            value: "reference",
                                          },
                                          { label: "DIRECT", value: "DIRECT" },
                                          { label: "REJECT", value: "REJECT" },
                                        ]}
                                      />
                                    </Form.Item>
                                    <Form.Item
                                      noStyle
                                      shouldUpdate={(prev, curr) =>
                                        prev.proxy_groups?.[field.name]
                                          ?.proxies?.[mf.name]?.type !==
                                        curr.proxy_groups?.[field.name]
                                          ?.proxies?.[mf.name]?.type
                                      }
                                    >
                                      {() => {
                                        const type = form.getFieldValue([
                                          "proxy_groups",
                                          field.name,
                                          "proxies",
                                          mf.name,
                                          "type",
                                        ]);
                                        if (type === "internal") {
                                          return (
                                            <Form.Item
                                              name={[mf.name, "value"]}
                                              noStyle
                                              rules={[{ required: true }]}
                                            >
                                              <Select
                                                placeholder="Select Group"
                                                style={{ width: 200 }}
                                                options={internalGroups.map(
                                                  (g) => ({
                                                    label: g.name,
                                                    value: String(g.id),
                                                  }),
                                                )}
                                              />
                                            </Form.Item>
                                          );
                                        }
                                        if (type === "reference") {
                                          const allGroups: ProxyGroupFormValue[] =
                                            form.getFieldValue("proxy_groups") || [];
                                          const currentName = form.getFieldValue([
                                            "proxy_groups",
                                            field.name,
                                            "name",
                                          ]);
                                          return (
                                            <Form.Item
                                              name={[mf.name, "value"]}
                                              noStyle
                                              rules={[{ required: true }]}
                                            >
                                              <Select
                                                placeholder="Select Proxy Group"
                                                style={{ width: 200 }}
                                                options={allGroups
                                                  .map((g) => g.name)
                                                  .filter((n) => n && n !== currentName)
                                                  .map((name) => ({ label: name, value: name }))}
                                              />
                                            </Form.Item>
                                          );
                                        }
                                        return (
                                          <Form.Item
                                            name={[mf.name, "value"]}
                                            noStyle
                                            rules={[{ required: true }]}
                                          >
                                            <Input
                                              placeholder="value"
                                              style={{ width: 200 }}
                                            />
                                          </Form.Item>
                                        );
                                      }}
                                    </Form.Item>
                                    <Button
                                      danger
                                      size="small"
                                      icon={<DeleteOutlined />}
                                      onClick={() => removeMember(mf.name)}
                                    />
                                  </Space>
                                ))}
                              </>
                            )}
                          </Form.List>
                        </>
                      ),
                    };
                  })}
                />
              </>
            )}
          </Form.List>
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

export default ClashConfigSubscriptionManager;

import React, { useEffect, useState } from "react";
import { ConfigProvider, theme, Layout, Menu, Typography, App as AntdApp } from "antd";
import ProviderManager from "./components/ProviderManager";
import ProxyGroupManager from "./components/ProxyGroupManager";
import RuleManager from "./components/RuleManager";
import SubscriptionManager from "./components/SubscriptionManager";
import "./App.css";

const { Header, Content, Footer } = Layout;
const { Title } = Typography;

const AppContent: React.FC = () => {
  const [currentTab, setCurrentTab] = useState("providers");
  const { token } = theme.useToken();

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Header style={{ display: "flex", alignItems: "center" }}>
        <Title
          level={3}
          style={{ color: token.colorTextLightSolid, margin: 0 }}
        >
          SubHub
        </Title>
        <Menu
          theme="dark"
          mode="horizontal"
          selectedKeys={[currentTab]}
          onClick={(e) => setCurrentTab(e.key)}
          items={[
            { key: "providers", label: "Providers" },
            { key: "groups", label: "Proxy Groups" },
            { key: "rules", label: "Rules" },
            { key: "subscriptions", label: "Subscriptions" },
          ]}
          style={{ flex: 1, marginLeft: "24px" }}
        />
      </Header>
      <Content style={{ padding: "0 50px" }}>
        <div
          style={{
            background: token.colorBgContainer,
            padding: 24,
            minHeight: 280,
            marginTop: "24px",
          }}
        >
          {currentTab === "providers" ? (
            <ProviderManager />
          ) : currentTab === "groups" ? (
            <ProxyGroupManager />
          ) : currentTab === "rules" ? (
            <RuleManager />
          ) : (
            <SubscriptionManager />
          )}
        </div>
      </Content>
      <Footer style={{ textAlign: "center" }}>
        SubHub ©2026 Created by Gemini CLI
      </Footer>
    </Layout>
  );
};

const App: React.FC = () => {
  const [isDarkMode, setIsDarkMode] = useState(
    window.matchMedia("(prefers-color-scheme: dark)").matches,
  );
  useEffect(() => {
    // 监听系统主题变化
    const query = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = (e: MediaQueryListEvent) => setIsDarkMode(e.matches);

    query.addEventListener("change", handler);
    return () => query.removeEventListener("change", handler);
  }, []);
  return (
    <ConfigProvider
      theme={{
        algorithm: isDarkMode ? theme.darkAlgorithm : theme.defaultAlgorithm,
      }}
    >
      <AntdApp>
        <AppContent />
      </AntdApp>
    </ConfigProvider>
  );
};

export default App;

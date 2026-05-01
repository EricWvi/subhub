import React from 'react';
import { Layout, Menu, Typography } from 'antd';
import ProviderManager from './components/ProviderManager';
import "./App.css";

const { Header, Content, Footer } = Layout;
const { Title } = Typography;

const App: React.FC = () => {
  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center' }}>
        <Title level={3} style={{ color: 'white', margin: 0 }}>SubHub</Title>
        <Menu
          theme="dark"
          mode="horizontal"
          defaultSelectedKeys={['1']}
          items={[{ key: '1', label: 'Providers' }]}
          style={{ flex: 1, marginLeft: '24px' }}
        />
      </Header>
      <Content style={{ padding: '0 50px' }}>
        <div style={{ background: '#fff', padding: 24, minHeight: 280, marginTop: '24px' }}>
          <ProviderManager />
        </div>
      </Content>
      <Footer style={{ textAlign: 'center' }}>SubHub ©2026 Created by Gemini CLI</Footer>
    </Layout>
  );
}

export default App;

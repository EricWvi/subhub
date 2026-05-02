import React from 'react';
import { Tabs } from 'antd';
import ClashConfigSubscriptionManager from './ClashConfigSubscriptionManager';
import ProxyProviderSubscriptionManager from './ProxyProviderSubscriptionManager';
import RuleProviderSubscriptionManager from './RuleProviderSubscriptionManager';

const SubscriptionManager: React.FC = () => {
  return (
    <Tabs
      defaultActiveKey="clash-configs"
      items={[
        { key: 'clash-configs', label: 'Clash Configs', children: <ClashConfigSubscriptionManager /> },
        { key: 'proxy-providers', label: 'Proxy Providers', children: <ProxyProviderSubscriptionManager /> },
        { key: 'rule-providers', label: 'Rule Providers', children: <RuleProviderSubscriptionManager /> },
      ]}
    />
  );
};

export default SubscriptionManager;

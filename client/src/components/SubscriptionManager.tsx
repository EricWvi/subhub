import React from 'react';
import { Tabs } from 'antd';
import ClashConfigSubscriptionManager from './ClashConfigSubscriptionManager';
import NodeProviderSubscriptionManager from './NodeProviderSubscriptionManager';
import ProxyProviderSubscriptionManager from './ProxyProviderSubscriptionManager';
import RuleProviderSubscriptionManager from './RuleProviderSubscriptionManager';

const SubscriptionManager: React.FC = () => {
  return (
    <Tabs
      defaultActiveKey="clash-configs"
      items={[
        { key: 'clash-configs', label: 'Clash Configs', children: <ClashConfigSubscriptionManager /> },
        { key: 'node-providers', label: 'Node Providers', children: <NodeProviderSubscriptionManager /> },
        { key: 'proxy-providers', label: 'Proxy Providers', children: <ProxyProviderSubscriptionManager /> },
        { key: 'rule-providers', label: 'Rule Providers', children: <RuleProviderSubscriptionManager /> },
      ]}
    />
  );
};

export default SubscriptionManager;

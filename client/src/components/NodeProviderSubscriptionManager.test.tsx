import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { ApiClient } from '../api';
import { renderWithProviders } from '../test/render';
import NodeProviderSubscriptionManager from './NodeProviderSubscriptionManager';

const nodes = [{ id: 1, name: 'Office' }, { id: 2, name: 'Home' }];

vi.mock('@monaco-editor/react', () => ({
  default: ({ value }: { value: string }) => <pre>{value}</pre>,
}));

describe('NodeProviderSubscriptionManager', () => {
  it('creates a subscription using a name and node checkboxes', async () => {
    const user = userEvent.setup();
    const apiClient = vi.fn<ApiClient>(async (path) =>
      Response.json(path === '/api/nodes' ? { nodes } : { subscriptions: [] }),
    );
    renderWithProviders(<NodeProviderSubscriptionManager />, { apiClient });
    await waitFor(() => expect(apiClient).toHaveBeenCalledWith('/api/nodes'));
    await user.click(screen.getByRole('button', { name: /add/i }));
    const modal = within(screen.getByRole('dialog'));
    await user.type(modal.getByLabelText('Name'), 'My Nodes');
    await user.click(modal.getByRole('button', { name: 'OK' }));
    expect(await modal.findByText('Select at least one node')).toBeInTheDocument();
    await user.click(modal.getByRole('checkbox', { name: 'Home' }));
    await user.click(modal.getByRole('button', { name: 'OK' }));
    await waitFor(() => expect(apiClient).toHaveBeenCalledWith('/api/subscriptions/node-providers', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'My Nodes', nodes: [2] }),
    }));
    expect(apiClient.mock.calls.some(([path]) => String(path).includes('proxy-groups'))).toBe(false);
  });

  it('edits selected nodes and previews and copies the node-provider output', async () => {
    const user = userEvent.setup();
    const sub = { id: 4, name: 'My Nodes', nodes: [1], updated_at: '2026-09-02T13:05:09+08:00' };
    const apiClient = vi.fn<ApiClient>(async (path) => {
      if (String(path).endsWith('/content')) return new Response('proxies:\n  - name: Office');
      return Response.json(path === '/api/nodes' ? { nodes } : { subscriptions: [sub] });
    });
    const writeText = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue();
    renderWithProviders(<NodeProviderSubscriptionManager />, { apiClient });
    expect(await screen.findByText('My Nodes')).toBeInTheDocument();
    expect(screen.getByText('Office')).toBeInTheDocument();
    await user.click(screen.getByTitle('Edit'));
    const modal = within(await screen.findByRole('dialog', { name: 'Edit' }));
    expect(modal.getByRole('checkbox', { name: 'Office' })).toBeChecked();
    expect(modal.getByRole('checkbox', { name: 'Home' })).not.toBeChecked();
    await user.click(modal.getByRole('checkbox', { name: 'Office' }));
    await user.click(modal.getByRole('checkbox', { name: 'Home' }));
    await user.click(modal.getByRole('button', { name: 'OK' }));
    await waitFor(() => expect(apiClient).toHaveBeenCalledWith('/api/subscriptions/node-providers/4', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'My Nodes', nodes: [2] }),
    }));
    await user.click(screen.getByTitle('Copy Link'));
    expect(writeText).toHaveBeenCalledWith(`${window.location.origin}/api/subscriptions/node-providers/4/content`);
    await user.click(screen.getByTitle('Preview'));
    expect(await screen.findByText(/proxies:/)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Close' }));
    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Content Preview' })).not.toBeInTheDocument());
  });
});

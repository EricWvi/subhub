import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ApiClient } from "../api";
import { renderWithProviders } from "../test/render";
import RuleManager from "./RuleManager";

const existingRule = {
  id: 1,
  rule_type: "DOMAIN-KEYWORD",
  pattern: "existing.example",
  proxy_group: "REJECT",
  created_at: "2026-09-02T05:05:09Z",
  updated_at: "2026-09-02T05:05:09Z",
};

describe("RuleManager", () => {
  it("keeps another rule's values when editing it after adding a rule", async () => {
    const user = userEvent.setup();
    let rules = [existingRule];
    const apiClient = vi.fn<ApiClient>(async (input, init) => {
      const url = input.toString();

      if (url === "/api/proxy-groups") {
        return Response.json({ groups: [] });
      }
      if (url.startsWith("/api/rules?") && !init?.method) {
        return Response.json({
          rules,
          page: 1,
          page_size: 20,
          total: rules.length,
        });
      }
      if (url === "/api/rules" && init?.method === "POST") {
        rules = [
          ...rules,
          {
            ...existingRule,
            id: 2,
            rule_type: "DOMAIN-SUFFIX",
            pattern: "added.example",
            proxy_group: "DIRECT",
          },
        ];
        return Response.json({}, { status: 201 });
      }

      return Response.json({}, { status: 404 });
    });

    renderWithProviders(<RuleManager />, { apiClient });

    expect(await screen.findByText("existing.example")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Add Rule/ }));
    await user.type(screen.getByLabelText("Pattern"), "added.example");
    await user.click(screen.getByRole("button", { name: "OK" }));

    await waitFor(() =>
      expect(apiClient).toHaveBeenCalledWith(
        "/api/rules",
        expect.objectContaining({ method: "POST" }),
      ),
    );
    expect(await screen.findByText("added.example")).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: "Edit existing.example" }),
    );

    const dialog = screen.getByRole("dialog", { name: "Edit Rule" });
    expect
      .soft(within(dialog).getByLabelText("Pattern"))
      .toHaveValue("existing.example");
    expect
      .soft(within(dialog).queryByText("DOMAIN-KEYWORD"))
      .toBeInTheDocument();
    expect.soft(within(dialog).queryByText("REJECT")).toBeInTheDocument();
  });
});

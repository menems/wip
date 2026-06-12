import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  RoleForm,
  PERMISSION_MATRIX,
  permissionsSetToArray,
  permissionsArrayToSet,
} from "./RoleForm";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const onSubmit = vi.fn();
const onCancel = vi.fn();

function renderForm(props: Partial<Parameters<typeof RoleForm>[0]> = {}) {
  return render(
    <RoleForm
      mode="create"
      isSubmitting={false}
      apiError={null}
      onSubmit={onSubmit}
      onCancel={onCancel}
      {...props}
    />
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("RoleForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // -------------------------------------------------------------------------
  describe("fields", () => {
    it("renders name and description inputs", () => {
      renderForm();

      expect(screen.getByLabelText(/name/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
    });

    it("pre-fills name and description from defaultValues", () => {
      renderForm({
        mode: "edit",
        defaultValues: {
          name: "viewer",
          description: "Read-only access",
          permissions: new Set(),
        },
      });

      expect(screen.getByLabelText(/name/i)).toHaveValue("viewer");
      expect(screen.getByLabelText(/description/i)).toHaveValue(
        "Read-only access"
      );
    });
  });

  // -------------------------------------------------------------------------
  describe("permission matrix", () => {
    it("renders a checkbox for each valid resource:action combination", () => {
      renderForm();

      // users: read, write, delete
      expect(
        screen.getByRole("checkbox", { name: "users read" })
      ).toBeInTheDocument();
      expect(
        screen.getByRole("checkbox", { name: "users write" })
      ).toBeInTheDocument();
      expect(
        screen.getByRole("checkbox", { name: "users delete" })
      ).toBeInTheDocument();

      // roles: read, write, delete
      expect(
        screen.getByRole("checkbox", { name: "roles read" })
      ).toBeInTheDocument();

      // audit_logs: read only
      expect(
        screen.getByRole("checkbox", { name: "audit_logs read" })
      ).toBeInTheDocument();
    });

    it("does not render write/delete checkboxes for audit_logs", () => {
      renderForm();

      expect(
        screen.queryByRole("checkbox", { name: "audit_logs write" })
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("checkbox", { name: "audit_logs delete" })
      ).not.toBeInTheDocument();
    });

    it("pre-checks permissions from defaultValues", () => {
      renderForm({
        defaultValues: {
          permissions: new Set(["users:read", "audit_logs:read"]),
        },
      });

      expect(
        screen.getByRole("checkbox", { name: "users read" })
      ).toBeChecked();
      expect(
        screen.getByRole("checkbox", { name: "audit_logs read" })
      ).toBeChecked();
      expect(
        screen.getByRole("checkbox", { name: "users write" })
      ).not.toBeChecked();
    });

    it("toggles a permission on click", async () => {
      renderForm();
      const user = userEvent.setup();

      const checkbox = screen.getByRole("checkbox", { name: "users read" });
      expect(checkbox).not.toBeChecked();

      await user.click(checkbox);
      expect(checkbox).toBeChecked();

      await user.click(checkbox);
      expect(checkbox).not.toBeChecked();
    });

    it("disables all checkboxes when isSystemRole=true", () => {
      renderForm({ isSystemRole: true });

      const checkboxes = screen.getAllByRole("checkbox");
      checkboxes.forEach((cb) => expect(cb).toBeDisabled());
    });

    it("disables all checkboxes when isSubmitting=true", () => {
      renderForm({ isSubmitting: true });

      const checkboxes = screen.getAllByRole("checkbox");
      checkboxes.forEach((cb) => expect(cb).toBeDisabled());
    });
  });

  // -------------------------------------------------------------------------
  describe("validation", () => {
    it("shows a name-required error when submitting an empty form", async () => {
      renderForm();
      const user = userEvent.setup();

      await user.click(screen.getByRole("button", { name: /create role/i }));

      expect(screen.getByText(/name is required/i)).toBeInTheDocument();
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it("clears the name error when the user starts typing", async () => {
      renderForm();
      const user = userEvent.setup();

      await user.click(screen.getByRole("button", { name: /create role/i }));
      expect(screen.getByText(/name is required/i)).toBeInTheDocument();

      await user.type(screen.getByLabelText(/name/i), "v");
      expect(
        screen.queryByText(/name is required/i)
      ).not.toBeInTheDocument();
    });
  });

  // -------------------------------------------------------------------------
  describe("submit", () => {
    it("calls onSubmit with correct values when the form is valid", async () => {
      renderForm();
      const user = userEvent.setup();

      await user.type(screen.getByLabelText(/name/i), "viewer");
      await user.click(
        screen.getByRole("checkbox", { name: "users read" })
      );
      await user.click(
        screen.getByRole("button", { name: /create role/i })
      );

      expect(onSubmit).toHaveBeenCalledOnce();
      const submitted = onSubmit.mock.calls[0][0] as Parameters<typeof onSubmit>[0];
      expect(submitted.name).toBe("viewer");
      expect(submitted.permissions.has("users:read")).toBe(true);
    });

    it("calls onSubmit even with an empty permission set (valid)", async () => {
      renderForm();
      const user = userEvent.setup();

      await user.type(screen.getByLabelText(/name/i), "empty-role");
      await user.click(
        screen.getByRole("button", { name: /create role/i })
      );

      expect(onSubmit).toHaveBeenCalledOnce();
    });
  });

  // -------------------------------------------------------------------------
  describe("create vs edit mode", () => {
    it("shows 'Create Role' button in create mode", () => {
      renderForm({ mode: "create" });
      expect(
        screen.getByRole("button", { name: /create role/i })
      ).toBeInTheDocument();
    });

    it("shows 'Save Changes' button in edit mode", () => {
      renderForm({ mode: "edit" });
      expect(
        screen.getByRole("button", { name: /save changes/i })
      ).toBeInTheDocument();
    });

    it("hides the submit button and shows 'Back' for system roles", () => {
      renderForm({ mode: "edit", isSystemRole: true });

      expect(
        screen.queryByRole("button", { name: /save changes/i })
      ).not.toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: /back/i })
      ).toBeInTheDocument();
    });
  });

  // -------------------------------------------------------------------------
  describe("API error", () => {
    it("displays the API error message", () => {
      renderForm({ apiError: "Role name already in use." });

      expect(screen.getByRole("alert")).toHaveTextContent(
        "Role name already in use."
      );
    });
  });

  // -------------------------------------------------------------------------
  describe("cancel", () => {
    it("calls onCancel when Cancel is clicked", async () => {
      renderForm();
      const user = userEvent.setup();

      await user.click(screen.getByRole("button", { name: /cancel/i }));

      expect(onCancel).toHaveBeenCalledOnce();
      expect(onSubmit).not.toHaveBeenCalled();
    });
  });
});

// ---------------------------------------------------------------------------
// Utility function tests
// ---------------------------------------------------------------------------

describe("permissionsSetToArray", () => {
  it("converts a Set of keys to an array of objects", () => {
    const result = permissionsSetToArray(
      new Set(["users:read", "roles:write"])
    );

    expect(result).toHaveLength(2);
    expect(result).toContainEqual({ resource: "users", action: "read" });
    expect(result).toContainEqual({ resource: "roles", action: "write" });
  });

  it("returns an empty array for an empty Set", () => {
    expect(permissionsSetToArray(new Set())).toEqual([]);
  });
});

describe("permissionsArrayToSet", () => {
  it("converts an array of objects to a Set of keys", () => {
    const result = permissionsArrayToSet([
      { resource: "users", action: "read" },
      { resource: "audit_logs", action: "read" },
    ]);

    expect(result.has("users:read")).toBe(true);
    expect(result.has("audit_logs:read")).toBe(true);
    expect(result.has("users:write")).toBe(false);
  });

  it("returns an empty Set for an empty array", () => {
    expect(permissionsArrayToSet([])).toEqual(new Set());
  });
});

describe("PERMISSION_MATRIX", () => {
  it("contains users, roles, and audit_logs resources", () => {
    const resources = PERMISSION_MATRIX.map((r) => r.resource);
    expect(resources).toContain("users");
    expect(resources).toContain("roles");
    expect(resources).toContain("audit_logs");
  });

  it("only includes read for audit_logs", () => {
    const auditEntry = PERMISSION_MATRIX.find(
      (r) => r.resource === "audit_logs"
    );
    expect(auditEntry?.actions).toEqual(["read"]);
  });

  it("includes read, write, delete for users and roles", () => {
    ["users", "roles"].forEach((resource) => {
      const entry = PERMISSION_MATRIX.find((r) => r.resource === resource);
      expect(entry?.actions).toContain("read");
      expect(entry?.actions).toContain("write");
      expect(entry?.actions).toContain("delete");
    });
  });
});

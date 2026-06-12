import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { UserForm } from "./UserForm";

// ---------------------------------------------------------------------------
// Mock useRoles so the Role <Select> has options without a real QueryClient
// ---------------------------------------------------------------------------

vi.mock("@/features/roles/useRoles", () => ({
  useRoles: () => ({
    data: [
      {
        id: "role-1",
        name: "admin",
        description: "Administrator",
        is_system: true,
        permissions: [],
        created_at: "",
      },
      {
        id: "role-2",
        name: "viewer",
        description: "Viewer",
        is_system: false,
        permissions: [],
        created_at: "",
      },
    ],
    isLoading: false,
  }),
}));

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const onSubmit = vi.fn();
const onCancel = vi.fn();

function renderForm(props: Partial<Parameters<typeof UserForm>[0]> = {}) {
  return render(
    <UserForm
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

describe("UserForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // -------------------------------------------------------------------------
  describe("create mode", () => {
    it("renders name, email, password, and role fields", () => {
      renderForm({ mode: "create" });

      expect(screen.getByLabelText(/name/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
      // Role label exists (the Select trigger renders with the label text nearby)
      expect(screen.getByText(/^role$/i)).toBeInTheDocument();
    });

    it("shows the 'Create User' submit button", () => {
      renderForm({ mode: "create" });
      expect(
        screen.getByRole("button", { name: /create user/i })
      ).toBeInTheDocument();
    });

    it("shows validation errors when submitting an empty form", async () => {
      renderForm({ mode: "create" });
      const user = userEvent.setup();

      await user.click(screen.getByRole("button", { name: /create user/i }));

      expect(screen.getByText(/name is required/i)).toBeInTheDocument();
      expect(screen.getByText(/email is required/i)).toBeInTheDocument();
      expect(screen.getByText(/password is required/i)).toBeInTheDocument();
      expect(screen.getByText(/please select a role/i)).toBeInTheDocument();
    });

    it("shows a length error when password is fewer than 8 characters", async () => {
      renderForm({ mode: "create" });
      const user = userEvent.setup();

      await user.type(screen.getByLabelText(/name/i), "Alice");
      await user.type(screen.getByLabelText(/email/i), "alice@example.com");
      await user.type(screen.getByLabelText(/password/i), "short");

      await user.click(screen.getByRole("button", { name: /create user/i }));

      expect(
        screen.getByText(/at least 8 characters/i)
      ).toBeInTheDocument();
    });

    it("clears a field error when the user starts typing in that field", async () => {
      renderForm({ mode: "create" });
      const user = userEvent.setup();

      // Trigger validation
      await user.click(screen.getByRole("button", { name: /create user/i }));
      expect(screen.getByText(/name is required/i)).toBeInTheDocument();

      // Typing in name should clear the name error
      await user.type(screen.getByLabelText(/name/i), "A");
      expect(
        screen.queryByText(/name is required/i)
      ).not.toBeInTheDocument();
    });

    it("does not call onSubmit when validation fails", async () => {
      renderForm({ mode: "create" });
      const user = userEvent.setup();

      await user.click(screen.getByRole("button", { name: /create user/i }));

      expect(onSubmit).not.toHaveBeenCalled();
    });
  });

  // -------------------------------------------------------------------------
  describe("edit mode", () => {
    it("does not render the password field", () => {
      renderForm({ mode: "edit" });

      expect(screen.queryByLabelText(/password/i)).not.toBeInTheDocument();
    });

    it("shows the 'Save Changes' submit button", () => {
      renderForm({ mode: "edit" });
      expect(
        screen.getByRole("button", { name: /save changes/i })
      ).toBeInTheDocument();
    });

    it("pre-fills name and email from defaultValues", () => {
      renderForm({
        mode: "edit",
        defaultValues: {
          name: "Bob",
          email: "bob@example.com",
          roleId: "role-1",
        },
      });

      expect(screen.getByLabelText(/name/i)).toHaveValue("Bob");
      expect(screen.getByLabelText(/email/i)).toHaveValue("bob@example.com");
    });

    it("calls onSubmit with the correct values when the form is valid", async () => {
      // In edit mode with a pre-filled roleId the form is immediately valid
      // once name and email are present.
      renderForm({
        mode: "edit",
        defaultValues: {
          name: "Bob",
          email: "bob@example.com",
          roleId: "role-1",
        },
      });
      const user = userEvent.setup();

      await user.click(screen.getByRole("button", { name: /save changes/i }));

      expect(onSubmit).toHaveBeenCalledOnce();
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "Bob",
          email: "bob@example.com",
          roleId: "role-1",
        })
      );
    });

    it("shows validation errors on empty submit in edit mode", async () => {
      renderForm({ mode: "edit" });
      const user = userEvent.setup();

      await user.click(screen.getByRole("button", { name: /save changes/i }));

      expect(screen.getByText(/name is required/i)).toBeInTheDocument();
      expect(screen.getByText(/email is required/i)).toBeInTheDocument();
      // No password error in edit mode
      expect(
        screen.queryByText(/password is required/i)
      ).not.toBeInTheDocument();
    });
  });

  // -------------------------------------------------------------------------
  describe("API error display", () => {
    it("shows the API error message when apiError is provided", () => {
      renderForm({ apiError: "Email already in use." });

      const alert = screen.getByRole("alert");
      expect(alert).toHaveTextContent("Email already in use.");
    });

    it("does not render an alert when apiError is null", () => {
      renderForm({ apiError: null });

      expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });
  });

  // -------------------------------------------------------------------------
  describe("cancel", () => {
    it("calls onCancel when the Cancel button is clicked", async () => {
      renderForm();
      const user = userEvent.setup();

      await user.click(screen.getByRole("button", { name: /cancel/i }));

      expect(onCancel).toHaveBeenCalledOnce();
    });

    it("does not submit the form when Cancel is clicked", async () => {
      renderForm();
      const user = userEvent.setup();

      await user.click(screen.getByRole("button", { name: /cancel/i }));

      expect(onSubmit).not.toHaveBeenCalled();
    });
  });

  // -------------------------------------------------------------------------
  describe("disabled state (isSubmitting=true)", () => {
    it("disables name and email inputs", () => {
      renderForm({ isSubmitting: true });

      expect(screen.getByLabelText(/name/i)).toBeDisabled();
      expect(screen.getByLabelText(/email/i)).toBeDisabled();
    });

    it("disables the Cancel button", () => {
      renderForm({ isSubmitting: true });

      expect(screen.getByRole("button", { name: /cancel/i })).toBeDisabled();
    });

    it("shows 'Creating…' on the submit button in create mode", () => {
      renderForm({ mode: "create", isSubmitting: true });

      expect(
        screen.getByRole("button", { name: /creating/i })
      ).toBeInTheDocument();
    });

    it("shows 'Saving…' on the submit button in edit mode", () => {
      renderForm({ mode: "edit", isSubmitting: true });

      expect(
        screen.getByRole("button", { name: /saving/i })
      ).toBeInTheDocument();
    });
  });
});

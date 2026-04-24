package commands

import (
	"encoding/json"
	"fmt"

	"github.com/freeloio/freelo-cli/internal/api/freelo"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func NewCustomFieldsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "custom-fields",
		Aliases: []string{"cf"},
		Short:   "Manage custom fields",
	}

	cmd.AddCommand(
		newCFTypesCmd(app),
		newCFListCmd(app),
		newCFCreateCmd(app),
		newCFRenameCmd(app),
		newCFDeleteCmd(app),
		newCFRestoreCmd(app),
		newCFSetValueCmd(app),
		newCFDeleteValueCmd(app),
		newCFEnumOptionsCmd(app),
		newCFEnumCreateCmd(app),
		newCFEnumEditCmd(app),
		newCFEnumDeleteCmd(app),
		newCFSetEnumValueCmd(app),
	)

	return cmd
}

// parseUUIDFlag parses a UUID string flag, surfacing a clear CLI error
// before the server would reject it.
func parseUUIDFlag(s, flag string) (uuid.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid --%s: not a UUID", flag)
	}
	return u, nil
}

func newCFTypesCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "types",
		Short: "List available custom field types",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			body, err := consumeAPIBody(app.FreeloClient.GetCustomFieldTypes(cmd.Context()))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
			var types any
			_ = json.Unmarshal(body, &types)
			out.OK(types, "", nil)
			return nil
		},
	}
}

func newCFListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List custom fields for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			if projectID == 0 {
				return fmt.Errorf("--project is required")
			}
			body, err := consumeAPIBody(app.FreeloClient.FindCustomFieldsByProject(cmd.Context(), projectID))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
			var fields any
			_ = json.Unmarshal(body, &fields)
			out.OK(fields, "", nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	return cmd
}

func newCFCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a custom field in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			name, _ := cmd.Flags().GetString("name")
			typeUUIDStr, _ := cmd.Flags().GetString("type-uuid")

			if projectID == 0 || name == "" || typeUUIDStr == "" {
				return fmt.Errorf("--project, --name, and --type-uuid are required")
			}

			typeUUID, err := parseUUIDFlag(typeUUIDStr, "type-uuid")
			if err != nil {
				return err
			}

			body := freelo.CreateCustomFieldJSONRequestBody{Name: name, Type: typeUUID}

			field, err := consumeAPIObject(app.FreeloClient.CreateCustomField(cmd.Context(), projectID, body))
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}
			out.OK(field, fmt.Sprintf("Custom field '%s' created", name), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	cmd.Flags().String("name", "", "Field name (required)")
	cmd.Flags().String("type-uuid", "", "Field type UUID (required, get from 'custom-fields types')")
	return cmd
}

func newCFRenameCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename <field-uuid>",
		Short: "Rename a custom field",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			fieldUUID, err := parseUUIDFlag(args[0], "field-uuid")
			if err != nil {
				return err
			}

			body := freelo.RenameCustomFieldJSONRequestBody{Name: name}
			field, err := consumeAPIObject(app.FreeloClient.RenameCustomField(cmd.Context(), fieldUUID, body))
			if err != nil {
				out.Err(err, "rename_failed", "")
				return err
			}
			out.OK(field, fmt.Sprintf("Field renamed to '%s'", name), nil)
			return nil
		},
	}
	cmd.Flags().String("name", "", "New name (required)")
	return cmd
}

func newCFDeleteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <field-uuid>",
		Short: "Delete a custom field",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			u, err := parseUUIDFlag(args[0], "field-uuid")
			if err != nil {
				return err
			}
			if _, err := consumeAPIObject(app.FreeloClient.DeleteCustomField(cmd.Context(), u)); err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}
			out.OK(map[string]any{"uuid": args[0], "deleted": true}, "Custom field deleted", nil)
			return nil
		},
	}
}

func newCFRestoreCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "restore <field-uuid>",
		Short: "Restore a deleted custom field",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			u, err := parseUUIDFlag(args[0], "field-uuid")
			if err != nil {
				return err
			}
			field, err := consumeAPIObject(app.FreeloClient.RestoreCustomField(cmd.Context(), u))
			if err != nil {
				out.Err(err, "restore_failed", "")
				return err
			}
			out.OK(field, "Custom field restored", nil)
			return nil
		},
	}
}

func newCFSetValueCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-value",
		Short: "Set a custom field value on a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			taskID, _ := cmd.Flags().GetInt("task")
			fieldUUIDStr, _ := cmd.Flags().GetString("field-uuid")
			value, _ := cmd.Flags().GetString("value")

			if taskID == 0 || fieldUUIDStr == "" || value == "" {
				return fmt.Errorf("--task, --field-uuid, and --value are required")
			}
			fieldUUID, err := parseUUIDFlag(fieldUUIDStr, "field-uuid")
			if err != nil {
				return err
			}

			body := freelo.AddOrEditCustomFieldValueJSONRequestBody{
				TaskId:          taskID,
				CustomFieldUuid: fieldUUID,
				Value:           value,
			}
			resp, err := consumeAPIObject(app.FreeloClient.AddOrEditCustomFieldValue(cmd.Context(), body))
			if err != nil {
				out.Err(err, "set_value_failed", "")
				return err
			}
			out.OK(resp, "Custom field value set", nil)
			return nil
		},
	}
	cmd.Flags().Int("task", 0, "Task ID (required)")
	cmd.Flags().String("field-uuid", "", "Custom field UUID (required)")
	cmd.Flags().String("value", "", "Value to set (required)")
	return cmd
}

func newCFDeleteValueCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete-value <value-uuid>",
		Short: "Delete a custom field value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			u, err := parseUUIDFlag(args[0], "value-uuid")
			if err != nil {
				return err
			}
			if _, err := consumeAPIObject(app.FreeloClient.DeleteCustomFieldValue(cmd.Context(), u)); err != nil {
				out.Err(err, "delete_value_failed", "")
				return err
			}
			out.OK(map[string]any{"uuid": args[0], "deleted": true}, "Value deleted", nil)
			return nil
		},
	}
}

func newCFEnumOptionsCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "enum-options <field-uuid>",
		Short: "List enum options for a custom field",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			u, err := parseUUIDFlag(args[0], "field-uuid")
			if err != nil {
				return err
			}
			body, rerr := consumeAPIBody(app.FreeloClient.GetEnumOptionsForCustomField(cmd.Context(), u))
			if rerr != nil {
				out.Err(rerr, "api_error", "")
				return rerr
			}
			var options any
			_ = json.Unmarshal(body, &options)
			out.OK(options, "", nil)
			return nil
		},
	}
}

func newCFEnumCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enum-create <field-uuid>",
		Short: "Create an enum option for a custom field",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			value, _ := cmd.Flags().GetString("value")
			if value == "" {
				// Accept the legacy --name flag as a synonym for --value so
				// existing scripts keep working.
				value, _ = cmd.Flags().GetString("name")
			}
			if value == "" {
				return fmt.Errorf("--value (or legacy --name) is required")
			}

			u, err := parseUUIDFlag(args[0], "field-uuid")
			if err != nil {
				return err
			}

			body := freelo.CreateEnumOptionJSONRequestBody{Value: value}

			option, err := consumeAPIObject(app.FreeloClient.CreateEnumOption(cmd.Context(), u, body))
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}
			out.OK(option, fmt.Sprintf("Enum option '%s' created", value), nil)
			return nil
		},
	}
	cmd.Flags().String("value", "", "Option value (required; --name accepted as legacy synonym)")
	cmd.Flags().String("name", "", "Deprecated: use --value")
	// The legacy --color flag was never honored by the API (the
	// CreateEnumOption schema has no color field). Dropped to match spec.
	return cmd
}

func newCFEnumEditCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enum-edit <enum-uuid>",
		Short: "Edit an enum option",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			value, _ := cmd.Flags().GetString("value")
			if value == "" {
				value, _ = cmd.Flags().GetString("name")
			}
			if value == "" {
				return fmt.Errorf("--value (or legacy --name) is required")
			}

			u, err := parseUUIDFlag(args[0], "enum-uuid")
			if err != nil {
				return err
			}

			body := freelo.EditEnumOptionJSONRequestBody{Value: value}
			option, err := consumeAPIObject(app.FreeloClient.EditEnumOption(cmd.Context(), u, body))
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}
			out.OK(option, "Enum option updated", nil)
			return nil
		},
	}
	cmd.Flags().String("value", "", "New value (required; --name accepted as legacy synonym)")
	cmd.Flags().String("name", "", "Deprecated: use --value")
	return cmd
}

func newCFEnumDeleteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enum-delete <enum-uuid>",
		Short: "Delete an enum option",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			force, _ := cmd.Flags().GetBool("force")

			u, err := parseUUIDFlag(args[0], "enum-uuid")
			if err != nil {
				return err
			}

			if force {
				if _, err := consumeAPIObject(app.FreeloClient.ForceDeleteEnumOption(cmd.Context(), u)); err != nil {
					out.Err(err, "delete_failed", "")
					return err
				}
			} else {
				if _, err := consumeAPIObject(app.FreeloClient.DeleteEnumOption(cmd.Context(), u)); err != nil {
					out.Err(err, "delete_failed", "If in use, try --force")
					return err
				}
			}
			out.OK(map[string]any{"uuid": args[0], "deleted": true}, "Enum option deleted", nil)
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Force delete even if in use")
	return cmd
}

func newCFSetEnumValueCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-enum-value",
		Short: "Set an enum custom field value on a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			taskID, _ := cmd.Flags().GetInt("task")
			fieldUUIDStr, _ := cmd.Flags().GetString("field-uuid")
			enumUUIDStr, _ := cmd.Flags().GetString("enum-uuid")

			if taskID == 0 || fieldUUIDStr == "" || enumUUIDStr == "" {
				return fmt.Errorf("--task, --field-uuid, and --enum-uuid are required")
			}
			fieldUUID, err := parseUUIDFlag(fieldUUIDStr, "field-uuid")
			if err != nil {
				return err
			}
			enumUUID, err := parseUUIDFlag(enumUUIDStr, "enum-uuid")
			if err != nil {
				return err
			}

			body := freelo.AddOrEditEnumValueJSONRequestBody{
				TaskId:          taskID,
				CustomFieldUuid: fieldUUID,
				Value:           enumUUID,
			}
			resp, err := consumeAPIObject(app.FreeloClient.AddOrEditEnumValue(cmd.Context(), body))
			if err != nil {
				out.Err(err, "set_enum_value_failed", "")
				return err
			}
			out.OK(resp, "Enum value set", nil)
			return nil
		},
	}
	cmd.Flags().Int("task", 0, "Task ID (required)")
	cmd.Flags().String("field-uuid", "", "Custom field UUID (required)")
	cmd.Flags().String("enum-uuid", "", "Enum option UUID (required)")
	return cmd
}

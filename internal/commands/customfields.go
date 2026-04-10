package commands

import (
	"encoding/json"
	"fmt"

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

func newCFTypesCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "types",
		Short: "List available custom field types",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			result, err := app.Client.Get("/custom-field/get-types")
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
			var types any
			_ = json.Unmarshal(result, &types)
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

			result, err := app.Client.Get(fmt.Sprintf("/custom-field/find-by-project/%d", projectID))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
			var fields any
			_ = json.Unmarshal(result, &fields)
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
			typeUUID, _ := cmd.Flags().GetString("type-uuid")

			if projectID == 0 || name == "" || typeUUID == "" {
				return fmt.Errorf("--project, --name, and --type-uuid are required")
			}

			result, err := app.Client.Post(fmt.Sprintf("/custom-field/create/%d", projectID), map[string]any{
				"name":      name,
				"type_uuid": typeUUID,
			})
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}
			var field map[string]any
			_ = json.Unmarshal(result, &field)
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

			result, err := app.Client.Post("/custom-field/rename/"+args[0], map[string]any{"name": name})
			if err != nil {
				out.Err(err, "rename_failed", "")
				return err
			}
			var field map[string]any
			_ = json.Unmarshal(result, &field)
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
			_, err := app.Client.Delete("/custom-field/delete/" + args[0])
			if err != nil {
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
			result, err := app.Client.Post("/custom-field/restore/"+args[0], nil)
			if err != nil {
				out.Err(err, "restore_failed", "")
				return err
			}
			var field map[string]any
			_ = json.Unmarshal(result, &field)
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
			fieldUUID, _ := cmd.Flags().GetString("field-uuid")
			value, _ := cmd.Flags().GetString("value")

			if taskID == 0 || fieldUUID == "" || value == "" {
				return fmt.Errorf("--task, --field-uuid, and --value are required")
			}

			result, err := app.Client.Post("/custom-field/add-or-edit-value", map[string]any{
				"task_id":           taskID,
				"custom_field_uuid": fieldUUID,
				"value":             value,
			})
			if err != nil {
				out.Err(err, "set_value_failed", "")
				return err
			}
			var resp any
			_ = json.Unmarshal(result, &resp)
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
			_, err := app.Client.Delete("/custom-field/delete-value/" + args[0])
			if err != nil {
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
			result, err := app.Client.Get("/custom-field-enum/get-for-custom-field/" + args[0])
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
			var options any
			_ = json.Unmarshal(result, &options)
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
			name, _ := cmd.Flags().GetString("name")
			color, _ := cmd.Flags().GetString("color")

			if name == "" {
				return fmt.Errorf("--name is required")
			}

			body := map[string]any{"name": name}
			if color != "" {
				body["color"] = color
			}

			result, err := app.Client.Post("/custom-field-enum/create/"+args[0], body)
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}
			var option map[string]any
			_ = json.Unmarshal(result, &option)
			out.OK(option, fmt.Sprintf("Enum option '%s' created", name), nil)
			return nil
		},
	}
	cmd.Flags().String("name", "", "Option name (required)")
	cmd.Flags().String("color", "", "Color hex code")
	return cmd
}

func newCFEnumEditCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enum-edit <enum-uuid>",
		Short: "Edit an enum option",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			body := map[string]any{}
			if name, _ := cmd.Flags().GetString("name"); name != "" {
				body["name"] = name
			}
			if color, _ := cmd.Flags().GetString("color"); color != "" {
				body["color"] = color
			}
			if len(body) == 0 {
				return fmt.Errorf("at least --name or --color is required")
			}

			result, err := app.Client.Post("/custom-field-enum/change/"+args[0], body)
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}
			var option map[string]any
			_ = json.Unmarshal(result, &option)
			out.OK(option, "Enum option updated", nil)
			return nil
		},
	}
	cmd.Flags().String("name", "", "New name")
	cmd.Flags().String("color", "", "New color")
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

			path := "/custom-field-enum/delete/" + args[0]
			if force {
				path = "/custom-field-enum/force-delete/" + args[0]
			}

			_, err := app.Client.Delete(path)
			if err != nil {
				out.Err(err, "delete_failed", "If in use, try --force")
				return err
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
			fieldUUID, _ := cmd.Flags().GetString("field-uuid")
			enumUUID, _ := cmd.Flags().GetString("enum-uuid")

			if taskID == 0 || fieldUUID == "" || enumUUID == "" {
				return fmt.Errorf("--task, --field-uuid, and --enum-uuid are required")
			}

			result, err := app.Client.Post("/custom-field/add-or-edit-enum-value", map[string]any{
				"task_id":                 taskID,
				"custom_field_uuid":       fieldUUID,
				"custom_field_enum_uuid":  enumUUID,
			})
			if err != nil {
				out.Err(err, "set_enum_value_failed", "")
				return err
			}
			var resp any
			_ = json.Unmarshal(result, &resp)
			out.OK(resp, "Enum value set", nil)
			return nil
		},
	}
	cmd.Flags().Int("task", 0, "Task ID (required)")
	cmd.Flags().String("field-uuid", "", "Custom field UUID (required)")
	cmd.Flags().String("enum-uuid", "", "Enum option UUID (required)")
	return cmd
}

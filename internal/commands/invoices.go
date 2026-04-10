package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func NewInvoicesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "invoices",
		Aliases: []string{"invoice"},
		Short:   "Manage invoices",
	}

	cmd.AddCommand(
		newInvoicesListCmd(app),
		newInvoicesShowCmd(app),
		newInvoicesMarkCmd(app),
	)

	return cmd
}

func newInvoicesListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issued invoices",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			page, _ := cmd.Flags().GetInt("page")

			path := "/issued-invoices"
			params := []string{}
			if projectID != 0 {
				params = append(params, fmt.Sprintf("projects_ids[]=%d", projectID))
			}
			if page > 0 {
				params = append(params, fmt.Sprintf("p=%d", page))
			}
			if len(params) > 0 {
				path += "?" + strings.Join(params, "&")
			}

			result, err := app.Client.Get(path)
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			invoices, _ := parsePaginatedItems(result)

			out.OK(invoices, fmt.Sprintf("%d invoices", len(invoices)), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Filter by project ID")
	cmd.Flags().Int("page", 0, "Page number")
	return cmd
}

func newInvoicesShowCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "show <invoice-id>",
		Short: "Show invoice detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			result, err := app.Client.Get("/issued-invoice/" + args[0])
			if err != nil {
				out.Err(err, "not_found", "")
				return err
			}
			var invoice map[string]any
			_ = json.Unmarshal(result, &invoice)
			out.OK(invoice, "", nil)
			return nil
		},
	}
}

func newInvoicesMarkCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mark-invoiced <invoice-id>",
		Short: "Mark invoice as invoiced",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			invoiceID := args[0]
			url, _ := cmd.Flags().GetString("url")
			subject, _ := cmd.Flags().GetString("subject")

			body := map[string]any{}
			if url != "" {
				body["url"] = url
			}
			if subject != "" {
				body["subject"] = subject
			}

			result, err := app.Client.Post("/issued-invoice/"+invoiceID+"/mark-as-invoiced", body)
			if err != nil {
				out.Err(err, "mark_failed", "")
				return err
			}
			var resp any
			_ = json.Unmarshal(result, &resp)
			out.OK(resp, fmt.Sprintf("Invoice %s marked as invoiced", invoiceID), nil)
			return nil
		},
	}
	cmd.Flags().String("url", "", "Invoice URL")
	cmd.Flags().String("subject", "", "Invoice subject")
	return cmd
}

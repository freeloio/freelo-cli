package commands

import (
	"fmt"

	freelo "github.com/freeloio/freelo-go/freeloapi"
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

			params := &freelo.GetIssuedInvoicesParams{}
			setProjectsFilter(&params.ProjectsIds, projectID)
			if err := setPageFilter(cmd, &params.P); err != nil {
				return err
			}

			body, err := consumeAPIBody(app.FreeloClient.GetIssuedInvoices(cmd.Context(), params))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			invoices, _ := parsePaginatedItems(body)
			out.OK(invoices, fmt.Sprintf("%d invoices", len(invoices)), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Filter by project ID")
	cmd.Flags().Int("page", 0, "Page number (>= 1; omit for first page)")
	return cmd
}

func newInvoicesShowCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "show <invoice-id>",
		Short: "Show invoice detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			invoiceID, err := parseIntArg(args[0], "invoice-id")
			if err != nil {
				return err
			}
			invoice, err := consumeAPIObject(app.FreeloClient.GetIssuedInvoiceDetail(cmd.Context(), invoiceID))
			if err != nil {
				out.Err(err, "not_found", "")
				return err
			}
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
			invoiceID, err := parseIntArg(args[0], "invoice-id")
			if err != nil {
				return err
			}
			url, _ := cmd.Flags().GetString("url")
			subject, _ := cmd.Flags().GetString("subject")

			// Spec marks both fields as required non-pointer strings —
			// catch empty values at CLI layer with a clear message rather
			// than letting the server 400 on a malformed payload.
			if url == "" || subject == "" {
				return fmt.Errorf("--url and --subject are required")
			}

			body := freelo.MarkAsInvoicedJSONRequestBody{Url: url, Subject: subject}
			resp, err := consumeAPIObject(app.FreeloClient.MarkAsInvoiced(cmd.Context(), invoiceID, body))
			if err != nil {
				out.Err(err, "mark_failed", "")
				return err
			}
			out.OK(resp, fmt.Sprintf("Invoice %d marked as invoiced", invoiceID), nil)
			return nil
		},
	}
	cmd.Flags().String("url", "", "Invoice URL (required)")
	cmd.Flags().String("subject", "", "Invoice subject (required)")
	return cmd
}

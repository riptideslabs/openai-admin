package cmd

import (
	"context"
	"fmt"
	"strconv"
	"text/tabwriter"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/cobra"
)

type Project struct {
	ID         string `json:"id"`
	Object     string `json:"object,omitempty"`
	Name       string `json:"name"`
	CreatedAt  int64  `json:"created_at,omitempty"`
	ArchivedAt *int64 `json:"archived_at"`
	Status     string `json:"status,omitempty"`
}

type ProjectListPage struct {
	Object  string    `json:"object,omitempty"`
	Data    []Project `json:"data"`
	FirstID string    `json:"first_id,omitempty"`
	LastID  string    `json:"last_id,omitempty"`
	HasMore bool      `json:"has_more"`
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage projects",
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		includeArchived, err := cmd.Flags().GetBool("include-archived")
		if err != nil {
			return err
		}
		limit, err := cmd.Flags().GetInt("limit")
		if err != nil {
			return err
		}
		if limit <= 0 || limit > 100 {
			limit = 100
		}

		out := cmd.OutOrStdout()
		tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		defer tw.Flush()

		fmt.Fprintln(tw, "ID\tNAME\tSTATUS\tCREATED_AT\tARCHIVED_AT")

		client := openai.NewClient()
		after := ""
		for {
			var page ProjectListPage
			opts := []option.RequestOption{
				option.WithQuery("limit", strconv.Itoa(limit)),
				option.WithQuery("include_archived", strconv.FormatBool(includeArchived)),
			}
			if after != "" {
				opts = append(opts, option.WithQuery("after", after))
			}

			err := client.Get(ctx, "/organization/projects", nil, &page, opts...)
			if err != nil {
				return err
			}

			for _, p := range page.Data {
				fmt.Fprintf(
					tw,
					"%s\t%s\t%s\t%s\t%s\n",
					p.ID,
					p.Name,
					p.Status,
					formatUnixSeconds(p.CreatedAt),
					formatUnixSecondsPtr(p.ArchivedAt),
				)
			}

			if !page.HasMore || len(page.Data) == 0 {
				break
			}
			if page.LastID != "" {
				after = page.LastID
			} else {
				after = page.Data[len(page.Data)-1].ID
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(projectsListCmd)
	projectsListCmd.Flags().Bool("include-archived", false, "Include archived projects")
	projectsListCmd.Flags().Int("limit", 100, "Max projects per page (1-100)")
}

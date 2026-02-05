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

type Organization struct {
	Object        string  `json:"object,omitempty"`
	ID            string  `json:"id"`
	Created       int64   `json:"created,omitempty"`
	Description   string  `json:"description,omitempty"`
	IsDefault     bool    `json:"is_default,omitempty"`
	IsSCIMManaged bool    `json:"is_scim_managed,omitempty"`
	Name          string  `json:"name,omitempty"`
	Personal      bool    `json:"personal,omitempty"`
	Role          string  `json:"role,omitempty"`
	Title         string  `json:"title,omitempty"`
	ParentOrgID   *string `json:"parent_org_id"`
}

type OrganizationListPage struct {
	Object  string         `json:"object,omitempty"`
	Data    []Organization `json:"data"`
	FirstID string         `json:"first_id,omitempty"`
	LastID  string         `json:"last_id,omitempty"`
	HasMore bool           `json:"has_more,omitempty"`
}

var organizationsCmd = &cobra.Command{
	Use:   "organizations",
	Short: "Manage organizations",
}

var organizationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List organizations",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
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

		fmt.Fprintln(tw, "DEFAULT\tID\tNAME\tTITLE\tPERSONAL\tROLE\tCREATED\tDESCRIPTION")

		client := openai.NewClient()
		after := ""
		for {
			var page OrganizationListPage
			opts := []option.RequestOption{option.WithQuery("limit", strconv.Itoa(limit))}
			if after != "" {
				opts = append(opts, option.WithQuery("after", after))
			}

			err := client.Get(ctx, "/organizations", nil, &page, opts...)
			if err != nil {
				return err
			}

			for _, o := range page.Data {
				fmt.Fprintf(
					tw,
					"%t\t%s\t%s\t%s\t%t\t%s\t%s\t%s\n",
					o.IsDefault,
					o.ID,
					o.Name,
					o.Title,
					o.Personal,
					o.Role,
					formatUnixSeconds(o.Created),
					o.Description,
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
	rootCmd.AddCommand(organizationsCmd)
	organizationsCmd.AddCommand(organizationsListCmd)
	organizationsListCmd.Flags().Int("limit", 100, "Max organizations per page (1-100)")
}

package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"text/tabwriter"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/cobra"
)

type ProjectAPIKey struct {
	Object        string `json:"object,omitempty"`
	ID            string `json:"id"`
	Name          string `json:"name,omitempty"`
	RedactedValue string `json:"redacted_value,omitempty"`
	CreatedAt     int64  `json:"created_at,omitempty"`
	LastUsedAt    *int64 `json:"last_used_at,omitempty"`
}

type ProjectAPIKeyListPage struct {
	Object  string          `json:"object,omitempty"`
	Data    []ProjectAPIKey `json:"data"`
	FirstID string          `json:"first_id,omitempty"`
	LastID  string          `json:"last_id,omitempty"`
	HasMore bool            `json:"has_more"`
}

type ProjectAPIKeyDeleteResponse struct {
	Object  string `json:"object,omitempty"`
	ID      string `json:"id,omitempty"`
	Deleted bool   `json:"deleted"`
}

func resolveProjectID(cmd *cobra.Command) (string, error) {
	projectID, err := cmd.Flags().GetString("project-id")
	if err != nil {
		return "", err
	}
	if projectID == "" {
		projectID = os.Getenv("OPENAI_PROJECT_ID")
	}
	if projectID == "" {
		return "", fmt.Errorf("--project-id is required (or set OPENAI_PROJECT_ID)")
	}
	return projectID, nil
}

var projectsAPIKeysCmd = &cobra.Command{
	Use:   "api-keys",
	Short: "Manage project API keys",
}

var projectsAPIKeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List project API keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		projectID, err := resolveProjectID(cmd)
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

		fmt.Fprintln(tw, "ID\tNAME\tCREATED_AT\tLAST_USED_AT\tREDACTED_VALUE")

		client := openai.NewClient()

		after := ""
		for {
			var page ProjectAPIKeyListPage
			path := "/organization/projects/" + url.PathEscape(projectID) + "/api_keys"
			opts := []option.RequestOption{option.WithQuery("limit", strconv.Itoa(limit))}
			if after != "" {
				opts = append(opts, option.WithQuery("after", after))
			}

			err := client.Get(ctx, path, nil, &page, opts...)
			if err != nil {
				return err
			}

			for _, k := range page.Data {
				fmt.Fprintf(
					tw,
					"%s\t%s\t%s\t%s\t%s\n",
					k.ID,
					k.Name,
					formatUnixSeconds(k.CreatedAt),
					formatUnixSecondsPtr(k.LastUsedAt),
					k.RedactedValue,
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

var projectsAPIKeysDeleteCmd = &cobra.Command{
	Use:   "delete <key_id>",
	Short: "Delete a project API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		projectID, err := resolveProjectID(cmd)
		if err != nil {
			return err
		}

		keyID := args[0]
		path := "/organization/projects/" + url.PathEscape(projectID) + "/api_keys/" + url.PathEscape(keyID)

		client := openai.NewClient()
		var res ProjectAPIKeyDeleteResponse
		err = client.Delete(ctx, path, nil, &res)
		if err != nil {
			return err
		}

		id := res.ID
		if id == "" {
			id = keyID
		}
		fmt.Fprintf(cmd.OutOrStdout(), "DELETED\t%t\t%s\n", res.Deleted, id)
		return nil
	},
}

func init() {
	projectsCmd.AddCommand(projectsAPIKeysCmd)
	projectsAPIKeysCmd.AddCommand(projectsAPIKeysListCmd)
	projectsAPIKeysCmd.AddCommand(projectsAPIKeysDeleteCmd)

	projectsAPIKeysCmd.PersistentFlags().String("project-id", "", "Project ID (or set OPENAI_PROJECT_ID)")
	projectsAPIKeysListCmd.Flags().Int("limit", 100, "Max API keys per page (1-100)")
}

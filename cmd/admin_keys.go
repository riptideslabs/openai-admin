package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/cobra"
)

type AdminKey struct {
	Object        string         `json:"object,omitempty"`
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	RedactedValue string         `json:"redacted_value,omitempty"`
	CreatedAt     int64          `json:"created_at,omitempty"`
	LastUsedAt    *int64         `json:"last_used_at,omitempty"`
	Owner         *AdminKeyOwner `json:"owner,omitempty"`
}

type AdminKeyOwner struct {
	Type      string `json:"type,omitempty"`
	Object    string `json:"object,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
	Role      string `json:"role,omitempty"`
}

type AdminKeyListPage struct {
	Object  string     `json:"object,omitempty"`
	Data    []AdminKey `json:"data"`
	FirstID string     `json:"first_id,omitempty"`
	LastID  string     `json:"last_id,omitempty"`
	HasMore bool       `json:"has_more"`
}

type AdminKeyDeleteResponse struct {
	Object  string `json:"object,omitempty"`
	ID      string `json:"id,omitempty"`
	Deleted bool   `json:"deleted"`
}

type AdminKeyCreateParams struct {
	Name string `json:"name"`
}

type AdminKeyCreateResponse struct {
	AdminKey
	Value  string `json:"value,omitempty"`
	Token  string `json:"token,omitempty"`
	Key    string `json:"key,omitempty"`
	APIKey string `json:"api_key,omitempty"`
}

func (r AdminKeyCreateResponse) TokenValue() string {
	if r.Value != "" {
		return r.Value
	}
	if r.Token != "" {
		return r.Token
	}
	if r.Key != "" {
		return r.Key
	}
	return r.APIKey
}

func formatUnixSeconds(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func formatUnixSecondsPtr(ts *int64) string {
	if ts == nil {
		return ""
	}
	return formatUnixSeconds(*ts)
}

var adminKeysCmd = &cobra.Command{
	Use:   "admin-keys",
	Short: "Manage admin API keys",
}

var adminKeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List admin API keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		out := cmd.OutOrStdout()
		tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		defer tw.Flush()

		fmt.Fprintln(tw, "ID\tNAME\tIS_ADMIN\tCREATED_AT\tLAST_USED_AT\tOWNER_TYPE\tOWNER_ROLE\tOWNER_NAME")

		client := openai.NewClient()

		after := ""
		for {
			var page AdminKeyListPage
			opts := []option.RequestOption{option.WithQuery("limit", strconv.Itoa(100))}
			if after != "" {
				opts = append(opts, option.WithQuery("after", after))
			}

			err := client.Get(ctx, "/organization/admin_api_keys", nil, &page, opts...)
			if err != nil {
				return err
			}

			for _, k := range page.Data {
				ownerType := ""
				ownerRole := ""
				ownerName := ""
				if k.Owner != nil {
					ownerType = k.Owner.Type
					ownerRole = k.Owner.Role
					ownerName = k.Owner.Name
				}

				isAdmin := strings.HasPrefix(k.RedactedValue, "sk-admin")

				fmt.Fprintf(
					tw,
					"%s\t%s\t%t\t%s\t%s\t%s\t%s\t%s\n",
					k.ID,
					k.Name,
					isAdmin,
					formatUnixSeconds(k.CreatedAt),
					formatUnixSecondsPtr(k.LastUsedAt),
					ownerType,
					ownerRole,
					ownerName,
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

var adminKeysDeleteCmd = &cobra.Command{
	Use:   "delete <key_id>",
	Short: "Delete an admin API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		client := openai.NewClient()
		keyID := args[0]
		path := "/organization/admin_api_keys/" + url.PathEscape(keyID)

		var res AdminKeyDeleteResponse
		err := client.Delete(ctx, path, nil, &res)
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

var adminKeysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an admin API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("--name is required")
		}

		params := AdminKeyCreateParams{Name: name}

		client := openai.NewClient()
		var created AdminKeyCreateResponse
		err = client.Post(ctx, "/organization/admin_api_keys", params, &created)
		if err != nil {
			return err
		}

		value := created.TokenValue()
		if value == "" {
			return fmt.Errorf("create response did not include a key value (expected one of: value, token, key, api_key)")
		}

		fmt.Fprintln(cmd.OutOrStdout(), value)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(adminKeysCmd)
	adminKeysCmd.AddCommand(adminKeysListCmd)
	adminKeysCreateCmd.Flags().String("name", "", "Name for the admin API key")
	adminKeysCmd.AddCommand(adminKeysCreateCmd)
	adminKeysCmd.AddCommand(adminKeysDeleteCmd)
}

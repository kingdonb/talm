// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aenix-io/talm/pkg/engine"
	"github.com/spf13/cobra"

	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/constants"
)

var templateCmdFlags struct {
	insecure          bool
	valueFiles        []string // --values
	templateFiles     []string // -t/--template
	stringValues      []string // --set-string
	values            []string // --set
	fileValues        []string // --set-file
	jsonValues        []string // --set-json
	literalValues     []string // --set-literal
	talosVersion      string
	withSecrets       string
	full              bool
	offline           bool
	kubernetesVersion string
}

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Render templates locally and display the output",
	Long:  ``,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if templateCmdFlags.offline {
			return template(args)(context.Background(), nil)
		}
		if templateCmdFlags.insecure {
			return WithClientMaintenance(nil, template(args))
		}

		return WithClient(template(args))
	},
}

func template(args []string) func(ctx context.Context, c *client.Client) error {

	return func(ctx context.Context, c *client.Client) error {
		opts := engine.Options{
			Insecure:          templateCmdFlags.insecure,
			ValueFiles:        templateCmdFlags.valueFiles,
			StringValues:      templateCmdFlags.stringValues,
			Values:            templateCmdFlags.values,
			FileValues:        templateCmdFlags.fileValues,
			JsonValues:        templateCmdFlags.jsonValues,
			LiteralValues:     templateCmdFlags.literalValues,
			TalosVersion:      templateCmdFlags.talosVersion,
			WithSecrets:       templateCmdFlags.withSecrets,
			Full:              templateCmdFlags.full,
			Root:              Config.RootDir,
			Offline:           templateCmdFlags.offline,
			KubernetesVersion: templateCmdFlags.kubernetesVersion,
			TemplateFiles:     templateCmdFlags.templateFiles,
		}

		result, err := engine.Render(ctx, c, opts)
		if err != nil {
			return fmt.Errorf("failed to render templates: %w", err)
		}

		modeline, err := generateModeline(args)
		if err != nil {
			return fmt.Errorf("failed generate modeline: %w", err)
		}

		// Print the result to the standard output
		fmt.Printf("%s\n%s", modeline, string(result))

		return nil
	}
}

func init() {
	templateCmd.Flags().BoolVarP(&templateCmdFlags.insecure, "insecure", "i", false, "template using the insecure (encrypted with no auth) maintenance service")
	templateCmd.Flags().StringSliceVarP(&templateCmdFlags.valueFiles, "values", "", []string{}, "specify values in a YAML file (can specify multiple)")
	templateCmd.Flags().StringSliceVarP(&templateCmdFlags.templateFiles, "template", "t", []string{}, "specify templates to rendered manifest from (can specify multiple)")
	templateCmd.Flags().StringArrayVar(&templateCmdFlags.values, "set", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	templateCmd.Flags().StringArrayVar(&templateCmdFlags.stringValues, "set-string", []string{}, "set STRING values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	templateCmd.Flags().StringArrayVar(&templateCmdFlags.fileValues, "set-file", []string{}, "set values from respective files specified via the command line (can specify multiple or separate values with commas: key1=path1,key2=path2)")
	templateCmd.Flags().StringArrayVar(&templateCmdFlags.jsonValues, "set-json", []string{}, "set JSON values on the command line (can specify multiple or separate values with commas: key1=jsonval1,key2=jsonval2)")
	templateCmd.Flags().StringArrayVar(&templateCmdFlags.literalValues, "set-literal", []string{}, "set a literal STRING value on the command line")
	templateCmd.Flags().StringVar(&templateCmdFlags.talosVersion, "talos-version", "", "the desired Talos version to generate config for (backwards compatibility, e.g. v0.8)")
	templateCmd.Flags().StringVar(&templateCmdFlags.withSecrets, "with-secrets", "", "use a secrets file generated using 'gen secrets'")
	templateCmd.Flags().BoolVarP(&templateCmdFlags.full, "full", "", false, "show full resulting config, not only patch")
	templateCmd.Flags().BoolVarP(&templateCmdFlags.offline, "offline", "", false, "disable gathering information and lookup functions")
	templateCmd.Flags().StringVar(&templateCmdFlags.kubernetesVersion, "kubernetes-version", constants.DefaultKubernetesVersion, "desired kubernetes version to run")

	templateCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		templateCmdFlags.valueFiles = append(Config.TemplateOptions.ValueFiles, templateCmdFlags.valueFiles...)
		templateCmdFlags.values = append(Config.TemplateOptions.Values, templateCmdFlags.values...)
		templateCmdFlags.stringValues = append(Config.TemplateOptions.StringValues, templateCmdFlags.stringValues...)
		templateCmdFlags.fileValues = append(Config.TemplateOptions.FileValues, templateCmdFlags.fileValues...)
		templateCmdFlags.jsonValues = append(Config.TemplateOptions.JsonValues, templateCmdFlags.jsonValues...)
		templateCmdFlags.literalValues = append(Config.TemplateOptions.LiteralValues, templateCmdFlags.literalValues...)
		if !cmd.Flags().Changed("talos-version") {
			templateCmdFlags.talosVersion = Config.TemplateOptions.TalosVersion
		}
		if !cmd.Flags().Changed("with-secrets") {
			templateCmdFlags.withSecrets = Config.TemplateOptions.WithSecrets
		}
		if !cmd.Flags().Changed("kubernetes-version") {
			templateCmdFlags.kubernetesVersion = Config.TemplateOptions.KubernetesVersion
		}
		if !cmd.Flags().Changed("full") {
			templateCmdFlags.full = Config.TemplateOptions.Full
		}
		if !cmd.Flags().Changed("offline") {
			templateCmdFlags.offline = Config.TemplateOptions.Offline
		}
		return nil
	}

	addCommand(templateCmd)
}

// generateModeline creates a modeline string using JSON formatting for values
func generateModeline(templates []string) (string, error) {
	// Convert Nodes to JSON
	nodesJSON, err := json.Marshal(GlobalArgs.Nodes)
	if err != nil {
		return "", fmt.Errorf("failed to marshal nodes: %v", err)
	}

	// Convert Endpoints to JSON
	endpointsJSON, err := json.Marshal(GlobalArgs.Endpoints)
	if err != nil {
		return "", fmt.Errorf("failed to marshal endpoints: %v", err)
	}

	// Convert Templates to JSON
	templatesJSON, err := json.Marshal(templates)
	if err != nil {
		return "", fmt.Errorf("failed to marshal templates: %v", err)
	}

	// Form the final modeline string
	modeline := fmt.Sprintf(`# talm: nodes=%s, endpoints=%s, templates=%s`, string(nodesJSON), string(endpointsJSON), string(templatesJSON))
	return modeline, nil
}

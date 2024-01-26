// Copyright Mia srl
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/mia-platform/miactl/internal/client"
	"github.com/mia-platform/miactl/internal/clioptions"
	"github.com/mia-platform/miactl/internal/iam"
	"github.com/mia-platform/miactl/internal/resources"
	"github.com/spf13/cobra"
)

func EditCmd(options *clioptions.CLIOptions) *cobra.Command {
	validArgs := []string{iam.UsersEntityName, iam.GroupsEntityName, iam.ServiceAccountsEntityName}
	cmd := &cobra.Command{
		Use:       "edit " + "[" + strings.Join(validArgs, "|") + "]",
		Short:     "Edit the role of an IAM entity for a project or one of its environment",
		Long:      "Edit the role of an IAM entity for a project or one of its environment",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: validArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			restConfig, err := options.ToRESTConfig()
			cobra.CheckErr(err)
			client, err := client.APIClientForConfig(restConfig)
			cobra.CheckErr(err)
			changes := roleChanges{
				companyID:       restConfig.CompanyID,
				projectID:       restConfig.ProjectID,
				entityID:        options.EntityID,
				entityType:      args[0],
				environmentName: options.Environment,
				environmentRole: options.EnvironmentIAMRole,
				projectRole:     &options.ProjectIAMRole,
			}
			return editRoleForEntity(cmd.Context(), client, changes)
		},
	}

	options.AddEditCompanyIAMFlags(cmd.Flags())
	cmd.MarkFlagsRequiredTogether("environment-role", "environment")
	cmd.MarkFlagsMutuallyExclusive("environment-role", "project-role")

	if err := cmd.RegisterFlagCompletionFunc("project-role", resources.IAMRoleCompletion(true)); err != nil {
		// we panic here because if we reach here, something nasty is happening in flag autocomplete registration
		panic(err)
	}

	if err := cmd.RegisterFlagCompletionFunc("environment-role", resources.IAMEnvironmentRoleCompletion()); err != nil {
		// we panic here because if we reach here, something nasty is happening in flag autocomplete registration
		panic(err)
	}

	return cmd
}

func editRoleForEntity(ctx context.Context, client *client.APIClient, changes roleChanges) error {
	if len(changes.companyID) == 0 {
		return fmt.Errorf("missing company id, please set one with the flag or context")
	}

	if len(changes.projectID) == 0 {
		return fmt.Errorf("missing project id, please set one with the flag or context")
	}

	if len(changes.entityID) == 0 {
		return fmt.Errorf("missing entity id, please set one with the flag")
	}

	if changes.projectRole != nil && len(*changes.projectRole) > 0 && !resources.IsValidIAMRole(resources.IAMRole(*changes.projectRole), true) {
		return fmt.Errorf("invalid role for project: %s", *changes.projectRole)
	}

	if len(changes.environmentRole) > 0 && !resources.IsValidEnvironmentRole(resources.IAMRole(changes.environmentRole)) {
		return fmt.Errorf("invalid role for environment: %s", changes.environmentRole)
	}

	payload := payloadForChanges(changes)

	resp, err := iam.EditIAMResourceRole(ctx, client, changes.companyID, changes.entityID, changes.entityType, payload)
	if err != nil {
		return fmt.Errorf("error executing request: %w", err)
	}

	if err := resp.Error(); err != nil {
		return err
	}

	fmt.Printf("%s %s role successfully updated\n", changes.entityType, changes.entityID)
	return nil
}

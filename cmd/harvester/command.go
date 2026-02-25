/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package harvester

import (
	"fmt"

	"github.com/konstructio/kubefirst/internal/catalog"
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/provision"
	"github.com/konstructio/kubefirst/internal/step"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/spf13/cobra"
)

var (
	// Supported git providers
	supportedGitProviders        = []string{"github", "gitlab"}
	supportedGitProtocolOverride = []string{"https", "ssh"}
)

func NewCommand() *cobra.Command {
	harvesterCmd := &cobra.Command{
		Use:   "harvester",
		Short: "kubefirst Harvester installation",
		Long:  "kubefirst Harvester cluster installation using existing kubeconfig",
	}

	// on error, doesnt show helper/usage
	harvesterCmd.SilenceUsage = true

	// wire up new commands
	harvesterCmd.AddCommand(Create(), Destroy(), RootCredentials())

	return harvesterCmd
}

func Create() *cobra.Command{
	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "create the kubefirst platform on Harvester",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cloudProvider := "harvester"
			estimatedTimeMin := 25
			ctx := cmd.Context()
			stepper := step.NewStepFactory(cmd.ErrOrStderr())

			stepper.DisplayLogHints(cloudProvider, estimatedTimeMin)

			stepper.NewProgressStep("Validate Configuration")

			cliFlags, err := utilities.GetFlags(cmd, cloudProvider)
			if err != nil {
				wrerr := fmt.Errorf("failed to get flags: %w", err)
				stepper.FailCurrentStep(wrerr)
				return wrerr
			}

			_, catalogApps, err := catalog.ValidateCatalogApps(ctx, cliFlags.InstallCatalogApps)
			if err != nil {
				wrerr := fmt.Errorf("validation of catalog apps failed: %w", err)
				stepper.FailCurrentStep(wrerr)
				return wrerr
			}

			err = ValidateProvidedFlags(cliFlags.GitProvider)
			if err != nil {
				wrerr := fmt.Errorf("provided flags validation failed: %w", err)
				stepper.FailCurrentStep(wrerr)
				return wrerr
			}

			stepper.CompleteCurrentStep()
			clusterClient := cluster.Client{}

			provision := provision.NewProvisioner(provision.NewProvisionWatcher(cliFlags.ClusterName, &clusterClient), stepper)

			if err := provision.ProvisionManagementCluster(ctx, cliFlags, catalogApps); err != nil {
				return fmt.Errorf("failed to create harvester management cluster: %w", err)
			}

			return nil
		},
	}

	// Harvester-specific flags
	createCmd.Flags().String("kubeconfig-path", "$HOME/.kube/harvester.yaml", "path to Harvester kubeconfig file")
	createCmd.Flags().String("alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().Bool("ci", false, "if running kubefirst in ci, set this flag to disable interactive features")
	createCmd.Flags().String("cloud-region", "on-premise", "NOT USED, PRESENT FOR COMPATIBILITY")
	createCmd.Flags().String("node-type", "on-premise", "NOT USED, PRESENT FOR COMPATIBILITY")
	createCmd.Flags().String("node-count", "1", "NOT USED, PRESENT FOR COMPATIBILITY")
	createCmd.Flags().String("cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().String("cluster-type", "mgmt", "the type of cluster to create (mgmt|workload)")
	createCmd.Flags().String("dns-provider", "cloudflare", "DNS provider - one of: cloudflare")
	createCmd.Flags().String("domain-name", "dittmanfamily.com", "the domain name for your cluster")
	createCmd.Flags().String("git-provider", "github", "git provider - one of: github, gitlab")
	createCmd.Flags().String("git-protocol", "ssh", "git protocol - one of: https, ssh")
	createCmd.Flags().String("github-org", "holybitsllc", "the GitHub organization for the new GitOps repository - required if using GitHub")
	createCmd.Flags().String("gitlab-group", "", "the GitLab group for the new GitOps project - required if using GitLab")
	createCmd.Flags().String("gitops-template-url", "https://github.com/konstructio/gitops-template.git", "the fully qualified url to the gitops-template repository")
	createCmd.Flags().String("gitops-template-branch", "", "the branch to use for the gitops-template repository")
	createCmd.Flags().String("install-catalog-apps", "", "comma separated values to install after provision")
	createCmd.Flags().Bool("install-kubefirst-pro", true, "whether or not to install kubefirst pro")
	createCmd.Flags().String("lb-ip-range", "10.0.12.0/24", "IP range for Harvester load balancer pool")

	// vCluster flags
	createCmd.Flags().StringSlice("vclusters", []string{"dev", "test", "prod"}, "comma-separated list of vCluster environments to create")
	
	// Istio/Gateway flags
	createCmd.Flags().Bool("install-istio", true, "install Istio in ambient mode")
	createCmd.Flags().String("istio-version", "latest", "version of Istio to install")
	createCmd.Flags().Bool("install-kgateway", true, "install Kubernetes Gateway API and Kgateway")

	// Git repository flags
	createCmd.Flags().String("gitops-repo", "harvester-argo", "name of the GitOps repository")

	// UniFi ingress flags
	createCmd.Flags().String("unifi-host", "", "UniFi controller host/IP for port-forward and SSL cert upload (e.g. 192.168.1.1)")
	createCmd.Flags().String("unifi-user", "admin", "UniFi controller username")
	createCmd.Flags().String("unifi-password", "", "UniFi controller password")

	// Staged provisioning — stop cleanly after the named phase:
	//   argocd   → ArgoCD installed + registry app deployed
	//   ingress  → Cloudflare DNS + UniFi port-forward live
	//   vcluster → platform-vcluster ArgoCD app Healthy/Synced
	//   vault    → vault ArgoCD app Healthy/Synced
	createCmd.Flags().String("stop-after", "", "halt provisioning after phase: argocd|ingress|vcluster|vault")

	return createCmd
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroy the kubefirst platform on Harvester",
		Long:  "destroy the kubefirst platform running on Harvester and remove all resources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Destroy implementation
			return fmt.Errorf("destroy command not yet implemented")
		},
	}

	return destroyCmd
}

func RootCredentials() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "root-credentials",
		Short: "retrieve root credentials for Harvester cluster",
		Long:  "retrieve root authentication information for Harvester resources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Root credentials implementation
			return fmt.Errorf("root-credentials command not yet implemented")
		},
	}

	return authCmd
}

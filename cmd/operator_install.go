// Copyright © 2019 IBM Corporation and others.
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

package cmd

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

type operatorInstallCommandConfig struct {
	*operatorCommandConfig
	all        bool
	watchspace string
}

func newOperatorInstallCmd(operatorConfig *operatorCommandConfig) *cobra.Command {
	config := &operatorInstallCommandConfig{operatorCommandConfig: operatorConfig}
	// installCmd represents the "appsody deploy install" command
	var installCmd = &cobra.Command{
		Use:   "install",
		Short: "Install the Appsody Operator into the configured Kubernetes cluster",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return operatorInstall(config)
		},
	}

	installCmd.PersistentFlags().StringVarP(&config.watchspace, "watchspace", "w", "", "The namespace which the operator will watch.")
	installCmd.PersistentFlags().BoolVar(&config.all, "watch-all", false, "The operator will watch all namespaces.")
	return installCmd
}

func operatorInstall(config *operatorInstallCommandConfig) error {
	rootConfig := config.RootCommandConfig
	namespace := config.namespace
	watchspace := config.watchspace

	operatorNamespace := "default"
	watchNamespace := "''"
	if namespace != "" {
		operatorNamespace = namespace
	}
	if watchspace == "" {
		watchNamespace = operatorNamespace
	}
	if watchspace != "" {
		watchNamespace = watchspace
	}
	if config.all {
		watchNamespace = ""
	}
	rootConfig.Debug.log("watchNamespace is:  ", watchNamespace)
	operatorExists, existsErr := operatorExistsInNamespace(rootConfig, operatorNamespace, config.Dryrun)
	if existsErr != nil {
		return existsErr
	}
	if operatorExists {
		existingOperatorWatchspace, err := getOperatorWatchspace(rootConfig, operatorNamespace, config.Dryrun)
		if err != nil {
			rootConfig.Debug.log("Could not retrieve the watchspace of this operator - this should never happen...")
		}
		if existingOperatorWatchspace == "" {
			existingOperatorWatchspace = "all namespaces"
		}
		return errors.Errorf("An operator already exists in namespace %s and it is watching the %s namespace.", operatorNamespace, existingOperatorWatchspace)
	}

	watchExists, existingNamespace, watchExistsErr := operatorExistsWithWatchspace(rootConfig, watchNamespace, config.Dryrun)
	if watchExistsErr != nil {

		return watchExistsErr
	}
	if watchExists {
		return errors.Errorf("An operator watching namespace %s or all namespaces already exists in namespace %s", watchNamespace, existingNamespace)
	}

	deployConfigDir, err := getDeployConfigDir(rootConfig)
	if err != nil {
		return errors.Errorf("Error getting deploy config dir: %v", err)
	}

	var crdURL = getOperatorHome(rootConfig) + "/" + appsodyCRDName
	appsodyCRD := filepath.Join(deployConfigDir, appsodyCRDName)
	var file string

	file, err = downloadCRDYaml(rootConfig, crdURL, appsodyCRD)
	if err != nil {
		return err
	}

	err = KubeApply(rootConfig, file, config.namespace, config.Dryrun)
	if err != nil {
		return err
	}
	rbacYaml := filepath.Join(deployConfigDir, operatorRBACName)
	var rbacURL = getOperatorHome(rootConfig) + "/" + operatorRBACName
	if (operatorNamespace != watchNamespace) || config.all {
		rootConfig.Debug.log("Downloading: ", rbacURL)
		file, err = downloadRBACYaml(rootConfig, rbacURL, operatorNamespace, rbacYaml, config.Dryrun)
		if err != nil {
			return err
		}

		err = KubeApply(rootConfig, file, config.namespace, config.Dryrun)
		if err != nil {
			return err
		}
	}

	operatorYaml := filepath.Join(deployConfigDir, operatorYamlName)
	var operatorURL = getOperatorHome(rootConfig) + "/" + operatorYamlName
	file, err = downloadOperatorYaml(rootConfig, operatorURL, operatorNamespace, watchNamespace, operatorYaml)
	if err != nil {
		return err
	}

	err = KubeApply(rootConfig, file, config.namespace, config.Dryrun)
	if err != nil {
		return err
	}

	rootConfig.Info.log("Appsody operator deployed to Kubernetes")
	return nil
}

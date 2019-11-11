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
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

//RunKubeExec issues kubectl exec <arg>
func RunKubeExec(rootConfig *RootCommandConfig, args []string, dryrun bool) (string, error) {
	rootConfig.Info.log("Attempting to get resource from Kubernetes ...")
	kargs := []string{"exec"}
	kargs = append(kargs, args...)
	return RunKube(rootConfig, kargs, dryrun)

}

//RunKubeGet issues kubectl get <arg>
func RunKubeGet(rootConfig *RootCommandConfig, args []string, dryrun bool) (string, error) {
	rootConfig.Info.log("Attempting to get resource from Kubernetes ...")
	kargs := []string{"get"}
	kargs = append(kargs, args...)
	return RunKube(rootConfig, kargs, dryrun)

}

//RunKubeDelete issues kubectl delete <args>
func RunKubeDelete(rootConfig *RootCommandConfig, args []string, dryrun bool) (string, error) {
	rootConfig.Info.log("Attempting to delete resource from Kubernetes ...")
	kargs := []string{"delete"}
	kargs = append(kargs, args...)
	return RunKube(rootConfig, kargs, dryrun)
}

//RunKube runs a generic kubectl command
func RunKube(rootConfig *RootCommandConfig, kargs []string, dryrun bool) (string, error) {
	kcmd := "kubectl"
	if dryrun {
		rootConfig.Info.log("Dry run - skipping execution of: ", kcmd, " ", strings.Join(kargs, " "))
		return "", nil
	}
	rootConfig.Info.log("Running command: ", kcmd, " ", strings.Join(kargs, " "))
	execCmd := exec.Command(kcmd, kargs...)
	kout, kerr := execCmd.Output()

	if kerr != nil {
		return "", errors.Errorf("kubectl command failed: %s", string(kout[:]))
	}
	rootConfig.Debug.log("Command successful...")
	return string(kout[:]), nil
}

/*
func downloadOperatorYaml(url string, operatorNamespace string, watchNamespace string, target string) (string, error) {
	if dryrun {
		Info.log("Skipping download of operator yaml: ", url)
		return "", nil

	}
	file, err := downloadYaml(url, target)
	if err != nil {
		return "", fmt.Errorf("Could not download Operator YAML file %s", url)
	}

	yamlReader, err := ioutil.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.Errorf("Downloaded file does not exist %s. ", target)

		}
		return "", errors.Errorf("Failed reading file %s", target)

	}

	//output := bytes.Replace(yamlReader, []byte("APPSODY_OPERATOR_NAMESPACE"), []byte(operatorNamespace), -1)
	output := bytes.Replace(yamlReader, []byte("APPSODY_WATCH_NAMESPACE"), []byte(watchNamespace), -1)

	err = ioutil.WriteFile(target, output, 0666)
	if err != nil {
		return "", errors.Errorf("Failed to write local operator definition file: %s", err)
	}
	return target, nil
}

func downloadRBACYaml(url string, operatorNamespace string, target string) (string, error) {
	if dryrun {
		Info.log("Skipping download of RBAC yaml: ", url)
		return "", nil

	}
	file, err := downloadYaml(url, target)
	if err != nil {
		return "", fmt.Errorf("Could not download RBAC YAML file %s", url)
	}

	yamlReader, err := ioutil.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.Errorf("Downloaded file does not exist %s. ", target)

		}
		return "", errors.Errorf("Failed reading file %s", target)

	}

	output := bytes.Replace(yamlReader, []byte("APPSODY_OPERATOR_NAMESPACE"), []byte(operatorNamespace), -1)
	//output = bytes.Replace(output, []byte("APPSODY_WATCH_NAMESPACE"), []byte(watchNamespace), -1)

	err = ioutil.WriteFile(target, output, 0666)
	if err != nil {
		return "", errors.Errorf("Failed to write local operator definition file: %s", err)
	}
	return target, nil
}
*/
func operatorExistsInNamespace(rootConfig *RootCommandConfig, operatorNamespace string, dryrun bool) (bool, error) {

	// check to see if this namespace already has an appsody-operator
	//var args = []string{"deployment", "appsody-operator", "-n", operatorNamespace}
	var args = []string{"deployments", "-o=jsonpath='{.items[?(@.metadata.name==\"appsody-operator\")].metadata.namespace}'", "-n", operatorNamespace}

	getOutput, getErr := RunKubeGet(rootConfig, args, dryrun)
	if getErr != nil {
		rootConfig.Debug.log("Received an err: ", getErr)
		return false, getErr
	}
	getOutput = strings.Trim(getOutput, "'")
	if getOutput == "" {
		rootConfig.Info.log("There are no deployments with appsody-operator")
		return false, nil
	}
	return true, nil

}

// Check to see if any other operator is watching the watchNameSpace
func operatorExistsWithWatchspace(rootConfig *RootCommandConfig, watchNamespace string, dryrun bool) (bool, string, error) {
	rootConfig.Debug.log("Looking for an operator matching watchspace: ", watchNamespace)
	var namespacesWithOperatorsGetArgs = []string{"pods", "-o=jsonpath='{.items[?(@.metadata.labels.name==\"appsody-operator\")].metadata.namespace}’", "--all-namespaces"}
	getNamespacesOutput, getNamespacesErr := RunKubeGet(rootConfig, namespacesWithOperatorsGetArgs, dryrun)

	if getNamespacesErr != nil {
		return false, "", getNamespacesErr
	}
	getNamespacesOutput = strings.Trim(getNamespacesOutput, "'’\n")

	if getNamespacesOutput == "" {
		rootConfig.Info.log("There are no deployments with appsody-operator")
		return false, "", nil
	}
	if watchNamespace == "" && getNamespacesOutput != "" {
		watchAllErr := errors.Errorf("You specified --watch-all, but there are already instances of the appsody operator on the cluster")
		return true, "", watchAllErr
	}

	namespaces := strings.Split(getNamespacesOutput, " ")
	rootConfig.Debug.log("namespaces with operators: ", namespaces)
	for _, podNamespace := range namespaces {

		podWatchSpace, watchspaceErr := getOperatorWatchspace(rootConfig, podNamespace, dryrun)
		if watchspaceErr != nil {
			return false, "", watchspaceErr
		}
		// the operator is watching all namespaces
		if podWatchSpace == "" {
			rootConfig.Info.logf("An operator exists in namespace %s, that is watching all namespaces", podNamespace)
			return true, podNamespace, nil
		}
		// split the podwatchSpace by using new function
		// do the iff "" check first
		// loop around this if check
		watchSpaces := getWatchSpaces(podWatchSpace, dryrun)
		for _, value := range watchSpaces {
			if value == watchNamespace {
				rootConfig.Debug.logf("An operator that is watching namespace %s already exists in namespace %s", watchNamespace, podNamespace)
				return true, podNamespace, nil
			}
		}

	}
	return false, "", nil
}

/*

func operatorExistsWithWatchspace(watchNamespace string, dryrun bool) (bool, string, error) {
	Debug.log("Looking for an operator matching watchspace: ", watchNamespace)
	var deploymentsWithOperatorsGetArgs = []string{"deployments", "-o=jsonpath='{.items[?(@.metadata.name==\"appsody-operator\")].metadata.namespace}'", "--all-namespaces"}
	getOutput, getErr := RunKubeGet(deploymentsWithOperatorsGetArgs, dryrun)
	if getErr != nil {
		return false, "", getErr
	}
	getOutput = strings.Trim(getOutput, "'")
	if getOutput == "" {
		Info.log("There are no deployments with appsody-operator")
		return false, "", nil
	}
	if watchNamespace == "" && getOutput != "" {
		watchAllErr := errors.Errorf("You specified --watch-all, but there are already instances of the appsody operator on the cluster")
		return true, "", watchAllErr
	}
	deployments := strings.Split(getOutput, " ")
	Debug.log("deployments with operators: ", deployments)
	for _, deploymentNamespace := range deployments {
		var getDeploymentWatchNamespaceArgs = []string{"deployment", "-o=jsonpath='{.items[?(@.metadata.name==\"appsody-operator\")].spec.template.spec.containers[0].env[?(@.name==\"WATCH_NAMESPACE\")].value}'", "-n", deploymentNamespace}
		getOutput, getErr = RunKubeGet(getDeploymentWatchNamespaceArgs, dryrun)
		Debug.logf("Deployment: %s is watching namespace %s", deploymentNamespace, getOutput)
		if getErr != nil {
			return false, "", getErr
		}
		if strings.Trim(getOutput, "'") == watchNamespace {
			Debug.logf("An operator that is watching namespace %s already exists in namespace %s", watchNamespace, deploymentNamespace)
			return true, deploymentNamespace, nil
		}
		// the operator is watching all namespaces
		if strings.Trim(getOutput, "'") == "" {

			Info.logf("An operator exists in namespace %s, that is watching all namespaces", deploymentNamespace)
			return true, deploymentNamespace, nil
		}
	}
	return false, "", nil
}

*/

func operatorCount(rootConfig *RootCommandConfig, dryrun bool) (int, error) {
	var getAllOperatorsArgs = []string{"deployments", "-o=jsonpath='{.items[?(@.metadata.name==\"appsody-operator\")].metadata.name}'", "--all-namespaces"}
	getOutput, getErr := RunKubeGet(rootConfig, getAllOperatorsArgs, dryrun)
	if getErr != nil {
		return 0, getErr
	}
	return strings.Count(getOutput, "appsody-operator"), nil
}

func appsodyApplicationCount(rootConfig *RootCommandConfig, namespace string, dryrun bool) (int, error) {
	var getAppsodyAppsArgs = []string{"AppsodyApplication", "-o=jsonpath='{.items[*].kind}'"}
	if namespace == "" {
		getAppsodyAppsArgs = append(getAppsodyAppsArgs, "--all-namespaces")
	} else {
		getAppsodyAppsArgs = append(getAppsodyAppsArgs, "-n", namespace)
	}
	getOutput, getErr := RunKubeGet(rootConfig, getAppsodyAppsArgs, dryrun)
	if getErr != nil {
		return 0, getErr
	}
	return strings.Count(getOutput, "AppsodyApplication"), nil
}

func deleteAppsodyApps(rootConfig *RootCommandConfig, namespace string, dryrun bool) (string, error) {
	var deleteAppsodyAppsArgs = []string{"AppsodyApplication", "--all"}
	if namespace != "" {
		deleteAppsodyAppsArgs = append(deleteAppsodyAppsArgs, "-n", namespace)
	}
	return RunKubeDelete(rootConfig, deleteAppsodyAppsArgs, dryrun)

}

func getOperatorWatchspace(rootConfig *RootCommandConfig, namespace string, dryrun bool) (string, error) {
	operatorExists, existsErr := operatorExistsInNamespace(rootConfig, namespace, dryrun)
	if existsErr != nil {
		return "", existsErr
	}
	if !operatorExists {
		return "", errors.Errorf("An appsody operator could not be found in namespace: %s", namespace)
	}

	var getPodWatchNamespaceArgs = []string{"pod", "-o=jsonpath='{.items[?(@.metadata.labels.name==\"appsody-operator\")].metadata.name}'", "-n", namespace}

	getPodsOutput, getPodsErr := RunKubeGet(rootConfig, getPodWatchNamespaceArgs, dryrun)

	if getPodsErr != nil {
		return "", getPodsErr
	}
	// we should now have the pod name
	podName := strings.Trim(getPodsOutput, "'’\n")

	getWatchspaceArgs := []string{"-n", namespace, "-it", podName, "--", "/bin/printenv", "WATCH_NAMESPACE"}

	getWatchspaceOutput, getWatchspaceErr := RunKubeExec(rootConfig, getWatchspaceArgs, dryrun)

	if getWatchspaceErr != nil {
		return "", getWatchspaceErr
	}

	watchspaceForOperator := strings.Trim(getWatchspaceOutput, "'’\n")
	if watchspaceForOperator == "" {
		rootConfig.Debug.log("This operator watches the entire cluster ")
	}
	rootConfig.Debug.Logf("Pod: %s in namespace: %s is watching namespace: %s", podName, namespace, watchspaceForOperator)

	return watchspaceForOperator, nil
}

// create a function to parse the watchlist
func getWatchSpaces(csvList string, dryrun bool) []string {
	if csvList == "" {
		return nil
	}
	// split the string and clean up any issues
	watchList := strings.Split(csvList, ",")
	for index := range watchList {
		watchList[index] = strings.Trim(watchList[index], " '’\n")

	}
	return watchList
}

/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	examplev1alpha1 "github.com/example/rest-operator/api/v1alpha1"
	"github.com/example/rest-operator/test/utils"
)

const namespace = "rest-operator-system"
const endpoint = "https://jsonplaceholder.typicode.com/todos/1"

var _ = Describe("controller", Ordered, func() {
	BeforeAll(func() {
		By("installing prometheus operator")
		Expect(utils.InstallPrometheusOperator()).To(Succeed())

		By("installing the cert-manager")
		Expect(utils.InstallCertManager()).To(Succeed())

		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	AfterAll(func() {
		By("uninstalling the Prometheus manager bundle")
		utils.UninstallPrometheusOperator()

		By("uninstalling the cert-manager bundle")
		utils.UninstallCertManager()

		By("removing manager namespace")
		cmd := exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	Context("Operator", func() {
		It("should run successfully", func() {
			var controllerPodName string
			var err error

			// projectimage stores the name of the image used in the example
			var projectimage = "example.com/rest-operator:v0.0.1"

			By("building the manager(Operator) image")
			cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectimage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("loading the the manager(Operator) image on Kind")
			err = utils.LoadImageToKindClusterWithName(projectimage)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("installing CRDs")
			cmd = exec.Command("make", "install")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("deploying the controller-manager")
			cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectimage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func() error {
				// Get pod name

				cmd = exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(string(podOutput))
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

				// Validate pod status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				status, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if string(status) != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())

		})
	})

	Context("RestCall Resource", func() {
		It("should create and reconcile a RestCall resource", func() {
			ctx := context.Background()
			resourceName := "e2e-test-resource"
			namespace := "default"

			restCall := &examplev1alpha1.RestCall{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: examplev1alpha1.RestCallSpec{
					Endpoint: endpoint, // Use a valid endpoint for testing
				},
			}

			By("ensuring any existing RestCall resource is deleted")
			_ = utils.DeleteResource(ctx, restCall) // Ignore error if resource does not exist

			By("creating a RestCall resource")
			err := utils.CreateResource(ctx, restCall)
			Expect(err).NotTo(HaveOccurred())

			By("verifying the RestCall resource is created")
			fetched := &examplev1alpha1.RestCall{}
			Eventually(func() error {
				return utils.GetResource(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, fetched)
			}, time.Minute, time.Second).Should(Succeed())

			Expect(fetched.Spec.Endpoint).To(Equal(endpoint))

			By("verifying the RestCall resource is reconciled")
			Eventually(func() string {
				err := utils.GetResource(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, fetched)
				if err != nil {
					return ""
				}
				return fetched.Status.Response
			}, time.Minute, time.Second).ShouldNot(BeEmpty())

			By("deleting the RestCall resource")
			err = utils.DeleteResource(ctx, restCall)
			Expect(err).NotTo(HaveOccurred())

			By("verifying the RestCall resource is deleted")
			Eventually(func() error {
				return utils.GetResource(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, fetched)
			}, time.Minute, time.Second).ShouldNot(Succeed())
		})
	})

})

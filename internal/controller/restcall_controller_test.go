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

package controller

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	examplev1alpha1 "github.com/example/rest-operator/api/v1alpha1"
)

var _ = Describe("RestCall Controller", func() {
	var (
		server     *httptest.Server
		httpClient *http.Client
		reconciler *RestCallReconciler
		// logger     logr.Logger
	)

	BeforeEach(func() {
		// Setup a test HTTP server
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test response"))
		}))

		httpClient = server.Client()
		// logger := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))

		reconciler = &RestCallReconciler{
			Client:     k8sClient,
			Scheme:     k8sClient.Scheme(),
			HTTPClient: httpClient,
		}
	})

	AfterEach(func() {
		server.Close()
	})

	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		restcall := &examplev1alpha1.RestCall{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind RestCall")
			err := k8sClient.Get(ctx, typeNamespacedName, restcall)
			if err != nil && errors.IsNotFound(err) {
				resource := &examplev1alpha1.RestCall{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: examplev1alpha1.RestCallSpec{
						Endpoint: server.URL, // Set the endpoint to the test server's URL
						// Add any other necessary fields here
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &examplev1alpha1.RestCall{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance RestCall")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &RestCallReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})

		It("should update the RestCall status with the response", func() {
			By("Reconciling the created resource")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the status update")
			err = k8sClient.Get(ctx, typeNamespacedName, restcall)
			Expect(err).NotTo(HaveOccurred())
			Expect(restcall.Status.Response).To(Equal("test response"))
			Expect(restcall.Status.LastCallTime).NotTo(BeEmpty())
		})

		It("should handle HTTP request errors gracefully", func() {
			// Simulate a server error
			server.Close()

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
		})
	})
})

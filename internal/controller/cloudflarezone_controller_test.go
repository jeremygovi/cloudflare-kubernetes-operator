/*
Copyright 2024.

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	cloudflarev1 "github.com/jeremygovi/cloudflare-kubernetes-operator/api/v1"
)

var _ = Describe("CloudflareZone Controller", func() {
	Context("When creating a CloudflareZone", func() {
		It("Should successfully create the resource", func() {
			ctx := context.Background()

			zone := &cloudflarev1.CloudflareZone{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-zone",
					Namespace: "default",
				},
				Spec: cloudflarev1.CloudflareZoneSpec{
					Name:      "example.com",
					AccountID: "test-account-id",
					Type:      cloudflarev1.ZoneTypeFull,
				},
			}

			Expect(k8sClient.Create(ctx, zone)).Should(Succeed())

			zoneLookupKey := types.NamespacedName{Name: "test-zone", Namespace: "default"}
			createdZone := &cloudflarev1.CloudflareZone{}

			err := k8sClient.Get(ctx, zoneLookupKey, createdZone)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdZone.Spec.Name).Should(Equal("example.com"))
			Expect(createdZone.Spec.AccountID).Should(Equal("test-account-id"))
			Expect(createdZone.Spec.Type).Should(Equal(cloudflarev1.ZoneTypeFull))
		})
	})

	Context("When deleting a CloudflareZone", func() {
		It("Should successfully delete the resource", func() {
			ctx := context.Background()
			zoneLookupKey := types.NamespacedName{Name: "test-zone", Namespace: "default"}

			zone := &cloudflarev1.CloudflareZone{}
			err := k8sClient.Get(ctx, zoneLookupKey, zone)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Delete(ctx, zone)).Should(Succeed())
		})
	})
})

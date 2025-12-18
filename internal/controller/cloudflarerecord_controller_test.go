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

var _ = Describe("CloudflareRecord Controller", func() {
	Context("When creating a CloudflareRecord", func() {
		It("Should successfully create the resource", func() {
			ctx := context.Background()

			ttl := 300
			record := &cloudflarev1.CloudflareRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-record",
					Namespace: "default",
				},
				Spec: cloudflarev1.CloudflareRecordSpec{
					Domain:  "example.com",
					Name:    "test",
					Type:    cloudflarev1.DNSRecordTypeA,
					Content: "192.0.2.1",
					TTL:     &ttl,
				},
			}

			Expect(k8sClient.Create(ctx, record)).Should(Succeed())

			recordLookupKey := types.NamespacedName{Name: "test-record", Namespace: "default"}
			createdRecord := &cloudflarev1.CloudflareRecord{}

			err := k8sClient.Get(ctx, recordLookupKey, createdRecord)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdRecord.Spec.Domain).Should(Equal("example.com"))
			Expect(createdRecord.Spec.Name).Should(Equal("test"))
			Expect(createdRecord.Spec.Type).Should(Equal(cloudflarev1.DNSRecordTypeA))
		})
	})

	Context("When deleting a CloudflareRecord", func() {
		It("Should successfully delete the resource", func() {
			ctx := context.Background()
			recordLookupKey := types.NamespacedName{Name: "test-record", Namespace: "default"}

			record := &cloudflarev1.CloudflareRecord{}
			err := k8sClient.Get(ctx, recordLookupKey, record)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Delete(ctx, record)).Should(Succeed())
		})
	})
})

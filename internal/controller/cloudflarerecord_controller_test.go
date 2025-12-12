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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	cloudflarev1 "github.com/jeremygovi/cloudflare-kubernetes-operator/api/v1"
)

var _ = Describe("CloudflareRecord Controller", func() {
	const (
		CloudflareRecordName      = "test-record"
		CloudflareRecordNamespace = "default"
		timeout                   = time.Second * 10
		interval                  = time.Millisecond * 250
	)

	Context("When creating a CloudflareRecord", func() {
		It("Should add finalizer to the resource", func() {
			ctx := context.Background()

			record := &cloudflarev1.CloudflareRecord{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "cloudflare.k8s.io/v1",
					Kind:       "CloudflareRecord",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      CloudflareRecordName,
					Namespace: CloudflareRecordNamespace,
				},
				Spec: cloudflarev1.CloudflareRecordSpec{
					ZoneID:  "test-zone-id",
					Name:    "test.example.com",
					Type:    cloudflarev1.DNSRecordTypeA,
					Content: "192.0.2.1",
				},
			}

			Expect(k8sClient.Create(ctx, record)).Should(Succeed())

			recordLookupKey := types.NamespacedName{Name: CloudflareRecordName, Namespace: CloudflareRecordNamespace}
			createdRecord := &cloudflarev1.CloudflareRecord{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, recordLookupKey, createdRecord)
				if err != nil {
					return false
				}
				// Check if finalizer was added
				return len(createdRecord.Finalizers) > 0
			}, timeout, interval).Should(BeTrue())

			Expect(createdRecord.Finalizers).Should(ContainElement("cloudflare.k8s.io/record-finalizer"))
		})

		It("Should set initial status to Pending", func() {
			ctx := context.Background()
			recordLookupKey := types.NamespacedName{Name: CloudflareRecordName, Namespace: CloudflareRecordNamespace}
			createdRecord := &cloudflarev1.CloudflareRecord{}

			Eventually(func() cloudflarev1.RecordState {
				err := k8sClient.Get(ctx, recordLookupKey, createdRecord)
				if err != nil {
					return ""
				}
				return createdRecord.Status.State
			}, timeout, interval).Should(Equal(cloudflarev1.RecordStatePending))
		})
	})

	Context("When deleting a CloudflareRecord", func() {
		It("Should remove the finalizer and delete the resource", func() {
			ctx := context.Background()
			recordLookupKey := types.NamespacedName{Name: CloudflareRecordName, Namespace: CloudflareRecordNamespace}
			
			record := &cloudflarev1.CloudflareRecord{}
			err := k8sClient.Get(ctx, recordLookupKey, record)
			Expect(err).NotTo(HaveOccurred())

			By("Deleting the CloudflareRecord")
			Expect(k8sClient.Delete(ctx, record)).Should(Succeed())

			By("Expecting the resource to be deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, recordLookupKey, record)
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})

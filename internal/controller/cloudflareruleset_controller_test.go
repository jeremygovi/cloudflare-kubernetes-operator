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

var _ = Describe("CloudflareRuleset Controller", func() {
	Context("When creating a CloudflareRuleset", func() {
		It("Should successfully create the resource", func() {
			ctx := context.Background()

			enabled := true
			ruleset := &cloudflarev1.CloudflareRuleset{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ruleset",
					Namespace: "default",
				},
				Spec: cloudflarev1.CloudflareRulesetSpec{
					ZoneID:      "test-zone-id",
					Name:        "Test Ruleset",
					Description: "Test security ruleset",
					Phase:       cloudflarev1.RulesetPhaseHTTPRequestFirewallCustom,
					Rules: []cloudflarev1.Rule{
						{
							Action:      "block",
							Expression:  "(cf.threat_score gt 50)",
							Description: "Block high threat score",
							Enabled:     &enabled,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, ruleset)).Should(Succeed())

			rulesetLookupKey := types.NamespacedName{Name: "test-ruleset", Namespace: "default"}
			createdRuleset := &cloudflarev1.CloudflareRuleset{}

			err := k8sClient.Get(ctx, rulesetLookupKey, createdRuleset)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdRuleset.Spec.ZoneID).Should(Equal("test-zone-id"))
			Expect(createdRuleset.Spec.Phase).Should(Equal(cloudflarev1.RulesetPhaseHTTPRequestFirewallCustom))
			Expect(len(createdRuleset.Spec.Rules)).Should(Equal(1))
		})
	})

	Context("When deleting a CloudflareRuleset", func() {
		It("Should successfully delete the resource", func() {
			ctx := context.Background()
			rulesetLookupKey := types.NamespacedName{Name: "test-ruleset", Namespace: "default"}

			ruleset := &cloudflarev1.CloudflareRuleset{}
			err := k8sClient.Get(ctx, rulesetLookupKey, ruleset)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Delete(ctx, ruleset)).Should(Succeed())
		})
	})
})

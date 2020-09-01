/*
Copyright 2020 The Kubernetes Authors.

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

package eks

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"

	infrav1 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/converters"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/tags"
)

func (s *Service) reconcileTags(cluster *eks.Cluster) error {
	clusterTags := converters.MapPtrToMap(cluster.Tags)
	buildParams := s.getEKSTagParams(*cluster.Arn)
	tagsBuilder := tags.New(buildParams, tags.WithEKS(s.EKSClient))
	if err := tagsBuilder.Ensure(clusterTags); err != nil {
		return fmt.Errorf("failed ensuring tags on cluster: %w", err)
	}

	return nil
}

func (s *Service) getEKSTagParams(id string) *infrav1.BuildParams {
	name := s.scope.KubernetesClusterName()

	return &infrav1.BuildParams{
		ClusterName: s.scope.Name(),
		ResourceID:  id,
		Lifecycle:   infrav1.ResourceLifecycleOwned,
		Name:        aws.String(name),
		Role:        aws.String(infrav1.CommonRoleTagValue),
		Additional:  s.scope.AdditionalTags(),
	}
}

func getTagUpdates(currentTags map[string]string, tags map[string]string) (untagKeys []string, newTags map[string]string) {
	untagKeys = []string{}
	newTags = make(map[string]string)
	for key := range currentTags {
		if _, ok := tags[key]; !ok {
			untagKeys = append(untagKeys, key)
		}
	}
	for key, value := range tags {
		if currentV, ok := currentTags[key]; !ok || value != currentV {
			newTags[key] = value
		}
	}
	return untagKeys, newTags
}

func (s *NodegroupService) reconcileTags(ng *eks.Nodegroup) error {
	tags := ngTags(s.scope.Name(), s.scope.AdditionalTags())

	untagKeys, newTags := getTagUpdates(aws.StringValueMap(ng.Tags), tags)

	if len(newTags) > 0 {
		tagInput := &eks.TagResourceInput{
			ResourceArn: ng.NodegroupArn,
			Tags:        aws.StringMap(newTags),
		}
		_, err := s.EKSClient.TagResource(tagInput)
		if err != nil {
			return err
		}
	}

	if len(untagKeys) > 0 {
		untagInput := &eks.UntagResourceInput{
			ResourceArn: ng.NodegroupArn,
			TagKeys:     aws.StringSlice(untagKeys),
		}
		_, err := s.EKSClient.UntagResource(untagInput)
		if err != nil {
			return err
		}
	}

	return nil
}

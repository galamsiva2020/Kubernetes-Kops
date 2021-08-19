/*
Copyright 2019 The Kubernetes Authors.

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

package components

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/loader"

	"github.com/blang/semver/v4"
)

// KubeAPIServerOptionsBuilder adds options for the apiserver to the model
type KubeAPIServerOptionsBuilder struct {
	*OptionsContext
}

var _ loader.OptionsBuilder = &KubeAPIServerOptionsBuilder{}

// BuildOptions is responsible for filling in the default settings for the kube apiserver
func (b *KubeAPIServerOptionsBuilder) BuildOptions(o interface{}) error {
	clusterSpec := o.(*kops.ClusterSpec)
	if clusterSpec.KubeAPIServer == nil {
		clusterSpec.KubeAPIServer = &kops.KubeAPIServerConfig{}
	}
	c := clusterSpec.KubeAPIServer

	if c.APIServerCount == nil {
		count := b.buildAPIServerCount(clusterSpec)
		if count == 0 {
			return fmt.Errorf("no instance groups found")
		}
		c.APIServerCount = fi.Int32(int32(count))
	}

	// @question: should the question every be able to set this?
	if c.StorageBackend == nil {
		// @note: we can use the first version as we enforce both running the same versions.
		// albeit feels a little weird to do this
		sem, err := semver.Parse(strings.TrimPrefix(clusterSpec.EtcdClusters[0].Version, "v"))
		if err != nil {
			return err
		}
		c.StorageBackend = fi.String(fmt.Sprintf("etcd%d", sem.Major))
	}

	if c.KubeletPreferredAddressTypes == nil {
		// We prioritize the internal IP above the hostname
		c.KubeletPreferredAddressTypes = []string{
			string(v1.NodeInternalIP),
			string(v1.NodeHostName),
			string(v1.NodeExternalIP),
		}
	}

	if clusterSpec.Authentication != nil {
		if clusterSpec.Authentication.Kopeio != nil {
			c.AuthenticationTokenWebhookConfigFile = fi.String("/etc/kubernetes/authn.config")
		}
	}

	if clusterSpec.Authorization == nil || clusterSpec.Authorization.IsEmpty() {
		// Do nothing - use the default as defined by the apiserver
		// (this won't happen anyway because of our default logic)
	} else if clusterSpec.Authorization.AlwaysAllow != nil {
		clusterSpec.KubeAPIServer.AuthorizationMode = fi.String("AlwaysAllow")
	} else if clusterSpec.Authorization.RBAC != nil {
		var modes []string

		if b.IsKubernetesGTE("1.19") || fi.BoolValue(clusterSpec.KubeAPIServer.EnableBootstrapAuthToken) {
			// Enable the Node authorizer, used for special per-node RBAC policies
			// Enable by default from 1.19 - it's an important part of limiting blast radius
			modes = append(modes, "Node")
		}
		modes = append(modes, "RBAC")

		clusterSpec.KubeAPIServer.AuthorizationMode = fi.String(strings.Join(modes, ","))
	}

	if err := b.configureAggregation(clusterSpec); err != nil {
		return nil
	}

	image, err := Image("kube-apiserver", clusterSpec, b.AssetBuilder)
	if err != nil {
		return err
	}
	c.Image = image

	switch kops.CloudProviderID(clusterSpec.CloudProvider) {
	case kops.CloudProviderAWS:
		c.CloudProvider = "aws"
	case kops.CloudProviderGCE:
		c.CloudProvider = "gce"
	case kops.CloudProviderDO:
		c.CloudProvider = "external"
	case kops.CloudProviderOpenstack:
		c.CloudProvider = "openstack"
	case kops.CloudProviderALI:
		c.CloudProvider = "alicloud"
	case kops.CloudProviderAzure:
		c.CloudProvider = "azure"
	default:
		return fmt.Errorf("unknown cloudprovider %q", clusterSpec.CloudProvider)
	}

	if clusterSpec.ExternalCloudControllerManager != nil {
		c.CloudProvider = "external"
	}

	c.LogLevel = 2
	c.SecurePort = 443

	if clusterSpec.IsIPv6Only() {
		c.BindAddress = "::"
	} else {
		c.BindAddress = "0.0.0.0"
	}

	c.AllowPrivileged = fi.Bool(true)
	c.ServiceClusterIPRange = clusterSpec.ServiceClusterIPRange
	c.EtcdServers = nil
	c.EtcdServersOverrides = nil

	for _, etcdCluster := range clusterSpec.EtcdClusters {
		protocol := "http"
		if etcdCluster.EnableEtcdTLS {
			protocol = "https"
		}
		switch etcdCluster.Name {
		case "main":
			c.EtcdServers = append(c.EtcdServers, protocol+"://127.0.0.1:4001")
		case "events":
			c.EtcdServersOverrides = append(c.EtcdServersOverrides, "/events#"+protocol+"://127.0.0.1:4002")
		}
	}

	// TODO: We can probably rewrite these more clearly in descending order
	// Based on recommendations from:
	// https://kubernetes.io/docs/admin/admission-controllers/#is-there-a-recommended-set-of-admission-controllers-to-use
	{
		c.EnableAdmissionPlugins = []string{
			"NamespaceLifecycle",
			"LimitRanger",
			"ServiceAccount",
			"PersistentVolumeLabel",
			"DefaultStorageClass",
			"DefaultTolerationSeconds",
			"MutatingAdmissionWebhook",
			"ValidatingAdmissionWebhook",
			"NodeRestriction",
			"ResourceQuota",
		}
		c.EnableAdmissionPlugins = append(c.EnableAdmissionPlugins, c.AppendAdmissionPlugins...)
	}

	// We make sure to disable AnonymousAuth
	c.AnonymousAuth = fi.Bool(false)

	// We query via the kube-apiserver-healthcheck proxy, which listens on port 3990
	c.InsecureBindAddress = ""
	c.InsecurePort = 0

	return nil
}

// buildAPIServerCount calculates the count of the api servers, essentially the number of node marked as Master role
func (b *KubeAPIServerOptionsBuilder) buildAPIServerCount(clusterSpec *kops.ClusterSpec) int {
	// The --apiserver-count flag is (generally agreed) to be something we need to get rid of in k8s

	// We should do something like this:

	//count := 0
	//for _, ig := range b.InstanceGroups {
	//	if !ig.IsMaster() {
	//		continue
	//	}
	//	size := fi.IntValue(ig.Spec.MaxSize)
	//	if size == 0 {
	//		size = fi.IntValue(ig.Spec.MinSize)
	//	}
	//	count += size
	//}

	// But if we do, we end up with a weird dependency on InstanceGroups.  We actually could tolerate
	// that in kops, but we don't really want to.

	// So instead, we assume that the etcd cluster size is the API Server Count.
	// We can re-examine this when we allow separate etcd clusters - at which time hopefully
	// the flag won't exist

	counts := make(map[string]int)
	for _, etcdCluster := range clusterSpec.EtcdClusters {
		counts[etcdCluster.Name] = len(etcdCluster.Members)
	}

	count := counts["main"]

	return count
}

// configureAggregation sets up the aggregation options
func (b *KubeAPIServerOptionsBuilder) configureAggregation(clusterSpec *kops.ClusterSpec) error {
	clusterSpec.KubeAPIServer.RequestheaderAllowedNames = []string{"aggregator"}
	clusterSpec.KubeAPIServer.RequestheaderExtraHeaderPrefixes = []string{"X-Remote-Extra-"}
	clusterSpec.KubeAPIServer.RequestheaderGroupHeaders = []string{"X-Remote-Group"}
	clusterSpec.KubeAPIServer.RequestheaderUsernameHeaders = []string{"X-Remote-User"}

	return nil
}

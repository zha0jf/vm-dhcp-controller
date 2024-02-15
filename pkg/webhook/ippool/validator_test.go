package ippool

import (
	"fmt"
	"testing"

	"github.com/harvester/webhook/pkg/server/admission"
	cniv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"

	networkv1 "github.com/harvester/vm-dhcp-controller/pkg/apis/network.harvesterhci.io/v1alpha1"
	"github.com/harvester/vm-dhcp-controller/pkg/controller/ippool"
	"github.com/harvester/vm-dhcp-controller/pkg/generated/clientset/versioned/fake"
	"github.com/harvester/vm-dhcp-controller/pkg/util/fakeclient"
)

const (
	testNADNamespace       = "default"
	testNADName            = "net-1"
	testIPPoolNamespace    = testNADNamespace
	testIPPoolName         = testNADName
	testCIDR               = "192.168.0.0/24"
	testServerIPOutOfRange = "192.168.100.2"
	testRouter             = "192.168.0.1"
	testNetworkName        = testNADNamespace + "/" + testNADName
)

func newTestIPPoolBuilder() *ippool.IPPoolBuilder {
	return ippool.NewIPPoolBuilder(testIPPoolNamespace, testIPPoolName)
}

func newTestNetworkAttachmentDefinitionBuilder() *ippool.NetworkAttachmentDefinitionBuilder {
	return ippool.NewNetworkAttachmentDefinitionBuilder(testNADNamespace, testNADName)
}

func TestValidator_Create(t *testing.T) {
	type input struct {
		ipPool *networkv1.IPPool
		nad    *cniv1.NetworkAttachmentDefinition
	}

	type output struct {
		err error
	}

	testCases := []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "valid server ip",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					ServerIP("192.168.0.2").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
		},
		{
			name: "invalid server ip which is out of range",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					ServerIP("192.168.100.2").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because server ip %s is not within subnet", testIPPoolNamespace, testIPPoolName, testServerIPOutOfRange),
			},
		},
		{
			name: "invalid server ip which is the same as network ip",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.128/25").
					ServerIP("192.168.0.128").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because server ip %s cannot be the same as network ip", testIPPoolNamespace, testIPPoolName, "192.168.0.128"),
			},
		},
		{
			name: "invalid server ip which is the same as broadcast ip",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/25").
					ServerIP("192.168.0.127").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because server ip %s cannot be the same as broadcast ip", testIPPoolNamespace, testIPPoolName, "192.168.0.127"),
			},
		},
		{
			name: "invalid server ip which is the same as router ip",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.254/24").
					ServerIP("192.168.0.254").
					Router("192.168.0.254").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because server ip %s cannot be the same as router ip", testIPPoolNamespace, testIPPoolName, "192.168.0.254"),
			},
		},
		{
			name: "invalid router ip which is malformed",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					Router("192.168.0.1000").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because ParseAddr(\"%s\"): IPv4 field has value >255", testIPPoolNamespace, testIPPoolName, "192.168.0.1000"),
			},
		},
		{
			name: "invalid router ip which is out of subnet",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					Router("192.168.1.1").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because router ip %s is not within subnet", testIPPoolNamespace, testIPPoolName, "192.168.1.1"),
			},
		},
		{
			name: "invalid router ip which is the same as network ip",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					Router("192.168.0.0").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because router ip %s is the same as network ip", testIPPoolNamespace, testIPPoolName, "192.168.0.0"),
			},
		},
		{
			name: "invalid router ip which is the same as broadcast ip",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					Router("192.168.0.255").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because router ip %s is the same as broadcast ip", testIPPoolNamespace, testIPPoolName, "192.168.0.255"),
			},
		},
		{
			name: "invalid start ip which is malformed",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					PoolRange("192.168.0.1000", "").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because ParseAddr(\"%s\"): IPv4 field has value >255", testIPPoolNamespace, testIPPoolName, "192.168.0.1000"),
			},
		},
		{
			name: "invalid start ip which is out of subnet",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					PoolRange("192.168.1.100", "").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because start ip %s is not within subnet", testIPPoolNamespace, testIPPoolName, "192.168.1.100"),
			},
		},
		{
			name: "invalid start ip which is the same as network ip",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					PoolRange("192.168.0.0", "").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because start ip %s is the same as network ip", testIPPoolNamespace, testIPPoolName, "192.168.0.0"),
			},
		},
		{
			name: "invalid start ip which is the same as broadcast ip",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					PoolRange("192.168.0.255", "").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because start ip %s is the same as broadcast ip", testIPPoolNamespace, testIPPoolName, "192.168.0.255"),
			},
		},
		{
			name: "invalid end ip which is malformed",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					PoolRange("", "192.168.0.1000").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because ParseAddr(\"%s\"): IPv4 field has value >255", testIPPoolNamespace, testIPPoolName, "192.168.0.1000"),
			},
		},
		{
			name: "invalid end ip which is out of subnet",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					PoolRange("", "192.168.1.100").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because end ip %s is not within subnet", testIPPoolNamespace, testIPPoolName, "192.168.1.100"),
			},
		},
		{
			name: "invalid emd ip which is the same as network ip",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					PoolRange("", "192.168.0.0").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because end ip %s is the same as network ip", testIPPoolNamespace, testIPPoolName, "192.168.0.0"),
			},
		},
		{
			name: "invalid end ip which is the same as broadcast ip",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					PoolRange("", "192.168.0.255").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because end ip %s is the same as broadcast ip", testIPPoolNamespace, testIPPoolName, "192.168.0.255"),
			},
		},
		{
			name: "non-existed network name",
			given: input{
				ipPool: newTestIPPoolBuilder().
					CIDR("192.168.0.0/24").
					NetworkName("nonexist").Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool default/net-1 because network-attachment-definitions.k8s.cni.cncf.io \"%s\" not found", "nonexist"),
			},
		},
	}

	nadGVR := schema.GroupVersionResource{
		Group:    "k8s.cni.cncf.io",
		Version:  "v1",
		Resource: "network-attachment-definitions",
	}

	for _, tc := range testCases {
		clientset := fake.NewSimpleClientset()
		err := clientset.Tracker().Create(nadGVR, tc.given.nad, tc.given.nad.Namespace)
		assert.Nil(t, err, "mock resource should add into fake controller tracker")

		nadCache := fakeclient.NetworkAttachmentDefinitionCache(clientset.K8sCniCncfIoV1().NetworkAttachmentDefinitions)
		vmnetCache := fakeclient.VirtualMachineNetworkConfigCache(clientset.NetworkV1alpha1().VirtualMachineNetworkConfigs)
		validator := NewValidator(nadCache, vmnetCache)

		err = validator.Create(&admission.Request{}, tc.given.ipPool)

		if tc.expected.err != nil {
			assert.Equal(t, tc.expected.err.Error(), err.Error(), tc.name)
		} else {
			assert.Nil(t, err, tc.name)
		}
	}
}

func TestValidator_Update(t *testing.T) {
	type input struct {
		oldIPPool *networkv1.IPPool
		newIPPool *networkv1.IPPool
		nad       *cniv1.NetworkAttachmentDefinition
	}

	type output struct {
		err error
	}

	testCases := []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "valid server ip",
			given: input{
				oldIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					ServerIP("192.168.0.2").
					NetworkName(testNetworkName).Build(),
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					ServerIP("192.168.0.254").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
		},
		{
			name: "invalid server ip which is out of range",
			given: input{
				oldIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					ServerIP("192.168.0.2").
					NetworkName(testNetworkName).Build(),
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					ServerIP("192.168.100.2").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because server ip %s is not within subnet", testIPPoolNamespace, testIPPoolName, testServerIPOutOfRange),
			},
		},
		{
			name: "invalid server ip which is the same as network ip",
			given: input{
				oldIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					ServerIP("192.168.0.2").
					NetworkName(testNetworkName).Build(),
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					ServerIP("192.168.0.0").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because server ip %s cannot be the same as network ip", testIPPoolNamespace, testIPPoolName, "192.168.0.0"),
			},
		},
		{
			name: "invalid server ip which is the same as broadcast ip",
			given: input{
				oldIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					ServerIP("192.168.0.2").
					NetworkName(testNetworkName).Build(),
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					ServerIP("192.168.0.255").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because server ip %s cannot be the same as broadcast ip", testIPPoolNamespace, testIPPoolName, "192.168.0.255"),
			},
		},
		{
			name: "invalid server ip which is the same as router ip",
			given: input{
				oldIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					ServerIP("192.168.0.2").
					Router("192.168.0.254").
					NetworkName(testNetworkName).Build(),
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					ServerIP("192.168.0.254").
					Router("192.168.0.254").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because server ip %s cannot be the same as router ip", testIPPoolNamespace, testIPPoolName, "192.168.0.254"),
			},
		},
		{
			name: "invalid server ip which collides with other allocated ips",
			given: input{

				oldIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					ServerIP("192.168.0.2").
					NetworkName(testNetworkName).
					Allocated("192.168.0.100", "11:22:33:44:55:66").Build(),
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					ServerIP("192.168.0.100").
					NetworkName(testNetworkName).
					Allocated("192.168.0.100", "11:22:33:44:55:66").Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because server ip %s is already allocated", testIPPoolNamespace, testIPPoolName, "192.168.0.100"),
			},
		},
		{
			name: "invalid router ip which is malformed",
			given: input{
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					Router("192.168.0.1000").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because ParseAddr(\"%s\"): IPv4 field has value >255", testIPPoolNamespace, testIPPoolName, "192.168.0.1000"),
			},
		},
		{
			name: "invalid router ip which is out of subnet",
			given: input{
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					Router("192.168.1.1").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because router ip %s is not within subnet", testIPPoolNamespace, testIPPoolName, "192.168.1.1"),
			},
		},
		{
			name: "invalid router ip which is the same as network ip",
			given: input{
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					Router("192.168.0.0").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because router ip %s is the same as network ip", testIPPoolNamespace, testIPPoolName, "192.168.0.0"),
			},
		},
		{
			name: "invalid router ip which is the same as broadcast ip",
			given: input{
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					Router("192.168.0.255").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because router ip %s is the same as broadcast ip", testIPPoolNamespace, testIPPoolName, "192.168.0.255"),
			},
		},
		{
			name: "invalid start ip which is malformed",
			given: input{
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					PoolRange("192.168.0.1000", "").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because ParseAddr(\"%s\"): IPv4 field has value >255", testIPPoolNamespace, testIPPoolName, "192.168.0.1000"),
			},
		},
		{
			name: "invalid start ip which is out of subnet",
			given: input{
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					PoolRange("192.168.1.100", "").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because start ip %s is not within subnet", testIPPoolNamespace, testIPPoolName, "192.168.1.100"),
			},
		},
		{
			name: "invalid start ip which is the same as network ip",
			given: input{
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					PoolRange("192.168.0.0", "").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because start ip %s is the same as network ip", testIPPoolNamespace, testIPPoolName, "192.168.0.0"),
			},
		},
		{
			name: "invalid start ip which is the same as broadcast ip",
			given: input{
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					PoolRange("192.168.0.255", "").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because start ip %s is the same as broadcast ip", testIPPoolNamespace, testIPPoolName, "192.168.0.255"),
			},
		},
		{
			name: "invalid end ip which is malformed",
			given: input{
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					PoolRange("", "192.168.0.1000").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because ParseAddr(\"%s\"): IPv4 field has value >255", testIPPoolNamespace, testIPPoolName, "192.168.0.1000"),
			},
		},
		{
			name: "invalid end ip which is out of subnet",
			given: input{
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					PoolRange("", "192.168.1.100").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because end ip %s is not within subnet", testIPPoolNamespace, testIPPoolName, "192.168.1.100"),
			},
		},
		{
			name: "invalid emd ip which is the same as network ip",
			given: input{
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					PoolRange("", "192.168.0.0").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because end ip %s is the same as network ip", testIPPoolNamespace, testIPPoolName, "192.168.0.0"),
			},
		},
		{
			name: "invalid end ip which is the same as broadcast ip",
			given: input{
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					PoolRange("", "192.168.0.255").
					NetworkName(testNetworkName).Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool %s/%s because end ip %s is the same as broadcast ip", testIPPoolNamespace, testIPPoolName, "192.168.0.255"),
			},
		},
		{
			name: "non-existed network name",
			given: input{
				newIPPool: newTestIPPoolBuilder().
					CIDR(testCIDR).
					NetworkName("nonexist").Build(),
				nad: newTestNetworkAttachmentDefinitionBuilder().Build(),
			},
			expected: output{
				err: fmt.Errorf("could not create IPPool default/net-1 because network-attachment-definitions.k8s.cni.cncf.io \"%s\" not found", "nonexist"),
			},
		},
	}

	nadGVR := schema.GroupVersionResource{
		Group:    "k8s.cni.cncf.io",
		Version:  "v1",
		Resource: "network-attachment-definitions",
	}

	for _, tc := range testCases {
		clientset := fake.NewSimpleClientset()
		err := clientset.Tracker().Create(nadGVR, tc.given.nad, tc.given.nad.Namespace)
		assert.Nil(t, err, "mock resource should add into fake controller tracker")

		nadCache := fakeclient.NetworkAttachmentDefinitionCache(clientset.K8sCniCncfIoV1().NetworkAttachmentDefinitions)
		vmnetCache := fakeclient.VirtualMachineNetworkConfigCache(clientset.NetworkV1alpha1().VirtualMachineNetworkConfigs)
		validator := NewValidator(nadCache, vmnetCache)

		err = validator.Update(&admission.Request{}, tc.given.oldIPPool, tc.given.newIPPool)

		if tc.expected.err != nil {
			assert.Equal(t, tc.expected.err.Error(), err.Error(), tc.name)
		} else {
			assert.Nil(t, err, tc.name)
		}
	}
}
package dns_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consulclient"
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/consul"
	"github.com/pivotal-cf-experimental/destiny/core"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Multiple hosts multiple services", func() {
	var (
		manifest consul.Manifest
		agent    consulclient.AgentStartStopper
	)

	BeforeEach(func() {
		var err error

		manifest, _, err = helpers.DeployConsulWithInstanceCount(3, client, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(manifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{"running"},
			{"running"},
			{"running"},
			{"running"},
		}))

		agent, err = helpers.NewConsulAgent(manifest, 4)
		Expect(err).NotTo(HaveOccurred())

		agent.Start()
	})

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := client.DeleteDeployment(manifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
		agent.Stop()
	})

	It("discovers multiples services on multiple hosts", func() {
		By("registering services", func() {
			healthCheck := fmt.Sprintf("curl -f http://%s:6769/health_check", manifest.Jobs[1].Networks[0].StaticIPs[0])
			manifest.Jobs[0].Properties.Consul.Agent.Services = core.JobPropertiesConsulAgentServices{
				"some-service": core.JobPropertiesConsulAgentService{
					Check: &core.JobPropertiesConsulAgentServiceCheck{
						Name:     "some-service-check",
						Script:   healthCheck,
						Interval: "1m",
					},
				},
				"some-other-service": core.JobPropertiesConsulAgentService{
					Check: &core.JobPropertiesConsulAgentServiceCheck{
						Name:     "some-other-service-check",
						Script:   healthCheck,
						Interval: "1m",
					},
				},
			}
		})

		By("deploying", func() {
			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			yaml, err = client.ResolveManifestVersions(yaml)
			Expect(err).NotTo(HaveOccurred())

			err = client.Deploy(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
				{"running"},
				{"running"},
				{"running"},
			}))
		})

		By("resolving service addresses", func() {
			Eventually(func() ([]string, error) {
				return checkService("some-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs))

			Eventually(func() ([]string, error) {
				return checkService("consul-z1-0.some-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs[0]))

			Eventually(func() ([]string, error) {
				return checkService("consul-z1-1.some-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs[1]))

			Eventually(func() ([]string, error) {
				return checkService("consul-z1-2.some-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs[2]))

			Eventually(func() ([]string, error) {
				return checkService("some-other-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs))

			Eventually(func() ([]string, error) {
				return checkService("consul-z1-0.some-other-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs[0]))

			Eventually(func() ([]string, error) {
				return checkService("consul-z1-1.some-other-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs[1]))

			Eventually(func() ([]string, error) {
				return checkService("consul-z1-2.some-other-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs[2]))
		})
	})
})

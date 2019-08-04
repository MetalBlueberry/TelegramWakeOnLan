package main_test

import (
	"encoding/xml"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	server "github.com/MetalBlueberry/TelegramWakeOnLan/cmd/server"
)

var _ = Describe("List", func() {
	Describe("When executing nmap", func() {
		It("Should return something", func() {
			out, err := server.NewNmapRun("192.168.1.0")
			Expect(err).To(BeNil())
			Expect(len(out.Host)).To(BeNumerically(">", 0))

			outString, err := xml.MarshalIndent(out, " ", " ")
			Expect(err).To(BeNil())
			fmt.Fprint(GinkgoWriter, string(outString))
		})
		It("Should return a list of addresses", func() {
			out, err := server.NewNmapRun("192.168.1.0")
			Expect(err).To(BeNil())
			Expect(len(out.Host)).To(BeNumerically(">", 0))

			addresses := out.GetAddressList()
			fmt.Print(addresses)
			Expect(len(addresses)).To(Equal(len(out.Host)))

		})
	})
})

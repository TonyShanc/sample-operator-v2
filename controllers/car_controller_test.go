package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	samplev1 "github.com/tonyshanc/sample-operator-v2/api/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	layout    = "2006-01-02T15:04:06Z"
	namespace = "sample"
	name      = "sampleat"
	schedule  = time.Now().Add(10 * time.Second).Format(layout)
	command   = "echo 'I am at 1'"
)

var _ = Describe("At Controller", func() {
	const (
		timeout  = time.Second * 60
		interval = timeout * 1
	)
	ctx := context.Background()

	BeforeEach(func() {})

	AfterEach(func() {})

	Context("At Run", func() {

		at := genAt()

		It("Should create at correctly", func() {

			Expect(k8sClient.Create(ctx, &at)).Should(Succeed())

			time.Sleep(time.Second * 2)

			fetched := &samplev1.At{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})

		It("Should delete at correctly", func() {

			Expect(k8sClient.Delete(ctx, &at)).Should((Succeed()))

			time.Sleep(time.Second * 2)

			fetched := &samplev1.At{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, fetched)
				return err == nil
			}, timeout, interval).ShouldNot(BeTrue())
		})
	})
})

func genAt() samplev1.At {
	return samplev1.At{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: samplev1.AtSpec{
			Schedule: schedule,
			Command:  command,
		},
	}
}

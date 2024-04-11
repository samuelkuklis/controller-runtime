/*
Copyright 2018 The Kubernetes Authors.

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

package controller_test

import (
	"context"
	"fmt"
	"time"

	"github.com/samuelkuklis/utils/pointer"
	"go.uber.org/goleak"
	corev1 "k8s.io/api/core/v1"

	"github.com/samuelkuklis/controller-runtime/pkg/client"
	"github.com/samuelkuklis/controller-runtime/pkg/config/v1alpha1"
	"github.com/samuelkuklis/controller-runtime/pkg/controller"
	"github.com/samuelkuklis/controller-runtime/pkg/event"
	"github.com/samuelkuklis/controller-runtime/pkg/handler"
	internalcontroller "github.com/samuelkuklis/controller-runtime/pkg/internal/controller"
	"github.com/samuelkuklis/controller-runtime/pkg/manager"
	"github.com/samuelkuklis/controller-runtime/pkg/reconcile"
	"github.com/samuelkuklis/controller-runtime/pkg/runtime/inject"
	"github.com/samuelkuklis/controller-runtime/pkg/source"
)

var _ = Describe("controller.Controller", func() {
	rec := reconcile.Func(func(context.Context, reconcile.Request) (reconcile.Result, error) {
		return reconcile.Result{}, nil
	})

	Describe("New", func() {
		It("should return an error if Name is not Specified", func() {
			m, err := manager.New(cfg, manager.Options{})
			Expect(err).NotTo(HaveOccurred())
			c, err := controller.New("", m, controller.Options{Reconciler: rec})
			Expect(c).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("must specify Name for Controller"))
		})

		It("should return an error if Reconciler is not Specified", func() {
			m, err := manager.New(cfg, manager.Options{})
			Expect(err).NotTo(HaveOccurred())

			c, err := controller.New("foo", m, controller.Options{})
			Expect(c).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("must specify Reconciler"))
		})

		It("NewController should return an error if injecting Reconciler fails", func() {
			m, err := manager.New(cfg, manager.Options{})
			Expect(err).NotTo(HaveOccurred())

			c, err := controller.New("foo", m, controller.Options{Reconciler: &failRec{}})
			Expect(c).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected error"))
		})

		It("should not return an error if two controllers are registered with different names", func() {
			m, err := manager.New(cfg, manager.Options{})
			Expect(err).NotTo(HaveOccurred())

			c1, err := controller.New("c1", m, controller.Options{Reconciler: rec})
			Expect(err).NotTo(HaveOccurred())
			Expect(c1).ToNot(BeNil())

			c2, err := controller.New("c2", m, controller.Options{Reconciler: rec})
			Expect(err).NotTo(HaveOccurred())
			Expect(c2).ToNot(BeNil())
		})

		It("should not leak goroutines when stopped", func() {
			currentGRs := goleak.IgnoreCurrent()

			watchChan := make(chan event.GenericEvent, 1)
			watch := &source.Channel{Source: watchChan}
			watchChan <- event.GenericEvent{Object: &corev1.Pod{}}

			reconcileStarted := make(chan struct{})
			controllerFinished := make(chan struct{})
			rec := reconcile.Func(func(context.Context, reconcile.Request) (reconcile.Result, error) {
				defer GinkgoRecover()
				close(reconcileStarted)
				// Make sure reconciliation takes a moment and is not quicker than the controllers
				// shutdown.
				time.Sleep(50 * time.Millisecond)
				// Explicitly test this on top of the leakdetection, as the latter uses Eventually
				// so might succeed even when the controller does not wait for all reconciliations
				// to finish.
				Expect(controllerFinished).NotTo(BeClosed())
				return reconcile.Result{}, nil
			})

			m, err := manager.New(cfg, manager.Options{})
			Expect(err).NotTo(HaveOccurred())

			c, err := controller.New("new-controller", m, controller.Options{Reconciler: rec})
			Expect(c.Watch(watch, &handler.EnqueueRequestForObject{})).To(Succeed())
			Expect(err).NotTo(HaveOccurred())

			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				defer GinkgoRecover()
				Expect(m.Start(ctx)).To(Succeed())
				close(controllerFinished)
			}()

			<-reconcileStarted
			cancel()
			<-controllerFinished

			// force-close keep-alive connections.  These'll time anyway (after
			// like 30s or so) but force it to speed up the tests.
			clientTransport.CloseIdleConnections()
			Eventually(func() error { return goleak.Find(currentGRs) }).Should(Succeed())
		})

		It("should not create goroutines if never started", func() {
			currentGRs := goleak.IgnoreCurrent()

			m, err := manager.New(cfg, manager.Options{})
			Expect(err).NotTo(HaveOccurred())

			_, err = controller.New("new-controller", m, controller.Options{Reconciler: rec})
			Expect(err).NotTo(HaveOccurred())

			// force-close keep-alive connections.  These'll time anyway (after
			// like 30s or so) but force it to speed up the tests.
			clientTransport.CloseIdleConnections()
			Eventually(func() error { return goleak.Find(currentGRs) }).Should(Succeed())
		})

		It("should default RecoverPanic from the manager", func() {
			m, err := manager.New(cfg, manager.Options{Controller: v1alpha1.ControllerConfigurationSpec{RecoverPanic: pointer.Bool(true)}})
			Expect(err).NotTo(HaveOccurred())

			c, err := controller.New("new-controller", m, controller.Options{
				Reconciler: reconcile.Func(nil),
			})
			Expect(err).NotTo(HaveOccurred())

			ctrl, ok := c.(*internalcontroller.Controller)
			Expect(ok).To(BeTrue())

			Expect(ctrl.RecoverPanic).NotTo(BeNil())
			Expect(*ctrl.RecoverPanic).To(BeTrue())
		})

		It("should not override RecoverPanic on the controller", func() {
			m, err := manager.New(cfg, manager.Options{Controller: v1alpha1.ControllerConfigurationSpec{RecoverPanic: pointer.Bool(true)}})
			Expect(err).NotTo(HaveOccurred())

			c, err := controller.New("new-controller", m, controller.Options{
				Reconciler:   reconcile.Func(nil),
				RecoverPanic: pointer.Bool(false),
			})
			Expect(err).NotTo(HaveOccurred())

			ctrl, ok := c.(*internalcontroller.Controller)
			Expect(ok).To(BeTrue())

			Expect(ctrl.RecoverPanic).NotTo(BeNil())
			Expect(*ctrl.RecoverPanic).To(BeFalse())
		})
	})
})

var _ reconcile.Reconciler = &failRec{}
var _ inject.Client = &failRec{}

type failRec struct{}

func (*failRec) Reconcile(context.Context, reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func (*failRec) InjectClient(client.Client) error {
	return fmt.Errorf("expected error")
}

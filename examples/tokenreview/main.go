/*
Copyright 2021 The Kubernetes Authors.

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

package main

import (
	"os"

	"github.com/samuelkuklis/controller-runtime/pkg/client/config"
	"github.com/samuelkuklis/controller-runtime/pkg/log"
	"github.com/samuelkuklis/controller-runtime/pkg/log/zap"
	"github.com/samuelkuklis/controller-runtime/pkg/manager"
	"github.com/samuelkuklis/controller-runtime/pkg/manager/signals"
	"github.com/samuelkuklis/controller-runtime/pkg/webhook/authentication"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func init() {
	log.SetLogger(zap.New())
}

func main() {
	entryLog := log.Log.WithName("entrypoint")

	// Setup a Manager
	entryLog.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		entryLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	// Setup webhooks
	entryLog.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()

	entryLog.Info("registering webhooks to the webhook server")
	hookServer.Register("/validate-v1-tokenreview", &authentication.Webhook{Handler: &authenticator{}})

	entryLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
}

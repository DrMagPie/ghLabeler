/*
Copyright © 2020 DrMagPie

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
	"fmt"
	"net/http"
	"os"

	_ "github.com/heroku/x/hmetrics/onload"
	log "github.com/sirupsen/logrus"
	"gopkg.in/go-playground/webhooks.v5/github"
)

type gh struct {
	hook *github.Webhook
}

func (gh *gh) webhook(w http.ResponseWriter, r *http.Request) {

	payload, err := gh.hook.Parse(r, github.PushEvent)
	if err != nil && err == github.ErrEventNotFound {
		log.Error("Event was not present in headdes")
	}

	log.Debug(fmt.Sprintf("%+v", payload))

	switch payload := payload.(type) {
	case github.PullRequestPayload:
		log.Info("PullRequestPayload")
		log.Info(payload)
	case github.ProjectCardPayload:
		log.Info("ProjectCardPayload")
		log.Info(payload)
	default:
		log.Warn("This event is not supported")
	}
	fmt.Fprintf(w, "HOME Page")
}

func main() {
	log.Info("Starting ghLabeler")
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}
	hook, err := github.New(github.Options.Secret(os.Getenv("ACCESS_TOKEN")))
	if err != nil {
		log.Fatal("Failed to create webhook", err)
	}
	gitHook := &gh{hook: hook}
	http.HandleFunc("/", gitHook.webhook)
	err = http.ListenAndServe(fmt.Sprint(":", port), nil)
	if err != nil {
		log.Fatal("ListenAndServe has failed", err)
	}
}

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
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v33/github"
	_ "github.com/heroku/x/hmetrics/onload"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	webhook "gopkg.in/go-playground/webhooks.v5/github"
)

type gh struct {
	hook     *webhook.Webhook
	client   *github.Client
	projects []string
}

func respond(w http.ResponseWriter, message error, httpStatusCode int) error {
	w.WriteHeader(httpStatusCode)
	_, err := w.Write([]byte(message.Error()))
	if err != nil {
		return fmt.Errorf(message.Error(), err)
	}
	return message
}

func labelExists(label string, labels []*github.Label) bool {
	for _, l := range labels {
		if *l.Name == label {
			return true
		}
	}
	return false
}

func itemInList(label string, labels []string) bool {
	for _, l := range labels {
		if label == l {
			return true
		}
	}
	return false
}

func (gh *gh) handleProjectCardEvent(ctx context.Context, w http.ResponseWriter, payload webhook.ProjectCardPayload) error {
	// get all labels
	var labels []*github.Label
	req, err := gh.client.NewRequest("GET", fmt.Sprintf("%s/labels", payload.Repository.URL), nil)
	if err != nil {
		return respond(w, fmt.Errorf("Failed to construct labels request: %w", err), http.StatusInternalServerError)
	}
	_, err = gh.client.Do(ctx, req, &labels)
	if err != nil {
		return respond(w, fmt.Errorf("Failed to list labels: %w", err), http.StatusInternalServerError)
	}

	// get project
	var project *github.Project
	req, err = gh.client.NewRequest("GET", payload.ProjectCard.ProjectURL, nil)
	req.Header.Add("Accept", "application/vnd.github.inertia-preview+json")
	if err != nil {
		return respond(w, fmt.Errorf("Failed to construct project request: %w", err), http.StatusInternalServerError)
	}
	_, err = gh.client.Do(ctx, req, &project)
	if err != nil {
		return respond(w, fmt.Errorf("Failed to get project: %w", err), http.StatusInternalServerError)
	}

	for i, tp := range gh.projects {
		if *project.Name == tp {
			break
		} else if i+1 == len(gh.projects) {
			return respond(w, fmt.Errorf("Project is not tracked"), http.StatusBadRequest)
		}
	}

	// get all columns in tracked repositories
	columns, _, err := gh.client.Projects.ListProjectColumns(ctx, *project.ID, &github.ListOptions{})
	if err != nil {
		return respond(w, fmt.Errorf("Failed to list project columns: %w", err), http.StatusInternalServerError)
	}

	// get issue
	var issue *github.Issue
	req, err = gh.client.NewRequest("GET", payload.ProjectCard.ContentURL, nil)
	if err != nil {
		return respond(w, fmt.Errorf("Failed to construct issue request: %w", err), http.StatusInternalServerError)
	}
	_, err = gh.client.Do(ctx, req, &issue)
	if err != nil {
		return respond(w, fmt.Errorf("Failed to get issue: %w", err), http.StatusInternalServerError)
	}

	for _, column := range columns {
		for _, item := range strings.Split(*column.Name, " ") {
			if payload.ProjectCard.ColumnID == *column.ID && strings.HasPrefix(item, "users:") {
				users := strings.Split(strings.TrimPrefix(item, "users:"), ",")
				var usersToRemove []string
				for _, assignee := range issue.Assignees {
					if !itemInList(*assignee.Login, users) {
						usersToRemove = append(usersToRemove, *assignee.Login)
					}
				}
				issue, _, err = gh.client.Issues.RemoveAssignees(ctx, payload.Repository.Owner.Login, payload.Repository.Name, *issue.Number, usersToRemove)
				if err != nil {
					return respond(w, fmt.Errorf("Failed to delete assignees: %w", err), http.StatusInternalServerError)
				}
				issue, _, err = gh.client.Issues.AddAssignees(ctx, payload.Repository.Owner.Login, payload.Repository.Name, *issue.Number, users)
				if err != nil {
					return respond(w, fmt.Errorf("Failed to add assignees: %w", err), http.StatusInternalServerError)
				}
			} else if strings.HasPrefix(item, "labels:") {
				columnLabels := strings.Split(strings.TrimPrefix(item, "labels:"), ",")

				for _, label := range columnLabels {
					if !labelExists(label, labels) {
						_, _, err = gh.client.Issues.CreateLabel(ctx, payload.Repository.Owner.Login, payload.Repository.Name, &github.Label{Name: &label})
						if err != nil {
							return respond(w, fmt.Errorf("Failed to add label: %w", err), http.StatusInternalServerError)
						}
					}
				}

				if payload.ProjectCard.ColumnID == *column.ID {
					_, _, err = gh.client.Issues.AddLabelsToIssue(ctx, payload.Repository.Owner.Login, payload.Repository.Name, *issue.Number, columnLabels)
					if err != nil {
						return respond(w, fmt.Errorf("Failed to add label: %w", err), http.StatusInternalServerError)
					}
					continue
				}

				for _, label := range issue.Labels {
					if itemInList(*label.Name, columnLabels) {
						_, err = gh.client.Issues.RemoveLabelForIssue(ctx, payload.Repository.Owner.Login, payload.Repository.Name, *issue.Number, *label.Name)
						if err != nil {
							return respond(w, fmt.Errorf("Failed to remove label: %w", err), http.StatusInternalServerError)
						}
					}
				}
			}
		}
	}
	return nil
}

func (gh *gh) webhook(w http.ResponseWriter, r *http.Request) {
	payload, err := gh.hook.Parse(r, webhook.ProjectCardEvent, webhook.PingEvent)
	if err != nil && err == webhook.ErrEventNotFound {
		log.Error(respond(w, fmt.Errorf("Event was not present in headdes: %w", err), http.StatusBadRequest))
		return
	}

	switch payload := payload.(type) {
	case webhook.ProjectCardPayload:
		err := gh.handleProjectCardEvent(r.Context(), w, payload)
		if err != nil {
			log.Error(err)
		}
	case webhook.PingPayload:
		fmt.Fprintf(w, "Connection established")
	default:
		log.Warn("This event is not supported")
	}
	fmt.Fprintf(w, "HOME Page")
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}
	webhookToken := os.Getenv("WEBHOOK_TOKEN")
	if webhookToken == "" {
		log.Fatal("$WEBHOOK_TOKEN must be set")
	}
	hook, err := webhook.New(webhook.Options.Secret(webhookToken))
	if err != nil {
		log.Fatal("Failed to create webhook", err)
	}
	accessToken := os.Getenv("ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("$ACCESS_TOKEN must be set")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	tc := oauth2.NewClient(context.Background(), ts)
	ghHook := &gh{hook: hook, client: github.NewClient(tc), projects: strings.Split(os.Getenv("PROJECT_NAMES"), ",")}
	http.HandleFunc("/", ghHook.webhook)
	log.Info("Starting ghLabeler")
	err = http.ListenAndServe(fmt.Sprint(":", port), nil)
	if err != nil {
		log.Fatal("ListenAndServe has failed", err)
	}
}

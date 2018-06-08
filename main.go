package main

import (
	"regexp"

	"github.com/prune998/docker-registry-client/registry"
	"github.com/sirupsen/logrus"

	"os"

	"github.com/namsral/flag"
)

var (
	// version is filled by -ldflags  at compile time
	version     = "no version set"
	registryURL = flag.String("registryURL", "https://us.gcr.io", "The Docker Registry URL")
	project     = flag.String("project", "", "The Docker project, if using gcloud registry")
	username    = flag.String("username", "", "The Docker Registry user name, use '_token' if using a a gcloud generated token")
	password    = flag.String("password", "", "The Docker Registry password. use 'gcloud auth print-access-token' if connecting to gcloud")
	logLevel    = flag.String("logLevel", "warn", "log level from debug, info, warning, error")
	delete      = flag.Bool("delete", false, "delete the selected images")
	filter      = flag.String("filter", "", "regular expression matching the image/tag to remove")
	log         = logrus.New()
)

func main() {
	flag.Parse()

	log.Out = os.Stdout
	log.Formatter = new(logrus.JSONFormatter)
	log.Level, _ = logrus.ParseLevel(*logLevel)

	if *delete && *filter == "" {
		log.Fatalf("you have to set a -filter when using -delete")
	}

	r := regexp.MustCompile(*filter)

	// connect to the Docker Registry
	hub, err := registry.New(*registryURL, *username, *password, log.Debugf)
	if err != nil {
		log.Fatalf("error connecting to hub, %v", err)
	}

	// get repository (images) list
	repositories, err := hub.Repositories()
	if err != nil {
		log.Fatalf("repositories error, %v", err)

	}

	for _, repo := range repositories {
		// search for each image tag (version)
		images, err := hub.Tags(repo)
		if err != nil {
			log.Fatalf("Listing image error, %v", err)
		}
		for _, image := range images {
			// we join the image tag with the repository name to check
			// against the regular expression
			fullRepo := repo + ":" + image

			if r.MatchString(fullRepo) {
				digest, err := hub.ManifestDigest(repo, image)

				if *delete {
					err = hub.DeleteManifest(repo, digest)
					if err != nil {
						log.Fatalf("error deleting %v: %v", digest, err)
					}
				}
				log.WithFields(logrus.Fields{
					"repository": repo,
					"image":      image,
					"delete":     *delete,
					"fullname":   fullRepo,
					"digest":     digest,
				}).Printf("image found")
			}
		}

	}

}

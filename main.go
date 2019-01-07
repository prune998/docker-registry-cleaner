package main

import (
	"strings"

	digest "github.com/opencontainers/go-digest"
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
	check       = flag.Bool("check", false, "only check for the selected images")
	filter      = flag.String("filter", "", "regular expression matching the image/tag to remove")
	log         = logrus.New()
)

func main() {
	flag.Parse()

	log.Out = os.Stdout
	log.Formatter = new(logrus.JSONFormatter)
	log.Level, _ = logrus.ParseLevel(*logLevel)

	if (*check || *delete) && *filter == "" {
		log.Fatalf("you have to set a -filter when using -check or -delete")
	}

	// r := regexp.MustCompile(*filter)
	repoMatch := strings.Split(*filter, ":")

	// connect to the Docker Registry
	hub, err := registry.New(*registryURL, *username, *password, log.Debugf)
	if err != nil {
		log.Fatalf("error connecting to hub, %v", err)
	}

	if *check {
		tags, err := hub.Tags(repoMatch[0])
		if err != nil {
			log.Errorf("Listing image tags error, %v", err)
			os.Exit(1)
		}
		for _, value := range tags {
			if value == repoMatch[1] {
				log.Debugf("image %s found", *filter)
				os.Exit(0)
			}
		}
		log.Debugf("image %s not found", *filter)
		os.Exit(1)
	}

	// get repository (images) list
	repositories, err := hub.Repositories()
	if err != nil {
		log.Fatalf("repositories error, %v", err)

	}

	log.Debugf("found %d repos", len(repositories))

	for _, repo := range repositories {
		if repo == repoMatch[0] {
			// search for each image tag (version)
			tags, err := hub.Tags(repo)
			log.Debugf("found %d tags for repos %s", len(tags), repo)
			if err != nil {
				log.Fatalf("Listing image tags error, %v", err)
			}
			// man, _ := hub.ManifestV2(repo, tags[0])

			for _, tag := range tags {
				// we join the image tag with the repository name to check
				// against the regular expression
				fullRepo := repo + ":" + tag

				// tags,_  := hub.Tags(image)
				// // man, _ := hub.ManifestV2(repo, image)
				// dig, _ := hub.ManifestDigest(repo, image)
				// layer, _ := hub.LayerMetadata(repo, dig)
				// log.Infof("   %s:  %v", image, tags)

				//if r.MatchString(fullRepo) {
				log.Infof("tag: %s   len: %d", tag, len(repoMatch))
				if len(repoMatch) == 2 && repoMatch[1] != tag {
					continue
				}

				// get the image digest and split it to only keep the hash
				// and drop the encoding
				// FYI a digest is like sha256:b618c166f0b066dd9bba7
				imageDigest, err := hub.ManifestDigest(repo, tag)
				digestParts := strings.Split(string(imageDigest), ":")
				if len(digestParts) != 2 {
					log.Errorf("image digest error: %v", imageDigest)
					break
				}

				if *delete {
					// delete the tag first
					err = hub.DeleteTags(repo, tag)
					if err != nil {
						log.Fatalf("error deleting tag %v: %v", tag, err)
					}
					err = hub.DeleteManifest(repo, digest.Digest(digestParts[1]))
					if err != nil {
						log.Fatalf("error deleting image %v: %v", imageDigest, err)
					}
				}

				// do a pretty json log
				log.WithFields(logrus.Fields{
					"repository": repo,
					"tag":        tag,
					"delete":     *delete,
					"fullname":   fullRepo,
					"digest":     imageDigest,
				}).Printf("image found")
			}

			break
		}

	}

}

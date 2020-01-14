package main

import (
	"regexp"
	"strings"

	"github.com/opencontainers/go-digest"
	"github.com/prune998/docker-registry-client/registry"
	"github.com/sirupsen/logrus"

	"os"

	"github.com/namsral/flag"
)

var (
	// version is filled by -ldflags  at compile time
	version        = "no version set"
	registryURL    = flag.String("registryURL", "https://us.gcr.io", "The Docker Registry URL")
	project        = flag.String("project", "", "The Docker project, if using gcloud registry")
	username       = flag.String("username", "", "The Docker Registry user name, use '_token' if using a a gcloud generated token")
	password       = flag.String("password", "", "The Docker Registry password. use 'gcloud auth print-access-token' if connecting to gcloud")
	logLevel       = flag.String("logLevel", "warn", "log level from debug, info, warning, error")
	delete         = flag.Bool("delete", false, "delete the selected images")
	check          = flag.Bool("check", false, "only check for the image defined in -filter. filter must use image:tag without regexp")
	deleteUntagged = flag.Bool("deleteUntagged", false, "delete images from a repository if they have no tags. Only work when the filter contains `.*` as tag version")
	filter         = flag.String("filter", "", "regular expression matching the image/tag to remove")
	log            = logrus.New()
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
	if len(repoMatch) != 2 {
		log.Fatalf("-filter option must be formed with <image>:<version>")
	}

	// connect to the Docker Registry
	hub, err := registry.New(*registryURL, *username, *password, log.Debugf)
	if err != nil {
		log.Fatalf("error connecting to hub, %v", err)
	}

	if *check {
		tags, err := hub.Tags(repoMatch[0])
		if err != nil {
			log.Errorf("check: Listing image tags error, %v", err)
			os.Exit(1)
		}
		for _, value := range tags {
			if value == repoMatch[1] {
				log.Debugf("check: image %s found", *filter)
				os.Exit(0)
			}
		}
		log.Debugf("check: image %s not found", *filter)
		os.Exit(1)
	}

	// pre-compile the regexp filters
	imageFilter := regexp.MustCompile(repoMatch[0])
	tagFilter := regexp.MustCompile(repoMatch[1])

	// get repository (images) list
	repositories, err := hub.Repositories()
	if err != nil {
		log.Fatalf("repositories error, %v", err)

	}

	log.Infof("found %d repos", len(repositories))

	for _, repo := range repositories {

		if !imageFilter.MatchString(repo) {
			log.Debugf("Skipping repos %s", repo)
			continue
		}

		log.Infof("found repos %s", repo)

		// search for each image tag (version)
		tags, err := hub.Tags(repo)
		log.Infof("found %d tags for repos %s", len(tags), repo)
		if err != nil {
			log.Fatalf("Listing image tags error, %v", err)
		}

		for _, tag := range tags {
			// we join the image tag with the repository name to check
			// against the regular expression
			fullRepo := repo + ":" + tag

			if !tagFilter.MatchString(tag) {
				log.Infof("skipping tag: %s", tag)
				continue
			}

			// get the image digest and split it to only keep the hash
			// and drop the encoding
			// FYI a digest is like sha256:b618c166f0b066dd9bba7
			imageDigest, err := hub.ManifestDigest(repo, tag)
			log.Infof("digest %s for tag %s", imageDigest, tag)
			// digestParts := strings.Split(string(imageDigest), ":")
			// if len(digestParts) != 2 {
			// 	log.Errorf("image digest error: %v", imageDigest)
			// 	break
			// }

			if *delete {
				// delete the tag first
				err = hub.DeleteTags(repo, tag)
				if err != nil {
					log.Fatalf("error deleting tag %v: %v", tag, err)
				}

				// also delete the manifest
				// if other tags are still there, silent the error
				err = hub.DeleteManifest(repo, imageDigest)
				if err != nil {
					log.Debugf("error deleting image %v: %v", imageDigest, err)
				}
			}

			// do a pretty json log
			log.WithFields(logrus.Fields{
				"repository": repo,
				"tag":        tag,
				"delete":     *delete,
				"fullname":   fullRepo,
				"digest":     imageDigest,
			}).Warn("image found")
		}

		if *deleteUntagged && repoMatch[1] == ".*" {
			// delete all images in this repo which are not tagged
			images, err := hub.Images(repo)
			if err != nil {
				log.Debugf("error getting image list for repo %d", repo)
			}
			for _, imageDigestStr := range images {
				log.Debugf("deleting untagged image %v", imageDigestStr)
				imageDigest, err := digest.Parse(imageDigestStr)
				if err != nil {
					log.Debugf("error getting digest from image %v: %v", imageDigestStr, err)
				}
				err = hub.DeleteManifest(repo, imageDigest)
				if err != nil {
					log.Debugf("error deleting image %v: %v", imageDigestStr, err)
				}
			}
		}
	}

}

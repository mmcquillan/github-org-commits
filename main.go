package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/google/go-github/v43/github"
	"golang.org/x/oauth2"
)

var rateCount = 0
var rateTime = time.Now()

func main() {

	// to be fixed
	var out = "./out"
	var org = os.Getenv("GITHUB_ORGANIZATION")
	var githubId = os.Getenv("GITHUB_ID")
	var startRepo = ""
	var startPage = 1

	// repos map to order for restart
	var repoList = []string{}

	// write to file
	file, err := os.OpenFile(out, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	fileOut := bufio.NewWriter(file)

	// github client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// list options
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{Page: 1},
	}

	// start output
	fmt.Print("Starting...")

	// list all repositories for the authenticated user
	loop := true
	for loop {

		rateLimit()
		repos, resp, err := client.Repositories.ListByOrg(ctx, org, opt)
		if err != nil {
			log.Print(err)
		}

		for _, r := range repos {
			repoList = append(repoList, *r.Name)
		}

		// end of list
		if resp.NextPage == 0 {
			loop = false
		}

		// next list
		opt.ListOptions.Page = resp.NextPage

	}

	// sort repos
	sort.Strings(repoList)

	for _, repo := range repoList {

		// skip to start repo
		if startRepo == "" || repo == startRepo {

			// in case we are skipping
			startRepo = ""
			fmt.Print("\n" + repo)

			// list options
			lopt := &github.CommitsListOptions{
				ListOptions: github.ListOptions{Page: 1},
			}
			if startPage > 1 {
				lopt.ListOptions.Page = startPage
				startPage = 1
			}

			cloop := true
			for cloop {

				rateLimit()
				commits, cresp, err := client.Repositories.ListCommits(ctx, org, repo, lopt)
				if err != nil {
					log.Print(err)
				}
				fmt.Printf(" %d", lopt.ListOptions.Page)

				for _, c := range commits {

					// check for nil
					var author = ""
					if c.Author != nil {
						if c.Author.Login != nil {
							author = *c.Author.Login
						}
					}
					var committer = ""
					if c.Committer != nil {
						if c.Committer.Login != nil {
							committer = *c.Committer.Login
						}
					}

					if author == githubId || committer == githubId {
						fileOut.WriteString(*c.SHA + " | " + repo + " | " + author + " | " + committer + "\n")
					}
				}

				// end of list
				if cresp.NextPage == 0 {
					cloop = false
				}

				// next list
				lopt.ListOptions.Page = cresp.NextPage
				fileOut.Flush()

			}
		} else {
			continue
		}

	}

	// finish up
	file.Close()

}

func rateLimit() {
	rateCount++
	if rateCount >= 4999 {
		if time.Since(rateTime).Minutes() > float64(61.00) {
			rateCount = 0
			rateTime = time.Now()
		} else {
			log.Printf("Rate Limit Wait (%f more minutes)", (float64(61.00) - time.Since(rateTime).Minutes()))
			time.Sleep(60 * time.Second)
		}
	}
}

package main

import (
	"github.com/bsphere/le_go"
	"github.com/codegangsta/cli"
	"github.com/colebrumley/dockeraction"
	"github.com/fsouza/go-dockerclient"
	log "github.com/sirupsen/logrus"
	"os"
)

var (
	leConn       *le_go.Logger
	dockerClient *dockeraction.ActionClient
)

func main() {
	app := cli.NewApp()
	app.Version = "v0.1"
	app.Name = "leddlogger"
	app.Usage = "Logentries Docker daemon logger: Like `docker logs`, but for all containers. To logentries."
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "token",
			Usage:  "Logentries token",
			Value:  "XXXX-XXXX-XXXX-XXXX",
			EnvVar: "LE_TOKEN",
		},
		cli.StringSliceFlag{
			Name:  "ignore",
			Usage: "list of images to ignore",
			Value: &cli.StringSlice{"weaveworks/weaveexec:1.0.2"},
		},
	}
	app.Action = watchEvents
	app.Run(os.Args)
}

func watchEvents(c *cli.Context) {
	var err error
	dockerClient, err = dockeraction.GetDefaultActionClient()
	if err != nil {
		log.Fatal(err)
	}
	token := c.String("token")

	eventChan := make(chan *docker.APIEvents)
	log.Println("Watching for Docker events")
	if err := dockerClient.AddEventListener(eventChan); err != nil {
		log.Fatal(err)
	}

	for event := range eventChan {
		if event.Status == "start" {
			skip := false
			for _, i := range c.StringSlice("ignore") {
				if i == event.From {
					skip = true
					break
				}
			}
			if !skip {
				go logContainer(event.ID[:12], token)
				log.Printf("Log stream for %s started (%s)", event.ID[:12], event.From)
			}
		}
	}
}

func logContainer(id, token string) {
	out := make(chan string)
	errChan := make(chan error, 1)
	le, err := le_go.Connect(token)
	if err != nil {
		log.Fatal(err)
	}
	defer le.Close()
	le.SetPrefix("[" + id + "]")
	le.Println("Starting log stream")
	go dockerClient.StreamLogs(id, out, errChan)
	for {
		select {
		case line := <-out:
			le.Println(line)
		case err := <-errChan:
			if err != nil {
				le.Println(err.Error())
				log.Error(err)
			}
			log.Printf("Stopping log stream from %s", id)
			le.Println("Stopping log stream")
			return
		}
	}
}

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aybabtme/go-ipc/sync"
	"github.com/urfave/cli/v2"
)

var Version = "devel"

var (
	semaphoreNameFlag = &cli.StringFlag{
		Name:  "name",
		Usage: "name of the semaphore",
	}
	semaphoreSizeFlag = &cli.Int64Flag{
		Name:  "size",
		Usage: "size of the semaphore",
	}
)

func main() {
	app := cli.App{
		Name:  "semaphore",
		Usage: "run shell commands while holding a semaphore",
		Authors: []*cli.Author{
			{Name: "Antoine Grondin", Email: "antoinegrondin@gmail.com"},
		},
		Version: Version,
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: "create a semaphore",
				Flags: []cli.Flag{semaphoreNameFlag, semaphoreSizeFlag},
				Action: func(c *cli.Context) error {
					semName := c.String(semaphoreNameFlag.Name)
					if semName == "" {
						return fmt.Errorf("required: --%s", semaphoreNameFlag.Name)
					}
					size := c.Int(semaphoreSizeFlag.Name)
					if size < 1 {
						return fmt.Errorf("required: --%s", semaphoreSizeFlag.Name)
					}
					sem, err := sync.NewSemaphore(semName, os.O_CREATE|os.O_TRUNC, 0644, size)
					if err != nil {
						return fmt.Errorf("creating semaphore %q: %v", semName, err)
					}
					if err := sem.Close(); err != nil {
						return fmt.Errorf("closing semaphore %q: %v", semName, err)
					}
					return nil
				},
			},
			{
				Name:  "acquire",
				Usage: "acquire a lock on a semaphore",
				Flags: []cli.Flag{semaphoreNameFlag},
				Action: func(c *cli.Context) error {
					semName := c.String(semaphoreNameFlag.Name)
					if semName == "" {
						return fmt.Errorf("required: --%s", semaphoreNameFlag.Name)
					}
					sem, err := sync.NewSemaphore(semName, os.O_RDWR, 0644, 0)
					if err != nil {
						return fmt.Errorf("creating semaphore %q: %v", semName, err)
					}
					sem.Wait()
					if err := sem.Close(); err != nil {
						return fmt.Errorf("closing semaphore %q: %v", semName, err)
					}
					return nil
				},
			},
			{
				Name:  "release",
				Usage: "release a lock on a semaphore",
				Flags: []cli.Flag{semaphoreNameFlag},
				Action: func(c *cli.Context) error {
					semName := c.String(semaphoreNameFlag.Name)
					if semName == "" {
						return fmt.Errorf("required: --%s", semaphoreNameFlag.Name)
					}
					sem, err := sync.NewSemaphore(semName, os.O_RDWR, 0644, 0)
					if err != nil {
						return fmt.Errorf("opening semaphore %q: %v", semName, err)
					}
					sem.Signal(1)
					if err := sem.Close(); err != nil {
						return fmt.Errorf("closing semaphore %q: %v", semName, err)
					}
					return nil
				},
			},
		},
	}
	log.SetFlags(0)
	log.SetPrefix(app.Name)
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

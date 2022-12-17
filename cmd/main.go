package main

import (
	"log"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wangao1236/my-runc/pkg/command"
)

const usage = `my-runc is a simple container runtime implementation.
               The purpose of this project is to learn how docker works and how to write a docker by ourselves
               Enjoy it, just for fun.`

func main() {
	app := cli.NewApp()
	app.Name = "my-runc"
	app.Usage = usage

	app.Commands = []cli.Command{
		command.RunCommand,
		command.InitCommand,
		command.CommitCommand,
		command.ListCommand,
		command.LogCommand,
		command.ExecCommand,
		command.StopCommand,
		command.RemoveCommand,
		command.NetworkCommand,
	}

	app.Before = func(context *cli.Context) error {
		logrus.SetReportCaller(true)
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
		logrus.SetOutput(os.Stdout)
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

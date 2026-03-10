package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/iami317/logx"
	"github.com/iami317/peaker"
	"github.com/iami317/peaker/plugins"
	"github.com/urfave/cli/v2"
)

func main() {
	RunApp()
}

func RunApp() {
	app := cli.NewApp()
	app.Usage = ""
	app.Name = "Peaker"
	app.Version = "v1.0.0-beta"
	app.Description = ""
	app.HelpName = "./peaker -h"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "ip_list",
			Aliases: []string{"i"},
			Value:   "./iplist.txt",
		},
		&cli.BoolFlag{
			Name:  "verbose",
			Value: true,
			Usage: "Set log level to debug",
		},
		&cli.BoolFlag{
			Name:    "check_alive",
			Aliases: []string{"cA"},
			Usage:   "Check if the target is alive",
		},
		&cli.IntFlag{
			Name:    "thread",
			Aliases: []string{"c"},
			Value:   30,
			Usage:   "Number of concurrent threads",
		},
		&cli.IntFlag{
			Name:    "timeout",
			Aliases: []string{"t"},
			Value:   50000,
			Usage:   "The maximum execution time of a single ip",
		},
		&cli.IntFlag{
			Name:    "timeout-single",
			Aliases: []string{"tS"},
			Value:   30,
			Usage:   "The maximum execution time of a single account",
		},
		&cli.IntFlag{
			Name:    "thread-single",
			Aliases: []string{"tC"},
			Value:   100,
			Usage:   "The number of concurrency for a single protocol",
		},
		&cli.BoolFlag{
			Name:  "protocol",
			Usage: "view supported protocols",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "output file path for JSON results (one per line)",
		},
	}
	app.Action = RunServer
	err := app.Run(os.Args)
	if err != nil {
		logx.Fatalf("engin err: %v", err)
		return
	}
}

func RunServer(ctx *cli.Context) error {
	if ctx.Bool("protocol") {
		for protocol, _ := range plugins.ScanMap {
			logx.Silent(string(protocol))
		}
		return nil
	}
	config := peaker.Config{
		Logger: logx.New(),
	}
	if ctx.IsSet("verbose") {
		config.Logger.SetLevel("verbose")
		config.DebugMode = true
	}

	if ctx.IsSet("check_alive") {
		config.CheckAlive = true
	}

	if ctx.IsSet("thread-single") {
		config.ThreadSingle = ctx.Int("thread-single")
	}
	config.Thread = ctx.Int("thread")
	config.TimeOut = time.Duration(ctx.Int("timeout")) * time.Second
	config.Ts = time.Duration(ctx.Int("tS")) * time.Second

	w := peaker.NewWeak(config)
	w.StartTime = time.Now()

	// 内置用户名和密码字典
	userDict := []string{
		"root",
		"admin",
		"user",
	}
	passDict := []string{
		"123456",
		"admin",
		"password",
		"root",
		"",
	}
	ipList, err := w.ReadIpList(ctx.String("ip_list"))
	if err != nil {
		return err
	}

	var (
		writer *bufio.Writer
		file   *os.File
	)
	if out := ctx.String("output"); out != "" {
		file, err = os.Create(out)
		if err != nil {
			return fmt.Errorf("create output file failed: %w", err)
		}
		defer file.Close()
		writer = bufio.NewWriter(file)
		defer writer.Flush()
	}

	resultChan := make(chan interface{}, 1)
	w.RunTask(ipList, userDict, passDict, resultChan)
	for v := range resultChan {
		r := v.(*peaker.ResultOut)
		if len(r.Crack) > 0 {
			data, err := json.Marshal(r)
			if err != nil {
				logx.Errorf("marshal result error: %v", err)
				continue
			}
			if writer != nil {
				if _, err := writer.Write(append(data, '\n')); err != nil {
					logx.Errorf("write result to file error: %v", err)
				}
			} else {
				fmt.Println(string(data))
			}
		}
	}
	return nil
}

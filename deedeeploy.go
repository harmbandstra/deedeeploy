package main

import (
	"bytes"
	"github.com/codegangsta/cli"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
)

const (
	YamlConfig  = "deedeeploy.yml"
	ProtocolSsh = "ssh"
	ProtocolFtp = "ftp"
	VcsGit      = "git"
	VcsSvn      = "svn"
)

type (
	Config struct {
		Name     string
		Hosts    []string
		Protocol string
		Vcs      string
		Subtype  string
		User     string
		Path     string
		Postcmd  []string
	}

	Yaml struct {
		Environments []Config
	}

	DDSession struct {
		Environment   string
		Revision      string
		Debug         bool
		Configuration Config
		SshConfig     *ssh.ClientConfig
	}
)

func (dd *DDSession) init() {
	data, fileError := ioutil.ReadFile(YamlConfig)
	if fileError != nil {
		log.Fatalf("Unable to read config file %s.", YamlConfig)
	}

	t := Yaml{}
	err := yaml.Unmarshal(data, &t)
	if err != nil {
		log.Fatalf("Unable to parse config file: %v", err)
	}

	var envFound = false
	for _, elem := range t.Environments {
		if elem.Name == dd.Environment {
			if dd.Debug == true {
				log.Printf("Enviroment '%s' found in config file.", elem.Name)
			}
			dd.Configuration = elem
			envFound = true
			break
		}
	}

	if envFound == false {
		log.Fatalf("Environment '%s' not found in config file.", dd.Environment)
	}

	if dd.Configuration.Protocol != ProtocolFtp && dd.Configuration.Protocol != ProtocolSsh {
		log.Fatalf("Invalid protocol specified: %s", dd.Configuration.Protocol)
	}

	if dd.Configuration.Vcs != VcsSvn && dd.Configuration.Vcs != VcsGit {
		log.Fatalf("Invalid vcs specified: %s", dd.Configuration.Vcs)
	}

	if dd.Configuration.User == "" {
		log.Fatalf("No username specified.")
	}

	if dd.Configuration.Path == "" {
		log.Fatalf("No path specified.")
	}

	// TODO: check if path is valid

	if len(dd.Configuration.Hosts) <= 0 {
		log.Fatalf("No host(s) found.")
	}

	// initialize the ssh agent and ssh config
	dd.initSshConfig()
}

func (dd *DDSession) initSshConfig() {
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		log.Fatal(err)
	}
	// defer conn.Close()

	ag := agent.NewClient(conn)
	auths := []ssh.AuthMethod{ssh.PublicKeysCallback(ag.Signers)}
	config := &ssh.ClientConfig{
		User: dd.Configuration.User,
		Auth: auths,
	}

	dd.SshConfig = config
}

func (dd *DDSession) deploy() {

	for _, host := range dd.Configuration.Hosts {
		if dd.Debug == true {
			log.Printf("Start deployment on host '%s'.", host)
		}

		client, err := ssh.Dial("tcp", host, dd.SshConfig)
		if err != nil {
			log.Fatalf("Unable to connect to %s: %v", host, err)
		}

		dd.updateRemoteCode(client)
		dd.runPostCmd(client)
	}

}

func (dd *DDSession) updateRemoteCode(client *ssh.Client) {
	if dd.Configuration.Vcs == VcsSvn {
		cmd := "cd " + dd.Configuration.Path + " && svn up"
		dd.runRemoteCommand(client, cmd)
	} else if dd.Configuration.Vcs == VcsGit {
		log.Fatal("Git not yet implemented!")
	}
}

func (dd *DDSession) runPostCmd(client *ssh.Client) {

	cmd := strings.Join(dd.Configuration.Postcmd, " && ")

	if dd.Debug == true {
		log.Printf("Running Post-commands")
		log.Printf(cmd)
	}

	dd.runRemoteCommand(client, cmd)
}

func (dd *DDSession) runRemoteCommand(client *ssh.Client, cmd string) string {
	session, err := client.NewSession()
	if err != nil {
		log.Fatalln("Failed to create session:", err)
	}
	defer session.Close()

	if dd.Debug == true {
		log.Printf("Running command: %s", cmd)
	}

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run(cmd); err != nil {
		log.Fatal("Failed to run: " + err.Error())
	}

	if dd.Debug == true {
		log.Printf(b.String())
	}

	return b.String()
}

func main() {
	dd := DDSession{}

	app := cli.NewApp()
	app.Name = "deedeeploy"
	app.Usage = "Easy automated deployment of your code"
	app.Version = "0.1"
	app.Author = "Harm Bandstra"
	app.Email = "hbandstra@gmail.com"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "environment, e",
			Value: "",
			Usage: "environment to deploy to (defined in config)",
		},
		cli.StringFlag{
			Name:  "revision, r",
			Value: "",
			Usage: "revision to update deploy target to (default to HEAD)",
		},
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "show verbose / debug output",
		},
	}
	app.Action = func(c *cli.Context) {

		if c.IsSet("debug") {
			dd.Debug = true
			log.Println("Debug mode on")
		} else {
			dd.Debug = false
		}

		if c.String("environment") == "" {
			cli.ShowAppHelp(c)
			os.Exit(0)
		} else {
			dd.Environment = c.String("environment")
		}

		if c.String("revision") == "" {
			dd.Revision = ""
			if dd.Debug == true {
				log.Println("No revision specified, defaulting to HEAD")
			}
		} else {
			dd.Revision = c.String("revision")
		}

		// load yaml, do basic checking
		dd.init()

		// if all checks pass, deploy
		dd.deploy()
	}

	app.Run(os.Args)
}

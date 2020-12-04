package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func runCommand(terminal *term.Terminal, command string, args []string) {
	cmd := exec.Command("ls", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprint(terminal, out.String())
}

var farewellText string = "So long and thanks for all the fish!"

func main() {
	var sshPort string
	envSshPort := os.Getenv("SSH_PORT")
	if envSshPort == "" {
		sshPort = ":9999"
	} else {
		sshPort = ":" + envSshPort
	}

	var idRsaFile string
	envIdRsaFile := os.Getenv("ID_RSA_FILE")
	if envIdRsaFile == "" {
		idRsaFile = "tmp/id_rsa"
	} else {
		idRsaFile = envIdRsaFile
	}

	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	privateBytes, err := ioutil.ReadFile(idRsaFile)
	if err != nil {
		panic("Failed to open private key from disk. Try running `ssh-keygen` in tmp/ to create one.")
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		panic("Failed to parse private key")
	}
	config.AddHostKey(private)
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0%s", sshPort))
	if err != nil {
		panic("failed to listen for connection")
	}
	for {
		nConn, err := listener.Accept()
		if err != nil {
			panic("failed to accept incoming connection")
		}
		go func() {
			_, chans, reqs, err := ssh.NewServerConn(nConn, config)
			if err != nil {
				fmt.Println("failed to handshake with new client:", err)
				return
			}
			go ssh.DiscardRequests(reqs)
			for newChannel := range chans {
				if newChannel.ChannelType() != "session" {
					newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
					continue
				}

				channel, requests, err := newChannel.Accept()
				if err != nil {
					fmt.Println("could not accept channel:", err)
					return
				}

				go func(in <-chan *ssh.Request) {
					for req := range in {
						if req.Type == "shell" {
							req.Reply(true, nil)
						}
					}
				}(requests)
				go func() {
					defer channel.Close()

					fmt.Fprintln(channel, "Welcome (^o^)/! Type help for available commands")
					fmt.Fprintln(channel, "")
					fmt.Fprintln(channel, "")
					terminal := term.NewTerminal(channel, `(^o^)/ ~ `)
					cmds := map[string]func([]string){
						"help": func(args []string) {
							// use tabwriter to neatly format command help
							helpWriter := tabwriter.NewWriter(terminal, 8, 8, 0, '\t', 0)
							commands := [][]string{
								{"ls", "list contents of current directory"},
								{"cat", "display contents of current file"},
								{"hi", "say hi"},
								{"exit", "exit the terminal"},
							}
							for _, command := range commands {
								fmt.Fprintf(helpWriter, " %s\t%s\r\n", command[0], command[1])
							}
							helpWriter.Flush()
						},
						"cat": func(args []string) {
							runCommand(terminal, "cat", args)
						},
						"ls": func(args []string) {
							runCommand(terminal, "ls", args)
						},
						"hi": func(args []string) {
							fmt.Fprintln(terminal, "Hi!")
						},
						"exit": func(args []string) {
							fmt.Fprintln(terminal, "So long, and thanks for all the fish!")
							channel.Close()
						},
					}
					for {
						line, err := terminal.ReadLine()
						if err != nil {
							fmt.Fprintln(terminal, farewellText)
							break
						}
						trimmedInput := strings.TrimSpace(line)

						inputElements := strings.Split(trimmedInput, " ")
						inputCmd := inputElements[0]
						inputArgs := inputElements[1:]

						if cmd, ok := cmds[inputCmd]; ok {
							fmt.Fprintln(terminal, "")
							cmd(inputArgs)
							fmt.Fprintln(terminal, "")
						} else {
							if inputCmd != "" {

								fmt.Fprintln(terminal, "")
								fmt.Fprintln(terminal, inputCmd, "is not a valid command.")
							}
						}
					}

				}()
			}
		}()
	}

}
